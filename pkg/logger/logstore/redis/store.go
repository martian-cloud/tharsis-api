// Package redis implements a Redis Stream-backed log store.
package redis

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	redigo "github.com/gomodule/redigo/redis"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger/logstore"
)

const (
	defaultKeyPrefix  = "admin:logtail:stream"
	defaultMaxEntries = 2000
	writeBufferSize   = 512
	subscribeChanSize = 256
)

// store is the Redis Stream-backed implementation of logstore.Store.
type store struct {
	pool       *redigo.Pool
	endpoint   string
	streamKey  string
	maxEntries int
	writeCh    chan *logstore.LogEntry
}

// New creates a Redis-backed Store; the background writer stops on ctx.
// pluginData: redis_endpoint (required), key_prefix (default "admin:logtail:stream"), max_entries (default 2000).
func New(ctx context.Context, pluginData map[string]string) (logstore.Store, error) {
	endpoint, ok := pluginData["redis_endpoint"]
	if !ok {
		return nil, fmt.Errorf("'redis_endpoint' is required for the redis log store plugin")
	}

	keyPrefix := defaultKeyPrefix
	if v := pluginData["key_prefix"]; v != "" {
		keyPrefix = v
	}

	maxEntries := defaultMaxEntries
	if v := pluginData["max_entries"]; v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n <= 0 {
			return nil, fmt.Errorf("'max_entries' must be a positive integer, got %q", v)
		}
		maxEntries = n
	}

	s := &store{
		endpoint:   endpoint,
		streamKey:  keyPrefix,
		maxEntries: maxEntries,
		writeCh:    make(chan *logstore.LogEntry, writeBufferSize),
	}

	// Pool serves short GetEntries reads; blocking live-tail reads dial their own connection.
	s.pool = &redigo.Pool{
		MaxIdle:   10,
		MaxActive: 100,
		Dial:      s.dial,
	}

	go s.drainWrites(ctx)

	return s, nil
}

func (s *store) dial() (redigo.Conn, error) {
	return redigo.DialURL(s.endpoint, redigo.DialConnectTimeout(10*time.Second))
}

// sleepOrDone waits for d or ctx cancellation (so reconnect backoffs honor shutdown).
func sleepOrDone(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		return false
	case <-time.After(d):
		return true
	}
}

// Write buffers the entry for async delivery to Redis.
func (s *store) Write(e *logstore.LogEntry) {
	select {
	case s.writeCh <- e:
	default:
		// drop if buffer full (same non-blocking contract as the in-memory store)
	}
}

// GetEntries returns up to limit entries newest-first, with optional level/search filtering.
func (s *store) GetEntries(levels []string, search string, limit int) ([]*logstore.LogEntry, error) {
	if limit <= 0 {
		return nil, nil
	}

	conn := s.pool.Get()
	defer conn.Close()

	// The stream is capped at maxEntries, so a single newest-first read covers the
	// whole buffer; filtering then happens in Go.
	values, err := redigo.Values(conn.Do("XREVRANGE", s.streamKey, "+", "-", "COUNT", s.maxEntries))
	if err != nil {
		return nil, fmt.Errorf("failed to read log entries from redis stream: %w", err)
	}

	result := make([]*logstore.LogEntry, 0, limit)

	for _, v := range values {
		id, fields, ok := streamEntryFields(v)
		if !ok {
			continue
		}

		e := parseStreamEntry(id, fields)
		if e == nil || !e.Matches(levels, search) {
			continue
		}

		result = append(result, e)

		if len(result) >= limit {
			break
		}
	}

	return result, nil
}

