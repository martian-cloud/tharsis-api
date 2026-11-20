// Package apikey generates and parses opaque API keys. A key is a product prefix, a type
// character, an underscore, 43 random base62 characters and a 6 character base62 CRC32 of the
// random part, e.g. tsc_<43 random><6 checksum>. Only the SHA-256 hash of a key should be stored.
package apikey

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"hash/crc32"
	"math/big"
	"strings"

	"gitlab.com/infor-cloud/martian-cloud/tharsis/tharsis-api/pkg/errors"
)

const (
	tharsisProductPrefix = "ts"
	headerSeparator      = '_'
	base62Alphabet       = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	randomPartLength     = 43
	checksumPartLength   = 6

	typeIndex      = len(tharsisProductPrefix)
	separatorIndex = typeIndex + 1
	headerLength   = separatorIndex + 1
	checksumIndex  = headerLength + randomPartLength
	totalKeyLength = checksumIndex + checksumPartLength
)

var errInvalidKey = errors.New("invalid API key", errors.WithErrorCode(errors.EUnauthorized))

// Type identifies what a key is used for and is the third character of the key.
type Type string

// Key types.
const (
	TypeSCIM Type = "c"
	TypeVCS  Type = "v"
)

func (t Type) valid() bool {
	return t == TypeSCIM || t == TypeVCS
}

// Generate returns a new key of the given type along with its storage hash.
func Generate(keyType Type) (key string, hash string, err error) {
	if !keyType.valid() {
		return "", "", errors.New("unsupported API key type %q", keyType)
	}

	alphabetSize := big.NewInt(int64(len(base62Alphabet)))
	randomPart := make([]byte, randomPartLength)
	for i := range randomPart {
		n, err := rand.Int(rand.Reader, alphabetSize)
		if err != nil {
			return "", "", errors.Wrap(err, "failed to generate random API key value")
		}

		randomPart[i] = base62Alphabet[n.Int64()]
	}

	key = tharsisProductPrefix + string(keyType) + string(headerSeparator) + string(randomPart) + checksumOf(string(randomPart))

	return key, GetHash(key), nil
}

// Parse validates the structure and checksum of a key without any lookup and returns its type and storage hash.
func Parse(key string) (Type, string, error) {
	if !IsAPIKey(key) || len(key) != totalKeyLength {
		return "", "", errInvalidKey
	}

	keyType := Type(key[typeIndex])
	if !keyType.valid() {
		return "", "", errInvalidKey
	}

	randomPart := key[headerLength:checksumIndex]
	checksumPart := key[checksumIndex:]
	if !isBase62(randomPart+checksumPart) || checksumPart != checksumOf(randomPart) {
		return "", "", errInvalidKey
	}

	return keyType, GetHash(key), nil
}

// IsAPIKey returns true if the token starts with an API key header, it does not verify the rest of the key.
func IsAPIKey(token string) bool {
	return len(token) >= headerLength && strings.HasPrefix(token, tharsisProductPrefix) && token[separatorIndex] == headerSeparator
}

// GetHash returns the hex encoded SHA-256 hash of a key.
func GetHash(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

func checksumOf(randomPart string) string {
	value := crc32.ChecksumIEEE([]byte(randomPart))

	encoded := make([]byte, checksumPartLength)
	for i := checksumPartLength - 1; i >= 0; i-- {
		encoded[i] = base62Alphabet[value%uint32(len(base62Alphabet))]
		value /= uint32(len(base62Alphabet))
	}

	return string(encoded)
}

func isBase62(value string) bool {
	return strings.IndexFunc(value, func(r rune) bool { return !strings.ContainsRune(base62Alphabet, r) }) < 0
}
