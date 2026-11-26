package payments

import (
	"net/url"
	"strings"
	"testing"
)

// RFC 4231 test case 2 proves the primitive: HMAC-SHA-512, key "Jefe".
func TestSignIsHMACSHA512OfTheRFC4231Vector(t *testing.T) {
	want := "164b7a7bfcf819e2e395fbe73b56e0a387bd64222e831fd610270cd7ea2505549758bf75c05a994a6d034f65f8f0e6fdcaeab1a34d4a6b4b636e070a38bce737"
	if got := sign("Jefe", "what do ya want for nothing?"); got != want {
		t.Fatalf("got %s", got)
	}
}

// The signed string is pinned as a literal: parameters that are not vnp_ or are the
// hash itself are out, an empty value is out, names are sorted, a space is + and
// the return address is escaped.
func TestTheSignedStringIsSortedEscapedAndLeavesOutEmptyValues(t *testing.T) {
	q := url.Values{}
	q.Set("vnp_ReturnUrl", "https://shop.example.com/payment/return")
	q.Set("vnp_OrderInfo", "Thanh toan don hang DOI-24818")
	q.Set("vnp_Amount", "314000000")
	q.Set("vnp_BankCode", "")
	q.Set("vnp_Command", "pay")
	q.Set("utm_source", "mail")
	q.Set("vnp_SecureHash", "abc")
	q.Set("vnp_SecureHashType", "HmacSHA512")
	want := "vnp_Amount=314000000&vnp_Command=pay&vnp_OrderInfo=Thanh+toan+don+hang+DOI-24818&vnp_ReturnUrl=https%3A%2F%2Fshop.example.com%2Fpayment%2Freturn"
	if got := canonicalString(q); got != want {
		t.Fatalf("got  %s\nwant %s", got, want)
	}
}

func signed(secret string, params map[string]string) url.Values {
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	q.Set("vnp_SecureHash", sign(secret, canonicalString(q)))
	return q
}

func TestVerifyAcceptsOnlyTheSignatureOfTheSameParametersAndSecret(t *testing.T) {
	params := map[string]string{"vnp_Amount": "314000000", "vnp_TxnRef": "DOI-24818-1", "vnp_ResponseCode": "00"}
	good := signed("s3cret", params)
	if !verify("s3cret", good) {
		t.Fatal("a correctly signed query was refused")
	}
	if verify("another", good) {
		t.Error("accepted with another secret")
	}

	hash := good.Get("vnp_SecureHash")
	flipped := hash[:127] + map[bool]string{true: "1", false: "0"}[hash[127] == '0']
	for name, mutate := range map[string]func(url.Values){
		"one character off": func(q url.Values) { q.Set("vnp_SecureHash", flipped) },
		"absent":            func(q url.Values) { q.Del("vnp_SecureHash") },
		"127 characters":    func(q url.Values) { q.Set("vnp_SecureHash", hash[:127]) },
		"129 characters":    func(q url.Values) { q.Set("vnp_SecureHash", hash+"0") },
		"not hex":           func(q url.Values) { q.Set("vnp_SecureHash", strings.Repeat("z", 128)) },
		"a changed amount":  func(q url.Values) { q.Set("vnp_Amount", "1000") },
		"an added field":    func(q url.Values) { q.Set("vnp_Extra", "1") },
	} {
		q := url.Values{}
		for k, v := range good {
			q[k] = append([]string(nil), v...)
		}
		mutate(q)
		if verify("s3cret", q) {
			t.Errorf("%s was accepted", name)
		}
	}

	upper := url.Values{}
	for k, v := range good {
		upper[k] = v
	}
	upper.Set("vnp_SecureHash", strings.ToUpper(hash))
	if !verify("s3cret", upper) {
		t.Error("an upper-case hash was refused: the comparison is on the bytes")
	}

	// A parameter that is not vnp_ is not signed, so adding one changes nothing.
	good.Set("utm_source", "mail")
	if !verify("s3cret", good) {
		t.Error("a parameter that is not vnp_ broke the signature")
	}
}
