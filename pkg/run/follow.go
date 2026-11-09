package run

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-hclog"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/client"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
	"google.golang.org/grpc"
)

const (
	defaultPollInterval = 30 * time.Second
	defaultRetryDelay   = time.Second
	maxRetryDelay       = 15 * time.Second
	// readAttempts bounds the retries of a read that fails with a transient error: about a minute in
	// all with the default delays, so a brief outage doesn't end following a run that may have been
	// waiting for hours.
	readAttempts = 8
	// resubscribeDelay is how long to wait before resubscribing to run events after the stream ends.
	resubscribeDelay = time.Second
	// defaultCleanupTimeout bounds the interrupt handling when no Options.CleanupContext is given.
	defaultCleanupTimeout = time.Minute
)

// Causes a decider's context is canceled with: the run moved on, or the wait timeout passed.
var (
	errDecisionNotNeeded = errors.New("the decision is no longer needed")
	errDecisionTimedOut  = errors.New("timed out waiting for a decision")
)

// errRunNotCancelable is reported when a run can't be canceled on an interrupt because it is no longer
// doing work.
var errRunNotCancelable = errors.New("the run is no longer in progress, so it was not canceled")

// follower holds what the stages share while a run is followed.
type follower struct {
	client  *client.GRPCClient
	logger  hclog.Logger
	opts    *Options
	decider Decider
	runID   string

	// changed is signaled (holding at most one pending notification) whenever the run may have
	// changed: on each run event and each (re)subscription.
	changed chan struct{}

	// newestRunCache holds the newest run delivered by a run event, which saves reading the run back.
	newestRunCache runCache

	// lastWaiting is the last Waiting event sent, while nothing else has been reported since.
	lastWaiting *Waiting

	// waitTimeout is Options.WaitTimeout, or DefaultWaitTimeout.
	waitTimeout time.Duration
	// blockedSince is when the stage now blocked on a decision became blocked, without a break;
	// it is zero while no stage is. The wait timeout is measured from it, whether the decider is
	// being asked or the decision is being made elsewhere.
	blockedSince time.Time

	// leaveChosen and leaveReason are the Chosen and Reason of the decision a stage just
	// returned decisionDetach for, carried into the Outcome.
	leaveChosen bool
	leaveReason string

	// firstStep is true while the stages are synced against the first snapshot, for SkipFinished.
	firstStep bool

	// advisories holds each shown check's failed advisory policies, by check ID; advisoryChecks is the
	// order the checks were first shown in. A re-evaluated check replaces its entry.
	advisories     map[string][]string
	advisoryChecks []string
}

// Follow follows a run through each of its stages in order, reporting each stage's progress to
// opts.Observer and asking opts.Decider when a stage blocks on a decision. Stages decide where they
// stand from fresh run snapshots rather than from individual events, so a missed event can't cause
// a stage to be skipped, and a run that moves backward (a retry or an undiscard) is followed back.
//
// Canceling ctx interrupts following; opts.Interrupter then decides what happens to the run, and
// the outcome is OutcomeInterrupted. An error is returned only when following itself failed, such
// as when the run can't be read.
func Follow(ctx context.Context, grpcClient *client.GRPCClient, run *pb.Run, opts *Options) (outcome *Outcome, err error) {
	if opts == nil || opts.Observer == nil {
		return nil, errors.New("an observer is required to follow a run")
	}

	f := &follower{
		client:  grpcClient,
		logger:  opts.Logger,
		opts:    opts,
		decider: opts.Decider,
		runID:   run.Metadata.Id,
		changed: make(chan struct{}, 1),
	}
	if f.logger == nil {
		f.logger = hclog.NewNullLogger()
	}
	f.waitTimeout = opts.WaitTimeout
	if f.waitTimeout <= 0 {
		f.waitTimeout = DefaultWaitTimeout
	}
	defer func() {
		if outcome != nil {
			outcome.Advisories = f.advisoryPolicies()
		}
	}()

	stages := buildPipeline(f, run)

	watchCtx, cancel := context.WithCancel(ctx)
	watchDone := make(chan struct{})
	go func() {
		defer close(watchDone)
		f.watch(watchCtx, run.WorkspaceId)
	}()
	// Stop watching before returning, so nothing runs after Follow has returned.
	defer func() {
		cancel()
		<-watchDone
	}()

	pollInterval := opts.PollInterval
	if pollInterval <= 0 {
		pollInterval = defaultPollInterval
	}

	lastRun := run
	f.firstStep = true
	// forceRead is set when the last snapshot is known to be behind: a stage reported it stale, or a
	// decision was acted on. A snapshot from an event sent before then may not show the change.
	forceRead := false
	for {
		current, err := f.refreshRun(ctx, lastRun, forceRead)
		if err != nil {
			if ctx.Err() != nil {
				return f.interrupted(ctx, lastRun), nil
			}
			return nil, err
		}
		lastRun = current

		outcome, wait, err := f.step(ctx, stages, current)
		f.firstStep = false
		if err != nil {
			if ctx.Err() != nil {
				return f.interrupted(ctx, lastRun), nil
			}
			return nil, err
		}
		if outcome != nil {
			return outcome, nil
		}
		if wait.decisionStage == nil {
			f.blockedSince = time.Time{}
		} else if f.blockedSince.IsZero() {
			f.blockedSince = time.Now()
		}
		forceRead = !wait.wait
		if !wait.wait {
			continue
		}

		delay := pollInterval
		if wait.decisionStage != nil {
			remaining := f.waitTimeout - time.Since(f.blockedSince)
			if remaining <= 0 {
				return &Outcome{Kind: OutcomeDetached, Run: current, Stage: *wait.decisionStage, TimedOut: true}, nil
			}
			delay = min(delay, remaining)
		}

		select {
		case <-f.changed:
		case <-time.After(delay):
		case <-ctx.Done():
			return f.interrupted(ctx, lastRun), nil
		}
	}
}

