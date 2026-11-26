package payments

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/config"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/orders"
	"github.com/jackc/pgx/v5"
)

// VNPayConfig is what the provider reads of the settings.
type VNPayConfig struct {
	TmnCode       string
	HashSecret    config.Secret
	PaymentURL    string
	PublicBaseURL string
}

// VNPay is the provider for the way to pay "vnpay". It satisfies orders.Provider.
type VNPay struct {
	cfg   VNPayConfig
	db    DB
	store store
	log   *slog.Logger
	now   func() time.Time
}

// NewVNPay makes the provider. It is registered even with no keys: a shop with no
// keys answers 503, never "not available yet". A missing key is logged once, by
// name only, and never stops the server. now is time.Now when nil.
func NewVNPay(cfg VNPayConfig, db DB, log *slog.Logger, now func() time.Time) *VNPay {
	if now == nil {
		now = time.Now
	}
	cfg.TmnCode = strings.TrimSpace(cfg.TmnCode)
	if cfg.PaymentURL == "" {
		cfg.PaymentURL = config.DefaultVNPayPage
	}
	v := &VNPay{cfg: cfg, db: db, store: pgStore{db: db, provider: "vnpay"}, log: log, now: now}
	if cfg.TmnCode == "" {
		log.Warn("VNPAY_TMN_CODE is not set; online payment is unavailable")
	}
	if !cfg.HashSecret.IsSet() {
		log.Warn("VNPAY_HASH_SECRET is not set; online payment is unavailable")
	}
	return v
}

// Available is true only when both keys are set.
func (v *VNPay) Available() bool {
	return v.cfg.TmnCode != "" && v.cfg.HashSecret.IsSet()
}

const insertAttemptSQL = `INSERT INTO payment_attempts (order_id, provider, reference, amount, expires_at)
VALUES ($1, 'vnpay', $2, $3, $4)`

// PaymentURL gives the address the customer pays at, and stores the attempt it is
// for. Only o.ID is trusted: the order is read again under its row lock, so an
// amount, a number or a time passed in from anywhere is never used. A refusal the
// order's state calls for comes back as a *orders.Refusal.
func (v *VNPay) PaymentURL(ctx context.Context, o orders.Payable, c orders.Client) (string, error) {
	if !v.Available() {
		return "", orders.ErrPaymentUnavailable
	}
	var address string
	err := retryDeadlock(func() error {
		var err error
		address, err = v.attempt(ctx, o.ID, c)
		return err
	})
	return address, err
}

// attempt is one transaction: lock the order's row, judge its state again (a
// notice or a cancel may have landed since the route looked), count its attempts,
// build and sign the address, store the attempt. A failure anywhere rolls back, so
// there is never an attempt without an address.
func (v *VNPay) attempt(ctx context.Context, id string, c orders.Client) (string, error) {
	tx, err := v.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return "", fmt.Errorf("begin attempt: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if _, err := tx.Exec(ctx, "SET LOCAL lock_timeout = '2s'"); err != nil {
		return "", fmt.Errorf("set lock timeout: %w", err)
	}
	locked, err := orders.LockOrder(ctx, tx, id)
	if err != nil {
		return "", err
	}
	now := v.now()
	if r := orders.CheckPayable(locked.State(), now); r != nil {
		return "", r
	}
	var made int
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM payment_attempts WHERE order_id = $1", locked.ID).Scan(&made); err != nil {
		return "", fmt.Errorf("count attempts: %w", err)
	}
	if made >= maxAttempts {
		return "", &orders.Refusal{Message: MsgTooManyAttempts}
	}
	reference := fmt.Sprintf("%s-%d", locked.Number, made+1)
	address, err := v.address(locked, reference, now, c)
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, insertAttemptSQL, locked.ID, reference, locked.Total, locked.PlacedAt.Add(orders.PayWindow)); err != nil {
		return "", fmt.Errorf("store attempt: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit attempt: %w", err)
	}
	return address, nil
}
