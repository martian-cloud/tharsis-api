// Package gid package
package gid

import (
	"encoding/base64"
	"fmt"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/internal/models/types"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

// GlobalID is a model ID with type information
type GlobalID struct {
	Code string
	ID   string
}

// NewGlobalID returns a new GlobalID
func NewGlobalID(mt types.ModelType, modelID string) *GlobalID {
	return &GlobalID{Code: mt.GIDCode(), ID: modelID}
}

// String returns the string representation of the global ID
func (g *GlobalID) String() string {
	return base64.RawURLEncoding.EncodeToString([]byte(fmt.Sprintf("%s_%s", g.Code, g.ID)))
}

// ParseGlobalID parses a global ID string and returns a GlobalID type
func ParseGlobalID(globalID string) (*GlobalID, error) {
	decodedBytes, err := base64.RawURLEncoding.DecodeString(globalID)
	if err != nil {
		return nil, errors.Wrap(err, "invalid ID", errors.WithErrorCode(errors.EInvalid))
	}

	decodedGlobalID := string(decodedBytes)

	index := strings.Index(decodedGlobalID, "_")
	if index == -1 {
		return nil, errors.New("invalid ID", errors.WithErrorCode(errors.EInvalid))
	}

	code := decodedGlobalID[:index]
	id := decodedGlobalID[index+1:]

	// Validate the code is not empty and not excessively long
	if code == "" || len(code) > 5 {
		return nil, errors.New("invalid GID code", errors.WithErrorCode(errors.EInvalid))
	}

	// Check if code contains only uppercase letters
	for _, r := range code {
		if !unicode.IsUpper(r) {
			return nil, errors.New("invalid GID code", errors.WithErrorCode(errors.EInvalid))
		}
	}

	if err := uuid.Validate(id); err != nil {
		return nil, errors.Wrap(err, "invalid ID", errors.WithErrorCode(errors.EInvalid))
	}

	return &GlobalID{
		Code: code,
		ID:   id,
	}, nil
}

// ToGlobalID converts a model type and DB ID to a global ID string
func ToGlobalID(mt types.ModelType, id string) string {
	return NewGlobalID(mt, id).String()
}

// FromGlobalID converts a global ID string to a DB ID string
func FromGlobalID(globalID string) string {
	gid, err := ParseGlobalID(globalID)
	if err != nil {
		return fmt.Sprintf("invalid[%s]", globalID)
	}
	return gid.ID
}
