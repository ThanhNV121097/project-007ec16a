package payments

import (
	"context"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/config"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/orders"
)

// Bank is the provider for the way to pay "bank": a VietQR transfer to the shop's own
// account. It satisfies orders.Provider and orders.Transferer. A bank order has no
// payment address, no attempt and no notice: its QR code is drawn from the order
// and the dealer marks it paid when the money has landed. It is registered even
// with no account, so an existing bank order of a shop whose owner removed the
// account answers 503, not "not available yet"; whether it is listed is the
// server's Offered list.
type Bank struct{ account config.Bank }

// NewBank makes the provider from the owner's four settings.
func NewBank(account config.Bank) *Bank { return &Bank{account: account} }

// Available is true only when all four settings are set and valid.
func (b *Bank) Available() bool { return b.account.Set() }

// PaymentURL is always empty and writes nothing: there is no page to go to.
func (*Bank) PaymentURL(context.Context, orders.Payable, orders.Client) (string, error) {
	return "", nil
}

// Transfer builds what the order page shows from the settings and the order's
// stored number and total. The caller reads both from the order, never from the
// request.
func (b *Bank) Transfer(number string, total int64) (orders.TransferDetails, error) {
	if !b.Available() {
		return orders.TransferDetails{}, orders.ErrPaymentUnavailable
	}
	payload, err := vietQR(b.account.BIN, b.account.AccountNumber, total, number)
	if err != nil {
		return orders.TransferDetails{}, err
	}
	rows, err := qrRows(payload)
	if err != nil {
		return orders.TransferDetails{}, err
	}
	return orders.TransferDetails{
		Holder: b.account.AccountName, Bank: b.account.Name, AccountNumber: b.account.AccountNumber,
		Amount: total, Note: number, Payload: payload, QR: rows,
	}, nil
}
