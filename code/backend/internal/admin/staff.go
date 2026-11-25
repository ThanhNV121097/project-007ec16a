// Package admin holds the endpoints only the shop's team uses. Staff accounts
// (this file and its two neighbours) are the admin's alone: every route is
// behind auth.Require(auth.Admin), and the package holds no SQL: it calls the
// functions of internal/auth.
//
// Routes carry NO /api prefix; the edge proxy strips it.
package admin

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
)

const (
	dbTimeout       = 10 * time.Second
	internalMessage = "Something went wrong. Please try again."
	unreadable      = "The request could not be read."
	adminMessage    = "The admin account cannot be switched off, changed or removed."
)

// RegisterStaff mounts the four staff routes. Any other method on these paths
// falls to the server's catch-all and answers the JSON 404.
func RegisterStaff(mux *http.ServeMux, db Pool, log *slog.Logger) {
	(&staffHandler{store: pgStore{db: db}, log: log}).mount(mux)
}

func (h *staffHandler) mount(mux *http.ServeMux) {
	guard := auth.Require(auth.Admin)
	mux.Handle("GET /v1/admin/staff", guard(http.HandlerFunc(h.list)))
	mux.Handle("POST /v1/admin/staff", guard(http.HandlerFunc(h.create)))
	mux.Handle("PUT /v1/admin/staff/{id}", guard(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /v1/admin/staff/{id}", guard(http.HandlerFunc(h.remove)))
}

type staffHandler struct {
	store store
	log   *slog.Logger
}

// accountBody is an account as every answer says it: exactly these five keys,
// written from this struct and never from a row, so no password or hash can
// reach a body by accident. name is the display name (the admin has none
// stored).
type accountBody struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Active bool   `json:"is_active"`
}

func bodyOf(u auth.User) accountBody {
	return accountBody{ID: u.ID, Name: u.DisplayName(), Email: u.Email, Role: string(u.Role), Active: u.Active}
}

// reply writes a JSON answer that is never cached.
func reply(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, status, v)
}

// refuse writes the error envelope, never cached.
func refuse(w http.ResponseWriter, status int, code, message string, fields map[string]string) {
	w.Header().Set("Cache-Control", "no-store")
	httpx.Error(w, status, code, message, fields)
}

func notFound(w http.ResponseWriter) {
	refuse(w, http.StatusNotFound, "not_found", "That account was not found.", nil)
}

func conflict(w http.ResponseWriter) {
	refuse(w, http.StatusConflict, "conflict", adminMessage, nil)
}

func invalid(w http.ResponseWriter, faults map[string]string) {
	refuse(w, http.StatusUnprocessableEntity, "invalid", "Check the form.", faults)
}

func emailTaken(w http.ResponseWriter) {
	invalid(w, map[string]string{"email": "That email already belongs to an account."})
}

// internal answers a failed statement or hash: the generic 500, and the cause
// (SQLSTATE and constraint only for a database error) in the log.
func (h *staffHandler) internal(w http.ResponseWriter, operation string, err error) {
	auth.LogFailure(h.log, "staff request failed", operation, err)
	refuse(w, http.StatusInternalServerError, "internal", internalMessage, nil)
}

// GET /v1/admin/staff
func (h *staffHandler) list(w http.ResponseWriter, r *http.Request) {
	page, limit, faults := readPaging(r.URL.Query())
	if len(faults) > 0 {
		refuse(w, http.StatusBadRequest, "bad_request", "Some parameters are not valid.", faults)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
	defer cancel()
	users, total, err := h.store.list(ctx, limit, (page-1)*limit)
	if err != nil {
		h.internal(w, "staff.list", err)
		return
	}
	items := make([]accountBody, len(users))
	for i, u := range users {
		items[i] = bodyOf(u)
	}
	reply(w, http.StatusOK, struct {
		Items []accountBody `json:"items"`
		Total int64         `json:"total"`
		Page  int64         `json:"page"`
		Limit int64         `json:"limit"`
	}{items, total, page, limit})
}

// POST /v1/admin/staff
func (h *staffHandler) create(w http.ResponseWriter, r *http.Request) {
	obj, ok := readObject(w, r)
	if !ok {
		return
	}
	in, faults, bad := readInput(obj, createFields)
	if bad {
		refuse(w, http.StatusBadRequest, "bad_request", unreadable, faults)
		return
	}
	name, email, faults := judgeCreate(in)
	ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
	defer cancel()
	if faults["email"] == "" {
		taken, err := h.store.emailTaken(ctx, email)
		if err != nil {
			h.internal(w, "staff.create", err)
			return
		}
		if taken {
			faults["email"] = "That email already belongs to an account."
		}
	}
	if len(faults) > 0 {
		invalid(w, faults)
		return
	}
	// Hashed after every rule passed and before the database is asked: a
	// refused request costs no hashing and no lock is held while it runs.
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		h.internal(w, "staff.create", err)
		return
	}
	u, err := h.store.create(ctx, name, email, hash)
	switch {
	case errors.Is(err, auth.ErrEmailTaken):
		emailTaken(w)
	case err != nil:
		h.internal(w, "staff.create", err)
	default:
		reply(w, http.StatusCreated, bodyOf(u))
	}
}

