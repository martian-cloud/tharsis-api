package resolver

import (
	"context"
	"strconv"

	graphql "github.com/graph-gophers/graphql-go"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services/adminlogtail"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger/logstore"
)

// defaultAdminLogTailLimit is the number of entries returned when the query omits "limit".
const defaultAdminLogTailLimit = 50

// maxAdminLogTailSearchLength caps the search filter length to bound server-side scanning.
const maxAdminLogTailSearchLength = 256

// AdminLogTailSubscriptionInput is the input type for the adminLogTailEvents subscription.
type AdminLogTailSubscriptionInput struct {
	Levels *[]string
	Search *string
}

// Validate checks the subscription input.
func (i *AdminLogTailSubscriptionInput) Validate() error {
	if i.Levels != nil && !logstore.AreValidLevels((*i.Levels)...) {
		return errors.New("levels must each be one of DEBUG, INFO, WARN, ERROR", errors.WithErrorCode(errors.EInvalid))
	}

	if i.Search != nil && len(*i.Search) > maxAdminLogTailSearchLength {
		return errors.New("search filter must be %d characters or fewer", maxAdminLogTailSearchLength, errors.WithErrorCode(errors.EInvalid))
	}

	return nil
}

// AdminLogTailQueryArgs are the arguments for the adminLogTail query.
type AdminLogTailQueryArgs struct {
	Limit  *int32
	Levels *[]string
	Search *string
}

// Validate checks the adminLogTail query arguments.
func (a *AdminLogTailQueryArgs) Validate() error {
	if a.Levels != nil && !logstore.AreValidLevels((*a.Levels)...) {
		return errors.New("levels must each be one of DEBUG, INFO, WARN, ERROR", errors.WithErrorCode(errors.EInvalid))
	}

	if a.Search != nil && len(*a.Search) > maxAdminLogTailSearchLength {
		return errors.New("search filter must be %d characters or fewer", maxAdminLogTailSearchLength, errors.WithErrorCode(errors.EInvalid))
	}

	return nil
}

// AdminLogTailEntryResolver resolves a single log entry.
type AdminLogTailEntryResolver struct {
	entry *logstore.LogEntry
}

// ID resolver
func (r *AdminLogTailEntryResolver) ID() graphql.ID {
	return graphql.ID(strconv.FormatUint(r.entry.Seq, 10))
}

// Timestamp resolver
func (r *AdminLogTailEntryResolver) Timestamp() graphql.Time {
	return graphql.Time{Time: r.entry.Timestamp}
}

// Level resolver
func (r *AdminLogTailEntryResolver) Level() string {
	return r.entry.Level
}

// Message resolver
func (r *AdminLogTailEntryResolver) Message() string {
	return r.entry.Message
}

// Fields resolver — returns the JSON string or nil when empty.
func (r *AdminLogTailEntryResolver) Fields() *string {
	if r.entry.Fields == "" {
		return nil
	}
	return &r.entry.Fields
}

// Caller resolver — returns the source location or nil when unknown.
func (r *AdminLogTailEntryResolver) Caller() *string {
	if r.entry.Caller == "" {
		return nil
	}
	return &r.entry.Caller
}

// Stack resolver — returns the stack trace or nil when absent.
func (r *AdminLogTailEntryResolver) Stack() *string {
	if r.entry.Stack == "" {
		return nil
	}
	return &r.entry.Stack
}

// AdminLogTailEventResolver resolves a live-tail subscription event: a log entry, or a
// terminal stream error when the entry's Err is set.
type AdminLogTailEventResolver struct {
	entry *logstore.LogEntry
}

// LogEntry resolver — nil on a terminal error event.
func (r *AdminLogTailEventResolver) LogEntry() *AdminLogTailEntryResolver {
	if r.entry.Err != nil {
		return nil
	}

	return &AdminLogTailEntryResolver{entry: r.entry}
}

// Error resolver — the terminal stream error message, or nil for a normal entry.
func (r *AdminLogTailEventResolver) Error() *string {
	if r.entry.Err == nil {
		return nil
	}

	return new(r.entry.Err.Error())
}

// adminLogTailQuery returns the most recent buffered log entries, newest first.
func adminLogTailQuery(ctx context.Context, args *AdminLogTailQueryArgs) ([]*AdminLogTailEntryResolver, error) {
	if err := args.Validate(); err != nil {
		return nil, err
	}

	limit := defaultAdminLogTailLimit
	if args.Limit != nil && *args.Limit > 0 {
		limit = min(int(*args.Limit), maxQueryLimit)
	}

	var levels []string
	if args.Levels != nil {
		levels = *args.Levels
	}

	var search string
	if args.Search != nil {
		search = *args.Search
	}

	entries, err := getServiceCatalog(ctx).AdminLogTailService.GetEntries(ctx, &adminlogtail.GetEntriesInput{
		Levels: levels,
		Search: search,
		Limit:  limit,
	})
	if err != nil {
		return nil, err
	}

	resolvers := make([]*AdminLogTailEntryResolver, len(entries))
	for i, e := range entries {
		resolvers[i] = &AdminLogTailEntryResolver{entry: e}
	}

	return resolvers, nil
}

// adminLogTailEventsSubscription streams new log entries to live-tail subscribers.
func adminLogTailEventsSubscription(ctx context.Context, args *struct {
	Input AdminLogTailSubscriptionInput
}) (<-chan *AdminLogTailEventResolver, error) {
	if err := args.Input.Validate(); err != nil {
		return nil, err
	}

	var levels []string
	if args.Input.Levels != nil {
		levels = *args.Input.Levels
	}

	var search string
	if args.Input.Search != nil {
		search = *args.Input.Search
	}

	ch, err := getServiceCatalog(ctx).AdminLogTailService.Subscribe(ctx)
	if err != nil {
		return nil, err
	}

	outgoing := make(chan *AdminLogTailEventResolver)

	go func() {
		defer close(outgoing)

		for entry := range ch {
			// A terminal error event ends the stream: log it (the store has no logger of its
			// own) and forward it so the client surfaces the failure, then stop.
			if entry.Err != nil {
				getLogger(ctx).Errorf("admin log tail subscription stream error: %v", entry.Err)
				select {
				case <-ctx.Done():
				case outgoing <- &AdminLogTailEventResolver{entry: entry}:
				}
				return
			}
			if !entry.Matches(levels, search) {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case outgoing <- &AdminLogTailEventResolver{entry: entry}:
			}
		}
	}()

	return outgoing, nil
}
