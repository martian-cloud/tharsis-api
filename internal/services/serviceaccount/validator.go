package serviceaccount

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwt"
	"github.com/ryanuber/go-glob"
)

// trustPolicyClaimValidator is a jwt.Validator that validates a claim value against a given value.
type trustPolicyClaimValidator struct {
	name    string
	value   string
	useGlob bool
}

func newClaimValueValidator(name string, value string, useGlob bool) jwt.Validator {
	return &trustPolicyClaimValidator{
		name:    name,
		value:   value,
		useGlob: useGlob,
	}
}

func (tpv *trustPolicyClaimValidator) Validate(_ context.Context, t jwt.Token) jwt.ValidationError {
	var claimVal interface{}
	// aud is a special case since it's not in the private claims map
	if tpv.name == "aud" {
		v := t.Audience()
		// If aud is an array of a single item then we can just use that item to simplify the comparison
		if len(v) == 1 {
			claimVal = v[0]
		} else {
			claimVal = v
		}
	} else {
		v, ok := t.Get(tpv.name)
		if !ok {
			return jwt.NewValidationError(fmt.Errorf(`claim %q not satisfied: claim %q does not exist`, tpv.name, tpv.name))
		}
		claimVal = v
	}

	// Parse claim value based on its type
	var parsedClaimVal string
	switch v := claimVal.(type) {
	case string:
		parsedClaimVal = v
	case []string:
		buf, err := json.Marshal(v)
		if err != nil {
			return jwt.NewValidationError(fmt.Errorf(`claim %q not satisfied: failed to marshal claim value: %v`, tpv.name, err))
		}
		parsedClaimVal = string(buf)
	case bool:
		parsedClaimVal = fmt.Sprintf("%t", v)
	case int:
		parsedClaimVal = fmt.Sprintf("%d", v)
	default:
		return jwt.NewValidationError(fmt.Errorf(`claim %q not satisfied: unsupported claim value type: %T`, tpv.name, v))
	}

	// Perform comparison
	var match bool
	if tpv.useGlob {
		match = glob.Glob(tpv.value, parsedClaimVal)
	} else {
		match = parsedClaimVal == tpv.value
	}

	// Return error if comparison fails
	if !match {
		return jwt.NewValidationError(fmt.Errorf(`claim %q not satisfied: values do not match: %q != %q`, tpv.name, tpv.value, parsedClaimVal))
	}

	return nil
}
