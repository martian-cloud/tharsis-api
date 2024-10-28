// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

package renderers

import (
	"encoding/json"
	"strings"
)

type evaluatedString struct {
	String string
	JSON   interface{}

	IsMultiline bool
	IsNull      bool
}

func evaluatePrimitiveString(value interface{}) evaluatedString {
	if value == nil {
		return evaluatedString{
			String: "null",
			IsNull: true,
		}
	}

	str := value.(string)

	if strings.HasPrefix(str, "{") || strings.HasPrefix(str, "[") {
		var jv interface{}
		if err := json.Unmarshal([]byte(str), &jv); err == nil {
			return evaluatedString{
				String: str,
				JSON:   jv,
			}
		}
	}

	if strings.Contains(str, "\n") {
		return evaluatedString{
			String:      strings.TrimSpace(str),
			IsMultiline: true,
		}
	}

	return evaluatedString{
		String: str,
	}
}
