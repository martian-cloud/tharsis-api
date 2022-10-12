// Package response providers support for returning http responses
package response

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"

	"github.com/hashicorp/jsonapi"

	te "gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/errors"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/logger"
)

const contentTypeJSON = "application/json"

type errorResponse struct {
	Detail string `json:"detail"`
}

// Writer provides utility functions for responding to http requests
type Writer interface {
	RespondWithError(w http.ResponseWriter, err error)
	RespondWithJSON(w http.ResponseWriter, model interface{}, statusCode int)
	RespondWithJSONAPI(w http.ResponseWriter, model interface{}, statusCode int)
	RespondWithPaginatedJSONAPI(w http.ResponseWriter, model interface{}, statusCode int)
}

type responseHelper struct {
	logger logger.Logger
}

var tharsisErrorToStatusCode = map[string]int{
	te.EInternal:        http.StatusInternalServerError,
	te.ENotImplemented:  http.StatusNotImplemented,
	te.EInvalid:         http.StatusBadRequest,
	te.EConflict:        http.StatusConflict,
	te.ENotFound:        http.StatusNotFound,
	te.EForbidden:       http.StatusForbidden,
	te.ETooManyRequests: http.StatusTooManyRequests,
	te.EUnauthorized:    http.StatusUnauthorized,
	te.ETooLarge:        http.StatusRequestEntityTooLarge,
}

// NewWriter creates an instance of Writer
func NewWriter(logger logger.Logger) Writer {
	return &responseHelper{logger}
}

// RespondWithError responds to an http request with an error response
func (rh *responseHelper) RespondWithError(w http.ResponseWriter, err error) {
	if err != context.Canceled && te.ErrorCode(err) != te.EUnauthorized && te.ErrorCode(err) != te.EForbidden && te.ErrorCode(err) != te.ENotFound {
		rh.logger.Errorf("Unexpected error occurred: %s", err.Error())
	}
	rh.respondWithError(w, ErrorCodeToStatusCode(te.ErrorCode(err)), te.ErrorMessage(err))
}

// RespondWithJSON responds to an http request with a json payload
func (rh *responseHelper) RespondWithJSON(w http.ResponseWriter, model interface{}, statusCode int) {
	w.Header().Set("Content-Type", contentTypeJSON)
	w.WriteHeader(statusCode)

	if model != nil {
		response, err := json.Marshal(model)
		if err != nil {
			rh.RespondWithError(w, err)
			return
		}

		if _, err := w.Write(response); err != nil {
			rh.RespondWithError(w, err)
			return
		}
	}
}

func (rh *responseHelper) RespondWithJSONAPI(w http.ResponseWriter, model interface{}, statusCode int) {
	w.Header().Set("Content-Type", jsonapi.MediaType)
	w.WriteHeader(statusCode)

	if model != nil {
		if err := jsonapi.MarshalPayload(w, model); err != nil {
			rh.RespondWithError(w, err)
		}
	}
}

func (rh *responseHelper) RespondWithPaginatedJSONAPI(w http.ResponseWriter, model interface{}, statusCode int) {
	w.Header().Set("Content-Type", jsonapi.MediaType)
	w.WriteHeader(statusCode)

	if model != nil {
		val := reflect.Indirect(reflect.ValueOf(model))

		// Verify that the value is a struct
		if val.Kind() != reflect.Struct {
			rh.RespondWithError(w, fmt.Errorf("unexpected error occurred: type is not a struct"))
			return
		}

		// Get the items and pagination fields
		items := val.FieldByName("Items")
		pagination := val.FieldByName("Pagination")

		// Verify that the items field is present
		if !items.IsValid() {
			rh.RespondWithError(w, fmt.Errorf("unexpected error occurred: items field is missing"))
			return
		}

		// Verify that the items field is of type slice
		if items.Type().Kind() != reflect.Slice {
			rh.RespondWithError(w, fmt.Errorf("unexpected error occurred: items field is not of type Slice"))
			return
		}

		payload, err := jsonapi.Marshal(items.Interface())
		if err != nil {
			rh.RespondWithError(w, fmt.Errorf("unexpected error occurred: failed to marshal pagination response"))
			return
		}

		// Pagination field is optional
		if pagination.IsValid() {
			var meta map[string]interface{}
			metaJSON, err := json.Marshal(pagination.Interface())
			if err != nil {
				rh.RespondWithError(w, fmt.Errorf("unexpected error occurred: failed to marshal pagination metadata"))
				return
			}

			if err := json.Unmarshal(metaJSON, &meta); err != nil {
				rh.RespondWithError(w, fmt.Errorf("unexpected error occurred: failed to unmarshal pagination metadata"))
				return
			}

			// Set pagination metadata on payload
			payload.(*jsonapi.ManyPayload).Meta = (*jsonapi.Meta)(&meta)
		}

		payload.(*jsonapi.ManyPayload).Included = []*jsonapi.Node{}

		if err := json.NewEncoder(w).Encode(payload); err != nil {
			rh.RespondWithError(w, err)
		}
	}
}

func (rh *responseHelper) respondWithError(w http.ResponseWriter, code int, msg string) {
	rh.RespondWithJSON(w, &errorResponse{Detail: msg}, code)
}

// ErrorCodeToStatusCode maps a tharsis error code string to a
// http status code integer.
func ErrorCodeToStatusCode(code string) int {
	// Otherwise map internal error codes to HTTP status codes.
	statusCode, ok := tharsisErrorToStatusCode[code]
	if ok {
		return statusCode
	}
	return http.StatusInternalServerError
}
