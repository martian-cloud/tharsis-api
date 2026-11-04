package tfe

import (
	"bytes"
	"encoding/json"
	stderrors "errors"
	"io"
	"net/http"
	"strings"

	"github.com/hashicorp/jsonapi"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// convertOrgToGroupPath takes an organization path and converts it to a tharsis group
func convertOrgToGroupPath(org string) string {
	parts := strings.Split(org, ".")
	return strings.Join(parts, "/")
}

// unmarshalJSONAPIRequest decodes a JSON:API request body into model. A body that does not decode is the
// client's mistake, so it is reported as invalid rather than as an internal error.
func unmarshalJSONAPIRequest(body io.Reader, model any) error {
	if err := jsonapi.UnmarshalPayload(body, model); err != nil {
		return requestBodyError(err)
	}
	return nil
}

// unmarshalJSONRequest decodes a plain JSON request body into model. go-tfe sends an options struct tagged
// json rather than jsonapi (e.g. RunApplyOptions) as plain JSON, and an empty body leaves model unchanged.
func unmarshalJSONRequest(body io.Reader, model any) error {
	data, err := io.ReadAll(body)
	if err != nil {
		return requestBodyError(err)
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	if err := json.Unmarshal(data, model); err != nil {
		return requestBodyError(err)
	}
	return nil
}

// requestBodyError reports a request body that could not be read or decoded as the client's error: too
// large when it is over the route's body limit, invalid otherwise.
func requestBodyError(err error) error {
	var maxBytesErr *http.MaxBytesError
	if stderrors.As(err, &maxBytesErr) {
		return errors.New("request body exceeds the maximum of %d bytes", maxBytesErr.Limit, errors.WithErrorCode(errors.ETooLarge))
	}
	return errors.Wrap(err, "invalid request body", errors.WithErrorCode(errors.EInvalid))
}
