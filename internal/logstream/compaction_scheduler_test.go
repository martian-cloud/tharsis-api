package logstream

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/db"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/maintenance"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

// matchClaimableBefore asserts the scheduler treats claims older than the TTL as reclaimable: the
// cutoff it passes is roughly now-TTL (a minute of slack for test execution time).
func matchClaimableBefore(claimableBefore time.Time) bool {
	age := time.Since(claimableBefore)
	return age >= compactionClaimTTL-time.Minute && age <= compactionClaimTTL+time.Minute
}

func newTestScheduler(mockLS *db.MockLogStreams, mgr Manager, mm maintenance.Monitor) *CompactionScheduler {
	logr, _ := logger.NewForTest()
	return &CompactionScheduler{
		dbClient:           &db.Client{LogStreams: mockLS},
		logger:             logr,
		manager:            mgr,
		maintenanceMonitor: mm,
	}
}

func TestCompactionSchedulerExecute(t *testing.T) {
	ctx := context.Background()

	t.Run("skips all work while in maintenance mode", func(t *testing.T) {
		mockLS := db.NewMockLogStreams(t)
		mockMgr := NewMockManager(t)
		mockMM := maintenance.NewMockMonitor(t)
		mockMM.On("InMaintenanceMode", mock.Anything).Return(true, nil)

		c := newTestScheduler(mockLS, mockMgr, mockMM)
		require.NoError(t, c.execute(ctx))
		// mockLS / mockMgr have no expectations: NewMock*(t) fails the test if either is called.
	})

	t.Run("compacts every stream returned in the batch", func(t *testing.T) {
		mockLS := db.NewMockLogStreams(t)
		mockMgr := NewMockManager(t)
		mockMM := maintenance.NewMockMonitor(t)
		mockMM.On("InMaintenanceMode", mock.Anything).Return(false, nil)

		mockLS.On("ClaimLogStreamsForCompaction", mock.Anything, compactionBatchSize, mock.MatchedBy(matchClaimableBefore)).
			Return([]models.LogStream{
				{Metadata: models.ResourceMetadata{ID: "ls1"}},
				{Metadata: models.ResourceMetadata{ID: "ls2"}},
			}, nil).Once()

		mockMgr.On("CompactStream", mock.Anything, mock.MatchedBy(func(ls *models.LogStream) bool {
			return ls.Metadata.ID == "ls1"
		})).Return(nil).Once()
		mockMgr.On("CompactStream", mock.Anything, mock.MatchedBy(func(ls *models.LogStream) bool {
			return ls.Metadata.ID == "ls2"
		})).Return(nil).Once()

		c := newTestScheduler(mockLS, mockMgr, mockMM)
		require.NoError(t, c.execute(ctx))
	})

	t.Run("isolates a per-stream failure and still compacts the rest", func(t *testing.T) {
		mockLS := db.NewMockLogStreams(t)
		mockMgr := NewMockManager(t)
		mockMM := maintenance.NewMockMonitor(t)
		mockMM.On("InMaintenanceMode", mock.Anything).Return(false, nil)

		mockLS.On("ClaimLogStreamsForCompaction", mock.Anything, compactionBatchSize, mock.MatchedBy(matchClaimableBefore)).
			Return([]models.LogStream{
				{Metadata: models.ResourceMetadata{ID: "ls1"}},
				{Metadata: models.ResourceMetadata{ID: "ls2"}},
			}, nil).Once()

		mockMgr.On("CompactStream", mock.Anything, mock.MatchedBy(func(ls *models.LogStream) bool {
			return ls.Metadata.ID == "ls1"
		})).Return(errors.New("compaction failed")).Once()
		mockMgr.On("CompactStream", mock.Anything, mock.MatchedBy(func(ls *models.LogStream) bool {
			return ls.Metadata.ID == "ls2"
		})).Return(nil).Once()

		c := newTestScheduler(mockLS, mockMgr, mockMM)
		// A single stream's failure is logged, not returned, so the run keeps going.
		require.NoError(t, c.execute(ctx))
	})

	t.Run("does nothing when the batch is empty", func(t *testing.T) {
		mockLS := db.NewMockLogStreams(t)
		mockMgr := NewMockManager(t)
		mockMM := maintenance.NewMockMonitor(t)
		mockMM.On("InMaintenanceMode", mock.Anything).Return(false, nil)

		mockLS.On("ClaimLogStreamsForCompaction", mock.Anything, compactionBatchSize, mock.MatchedBy(matchClaimableBefore)).
			Return([]models.LogStream{}, nil).Once()

		c := newTestScheduler(mockLS, mockMgr, mockMM)
		require.NoError(t, c.execute(ctx))
		// mockMgr.CompactStream is never expected.
	})

	t.Run("propagates a claim error", func(t *testing.T) {
		mockLS := db.NewMockLogStreams(t)
		mockMgr := NewMockManager(t)
		mockMM := maintenance.NewMockMonitor(t)
		mockMM.On("InMaintenanceMode", mock.Anything).Return(false, nil)

		mockLS.On("ClaimLogStreamsForCompaction", mock.Anything, compactionBatchSize, mock.MatchedBy(matchClaimableBefore)).
			Return(nil, errors.New("db down")).Once()

		c := newTestScheduler(mockLS, mockMgr, mockMM)
		err := c.execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to claim log streams")
	})

	t.Run("propagates a maintenance-mode check error without querying streams", func(t *testing.T) {
		mockLS := db.NewMockLogStreams(t)
		mockMgr := NewMockManager(t)
		mockMM := maintenance.NewMockMonitor(t)
		mockMM.On("InMaintenanceMode", mock.Anything).Return(false, errors.New("monitor error"))

		c := newTestScheduler(mockLS, mockMgr, mockMM)
		err := c.execute(ctx)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to check for maintenance mode")
		// mockLS has no GetLogStreams expectation: it must not be queried.
	})
}
