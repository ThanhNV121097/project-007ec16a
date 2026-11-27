package data

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
)

// queryTimeout bounds the statements of one request; the request's own context
// is 10 seconds (project contract), and a read that takes half of that is a
// fault, not a slow page.
const queryTimeout = 5 * time.Second

// Register mounts the eighteen read routes on mux: GET /v1/<collection> and
// GET /v1/<collection>/{id} for each of the nine collections. They are public:
// no session, no cookie is read. Routes carry no /api prefix — the edge proxy
// strips it.
//
// Orders are not among them: they hold a customer's details, so internal/orders
// serves them — a record by the order's own token, the list to the shop's team.
func Register(mux *http.ServeMux, db Querier, log *slog.Logger) {
	h := &handler{store: store{db: db}, log: log}
	for _, c := range registry {
		mux.HandleFunc("GET /v1/"+c.Name, h.list(c))
		mux.HandleFunc("GET /v1/"+c.Name+"/{id}", h.record(c))
	}
}

// ListHandler is the list reader of one collection that is not in the registry,
// for a package that serves it under a guard of its own (internal/orders mounts
// the team's list of orders behind auth.Require). It is the reader the registry's
// collections use: the same grammar for filters, ranges, sort, search and paging,
// the same answers and the same errors. A second query language is never written.
func ListHandler(db Querier, log *slog.Logger, c *Collection) http.Handler {
	return ListHandlerServing(db, log, c)
}

// ListHandlerServing is ListHandler for a handler that serves more than the
// Public columns: it selects the columns of the access classes it names (the
// team's orders list names Public and Private). What a request may filter, sort
// or search does not change: only Public columns, so a filter on a Private one
// is a 400 whoever asks. With no class named it serves Public, which is what
// ListHandler is.
func ListHandlerServing(db Querier, log *slog.Logger, c *Collection, served ...Access) http.Handler {
	h := &handler{store: store{db: db}, log: log, served: served}
	return h.list(c)
}

type handler struct {
	store  store
	log    *slog.Logger
	served []Access
}

// listAnswer is the body of a list: the records, how many match in all, and the
// page and limit that were applied.
type listAnswer struct {
	Items json.RawMessage `json:"items"`
	Total int64           `json:"total"`
	Page  int64           `json:"page"`
	Limit int64           `json:"limit"`
}

// list answers GET /v1/<collection>. data.list is the handler that leaves the
// private and bookkeeping columns out of the query: it reads only the Public
// columns of the collection's registry entry (collections.go).
func (h *handler) list(c *Collection) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, faults := parseList(c, r.URL.Query())
		if faults != nil {
			httpx.Error(w, http.StatusBadRequest, "bad_request", badParameters, faults)
			return
		}
		req.served = h.served
		ctx, cancel := context.WithTimeout(r.Context(), queryTimeout)
		defer cancel()
		p, err := h.store.list(ctx, c, req)
		if err != nil {
			h.fail(w, c, err)
			return
		}
		setReadHeaders(w)
		httpx.JSON(w, http.StatusOK, listAnswer{Items: p.items, Total: p.total, Page: req.page, Limit: req.limit})
	}
}

// record answers GET /v1/<collection>/{id}. data.record, like data.list, reads
// only the Public columns.
func (h *handler) record(c *Collection) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if !validID(id) {
			notFound(w)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), queryTimeout)
		defer cancel()
		body, err := h.store.record(ctx, c, id)
		if errors.Is(err, errNotFound) {
			notFound(w)
			return
		}
		if err != nil {
			h.fail(w, c, err)
			return
		}
		setReadHeaders(w)
		httpx.JSON(w, http.StatusOK, body)
	}
}

// fail answers a statement that failed. Whatever the cause — the database
// cannot be reached, a statement timed out, the SQL is wrong — the caller gets
// the generic 500 and nothing of the cause: no record, SQL, host or stack. The
// cause goes to the log beside the collection; never the query string, which
// holds filter values.
func (h *handler) fail(w http.ResponseWriter, c *Collection, err error) {
	h.log.Error("data read failed", "collection", c.Name, "error", err)
	httpx.Error(w, http.StatusInternalServerError, "internal", "Something went wrong. Please try again.", nil)
}

func notFound(w http.ResponseWriter) {
	httpx.Error(w, http.StatusNotFound, "not_found", "Not found.", nil)
}

// setReadHeaders marks an answer as not to be cached (a dealer's edit must be
// on the page at once, and the orders list carries customer names) and not to
// be sniffed.
func setReadHeaders(w http.ResponseWriter) {
	h := w.Header()
	h.Set("Cache-Control", "no-store")
	h.Set("X-Content-Type-Options", "nosniff")
}
