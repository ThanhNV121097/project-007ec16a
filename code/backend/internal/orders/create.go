package orders

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
)

// answer is the body of a successful placing.
type answer struct {
	Order      json.RawMessage `json:"order"`
	Token      string          `json:"token"`
	PaymentURL string          `json:"payment_url"`
}

// create answers POST /v1/orders. Order of checks: the body (415, 413, 400), the
// header (400), every field at once (422), then the transaction (409, 201, 200).
func (s *service) create(w http.ResponseWriter, r *http.Request) {
	req, ok := decodeBody(w, r)
	if !ok {
		return
	}
	if bad := req.clean(); bad != "" {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "The order holds text the shop cannot store.", fieldErrors{bad: "Use plain text."})
		return
	}
	key := r.Header.Get(keyHeader)
	if !keyPattern.MatchString(key) {
		httpx.Error(w, http.StatusBadRequest, "bad_request", "The order needs an Idempotency-Key.", fieldErrors{keyHeader: keySentence})
		return
	}

	faults := req.validate()
	if err := s.checkSizes(r.Context(), req, faults); err != nil {
		s.internal(w, r, "read the sizes of the named shoes", err)
		return
	}
	if len(faults) > 0 {
		httpx.Error(w, http.StatusUnprocessableEntity, "invalid", messageCheck, faults)
		return
	}

	res, err := s.place(r.Context(), key, req, callerAccount(r))
	if err != nil {
		s.refuse(w, r, err)
		return
	}

	status := http.StatusCreated
	if res.replayed {
		status = http.StatusOK
		w.Header().Set("Idempotent-Replayed", "true")
	} else {
		w.Header().Set("Location", "/v1/orders/"+res.payable.ID)
	}
	noStore(w)
	httpx.JSON(w, status, answer{Order: res.view, Token: res.token, PaymentURL: s.paymentURL(r, res)})
}

// refuse maps what place gave back to the contract's answers. Anything it does not
// know is a defect of this service, not a service that is briefly out: 500, with
// a generic body and the cause in the log (never the key, token, phone or address).
func (s *service) refuse(w http.ResponseWriter, r *http.Request, err error) {
	var invalid *invalidError
	var short *stockError
	switch {
	case errors.As(err, &invalid):
		httpx.Error(w, http.StatusUnprocessableEntity, "invalid", messageCheck, invalid.fields)
	case errors.As(err, &short):
		httpx.Error(w, http.StatusConflict, "stock_conflict", "Some pairs could not be set aside.", short.fields)
	case errors.Is(err, errKeyConflict):
		httpx.Error(w, http.StatusConflict, "conflict", "That order key was already used for a different order.", nil)
	default:
		s.internal(w, r, "place order", err)
	}
}

// checkSizes adds a fault for every pair that names a size its shoe does not
// list. It is the one read a request with a field fault may make, and it speaks
// only for a shoe that exists and is live: a missing or draft shoe is a 409 (no
// longer in the shop) inside the transaction, whatever size the line names.
func (s *service) checkSizes(ctx context.Context, req orderRequest, faults fieldErrors) error {
	if len(req.Items) == 0 || len(req.Items) > maxPairs {
		return nil
	}
	var ids []string
	seen := map[string]bool{}
	for i, it := range req.Items {
		if _, bad := faults["items."+strconv.Itoa(i)]; bad || seen[it.Product] {
			continue
		}
		seen[it.Product] = true
		ids = append(ids, it.Product)
	}
	if len(ids) == 0 {
		return nil
	}
	ctx, cancel := context.WithTimeout(ctx, txTimeout)
	defer cancel()
	rows, err := s.db.Query(ctx, `SELECT p.id, p.name, coalesce(array_agg(s.size) FILTER (WHERE s.size IS NOT NULL), '{}')
		FROM products p LEFT JOIN product_sizes s ON s.product_id = p.id
		WHERE p.id = ANY($1) AND p.status = 'live' GROUP BY p.id, p.name`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	type listing struct {
		name  string
		sizes map[string]bool
	}
	listed := map[string]listing{}
	for rows.Next() {
		var id, name string
		var sizes []string
		if err := rows.Scan(&id, &name, &sizes); err != nil {
			return err
		}
		l := listing{name: name, sizes: map[string]bool{}}
		for _, sz := range sizes {
			l.sizes[sz] = true
		}
		listed[id] = l
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for i, it := range req.Items {
		name := "items." + strconv.Itoa(i)
		if _, bad := faults[name]; bad {
			continue
		}
		if l, ok := listed[it.Product]; ok && !l.sizes[it.Size] {
			faults[name] = fmt.Sprintf("%s · EU %s — not a size this shoe comes in.", l.name, it.Size)
		}
	}
	return nil
}

// paymentURL asks the order's provider, outside the transaction, for the address
// to pay at. No provider, an order that is paid or cancelled, an error or a
// provider slower than 3 s all leave it empty: the order stays saved and unpaid.
func (s *service) paymentURL(r *http.Request, p placed) string {
	prov := s.providers[p.payment]
	if prov == nil || p.paid || p.cancelled {
		return ""
	}
	ctx, cancel := context.WithTimeout(r.Context(), providerTimeout)
	defer cancel()
	url, err := prov.PaymentURL(ctx, p.payable, clientOf(r))
	if err != nil {
		s.log.Error("no payment address was given", "order", p.payable.ID, "error", err)
		return ""
	}
	return url
}

// clientOf is where the customer is, as the edge proxy forwards it.
func clientOf(r *http.Request) Client {
	ip := ""
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		ip = strings.TrimSpace(parts[len(parts)-1])
	}
	if ip == "" {
		if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			ip = host
		} else {
			ip = r.RemoteAddr
		}
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if p := r.Header.Get("X-Forwarded-Proto"); p != "" {
		scheme = strings.TrimSpace(strings.Split(p, ",")[0])
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	return Client{IP: ip, Origin: scheme + "://" + host}
}
