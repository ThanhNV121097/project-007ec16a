package admin

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/stock"
	"github.com/jackc/pgx/v5"
)

const recNotAccepted = "This field is not accepted."

// sizeAnswer is PATCH /v1/products/{id}/sizes' success: the size, its new count
// and the shoe's lists redrawn from the database.
type sizeAnswer struct {
	Size  string `json:"size"`
	Pairs int    `json:"pairs"`
	stock.Lists
}

// setSize answers PATCH /v1/products/{id}/sizes: one size's count on its own,
// changed only if it still holds the count the person last saw.
func (h *recHandler) setSize(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if !recValidID(id) {
		recNotFound(w)
		return
	}
	obj, ok := readObject(w, r)
	if !ok {
		return
	}
	undefined := map[string]string{}
	for key := range obj {
		if key != "size" && key != "pairs" && key != "seen_pairs" {
			undefined[faultName(key)] = recNotAccepted
		}
	}
	if len(undefined) > 0 {
		refuse(w, http.StatusBadRequest, "bad_request", unreadable, undefined)
		return
	}
	faults := map[string]string{}
	var typed string
	if raw, sent := obj["size"]; !sent || isNull(raw) || json.Unmarshal(raw, &typed) != nil {
		faults["size"] = "Choose a size."
	}
	pairs, seen := 0, 0
	for _, c := range []struct {
		name string
		into *int
	}{{"pairs", &pairs}, {"seen_pairs", &seen}} {
		raw, sent := obj[c.name]
		n, whole := wholeNumber(raw)
		if !sent || !whole || n < 0 {
			faults[c.name] = recWholePairs
			continue
		}
		if n > recMaxPairs {
			faults[c.name] = recManyPairs
			continue
		}
		*c.into = int(n)
	}
	if len(faults) > 0 {
		invalid(w, faults)
		return
	}
	size, _, _ := stock.ParseSize(typed)

	ctx, cancel := context.WithTimeout(r.Context(), dbTimeout)
	defer cancel()
	var answer sizeAnswer
	err := h.inTx(ctx, func(tx pgx.Tx) error {
		exists, err := recExists(ctx, tx, "products", id)
		if err != nil {
			return err
		}
		if !exists {
			return recMissing()
		}
		switch err := stock.SetOne(ctx, tx, id, size, pairs, seen); err {
		case nil:
		case stock.ErrMoved:
			return &recRefusal{status: http.StatusConflict, code: "conflict", message: recTaken}
		case stock.ErrNoSize:
			return recMissing()
		default:
			return err
		}
		lists, err := stock.ReadLists(ctx, tx, id)
		answer = sizeAnswer{Size: size, Pairs: pairs, Lists: lists}
		return err
	})
	if err != nil {
		h.fail(w, "records.size", err)
		return
	}
	reply(w, http.StatusOK, answer)
}
