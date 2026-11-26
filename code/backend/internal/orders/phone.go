package orders

import (
	"strings"
	"unicode/utf8"
)

// The checkout's phone rule: at most 20 characters and at least 9 ASCII digits;
// spaces, +, - and brackets are simply not counted. Checkout, sign-up, the saved
// details and the number-and-phone matcher all use it, from here.
const (
	minPhoneDigits = 9
	maxPhoneChars  = 20
)

// digitsOnly is the ASCII digits of s, in order.
func digitsOnly(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			b.WriteByte(s[i])
		}
	}
	return b.String()
}

// PhoneDigits is the ASCII digits of a typed phone.
func PhoneDigits(raw string) string { return digitsOnly(raw) }

// ValidPhone is the rule above, for a value that is already trimmed. A value
// holding U+0000 is refused: PostgreSQL cannot store it.
func ValidPhone(raw string) bool {
	return !strings.ContainsRune(raw, 0) &&
		utf8.RuneCountInString(raw) <= maxPhoneChars &&
		len(PhoneDigits(raw)) >= minPhoneDigits
}

// PhoneKey is what two phones are compared on: their digits, a leading "84"
// read as "0" (+84 912 345 678 and 0912345678 are one number). It is "" when the
// phone holds no digit, and an empty key never matches anything.
func PhoneKey(raw string) string {
	d := digitsOnly(raw)
	if rest, ok := strings.CutPrefix(d, "84"); ok {
		return "0" + rest
	}
	return d
}
