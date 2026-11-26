package payments

import (
	"strings"
	"testing"
)

// RFC 4231 test cases 1 and 2: the independent check of the HMAC-SHA256 the signature uses.
func TestHMACSHA256MatchesRFC4231(t *testing.T) {
	for _, c := range []struct{ key, text, want string }{
		{strings.Repeat("\x0b", 20), "Hi There", "b0344c61d8db38535ca8afceaf0bf12b881dc200c9833da726e9376c2e32cff7"},
		{"Jefe", "what do ya want for nothing?", "5bdcc146bf60754e6a042426089575c75a003f089d2739839dec58b964ec3843"},
	} {
		if got := hmacSHA256(c.key, c.text); got != c.want {
			t.Errorf("hmac(%q) = %s", c.text, got)
		}
	}
}

// The names are signed in alphabetical order, each as name=value, joined by &.
func TestTheStringToSignHasTheNamesInOrderAndValuesAsSent(t *testing.T) {
	got := momoCanonical(map[string]string{
		"requestType": "captureWallet", "accessKey": "AK", "orderInfo": "Thanh toan don hang DOI-24818",
		"orderId": "DOI-24818-1", "extraData": "", "amount": "3140000",
	})
	want := "accessKey=AK&amount=3140000&extraData=&orderId=DOI-24818-1&orderInfo=Thanh toan don hang DOI-24818&requestType=captureWallet"
	if got != want {
		t.Errorf("got  %s\nwant %s", got, want)
	}
}

func TestASignatureVerifiesOnlyWithTheSameSecretAndTheSameFields(t *testing.T) {
	fields := map[string]string{"amount": "3140000", "orderId": "DOI-24818-1", "extraData": ""}
	sig := momoSignature("key-one", fields)
	if len(sig) != 64 || !momoVerify("key-one", fields, sig) || !momoVerify("key-one", fields, strings.ToUpper(sig)) {
		t.Fatalf("a signature does not verify with its own key: %s", sig)
	}
	altered := sig[:63] + map[bool]string{true: "1", false: "0"}[sig[63] == '0']
	for name, c := range map[string]struct {
		key, sig string
		fields   map[string]string
	}{
		"one character off":    {"key-one", altered, fields},
		"another secret":       {"key-two", sig, fields},
		"63 characters":        {"key-one", sig[:63], fields},
		"65 characters":        {"key-one", sig + "0", fields},
		"not hexadecimal":      {"key-one", strings.Repeat("z", 64), fields},
		"empty":                {"key-one", "", fields},
		"another amount":       {"key-one", sig, map[string]string{"amount": "1000", "orderId": "DOI-24818-1", "extraData": ""}},
		"an extra field":       {"key-one", sig, map[string]string{"amount": "3140000", "orderId": "DOI-24818-1", "extraData": "", "x": "1"}},
	} {
		if momoVerify(c.key, c.fields, c.sig) {
			t.Errorf("%s: verified", name)
		}
	}
}
