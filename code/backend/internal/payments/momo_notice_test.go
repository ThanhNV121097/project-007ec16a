package payments

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/config"
)

const (
	momoPartner   = "MOMOTEST"
	momoAccess    = "access-key-distinct"
	momoSecretKey = "secret-key-distinct"
)

func newMoMoProvider(f *fakeStore, log *slog.Logger) *MoMo {
	m := NewMoMo(MoMoConfig{
		PartnerCode: momoPartner, AccessKey: config.NewSecret(momoAccess), SecretKey: config.NewSecret(momoSecretKey),
		APIURL: "https://test-payment.momo.vn", PublicBaseURL: "https://shop.example.test",
	}, nil, log, nil)
	m.store = f
	return m
}

func goodMoMoNotice() map[string]any {
	return map[string]any{
		"partnerCode": momoPartner, "orderId": "DOI-24818-1", "requestId": "DOI-24818-1", "amount": 3140000,
		"orderInfo": "Thanh toan don hang DOI-24818", "orderType": "momo_wallet", "transId": 4014012345, "resultCode": 0,
		"message": "Successful.", "payType": "qr", "responseTime": 1795691100000, "extraData": "",
	}
}

// momoWith copies the notice and applies the changes; a nil value deletes the member.
func momoWith(base map[string]any, changes map[string]any) map[string]any {
	out := map[string]any{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range changes {
		if v == nil {
			delete(out, k)
		} else {
			out[k] = v
		}
	}
	return out
}

// signedBody is what MoMo's server sends: the members and the signature of the signed ones.
func signedBody(t *testing.T, fields map[string]any, key string) []byte {
	t.Helper()
	signed := map[string]string{"accessKey": momoAccess}
	for _, name := range []string{"amount", "extraData", "message", "orderId", "orderInfo", "orderType", "partnerCode", "payType", "requestId", "responseTime", "resultCode", "transId"} {
		if v, ok := fields[name]; ok && v != nil {
			signed[name] = fmt.Sprint(v)
		} else {
			signed[name] = ""
		}
	}
	out := momoWith(fields, nil)
	out["signature"] = momoSignature(key, signed)
	body, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func postMoMo(h http.Handler, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/v1/payments/momo/ipn", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func momoMux(m *MoMo) http.Handler {
	mux := http.NewServeMux()
	Register(mux, NewVNPay(VNPayConfig{}, nil, quiet(), nil), m)
	return mux
}

func flipLast(s string) string {
	return s[:len(s)-1] + map[bool]string{true: "1", false: "0"}[s[len(s)-1] == '0']
}

// PAY-021 AC-4 to AC-6, SC-25: every row of the contract, as 204, 400 or 500.
func TestTheMoMoNoticeAnswersEveryRowOfTheContract(t *testing.T) {
	good := goodMoMoNotice()
	valid := signedBody(t, good, momoSecretKey)
	var sig struct{ Signature string }
	_ = json.Unmarshal(valid, &sig)
	bodyWithSignature := func(s string) []byte {
		var m map[string]any
		_ = json.Unmarshal(valid, &m)
		m["signature"] = s
		b, _ := json.Marshal(m)
		return b
	}
	paid := unpaidAttempt()
	paid.Paid = true
	cases := []struct {
		name      string
		body      []byte
		store     fakeStore
		want      int
		successes int
		failures  int
		finds     int
	}{
		{"a signed success for the right amount", valid, fakeStore{attempt: unpaidAttempt(), wonRace: true}, 204, 1, 0, 1},
		{"a success a twin already recorded", valid, fakeStore{attempt: unpaidAttempt()}, 204, 0, 0, 1},
		{"an order already paid", valid, fakeStore{attempt: paid, wonRace: true}, 204, 0, 0, 1},
		{"the signature in capitals", bodyWithSignature(strings.ToUpper(sig.Signature)), fakeStore{attempt: unpaidAttempt(), wonRace: true}, 204, 1, 0, 1},
		{"a member MoMo added", signedBody(t, momoWith(good, map[string]any{"newMember": "x"}), momoSecretKey), fakeStore{attempt: unpaidAttempt(), wonRace: true}, 204, 1, 0, 1},
		{"no payType, which is optional", signedBody(t, momoWith(good, map[string]any{"payType": nil}), momoSecretKey), fakeStore{attempt: unpaidAttempt(), wonRace: true}, 204, 1, 0, 1},
		{"a customer who cancelled", signedBody(t, momoWith(good, map[string]any{"resultCode": 1006}), momoSecretKey), fakeStore{attempt: unpaidAttempt(), wonRace: true}, 204, 0, 1, 1},
		{"a signature one character off", bodyWithSignature(flipLast(sig.Signature)), fakeStore{attempt: unpaidAttempt(), wonRace: true}, 400, 0, 0, 0},
		{"a signature of 63 characters", bodyWithSignature(sig.Signature[:63]), fakeStore{attempt: unpaidAttempt(), wonRace: true}, 400, 0, 0, 0},
		{"a signature that is not hexadecimal", bodyWithSignature(strings.Repeat("z", 64)), fakeStore{attempt: unpaidAttempt(), wonRace: true}, 400, 0, 0, 0},
		{"signed with another secret", signedBody(t, good, "another"), fakeStore{attempt: unpaidAttempt(), wonRace: true}, 400, 0, 0, 0},
		{"another partner, signed correctly", signedBody(t, momoWith(good, map[string]any{"partnerCode": "OTHER"}), momoSecretKey), fakeStore{attempt: unpaidAttempt(), wonRace: true}, 400, 0, 0, 0},
		{"a reference that is no MoMo attempt's", valid, fakeStore{findErr: errNoAttempt}, 400, 0, 0, 1},
		{"a wrong amount", signedBody(t, momoWith(good, map[string]any{"amount": 1000}), momoSecretKey), fakeStore{attempt: unpaidAttempt(), wonRace: true}, 400, 0, 0, 1},
		{"a missing required member", signedBody(t, momoWith(good, map[string]any{"transId": nil}), momoSecretKey), fakeStore{attempt: unpaidAttempt(), wonRace: true}, 400, 0, 0, 0},
		{"an amount that is a string", signedBody(t, momoWith(good, map[string]any{"amount": "3140000"}), momoSecretKey), fakeStore{attempt: unpaidAttempt(), wonRace: true}, 400, 0, 0, 0},
		{"an amount that is null", []byte(strings.Replace(string(valid), `"amount":3140000`, `"amount":null`, 1)), fakeStore{attempt: unpaidAttempt(), wonRace: true}, 400, 0, 0, 0},
		{"an amount with a fraction", signedBody(t, momoWith(good, map[string]any{"amount": 3140000.5}), momoSecretKey), fakeStore{attempt: unpaidAttempt(), wonRace: true}, 400, 0, 0, 0},
		{"a string member that is a number", signedBody(t, momoWith(good, map[string]any{"orderId": 24818}), momoSecretKey), fakeStore{attempt: unpaidAttempt(), wonRace: true}, 400, 0, 0, 0},
		{"a body that is not JSON", []byte("orderId=DOI-24818-1"), fakeStore{attempt: unpaidAttempt()}, 400, 0, 0, 0},
		{"a body that is an array", []byte("[]"), fakeStore{attempt: unpaidAttempt()}, 400, 0, 0, 0},
		{"a body that is null", []byte("null"), fakeStore{attempt: unpaidAttempt()}, 400, 0, 0, 0},
		{"two objects", append(append([]byte{}, valid...), valid...), fakeStore{attempt: unpaidAttempt()}, 400, 0, 0, 0},
		{"no body", nil, fakeStore{attempt: unpaidAttempt()}, 400, 0, 0, 0},
		{"a body over 16 KiB", signedBody(t, momoWith(good, map[string]any{"message": strings.Repeat("a", 17000)}), momoSecretKey), fakeStore{attempt: unpaidAttempt(), wonRace: true}, 400, 0, 0, 0},
		{"a database that fails to find", valid, fakeStore{findErr: errors.New("down")}, 500, 0, 0, 1},
		{"a database that fails to record a success", valid, fakeStore{attempt: unpaidAttempt(), successErr: errors.New("down")}, 500, 0, 0, 1},
		{"a database that fails to record a failure", signedBody(t, momoWith(good, map[string]any{"resultCode": 1006}), momoSecretKey), fakeStore{attempt: unpaidAttempt(), failErr: errors.New("down")}, 500, 0, 0, 1},
	}
	for _, c := range cases {
		store := c.store
		rec := postMoMo(momoMux(newMoMoProvider(&store, quiet())), c.body)
		if rec.Code != c.want || rec.Header().Get("Cache-Control") != "no-store" {
			t.Errorf("%s: %d %v %s", c.name, rec.Code, rec.Header(), rec.Body)
		}
		switch c.want {
		case 204:
			if rec.Body.Len() != 0 {
				t.Errorf("%s: a 204 with a body %s", c.name, rec.Body)
			}
		case 400:
			var env map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil || env["error"] != "bad_request" || env["message"] != "The notice was refused." || len(env) != 2 {
				t.Errorf("%s: the refusal is %s", c.name, rec.Body)
			}
		case 500:
			if !strings.Contains(rec.Body.String(), `"internal"`) {
				t.Errorf("%s: the failure is %s", c.name, rec.Body)
			}
		}
		if len(store.successes) != c.successes || len(store.failures) != c.failures || store.finds != c.finds {
			t.Errorf("%s: %d successes, %d failures, %d finds; want %d, %d, %d", c.name, len(store.successes), len(store.failures), store.finds, c.successes, c.failures, c.finds)
		}
	}
}

// The notice keeps what MoMo said, as MoMo said it: no division by 100, the integer as text.
func TestTheMoMoNoticeKeepsOnlyWhatMoMoSaid(t *testing.T) {
	store := &fakeStore{attempt: unpaidAttempt(), wonRace: true}
	postMoMo(momoMux(newMoMoProvider(store, quiet())), signedBody(t, goodMoMoNotice(), momoSecretKey))
	if len(store.successes) != 1 {
		t.Fatal("no success was recorded")
	}
	got := store.successes[0]
	if got.TransactionNo != "4014012345" || got.BankCode != "qr" || got.Amount != 3140000 || got.ResponseCode != "0" || got.TransactionStatus != "" || got.PaidTime == nil {
		t.Errorf("facts %+v", got)
	}
	store = &fakeStore{attempt: unpaidAttempt(), wonRace: true}
	postMoMo(momoMux(newMoMoProvider(store, quiet())), signedBody(t, momoWith(goodMoMoNotice(), map[string]any{"responseTime": 12}), momoSecretKey))
	if len(store.successes) != 1 || store.successes[0].PaidTime != nil {
		t.Errorf("a time outside 2000 to 2100 refused a payment or was kept: %+v", store.successes)
	}
}

// SC-28: with the settings unset the notice address answers 400 to everything and asks nothing.
func TestWithoutItsSettingsTheMoMoNoticeRefusesEverything(t *testing.T) {
	store := &fakeStore{attempt: unpaidAttempt(), wonRace: true}
	m := NewMoMo(MoMoConfig{}, nil, quiet(), nil)
	m.store = store
	for _, body := range [][]byte{nil, signedBody(t, goodMoMoNotice(), momoSecretKey), []byte("{}")} {
		if rec := postMoMo(momoMux(m), body); rec.Code != http.StatusBadRequest {
			t.Errorf("got %d", rec.Code)
		}
	}
	if store.finds != 0 {
		t.Error("the database was asked")
	}
}

// SC-36: the keys, the signature and the body are in no log line and no response.
func TestTheMoMoLogAndAnswersHoldNoKeyNoSignatureAndNoBody(t *testing.T) {
	var logged bytes.Buffer
	store := &fakeStore{attempt: unpaidAttempt(), wonRace: true}
	h := momoMux(newMoMoProvider(store, slog.New(slog.NewTextHandler(&logged, nil))))
	body := signedBody(t, goodMoMoNotice(), momoSecretKey)
	var sig struct{ Signature string }
	_ = json.Unmarshal(body, &sig)
	bad := postMoMo(h, signedBody(t, goodMoMoNotice(), "another"))
	good := postMoMo(h, body)
	failing := newMoMoProvider(&fakeStore{findErr: errors.New("down")}, slog.New(slog.NewTextHandler(&logged, nil)))
	down := postMoMo(momoMux(failing), body)
	for _, text := range []string{logged.String(), bad.Body.String(), good.Body.String(), down.Body.String()} {
		for _, leak := range []string{momoAccess, momoSecretKey, sig.Signature, "4014012345"} {
			if strings.Contains(text, leak) {
				t.Errorf("%q is in %q", leak, text)
			}
		}
	}
	if !strings.Contains(logged.String(), "DOI-24818") {
		t.Errorf("the log does not name the order: %s", logged.String())
	}
}

// The route table: only POST reaches MoMo's notice; the VNPay address is not shadowed.
func TestOnlyAPostReachesTheMoMoNotice(t *testing.T) {
	h := momoMux(newMoMoProvider(&fakeStore{attempt: unpaidAttempt()}, quiet()))
	get := httptest.NewRecorder()
	h.ServeHTTP(get, httptest.NewRequest(http.MethodGet, "/v1/payments/momo/ipn", nil))
	if get.Code == http.StatusOK || get.Code == http.StatusNoContent || get.Code == http.StatusBadRequest {
		t.Errorf("a GET reached the MoMo notice: %d", get.Code)
	}
}
