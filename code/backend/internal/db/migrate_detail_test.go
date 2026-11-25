package db

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
)

// A duplicate key's id is in PostgreSQL's DETAIL line, which PgError.Error()
// leaves out: a seed that meets a record already there must name it.
func TestWithDetailNamesTheClashingID(t *testing.T) {
	pg := &pgconn.PgError{Message: "duplicate key value violates unique constraint \"lines_pkey\"", Detail: "Key (id)=(samba) already exists."}
	got := withDetail(fmt.Errorf("insert: %w", pg))
	if !strings.Contains(got.Error(), "Key (id)=(samba) already exists.") {
		t.Errorf("the id is missing from %q", got)
	}
	var still *pgconn.PgError
	if !errors.As(got, &still) {
		t.Error("the PostgreSQL error can no longer be unwrapped")
	}
}

func TestWithDetailLeavesOtherErrorsAlone(t *testing.T) {
	plain := errors.New("boom")
	if got := withDetail(plain); got != plain {
		t.Errorf("got %v", got)
	}
	bare := &pgconn.PgError{Message: "x"}
	if got := withDetail(bare); got != error(bare) {
		t.Errorf("got %v", got)
	}
}
