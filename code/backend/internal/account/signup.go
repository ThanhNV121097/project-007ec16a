package account

import (
	"errors"
	"net/http"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/ratelimit"
)

// Ten sign-ups an hour from one network address; the eleventh is refused.
const (
	signUpLimit   = 10
	signUpWindow  = time.Hour
	signUpTimeout = 20 * time.Second
)

// signedUp is the 201 body: exactly these three keys.
type signedUp struct {
	Name  string    `json:"name"`
	Email string    `json:"email"`
	Role  auth.Role `json:"role"`
}

// signUp is POST /v1/account/sign-up. The order of its checks is the contract:
// media type, body, the address limit, every field, the hash, one transaction
// (account and first session), the cookie. There is no SELECT of the email: the
// unique index decides, so two sign-ups at one moment give one 201 and one 409,
// and a taken email costs what a free one costs.
func (h *handler) signUp(w http.ResponseWriter, r *http.Request) {
	obj, ok := readObject(w, r)
	if !ok {
		return
	}
	faults, bad := obj.undefined("name", "email", "phone", "password")
	var in signUpInput
	for key, dst := range map[string]*string{"name": &in.name, "email": &in.email, "phone": &in.phone, "password": &in.password} {
		value, k := obj.text(key)
		if k == wrongType {
			faults[key], bad = wrongTypeWords, true
		}
		*dst = value
	}
	if bad {
		badRequest(w, faults)
		return
	}

	ctx, cancel := withTimeout(r)
	defer cancel()
	wait, err := ratelimit.Hit(ctx, h.db, ratelimit.SignUpAddress,
		ratelimit.Subject(ratelimit.SignUpAddress, httpx.ClientAddr(r)), signUpLimit, signUpWindow)
	if err != nil {
		h.internal(w, r, err)
		return
	}
	if wait > 0 {
		retryAfter(w, wait, signUpWindow)
		refuse(w, http.StatusTooManyRequests, "rate_limited", "Too many tries. Please wait and try again.", nil)
		return
	}

	email, problems := in.check()
	if len(problems) > 0 {
		refuse(w, http.StatusUnprocessableEntity, "invalid", messageCheck, problems)
		return
	}

	// The hash runs outside any transaction, through the same bounded slots as
	// sign-in's verification, and it runs whether or not the email is taken.
	hash, err := auth.HashPassword(in.password)
	if err != nil {
		h.internal(w, r, err)
		return
	}
	hashed, cancelHash := withTimeoutFor(r, signUpTimeout)
	defer cancelHash()
	token, err := h.auth.RegisterCustomer(hashed, h.db, auth.NewUser{
		Email: email, Name: in.name, Phone: in.phone, PasswordHash: hash,
	})
	switch {
	case errors.Is(err, auth.ErrEmailTaken):
		refuse(w, http.StatusConflict, "conflict", "An account with this email already exists.",
			map[string]string{"email": sentenceTaken})
		return
	case err != nil:
		h.internal(w, r, err)
		return
	}
	auth.SetCookie(w, r, token)
	reply(w, http.StatusCreated, signedUp{Name: in.name, Email: email, Role: auth.Customer})
}
