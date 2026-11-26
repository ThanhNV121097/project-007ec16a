package orders

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
)

func TestNumberKeyReadsTheDigitsOfWhatWasTyped(t *testing.T) {
	for _, c := range []struct {
		in, want string
		ok       bool
	}{
		{"DOI-24818", "24818", true},
		{"doi 24818", "24818", true},
		{"24818", "24818", true},
		{" d-o-i 2 4 8 1 8 ", "24818", true},
		{"123456789012345678", "123456789012345678", true},
		{"1234567890123456789", "1234567890123456789", false},
		{"DOI-", "", false},
		{"", "", false},
	} {
		got, ok := NumberKey(c.in)
		if got != c.want || ok != c.ok {
			t.Errorf("NumberKey(%q) = %q, %v; want %q, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestPhoneKeyReadsALeading84AsZeroAndComparesInFull(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"+84 912 345 678", "0912345678"},
		{"+84912345678", "0912345678"},
		{"0912 345 678", "0912345678"},
		{"(0912) 345-678", "0912345678"},
		{"", ""},
		{"call me", ""},
	} {
		if got := PhoneKey(c.in); got != c.want {
			t.Errorf("PhoneKey(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	if PhoneKey("552") == PhoneKey("0912345552") {
		t.Error("a part of a phone matched the whole")
	}
}

func TestValidPhoneIsTheCheckoutRule(t *testing.T) {
	for in, want := range map[string]bool{
		"0912 345 678":       true,
		"+84 (912) 345-678":  true,
		"091234567":          true,
		"0912 34":            false,
		"09123456":           false,
		"012345678901234567890": false,
		"0912345678\x00":     false,
		"":                   false,
	} {
		if got := ValidPhone(in); got != want {
			t.Errorf("ValidPhone(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestAnAccountOrderIsExactlyFiveKeysWithTheShopsClock(t *testing.T) {
	at := time.Date(2026, 11, 25, 17, 30, 0, 0, time.UTC)
	raw, err := json.Marshal(AccountOrder{ID: "doi-24817", Number: "DOI-24817", Status: "shipped", PlacedAt: at, Total: 6180000})
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != 5 || got["id"] != "doi-24817" || got["number"] != "DOI-24817" || got["status"] != "shipped" || got["total"] != float64(6180000) {
		t.Errorf("an account order: %s", raw)
	}
	if got["placed_at"] != "2026-11-26T00:30:00+07:00" {
		t.Errorf("placed_at: %v", got["placed_at"])
	}
}

// countingQ answers every read with no row and counts the reads.
type countingQ struct {
	calls int
	args  []any
}

type noRow struct{}

func (noRow) Scan(...any) error { return pgx.ErrNoRows }

func (c *countingQ) QueryRow(_ context.Context, _ string, args ...any) pgx.Row {
	c.calls++
	c.args = args
	return noRow{}
}

const someAccount = "6f1c1e2a-0d4b-4c58-9b53-2f1f0f0a1111"

func TestReadCostsNoQueryForAGuestWithNoWayIn(t *testing.T) {
	for _, a := range []Access{{}, {Token: "not a token!"}, {AccountID: "not-a-uuid"}} {
		q := &countingQ{}
		if _, err := Read(context.Background(), q, "doi-24817", a); !errors.Is(err, ErrNotFound) || q.calls != 0 {
			t.Errorf("%+v: %v after %d queries", a, err, q.calls)
		}
	}
	q := &countingQ{}
	if _, err := Read(context.Background(), q, "DOI 24817", Access{AccountID: someAccount}); !errors.Is(err, ErrNotFound) || q.calls != 0 {
		t.Errorf("a bad id: %v after %d queries", err, q.calls)
	}
}

func TestReadAsksOnceForACustomerWithNoTokenAndForAGuestWithOne(t *testing.T) {
	q := &countingQ{}
	if _, err := Read(context.Background(), q, "doi-24817", Access{AccountID: someAccount}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("a customer with no token: %v", err)
	}
	if q.calls != 1 || q.args[1] != nil || q.args[2] != someAccount {
		t.Errorf("a customer with no token: %d queries, args %v", q.calls, q.args)
	}
	q = &countingQ{}
	if _, err := Read(context.Background(), q, "doi-24817", Access{Token: "abcdefghijklmnopqrstuvwxyz012345"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("a guest with a token: %v", err)
	}
	hash, ok := q.args[1].([]byte)
	if q.calls != 1 || !ok || len(hash) != 32 || q.args[2] != nil {
		t.Errorf("a guest with a token: %d queries, args %v", q.calls, q.args)
	}
}
