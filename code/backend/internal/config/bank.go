package config

import (
	"unicode"
	"unicode/utf8"
	"strings"
)

// Bank is the shop's own bank account for VietQR transfers, read from four
// settings the owner supplies. Bank transfer is offered only while all four are
// valid; the account is served by the order's own transfer-details call and by
// nothing else.
type Bank struct {
	// BIN is the bank's six-digit NAPAS identifier (for example 970436).
	BIN string
	// Name is the bank's name as a person reads it.
	Name string
	// AccountNumber is the account's number: 1 to 19 ASCII letters or digits.
	AccountNumber string
	// AccountName is the account holder's name.
	AccountName string
}

// Set says whether all four values were given and valid.
func (b Bank) Set() bool {
	return b.BIN != "" && b.Name != "" && b.AccountNumber != "" && b.AccountName != ""
}

// bankFrom reads the four VIETQR_ settings. Nothing given is a normal state and
// says nothing. Anything given that is blank or malformed leaves the way not set
// and returns one warning per such key, naming the key and never its value.
func bankFrom(get func(string) string) (Bank, []string) {
	bin := strings.TrimSpace(get("VIETQR_BANK_BIN"))
	name := strings.TrimSpace(get("VIETQR_BANK_NAME"))
	number := strings.TrimSpace(get("VIETQR_ACCOUNT_NUMBER"))
	holder := strings.TrimSpace(get("VIETQR_ACCOUNT_NAME"))
	if bin == "" && name == "" && number == "" && holder == "" {
		return Bank{}, nil
	}
	var warnings []string
	check := func(key string, ok bool) {
		if !ok {
			warnings = append(warnings, key+" is not valid; bank transfer is not offered")
		}
	}
	check("VIETQR_BANK_BIN", len(bin) == 6 && allIn(bin, isDigit))
	check("VIETQR_BANK_NAME", plainText(name, 100))
	check("VIETQR_ACCOUNT_NUMBER", len(number) >= 1 && len(number) <= 19 && allIn(number, isAlnum))
	check("VIETQR_ACCOUNT_NAME", plainText(holder, 100))
	if len(warnings) > 0 {
		return Bank{}, warnings
	}
	return Bank{BIN: bin, Name: name, AccountNumber: number, AccountName: holder}, nil
}

func isDigit(r byte) bool { return r >= '0' && r <= '9' }

func isAlnum(r byte) bool {
	return isDigit(r) || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z'
}

func allIn(s string, ok func(byte) bool) bool {
	for i := 0; i < len(s); i++ {
		if !ok(s[i]) {
			return false
		}
	}
	return true
}

// plainText is 1 to max characters of valid UTF-8 with no control character.
func plainText(s string, max int) bool {
	if s == "" || !utf8.ValidString(s) || utf8.RuneCountInString(s) > max {
		return false
	}
	return !strings.ContainsFunc(s, unicode.IsControl)
}
