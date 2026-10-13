package response

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	te "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestErrorCodeToStatusCode(t *testing.T) {
	tests := []struct {
		code   te.CodeType
		expect int
	}{
		{code: te.EInternal, expect: http.StatusInternalServerError},
		{code: te.ENotFound, expect: http.StatusNotFound},
		{code: te.EServiceUnavailable, expect: http.StatusServiceUnavailable},
		// Data loss is a server fault, so it is a 500 rather than a 404 for the resource being read.
		{code: te.EDataLoss, expect: http.StatusInternalServerError},
	}

	for _, test := range tests {
		t.Run(string(test.code), func(t *testing.T) {
			assert.Equal(t, test.expect, ErrorCodeToStatusCode(test.code))
		})
	}
}
