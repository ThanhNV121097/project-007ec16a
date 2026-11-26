package config

import (
	"errors"
	"net/url"
	"strings"
)

// DefaultVNPayPage is VNPay's sandbox payment page. Going live changes
// VNPAY_PAYMENT_URL, never code.
const DefaultVNPayPage = "https://sandbox.vnpayment.vn/paymentv2/vpcpay.html"

// paymentPage reads VNPAY_PAYMENT_URL: an absolute http or https address with a
// host and no query and no fragment, because the signed query string is appended
// to it. A blank value is the sandbox. The error names the key, never its value.
func paymentPage(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return DefaultVNPayPage, nil
	}
	u, err := url.Parse(raw)
	if err != nil || !webScheme(u) || u.Host == "" || u.User != nil || strings.ContainsAny(raw, "?#") {
		return "", errors.New("VNPAY_PAYMENT_URL must be an http or https address with a host and no query or fragment")
	}
	return raw, nil
}

// publicOrigin reads PUBLIC_BASE_URL: the shop's own origin, a scheme and a host
// with an optional port and nothing after it. A trailing slash is removed. Blank
// means the origin is read from the request. The error names the key, never its value.
func publicOrigin(raw string) (string, error) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return "", nil
	}
	u, err := url.Parse(raw)
	if err != nil || !webScheme(u) || u.Host == "" || u.User != nil || u.Path != "" || strings.ContainsAny(raw, "?#") {
		return "", errors.New("PUBLIC_BASE_URL must be a scheme and a host, such as https://shop.example.com, with no path, query or fragment")
	}
	return raw, nil
}

func webScheme(u *url.URL) bool { return u.Scheme == "http" || u.Scheme == "https" }
