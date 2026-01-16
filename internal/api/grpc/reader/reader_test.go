package reader

import (
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

func TestLimitReader(t *testing.T) {
	testData := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

	type testCase struct {
		expectError error
		name        string
		totalSupply string
		expectData  string
		limit       int64
		bufSize     int
		injectSize  int
		injectError bool
	}

	testCases := []testCase{
		{
			name:        "exact match, one read",
			totalSupply: testData,
			limit:       int64(len(testData)),
			bufSize:     len(testData),
			injectSize:  len(testData),
			expectData:  testData,
		},
		{
			name:        "exact match, multiple reads",
			totalSupply: testData[:41],
			limit:       41,
			bufSize:     11,
			injectSize:  11,
			expectData:  testData[:41],
		},
		{
			name:        "inject larger pieces",
			totalSupply: testData[:41],
			limit:       41,
			bufSize:     5,
			injectSize:  11,
			expectData:  testData[:41],
		},
		{
			name:        "inject smaller pieces",
			totalSupply: testData[:41],
			limit:       41,
			bufSize:     11,
			injectSize:  5,
			expectData:  testData[:41],
		},
		{
			name:        "exceeds limit: exact match, one read",
			totalSupply: testData,
			limit:       int64(len(testData) - 1),
			bufSize:     len(testData),
			injectSize:  len(testData),
			expectError: errors.New("exceeded file size limit", errors.WithErrorCode(errors.ETooLarge)),
		},
		{
			name:        "exceeds limit: exact match, multiple reads",
			totalSupply: testData[:41],
			limit:       40,
			bufSize:     11,
			injectSize:  11,
			expectError: errors.New("exceeded file size limit", errors.WithErrorCode(errors.ETooLarge)),
		},
		{
			name:        "exceeds limit: inject larger pieces",
			totalSupply: testData[:41],
			limit:       40,
			bufSize:     5,
			injectSize:  11,
			expectError: errors.New("exceeded file size limit", errors.WithErrorCode(errors.ETooLarge)),
		},
		{
			name:        "exceeds limit: inject smaller pieces",
			totalSupply: testData[:41],
			limit:       40,
			bufSize:     11,
			injectSize:  5,
			expectError: errors.New("exceeded file size limit", errors.WithErrorCode(errors.ETooLarge)),
		},
		{
			name:        "inject error",
			bufSize:     11,
			injectError: true,
			expectError: fmt.Errorf("injected error"),
		},
	}

	for _, test := range testCases {
		t.Run(test.name, func(t *testing.T) {
			remainingSupply := test.totalSupply
			oldRemainingSupply := remainingSupply // trailing value for mock-out error function below

			rut := NewLimitReader(func() ([]byte, error) {
				if test.injectError {
					return nil, fmt.Errorf("injected error")
				}

				// If finished.
				// The testing platform appears to run the above function before this one,
				// so this function must use the old remaining supply value.
				if len(oldRemainingSupply) == 0 {
					return nil, io.EOF
				}

				// Now, it's safe to update oldRemainingSupply.
				oldRemainingSupply = remainingSupply

				var toReturn string
				switch {
				case len(remainingSupply) == 0:
					toReturn = ""
				case len(remainingSupply) >= test.injectSize:
					toReturn = remainingSupply[:test.injectSize]
					remainingSupply = remainingSupply[test.injectSize:]
				default:
					toReturn = remainingSupply
					remainingSupply = ""
				}

				return []byte(toReturn), nil
			}, test.limit)

			// Read all the data that's available.
			actualData := []byte{}
			var actualError error
			var nGot int
			buf := make([]byte, test.bufSize)
			for {

				nGot, actualError = rut.Read(buf)
				if actualError != nil {
					// break on any error, including EOF
					break
				}

				actualData = append(actualData, buf[:nGot]...)
			}

			if test.expectError == nil {
				assert.Equal(t, io.EOF, actualError)
				assert.Equal(t, test.expectData, string(actualData))
			} else {
				assert.Equal(t, test.expectError, actualError)
			}
		})
	}
}
