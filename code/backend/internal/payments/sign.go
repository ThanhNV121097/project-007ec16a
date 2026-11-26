package payments

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"net/url"
	"strings"
)

// canonicalString is the string VNPay's signature covers, made from every vnp_
// parameter except vnp_SecureHash and vnp_SecureHashType. Parameters whose value
// is empty are left out (VNPay's own reference code does so, and the shop's
// address has none). Names are sorted bytewise and each is written
// QueryEscape(name)=QueryEscape(value), a space as +, joined by &: exactly what
// url.Values.Encode makes. The address carries this very string, so the address
// cannot differ from what was signed.
//
// Known limit: this is Go's escaping. VNPay's Java reference does not escape * and
// escapes ~; the values the shop sends and the values VNPay sends back hold neither.
// If a real sandbox notice ever fails to verify, this function and the empty-value
// rule are the two places to look, and they are this one file.
func canonicalString(params url.Values) string {
	signed := make(url.Values, len(params))
	for name, values := range params {
		if !strings.HasPrefix(name, "vnp_") || name == "vnp_SecureHash" || name == "vnp_SecureHashType" {
			continue
		}
		if len(values) == 0 || values[0] == "" {
			continue
		}
		signed[name] = values[:1]
	}
	return signed.Encode()
}

// sign is the lower-case hex HMAC-SHA512 of s, keyed with the hash secret.
func sign(secret, s string) string {
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write([]byte(s))
	return hex.EncodeToString(mac.Sum(nil))
}

// verify says whether q carries a vnp_SecureHash that is the signature of its own
// vnp_ parameters. The received hash must be exactly 128 hex characters and is
// compared in constant time, so its case does not matter.
func verify(secret string, q url.Values) bool {
	received := q.Get("vnp_SecureHash")
	if len(received) != 128 {
		return false
	}
	raw, err := hex.DecodeString(received)
	if err != nil {
		return false
	}
	mac := hmac.New(sha512.New, []byte(secret))
	mac.Write([]byte(canonicalString(q)))
	return hmac.Equal(raw, mac.Sum(nil))
}
