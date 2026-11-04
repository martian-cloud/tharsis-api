package servers

import (
	"context"
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/services"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	pb "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/protos/gen"
)

func TestUserServer_ActivateAdminModeDurationOverflow(t *testing.T) {
	// The request is rejected before the user service is reached, so the catalog has none.
	server := NewUserServer(&services.Catalog{})

	// Minutes past what a time.Duration can hold would otherwise wrap to a short (or negative) duration.
	for _, minutes := range []int32{math.MaxInt32, math.MinInt32} {
		_, err := server.ActivateAdminMode(context.Background(), &pb.ActivateAdminModeRequest{DurationMinutes: &minutes})
		require.Error(t, err, "minutes %d", minutes)
		assert.Equal(t, errors.EInvalid, errors.ErrorCode(err))
	}
}
