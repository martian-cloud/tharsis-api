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

type SearchResult struct {
	Data models.Model
	Type types.ModelType
}

type SearchRequest struct {
	Query string
}

type SearchResponse struct {
	Results []*SearchResult
}

// searchFunc defines the search function type
type searchFunc func(ctx context.Context, query string, limit int32) ([]*SearchResult, error)

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

func NewManager(catalog *services.Catalog, logger logger.Logger) Manager {
	return newManager(catalog, logger, map[types.ModelType]searchFunc{
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

	if request.Query == "" {
		return &SearchResponse{Results: []*SearchResult{}}, nil
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

	// Single pass allocation using optimal slots
	for _, modelType := range s.modelTypes {
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
