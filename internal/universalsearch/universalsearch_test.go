package universalsearch

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// Test 1: Empty query returns empty results
func TestSearchManager_Search_EmptyQuery(t *testing.T) {
	logger, _ := logger.NewForTest()
	searchManager := newManager(&services.Catalog{}, logger, make(map[types.ModelType]searchFunc))

	request := SearchRequest{
		Query: "", // ← Testing this edge case
	}

	response, err := searchManager.Search(context.Background(), request)

	// Verify empty query behavior
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Empty(t, response.Results)
}

// Test 2: No searchers used (empty resource types filter)
func TestSearchManager_Search_NoSearchers(t *testing.T) {
	// What it tests: SearchManager behavior when no searchers are selected via empty resource types
	logger, _ := logger.NewForTest()
	searchManager := newManager(&services.Catalog{}, logger, make(map[types.ModelType]searchFunc))

	request := SearchRequest{
		Query: "test",
	}

	response, err := searchManager.Search(context.Background(), request)

	// Should return empty results, not error
	assert.NoError(t, err)
	assert.Empty(t, response.Results)
}

// Test 3: Successful search with direct function searchers
func TestSearchManager_Search_WithDirectSearchers(t *testing.T) {
	// What it tests: Actual search execution with direct functions

	searchers := make(map[types.ModelType]searchFunc)

	// Create direct searcher functions - CHANGED: Using []*SearchResult
	groupSearcher := func(ctx context.Context, query string, limit int32) ([]*SearchResult, error) {
		return []*SearchResult{
			{Data: &models.Group{Metadata: models.ResourceMetadata{ID: "group1"}}, Type: types.GroupModelType},
			{Data: &models.Group{Metadata: models.ResourceMetadata{ID: "group2"}}, Type: types.GroupModelType},
		}, nil
	}

	workspaceSearcher := func(ctx context.Context, query string, limit int32) ([]*SearchResult, error) {
		return []*SearchResult{
			{Data: &models.Workspace{Metadata: models.ResourceMetadata{ID: "workspace1"}}, Type: types.WorkspaceModelType},
		}, nil
	}

	// Register searchers
	searchers[types.GroupModelType] = groupSearcher
	searchers[types.WorkspaceModelType] = workspaceSearcher

	logger, _ := logger.NewForTest()

	searchManager := newManager(&services.Catalog{}, logger, searchers)

	// Execute test
	request := SearchRequest{
		Query: "test",
	}

	response, err := searchManager.Search(context.Background(), request)

	// Verify results
	assert.NoError(t, err)
	assert.Len(t, response.Results, 3)

	// Verify actual result content
	resultIDs := make([]string, len(response.Results))
	for i, result := range response.Results {
		resultIDs[i] = result.Data.GetID()
	}
	assert.Contains(t, resultIDs, "group1")
	assert.Contains(t, resultIDs, "group2")
	assert.Contains(t, resultIDs, "workspace1")
}

// Test 4: Optimal slot allocation logic
func TestSearchManager_CalculateOptimalSlots(t *testing.T) {
	tests := []struct {
		name          string
		resultsByType map[types.ModelType][]*SearchResult
		expectedSlots map[types.ModelType]int32
	}{
		{
			name:          "No results",
			resultsByType: map[types.ModelType][]*SearchResult{},
			expectedSlots: map[types.ModelType]int32{},
		},
		{
			name: "Results fit within limit",
			resultsByType: map[types.ModelType][]*SearchResult{
				types.GroupModelType:     {{}, {}},     // 2 results
				types.WorkspaceModelType: {{}, {}, {}}, // 3 results
			},
			expectedSlots: map[types.ModelType]int32{
				types.GroupModelType:     2,
				types.WorkspaceModelType: 3,
			},
		},
		{
			name: "Even distribution needed",
			resultsByType: map[types.ModelType][]*SearchResult{
				types.GroupModelType:     {{}, {}, {}, {}, {}}, // 5 results
				types.WorkspaceModelType: {{}, {}, {}, {}, {}}, // 5 results
			},
			expectedSlots: map[types.ModelType]int32{
				types.GroupModelType:     5, // All 5 results (under defaultSearchLimit, so no constraint)
				types.WorkspaceModelType: 5, // All 5 results
			},
		},
		{
			name: "No redistribution needed",
			resultsByType: map[types.ModelType][]*SearchResult{
				types.GroupModelType:     make([]*SearchResult, 10), // 10 results
				types.WorkspaceModelType: make([]*SearchResult, 5),  // 5 results
				types.RunModelType:       make([]*SearchResult, 5),  // 5 results
			},
			expectedSlots: map[types.ModelType]int32{
				types.GroupModelType:     10, // All 10 results (fits in limit)
				types.WorkspaceModelType: 5,  // All 5 results
				types.RunModelType:       5,  // All 5 results (limited by available)
			},
		},
		{
			name: "Redistribution occurs",
			resultsByType: map[types.ModelType][]*SearchResult{
				types.GroupModelType:     make([]*SearchResult, 2),  // 2 results (limited)
				types.WorkspaceModelType: make([]*SearchResult, 10), // 10 results
				types.RunModelType:       make([]*SearchResult, 10), // 10 results
			},
			expectedSlots: map[types.ModelType]int32{
				types.GroupModelType:     2,  // All 2 results
				types.RunModelType:       10, // Gets redistributed slots (alphabetically first)
				types.WorkspaceModelType: 8,  // Gets remaining slots
			},
		},
		{
			name: "Partial redistribution",
			resultsByType: map[types.ModelType][]*SearchResult{
				types.GroupModelType:     make([]*SearchResult, 2),  // 2 results (limited)
				types.WorkspaceModelType: make([]*SearchResult, 3),  // 3 results (limited)
				types.RunModelType:       make([]*SearchResult, 10), // 10 results
			},
			expectedSlots: map[types.ModelType]int32{
				types.GroupModelType:     2,  // All 2 results
				types.WorkspaceModelType: 3,  // All 3 results
				types.RunModelType:       10, // Gets all remaining slots (defaultSearchLimit-2-3=15, but limited to 10)
			},
		}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger, _ := logger.NewForTest()
			searchManager := newManager(&services.Catalog{}, logger, make(map[types.ModelType]searchFunc))
			result := searchManager.calculateOptimalSlots(tt.resultsByType)
			assert.Equal(t, tt.expectedSlots, result)
		})
	}
}
