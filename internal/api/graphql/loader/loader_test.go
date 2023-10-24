package loader

import (
	"context"
	"testing"

	"github.com/graph-gophers/dataloader"
	"github.com/stretchr/testify/assert"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestLoaderBatchFunc(t *testing.T) {
	// Test cases
	tests := []struct {
		batchErr      error
		batchResponse DataBatch
		name          string
		expectErrCode errors.CodeType
		keys          []string
		expectResults []dataloader.Result
	}{
		{
			name:          "load batch with no errors and multiple keys",
			keys:          []string{"key1", "key2"},
			batchResponse: DataBatch{"key1": "r1", "key2": "r2"},
			expectResults: []dataloader.Result{{Data: "r1"}, {Data: "r2"}},
		},
		{
			name:          "load batch with missing data",
			keys:          []string{"key1", "key2"},
			batchResponse: DataBatch{"key1": "r1"},
			expectResults: []dataloader.Result{{Data: "r1"}, {Error: errors.New("resource with ID key2 not found", errors.WithErrorCode(errors.ENotFound))}},
		},
		{
			name:          "load batch with single key",
			keys:          []string{"key1"},
			batchResponse: DataBatch{"key1": "r1"},
			expectResults: []dataloader.Result{{Data: "r1"}},
		},
		{
			name:          "load batch with error response",
			keys:          []string{"key1"},
			batchResponse: DataBatch{"key1": "r1"},
			batchErr:      errors.New("Failed to execute batch function", errors.WithErrorCode(errors.ENotFound)),
			expectResults: []dataloader.Result{{Error: errors.New("Failed to execute batch function", errors.WithErrorCode(errors.ENotFound))}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			batchFunc := newLoader(func(ctx context.Context, ids []string) (DataBatch, error) {
				return test.batchResponse, test.batchErr
			})

			keys := dataloader.NewKeysFromStrings(test.keys)
			results := batchFunc(ctx, keys)

			assert.Equal(t, len(test.keys), len(results))
			for i, result := range results {
				assert.Equal(t, test.expectResults[i].Data, result.Data)
				assert.Equal(t, test.expectResults[i].Error, result.Error)
			}
		})
	}
}
