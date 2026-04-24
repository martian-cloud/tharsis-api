// Package agent implements the AI agent system for Tharsis.
package agent

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/m-mizutani/gollem"
	simplestrategy "github.com/m-mizutani/gollem/strategy/simple"
	"github.com/m-mizutani/gollem/trace"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/plugin/llm"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const systemPrompt = `
	You are a tharsis assistant that helps with various tasks. Always try to use the tharsis tools when appropriate and assume the tasks in the context of tharsis.

	MANDATORY FIRST STEP — SKILL CHECK:
	On EVERY user message, you MUST call the load_skill tool FIRST before calling any other tool.
	- Look at the available skills and determine if any match the user's request.
	- If a matching skill exists, call load_skill to retrieve its instructions, then follow them.
	- NEVER call other tools before completing this skill check.
	- Only skip this step if no skills are relevant to the request.

	If the user is making a general comment such as "Hello, how are you?", you can respond in a more conversational style without the need for strict markdown formatting.
	However, if the user is asking a question, structure your response with the following format and use Markdown for formatting - no introductory chit-chat, no apologies, no meta comments outside the content.

	When responding to a user request which is not a simple question, respond with the following markdown format:

	### Brief Summary (1–3 sentences max)

	#### Detailed Explanation
	- bullet points
	- or numbered steps if procedural

	#### Key Takeaways / Recommendations
	- point A
	- point B

	(Use tables, code blocks, quotes, etc. when appropriate)

	CRITICAL:
	- Only call a tool if the required arguments are explicitly in the context data. Do NOT make assumptions or attempt to infer missing arguments.
	- Do not echo back out tool responses in your final answer. Instead, use the information from the tool response to construct a complete and direct answer to the user's question.
`

// SystemAgent orchestrates AI agent runs using an LLM client and tool sets.
type SystemAgent struct {
	logger     logger.Logger
	dbClient   *db.Client
	agentStore Store
	llmClient  llm.Client
	skills     []skill
}

// NewSystemAgent creates a new SystemAgent, loading skills from embedded files.
func NewSystemAgent(logger logger.Logger, dbClient *db.Client, agentStore Store, llmClient llm.Client) *SystemAgent {
	skills, err := loadSkills()
	if err != nil {
		logger.Errorf("failed to load agent skills: %v", err)
	}
	return &SystemAgent{
		logger:     logger,
		dbClient:   dbClient,
		agentStore: agentStore,
		llmClient:  llmClient,
		skills:     skills,
	}
}

// RunInput contains the parameters for an agent run.
type RunInput struct {
	Session         *models.AgentSession
	Run             *models.AgentSessionRun
	PreviousRun     *models.AgentSessionRun
	ToolSets        []gollem.ToolSet
	ContextMessages []string
	Task            string
	Timeout         time.Duration
}

// Run starts an agent run with the given input, managing its lifecycle and persistence.
func (a *SystemAgent) Run(ctx context.Context, input *RunInput) {
	runID := input.Run.Metadata.ID

	runExecutionError := a.execute(ctx, input)

	// Close any toolsets that implement io.Closer
	for _, ts := range input.ToolSets {
		if c, ok := ts.(io.Closer); ok {
			if closeErr := c.Close(); closeErr != nil {
				a.logger.Errorf("failed to close toolset: %v", closeErr)
			}
		}
	}

	// Re-fetch the run to check if cancel was requested
	run, err := a.dbClient.AgentSessionRuns.GetAgentSessionRunByID(ctx, runID)
	if err != nil {
		a.logger.Errorf("failed to query agent run after execution: %v", err)
		return
	}

	if run == nil {
		a.logger.Errorf("agent run with id %s not found", runID)
		return
	}

	if run.CancelRequested {
		run.Status = models.AgentSessionRunCancelled
	} else if runExecutionError != nil {
		if errors.ErrorCode(runExecutionError) == errors.EInternal {
			a.logger.Errorw("agent run failed: %w", runExecutionError, "run_id", run.Metadata.ID)
		}
		errMsg := errors.ErrorMessage(runExecutionError)
		run.Status = models.AgentSessionRunErrored
		run.ErrorMessage = &errMsg
	} else {
		run.Status = models.AgentSessionRunFinished
	}

	if _, dbErr := a.dbClient.AgentSessionRuns.UpdateAgentSessionRun(ctx, run); dbErr != nil {
		a.logger.Errorf("failed to update run status: %v", dbErr)
	}
}

func (a *SystemAgent) execute(ctx context.Context, input *RunInput) error {
	session := input.Session
	run := input.Run
	runID := run.Metadata.ID
	sessionID := session.Metadata.ID

	var lastMessageID *string
	var prevRunCancelled bool
	if input.PreviousRun != nil {
		lastMessageID = input.PreviousRun.LastMessageID
		prevRunCancelled = input.PreviousRun.Status == models.AgentSessionRunCancelled
	}

	// Create persistence middleware
	persister := newMessagePersister(a.dbClient, a.agentStore, a.logger, sessionID, runID, lastMessageID, []string{loadSkillToolName})

	// Save the user input message before execution
	if err := persister.saveUserInput(ctx, input.Task); err != nil {
		return fmt.Errorf("failed to save user input: %w", err)
	}

	// Wrap with cancellable strategy to check for cancel requests on each iteration
	cancelStrategy := &cancellableStrategy{
		base: simplestrategy.New(),
		checkCancelled: func(ctx context.Context) bool {
			r, err := a.dbClient.AgentSessionRuns.GetAgentSessionRunByID(ctx, runID)
			return err == nil && r != nil && r.CancelRequested
		},
	}

	traceRecorder := trace.New(
		trace.WithRepository(newTraceRepository(a.agentStore, sessionID)),
		trace.WithTraceID(runID),
	)

	agentOpts := []gollem.Option{
		gollem.WithLoopLimit(20),
		gollem.WithStrategy(cancelStrategy),
		gollem.WithToolSets(input.ToolSets...),
		gollem.WithTools(newSkillsTool(a.skills)),
		gollem.WithSystemPrompt(systemPrompt),
		gollem.WithContentBlockMiddleware((&quotaMiddleware{
			dbClient:  a.dbClient,
			llmClient: a.llmClient,
			session:   session,
		}).Middleware()),
		gollem.WithContentBlockMiddleware(persister.ContentBlockMiddleware()),
		gollem.WithToolMiddleware(persister.ToolMiddleware()),
		gollem.WithTrace(traceRecorder),
		gollem.WithHistoryRepository(newHistoryRepository(a.agentStore), sessionID),
	}

	// Execute with timeout
	agentCtx, cancel := context.WithTimeout(ctx, input.Timeout)
	defer cancel()

	agentCtx = llm.WithSessionID(agentCtx, sessionID)

	llmAgent := gollem.New(a.llmClient, agentOpts...)

	var inputs []gollem.Input
	if prevRunCancelled {
		inputs = append(inputs, gollem.Text("[System: Your previous response was cancelled by the user and may appear truncated in the conversation history. Do not attempt to continue or reference that incomplete response. Focus only on the new request below.]"))
	}

	task := input.Task
	if len(input.ContextMessages) > 0 {
		task = fmt.Sprintf("%s\n\n%s", strings.Join(input.ContextMessages, "\n"), input.Task)
	}

	inputs = append(inputs, gollem.Text(task))

	_, execErr := llmAgent.Execute(agentCtx, inputs...)

	// Persist trace regardless of execution outcome
	if finishErr := traceRecorder.Finish(ctx); finishErr != nil {
		a.logger.Errorf("failed to persist trace: %v", finishErr)
	}

	return execErr
}
