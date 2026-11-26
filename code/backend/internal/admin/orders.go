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
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/orders"
	"github.com/jackc/pgx/v5"
)

// orderTimeout bounds one dealer write, lock waits included.
const orderTimeout = 5 * time.Second

// RegisterOrders mounts the dealer's two writes on an order: PUT (a move, the
// courier's details, the customer's details, the note) and DELETE (a cancelled,
// unpaid order). Both sit behind auth.Require(admin, staff), which answers 401 or
// 403 before the body, the id or the database is read. The rules and the SQL are
// internal/orders'; this file is the boundary: it reads the body, judges every
// field it can, locks the order and answers. Routes carry no /api prefix.
func RegisterOrders(mux *http.ServeMux, db Pool, log *slog.Logger) {
	h := &ordersHandler{db: db, log: log}
	guard := auth.Require(auth.Admin, auth.Staff)
	mux.Handle("PUT /v1/orders/{id}", guard(http.HandlerFunc(h.update)))
	mux.Handle("DELETE /v1/orders/{id}", guard(http.HandlerFunc(h.remove)))
}

type ordersHandler struct {
	db  Pool
	log *slog.Logger
}

func orderGone(w http.ResponseWriter) {
	refuse(w, http.StatusNotFound, "not_found", "Not found.", nil)
}

// internal answers a failed statement, a lock that timed out or a database that
// cannot be reached: the generic 500, with the cause (SQLSTATE and constraint
// only) in the log and never a field value.
func (h *ordersHandler) internal(w http.ResponseWriter, operation string, err error) {
	auth.LogFailure(h.log, "order request failed", operation, err)
	refuse(w, http.StatusInternalServerError, "internal", internalMessage, nil)
}

// begin opens a READ COMMITTED transaction whose lock waits end after three
// seconds, so a stuck lock is a retryable 500 and never a hang.
func (h *ordersHandler) begin(ctx context.Context) (pgx.Tx, error) {
	tx, err := h.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, "SET LOCAL lock_timeout = '3s'"); err != nil {
		_ = tx.Rollback(context.WithoutCancel(ctx))
		return nil, err
	}
	return tx, nil
}

// PUT /v1/orders/{id}. The first check that fails answers; the value rules and
// the state rules are collected, so one save names every field at fault, and a
// refused body applies nothing. A missing order is told first (404 beats 422).
func (h *ordersHandler) update(w http.ResponseWriter, r *http.Request) {
	obj, ok := readOrderObject(w, r)
	if !ok {
		return
	}
	id := r.PathValue("id")
	if len(id) > 100 || !orderID.MatchString(id) {
		orderGone(w)
		return
	}
	body := parseOrderBody(obj)
	if len(body.bad) > 0 {
		refuse(w, http.StatusBadRequest, "bad_request", unreadable, body.bad)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), orderTimeout)
	defer cancel()
	tx, err := h.begin(ctx)
	if err != nil {
		h.internal(w, "orders.update", err)
		return
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	stored, err := orders.LockOrder(ctx, tx, id)
	if errors.Is(err, orders.ErrNotFound) {
		orderGone(w)
		return
	}
	if err != nil {
		h.internal(w, "orders.update", err)
		return
	}
	faults := body.faults
	for key, sentence := range orders.Judge(stored, body.patch) {
		if key == "status" && faults[key] != "" {
			continue // a word that is no status keeps its own sentence
		}
		faults[key] = sentence
	}
	if len(faults) > 0 {
		refuse(w, http.StatusUnprocessableEntity, "invalid", checkOrder, faults)
		return
	}
	if err := orders.Update(ctx, tx, stored, body.patch); err != nil {
		h.internal(w, "orders.update", err)
		return
	}
	view, err := orders.ReadDealer(ctx, tx, id)
	if err != nil {
		h.internal(w, "orders.update", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		h.internal(w, "orders.update", err)
		return
	}
	action := "update"
	if body.patch.Status != nil && *body.patch.Status == "cancelled" {
		action = "cancel"
	}
	// The names of what was sent, never a value: a note or an address is a customer's.
	h.log.Info("order write", "action", action, "id", id, "fields", body.patch.Names(), "request_id", httpx.RequestIDFrom(ctx))
	reply(w, http.StatusOK, view)
}

// DELETE /v1/orders/{id}. A request body, if any, is neither read nor limited.
func (h *ordersHandler) remove(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if len(id) > 100 || !orderID.MatchString(id) {
		orderGone(w)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), orderTimeout)
	defer cancel()
	tx, err := h.begin(ctx)
	if err != nil {
		h.internal(w, "orders.delete", err)
		return
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	switch err := orders.Delete(ctx, tx, id); {
	case errors.Is(err, orders.ErrNotFound):
		orderGone(w)
		return
	case errors.Is(err, orders.ErrNotCancelled), errors.Is(err, orders.ErrPaid):
		refuse(w, http.StatusConflict, "conflict", err.Error(), nil)
		return
	case err != nil:
		h.internal(w, "orders.delete", err)
		return
	}
	if err := tx.Commit(ctx); err != nil {
		h.internal(w, "orders.delete", err)
		return
	}
	h.log.Info("order write", "action", "delete", "id", id, "request_id", httpx.RequestIDFrom(ctx))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusNoContent)
}

// readOrderObject reads a PUT's body as one JSON object of at most 64 KiB and
// answers the refusal itself: 415 for another media type, 413 over the limit, 400
// naming `body` for an empty body, anything that is not one object, or data
// after it.
func readOrderObject(w http.ResponseWriter, r *http.Request) (map[string]json.RawMessage, bool) {
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
	refuse(w, http.StatusBadRequest, "bad_request", unreadable, map[string]string{"body": "Send one JSON object."})
	return nil, false
}
