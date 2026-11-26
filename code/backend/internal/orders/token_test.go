package orders

import (
	"regexp"
	"testing"
)

// SC-20: at least 22 URL-safe characters, at least 128 bits of randomness.
func TestATokenIs32URLSafeCharactersAndNeverRepeats(t *testing.T) {
	urlSafe := regexp.MustCompile(`^[A-Za-z0-9_-]{32}$`)
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		token, err := newToken()
		if err != nil {
			t.Fatal(err)
		}
		if !urlSafe.MatchString(token) || seen[token] {
			t.Fatalf("token %q is not 32 URL-safe characters or repeats", token)
		}
		seen[token] = true
	}
}

func TestHashTokenIs32BytesAndDiffersPerToken(t *testing.T) {
	a, b := hashToken("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), hashToken("aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaab")
	if len(a) != 32 || string(a) == string(b) {
		t.Errorf("hashes: %x %x", a, b)
	}
}
