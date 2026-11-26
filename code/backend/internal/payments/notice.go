package payments

import (
	"context"
	"crypto/hmac"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
)

const (
	// maxQuery is the longest raw query a notice may carry.
	maxQuery = 8 << 10
	// ipnTimeout bounds one notice, database included.
	ipnTimeout = 5 * time.Second
)

// reply is VNPay's answer shape, not the project's error envelope.
type reply struct {
	Code    string `json:"RspCode"`
	Message string `json:"Message"`
}

var (
	replyOK        = reply{"00", "Confirm Success"}
	replyNoOrder   = reply{"01", "Order not found"}
	replyAlready   = reply{"02", "Order already confirmed"}
	replyAmount    = reply{"04", "Invalid amount"}
	replySignature = reply{"97", "Invalid signature"}
	replyUnknown   = reply{"99", "Unknown error"}
)

// attemptRow is the attempt a notice names and its order's state.
type attemptRow struct {
	ID, OrderID, Number string
	Amount              int64
	Paid                bool
}

// facts are what a notice keeps: no card number is received, none is stored.
type facts struct {
	TransactionNo, BankCode, ResponseCode, TransactionStatus string
	Amount                                                  int64
	PaidTime                                                *time.Time
}

var errNoAttempt = errors.New("no attempt has that reference")

// store is everything the notice reads and writes.
type store interface {
	// find returns the attempt of provider vnpay with that reference: any earlier
	// reference of an order qualifies. errNoAttempt when there is none.
	find(ctx context.Context, reference string) (attemptRow, error)
	// recordFailure stores a failure notice; an identical one is not stored again.
	recordFailure(ctx context.Context, a attemptRow, f facts) error
	// recordSuccess marks the order paid and stores the notice in one transaction.
	// false means the order was already paid, or a twin notice won the race: nothing
	// was written.
	recordSuccess(ctx context.Context, a attemptRow, f facts) (bool, error)
}

// ipn answers VNPay's server-to-server notice, and is the only thing that marks an
// order paid. Every answer is HTTP 200 with a JSON body of exactly RspCode and
// Message: VNPay reads the code, not the status. The signature is the validation,
// so every vnp_ parameter VNPay sends is accepted. The log line holds the code, the
// order's number once it is known and the request id, and never the query, the
// signature, a transaction number or the secret.
func (v *VNPay) ipn(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), ipnTimeout)
	defer cancel()
	rep, number, cause := v.process(ctx, r.URL.RawQuery)
	if cause != nil {
		v.log.Error("vnpay notice failed", "code", rep.Code, "order", number, "request_id", httpx.RequestIDFrom(r.Context()), "error", cause)
	} else {
		v.log.Info("vnpay notice", "code", rep.Code, "order", number, "request_id", httpx.RequestIDFrom(r.Context()))
	}
	body, _ := json.Marshal(rep)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

// process runs the checks in order; the first that fails answers and changes
// nothing. Every check before the first write is arithmetic or cryptography, so a
// stranger costs no database work. cause is set only for a 99.
func (v *VNPay) process(ctx context.Context, raw string) (rep reply, number string, cause error) {
	// 0 and 1: the keys, the size, the shape, the signature, the terminal.
	if !v.Available() || len(raw) > maxQuery {
		return replySignature, "", nil
	}
	q, err := url.ParseQuery(raw)
	if err != nil {
		return replySignature, "", nil
	}
	for name, values := range q {
		// "The first value" and "the last value" are two different truths and the
		// signature covers neither.
		if strings.HasPrefix(name, "vnp_") && len(values) != 1 {
			return replySignature, "", nil
		}
	}
	if !verify(v.cfg.HashSecret.Reveal(), q) || !hmac.Equal([]byte(q.Get("vnp_TmnCode")), []byte(v.cfg.TmnCode)) {
		return replySignature, "", nil
	}

	// 2: the reference is an attempt's.
	a, err := v.store.find(ctx, q.Get("vnp_TxnRef"))
	switch {
	case errors.Is(err, errNoAttempt):
		return replyNoOrder, "", nil
	case err != nil:
		return replyUnknown, "", err
	}
	number = a.Number

	// 3: the amount is the attempt's, in VNPay's unit.
	amount, ok := amountOf(q.Get("vnp_Amount"))
	if !ok || amount != a.Amount {
		return replyAmount, number, nil
	}

	// 4: an order already paid is not paid twice.
	if a.Paid {
		return replyAlready, number, nil
	}

	f := facts{
		TransactionNo: q.Get("vnp_TransactionNo"), BankCode: q.Get("vnp_BankCode"),
		ResponseCode: q.Get("vnp_ResponseCode"), TransactionStatus: q.Get("vnp_TransactionStatus"),
		Amount: amount, PaidTime: payDate(q.Get("vnp_PayDate")),
	}

	// 5: a failure or a cancelled payment is received and changes nothing about the order.
	if f.ResponseCode != "00" || f.TransactionStatus != "00" {
		if err := v.store.recordFailure(ctx, a, f); err != nil {
			return replyUnknown, number, err
		}
		return replyOK, number, nil
	}

	// 6: a success pays the order once.
	paid, err := v.store.recordSuccess(ctx, a, f)
	if err != nil {
		return replyUnknown, number, err
	}
	if !paid {
		return replyAlready, number, nil
	}
	return replyOK, number, nil
}

// amountOf reads vnp_Amount: 1 to 18 ASCII digits, divisible by 100; the result is
// in đồng.
func amountOf(s string) (int64, bool) {
	if len(s) < 1 || len(s) > 18 {
		return 0, false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return 0, false
		}
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil || n%100 != 0 {
		return 0, false
	}
	return n / 100, true
}

// payDate reads vnp_PayDate as Vietnam time. nil when it is absent or does not
// parse: a payment is never refused for a date.
func payDate(s string) *time.Time {
	t, err := time.ParseInLocation(dateLayout, s, ict)
	if err != nil {
		return nil
	}
	return &t
}
