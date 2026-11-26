package payments

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/orders"
	"github.com/jackc/pgx/v5"
)

// pgStore is the notice's store over PostgreSQL.
type pgStore struct{ db DB }

const findSQL = `SELECT a.id::text, a.order_id, a.amount, o.number, o.paid
FROM payment_attempts a JOIN orders o ON o.id = a.order_id
WHERE a.reference = $1 AND a.provider = 'vnpay'`

func (s pgStore) find(ctx context.Context, reference string) (attemptRow, error) {
	var a attemptRow
	err := s.db.QueryRow(ctx, findSQL, reference).Scan(&a.ID, &a.OrderID, &a.Amount, &a.Number, &a.Paid)
	if errors.Is(err, pgx.ErrNoRows) {
		return attemptRow{}, errNoAttempt
	}
	if err != nil {
		return attemptRow{}, fmt.Errorf("find attempt: %w", err)
	}
	return a, nil
}

const insertFailureSQL = `INSERT INTO payment_notices
    (attempt_id, transaction_no, bank_code, amount, response_code, transaction_status, is_success, paid_time)
VALUES ($1::uuid, $2, $3, $4, $5, $6, false, $7)
ON CONFLICT (attempt_id, transaction_no, response_code, transaction_status) WHERE NOT is_success DO NOTHING`

// recordFailure inserts the notice. An identical failure is one row: the return
// address VNPay gives the customer carries the same signed fields, so it could be
// replayed here without bound. The answer is 00 either way.
func (s pgStore) recordFailure(ctx context.Context, a attemptRow, f facts) error {
	if _, err := s.db.Exec(ctx, insertFailureSQL, a.ID, f.TransactionNo, f.BankCode, f.Amount, f.ResponseCode, f.TransactionStatus, f.PaidTime); err != nil {
		return fmt.Errorf("store failure notice: %w", err)
	}
	return nil
}

const insertSuccessSQL = `INSERT INTO payment_notices
    (attempt_id, transaction_no, bank_code, amount, response_code, transaction_status, is_success, paid_time)
VALUES ($1::uuid, $2, $3, $4, $5, $6, true, $7)
ON CONFLICT (attempt_id) WHERE is_success DO NOTHING`

func (s pgStore) recordSuccess(ctx context.Context, a attemptRow, f facts) (bool, error) {
	var paid bool
	err := retryDeadlock(func() error {
		var err error
		paid, err = s.success(ctx, a, f)
		return err
	})
	return paid, err
}

// success is one transaction. The order's row is taken first, by the conditional
// UPDATE: two notices at the same moment both pass the earlier checks, both run
// MarkPaid, and the second waits on the row, finds it paid and changes nothing. A
// result of zero rows from either statement rolls the transaction back.
func (s pgStore) success(ctx context.Context, a attemptRow, f facts) (bool, error) {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, fmt.Errorf("begin notice: %w", err)
	}
	defer func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }()
	if _, err := tx.Exec(ctx, "SET LOCAL lock_timeout = '3s'"); err != nil {
		return false, fmt.Errorf("set lock timeout: %w", err)
	}
	changed, err := orders.MarkPaid(ctx, tx, a.OrderID)
	if err != nil || !changed {
		return false, err
	}
	tag, err := tx.Exec(ctx, insertSuccessSQL, a.ID, f.TransactionNo, f.BankCode, f.Amount, f.ResponseCode, f.TransactionStatus, f.PaidTime)
	if err != nil {
		return false, fmt.Errorf("store success notice: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return false, nil
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit notice: %w", err)
	}
	return true, nil
}

var _ = time.Second
