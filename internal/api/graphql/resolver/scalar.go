package resolver

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
)

// Int64 is a custom GraphQL scalar for 64-bit signed integers. GraphQL's built-in Int is 32-bit,
// and Float loses integer precision beyond 2^53, so counters that can exceed those ranges (byte
// and packet totals, CPU-time in milliseconds) use this type. It must be added to the schema via
// "scalar Int64".
type Int64 int64

// ImplementsGraphQLType maps this Go type to the Int64 scalar in the schema.
func (Int64) ImplementsGraphQLType(name string) bool {
	return name == "Int64"
}

// UnmarshalGraphQL is the custom unmarshaler, called when Int64 is used as an input.
func (i *Int64) UnmarshalGraphQL(input interface{}) error {
	switch input := input.(type) {
	case int32:
		*i = Int64(input)
		return nil
	case int64:
		*i = Int64(input)
		return nil
	case float64:
		// JSON variables decode to float64. Converting a fractional, non-finite or out of range
		// float would silently truncate it (or produce an implementation-defined value), so only
		// a whole number within the int64 range is accepted. -2^63 is representable exactly;
		// 2^63 (float64(math.MaxInt64) rounds up to it) is not in range.
		if input != math.Trunc(input) || input < math.MinInt64 || input >= math.MaxInt64 {
			return fmt.Errorf("value for Int64 must be a whole number within the 64-bit integer range: %v", input)
		}
		*i = Int64(input)
		return nil
	case string:
		v, err := strconv.ParseInt(input, 10, 64)
		if err != nil {
			return err
		}
		*i = Int64(v)
		return nil
	default:
		return fmt.Errorf("wrong type for Int64: %T", input)
	}
}

// MarshalJSON is the custom marshaler, called when querying a field that returns Int64.
func (i Int64) MarshalJSON() ([]byte, error) {
	return json.Marshal(int64(i))
}
