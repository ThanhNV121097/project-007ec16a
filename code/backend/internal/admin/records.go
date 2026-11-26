package admin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/data"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// RegisterRecords mounts what the shop's team writes: POST, PUT and DELETE on
// products, lines, categories and reviews, PATCH of one size's count, and the
// pictures (POST /v1/uploads, and GET /v1/uploads/{name} for everyone). Every
// write is behind auth.Require(admin, staff): 401 with no session, 403 for a
// customer. A collection with no write route here (delivery, payments, shop,
// sizes, faqs) answers a write with the server's JSON 404. Routes carry no /api
// prefix; the edge proxy strips it.
func RegisterRecords(mux *http.ServeMux, db Pool, log *slog.Logger) {
	h := &recHandler{db: db, log: log}
	guard := auth.Require(auth.Admin, auth.Staff)
	for _, s := range recSpecs {
		mux.Handle("POST /v1/"+s.name, guard(h.create(s)))
		mux.Handle("PUT /v1/"+s.name+"/{id}", guard(h.update(s)))
		mux.Handle("DELETE /v1/"+s.name+"/{id}", guard(h.remove(s)))
	}
	mux.Handle("PATCH /v1/products/{id}/sizes", guard(http.HandlerFunc(h.setSize)))
	mux.Handle("POST /v1/uploads", guard(http.HandlerFunc(h.upload)))
	mux.HandleFunc("GET /v1/uploads/{name}", h.picture)
}

type recHandler struct {
	db  Pool
	log *slog.Logger
}

// recBefore is a record as it stood when the write locked it.
type recBefore struct {
	found   bool
	status  string // a shoe's or a review's
	product string // a review's shoe
	rating  int    // a review's stars, 0 for none
}

// lockRecord takes a record FOR UPDATE and reads what a write needs to know
// about how it stood.
func lockRecord(ctx context.Context, tx pgx.Tx, s *recSpec, id string) (recBefore, error) {
	var b recBefore
	var row pgx.Row
	switch s.table {
	case "products":
		row = tx.QueryRow(ctx, `SELECT status FROM products WHERE id = $1 FOR UPDATE`, id)
		if err := row.Scan(&b.status); err != nil {
			return recBefore{}, noRow(err)
		}
	case "reviews":
		row = tx.QueryRow(ctx, `SELECT status, product_id, coalesce(rating, 0)::integer FROM reviews WHERE id = $1 FOR UPDATE`, id)
		if err := row.Scan(&b.status, &b.product, &b.rating); err != nil {
			return recBefore{}, noRow(err)
		}
	default:
		var one int
		if err := tx.QueryRow(ctx, "SELECT 1 FROM "+s.table+" WHERE id = $1 FOR UPDATE", id).Scan(&one); err != nil {
			return recBefore{}, noRow(err)
		}
	}
	b.found = true
	return b, nil
}

// noRow turns "no such record" into the zero answer (found false) and keeps every
// other error.
func noRow(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	return err
}

func isFeatured(in recInput) bool {
	v, ok := in.value("featured")
	on, _ := v.arg.(bool)
	return ok && on
}

// clearFeatured turns "Opens the home page" off on every shoe but this one: one
// shoe at a time has it.
func clearFeatured(ctx context.Context, tx pgx.Tx, id string) error {
	_, err := tx.Exec(ctx, `UPDATE products SET featured = false WHERE featured AND id <> $1`, id)
	return err
}

