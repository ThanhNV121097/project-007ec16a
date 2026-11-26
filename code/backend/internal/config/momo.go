package config

import (
	"net/url"
	"strings"
)

// DefaultMoMoAPI is MoMo's sandbox. Going live changes MOMO_API_URL, never code.
const DefaultMoMoAPI = "https://test-payment.momo.vn"

// MoMo is the owner's MoMo merchant settings. MoMo is offered only while the
// three keys are set and the address is valid. The two keys print as [redacted]
// however the config is printed.
type MoMo struct {
	PartnerCode string
	AccessKey   Secret
	SecretKey   Secret
	// APIURL is MoMo's base address, with no trailing slash.
	APIURL string
}

// Set says whether MoMo can be offered.
func (m MoMo) Set() bool {
	return m.PartnerCode != "" && m.AccessKey.IsSet() && m.SecretKey.IsSet() && m.APIURL != ""
}

// momoFrom reads the four MOMO_ settings. None of the three keys given is a
// normal state and says nothing. Anything else that is blank or malformed leaves
// MoMo not set and returns one warning per such key, naming the key and never its
// value. The server starts either way.
func momoFrom(get func(string) string) (MoMo, []string) {
	partner := strings.TrimSpace(get("MOMO_PARTNER_CODE"))
	access := NewSecret(get("MOMO_ACCESS_KEY"))
	secret := NewSecret(get("MOMO_SECRET_KEY"))
	if partner == "" && !access.IsSet() && !secret.IsSet() {
		return MoMo{}, nil
	}
	var warnings []string
	check := func(key string, ok bool) {
		if !ok {
			warnings = append(warnings, key+" is not valid; MoMo is not offered")
		}
	}
	check("MOMO_PARTNER_CODE", partner != "")
	check("MOMO_ACCESS_KEY", access.IsSet())
	check("MOMO_SECRET_KEY", secret.IsSet())
	api, ok := momoAPI(get("MOMO_API_URL"))
	check("MOMO_API_URL", ok)
	if len(warnings) > 0 {
		return MoMo{}, warnings
	}
	return MoMo{PartnerCode: partner, AccessKey: access, SecretKey: secret, APIURL: api}, nil
}

// momoAPI reads MOMO_API_URL: blank is the sandbox; otherwise an absolute http or
// https address with a host and no query, fragment or user info. A trailing slash
// is removed.
func momoAPI(raw string) (string, bool) {
	raw = strings.TrimRight(strings.TrimSpace(raw), "/")
	if raw == "" {
		return DefaultMoMoAPI, true
	}
	u, err := url.Parse(raw)
	if err != nil || !webScheme(u) || u.Host == "" || u.User != nil || strings.ContainsAny(raw, "?#") {
		return "", false
	}
	return raw, true
}
