package universalsearch

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const (
	searchTimeout      = 60 * time.Second
	defaultSearchLimit = int32(20)
)

// SearchResult represents a single search result with its data and type.
type SearchResult struct {
	Data models.Model
	Type types.ModelType
}

// SearchRequest represents a search query request.
type SearchRequest struct {
	Query string
}

// SearchResponse contains the results of a search operation.
type SearchResponse struct {
	Results []*SearchResult
}

// searchFunc defines the search function type
type searchFunc func(ctx context.Context, query string, limit int32) ([]*SearchResult, error)

// resourceTypeOrder defines the display order of resource types in search results
var resourceTypeOrder = []types.ModelType{
	types.NamespaceFavoriteModelType,
	types.GroupModelType,
	types.WorkspaceModelType,
	types.TeamModelType,
	types.TerraformModuleModelType,
	types.TerraformProviderModelType,
}

// Manager defines the main universal search interface
type Manager interface {
	Search(ctx context.Context, request SearchRequest) (*SearchResponse, error)
}

type manager struct {
	catalog    *services.Catalog
	searchers  map[types.ModelType]searchFunc
	modelTypes []types.ModelType
	logger     logger.Logger
}

// NewManager creates a new universal search manager.
func NewManager(catalog *services.Catalog, logger logger.Logger) Manager {
	return newManager(catalog, logger, map[types.ModelType]searchFunc{
		types.NamespaceFavoriteModelType: favoriteSearcher(catalog.UserService),
		types.GroupModelType:             groupSearcher(catalog.GroupService),
		types.WorkspaceModelType:         workspaceSearcher(catalog.WorkspaceService),
		types.TeamModelType:              teamSearcher(catalog.TeamService),
		types.TerraformModuleModelType:   moduleSearcher(catalog.TerraformModuleRegistryService),
		types.TerraformProviderModelType: providerSearcher(catalog.TerraformProviderRegistryService),
	})
}

func newManager(catalog *services.Catalog, logger logger.Logger, searchers map[types.ModelType]searchFunc) *manager {
	// Build ordered list of model types for consistent searcher iteration
	modelTypes := make([]types.ModelType, 0, len(searchers))
	for modelType := range searchers {
		modelTypes = append(modelTypes, modelType)
	}
	sort.Slice(modelTypes, func(i, j int) bool {
		return modelTypes[i].Name() < modelTypes[j].Name()
	})

	return &manager{
		catalog:    catalog,
		logger:     logger,
		searchers:  searchers,
		modelTypes: modelTypes,
	}
}

func (s *manager) Search(ctx context.Context, request SearchRequest) (*SearchResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, searchTimeout)
	defer cancel()

	// For empty query, only display favorites
	if request.Query == "" {
		favoriteSearcher, ok := s.searchers[types.NamespaceFavoriteModelType]
		if !ok {
			return &SearchResponse{Results: []*SearchResult{}}, nil
		}
		favorites, err := favoriteSearcher(ctx, request.Query, defaultSearchLimit)
		if err != nil {
			return nil, err
		}
		return &SearchResponse{Results: favorites}, nil
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	resultsByType := make(map[types.ModelType][]*SearchResult)
	var searchErrors []error

	for modelType, searcher := range s.searchers {
		wg.Add(1)
		go func(mt types.ModelType, searchFunc searchFunc) {
			defer wg.Done()
			results, err := searchFunc(ctx, request.Query, defaultSearchLimit)
			mu.Lock()
			if err == nil {
				resultsByType[mt] = results
			} else {
				searchErrors = append(searchErrors, err)
			}
			mu.Unlock()
		}(modelType, searcher)
	}

	wg.Wait()

	// Return composite error if any searches failed
	if len(searchErrors) > 0 {
		return nil, errors.Join(searchErrors...)
	}

	// Calculate optimal slots based on actual result counts
	optimalSlots := s.calculateOptimalSlots(resultsByType)
	var allResults []*SearchResult

	// Add results in defined order
	for _, modelType := range resourceTypeOrder {
		results, ok := resultsByType[modelType]
		if !ok || len(results) == 0 {
			continue
		}
		allocation := optimalSlots[modelType]
		allResults = append(allResults, results[:allocation]...)
	}

	return &SearchResponse{
		Results: allResults,
	}, nil
}

func (s *manager) calculateOptimalSlots(resultsByType map[types.ModelType][]*SearchResult) map[types.ModelType]int32 {

	// Get types with results for slot allocation
	eligibleTypes := make([]types.ModelType, 0)
	for modelType, results := range resultsByType {
		if len(results) > 0 {
			eligibleTypes = append(eligibleTypes, modelType)
		}
	}

	if len(eligibleTypes) == 0 {
		return make(map[types.ModelType]int32)
	}

	// Sort for deterministic ordering
	sort.Slice(eligibleTypes, func(i, j int) bool {
		return eligibleTypes[i].Name() < eligibleTypes[j].Name()
	})

	slots := make(map[types.ModelType]int32)
	numTypes := int32(len(eligibleTypes))

	// Initial allocation
	unusedSlots := int32(0)
	for i, modelType := range eligibleTypes {
		baseAllocation := defaultSearchLimit / numTypes
		if int32(i) < defaultSearchLimit%numTypes {
			baseAllocation++
		}

		resultCount := int32(len(resultsByType[modelType]))
		allocated := min(baseAllocation, resultCount)
		slots[modelType] = allocated
		unusedSlots += baseAllocation - allocated
	}

	// Redistribute unused slots
	for unusedSlots > 0 {
		distributed := false
		for _, modelType := range eligibleTypes {
			if unusedSlots <= 0 {
				break
			}
			resultCount := int32(len(resultsByType[modelType]))
			if slots[modelType] < resultCount {
				slots[modelType]++
				unusedSlots--
				distributed = true
			}
		}
		if !distributed {
			break
		}
	}

	return slots
}

func min(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}
