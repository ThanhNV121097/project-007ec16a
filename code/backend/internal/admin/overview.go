package admin

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
	"github.com/jackc/pgx/v5"
)

// overviewTimeout is the deadline of one overview request: ten times the p95
// bound, and shorter than the browser client's 15 seconds, so the handler
// answers 500 before the client gives up.
const overviewTimeout = 5 * time.Second

// overviewDB is the slice of *pgxpool.Pool the overview needs.
type overviewDB interface {
	BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error)
}

// overviewReader is what the handler asks of the shop: the whole answer, as of
// one instant.
type overviewReader interface {
	read(ctx context.Context, now time.Time) (overviewAnswer, error)
}

type overviewHandler struct {
	reader overviewReader
	log    *slog.Logger
	now    func() time.Time
}

// RegisterOverview mounts GET /v1/admin/overview for the shop's team (admin and
// staff). now is the clock: the real one in the running server, a fixed instant
// in a test; nil means time.Now. No environment variable or request parameter
// moves it. Any other method on the path falls to the server's catch-all.
func RegisterOverview(mux *http.ServeMux, db overviewDB, log *slog.Logger, now func() time.Time) {
	if now == nil {
		now = time.Now
	}
	(&overviewHandler{reader: overviewStore{db: db}, log: log, now: now}).mount(mux)
}

func (h *overviewHandler) mount(mux *http.ServeMux) {
	mux.Handle("GET /v1/admin/overview", auth.Require(auth.Admin, auth.Staff)(http.HandlerFunc(h.get)))
}

// The answer. These structs are the privacy rule: a field that is not here
// cannot be sent, so no phone, email, address or note can reach a body.
type overviewAnswer struct {
	Month         overviewMonth     `json:"month"`
	WaitingOrders int64             `json:"waiting_orders"`
	PairsOnShelf  int64             `json:"pairs_on_shelf"`
	Shoes         int64             `json:"shoes"`
	RunningLow    []overviewLowShoe `json:"running_low"`
	SalesByDay    []overviewDay     `json:"sales_by_day"`
	PairsByLine   []overviewLine    `json:"pairs_by_line"`
	LatestOrders  []overviewOrder   `json:"latest_orders"`
}

type overviewMonth struct {
	Sales  int64 `json:"sales"`
	Orders int64 `json:"orders"`
}

type overviewLowShoe struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Category string   `json:"category"`
	Image    string   `json:"image"`
	Stock    int64    `json:"stock"`
	SizesLow []string `json:"sizes_low"`
}

type overviewDay struct {
	Date  string `json:"date"`
	Sales int64  `json:"sales"`
}

type overviewLine struct {
	Line  string `json:"line"`
	Name  string `json:"name"`
	Pairs int64  `json:"pairs"`
}

type overviewOrder struct {
	ID          string `json:"id"`
	Number      string `json:"number"`
	Customer    string `json:"customer"`
	Product     string `json:"product"`
	ProductName string `json:"product_name"`
	Size        string `json:"size"`
	Status      string `json:"status"`
	PlacedAt    string `json:"placed_at"`
	Total       int64  `json:"total"`
}

// get answers GET /v1/admin/overview. The guard has already refused a visitor
// (401) and a customer (403). The query string and the body are never read, so
// an unknown parameter changes nothing; the clock is read once.
func (h *overviewHandler) get(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), overviewTimeout)
	defer cancel()
	answer, err := h.reader.read(ctx, h.now())
	if err != nil {
		// A browser that went away is not a failure of the shop.
		if errors.Is(err, context.Canceled) && r.Context().Err() != nil {
			return
		}
		auth.LogFailure(h.log, "overview request failed", "overview.read", err)
		refuse(w, http.StatusInternalServerError, "internal", internalMessage, nil)
		return
	}
	reply(w, http.StatusOK, answer)
}
