package jobexecutor

import (
	"context"
	"errors"
)

func isCancellationError(err error) bool {
	return err != nil && (errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded))
}
