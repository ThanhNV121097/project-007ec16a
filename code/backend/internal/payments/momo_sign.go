package payments

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"sort"
	"strings"
)

// momoCanonical is MoMo's string to sign: every field as name=value, the names in
// alphabetical order, joined by &. The values are exactly as sent or received.
func momoCanonical(fields map[string]string) string {
	names := make([]string, 0, len(fields))
	for name := range fields {
		names = append(names, name)
	}
	sort.Strings(names)
	var b strings.Builder
	for i, name := range names {
		if i > 0 {
			b.WriteByte('&')
		}
		b.WriteString(name)
		b.WriteByte('=')
		b.WriteString(fields[name])
	}
	return b.String()
}

// hmacSHA256 is the lower-case hex HMAC-SHA256 of text, keyed with secret.
func hmacSHA256(secret, text string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(text))
	return hex.EncodeToString(mac.Sum(nil))
}

// momoSignature signs the fields with the shop's secret key.
func momoSignature(secret string, fields map[string]string) string {
	return hmacSHA256(secret, momoCanonical(fields))
}

// momoVerify says whether signature is the signature of fields. The received
// signature is hex-decoded (a wrong length or a character that is not hex fails at
// once; case does not matter) and compared in constant time.
func momoVerify(secret string, fields map[string]string, signature string) bool {
	got, err := hex.DecodeString(signature)
	if err != nil || len(got) != sha256.Size {
		return false
	}
	want, err := hex.DecodeString(momoSignature(secret, fields))
	return err == nil && hmac.Equal(got, want)
}