// Subscribe returns a channel that receives new log entries in real time. The
// stream stops (and the channel closes) when ctx is cancelled or a read error
// occurs. An error is returned if the stream cannot be reached when the
// subscription starts.
func (s *store) Subscribe(ctx context.Context) (<-chan *logstore.LogEntry, error) {
	// Dedicated connection (not pooled) so a long blocking read never consumes pool capacity.
	conn, err := s.dial()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to redis log stream: %w", err)
	}

	// Seed lastID from the stream end so we only stream new entries; doing it synchronously
	// also surfaces connectivity errors.
	vals, err := redigo.Values(conn.Do("XREVRANGE", s.streamKey, "+", "-", "COUNT", 1))
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to read from redis log stream: %w", err)
	}

	lastID := "0"
	if len(vals) > 0 {
		if id, _, ok := streamEntryFields(vals[0]); ok {
			lastID = id
		}
	}

	ch := make(chan *logstore.LogEntry, subscribeChanSize)

	go func() {
		defer close(ch)
		defer conn.Close()

		for ctx.Err() == nil {
			// DoContext lets ctx cancellation interrupt the blocking read; BLOCK also bounds
			// the read so an idle stream doesn't hold the connection indefinitely.
			reply, err := redigo.Values(redigo.DoContext(conn, ctx, "XREAD", "BLOCK", "5000", "COUNT", "50", "STREAMS", s.streamKey, lastID))
			if err == redigo.ErrNil {
				continue
			}

			// On a read error, deliver a terminal error event and end the stream rather than
			// reconnecting under the client. ctx.Err() guards against emitting on shutdown,
			// where DoContext also returns an error.
			if err != nil {
				if ctx.Err() == nil {
					select {
					case ch <- &logstore.LogEntry{Err: fmt.Errorf("log stream connection lost: %w", err)}:
					case <-ctx.Done():
					}
				}
				return
			}

			for _, streamRaw := range reply {
				stream, ok := streamRaw.([]any)
				if !ok || len(stream) != 2 {
					continue
				}

				entries, ok := stream[1].([]any)
				if !ok {
					continue
				}

				for _, entryRaw := range entries {
					id, fields, ok := streamEntryFields(entryRaw)
					if !ok {
						continue
					}

					e := parseStreamEntry(id, fields)
					if e == nil {
						continue
					}

					select {
					case ch <- e:
						lastID = id
					case <-ctx.Done():
						return
					default:
						// subscriber too slow — drop entry; do not advance lastID so
						// the entry can be replayed on reconnect
					}
				}
			}
		}
	}()

	return ch, nil
}

func (s *store) drainWrites(ctx context.Context) {
	conn := s.pool.Get()
	defer func() { conn.Close() }()

	for {
		select {
		case <-ctx.Done():
			return
		case e, ok := <-s.writeCh:
			if !ok {
				return
			}
			if err := s.writeToStream(conn, e); err != nil {
				conn.Close()
				if !sleepOrDone(ctx, time.Second) { // avoid a dial per dropped entry while redis is unavailable
					return
				}
				conn = s.pool.Get()
			}
		}
	}
}

func (s *store) writeToStream(conn redigo.Conn, e *logstore.LogEntry) error {
	_, err := conn.Do("XADD", s.streamKey,
		"MAXLEN", "~", s.maxEntries,
		"*",
		"level", e.Level,
		"message", e.Message,
		"caller", e.Caller,
		"stack", e.Stack,
		"fields", e.Fields,
		"timestamp", e.Timestamp.UTC().Format(time.RFC3339Nano),
	)
	return err
}

// streamIDToSeq encodes a Redis stream ID "{ms}-{n}" as a bijective uint64.
// Using ms<<20|n reserves 20 bits for the per-ms sequence (>1M entries/ms),
// avoiding the collision in ms*1000+n when n≥1000.
func streamIDToSeq(id string) uint64 {
	ms, n, ok := parseStreamID(id)
	if !ok {
		return 0
	}

	return ms<<20 | n
}

func parseStreamID(id string) (ms, n uint64, ok bool) {
	msStr, nStr, found := strings.Cut(id, "-")
	if !found {
		return 0, 0, false
	}

	ms, err := strconv.ParseUint(msStr, 10, 64)
	if err != nil {
		return 0, 0, false
	}

	n, err = strconv.ParseUint(nStr, 10, 64)
	if err != nil {
		return 0, 0, false
	}

	return ms, n, true
}

// streamEntryFields splits a stream entry [id, [field, value, ...]] into id and fields.
func streamEntryFields(v any) (id string, fields []any, ok bool) {
	pair, ok := v.([]any)
	if !ok || len(pair) != 2 {
		return "", nil, false
	}

	rawID, ok := pair[0].([]byte)
	if !ok {
		return "", nil, false
	}

	fields, ok = pair[1].([]any)
	if !ok {
		return "", nil, false
	}

	return string(rawID), fields, true
}

func parseStreamEntry(id string, fields []any) *logstore.LogEntry {
	m := make(map[string]string, len(fields)/2)

	for i := 0; i+1 < len(fields); i += 2 {
		k, ok1 := fields[i].([]byte)
		v, ok2 := fields[i+1].([]byte)

		if ok1 && ok2 {
			m[string(k)] = string(v)
		}
	}

	level := m["level"]
	message := m["message"]

	if level == "" && message == "" {
		return nil
	}

	e := &logstore.LogEntry{
		Seq:     streamIDToSeq(id),
		Level:   level,
		Message: message,
		Caller:  m["caller"],
		Stack:   m["stack"],
		Fields:  m["fields"],
	}

	if ts, err := time.Parse(time.RFC3339Nano, m["timestamp"]); err == nil {
		e.Timestamp = ts
	}

	return e
}