// create answers POST /v1/<collection>: 201, Location, the record in the
// dealer's view.
func (h *recHandler) create(s *recSpec) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		obj, ok := readObject(w, r)
		if !ok {
			return
		}
		in := s.read(obj, true)
		ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
		defer cancel()
		var id string
		var saved json.RawMessage
		err := h.inTx(ctx, func(tx pgx.Tx) error {
			if err := checkRefs(ctx, tx, in.values, in.faults); err != nil {
				return err
			}
			if len(in.faults) > 0 {
				return recInvalid(in.faults)
			}
			name := ""
			if v, ok := in.value("name"); ok {
				name, _ = v.arg.(string)
			}
			var err error
			if id, err = newID(ctx, tx, s, name); err != nil {
				return err
			}
			if isFeatured(in) {
				if err := clearFeatured(ctx, tx, id); err != nil {
					return err
				}
			}
			if err := insertRecord(ctx, tx, s, id, in.values); err != nil {
				return err
			}
			if s.shoes && in.stock.touched() {
				if refusal, err := applyStock(ctx, tx, id, in.stock); err != nil || refusal != nil {
					if err == nil {
						err = refusal
					}
					return err
				}
			}
			if s.table == "reviews" {
				shoe, _ := in.valueString("product")
				if err := recomputeRatings(ctx, tx, shoe); err != nil {
					return err
				}
			}
			saved, err = data.Record(ctx, tx, s.name, id)
			return err
		})
		if err != nil {
			h.fail(w, "records.create."+s.name, err)
			return
		}
		w.Header().Set("Location", "/v1/"+s.name+"/"+id)
		reply(w, http.StatusCreated, saved)
	}
}

func (in recInput) valueString(name string) (string, bool) {
	v, ok := in.value(name)
	if !ok {
		return "", false
	}
	s, isText := v.arg.(string)
	return s, ok && isText
}

// insertRecord inserts the record under its id with the fields that were sent;
// the table's defaults make the rest (a shoe is a draft, a review is shown).
func insertRecord(ctx context.Context, tx pgx.Tx, s *recSpec, id string, values []recValue) error {
	cols, marks, args := []string{"id"}, []string{"$1"}, []any{id}
	for _, v := range values {
		args = append(args, v.arg)
		cols = append(cols, v.field.column)
		marks = append(marks, v.field.place(len(args)))
	}
	_, err := tx.Exec(ctx, "INSERT INTO "+s.table+" ("+strings.Join(cols, ", ")+") VALUES ("+strings.Join(marks, ", ")+")", args...)
	return err
}

// update answers PUT /v1/<collection>/{id}: only the fields sent change.
func (h *recHandler) update(s *recSpec) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !recValidID(id) {
			recNotFound(w)
			return
		}
		obj, ok := readObject(w, r)
		if !ok {
			return
		}
		in := s.read(obj, false)
		ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
		defer cancel()
		var saved json.RawMessage
		err := h.inTx(ctx, func(tx pgx.Tx) error {
			before, err := lockRecord(ctx, tx, s, id)
			if err != nil {
				return err
			}
			if !before.found {
				return recMissing()
			}
			if err := checkRefs(ctx, tx, in.values, in.faults); err != nil {
				return err
			}
			if len(in.faults) > 0 {
				return recInvalid(in.faults)
			}
			if isFeatured(in) {
				if err := clearFeatured(ctx, tx, id); err != nil {
					return err
				}
			}
			if err := updateRecord(ctx, tx, s, id, in.values); err != nil {
				return err
			}
			if s.shoes && in.stock.touched() {
				if refusal, err := applyStock(ctx, tx, id, in.stock); err != nil || refusal != nil {
					if err == nil {
						err = refusal
					}
					return err
				}
			}
			if err := rateAfter(ctx, tx, s, before, in); err != nil {
				return err
			}
			saved, err = data.Record(ctx, tx, s.name, id)
			return err
		})
		if err != nil {
			h.fail(w, "records.update."+s.name, err)
			return
		}
		reply(w, http.StatusOK, saved)
	}
}

