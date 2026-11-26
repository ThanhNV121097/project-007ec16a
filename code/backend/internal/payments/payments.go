// Package payments is the VNPay provider behind the order's payment seam.
//
// It builds the signed payment address (PaymentURL), answers VNPay's own
// server-to-server notice (GET /v1/payments/vnpay/ipn), which is the ONLY thing
// that marks an order paid, and gives up an unpaid order after 30 minutes
// (RunExpiry). It is the only writer of payment_attempts and payment_notices, and
// it changes an order only through orders.LockOrder, orders.MarkPaid and
// orders.Cancel, so the order's status and a shelf's counts have one writer each.
// orders never imports this package.
//
// The routes carry no /api prefix: the edge proxy strips it.
package payments

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	// maxAttempts bounds the rows one order's token can make by pressing "Pay now".
	maxAttempts = 50
	// MsgTooManyAttempts is the 409 sentence of the 51st attempt.
	MsgTooManyAttempts = "This order has had too many payment attempts. Please place a new order."
)

// ict is Vietnam's fixed offset: VNPay reads its dates in it, and the server's own
// zone and the tzdata of the image are never consulted.
var ict = time.FixedZone("ICT", 7*3600)

// dateLayout is VNPay's yyyyMMddHHmmss.
const dateLayout = "20060102150405"

// DB is the slice of *pgxpool.Pool this package uses.
type DB interface {
	BeginTx(ctx context.Context, opts pgx.TxOptions) (pgx.Tx, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// Register mounts VNPay's notice address. A request with any other method falls
// to the server's catch-all and answers 404 in the project's envelope: it is not
// VNPay calling.
func Register(mux *http.ServeMux, v *VNPay) {
	mux.HandleFunc("GET /v1/payments/vnpay/ipn", v.ipn)
}

// isDeadlock is PostgreSQL's 40P01. None is expected (every transaction here takes
// the order's row first), so the one retry is a guard and not a design.
func isDeadlock(err error) bool {
	var pg *pgconn.PgError
	return errors.As(err, &pg) && pg.Code == "40P01"
}

func retryDeadlock(run func() error) error {
	err := run()
	if isDeadlock(err) {
		err = run()
	}
	return err
}
