package payments

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/orders"
)

const (
	// momoCreatePath is appended to the base address.
	momoCreatePath = "/v2/gateway/api/create"
	// momoMaxAnswer bounds what is read of MoMo's answer.
	momoMaxAnswer = 64 << 10
	// momoMaxPayURL bounds the address MoMo gives.
	momoMaxPayURL = 2048
)

// momoCreate is MoMo's create-payment request. The access key is in the string that
// is signed and in no member.
type momoCreate struct {
	PartnerCode string `json:"partnerCode"`
	RequestID   string `json:"requestId"`
	OrderID     string `json:"orderId"`
	// Amount is whole VND, a JSON integer: MoMo's unit is not x100.
	Amount      int64  `json:"amount"`
	OrderInfo   string `json:"orderInfo"`
	RedirectURL string `json:"redirectUrl"`
	IPNURL      string `json:"ipnUrl"`
	RequestType string `json:"requestType"`
	ExtraData   string `json:"extraData"`
	Lang        string `json:"lang"`
	Signature   string `json:"signature"`
}

// signedFields are the members MoMo signs, with the access key.
func (r momoCreate) signedFields(accessKey string) map[string]string {
	return map[string]string{
		"accessKey": accessKey, "amount": strconv.FormatInt(r.Amount, 10), "extraData": r.ExtraData,
		"ipnUrl": r.IPNURL, "orderId": r.OrderID, "orderInfo": r.OrderInfo, "partnerCode": r.PartnerCode,
		"redirectUrl": r.RedirectURL, "requestId": r.RequestID, "requestType": r.RequestType,
	}
}

// build makes the signed request for one attempt of the locked order. The amount,
// the number and the reference are the locked row's and the reservation's:
// nothing the browser said is used. The two addresses are the shop's own origin;
// the notice address is the public path, with /api.
func (m *MoMo) build(o orders.Locked, reference string, c orders.Client) (momoCreate, error) {
	origin, err := returnOrigin(m.cfg.PublicBaseURL, c.Origin)
	if err != nil {
		return momoCreate{}, err
	}
	if o.Total < 1 {
		return momoCreate{}, errors.New("the order total is not an amount MoMo takes")
	}
	req := momoCreate{
		PartnerCode: m.cfg.PartnerCode, RequestID: reference, OrderID: reference, Amount: o.Total,
		OrderInfo:   "Thanh toan don hang " + asciiOnly(o.Number),
		RedirectURL: origin + "/payment/return", IPNURL: origin + "/api/v1/payments/momo/ipn",
		RequestType: "captureWallet", ExtraData: "", Lang: "en",
	}
	req.Signature = momoSignature(m.cfg.SecretKey.Reveal(), req.signedFields(m.cfg.AccessKey.Reveal()))
	return req, nil
}

// momoAnswer is the part of MoMo's answer that is read.
type momoAnswer struct {
	ResultCode *int64 `json:"resultCode"`
	Message    string `json:"message"`
	OrderID    string `json:"orderId"`
	PayURL     string `json:"payUrl"`
}

// momoPayURL accepts an absolute http or https address with a host, up to 2,048 characters.
func momoPayURL(s string) (string, bool) {
	if s == "" || len(s) > momoMaxPayURL {
		return "", false
	}
	u, err := url.Parse(s)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", false
	}
	return s, true
}

// ask posts the request to MoMo with no transaction and no lock held, under the
// caller's remaining time, once: a second call would be a second payment address
// for one attempt. Anything but HTTP 200 with resultCode 0 and a usable address is
// an error. The log line holds the order's number, the HTTP status and MoMo's
// resultCode and message, never the request or the response body, the signature,
// the address or a key.
func (m *MoMo) ask(ctx context.Context, number string, req momoCreate) (string, error) {
	body, err := json.Marshal(req)
	if err != nil {
		return "", fmt.Errorf("encode the MoMo request: %w", err)
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, m.cfg.APIURL+momoCreatePath, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("build the MoMo request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json; charset=UTF-8")
	resp, err := m.client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("MoMo could not be asked: %w", err)
	}
	defer resp.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(resp.Body, momoMaxAnswer))
	if err != nil {
		return "", fmt.Errorf("read MoMo's answer: %w", err)
	}
	var a momoAnswer
	parsed := json.Unmarshal(raw, &a) == nil
	code := int64(-1)
	if a.ResultCode != nil {
		code = *a.ResultCode
	}
	address, usable := momoPayURL(a.PayURL)
	if resp.StatusCode != http.StatusOK || !parsed || code != 0 || !usable || (a.OrderID != "" && a.OrderID != req.OrderID) {
		m.log.Error("MoMo did not give a payment address", "order", number, "status", resp.StatusCode, "result_code", code, "message", clip(a.Message, 200))
		return "", errors.New("MoMo did not give a payment address")
	}
	return address, nil
}

// clip keeps at most n bytes of s, so a long answer cannot fill a log line.
func clip(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}
