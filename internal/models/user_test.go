package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Hashes created outside the API by htpasswd must verify.
func TestUser_VerifyPassword_HtpasswdHash(t *testing.T) {
	// htpasswd -inBC 10 output with $2y rewritten to $2a, as seed.sh does
	user := &User{PasswordHash: []byte("$2a$10$zKzRkEc03S8HIHuiyEy6SelVFlLfNt3B7OcqplczP9bZkZy08/nYq")}

	assert.True(t, user.VerifyPassword("L-SU44Et5HfpHO_-vPWSGojtyrN_1Uo92rMOKR7JvLA"))
	assert.False(t, user.VerifyPassword("L-SU44Et5HfpHO_-vPWSGojtyrN_1Uo92rMOKR7JvLB"))
	assert.False(t, user.VerifyPassword(""))
	assert.False(t, (&User{}).VerifyPassword("L-SU44Et5HfpHO_-vPWSGojtyrN_1Uo92rMOKR7JvLA"))
}