// target is the id's three checks, shared by PUT and DELETE and made before any
// body is read: a canonical uuid, an account that is not a customer's, and not
// the admin's. It answers the refusal itself.
func (h *staffHandler) target(w http.ResponseWriter, r *http.Request, operation string) (auth.User, bool) {
	id := r.PathValue("id")
	if !auth.IsID(id) {
		notFound(w)
		return auth.User{}, false
	}
	ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
	defer cancel()
	u, err := h.store.lookup(ctx, id)
	switch {
	case errors.Is(err, auth.ErrNotFound):
		notFound(w)
		return auth.User{}, false
	case err != nil:
		h.internal(w, operation, err)
		return auth.User{}, false
	case u.Role == auth.Admin:
		conflict(w)
		return auth.User{}, false
	}
	return u, true
}

// PUT /v1/admin/staff/{id}
func (h *staffHandler) update(w http.ResponseWriter, r *http.Request) {
	target, ok := h.target(w, r, "staff.update")
	if !ok {
		return
	}
	obj, ok := readObject(w, r)
	if !ok {
		return
	}
	in, faults, bad := readInput(obj, updateFields)
	if bad {
		refuse(w, http.StatusBadRequest, "bad_request", unreadable, faults)
		return
	}
	if faults := judgeUpdate(&in); len(faults) > 0 {
		invalid(w, faults)
		return
	}
	if !in.Has["name"] && !in.Has["is_active"] && !in.Has["password"] {
		reply(w, http.StatusOK, bodyOf(target))
		return
	}
	var c change
	if in.Has["name"] {
		c.name = &in.Name
	}
	if in.Has["is_active"] {
		c.active = &in.Active
	}
	if in.Has["password"] {
		hash, err := auth.HashPassword(in.Password)
		if err != nil {
			h.internal(w, "staff.update", err)
			return
		}
		c.hash = &hash
	}
	ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
	defer cancel()
	u, err := h.store.update(ctx, target.ID, c)
	switch {
	case errors.Is(err, auth.ErrNotFound):
		notFound(w)
	case errors.Is(err, errAdminAccount):
		conflict(w)
	case err != nil:
		h.internal(w, "staff.update", err)
	default:
		reply(w, http.StatusOK, bodyOf(u))
	}
}

// DELETE /v1/admin/staff/{id}. A request body, if any, is neither read nor
// limited.
func (h *staffHandler) remove(w http.ResponseWriter, r *http.Request) {
	target, ok := h.target(w, r, "staff.delete")
	if !ok {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
	defer cancel()
	switch err := h.store.remove(ctx, target.ID); {
	case errors.Is(err, auth.ErrNotFound):
		notFound(w)
	case err != nil:
		h.internal(w, "staff.delete", err)
	default:
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusNoContent)
	}
}

// readObject reads the request's body as one JSON object of at most 64 KiB and
// answers the refusal itself: 415 for another media type, 413 for a body over
// the limit, 400 for an empty body, anything that is not one object, or data
// after it.
func readObject(w http.ResponseWriter, r *http.Request) (map[string]json.RawMessage, bool) {
	if mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type")); err != nil || mt != "application/json" {
		refuse(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Send the request as JSON.", nil)
		return nil, false
	}
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBodyBytes))
	var obj map[string]json.RawMessage
	err := dec.Decode(&obj)
	if err == nil {
		if _, next := dec.Token(); next != io.EOF {
			err = errors.New("data after the object")
		}
	}
	if err == nil && obj == nil {
		err = errors.New("not an object")
	}
	if err == nil {
		return obj, true
	}
	var tooBig *http.MaxBytesError
	if errors.As(err, &tooBig) {
		refuse(w, http.StatusRequestEntityTooLarge, "payload_too_large", "The request is too large.", nil)
		return nil, false
	}
	refuse(w, http.StatusBadRequest, "bad_request", unreadable, nil)
	return nil, false
}
