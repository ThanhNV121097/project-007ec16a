// Package catalogue serves the shop's read side that is not a plain list of
// records: the colours and sizes its filter rail offers.
//
// The list endpoints themselves — search, "any of" filters, ranges, sort and the
// EU size filter — are the data package's: they need its registry of columns
// and its one list statement, and a second handler for GET /v1/products would
// be a duplicate route.
package catalogue

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
)

// queryTimeout bounds the statement of one request.
const queryTimeout = 5 * time.Second

// Register mounts GET /v1/product-filters on mux. The route carries no /api
// prefix: the edge proxy strips it. It is public and reads no session.
func Register(mux *http.ServeMux, db Querier, log *slog.Logger) {
	h := &handler{store: store{db: db}, log: log}
	mux.HandleFunc("GET /v1/product-filters", h.filters)
}

type handler struct {
	store store
	log   *slog.Logger
}

// filters answers GET /v1/product-filters: every colour and EU size a live shoe
// has, whatever the visitor has filtered. Any query parameter is ignored.
func (h *handler) filters(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), queryTimeout)
	defer cancel()
	f, err := h.store.filters(ctx)
	if err != nil {
		// Whatever the cause, the caller gets the generic 500 and nothing of
		// it; the cause goes to the log, never a query string.
		h.log.Error("product filters failed", "error", err)
		httpx.Error(w, http.StatusInternalServerError, "internal", "Something went wrong. Please try again.", nil)
		return
	}
	// A dealer's edit must show on the next read.
	h2 := w.Header()
	h2.Set("Cache-Control", "no-store")
	h2.Set("X-Content-Type-Options", "nosniff")
	httpx.JSON(w, http.StatusOK, f)
}
