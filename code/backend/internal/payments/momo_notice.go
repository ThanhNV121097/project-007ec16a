package payments

import (
	"bytes"
	"context"
	"crypto/hmac"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
)

const (
	// momoMaxBody is the largest notice read.
	momoMaxBody = 16 << 10
	// momoNoticeTimeout bounds one notice, database included.
	momoNoticeTimeout = 5 * time.Second
)

// momoIntText is the text of an integer member: 1 to 18 digits, an optional minus.
// A fraction, an exponent, a string or null is not one.
var momoIntText = regexp.MustCompile(`^-?[0-9]{1,18}$`)

// momoNotice is a notice that has the right members. Nothing in it is believed yet.
type momoNotice struct {
	partnerCode, orderID, signature, payType string
	amount, transID, resultCode, responseTime int64
	// signed holds every signed member but the access key, as MoMo sent it.
	signed map[string]string
}

// parseMoMoNotice reads the body as one JSON object holding every required member
// with the right type: the strings partnerCode, orderId, requestId and signature,
// the integers amount, transId, resultCode and responseTime. Five more strings may
// be absent (they are "" then). A member it does not know is ignored. The text of
// each signed member is kept as it was sent, so the signature is checked against
// what MoMo signed.
func parseMoMoNotice(body []byte) (momoNotice, bool) {
	dec := json.NewDecoder(bytes.NewReader(body))
	var raw map[string]json.RawMessage
	if err := dec.Decode(&raw); err != nil || raw == nil {
		return momoNotice{}, false
	}
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return momoNotice{}, false
	}
	ok := true
	text := func(name string, required bool) string {
		v, present := raw[name]
		if !present {
			ok = ok && !required
			return ""
		}
		var s string
		if t := bytes.TrimSpace(v); len(t) == 0 || t[0] != '"' || json.Unmarshal(t, &s) != nil {
			ok = false
			return ""
		}
		return s
	}
	number := func(name string) (string, int64) {
		t := strings.TrimSpace(string(raw[name]))
		if !momoIntText.MatchString(t) {
			ok = false
			return "", 0
		}
		n, err := strconv.ParseInt(t, 10, 64)
		if err != nil {
			ok = false
			return "", 0
		}
		return t, n
	}
	var n momoNotice
	n.partnerCode = text("partnerCode", true)
	n.orderID = text("orderId", true)
	requestID := text("requestId", true)
	n.signature = text("signature", true)
	orderInfo, orderType := text("orderInfo", false), text("orderType", false)
	message, extraData := text("message", false), text("extraData", false)
	n.payType = text("payType", false)
	amountText, amount := number("amount")
	transText, transID := number("transId")
	resultText, resultCode := number("resultCode")
	timeText, responseTime := number("responseTime")
	n.amount, n.transID, n.resultCode, n.responseTime = amount, transID, resultCode, responseTime
	n.signed = map[string]string{
		"amount": amountText, "extraData": extraData, "message": message, "orderId": n.orderID,
		"orderInfo": orderInfo, "orderType": orderType, "partnerCode": n.partnerCode, "payType": n.payType,
		"requestId": requestID, "responseTime": timeText, "resultCode": resultText, "transId": transText,
	}
	return n, ok
}

// momoPaidTime reads responseTime (epoch milliseconds). nil outside the years
// 2000 to 2100: a payment is never refused for a time.
func momoPaidTime(ms int64) *time.Time {
	if ms < 946684800000 || ms > 4102444800000 {
		return nil
	}
	t := time.UnixMilli(ms)
	return &t
}

type momoOutcome int

const (
	// momoAccepted is a 204: the notice was recorded, or had been already.
	momoAccepted momoOutcome = iota
	// momoRefused is a 400: nothing was changed.
	momoRefused
	// momoFailed is a 500: the database failed and nothing is half-written.
	momoFailed
)

// ipn answers MoMo's server-to-server notice, which (with VNPay's) is the only
// thing that marks an online order paid. 204 means the notice was accepted or had
// been processed already (both stop MoMo's retries); 400 means it was refused, in
// the project's envelope with no `errors`, so a stranger is not told which check
// failed; 500 means the database failed, so MoMo sends it again. The log line holds
// the outcome, the order's number once known and the request id, never the body,
// the signature, a transaction number or a key.
func (m *MoMo) ipn(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), momoNoticeTimeout)
	defer cancel()
	w.Header().Set("Cache-Control", "no-store")
	outcome, number, cause := m.process(ctx, http.MaxBytesReader(w, r.Body, momoMaxBody))
	rid := httpx.RequestIDFrom(r.Context())
	switch outcome {
	case momoAccepted:
		m.log.Info("momo notice", "outcome", "accepted", "order", number, "request_id", rid)
		w.WriteHeader(http.StatusNoContent)
	case momoRefused:
		m.log.Info("momo notice", "outcome", "refused", "order", number, "request_id", rid)
		httpx.Error(w, http.StatusBadRequest, "bad_request", "The notice was refused.", nil)
	default:
		m.log.Error("momo notice failed", "order", number, "request_id", rid, "error", cause)
		httpx.Error(w, http.StatusInternalServerError, "internal", "Something went wrong. Please try again.", nil)
	}
}

// process runs the checks in order; the first that fails answers and changes
// nothing. Every check before the first read of the database is arithmetic or
// cryptography, so a stranger costs no database work.
func (m *MoMo) process(ctx context.Context, body io.Reader) (momoOutcome, string, error) {
	if !m.Available() {
		return momoRefused, "", nil
	}
	raw, err := io.ReadAll(body)
	if err != nil {
		return momoRefused, "", nil
	}
	n, ok := parseMoMoNotice(raw)
	if !ok {
		return momoRefused, "", nil
	}
	fields := map[string]string{"accessKey": m.cfg.AccessKey.Reveal()}
	for k, v := range n.signed {
		fields[k] = v
	}
	if !momoVerify(m.cfg.SecretKey.Reveal(), fields, n.signature) || !hmac.Equal([]byte(n.partnerCode), []byte(m.cfg.PartnerCode)) {
		return momoRefused, "", nil
	}

	a, err := m.store.find(ctx, n.orderID)
	switch {
	case errors.Is(err, errNoAttempt):
		return momoRefused, "", nil
	case err != nil:
		return momoFailed, "", err
	}
	number := a.Number
	if n.amount != a.Amount {
		return momoRefused, number, nil
	}
	if a.Paid {
		return momoAccepted, number, nil
	}

	f := facts{
		TransactionNo: strconv.FormatInt(n.transID, 10), BankCode: n.payType,
		ResponseCode: strconv.FormatInt(n.resultCode, 10), TransactionStatus: "",
		Amount: n.amount, PaidTime: momoPaidTime(n.responseTime),
	}
	if n.resultCode != 0 {
		if err := m.store.recordFailure(ctx, a, f); err != nil {
			return momoFailed, number, err
		}
		return momoAccepted, number, nil
	}
	// A success pays the order once; a twin that won the race left nothing to do.
	if _, err := m.store.recordSuccess(ctx, a, f); err != nil {
		return momoFailed, number, err
	}
	return momoAccepted, number, nil
}
