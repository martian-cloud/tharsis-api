package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/response"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/auth"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/vcs"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

const (
	// gitLabEventHeader is the header containing the event type for GitLab.
	gitLabEventHeader = "X-Gitlab-Event"

	// gitHubEventHeader is the header containing the event type for GitHub.
	gitHubEventHeader = "X-GitHub-Event"

	// oAuthCallbackResponseBody is the response returned for a successful
	// OAuth flow completion.
	oAuthCallbackResponseBody = `
	<html>
	<head>
	<title>VCS Provider OAuth Setup</title>
	<style type="text/css">
	body {
		font-family: monospace;
		color: #fff;
		background-color: #000;
	}
	</style>
	</head>
	<body>
	<p>Tharsis has authenticated with the VCS provider successfully. This page can now be closed.</p>
	</body>
	</html>
	`
)

// gitLabWebhookRequest represents a GitLab webhook request.
type gitLabWebhookRequest struct {
	// Used only for merge requests.
	ObjectAttributes struct {
		Source struct {
			// Contains the full path of the source repository.
			PathWithNamespace string `json:"path_with_namespace"`
		} `json:"source"`

		// The branch this MR is for.
		TargetBranch string `json:"target_branch"`
		// The branch this MR originated from.
		SourceBranch string `json:"source_branch"`
		// Allows filtering merge requests based on action.
		Action string `json:"action"`

		// Used only for merge requests.
		LastCommit struct {
			// Contains the ID for the head commit.
			ID string `json:"id"`
		} `json:"last_commit"`
	} `json:"object_attributes"`

	Before string `json:"before"` // Commit SHA before push event.
	After  string `json:"after"`  // Commit SHA after push event.
	Ref    string `json:"ref"`    // Tag or branch name.
}

// gitHubWebhookRequest represents a GitHub webhook request.
type gitHubWebhookRequest struct {
	// Allows filtering merge requests based on action.
	Action string `json:"action"`

	// Used only for pull requests.
	PullRequest struct {
		Head struct {
			CommitSHA    string `json:"sha"`
			SourceBranch string `json:"ref"`
			Repo         struct {
				SourceRepository string `json:"full_name"`
			} `json:"repo"`
		} `json:"head"`

		Base struct {
			TargetBranch string `json:"ref"`
		} `json:"base"`
	} `json:"pull_request"`

	Before string `json:"before"` // Commit SHA before push event.
	After  string `json:"after"`  // Commit SHA after push event.
	Ref    string `json:"ref"`    // Tag or branch name.
}

type vcsController struct {
	logger        logger.Logger
	respWriter    response.Writer
	authenticator *auth.Authenticator
	vcsService    vcs.Service
}

// NewVCSController creates an instance of vcsController.
func NewVCSController(
	logger logger.Logger,
	respWriter response.Writer,
	authenticator *auth.Authenticator,
	vcsService vcs.Service,
) Controller {
	return &vcsController{
		logger,
		respWriter,
		authenticator,
		vcsService,
	}
}

// RegisterRoutes adds routes to the router.
func (c *vcsController) RegisterRoutes(router chi.Router) {
	// OAuth handler.
	router.Get("/vcs/auth/callback", c.OAuthHandler)

	// Webhook handler.
	router.Post("/vcs/events", c.DesignateEventHandler)
}

func (c *vcsController) OAuthHandler(w http.ResponseWriter, r *http.Request) {
	queries := r.URL.Query()

	// Use the system caller.
	request := r.WithContext(auth.WithCaller(r.Context(), &auth.SystemCaller{}))

	if err := c.vcsService.ProcessOAuth(request.Context(), &vcs.ProcessOAuthInput{
		AuthorizationCode: queries.Get("code"),
		State:             queries.Get("state"),
	}); err != nil {
		// Return a simple EUnauthorized here.
		c.logger.Infof("Unauthorized request to %s %s: %v", r.Method, r.URL.Path, err)
		c.respWriter.RespondWithError(w, errors.New(errors.EUnauthorized, "Unauthorized"))
		return
	}

	// Return some HTML indicating the flow has completed.
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(oAuthCallbackResponseBody)); err != nil {
		c.logger.Errorf("failed to write callback response body in OAuthHandler: %v", err)
		c.respWriter.RespondWithError(w, errors.New(errors.EInternal, "Internal error has occurred"))
	}
}

func (c *vcsController) DesignateEventHandler(w http.ResponseWriter, r *http.Request) {
	// Authenticate the request.
	caller, err := c.authenticator.Authenticate(r.Context(), findToken(r), false)
	if err != nil {
		c.logger.Infof("Unauthorized request to %s %s: %v", r.Method, r.URL.Path, err)
		c.respWriter.RespondWithError(w, errors.Wrap(err, errors.EUnauthorized, "unauthorized"))
		return
	}

	// Make sure this is a VCS caller.
	vcsCaller, ok := caller.(*auth.VCSWorkspaceLinkCaller)
	if !ok {
		c.respWriter.RespondWithError(w, errors.New(errors.EForbidden, "Invalid token"))
		return
	}

	// Add caller to request context.
	r = r.WithContext(auth.WithCaller(r.Context(), caller))

	// Call the appropriate handler for provider type.
	switch models.VCSProviderType(vcsCaller.Provider.Type) {
	case models.GitLabProviderType:
		err = c.gitLabHandler(r)
	case models.GitHubProviderType:
		err = c.gitHubHandler(r)

	default:
		// Should never happen, but we'll handle it anyway.
		err = errors.New(errors.EInvalid, "invalid provider type: %s", vcsCaller.Provider.Type)
	}

	if err != nil {
		c.respWriter.RespondWithError(w, err)
		return
	}

	c.respWriter.RespondWithJSON(w, nil, http.StatusOK)
}

func (c *vcsController) gitLabHandler(r *http.Request) error {
	var req gitLabWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return err
	}

	return c.vcsService.ProcessWebhookEvent(r.Context(), &vcs.ProcessWebhookEventInput{
		Action:           req.ObjectAttributes.Action,
		HeadCommitID:     req.ObjectAttributes.LastCommit.ID,
		EventHeader:      r.Header.Get(gitLabEventHeader),
		SourceRepository: req.ObjectAttributes.Source.PathWithNamespace,
		SourceBranch:     req.ObjectAttributes.SourceBranch,
		TargetBranch:     req.ObjectAttributes.TargetBranch,
		Before:           req.Before,
		After:            req.After,
		Ref:              req.Ref,
	})
}

func (c *vcsController) gitHubHandler(r *http.Request) error {
	var req gitHubWebhookRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return err
	}

	return c.vcsService.ProcessWebhookEvent(r.Context(), &vcs.ProcessWebhookEventInput{
		Action:           req.Action,
		HeadCommitID:     req.PullRequest.Head.CommitSHA,
		EventHeader:      r.Header.Get(gitHubEventHeader),
		SourceRepository: req.PullRequest.Head.Repo.SourceRepository,
		SourceBranch:     req.PullRequest.Head.SourceBranch,
		TargetBranch:     req.PullRequest.Base.TargetBranch,
		Before:           req.Before,
		After:            req.After,
		Ref:              req.Ref,
	})
}

// findToken find the token from the request and returns it.
func findToken(r *http.Request) string {
	// Check if GitHub token, passed in via query param.
	gitHub := r.URL.Query().Get("token")
	if len(gitHub) > 0 {
		return gitHub
	}

	// Check if GitLab webhook token.
	return r.Header.Get("X-Gitlab-Token")
}
