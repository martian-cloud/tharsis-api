package pagination

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

type primaryCursorField struct {
	name  string
	value string
}

type secondaryCursorField struct {
	name  string
	value *string
}

type cursor struct {
	primary   *primaryCursorField
	secondary *secondaryCursorField
}

func newCursor(v string) (*cursor, error) {
	var parts []*string

	bytes, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return nil, errors.Wrap(err, "invalid cursor", errors.WithErrorCode(errors.EInvalid))
	}

	if err := json.Unmarshal(bytes, &parts); err != nil {
		return nil, errors.Wrap(err, "invalid cursor", errors.WithErrorCode(errors.EInvalid))
	}

	if len(parts) < 2 || len(parts) > 4 {
		return nil, errors.New("invalid cursor format: expected 2 or 4 parts", errors.WithErrorCode(errors.EInvalid))
	}

	if parts[0] == nil || parts[1] == nil {
		return nil, errors.New("invalid cursor format: primary cursor fields cannot be nil", errors.WithErrorCode(errors.EInvalid))
	}

	c := cursor{primary: &primaryCursorField{name: *parts[0], value: *parts[1]}}

	if len(parts) == 4 {
		if parts[2] == nil {
			return nil, errors.New("invalid cursor format: secondary field name cannot be nil", errors.WithErrorCode(errors.EInvalid))
		}

		c.secondary = &secondaryCursorField{name: *parts[2], value: parts[3]}
	}

	return &c, nil
}

func (c *cursor) encode() (string, error) {
	// Encode cursor into an array
	parts := []*string{&c.primary.name, &c.primary.value}

	if c.secondary != nil {
		parts = append(parts, &c.secondary.name)
		parts = append(parts, c.secondary.value)
	}

	bytes, err := json.Marshal(parts)
	if err != nil {
		return "", fmt.Errorf("failed to encode cursor: %w", err)
	}

	return base64.StdEncoding.EncodeToString(bytes), nil
}