// updateRecord sets the fields that were sent and no others.
func updateRecord(ctx context.Context, tx pgx.Tx, s *recSpec, id string, values []recValue) error {
	if len(values) == 0 {
		return nil
	}
	sets, args := make([]string, 0, len(values)), make([]any, 0, len(values)+1)
	for _, v := range values {
		args = append(args, v.arg)
		sets = append(sets, v.field.column+" = "+v.field.place(len(args)))
	}
	args = append(args, id)
	_, err := tx.Exec(ctx, "UPDATE "+s.table+" SET "+strings.Join(sets, ", ")+fmt.Sprintf(" WHERE id = $%d", len(args)), args...)
	return err
}

// rateAfter recomputes the figures an event of this save changes: a review's
// creation, stars, status or shoe changing moves the ratings of the shoes it
// touches and the shop's; a shoe's own status changing moves the shop's. A save
// of a shoe's other fields never recomputes its own rating.
func rateAfter(ctx context.Context, tx pgx.Tx, s *recSpec, before recBefore, in recInput) error {
	switch s.table {
	case "products":
		if status, sent := in.valueString("status"); sent && status != before.status {
			return recomputeShop(ctx, tx)
		}
	case "reviews":
		shoe, shoeSent := in.valueString("product")
		status, statusSent := in.valueString("status")
		stars, starsSent := 0, false
		if v, ok := in.value("rating"); ok {
			starsSent = true
			if n, isNum := v.arg.(int64); isNum {
				stars = int(n)
			}
		}
		moved := shoeSent && shoe != before.product
		if moved || (statusSent && status != before.status) || (starsSent && stars != before.rating) {
			return recomputeRatings(ctx, tx, before.product, shoe)
		}
	}
	return nil
}

// remove answers DELETE /v1/<collection>/{id}: 204, or 409 with the sentence a
// person needs when something still depends on the record.
func (h *recHandler) remove(s *recSpec) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !recValidID(id) {
			recNotFound(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
		defer cancel()
		err := h.inTx(ctx, func(tx pgx.Tx) error {
			before, err := lockRecord(ctx, tx, s, id)
			if err != nil {
				return err
			}
			if !before.found {
				return recMissing()
			}
			if refusal, err := dependants(ctx, tx, s, id); err != nil || refusal != nil {
				if err == nil {
					err = refusal
				}
				return err
			}
			if _, err := tx.Exec(ctx, "DELETE FROM "+s.table+" WHERE id = $1", id); err != nil {
				return err
			}
			switch s.table {
			case "products":
				return recomputeShop(ctx, tx)
			case "reviews":
				return recomputeRatings(ctx, tx, before.product)
			}
			return nil
		})
		var pg *pgconn.PgError
		if errors.As(err, &pg) && pg.Code == "23503" {
			// An order or a shoe appeared between the check and the delete.
			err = recStillUsed(s, "")
		}
		if err != nil {
			h.fail(w, "records.delete."+s.name, err)
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		w.WriteHeader(http.StatusNoContent)
	}
}

func recStillUsed(s *recSpec, count string) *recRefusal {
	message := "Orders refer to this shoe. Set it to draft instead."
	if s.table != "products" {
		if count == "" {
			count = "Some"
		}
		message = count + " shoes use this. Move them first."
	}
	return &recRefusal{status: http.StatusConflict, code: "conflict", message: message}
}

// dependants refuses the delete of a shoe an order refers to, and of a line or a
// category some shoe uses.
func dependants(ctx context.Context, tx pgx.Tx, s *recSpec, id string) (*recRefusal, error) {
	switch s.table {
	case "products":
		var ordered bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM order_items WHERE product_id = $1)`, id).Scan(&ordered); err != nil {
			return nil, err
		}
		if ordered {
			return recStillUsed(s, ""), nil
		}
	case "lines", "categories":
		column := map[string]string{"lines": "line_id", "categories": "category_id"}[s.table]
		var n int
		if err := tx.QueryRow(ctx, "SELECT count(*)::integer FROM products WHERE "+column+" = $1", id).Scan(&n); err != nil {
			return nil, err
		}
		if n > 0 {
			return recStillUsed(s, fmt.Sprint(n)), nil
		}
	}
	return nil, nil
}