// waitState is what step found the run waiting on, when it isn't over.
type waitState struct {
	// wait is true when the run has to change before anything else can happen.
	wait bool
	// decisionStage, when set, is the stage blocked on a decision, being made elsewhere or just acted
	// on; the wait timeout keeps running while it is set.
	decisionStage *StageRef
}

// step walks the stages from the first one against a single snapshot. It returns an outcome when
// following is over, or a wait state saying whether the run has to change before anything else can
// happen.
func (f *follower) step(ctx context.Context, stages []stage, run *pb.Run) (*Outcome, waitState, error) {
	for _, st := range stages {
		status, err := st.sync(ctx, run)
		if err != nil {
			return nil, waitState{}, err
		}

		switch status {
		case stageDone:
			continue
		case stageStale:
			return nil, waitState{}, nil
		case stageFailed:
			return &Outcome{Kind: OutcomeFailed, Run: run, Stage: st.ref()}, waitState{}, nil
		case stageWaiting, stageWaitingOnDecision:
			// A run that is over can't move this stage on, so don't wait on it forever.
			if IsRunFinal(run.Status) {
				return finalOutcome(run, st.ref()), waitState{}, nil
			}
			wait := waitState{wait: true}
			if status == stageWaitingOnDecision {
				ref := st.ref()
				wait.decisionStage = &ref
			}
			return nil, wait, nil
		case stageNeedsDecision:
			ds, ok := st.(decidingStage)
			if !ok {
				return nil, waitState{}, fmt.Errorf("stage %s needs a decision but cannot ask for one", st.ref().Name)
			}
			if f.blockedSince.IsZero() {
				f.blockedSince = time.Now()
			}
			result, err := ds.decide(ctx, run)
			if err != nil {
				return nil, waitState{}, err
			}
			ref := st.ref()
			switch result {
			case decisionDetach:
				return &Outcome{Kind: OutcomeDetached, Run: run, Stage: ref, LeaveChosen: f.leaveChosen, LeaveReason: f.leaveReason}, waitState{}, nil
			case decisionDiscarded:
				return &Outcome{Kind: OutcomeDiscarded, Run: run}, waitState{}, nil
			case decisionTimedOut:
				return &Outcome{Kind: OutcomeDetached, Run: run, Stage: ref, TimedOut: true}, waitState{}, nil
			default:
				// Acted: re-read the run and carry on. The stage may still be blocked (on other approvers,
				// or a decision deferred elsewhere), so the wait timeout keeps running until it isn't.
				return nil, waitState{decisionStage: &ref}, nil
			}
		default:
			return nil, waitState{}, fmt.Errorf("stage %s reported an unknown status %d", st.ref().Name, status)
		}
	}

	return finalOutcome(run, StageRef{}), waitState{}, nil
}

// finalOutcome is the outcome for a run whose stages have nothing left to show: failed when the
// run ended without completing (naming the stage it stopped at, if known), completed otherwise.
func finalOutcome(run *pb.Run, stage StageRef) *Outcome {
	if IsRunFailed(run.Status) {
		return &Outcome{Kind: OutcomeFailed, Run: run, Stage: stage}
	}
	return &Outcome{Kind: OutcomeCompleted, Run: run}
}

