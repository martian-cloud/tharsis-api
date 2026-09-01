package cleanup

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
)

// fakeCandidate is a minimal stand-in for a *models.X row in deleteInChunks tests.
type fakeCandidate struct {
	id  string
	trn string
}

func makeFakeCandidates(n int) []fakeCandidate {
	out := make([]fakeCandidate, n)
	for i := range out {
		out[i] = fakeCandidate{id: fmt.Sprintf("id%d", i), trn: fmt.Sprintf("trn%d", i)}
	}
	return out
}

func idsOf(cs []fakeCandidate) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.id
	}
	return out
}

func trnsOf(cs []fakeCandidate) []string {
	out := make([]string, len(cs))
	for i, c := range cs {
		out[i] = c.trn
	}
	return out
}

func TestDeleteInChunks(t *testing.T) {
	ctx := t.Context()
	errBoom := errors.New("boom")

	idOf := func(c fakeCandidate) string { return c.id }
	trnOf := func(c fakeCandidate) string { return c.trn }

	c100 := makeFakeCandidates(100)
	c101 := makeFakeCandidates(101)
	c200 := makeFakeCandidates(200)

	tests := []struct {
		name        string
		candidates  []fakeCandidate
		failOnBatch int // 0-indexed; -1 = never fail
		wantErr     error
		wantBatches [][]string
	}{
		{
			name:        "empty: deleteBatch never called",
			candidates:  nil,
			failOnBatch: -1,
		},
		{
			name:        "single candidate: one batch",
			candidates:  makeFakeCandidates(1),
			failOnBatch: -1,
			wantBatches: [][]string{idsOf(makeFakeCandidates(1))},
		},
		{
			name:        "exactly 100: one batch",
			candidates:  c100,
			failOnBatch: -1,
			wantBatches: [][]string{idsOf(c100)},
		},
		{
			name:        "101: two batches",
			candidates:  c101,
			failOnBatch: -1,
			wantBatches: [][]string{idsOf(c101[:100]), idsOf(c101[100:])},
		},
		{
			name:        "200: two batches of 100",
			candidates:  c200,
			failOnBatch: -1,
			wantBatches: [][]string{idsOf(c200[:100]), idsOf(c200[100:])},
		},
		{
			name:        "error on first chunk: second never called",
			candidates:  c101,
			failOnBatch: 0,
			wantErr:     errBoom,
			wantBatches: [][]string{idsOf(c101[:100])},
		},
		{
			name:        "error on second chunk: first succeeds",
			candidates:  c101,
			failOnBatch: 1,
			wantErr:     errBoom,
			wantBatches: [][]string{idsOf(c101[:100]), idsOf(c101[100:])},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var gotBatches [][]string
			var gotTRNs []string
			call := 0
			deleteBatch := func(_ context.Context, chunk []fakeCandidate) ([]string, error) {
				ids := idsOf(chunk)
				gotBatches = append(gotBatches, ids)
				n := call
				call++
				if tc.failOnBatch >= 0 && n == tc.failOnBatch {
					return nil, tc.wantErr
				}
				return ids, nil
			}

			err := deleteInChunks(ctx, tc.candidates, idOf, trnOf, deleteBatch, func(trns ...string) {
				gotTRNs = append(gotTRNs, trns...)
			})

			if tc.wantErr != nil {
				require.ErrorIs(t, err, tc.wantErr)
			} else {
				require.NoError(t, err)
				assert.ElementsMatch(t, trnsOf(tc.candidates), gotTRNs)
			}
			assert.Equal(t, tc.wantBatches, gotBatches)
		})
	}

	t.Run("deleteBatch returns subset with no error — onDelete receives only deleted TRNs", func(t *testing.T) {
		candidates := makeFakeCandidates(3) // id0/trn0, id1/trn1, id2/trn2

		deleteBatch := func(_ context.Context, chunk []fakeCandidate) ([]string, error) {
			return []string{chunk[0].id, chunk[2].id}, nil
		}

		var gotTRNs []string
		err := deleteInChunks(ctx, candidates, idOf, trnOf, deleteBatch, func(trns ...string) {
			gotTRNs = append(gotTRNs, trns...)
		})

		require.NoError(t, err)
		assert.Equal(t, []string{"trn0", "trn2"}, gotTRNs)
	})

	t.Run("ErrOptimisticLockError is not fatal — onDelete gets the survivors, no error returned", func(t *testing.T) {
		candidates := makeFakeCandidates(3) // id0/trn0, id1/trn1, id2/trn2

		deleteBatch := func(_ context.Context, chunk []fakeCandidate) ([]string, error) {
			// id1 lost its optimistic lock race; id0 and id2 still deleted.
			return []string{chunk[0].id, chunk[2].id}, db.ErrOptimisticLockError
		}

		var gotTRNs []string
		err := deleteInChunks(ctx, candidates, idOf, trnOf, deleteBatch, func(trns ...string) {
			gotTRNs = append(gotTRNs, trns...)
		})

		require.NoError(t, err, "ErrOptimisticLockError must not be propagated")
		assert.Equal(t, []string{"trn0", "trn2"}, gotTRNs)
	})

	t.Run("non-OLE error on a later chunk still stops the pass", func(t *testing.T) {
		candidates := c101 // two chunks

		call := 0
		deleteBatch := func(_ context.Context, chunk []fakeCandidate) ([]string, error) {
			n := call
			call++
			if n == 1 {
				return nil, errBoom
			}
			return idsOf(chunk), nil
		}

		var gotTRNs []string
		err := deleteInChunks(ctx, candidates, idOf, trnOf, deleteBatch, func(trns ...string) {
			gotTRNs = append(gotTRNs, trns...)
		})

		require.ErrorIs(t, err, errBoom)
		assert.Equal(t, trnsOf(c101[:100]), gotTRNs, "first chunk's TRNs still reported before the fatal error")
	})
}
