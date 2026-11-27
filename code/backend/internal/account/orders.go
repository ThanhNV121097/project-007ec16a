package account

import (
	"errors"
	"math"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/orders"
)

const (
	defaultLimit = 10
	maxLimit     = 100
	lookupLock   = 15 * time.Minute
)

var wholeNumber = regexp.MustCompile(`^[0-9]+$`)

// paging reads ?page= and ?limit=: page is a whole number from 1 (default 1) and
// limit a whole number from 1 (default 10, served as 100 when above). Anything
// else is a fault; any other parameter is ignored.
func paging(q url.Values) (page, limit int, faults map[string]string) {
	page, limit, faults = 1, defaultLimit, map[string]string{}
	if q.Has("page") {
		n, err := strconv.ParseInt(q.Get("page"), 10, 32)
		if !wholeNumber.MatchString(q.Get("page")) || err != nil || n < 1 {
			faults["page"] = "Use a whole number from 1."
		} else {
			page = int(n)
		}
	}
	if q.Has("limit") {
		v := q.Get("limit")
		n, err := strconv.ParseInt(v, 10, 64)
		switch {
		case !wholeNumber.MatchString(v):
			faults["limit"] = "Use a whole number from 1 to 100."
		case err != nil: // more digits than an integer holds: above 100
			limit = maxLimit
		case n < 1:
			faults["limit"] = "Use a whole number from 1 to 100."
		default:
			limit = int(min(n, maxLimit))
		}
	}
	return page, limit, faults
}

// listOrders is GET /v1/account/orders: the session's own orders, newest first.
func (h *handler) listOrders(w http.ResponseWriter, r *http.Request) {
	u, ok := caller(w, r)
	if !ok {
		return
	}
	page, limit, faults := paging(r.URL.Query())
	if len(faults) > 0 {
		refuse(w, http.StatusBadRequest, "bad_request", "Check the request.", faults)
		return
	}
	ctx, cancel := withTimeout(r)
	defer cancel()
	items, total, err := orders.ListFor(ctx, h.db, u.ID, page, limit)
	if err != nil {
		h.internal(w, r, err)
		return
	}
	reply(w, http.StatusOK, struct {
		Items []orders.AccountOrder `json:"items"`
		Total int                   `json:"total"`
		Page  int                   `json:"page"`
		Limit int                   `json:"limit"`
	}{items, total, page, limit})
}

// oneOrder is GET /v1/account/orders/{id}: the public view of one of the
// session's own orders. Another customer's order, a guest order and an id that
// does not exist all run the same read and answer the same bytes.
func (h *handler) oneOrder(w http.ResponseWriter, r *http.Request) {
	u, ok := caller(w, r)
	if !ok {
		return
	}
	ctx, cancel := withTimeout(r)
	defer cancel()
	body, err := orders.Read(ctx, h.db, r.PathValue("id"), orders.Access{AccountID: u.ID})
	if errors.Is(err, orders.ErrNotFound) {
		orders.WriteNotFound(w)
		return
	}
	if err != nil {
		h.internal(w, r, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httpx.JSON(w, http.StatusOK, body)
}

// addOrder is POST /v1/account/orders: a guest order, by its number and the phone
// it was placed with, joins the session's account. Both keys must be present
// strings (an empty string is present and is a miss). The match, the counters and
// the link are orders.AddToAccount's; a wrong phone, an unknown number, an empty
// phone and another account's order are one 404 with one body.
func (h *handler) addOrder(w http.ResponseWriter, r *http.Request) {
	u, ok := caller(w, r)
	if !ok {
		return
	}
	obj, ok := readObject(w, r)
	if !ok {
		return
	}
	faults, bad := obj.undefined("number", "phone")
	var pair orders.Pair
	for key, dst := range map[string]*string{"number": &pair.Number, "phone": &pair.Phone} {
		value, k := obj.text(key)
		switch k {
		case isText:
			*dst = value
		case absent:
			faults[key], bad = "This field is needed.", true
		default:
			faults[key], bad = wrongTypeWords, true
		}
	}
	if bad {
		badRequest(w, faults)
		return
	}
	ctx, cancel := withTimeout(r)
	defer cancel()
	row, err := orders.AddToAccount(ctx, h.db, httpx.ClientAddr(r), pair, u.ID)
	var locked *orders.LockedError
	switch {
	case errors.As(err, &locked):
		secs := retryAfter(w, locked.Wait, lookupLock)
		mins := max(int(math.Ceil(float64(secs)/60)), 1)
		unit := " minutes."
		if mins == 1 {
			unit = " minute."
		}
		refuse(w, http.StatusTooManyRequests, "rate_limited", "Too many tries. Please wait "+strconv.Itoa(mins)+unit, nil)
	case errors.Is(err, orders.ErrNoMatch):
		refuse(w, http.StatusNotFound, "not_found", "No order matches that number and phone.", nil)
	case err != nil:
		h.internal(w, r, err)
	default:
		reply(w, http.StatusOK, row)
	}
}