// shouldSkipFinished reports whether stages that already finished should be passed over silently: the
// caller asked for it, and this is the first snapshot.
func (f *follower) shouldSkipFinished() bool {
	return f.opts.SkipFinished && f.firstStep
}

// decideOutcome is how a call to the decider ended, other than with a decision.
type decideOutcome int

const (
	decided decideOutcome = iota
	// abandoned means the run moved on, so the decision was no longer needed.
	abandoned
	// timedOut means the wait timeout passed first.
	timedOut
)

// decide runs ask with a context that is canceled once the decision is no longer needed, or when the
// wait timeout passes: whenever the run may have changed, it is re-read, and stillNeeded is asked
// whether the decision is still needed. run is the snapshot the decision is being asked on.
func decide[T any](
	ctx context.Context,
	f *follower,
	run *pb.Run,
	stillNeeded func(*pb.Run) bool,
	ask func(context.Context) (T, error),
) (result T, how decideOutcome, err error) {
	deadlineCtx, cancelDeadline := context.WithDeadlineCause(ctx, f.blockedSince.Add(f.waitTimeout), errDecisionTimedOut)
	defer cancelDeadline()
	askCtx, cancel := context.WithCancelCause(deadlineCtx)
	defer cancel(nil)

	done := make(chan struct{})
	var wg sync.WaitGroup
	wg.Go(func() {
		latest := run
		for {
			select {
			case <-done:
				return
			case <-askCtx.Done():
				return
			case <-f.changed:
			}
			var err error
			latest, err = f.refreshRun(askCtx, latest, false)
			if err != nil {
				if askCtx.Err() == nil {
					f.logger.Debug("failed to read run while waiting for a decision", "error", err)
				}
				continue
			}
			if !stillNeeded(latest) {
				cancel(errDecisionNotNeeded)
				return
			}
		}
	})

	result, err = ask(askCtx)
	close(done)
	wg.Wait()

	// The decider may have shown output of its own, so the next wait is reported again.
	f.lastWaiting = nil

	// Cut off only if the decider returned the context's error: an answer given just as the run moved
	// on (or the timeout passed) is acted on, and the action's conflict handling copes with the run
	// having moved, while an unrelated error is passed on even if askCtx ended at the same time.
	if err != nil && ctx.Err() == nil && isContextErr(askCtx, err) {
		var zero T
		switch cause := context.Cause(askCtx); {
		case errors.Is(cause, errDecisionNotNeeded):
			return zero, abandoned, nil
		case errors.Is(cause, errDecisionTimedOut):
			return zero, timedOut, nil
		}
	}
	return result, decided, err
}

// isContextErr reports whether err is ctx's error or its cause, so it came from ctx ending.
func isContextErr(ctx context.Context, err error) bool {
	return errors.Is(err, context.Canceled) ||
		errors.Is(err, context.DeadlineExceeded) ||
		errors.Is(err, context.Cause(ctx))
}

// emit reports an event, and resets the waiting line so the next wait is reported again.
func (f *follower) emit(e Event) {
	f.lastWaiting = nil
	f.opts.Observer.Event(e)
}

// waiting reports what the run is waiting on, once per change of reason. A zero reason reports
// nothing.
func (f *follower) waiting(stage StageRef, reason WaitReason, detail string) {
	if reason == 0 {
		return
	}
	w := Waiting{Stage: stage, Reason: reason, Detail: detail}
	if f.lastWaiting != nil && *f.lastWaiting == w {
		return
	}
	f.opts.Observer.Event(w)
	f.lastWaiting = &w
}

// recordAdvisories remembers a check's failed advisory policies, replacing what an earlier
// evaluation of the same check reported.
func (f *follower) recordAdvisories(checkID string, names []string) {
	if f.advisories == nil {
		f.advisories = map[string][]string{}
	}
	if _, seen := f.advisories[checkID]; !seen {
		f.advisoryChecks = append(f.advisoryChecks, checkID)
	}
	f.advisories[checkID] = names
}

// advisoryPolicies lists the failed advisory policies across every check shown, each name once.
func (f *follower) advisoryPolicies() []string {
	var names []string
	seen := map[string]bool{}
	for _, checkID := range f.advisoryChecks {
		for _, name := range f.advisories[checkID] {
			if !seen[name] {
				seen[name] = true
				names = append(names, name)
			}
		}
	}
	return names
}

