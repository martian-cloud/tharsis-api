// Copyright (c) Tharsis Authors
// Copyright (c) HashiCorp, Inc.

// This file contains code from the v1.5.7 tag in the Terraform
// repo which is licensed under the MPL license. The original
// source code can be found here:
// https://github.com/hashicorp/terraform/tree/v1.5.7

package jsondiff

import "fmt"

// Type represents the type of a JSON value
type Type string

// Type constants
const (
	Number Type = "number"
	Object Type = "object"
	Array  Type = "array"
	Bool   Type = "bool"
	String Type = "string"
	Null   Type = "null"
)

// GetType returns the type of the given JSON value
func GetType(json interface{}) (Type, error) {
	switch json.(type) {
	case []interface{}:
		return Array, nil
	case float64:
		return Number, nil
	case string:
		return String, nil
	case bool:
		return Bool, nil
	case nil:
		return Null, nil
	case map[string]interface{}:
		return Object, nil
	default:
		return "", fmt.Errorf("unrecognized json type %T", json)
	}
}
