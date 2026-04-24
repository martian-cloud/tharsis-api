package agent

import (
	"context"

	"github.com/m-mizutani/gollem"
)

// cancellableStrategy wraps a base strategy and checks a cancel condition on each iteration.
// If cancelled, it returns an empty ExecuteResponse to gracefully stop the execution loop.
type cancellableStrategy struct {
	base           gollem.Strategy
	checkCancelled func(ctx context.Context) bool
}

func (s *cancellableStrategy) Init(ctx context.Context, inputs []gollem.Input) error {
	return s.base.Init(ctx, inputs)
}

func (s *cancellableStrategy) Handle(ctx context.Context, state *gollem.StrategyState) ([]gollem.Input, *gollem.ExecuteResponse, error) {
	if state.Iteration > 0 && s.checkCancelled(ctx) {
		return nil, &gollem.ExecuteResponse{}, nil
	}
	return s.base.Handle(ctx, state)
}

func (s *cancellableStrategy) Tools(ctx context.Context) ([]gollem.Tool, error) {
	return s.base.Tools(ctx)
}
