package payments

import (
	"errors"
	"math"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/orders"
)

// originPattern is the only shape a return address's origin may have: a scheme, a
// host of letters, digits, dots and dashes, and an optional port. A request that
// forges a host that fails it makes the call an error, never an address that sends
// a customer somewhere unintended.
var originPattern = regexp.MustCompile(`^https?://[A-Za-z0-9.-]+(:[0-9]{1,5})?$`)

// returnOrigin is the shop's own origin: PUBLIC_BASE_URL when it is set, else the
// one the request came in on.
func returnOrigin(configured, fromRequest string) (string, error) {
	origin := configured
	if origin == "" {
		origin = fromRequest
	}
	origin = strings.TrimRight(origin, "/")
	if !originPattern.MatchString(origin) {
		return "", errors.New("the shop's own address cannot be used as the return address")
	}
	return origin, nil
}

// asciiOnly keeps the characters VNPay's order info may hold here.
func asciiOnly(s string) string {
	return strings.Map(func(r rune) rune {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || r == '-' {
			return r
		}
		return -1
	}, s)
}

// address builds the signed payment address for one attempt of the locked order.
// The amount, the number and the deadline are the locked row's: nothing the
// browser said is used. No call is made to VNPay. The secret is never in the
// address; the terminal code and the signature are.
func (v *VNPay) address(o orders.Locked, reference string, now time.Time, c orders.Client) (string, error) {
	origin, err := returnOrigin(v.cfg.PublicBaseURL, c.Origin)
	if err != nil {
		return "", err
	}
	if o.Total < 0 || o.Total > math.MaxInt64/100 {
		return "", errors.New("the order total does not fit VNPay's amount")
	}
	created := now.In(ict).Format(dateLayout)
	// The page stops taking payment at the order's own deadline, not 30 minutes from
	// now: an order placed 10 minutes ago expires in 20.
	expires := o.PlacedAt.Add(orders.PayWindow).In(ict).Format(dateLayout)
	if expires <= created {
		return "", &orders.Refusal{Message: orders.MsgRunOut}
	}
	ip := c.IP
	if net.ParseIP(ip) == nil {
		ip = "127.0.0.1"
	}

	params := url.Values{}
	params.Set("vnp_Version", "2.1.0")
	params.Set("vnp_Command", "pay")
	params.Set("vnp_TmnCode", v.cfg.TmnCode)
	params.Set("vnp_Amount", strconv.FormatInt(o.Total*100, 10))
	params.Set("vnp_CurrCode", "VND")
	params.Set("vnp_TxnRef", reference)
	params.Set("vnp_OrderInfo", "Thanh toan don hang "+asciiOnly(o.Number))
	params.Set("vnp_OrderType", "other")
	params.Set("vnp_Locale", "en")
	params.Set("vnp_ReturnUrl", origin+"/payment/return")
	params.Set("vnp_IpAddr", ip)
	params.Set("vnp_CreateDate", created)
	params.Set("vnp_ExpireDate", expires)

	signed := canonicalString(params)
	return v.cfg.PaymentURL + "?" + signed + "&vnp_SecureHash=" + sign(v.cfg.HashSecret.Reveal(), signed), nil
}
