// Package response providers support for returning http responses
package response

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/hashicorp/jsonapi"

	te "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/logger"
)

const contentTypeJSON = "application/json"

type errorResponse struct {
	Detail string `json:"detail"`
}

// Writer provides utility functions for responding to http requests
type Writer interface {
	RespondWithError(ctx context.Context, w http.ResponseWriter, err error)
	RespondWithJSON(ctx context.Context, w http.ResponseWriter, model interface{}, statusCode int)
	RespondWithJSONAPI(ctx context.Context, w http.ResponseWriter, model interface{}, statusCode int)
	RespondWithPaginatedJSONAPI(ctx context.Context, w http.ResponseWriter, model interface{}, statusCode int)
}

type responseHelper struct {
	logger logger.Logger
}

var tharsisErrorToStatusCode = map[te.CodeType]int{
	te.EInternal:           http.StatusInternalServerError,
	te.ENotImplemented:     http.StatusNotImplemented,
	te.EInvalid:            http.StatusBadRequest,
	te.EConflict:           http.StatusConflict,
	te.ENotFound:           http.StatusNotFound,
	te.EForbidden:          http.StatusForbidden,
	te.ETooManyRequests:    http.StatusTooManyRequests,
	te.EUnauthorized:       http.StatusUnauthorized,
	te.ETooLarge:           http.StatusRequestEntityTooLarge,
	te.EServiceUnavailable: http.StatusServiceUnavailable,
}

// NewWriter creates an instance of Writer
func NewWriter(logger logger.Logger) Writer {
	return &responseHelper{logger}
}

// RespondWithError responds to an http request with an error response
func (rh *responseHelper) RespondWithError(ctx context.Context, w http.ResponseWriter, err error) {
	if !te.IsContextCanceledError(err) && te.ErrorCode(err) == te.EInternal {
		// Log error message
		rh.logger.WithContextFields(ctx).Errorf("Unexpected error occurred: %s", err.Error())
	}
	rh.respondWithError(ctx, w, ErrorCodeToStatusCode(te.ErrorCode(err)), te.ErrorMessage(err))
}

// RespondWithJSON responds to an http request with a json payload
func (rh *responseHelper) RespondWithJSON(ctx context.Context, w http.ResponseWriter, model interface{}, statusCode int) {
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(statusCode)

	if model != nil {
		response, err := json.Marshal(model)
		if err != nil {
			rh.RespondWithError(ctx, w, err)
			return
		}

		if _, err := w.Write(response); err != nil {
			rh.RespondWithError(ctx, w, err)
			return
		}
	}
}

func (rh *responseHelper) RespondWithJSONAPI(ctx context.Context, w http.ResponseWriter, model interface{}, statusCode int) {
	w.Header().Set("Content-Type", jsonapi.MediaType)
	w.WriteHeader(statusCode)

	if model != nil {
		if err := jsonapi.MarshalPayload(w, model); err != nil {
			rh.RespondWithError(ctx, w, err)
		}
	}
}

func (rh *responseHelper) RespondWithPaginatedJSONAPI(ctx context.Context, w http.ResponseWriter, model interface{}, statusCode int) {
	w.Header().Set("Content-Type", jsonapi.MediaType)
	w.WriteHeader(statusCode)

	if model != nil {
		val := reflect.Indirect(reflect.ValueOf(model))

		// Verify that the value is a struct
		if val.Kind() != reflect.Struct {
			rh.RespondWithError(ctx, w, fmt.Errorf("unexpected error occurred: type is not a struct"))
			return
		}

		// Get the items and pagination fields
		items := val.FieldByName("Items")
		pagination := val.FieldByName("Pagination")

		// Verify that the items field is present
		if !items.IsValid() {
			rh.RespondWithError(ctx, w, fmt.Errorf("unexpected error occurred: items field is missing"))
			return
		}

		// Verify that the items field is of type slice
		if items.Type().Kind() != reflect.Slice {
			rh.RespondWithError(ctx, w, fmt.Errorf("unexpected error occurred: items field is not of type Slice"))
			return
		}

		payload, err := jsonapi.Marshal(items.Interface())
		if err != nil {
			rh.RespondWithError(ctx, w, fmt.Errorf("unexpected error occurred: failed to marshal pagination response"))
			return
		}

		// Pagination field is optional
		if pagination.IsValid() {
			var meta map[string]interface{}
			metaJSON, err := json.Marshal(pagination.Interface())
			if err != nil {
				rh.RespondWithError(ctx, w, fmt.Errorf("unexpected error occurred: failed to marshal pagination metadata"))
				return
			}

			if err := json.Unmarshal(metaJSON, &meta); err != nil {
				rh.RespondWithError(ctx, w, fmt.Errorf("unexpected error occurred: failed to unmarshal pagination metadata"))
				return
			}

			// Set pagination metadata on payload
			payload.(*jsonapi.ManyPayload).Meta = (*jsonapi.Meta)(&meta)
		}

		payload.(*jsonapi.ManyPayload).Included = []*jsonapi.Node{}

		if err := json.NewEncoder(w).Encode(payload); err != nil {
			rh.RespondWithError(ctx, w, err)
		}
	}
}

func (rh *responseHelper) respondWithError(ctx context.Context, w http.ResponseWriter, code int, msg string) {
	rh.RespondWithJSON(ctx, w, &errorResponse{Detail: msg}, code)
}

// ErrorCodeToStatusCode maps a tharsis error code string to a
// http status code integer.
func ErrorCodeToStatusCode(code te.CodeType) int {
	// Otherwise map internal error codes to HTTP status codes.
	statusCode, ok := tharsisErrorToStatusCode[code]
	if ok {
		return statusCode
	}
	return http.StatusInternalServerError
}
