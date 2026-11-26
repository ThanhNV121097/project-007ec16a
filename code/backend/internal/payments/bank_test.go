package payments

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/config"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/orders"
)

var shopAccount = config.Bank{BIN: "970436", Name: "Vietcombank", AccountNumber: "0071001234567", AccountName: "NGUYEN VAN AN"}

// PAY-020 items 1 and 2: the details are the settings plus the stored amount and number.
func TestTheBankGivesTheSettingsTheStoredAmountAndTheNumber(t *testing.T) {
	b := NewBank(shopAccount)
	if !b.Available() {
		t.Fatal("a shop with its account set cannot take transfers")
	}
	d, err := b.Transfer("DOI-24818", 3140000)
	if err != nil {
		t.Fatal(err)
	}
	want, _ := vietQR("970436", "0071001234567", 3140000, "DOI-24818")
	if d.Holder != "NGUYEN VAN AN" || d.Bank != "Vietcombank" || d.AccountNumber != "0071001234567" ||
		d.Amount != 3140000 || d.Note != "DOI-24818" || d.Payload != want || len(d.QR) < 21 {
		t.Errorf("details %+v", d)
	}
}

func TestTheBankWithNoAccountIsUnavailableAndGivesNothing(t *testing.T) {
	b := NewBank(config.Bank{})
	if b.Available() {
		t.Error("no account, yet available")
	}
	if _, err := b.Transfer("DOI-1", 1000); !errors.Is(err, orders.ErrPaymentUnavailable) {
		t.Errorf("got %v", err)
	}
	if _, err := NewBank(shopAccount).Transfer(strings.Repeat("D", 26), 1000); err == nil {
		t.Error("a note over 25 characters was built")
	}
}

// A bank order has no payment address and writes no attempt.
func TestTheBankHasNoPaymentAddress(t *testing.T) {
	url, err := NewBank(shopAccount).PaymentURL(context.Background(), orders.Payable{}, orders.Client{})
	if url != "" || err != nil {
		t.Errorf("%q %v", url, err)
	}
}
