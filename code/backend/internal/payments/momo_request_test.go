package payments

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/config"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/orders"
)

func momoAt(api string, origin string) *MoMo {
	return NewMoMo(MoMoConfig{
		PartnerCode: momoPartner, AccessKey: config.NewSecret(momoAccess), SecretKey: config.NewSecret(momoSecretKey),
		APIURL: api, PublicBaseURL: origin,
	}, nil, quiet(), nil)
}

var momoOrder = orders.Locked{ID: "doi-24818", Number: "DOI-24818", Total: 3140000}

// PAY-021 AC-3, SC-6, SC-7: every member of the request for a 3.140.000 VND order, and a
// signature that verifies with the shop's secret and with no other.
func TestTheMoMoRequestCarriesTheStoredAmountAndAVerifiableSignature(t *testing.T) {
	m := momoAt("https://test-payment.momo.vn", "https://shop.example.test")
	// The browser's own origin is never used when the shop's is configured.
	req, err := m.build(momoOrder, "DOI-24818-1", orders.Client{Origin: "https://evil.example"})
	if err != nil {
		t.Fatal(err)
	}
	want := momoCreate{
		PartnerCode: momoPartner, RequestID: "DOI-24818-1", OrderID: "DOI-24818-1", Amount: 3140000,
		OrderInfo:   "Thanh toan don hang DOI-24818", RedirectURL: "https://shop.example.test/payment/return",
		IPNURL: "https://shop.example.test/api/v1/payments/momo/ipn", RequestType: "captureWallet", ExtraData: "", Lang: "en",
		Signature: req.Signature,
	}
	if req != want {
		t.Errorf("got  %+v\nwant %+v", req, want)
	}
	canonical := momoCanonical(req.signedFields(momoAccess))
	wantText := "accessKey=access-key-distinct&amount=3140000&extraData=&ipnUrl=https://shop.example.test/api/v1/payments/momo/ipn" +
		"&orderId=DOI-24818-1&orderInfo=Thanh toan don hang DOI-24818&partnerCode=MOMOTEST" +
		"&redirectUrl=https://shop.example.test/payment/return&requestId=DOI-24818-1&requestType=captureWallet"
	if canonical != wantText {
		t.Errorf("the string to sign:\n%s\n%s", canonical, wantText)
	}
	if !momoVerify(momoSecretKey, req.signedFields(momoAccess), req.Signature) || momoVerify("another", req.signedFields(momoAccess), req.Signature) {
		t.Error("the signature verifies with the wrong secret or not with the right one")
	}
	body, _ := json.Marshal(req)
	if strings.Contains(string(body), momoAccess) || strings.Contains(string(body), momoSecretKey) || !strings.Contains(string(body), `"amount":3140000`) {
		t.Errorf("the body: %s", body)
	}
	next, _ := m.build(momoOrder, "DOI-24818-2", orders.Client{})
	if next.RequestID != "DOI-24818-2" || next.Signature == req.Signature {
		t.Error("a second attempt must have its own reference and signature")
	}
}

// With no configured origin the request's own is used, and a forged host is an error, never an address.
func TestTheMoMoAddressesFollowTheOriginRule(t *testing.T) {
	m := momoAt("https://test-payment.momo.vn", "")
	req, err := m.build(momoOrder, "DOI-24818-1", orders.Client{Origin: "https://shop.example.com"})
	if err != nil || req.RedirectURL != "https://shop.example.com/payment/return" {
		t.Errorf("%v %+v", err, req)
	}
	for _, origin := range []string{"javascript:alert(1)", "https://shop.example.com/x y", "https://a@b.test", ""} {
		if _, err := m.build(momoOrder, "DOI-24818-1", orders.Client{Origin: origin}); err == nil {
			t.Errorf("the origin %q was accepted", origin)
		}
	}
	if _, err := m.build(orders.Locked{Number: "DOI-1", Total: 0}, "DOI-1-1", orders.Client{Origin: "https://shop.example.com"}); err == nil {
		t.Error("an order of no amount was built")
	}
}

type standIn struct {
	status int
	body   string
	delay  time.Duration
	mu     sync.Mutex
	seen   []string
}

func (s *standIn) lines() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.seen...)
}

func (s *standIn) server(t *testing.T) *httptest.Server {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		s.mu.Lock()
		s.seen = append(s.seen, r.Method+" "+r.URL.Path+" "+r.Header.Get("Content-Type")+" "+string(raw))
		s.mu.Unlock()
		if s.delay > 0 {
			select {
			case <-time.After(s.delay):
			case <-r.Context().Done():
			}
		}
		if s.status == http.StatusFound {
			http.Redirect(w, r, "https://elsewhere.example/pay", http.StatusFound)
			return
		}
		w.WriteHeader(s.status)
		_, _ = w.Write([]byte(s.body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// SC-8, E3: what MoMo can answer, and that every answer that is not an address is an error.
func TestMoMoMustGiveAUsableAddressOrTheAskFails(t *testing.T) {
	good := `{"resultCode":0,"message":"Successful.","orderId":"DOI-24818-1","payUrl":"https://test-payment.momo.vn/pay/abc"}`
	for name, c := range map[string]struct {
		status  int
		body    string
		delay   time.Duration
		wantURL string
	}{
		"an address":                        {200, good, 0, "https://test-payment.momo.vn/pay/abc"},
		"an address with no orderId echoed": {200, `{"resultCode":0,"payUrl":"https://test-payment.momo.vn/pay/abc"}`, 0, "https://test-payment.momo.vn/pay/abc"},
		"a refusal of the amount":           {200, `{"resultCode":22,"message":"Amount invalid","payUrl":"https://test-payment.momo.vn/x"}`, 0, ""},
		"no address":                        {200, `{"resultCode":0,"message":"Successful."}`, 0, ""},
		"an address that is not http":       {200, `{"resultCode":0,"payUrl":"javascript:alert(1)"}`, 0, ""},
		"another order's reference":         {200, `{"resultCode":0,"orderId":"DOI-1-1","payUrl":"https://test-payment.momo.vn/x"}`, 0, ""},
		"a body that is not JSON":           {200, `<html>`, 0, ""},
		"a server error":                    {500, good, 0, ""},
		"a redirect, never followed":        {302, "", 0, ""},
		"no answer in time":                 {200, good, 5 * time.Second, ""},
	} {
		stub := &standIn{status: c.status, body: c.body, delay: c.delay}
		m := momoAt(stub.server(t).URL, "https://shop.example.test")
		req, _ := m.build(momoOrder, "DOI-24818-1", orders.Client{})
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		got, err := m.ask(ctx, "DOI-24818", req)
		cancel()
		if (err == nil) != (c.wantURL != "") || got != c.wantURL {
			t.Errorf("%s: %q %v", name, got, err)
		}
		seen := stub.lines()
		if len(seen) != 1 || !strings.HasPrefix(seen[0], "POST /v2/gateway/api/create application/json") {
			t.Errorf("%s: the stand-in saw %v (one POST, never a retry)", name, seen)
		}
		for _, line := range seen {
			if strings.Contains(line, momoAccess) || strings.Contains(line, momoSecretKey) {
				t.Errorf("%s: a key left the shop in %s", name, line)
			}
		}
	}
}
