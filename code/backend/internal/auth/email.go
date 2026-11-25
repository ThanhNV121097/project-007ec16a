package auth

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// NormalizeEmail is the one rule for every write to users.email: surrounding
// whitespace trimmed, printable ASCII only (no space), 3 to 200 characters,
// exactly one @, something before it and a domain with a dot that neither
// starts nor ends with one; the result is lower case. It returns ("", false)
// for anything else. ASCII is checked before lower-casing, so a letter that
// only lower-cases into ASCII (the Kelvin sign) is refused, not folded onto a
// second spelling of an existing address.
func NormalizeEmail(raw string) (string, bool) {
	s := strings.TrimSpace(raw)
	if len(s) < 3 || len(s) > 200 {
		return "", false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '!' || s[i] > '~' {
			return "", false
		}
	}
	if strings.Count(s, "@") != 1 {
		return "", false
	}
	local, domain, _ := strings.Cut(s, "@")
	if local == "" || !strings.Contains(domain, ".") || strings.HasPrefix(domain, ".") || strings.HasSuffix(domain, ".") {
		return "", false
	}
	return strings.ToLower(s), true
}

// DisplayName is the name shown for an account: the stored name when it is not
// blank; otherwise the part of the email before @, split on ".", "_" and "-",
// each word's first letter upper-cased and the words joined with one space
// (lan.tran@... is Lan Tran, dealer@... is Dealer); the email itself when
// nothing is left.
func DisplayName(name, email string) string {
	if strings.TrimSpace(name) != "" {
		return name
	}
	local, _, _ := strings.Cut(email, "@")
	words := strings.FieldsFunc(local, func(r rune) bool { return r == '.' || r == '_' || r == '-' })
	for i, w := range words {
		first, size := utf8.DecodeRuneInString(w)
		words[i] = string(unicode.ToUpper(first)) + w[size:]
	}
	if len(words) == 0 {
		return email
	}
	return strings.Join(words, " ")
}
