package orders

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

var settledAt = time.Date(2026, 11, 26, 3, 0, 0, 0, time.UTC)

// PAY-001, PAY-003: the four state rules, in order, and the exact deadline counts as over.
func TestCheckPayableAppliesTheRulesInOrder(t *testing.T) {
	fresh := settledAt.Add(10 * time.Minute)
	over := settledAt.Add(31 * time.Minute)
	for _, c := range []struct {
		name string
		in   PayState
		now  time.Time
		want string
	}{
		{"payable", PayState{"placed", "vnpay", false, settledAt}, fresh, ""},
		{"29:59", PayState{"placed", "vnpay", false, settledAt}, settledAt.Add(29*time.Minute + 59*time.Second), ""},
		{"exactly 30:00", PayState{"placed", "vnpay", false, settledAt}, settledAt.Add(30 * time.Minute), MsgRunOut},
		{"31:00", PayState{"placed", "vnpay", false, settledAt}, over, MsgRunOut},
		{"confirmed and packed are payable", PayState{"packed", "vnpay", false, settledAt}, fresh, ""},
		{"cancelled beats everything", PayState{"cancelled", "cod", true, settledAt}, over, msgCancelled},
		{"paid beats a way that is not online", PayState{"placed", "cod", true, settledAt}, over, msgPaid},
		{"not online beats the clock", PayState{"placed", "cod", false, settledAt}, over, msgNotOnline},
		{"a bank transfer is paid by its QR code, not on a page", PayState{"placed", "bank", false, settledAt}, fresh, msgBankOrder},
		{"a bank transfer beats the clock", PayState{"placed", "bank", false, settledAt}, over, msgBankOrder},
		{"an unknown way is not online", PayState{"placed", "barter", false, settledAt}, fresh, msgNotOnline},
		{"momo is payable on its provider's page", PayState{"placed", "momo", false, settledAt}, fresh, ""},
		{"momo's 30 minutes run out like VNPay's", PayState{"placed", "momo", false, settledAt}, over, MsgRunOut},
		{"a paid bank order says paid", PayState{"placed", "bank", true, settledAt}, fresh, msgPaid},
		{"a shipped unpaid order past its window", PayState{"shipped", "vnpay", false, settledAt}, over, MsgRunOut},
	} {
		got := CheckPayable(c.in, c.now)
		switch {
		case c.want == "" && got != nil:
			t.Errorf("%s: refused with %q", c.name, got.Message)
		case c.want != "" && (got == nil || got.Message != c.want):
			t.Errorf("%s: got %v, want %q", c.name, got, c.want)
		}
	}
	messages := map[string]bool{msgCancelled: true, msgPaid: true, msgNotOnline: true, msgNotYet: true, MsgRunOut: true, msgBankOrder: true, msgNotBank: true}
	if len(messages) != 7 || MsgRunOut != "The time to pay this order has run out." {
		t.Error("the sentences must differ and the new one must be exactly as the contract words it")
	}
}

type settleProvider struct{ open bool }

func (p settleProvider) Available() bool { return p.open }
func (settleProvider) PaymentURL(context.Context, Payable, Client) (string, error) {
	return "https://pay.example.test/x", nil
}

// decide is CheckPayable, then no provider, then a provider that is not available;
// the clock is passed in and the provider is never asked.
func TestDecideTakesTheClockAndKeepsTheProvidersAfterTheStateRules(t *testing.T) {
	ref := Ref{ID: "doi-1", Status: "placed", Payment: "vnpay", PlacedAt: settledAt}
	inside, outside := settledAt.Add(5*time.Minute), settledAt.Add(31*time.Minute)
	closed := map[string]Provider{"vnpay": settleProvider{}}
	open := map[string]Provider{"vnpay": settleProvider{open: true}}

	var refusal *Refusal
	if _, err := decide(ref, closed, outside); !errors.As(err, &refusal) || refusal.Message != MsgRunOut {
		t.Errorf("run out with keys missing: %v (a refusal of the state comes before 503)", err)
	}
	if _, err := decide(ref, closed, inside); !errors.Is(err, ErrPaymentUnavailable) {
		t.Errorf("keys missing: %v", err)
	}
	if _, err := decide(ref, nil, inside); !errors.As(err, &refusal) || refusal.Message != msgNotYet {
		t.Errorf("no provider: %v", err)
	}
	if p, err := decide(ref, open, inside); err != nil || p == nil {
		t.Errorf("payable: %v", err)
	}
	if _, err := decide(ref, open, outside); !errors.As(err, &refusal) || refusal.Message != MsgRunOut {
		t.Errorf("run out: %v", err)
	}
}

// PAY-003 item 1 and 4: the expiry's cancel carries the four extra conditions the
// dealer's cancel does not, and both judge the status in their own WHERE.
func TestTheExpiryCancelAddsExactlyTheFourConditions(t *testing.T) {
	for _, part := range []string{"NOT paid", "NOT is_sample", "payment_method_id = ANY($2)", "placed_at <= $3"} {
		if !strings.Contains(cancelExpirySQL, part) {
			t.Errorf("the expiry's cancel lacks %q", part)
		}
		if strings.Contains(cancelSQL, part) {
			t.Errorf("the dealer's cancel holds %q", part)
		}
	}
	for _, sql := range []string{cancelSQL, cancelExpirySQL} {
		if !strings.Contains(sql, "status IN ('placed', 'confirmed', 'packed')") || !strings.Contains(sql, "RETURNING id") {
			t.Errorf("a cancel that does not judge the status in its own WHERE: %s", sql)
		}
	}
	if !strings.Contains(markPaidSQL, "WHERE id = $1 AND NOT paid") || !strings.Contains(markPaidSQL, "WHEN status = 'placed' THEN 'confirmed'") {
		t.Errorf("MarkPaid: %s", markPaidSQL)
	}
}