// interrupted handles an interrupt: it reports it, re-reads the run and, unless the run is already
// over, does what the interrupter chooses. The interrupt handling runs under Options.CleanupContext,
// since ctx is already done.
func (f *follower) interrupted(ctx context.Context, lastRun *pb.Run) *Outcome {
	outcome := &Outcome{Kind: OutcomeInterrupted, Run: lastRun}
	f.emit(Interrupted{})

	cleanupCtx := f.opts.CleanupContext
	if cleanupCtx == nil {
		var cancel context.CancelFunc
		cleanupCtx, cancel = context.WithTimeout(context.WithoutCancel(ctx), defaultCleanupTimeout)
		defer cancel()
	}

	run, err := f.getRun(cleanupCtx)
	if err != nil {
		if cleanupCtx.Err() == nil {
			f.logger.Debug("failed to read run after interrupt", "error", err)
		}
		return outcome
	}
	outcome.Run = run

	if IsRunFinal(run.Status) || f.opts.Interrupter == nil {
		return outcome
	}
	action := f.opts.Interrupter.OnInterrupt(cleanupCtx, run)
	if cleanupCtx.Err() != nil {
		// Interrupted again; leave the run going.
		return outcome
	}
	if action == InterruptLeave {
		return outcome
	}

	// The interrupter may have taken a while (asking the user, say), so act on the run as it is now:
	// a run that has since moved to a decision mustn't be canceled, nor one now doing work discarded.
	if latest, err := f.getRun(cleanupCtx); err == nil {
		run = latest
		outcome.Run = latest
	} else if cleanupCtx.Err() == nil {
		f.logger.Debug("failed to re-read run after interrupt decision", "error", err)
	}

	switch action {
	case InterruptEnd:
		if IsRunAwaitingDecision(run.Status) {
			_, err := f.client.RunsClient.DiscardRun(cleanupCtx, &pb.DiscardRunRequest{RunId: f.runID})
			switch {
			case err == nil:
				outcome.RunDiscarded = true
				return outcome
			case !isStateConflict(err):
				f.emit(InterruptFailed{Action: "discard", Err: err})
				return outcome
			}
			// The decision was made in the meantime and the run moved on; cancel it if it's now running.
			if run, err = f.getRun(cleanupCtx); err != nil {
				f.logger.Debug("failed to read run after discard conflict", "error", err)
				return outcome
			}
			outcome.Run = run
		}
		// The run may have ended in the meantime, which is nothing to report on an end.
		f.cancelRun(cleanupCtx, outcome, false)
	case InterruptCancelRunIfInProgress:
		f.cancelRun(cleanupCtx, outcome, true)
	}
	return outcome
}

// cancelRun cancels the outcome's run if it is doing work. When reportNotCancelable is set, a run
// that is no longer doing work is reported, rather than leaving a request to cancel it unanswered.
func (f *follower) cancelRun(ctx context.Context, outcome *Outcome, reportNotCancelable bool) {
	if !IsRunCancelable(outcome.Run.Status) {
		if reportNotCancelable {
			f.emit(InterruptFailed{Action: "cancel", Err: errRunNotCancelable})
		}
		return
	}
	if _, err := f.client.RunsClient.CancelRun(ctx, &pb.CancelRunRequest{Id: f.runID}); err != nil {
		if isStateConflict(err) {
			if !reportNotCancelable {
				return
			}
			err = errRunNotCancelable
		}
		f.emit(InterruptFailed{Action: "cancel", Err: err})
		return
	}
	outcome.RunCanceled = true
}

// watch keeps a run event subscription open for as long as ctx lives, keeping each event's run and
// signaling changed on every event and every (re)subscription.
func (f *follower) watch(ctx context.Context, workspaceID string) {
	for ctx.Err() == nil {
		err := client.StreamWithReconnect(ctx,
			func(streamCtx context.Context) (grpc.ServerStreamingClient[pb.RunEvent], error) {
				stream, err := f.client.RunsClient.SubscribeToRunEvents(streamCtx, &pb.SubscribeToRunEventsRequest{
					WorkspaceId: &workspaceID,
					RunId:       &f.runID,
				})
				if err == nil {
					// Events sent while we were not subscribed aren't replayed, so the run delivered before
					// may be behind: forget it, so the run is read back.
					f.newestRunCache.clear()
					f.notify()
				}
				return stream, err
			},
			func(event *pb.RunEvent) (bool, error) {
				f.newestRunCache.offer(event.GetRun())
				f.notify()
				return false, nil
			},
		)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			// The poll fallback in Follow keeps the run moving; keep trying to resubscribe.
			f.logger.Debug("run event subscription ended", "error", err)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(resubscribeDelay):
		}
	}
}

