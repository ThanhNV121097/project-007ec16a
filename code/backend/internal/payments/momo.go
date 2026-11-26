package payments

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/config"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/orders"
	"github.com/jackc/pgx/v5"
)

// momoReferenceMax is the longest orderId MoMo accepts.
const momoReferenceMax = 50

// MoMoConfig is what the provider reads of the settings.
type MoMoConfig struct {
	PartnerCode string
	AccessKey   config.Secret
	SecretKey   config.Secret
	// APIURL is MoMo's base address, with no trailing slash.
	APIURL        string
	PublicBaseURL string
}

// MoMo is the provider for the way to pay "momo". It satisfies orders.Provider.
type MoMo struct {
	cfg    MoMoConfig
	db     DB
	store  store
	log    *slog.Logger
	now    func() time.Time
	client *http.Client
}

// NewMoMo makes the provider. It is registered even with no keys, so a MoMo order
// of a shop whose keys were removed answers 503 on "Pay now", never "not available
// yet". Redirects are never followed. now is time.Now when nil.
func NewMoMo(cfg MoMoConfig, db DB, log *slog.Logger, now func() time.Time) *MoMo {
	if now == nil {
		now = time.Now
	}
	cfg.PartnerCode = strings.TrimSpace(cfg.PartnerCode)
	cfg.APIURL = strings.TrimRight(strings.TrimSpace(cfg.APIURL), "/")
	return &MoMo{
		cfg: cfg, db: db, store: pgStore{db: db, provider: "momo"}, log: log, now: now,
		client: &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
	}
}

// Available is true only when the partner code, the access key and the secret key
// are all set and MoMo's address is known.
func (m *MoMo) Available() bool {
	return m.cfg.PartnerCode != "" && m.cfg.AccessKey.IsSet() && m.cfg.SecretKey.IsSet() && m.cfg.APIURL != ""
}

const insertMoMoAttemptSQL = `INSERT INTO payment_attempts (order_id, provider, reference, amount, expires_at)
VALUES ($1, 'momo', $2, $3, $4)`

// PaymentURL gives the address the customer pays at on MoMo's page. Only o.ID is
// trusted: the order is read again under its row lock, so an amount, a number or a
// time passed in from anywhere is never used. The attempt is reserved and
// COMMITTED first, in its own transaction; MoMo is asked afterwards with no
// transaction and no lock held, so a slow MoMo pins neither a row nor a connection,
// and an address MoMo gave always names a reference the shop knows. A call that
// then fails leaves an attempt no address ever referred to: it holds no money. A
// refusal the order's state calls for comes back as a *orders.Refusal.
func (m *MoMo) PaymentURL(ctx context.Context, o orders.Payable, c orders.Client) (string, error) {
	if !m.Available() {
		return "", orders.ErrPaymentUnavailable
	}
	var req momoCreate
	var number string
	err := retryDeadlock(func() error {
		var err error
		req, number, err = m.reserve(ctx, o.ID, c)
		return err
	})
	if err != nil {
		return "", err
	}
	return m.ask(ctx, number, req)
}

// reserve is one transaction: lock the order's row, judge its state again (a notice
// or a cancel may have landed since the route looked), count its attempts, build
// and sign the request, store the attempt, commit.
func (m *MoMo) reserve(ctx context.Context, id string, c orders.Client) (momoCreate, string, error) {
	tx, err := m.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return momoCreate{}, "", fmt.Errorf("begin attempt: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if _, err := tx.Exec(ctx, "SET LOCAL lock_timeout = '2s'"); err != nil {
		return momoCreate{}, "", fmt.Errorf("set lock timeout: %w", err)
	}
	locked, err := orders.LockOrder(ctx, tx, id)
	if err != nil {
		return momoCreate{}, "", err
	}
	if r := orders.CheckPayable(locked.State(), m.now()); r != nil {
		return momoCreate{}, "", r
	}
	var made int
	if err := tx.QueryRow(ctx, "SELECT count(*) FROM payment_attempts WHERE order_id = $1", locked.ID).Scan(&made); err != nil {
		return momoCreate{}, "", fmt.Errorf("count attempts: %w", err)
	}
	if made >= maxAttempts {
		return momoCreate{}, "", &orders.Refusal{Message: MsgTooManyAttempts}
	}
	reference := fmt.Sprintf("%s-%d", locked.Number, made+1)
	if len(reference) > momoReferenceMax {
		return momoCreate{}, "", errors.New("the reference is longer than MoMo accepts")
	}
	req, err := m.build(locked, reference, c)
	if err != nil {
		return momoCreate{}, "", err
	}
	if _, err := tx.Exec(ctx, insertMoMoAttemptSQL, locked.ID, reference, locked.Total, locked.PlacedAt.Add(orders.PayWindow)); err != nil {
		return momoCreate{}, "", fmt.Errorf("store attempt: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return momoCreate{}, "", fmt.Errorf("commit attempt: %w", err)
	}
	return req, locked.Number, nil
}
