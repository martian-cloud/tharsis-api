package resolver

import (
	"encoding/json"
	"math"
	"math/big"
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// int64Sentinel is the value an Int64 holds before UnmarshalGraphQL, so a rejected input that
// still wrote to the receiver is noticed.
const int64Sentinel = Int64(-424242)

func FuzzInt64UnmarshalGraphQL(f *testing.F) {
	// kind selects the Go type the input arrives as: 0 int32, 1 int64, 2 float64, 3 string,
	// 4 bool, 5 nil, 6 json.Number.
	f.Add(uint8(0), int64(42), 0.0, "", false)
	f.Add(uint8(0), int64(math.MaxInt64), 0.0, "", false)
	f.Add(uint8(1), int64(math.MinInt64), 0.0, "", false)
	f.Add(uint8(1), int64(math.MaxInt64), 0.0, "", false)
	f.Add(uint8(2), int64(0), 12345.0, "", false)
	f.Add(uint8(2), int64(0), -0.0, "", false)
	// Fractional floats used to be silently truncated toward zero.
	f.Add(uint8(2), int64(0), 1.5, "", false)
	f.Add(uint8(2), int64(0), -0.999, "", false)
	// Out of range and non-finite floats used to convert to an implementation-defined value.
	f.Add(uint8(2), int64(0), 1e300, "", false)
	f.Add(uint8(2), int64(0), math.Inf(1), "", false)
	f.Add(uint8(2), int64(0), math.Inf(-1), "", false)
	f.Add(uint8(2), int64(0), math.NaN(), "", false)
	f.Add(uint8(2), int64(0), 9223372036854775807.0, "", false) // rounds to 2^63
	f.Add(uint8(2), int64(0), -9223372036854775808.0, "", false)
	f.Add(uint8(2), int64(0), 9007199254740993.0, "", false)
	f.Add(uint8(3), int64(0), 0.0, "9223372036854775807", false)
	f.Add(uint8(3), int64(0), 0.0, "-9223372036854775808", false)
	f.Add(uint8(3), int64(0), 0.0, "9223372036854775808", false)
	f.Add(uint8(3), int64(0), 0.0, "+007", false)
	f.Add(uint8(3), int64(0), 0.0, "1.5", false)
	f.Add(uint8(3), int64(0), 0.0, "1e3", false)
	f.Add(uint8(3), int64(0), 0.0, "1_000", false)
	f.Add(uint8(3), int64(0), 0.0, " 1", false)
	f.Add(uint8(3), int64(0), 0.0, "0x10", false)
	f.Add(uint8(3), int64(0), 0.0, "", false)
	f.Add(uint8(4), int64(0), 0.0, "", true)
	f.Add(uint8(5), int64(0), 0.0, "", false)
	f.Add(uint8(6), int64(0), 0.0, "12", false)
	f.Add(uint8(6), int64(0), 0.0, "1.5", false)

	f.Fuzz(func(t *testing.T, kind uint8, n int64, fl float64, s string, b bool) {
		var (
			input any
			// accept reports whether the input must be accepted, and want is the value it must
			// unmarshal to. A nil accept means either outcome is allowed, as long as an accepted
			// value is exactly want.
			accept *bool
			want   *big.Int
		)
		yes, no := true, false

		switch kind % 7 {
		case 0:
			input = int32(n)
			accept, want = &yes, big.NewInt(int64(int32(n)))
		case 1:
			input = n
			accept, want = &yes, big.NewInt(n)
		case 2:
			input = fl
			// Only a finite, whole float has an exact integer value; the range check follows.
			accept = &no
			if !math.IsNaN(fl) && !math.IsInf(fl, 0) {
				if exact, acc := big.NewFloat(fl).Int(nil); acc == big.Exact {
					want, accept = exact, &yes
				}
			}
		case 3:
			input = s
			// A base 10 integer with an optional sign; big.Int rejects everything else here.
			accept = &no
			if v, ok := new(big.Int).SetString(s, 10); ok {
				want, accept = v, &yes
			}
		case 4:
			input = b
			accept = &no
		case 5:
			input = nil
			accept = &no
		case 6:
			// Not produced by the GraphQL library today; accepting it is optional, but an accepted
			// value must be its exact integer value.
			input = json.Number(s)
			if v, ok := new(big.Int).SetString(s, 10); ok {
				want = v
			}
		}

		// Anything outside the int64 range must be rejected rather than wrapped or clamped.
		if want != nil && !want.IsInt64() {
			want, accept = nil, &no
		}

		v := int64Sentinel
		err := v.UnmarshalGraphQL(input)

		if accept != nil {
			require.Equal(t, *accept, err == nil, "input %T(%v): err=%v", input, input, err)
		}
		if err != nil {
			assert.Equal(t, int64Sentinel, v, "a rejected input must not be written")
			return
		}
		require.NotNil(t, want, "accepted %T(%v) has no exact integer value", input, input)
		require.Equal(t, want.Int64(), int64(v), "input %T(%v) was not preserved exactly", input, input)

		// An accepted value serializes as the same integer, and parses back to itself both from
		// JSON and from its string form.
		out, err := v.MarshalJSON()
		require.NoError(t, err)
		assert.Equal(t, strconv.FormatInt(int64(v), 10), string(out))

		var decoded int64
		require.NoError(t, json.Unmarshal(out, &decoded))
		assert.Equal(t, int64(v), decoded)

		roundTrip := int64Sentinel
		require.NoError(t, roundTrip.UnmarshalGraphQL(string(out)))
		assert.Equal(t, v, roundTrip)
	})
}
