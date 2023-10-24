// Package loader package
package loader

import (
	"context"
	"fmt"

	"github.com/graph-gophers/dataloader"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// DataBatch type contains the results from the batch loader callback
type DataBatch map[string]interface{}

// BatchFunc is the function clients provide for creating loaders
type BatchFunc func(ctx context.Context, ids []string) (DataBatch, error)

// Key type is used for attaching loaders to the context
type key string

// Collection holds an internal lookup of initialized batch data load functions.
type Collection struct {
	batchFunctions map[string]dataloader.BatchFunc
}

// NewCollection creates an empty loader collection
func NewCollection() *Collection {
	return &Collection{batchFunctions: map[string]dataloader.BatchFunc{}}
}

// Register will register a new loader batch function
func (c Collection) Register(key string, callback BatchFunc) {
	c.batchFunctions[key] = newLoader(callback)
}

// Attach creates new instances of dataloader.Loader and attaches the instances on the request context.
func (c Collection) Attach(ctx context.Context, opts ...dataloader.Option) context.Context {
	for k, batchFn := range c.batchFunctions {
		ctx = context.WithValue(ctx, key(k), dataloader.NewBatchedLoader(batchFn, opts...))
	}
	return ctx
}

type loader struct {
	batchFunc BatchFunc
}

func newLoader(batchFunc BatchFunc) dataloader.BatchFunc {
	return loader{batchFunc: batchFunc}.loadBatch
}

func (ldr loader) loadBatch(ctx context.Context, keys dataloader.Keys) []*dataloader.Result {
	var (
		idList   = getKeysAsStrings(keys)
		keyCount = len(keys)
		results  = make([]*dataloader.Result, keyCount)
	)

	batch, err := ldr.batchFunc(ctx, idList)
	if err != nil {
		return buildErrorResults(results, err)
	}

	for i, id := range idList {
		results[i] = &dataloader.Result{}

		data, found := batch[id]
		if !found {
			results[i].Error = errors.New("resource with ID %s not found", id, errors.WithErrorCode(errors.ENotFound))
		}
		results[i].Data = data
	}

	return results
}

func (k key) String() string {
	return fmt.Sprintf("gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/api/graphql/loader %s", string(k))
}

// Extract will extract the loader from the context
func Extract(ctx context.Context, k string) (*dataloader.Loader, error) {
	ldr, ok := ctx.Value(key(k)).(*dataloader.Loader)
	if !ok {
		return nil, fmt.Errorf("unable to find %s loader on the request context", k)
	}

	return ldr, nil
}

func getKeysAsStrings(keys dataloader.Keys) []string {
	values := make([]string, len(keys))
	for i, key := range keys {
		values[i] = key.String()
	}
	return values
}

func buildErrorResults(results []*dataloader.Result, err error) []*dataloader.Result {
	for i := range results {
		results[i] = &dataloader.Result{Error: err}
	}
	return results
}
