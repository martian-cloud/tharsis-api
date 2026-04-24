package agent

import (
	"context"
	"testing"

	"github.com/m-mizutani/gollem"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCancellableStrategy_NotCancelledPassesThrough(t *testing.T) {
	baseCalled := false
	s := &cancellableStrategy{
		base: &fakeStrategy{handleFn: func(_ context.Context, _ *gollem.StrategyState) ([]gollem.Input, *gollem.ExecuteResponse, error) {
			baseCalled = true
			return nil, &gollem.ExecuteResponse{}, nil
		}},
		checkCancelled: func(_ context.Context) bool { return false },
	}

	_, resp, err := s.Handle(context.Background(), &gollem.StrategyState{Iteration: 1})
	require.Nil(t, err)
	assert.NotNil(t, resp)
	assert.True(t, baseCalled)
}

func TestCancellableStrategy_CancelledReturnsEmpty(t *testing.T) {
	baseCalled := false
	s := &cancellableStrategy{
		base: &fakeStrategy{handleFn: func(_ context.Context, _ *gollem.StrategyState) ([]gollem.Input, *gollem.ExecuteResponse, error) {
			baseCalled = true
			return nil, nil, nil
		}},
		checkCancelled: func(_ context.Context) bool { return true },
	}

	_, resp, err := s.Handle(context.Background(), &gollem.StrategyState{Iteration: 1})
	require.Nil(t, err)
	assert.NotNil(t, resp)
	assert.False(t, baseCalled, "base strategy should not be called when cancelled")
}

func TestCancellableStrategy_FirstIterationSkipsCancelCheck(t *testing.T) {
	baseCalled := false
	s := &cancellableStrategy{
		base: &fakeStrategy{handleFn: func(_ context.Context, _ *gollem.StrategyState) ([]gollem.Input, *gollem.ExecuteResponse, error) {
			baseCalled = true
			return nil, &gollem.ExecuteResponse{}, nil
		}},
		checkCancelled: func(_ context.Context) bool { return true },
	}

	// Iteration 0 should not check cancel
	_, _, err := s.Handle(context.Background(), &gollem.StrategyState{Iteration: 0})
	require.Nil(t, err)
	assert.True(t, baseCalled)
}

func TestCancellableStrategy_Init(t *testing.T) {
	initCalled := false
	s := &cancellableStrategy{
		base: &fakeStrategy{initFn: func(_ context.Context, _ []gollem.Input) error {
			initCalled = true
			return nil
		}},
	}

	err := s.Init(context.Background(), nil)
	require.Nil(t, err)
	assert.True(t, initCalled)
}

func TestCancellableStrategy_Tools(t *testing.T) {
	s := &cancellableStrategy{
		base: &fakeStrategy{toolsFn: func(_ context.Context) ([]gollem.Tool, error) {
			return []gollem.Tool{}, nil
		}},
	}

	tools, err := s.Tools(context.Background())
	require.Nil(t, err)
	assert.Empty(t, tools)
}

// fakeStrategy is a minimal gollem.Strategy for testing.
type fakeStrategy struct {
	initFn   func(ctx context.Context, inputs []gollem.Input) error
	handleFn func(ctx context.Context, state *gollem.StrategyState) ([]gollem.Input, *gollem.ExecuteResponse, error)
	toolsFn  func(ctx context.Context) ([]gollem.Tool, error)
}

func (f *fakeStrategy) Init(ctx context.Context, inputs []gollem.Input) error {
	if f.initFn != nil {
		return f.initFn(ctx, inputs)
	}
	return nil
}

func (f *fakeStrategy) Handle(ctx context.Context, state *gollem.StrategyState) ([]gollem.Input, *gollem.ExecuteResponse, error) {
	if f.handleFn != nil {
		return f.handleFn(ctx, state)
	}
	return nil, nil, nil
}

func (f *fakeStrategy) Tools(ctx context.Context) ([]gollem.Tool, error) {
	if f.toolsFn != nil {
		return f.toolsFn(ctx)
	}
	return nil, nil
}
