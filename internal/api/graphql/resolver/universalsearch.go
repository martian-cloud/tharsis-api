package resolver

import (
	"context"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/universalsearch"
)

// SearchArgs represents the arguments for a search query
type SearchArgs struct {
	Query string
}

// SearchResponseResolver resolves search response data
type SearchResponseResolver struct {
	response *universalsearch.SearchResponse
}

// Search performs a search across multiple resource types
func (r *RootResolver) Search(ctx context.Context, args *SearchArgs) (*SearchResponseResolver, error) {
	// Get cached universal search Service
	searchManager := getUniversalSearchManager(ctx)

	// Execute search - search all resource types since no filtering
	request := universalsearch.SearchRequest{
		Query: args.Query,
	}

	response, err := searchManager.Search(ctx, request)
	if err != nil {
		return nil, err
	}

	return &SearchResponseResolver{
		response: response,
	}, nil
}

// Results returns the search results as Node interface implementations
func (r *SearchResponseResolver) Results() []*NodeResolver {
	nodes := make([]*NodeResolver, len(r.response.Results))
	for i, result := range r.response.Results {
		nodes[i] = &NodeResolver{result: result.Data}
	}
	return nodes
}
