package models

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateOutputVisibility(t *testing.T) {
	t.Run("nil value is valid (inherit)", func(t *testing.T) {
		err := validateOutputVisibility(nil)
		require.NoError(t, err)
	})

	for _, v := range ValidOutputVisibilities {
		t.Run(fmt.Sprintf("%s is valid", v), func(t *testing.T) {
			val := v
			err := validateOutputVisibility(&val)
			require.NoError(t, err)
		})
	}

	t.Run("invalid value is rejected", func(t *testing.T) {
		v := NamespaceOutputVisibilityLevel("invalid_value")
		err := validateOutputVisibility(&v)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid output visibility value")
	})
}
