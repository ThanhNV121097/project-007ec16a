package orders

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
)

// The sentences a refused "Pay now" can carry. They are different sentences under
// one code, `conflict`: the page branches on none, a person reads the message.
// MsgRunOut (settle.go) is the one for an order whose 30 minutes are over.
const (
	msgCancelled   = "This order was cancelled, so it cannot be paid."
	msgPaid        = "This order is already paid."
	msgNotOnline   = "This order cannot be paid online."
	msgNotYet      = "Online payment is not available yet."
	msgUnavailable = "Online payment is not available right now. Please try again later."
	msgNoBody      = "This call takes no body."
	msgNoBodyField = "This call takes no request body."
)

// Refusal is a call the shop refuses because of the order's state: 409 conflict
// with its message.
type Refusal struct{ Message string }

func (r *Refusal) Error() string { return r.Message }

// ErrPaymentUnavailable means a provider is registered for the order's way to pay
// and its keys are not set: 503 payment_unavailable.
var ErrPaymentUnavailable = errors.New("payment provider unavailable")

// decide is what the route refuses, in order: cancelled, then paid, then a way to
// pay the shop does not complete online, then its 30 minutes having run out (all
// four are CheckPayable, the one function that holds the state rules), then no
// provider registered, then a provider that is not available. It reads the order's
// row, the provider map and the clock passed in, and nothing else. Cancelled comes
// before paid: an order cancelled after it was paid must say it was cancelled.
func decide(ref Ref, providers map[string]Provider, now time.Time) (Provider, error) {
	if r := CheckPayable(PayState{Status: ref.Status, Payment: ref.Payment, Paid: ref.Paid, PlacedAt: ref.PlacedAt}, now); r != nil {
		return nil, r
	}
	p := providers[ref.Payment]
	if p == nil {
		return nil, &Refusal{msgNotYet}
	}
	if !p.Available() {
		return nil, ErrPaymentUnavailable
	}
	return p, nil
}

// payURL runs the state and availability checks and then asks the provider, with a
// 3-second limit, for the address to pay at. An error, a timeout or an empty
// address is an internal error, except a *Refusal the provider found under its
// own lock, which stays the 409 it names. This function writes nothing and makes
// no call of its own: the provider keeps its own attempt.
func (s *service) payURL(ctx context.Context, ref Ref, c Client) (string, error) {
	prov, err := decide(ref, s.providers, s.now())
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, providerTimeout)
	defer cancel()
	url, err := prov.PaymentURL(ctx, Payable{ID: ref.ID, Number: ref.Number, Total: ref.Total, PlacedAt: ref.PlacedAt}, c)
	if err != nil {
		return "", fmt.Errorf("payment address: %w", err)
	}
	if url == "" {
		return "", errors.New("payment address: the provider gave none")
	}
	return url, nil
}

// hasBody reads at most one byte of the request body: a byte read, or a read
// error other than io.EOF, is a body, whatever its content type or declared
// length. Nothing of it is parsed, stored or logged.
func hasBody(r *http.Request) bool {
	if r.Body == nil || r.Body == http.NoBody {
		return false
	}
	var b [1]byte
	n, err := io.ReadFull(io.LimitReader(r.Body, 1), b[:])
	return n > 0 || (err != nil && !errors.Is(err, io.EOF))
}

// payHandler answers POST /v1/orders/{id}/payment: "Pay now". Order of checks:
// access (404), a body was sent (400), the order's state (409), availability (409
// or 503), the address (200, 409 or 500). Every answer is no-store. The access
// statement and the ask for the address share one txTimeout deadline, as GET's
// read does: an access statement that times out is a 500, never a hung request.
func (s *service) payHandler(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	id := r.PathValue("id")
	ctx, cancel := context.WithTimeout(r.Context(), txTimeout)
	defer cancel()
	ref, err := authorize(ctx, s.db, id, Access{Token: r.Header.Get("X-Order-Token"), AccountID: callerAccount(r)}, modePay)
	if err != nil {
		s.fail(w, r, id, err)
		return
	}
	if hasBody(r) {
		httpx.Error(w, http.StatusBadRequest, "bad_request", msgNoBody, fieldErrors{"body": msgNoBodyField})
		return
	}
	url, err := s.payURL(ctx, ref, clientOf(r))
	if err != nil {
		s.fail(w, r, id, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]string{"payment_url": url})
}

// fail is the one error mapper of the two order routes: the not-found body is
// written by one function, so it cannot differ between GET and POST. Anything it
// does not know is a defect of this service: 500 with a generic body, the cause
// in the log beside the order and the request id, never the token.
func (s *service) fail(w http.ResponseWriter, r *http.Request, id string, err error) {
	var refusal *Refusal
	switch {
	case errors.Is(err, ErrNotFound):
		WriteNotFound(w)
	case errors.As(err, &refusal):
		httpx.Error(w, http.StatusConflict, "conflict", refusal.Message, nil)
	case errors.Is(err, ErrPaymentUnavailable):
		httpx.Error(w, http.StatusServiceUnavailable, "payment_unavailable", msgUnavailable, nil)
	default:
		s.log.Error("order request failed", "order", id, "request_id", httpx.RequestIDFrom(r.Context()), "error", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "Something went wrong. Please try again.", nil)
	}
}
