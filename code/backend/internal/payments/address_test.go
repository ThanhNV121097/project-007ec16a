package payments

import (
	"errors"
	"io"
	"log/slog"
	"math"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/config"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/orders"
)

const secret = "hash-secret-9d41"

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func provider(base string) *VNPay {
	return NewVNPay(VNPayConfig{
		TmnCode: "TMN123", HashSecret: config.NewSecret(secret), PaymentURL: config.DefaultVNPayPage, PublicBaseURL: base,
	}, nil, quiet(), nil)
}

var placedAt = time.Date(2026, 11, 26, 10, 0, 0, 0, ict)

func lockedOrder() orders.Locked {
	return orders.Locked{ID: "doi-24818", Number: "DOI-24818", Status: "placed", Payment: "vnpay", Total: 3140000, PlacedAt: placedAt}
}

var shopClient = orders.Client{IP: "203.0.113.9", Origin: "https://shop.example.com"}

func built(t *testing.T, v *VNPay, o orders.Locked, ref string, now time.Time, c orders.Client) (string, url.Values) {
	t.Helper()
	addr, err := v.address(o, ref, now, c)
	if err != nil {
		t.Fatal(err)
	}
	u, err := url.Parse(addr)
	if err != nil {
		t.Fatal(err)
	}
	return addr, u.Query()
}

// PAY-001 AC-1, AC-4, AC-5, AC-6: every parameter of the address for a 3.140.000 VND
// order placed 10 minutes ago, with the signature that verifies.
func TestTheAddressCarriesEveryParameterAndASignatureThatVerifies(t *testing.T) {
	v := provider("")
	addr, q := built(t, v, lockedOrder(), "DOI-24818-1", placedAt.Add(10*time.Minute), shopClient)
	want := map[string]string{
		"vnp_Version": "2.1.0", "vnp_Command": "pay", "vnp_TmnCode": "TMN123", "vnp_Amount": "314000000",
		"vnp_CurrCode": "VND", "vnp_TxnRef": "DOI-24818-1", "vnp_OrderInfo": "Thanh toan don hang DOI-24818",
		"vnp_OrderType": "other", "vnp_Locale": "en", "vnp_ReturnUrl": "https://shop.example.com/payment/return",
		"vnp_IpAddr": "203.0.113.9", "vnp_CreateDate": "20261126101000",
		// The order's deadline, not now + 30: 20 minutes from now.
		"vnp_ExpireDate": "20261126103000",
	}
	for name, value := range want {
		if q.Get(name) != value {
			t.Errorf("%s = %q, want %q", name, q.Get(name), value)
		}
	}
	if q.Get("vnp_BankCode") != "" {
		t.Error("a bank code was sent: the customer chooses on VNPay's page")
	}
	if !strings.HasPrefix(addr, config.DefaultVNPayPage+"?") || !strings.Contains(addr, "&vnp_SecureHash=") || strings.Index(addr, "vnp_SecureHash") < strings.Index(addr, "vnp_ReturnUrl") {
		t.Errorf("the address is not the page, the signed string and the hash last: %s", addr)
	}
	if !verify(secret, q) {
		t.Error("the signature does not verify with the same secret")
	}
	if verify("another", q) {
		t.Error("the signature verifies with another secret")
	}
	if strings.Contains(addr, secret) {
		t.Error("the secret is in the address")
	}
}

// The deadline is the order's own, and a page with less than a second left is not built.
func TestTheExpiryIsTheOrdersDeadlineAndTheLastSecondIsRefused(t *testing.T) {
	v := provider("")
	_, q := built(t, v, lockedOrder(), "DOI-24818-1", placedAt.Add(29*time.Minute+30*time.Second), shopClient)
	if q.Get("vnp_ExpireDate") != "20261126103000" || q.Get("vnp_CreateDate") != "20261126102930" {
		t.Errorf("30 seconds left: create %s, expire %s", q.Get("vnp_CreateDate"), q.Get("vnp_ExpireDate"))
	}
	_, q = built(t, v, lockedOrder(), "DOI-24818-1", placedAt.Add(29*time.Minute+59*time.Second+500*time.Millisecond), shopClient)
	if q.Get("vnp_ExpireDate") != "20261126103000" {
		t.Errorf("just under the deadline: %s", q.Get("vnp_ExpireDate"))
	}
	for _, elapsed := range []time.Duration{30 * time.Minute, 31 * time.Minute} {
		_, err := v.address(lockedOrder(), "DOI-24818-1", placedAt.Add(elapsed), shopClient)
		var refusal *orders.Refusal
		if !errors.As(err, &refusal) || refusal.Message != orders.MsgRunOut {
			t.Errorf("%s after placing: %v", elapsed, err)
		}
	}
}

