package client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/aws/smithy-go/ptr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// discoveryServer serves the gRPC discovery document at ServiceDiscoveryPath so tests
// hit a loopback httptest server instead of a real endpoint.
func discoveryServer(t *testing.T, statusCode int, body string) *httptest.Server {
	t.Helper()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, ServiceDiscoveryPath, r.URL.Path)
		w.WriteHeader(statusCode)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	return srv
}

// fakeStream implements grpc.ServerStreamingClient by stubbing only Recv; the
// embedded nil ClientStream satisfies the rest of the interface at compile time.
type fakeStream[T any] struct {
	grpc.ClientStream
	recv func() (*T, error)
}

func (f *fakeStream[T]) Recv() (*T, error) {
	return f.recv()
}

// recvScript returns a Recv func that yields each msg in order, then finalErr.
func recvScript[T any](msgs []*T, finalErr error) func() (*T, error) {
	i := 0
	return func() (*T, error) {
		if i < len(msgs) {
			m := msgs[i]
			i++
			return m, nil
		}
		return nil, finalErr
	}
}

// fastRetry shortens the backoff so reconnect tests don't sleep.
func fastRetry() []retry.Option {
	return []retry.Option{retry.Delay(time.Millisecond), retry.MaxDelay(5 * time.Millisecond)}
}

func TestStreamWithReconnect(t *testing.T) {
	t.Run("delivers messages then returns nil on EOF", func(t *testing.T) {
		opens := 0
		open := func(_ context.Context) (grpc.ServerStreamingClient[int], error) {
			opens++
			return &fakeStream[int]{recv: recvScript([]*int{ptr.Int(1), ptr.Int(2)}, io.EOF)}, nil
		}

		var got []int
		err := StreamWithReconnect(t.Context(), open, func(v *int) (bool, error) {
			got = append(got, *v)
			return false, nil
		}, fastRetry()...)

		require.NoError(t, err)
		assert.Equal(t, []int{1, 2}, got)
		assert.Equal(t, 1, opens)
	})

	t.Run("reconnects on Unavailable", func(t *testing.T) {
		opens := 0
		open := func(_ context.Context) (grpc.ServerStreamingClient[int], error) {
			opens++
			if opens == 1 {
				return &fakeStream[int]{recv: recvScript([]*int{ptr.Int(1)}, status.Error(codes.Unavailable, "draining"))}, nil
			}
			return &fakeStream[int]{recv: recvScript([]*int{ptr.Int(2)}, io.EOF)}, nil
		}

		reconnects := 0
		var got []int
		err := StreamWithReconnect(t.Context(), open, func(v *int) (bool, error) {
			got = append(got, *v)
			return false, nil
		}, append(fastRetry(), retry.OnRetry(func(uint, error) { reconnects++ }))...)

		require.NoError(t, err)
		assert.Equal(t, []int{1, 2}, got)
		assert.Equal(t, 2, opens)
		assert.Equal(t, 1, reconnects)
	})

	t.Run("returns non-reconnectable stream error without reconnecting", func(t *testing.T) {
		opens := 0
		// NotFound is not in reconnectableStreamCodes, so it must surface immediately.
		open := func(_ context.Context) (grpc.ServerStreamingClient[int], error) {
			opens++
			return &fakeStream[int]{recv: recvScript[int](nil, status.Error(codes.NotFound, "gone"))}, nil
		}

		err := StreamWithReconnect(t.Context(), open, func(*int) (bool, error) { return false, nil }, fastRetry()...)

		require.Error(t, err)
		assert.Equal(t, codes.NotFound, status.Code(err))
		assert.Equal(t, 1, opens)
	})

	t.Run("stops reconnecting when context is cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		opens := 0
		open := func(_ context.Context) (grpc.ServerStreamingClient[int], error) {
			opens++
			cancel()
			return &fakeStream[int]{recv: recvScript[int](nil, status.Error(codes.Unavailable, "draining"))}, nil
		}

		err := StreamWithReconnect(ctx, open, func(*int) (bool, error) { return false, nil }, fastRetry()...)

		assert.ErrorIs(t, err, context.Canceled)
		assert.Equal(t, 1, opens)
	})

	t.Run("returns handle error without retrying", func(t *testing.T) {
		open := func(_ context.Context) (grpc.ServerStreamingClient[int], error) {
			return &fakeStream[int]{recv: recvScript([]*int{ptr.Int(1)}, io.EOF)}, nil
		}

		err := StreamWithReconnect(t.Context(), open, func(*int) (bool, error) { return false, assert.AnError }, fastRetry()...)

		assert.ErrorIs(t, err, assert.AnError)
	})

	t.Run("stops when handle reports done", func(t *testing.T) {
		opens := 0
		open := func(_ context.Context) (grpc.ServerStreamingClient[int], error) {
			opens++
			return &fakeStream[int]{recv: recvScript([]*int{ptr.Int(1), ptr.Int(2)}, status.Error(codes.Unavailable, "x"))}, nil
		}

		var got []int
		err := StreamWithReconnect(t.Context(), open, func(v *int) (bool, error) {
			got = append(got, *v)
			return true, nil
		}, fastRetry()...)

		require.NoError(t, err)
		assert.Equal(t, []int{1}, got)
		assert.Equal(t, 1, opens)
	})
}

