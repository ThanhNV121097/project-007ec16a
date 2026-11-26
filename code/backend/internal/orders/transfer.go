package orders

import (
	"context"
	"fmt"
	"net/http"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
)

// bankWay is the id of the way to pay whose order is paid by a QR code on its own
// page: it has no payment address and no attempt.
const bankWay = "bank"

// TransferDetails is what the order page needs to show "Pay by bank transfer": the
// shop's account, the amount, the note to type and the QR code. The account is in
// no other answer of the shop.
type TransferDetails struct {
	Holder        string `json:"holder"`
	Bank          string `json:"bank"`
	AccountNumber string `json:"account_number"`
	// Amount is the order's stored total, in whole VND.
	Amount int64 `json:"amount"`
	// Note is the order's number: what the customer types as the transfer note.
	Note string `json:"note"`
	// Payload is the VietQR text the code carries.
	Payload string `json:"payload"`
	// QR is the code, one string of 0 and 1 per row of modules, square, without the
	// quiet zone.
	QR []string `json:"qr"`
}

// Transferer is what a provider that takes bank transfers adds to Provider. This
// package declares it, so the provider satisfies it without importing orders'
// private parts and orders imports nothing from payments.
type Transferer interface {
	// Transfer builds the details from the shop's settings and the order's stored
	// number and total. ErrPaymentUnavailable when the account is not set.
	Transfer(number string, total int64) (TransferDetails, error)
}

// transfer answers GET /v1/orders/{id}/transfer. Order of checks: access (404: the
// order's own token, or its linked customer, through the same authorize as "Pay
// now"), the order's state (409: cancelled, paid, not a bank transfer), the
// account being set (503), then the details (200). Nothing the request carries but
// the token is read: the amount, the note and the account are the shop's own. The
// route writes nothing, and no answer holds a phone, an email, an address or a
// note of the order.
func (s *service) transfer(w http.ResponseWriter, r *http.Request) {
	noStore(w)
	id := r.PathValue("id")
	ctx, cancel := context.WithTimeout(r.Context(), txTimeout)
	defer cancel()
	ref, err := authorize(ctx, s.db, id, Access{Token: r.Header.Get("X-Order-Token")}, modePay)
	if err != nil {
		s.fail(w, r, id, err)
		return
	}
	if refusal := CheckTransfer(PayState{Status: ref.Status, Payment: ref.Payment, Paid: ref.Paid, PlacedAt: ref.PlacedAt}); refusal != nil {
		s.fail(w, r, id, refusal)
		return
	}
	prov := s.providers[bankWay]
	t, ok := prov.(Transferer)
	if !ok || !prov.Available() {
		s.fail(w, r, id, ErrPaymentUnavailable)
		return
	}
	details, err := t.Transfer(ref.Number, ref.Total)
	if err != nil {
		s.fail(w, r, id, fmt.Errorf("transfer details: %w", err))
		return
	}
	httpx.JSON(w, http.StatusOK, details)
}