// PAY-001 AC-2: the amount is the locked row's total, whatever the caller thinks.
func TestTheAmountIsTheLockedRowsTotalTimes100(t *testing.T) {
	v := provider("")
	o := lockedOrder()
	o.Total = 1000
	_, q := built(t, v, o, "DOI-24818-1", placedAt, shopClient)
	if q.Get("vnp_Amount") != "100000" {
		t.Errorf("amount %s", q.Get("vnp_Amount"))
	}
	o.Total = math.MaxInt64/100 + 1
	if _, err := v.address(o, "DOI-24818-1", placedAt, shopClient); err == nil {
		t.Error("a total whose x100 overflows was accepted")
	}
}

// The return address is the shop's own: PUBLIC_BASE_URL over the request's, and a
// forged host that fails the shape is an error, never an address.
func TestTheReturnAddressIsTheShopsOwnOrigin(t *testing.T) {
	_, q := built(t, provider("https://pay.example.com"), lockedOrder(), "DOI-24818-1", placedAt, shopClient)
	if q.Get("vnp_ReturnUrl") != "https://pay.example.com/payment/return" {
		t.Errorf("configured origin: %s", q.Get("vnp_ReturnUrl"))
	}
	_, q = built(t, provider(""), lockedOrder(), "DOI-24818-1", placedAt, orders.Client{IP: "203.0.113.9", Origin: "http://localhost:5173"})
	if q.Get("vnp_ReturnUrl") != "http://localhost:5173/payment/return" {
		t.Errorf("request origin: %s", q.Get("vnp_ReturnUrl"))
	}
	for _, forged := range []string{"", "https://evil.com/x", "https://exa mple.com", "javascript://x", "https://a.com?x=1", "ftp://a.com"} {
		if _, err := provider("").address(lockedOrder(), "DOI-24818-1", placedAt, orders.Client{IP: "1.2.3.4", Origin: forged}); err == nil {
			t.Errorf("origin %q was used", forged)
		}
	}
}

func TestAnAddressThatIsNotAnIPBecomesLoopback(t *testing.T) {
	_, q := built(t, provider(""), lockedOrder(), "DOI-24818-1", placedAt, orders.Client{IP: "not an ip", Origin: "https://shop.example.com"})
	if q.Get("vnp_IpAddr") != "127.0.0.1" {
		t.Errorf("ip %s", q.Get("vnp_IpAddr"))
	}
}

func TestThePaymentPageIsTheBaseOfEveryAddress(t *testing.T) {
	v := NewVNPay(VNPayConfig{TmnCode: "TMN123", HashSecret: config.NewSecret(secret), PaymentURL: "https://pay.vnpay.vn/vpcpay.html"}, nil, quiet(), nil)
	addr, _ := built(t, v, lockedOrder(), "DOI-24818-1", placedAt, shopClient)
	if !strings.HasPrefix(addr, "https://pay.vnpay.vn/vpcpay.html?") {
		t.Errorf("address %s", addr)
	}
}

// PAY-005: both keys, or the provider is not available.
func TestAvailableNeedsBothKeys(t *testing.T) {
	for name, c := range map[string]struct {
		tmn, key string
		want     bool
	}{"both": {"T", "k", true}, "no code": {"", "k", false}, "no secret": {"T", "", false}, "blank": {"  ", "  ", false}, "neither": {"", "", false}} {
		v := NewVNPay(VNPayConfig{TmnCode: c.tmn, HashSecret: config.NewSecret(c.key)}, nil, quiet(), nil)
		if v.Available() != c.want {
			t.Errorf("%s: Available() = %v", name, !c.want)
		}
	}
}

// With a key missing the ask for an address is the 503 of the seam, before any database work.
func TestAskingWithoutKeysIsUnavailableAndTouchesNoDatabase(t *testing.T) {
	v := NewVNPay(VNPayConfig{}, nil, quiet(), nil)
	_, err := v.PaymentURL(t.Context(), orders.Payable{ID: "doi-24818"}, shopClient)
	if !errors.Is(err, orders.ErrPaymentUnavailable) {
		t.Fatalf("got %v", err)
	}
}