func TestNewGRPCDiscoveryDocument(t *testing.T) {
	t.Run("parses the discovery document", func(t *testing.T) {
		srv := discoveryServer(t, http.StatusOK, `{"grpc":{"host":"grpc.example.com","transport_security":"tls","port":"443"}}`)

		doc, err := NewGRPCDiscoveryDocument(t.Context(), srv.URL)
		require.NoError(t, err)
		assert.Equal(t, "grpc.example.com", doc.Host)
		assert.Equal(t, "443", doc.Port)
		assert.True(t, doc.HasTransportSecurity())
	})

	t.Run("errors on non-200 status", func(t *testing.T) {
		srv := discoveryServer(t, http.StatusNotFound, "")

		_, err := NewGRPCDiscoveryDocument(t.Context(), srv.URL)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "404")
	})

	t.Run("errors on malformed body", func(t *testing.T) {
		srv := discoveryServer(t, http.StatusOK, "not json")

		_, err := NewGRPCDiscoveryDocument(t.Context(), srv.URL)
		assert.Error(t, err)
	})
}

func TestNewGRPCClient(t *testing.T) {
	// Plaintext transport so the lazy gRPC connection needs no TLS and no real dial.
	srv := discoveryServer(t, http.StatusOK, `{"grpc":{"host":"localhost","transport_security":"plaintext","port":"50051"}}`)

	client, err := NewGRPCClient(t.Context(), &GRPCClientConfig{
		HTTPEndpoint:  srv.URL,
		TokenResolver: &mockTokenResolver{token: "test"},
	})
	require.NoError(t, err)
	require.NotNil(t, client)
	assert.NotNil(t, client.RunsClient)
	assert.NotNil(t, client.WorkspacesClient)
	assert.NoError(t, client.Close())
}

func TestGRPCDiscoveryDocument_HasTransportSecurity(t *testing.T) {
	assert.False(t, (&GRPCDiscoveryDocument{TransportSecurity: "plaintext"}).HasTransportSecurity())
	assert.True(t, (&GRPCDiscoveryDocument{TransportSecurity: "tls"}).HasTransportSecurity())
}

func TestContextCredentials(t *testing.T) {
	t.Run("sets the authorization metadata from the resolver", func(t *testing.T) {
		creds := &contextCredentials{resolver: &mockTokenResolver{token: "abc"}}

		md, err := creds.GetRequestMetadata(t.Context())
		require.NoError(t, err)
		assert.Equal(t, "abc", md["authorization"])
		assert.False(t, creds.RequireTransportSecurity())
	})

	t.Run("propagates a resolver error", func(t *testing.T) {
		creds := &contextCredentials{resolver: &mockTokenResolver{err: assert.AnError}}

		_, err := creds.GetRequestMetadata(t.Context())
		assert.ErrorIs(t, err, assert.AnError)
	})
}