// versionOf is a run's version, or zero if it has none.
func versionOf(run *pb.Run) int64 {
	if run == nil || run.Metadata == nil {
		return 0
	}
	return run.Metadata.Version
}

// runCache holds the newest run delivered by a run event. The server bumps a run's version on every
// change, its plan, apply and task stages included, so versions alone say which of two snapshots is
// newer, and a snapshot no newer than one already seen has nothing new.
//
// Only events are offered, and both offer and clear are called from the watch goroutine in the order
// the stream delivers events, so a run from before a resubscribe can't be offered after the clear.
type runCache struct {
	mu  sync.Mutex
	run *pb.Run
}

// offer keeps run if it is newer than the one held.
func (c *runCache) offer(run *pb.Run) {
	if run == nil || run.Metadata == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if versionOf(run) > versionOf(c.run) {
		c.run = run
	}
}

// clear forgets the run held.
func (c *runCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.run = nil
}

// newerThan returns the run held if it is newer than since, or nil.
func (c *runCache) newerThan(since *pb.Run) *pb.Run {
	c.mu.Lock()
	defer c.mu.Unlock()
	if versionOf(c.run) > versionOf(since) {
		return c.run
	}
	return nil
}

// refreshRun returns the run as it is now: the run delivered by an event if it is newer than since,
// the snapshot the caller last used, and otherwise (or when forceRead is set) the newer of a read and
// the run delivered by an event.
func (f *follower) refreshRun(ctx context.Context, since *pb.Run, forceRead bool) (*pb.Run, error) {
	if !forceRead {
		if run := f.newestRunCache.newerThan(since); run != nil {
			return run, nil
		}
	}
	run, err := f.getRun(ctx)
	if err != nil {
		return nil, err
	}
	// An event may have delivered a newer run while the read was in flight.
	if newer := f.newestRunCache.newerThan(run); newer != nil {
		return newer, nil
	}
	return run, nil
}

// notify signals that the run may have changed, without blocking.
func (f *follower) notify() {
	select {
	case f.changed <- struct{}{}:
	default:
	}
}

func (f *follower) getRun(ctx context.Context) (*pb.Run, error) {
	run, err := retryRead(ctx, f, func(ctx context.Context) (*pb.Run, error) {
		return f.client.RunsClient.GetRunByID(ctx, &pb.GetRunByIDRequest{Id: f.runID})
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get run: %w", err)
	}
	return run, nil
}

// retryRead calls read, retrying with backoff while it fails with a transient error.
func retryRead[T any](ctx context.Context, f *follower, read func(context.Context) (T, error)) (T, error) {
	delay := f.opts.RetryDelay
	if delay <= 0 {
		delay = defaultRetryDelay
	}
	for attempt := 1; ; attempt++ {
		v, err := read(ctx)
		if err == nil || !isTransient(err) || attempt >= readAttempts || ctx.Err() != nil {
			return v, err
		}
		f.logger.Debug("retrying a failed read", "attempt", attempt, "delay", delay, "error", err)

		select {
		case <-ctx.Done():
			return v, err
		case <-time.After(delay):
		}
		delay = min(delay*2, maxRetryDelay)
	}
}

// streamJobLogs streams a job's logs to the observer until the job completes.
func (f *follower) streamJobLogs(ctx context.Context, stage StageRef, jobID string) error {
	f.lastWaiting = nil
	var lastSeen int32

	// Resume from the last byte seen so a reconnect (e.g. server drain) doesn't reprint logs.
	return client.StreamWithReconnect(ctx,
		func(streamCtx context.Context) (grpc.ServerStreamingClient[pb.JobLogStreamEvent], error) {
			return f.client.JobsClient.SubscribeToJobLogStream(streamCtx, &pb.SubscribeToJobLogStreamRequest{
				JobId:           jobID,
				LastSeenLogSize: &lastSeen,
			})
		},
		func(event *pb.JobLogStreamEvent) (bool, error) {
			lastSeen = event.Size
			if event.Data != nil && event.Data.Logs != "" {
				if err := f.opts.Observer.Logs(stage, event.Data.Logs); err != nil {
					return false, err
				}
			}
			return event.Completed, nil
		},
	)
}
