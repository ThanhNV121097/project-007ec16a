package orders

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// newToken is the secret that opens one order: 24 bytes (192 bits) from
// crypto/rand, written as 32 URL-safe characters. It exists only in the answer
// and in the customer's browser: the database holds its SHA-256 and nothing
// else, and no log line, address or page text ever holds it.
func newToken() (string, error) {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// hashToken is what order_tokens.token_hash holds and what a read compares: the
// lookup is by hash, so the secret itself is never compared.
func hashToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
