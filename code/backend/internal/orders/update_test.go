package orders

import (
	"slices"
	"strings"
	"testing"
)

func sp(s string) *string { return &s }

var sixStatuses = []string{"placed", "confirmed", "packed", "shipped", "delivered", "cancelled"}

// SC-4 to SC-6, SC-10: every (from, to) of the 6 x 6 grid. One step forward, or
// cancelled from the first three; the current status again is a no-op except on
// a delivered or cancelled order, which accepts no status at all.
func TestJudgeAnswersEveryMoveOfTheGrid(t *testing.T) {
	allowed := map[string][]string{
		"placed":    {"placed", "confirmed", "cancelled"},
		"confirmed": {"confirmed", "packed", "cancelled"},
		"packed":    {"packed", "shipped", "cancelled"},
		"shipped":   {"shipped", "delivered"},
		"delivered": {},
		"cancelled": {},
	}
	for _, from := range sixStatuses {
		for _, to := range sixStatuses {
			faults := Judge(Locked{Status: from}, Patch{Status: sp(to)})
			want := "An order cannot go from " + from + " to " + to + "."
			if slices.Contains(allowed[from], to) {
				if len(faults) != 0 {
					t.Errorf("%s -> %s was refused: %v", from, to, faults)
				}
			} else if faults["status"] != want {
				t.Errorf("%s -> %s: %q, want %q", from, to, faults["status"], want)
			}
		}
	}
}

// SC-13, SC-16: a delivered or cancelled order takes only its note; the other
// eight fields are each refused by name. An open order takes every one.
func TestJudgeLocksTheCourierAndCustomerFieldsOfAClosedOrder(t *testing.T) {
	every := Patch{
		Carrier: sp("GHN"), Tracking: sp("x"), ETA: sp("2026-11-26"), Customer: sp("c"), Phone: sp("0912345678"),
		Email: sp(""), Address: sp("a"), City: sp("c"), Note: sp("n"),
	}
	for _, status := range []string{"delivered", "cancelled"} {
		faults := Judge(Locked{Status: status}, every)
		if len(faults) != 8 || faults["note"] != "" {
			t.Errorf("%s: %v", status, faults)
		}
		if faults["address"] != "An order that is "+status+" cannot be changed." {
			t.Errorf("%s: address says %q", status, faults["address"])
		}
		if f := Judge(Locked{Status: status}, Patch{Note: sp("n")}); len(f) != 0 {
			t.Errorf("%s: the note was refused: %v", status, f)
		}
	}
	for _, status := range []string{"placed", "confirmed", "packed", "shipped"} {
		if f := Judge(Locked{Status: status}, every); len(f) != 0 {
			t.Errorf("%s: %v", status, f)
		}
	}
}

// One body may move a shipped order and date it: the rule reads the status
// before the move. The same body on a delivered order is two faults.
func TestJudgeReadsTheStatusBeforeTheMove(t *testing.T) {
	body := Patch{Status: sp("delivered"), ETA: sp("2026-11-27")}
	if f := Judge(Locked{Status: "shipped"}, body); len(f) != 0 {
		t.Errorf("shipped: %v", f)
	}
	if f := Judge(Locked{Status: "delivered"}, body); len(f) != 2 || f["status"] == "" || f["eta"] == "" {
		t.Errorf("delivered: %v", f)
	}
	if f := Judge(Locked{Status: "packed"}, Patch{}); len(f) != 0 {
		t.Errorf("an empty patch: %v", f)
	}
}

func TestPatchNamesWhatWasSentAndNeverAValue(t *testing.T) {
	got := Patch{Note: sp("secret"), Status: sp("packed"), ETA: sp("")}.Names()
	if strings.Join(got, ",") != "status,eta,note" {
		t.Errorf("names %v", got)
	}
	if len(Patch{}.Names()) != 0 {
		t.Error("an empty patch names a field")
	}
}

// SC-2, SC-3: the public statement names no private column; the dealer's names
// the five it adds and none of the bookkeeping the shop keeps to itself.
func TestTheTwoViewsOfAnOrderAreTwoStatements(t *testing.T) {
	for _, name := range []string{"phone", "email", "address", "note", "transaction_no", "is_sample", "idempotency_key", "account_id"} {
		if strings.Contains(viewSQL, name) {
			t.Errorf("the public view names %s", name)
		}
	}
	for _, name := range []string{"'phone', o.phone", "'email', o.email", "'address', o.address", "'note', o.note", "'transaction_no'", "'items'"} {
		if !strings.Contains(dealerViewSQL, name) {
			t.Errorf("the dealer's view lacks %s", name)
		}
	}
	for _, name := range []string{"is_sample", "idempotency_key", "account_id", "source", "invented"} {
		if strings.Contains(dealerViewSQL, name) {
			t.Errorf("the dealer's view names %s", name)
		}
	}
}
