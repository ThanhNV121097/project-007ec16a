package account

import (
	"errors"
	"net/http"
	"strings"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
)

// details is the saved delivery details as every answer says them: exactly
// these five keys, in this order, and never a password or a hash.
type details struct {
	Name    string `json:"name"`
	Email   string `json:"email"`
	Phone   string `json:"phone"`
	Address string `json:"address"`
	City    string `json:"city"`
}

func detailsOf(p auth.Profile) details {
	return details{Name: p.Name, Email: p.Email, Phone: p.Phone, Address: p.Address, City: p.City}
}

// noAccount is the answer when the account vanished since the session was read.
func noAccount(w http.ResponseWriter) {
	refuse(w, http.StatusUnauthorized, "unauthenticated", "Sign in to continue.", nil)
}

// profile is GET /v1/account: the session's own details.
func (h *handler) profile(w http.ResponseWriter, r *http.Request) {
	u, ok := caller(w, r)
	if !ok {
		return
	}
	ctx, cancel := withTimeout(r)
	defer cancel()
	p, err := auth.GetProfile(ctx, h.db, u.ID)
	if errors.Is(err, auth.ErrNoAccount) {
		noAccount(w)
		return
	}
	if err != nil {
		h.internal(w, r, err)
		return
	}
	reply(w, http.StatusOK, detailsOf(p))
}

// saveProfile is PUT /v1/account. After Require: media type (415), the body
// (413, 400 for anything but one object), every key must be one the route knows
// and every one of the four details a string (400), then every fault at once
// (422: email, role and password whatever their value, and each limit), then the
// write. A refused body changes nothing, even for the other fields in it.
func (h *handler) saveProfile(w http.ResponseWriter, r *http.Request) {
	u, ok := caller(w, r)
	if !ok {
		return
	}
	obj, ok := readObject(w, r)
	if !ok {
		return
	}
	faults, bad := obj.undefined("name", "phone", "address", "city", "email", "role", "password")
	var patch auth.ProfilePatch
	for key, dst := range map[string]**string{"name": &patch.Name, "phone": &patch.Phone, "address": &patch.Address, "city": &patch.City} {
		value, k := obj.text(key)
		switch k {
		case isText:
			value = strings.TrimSpace(value)
			*dst = &value
		case isNull, wrongType:
			faults[key], bad = wrongTypeWords, true
		}
	}
	if bad {
		badRequest(w, faults)
		return
	}

	problems := map[string]string{}
	for _, key := range []string{"email", "role", "password"} {
		if _, present := obj[key]; present {
			problems[key] = sentenceReadOnly
		}
	}
	checks := []struct {
		key   string
		value *string
		rule  func(string) string
	}{
		{"name", patch.Name, nameProblem}, {"phone", patch.Phone, phoneProblem},
		{"address", patch.Address, addressProblem}, {"city", patch.City, cityProblem},
	}
	for _, c := range checks {
		if c.value != nil {
			if s := c.rule(*c.value); s != "" {
				problems[c.key] = s
			}
		}
	}
	if len(problems) > 0 {
		refuse(w, http.StatusUnprocessableEntity, "invalid", messageCheck, problems)
		return
	}

	ctx, cancel := withTimeout(r)
	defer cancel()
	// {} changes nothing: the current details are the answer.
	var saved auth.Profile
	var err error
	if patch.Name == nil && patch.Phone == nil && patch.Address == nil && patch.City == nil {
		saved, err = auth.GetProfile(ctx, h.db, u.ID)
	} else {
		saved, err = auth.UpdateProfile(ctx, h.db, u.ID, patch)
	}
	if errors.Is(err, auth.ErrNoAccount) {
		noAccount(w)
		return
	}
	if err != nil {
		h.internal(w, r, err)
		return
	}
	reply(w, http.StatusOK, detailsOf(saved))
}
