package client

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildUserAgent(t *testing.T) {
	result := BuildUserAgent("tharsis-cli", "1.0.0")
	assert.Equal(t, "tharsis-cli/1.0.0 ("+runtime.GOOS+"; "+runtime.GOARCH+")", result)
}
