package pagination

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

type cursorField struct {
	name  string
	value string
}

type cursor struct {
	primary   *cursorField
	secondary *cursorField
}

func newCursor(v string) (*cursor, error) {
	var parts []string

	bytes, err := base64.StdEncoding.DecodeString(v)
	if err != nil {
		return nil, errors.Wrap(err, "invalid cursor", errors.WithErrorCode(errors.EInvalid))
	}

	if err := json.Unmarshal(bytes, &parts); err != nil {
		return nil, errors.Wrap(err, "invalid cursor", errors.WithErrorCode(errors.EInvalid))
	}

	c := cursor{primary: &cursorField{name: parts[0], value: parts[1]}}

	if len(parts) == 4 {
		c.secondary = &cursorField{name: parts[2], value: parts[3]}
	}

	return &c, nil
}

func (c *cursor) encode() (string, error) {
	// Encode cursor into an array
	parts := []string{c.primary.name, c.primary.value}

	if c.secondary != nil {
		parts = append(parts, c.secondary.name)
		parts = append(parts, c.secondary.value)
	}

	bytes, err := json.Marshal(parts)
	if err != nil {
		return "", fmt.Errorf("failed to encode cursor: %w", err)
	}

	return base64.StdEncoding.EncodeToString(bytes), nil
}
