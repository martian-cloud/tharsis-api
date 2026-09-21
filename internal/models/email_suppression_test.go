package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
)

func TestEmailSuppression(t *testing.T) {
	suppression := &EmailSuppression{Address: "user@x.com", Cause: EmailSuppressionCauseHardBounce}
	suppression.Metadata.ID = "supp-1"

	assert.Equal(t, "supp-1", suppression.GetID())
	assert.Equal(t, types.EmailSuppressionModelType, suppression.GetModelType())
	assert.NotEmpty(t, suppression.GetGlobalID())
	assert.NoError(t, suppression.Validate())

	val, err := suppression.ResolveMetadata("id")
	require.NoError(t, err)
	require.NotNil(t, val)
	assert.Equal(t, "supp-1", *val)
}
