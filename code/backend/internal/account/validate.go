package account

import (
	"strings"
	"unicode/utf8"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/orders"
)

// Every sentence the shop sends for a field, in one place. They are the page's
// own words (src/lib/account-words.ts, W.rules), so the shop's sentence and the
// page's never disagree.
const (
	sentenceName          = "Fill in your name"
	sentenceEmail         = "Enter a valid email address"
	sentencePhone         = "Check the phone number"
	sentencePasswordShort = "Use at least 8 characters"
	sentenceMax100        = "Use at most 100 characters"
	sentenceMax200        = "Use at most 200 characters"
	sentenceReadOnly      = "This cannot be changed here"
	sentenceTaken         = "An account with this email already exists. Sign in instead."
	messageCheck          = "Check the form."
)

const (
	minPassword = 8
	maxPassword = 100
)

// A value holding U+0000 is refused with the field's own sentence: PostgreSQL
// cannot store it and the insert would otherwise fail as a 500.
func hasNUL(s string) bool { return strings.ContainsRune(s, 0) }

// Each rule takes a value already trimmed and answers "" when it is fine.

func nameProblem(s string) string {
	switch n := utf8.RuneCountInString(s); {
	case n == 0 || hasNUL(s):
		return sentenceName
	case n > 100:
		return sentenceMax100
	}
	return ""
}

func phoneProblem(s string) string {
	if orders.ValidPhone(s) {
		return ""
	}
	return sentencePhone
}

func addressProblem(s string) string {
	if utf8.RuneCountInString(s) > 200 || hasNUL(s) {
		return sentenceMax200
	}
	return ""
}

func cityProblem(s string) string {
	if utf8.RuneCountInString(s) > 100 || hasNUL(s) {
		return sentenceMax100
	}
	return ""
}

// passwordProblem counts the password as typed: it is never trimmed.
func passwordProblem(s string) string {
	switch n := utf8.RuneCountInString(s); {
	case n < minPassword:
		return sentencePasswordShort
	case n > maxPassword:
		return sentenceMax100
	}
	return ""
}

// collect keeps the sentences that are not empty.
func collect(found map[string]string) map[string]string {
	out := map[string]string{}
	for field, sentence := range found {
		if sentence != "" {
			out[field] = sentence
		}
	}
	return out
}

// signUpInput is a sign-up as typed; check trims it and returns the email as it
// will be stored.
type signUpInput struct{ name, email, phone, password string }

func (in *signUpInput) check() (email string, faults map[string]string) {
	in.name, in.phone = strings.TrimSpace(in.name), strings.TrimSpace(in.phone)
	email, ok := auth.NormalizeEmail(in.email)
	emailSentence := ""
	if !ok {
		emailSentence = sentenceEmail
	}
	return email, collect(map[string]string{
		"name":     nameProblem(in.name),
		"email":    emailSentence,
		"phone":    phoneProblem(in.phone),
		"password": passwordProblem(in.password),
	})
}
