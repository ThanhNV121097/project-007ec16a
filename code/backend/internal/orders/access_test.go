package orders

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

const (
	spyAccount = "6f1c1e2a-0d4b-4c58-9b53-2f1f0f0a2222"
	spyToken   = "abcdefghijklmnopqrstuvwxyz012345"
)

// accessSpy remembers the statement authorize ran and the parameters it gave, and
// finds no order.
type accessSpy struct {
	sql  string
	args []any
}

func (s *accessSpy) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	s.sql, s.args = sql, args
	return errRow{pgx.ErrNoRows}
}

// A caller who holds only a token runs the statement of the order page unchanged:
// the order's id and the hash of the token, and nothing else.
func TestAGuestWithATokenRunsTheTwoParameterStatement(t *testing.T) {
	spy := &accessSpy{}
	if _, err := authorize(context.Background(), spy, "doi-24818", Access{Token: spyToken}, modeRead); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
	hash, ok := spy.args[1].([]byte)
	if spy.sql != accessSQL || len(spy.args) != 2 || spy.args[0] != "doi-24818" || !ok || len(hash) != 32 {
		t.Errorf("statement %q, parameters %v", spy.sql, spy.args)
	}
}

// A signed-in customer runs the statement with the third way in: NULL for the
// token when none is held, the account id from the session, never from the request.
func TestACustomerRunsTheStatementWithTheAccountAndANullTokenWhenNoneIsHeld(t *testing.T) {
	spy := &accessSpy{}
	if _, err := authorize(context.Background(), spy, "doi-24818", Access{AccountID: spyAccount}, modeRead); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
	if spy.sql != accessAccountSQL || len(spy.args) != 3 || spy.args[0] != "doi-24818" || spy.args[1] != nil || spy.args[2] != spyAccount {
		t.Errorf("no token: statement %q, parameters %v", spy.sql, spy.args)
	}

	spy = &accessSpy{}
	if _, err := authorize(context.Background(), spy, "doi-24818", Access{Token: spyToken, AccountID: spyAccount}, modePay); !errors.Is(err, ErrNotFound) {
		t.Fatalf("got %v", err)
	}
	hash, ok := spy.args[1].([]byte)
	if spy.sql != accessAccountSQL || len(spy.args) != 3 || !ok || len(hash) != 32 || spy.args[2] != spyAccount {
		t.Errorf("a token and an account: statement %q, parameters %v", spy.sql, spy.args)
	}
}

// Neither statement selects a private column, and each names order_tokens once, in
// the one text that holds the token's rule.
func TestNeitherAccessStatementSelectsAPrivateColumn(t *testing.T) {
	for _, sql := range []string{accessSQL, accessAccountSQL} {
		if strings.Count(sql, "order_tokens") != 1 {
			t.Errorf("order_tokens is named %d times in %q", strings.Count(sql, "order_tokens"), sql)
		}
		for _, name := range []string{"phone", "email", "address", "note", "is_sample", "source", "idempotency_key"} {
			if strings.Contains(sql, "o."+name) {
				t.Errorf("an access statement selects %s", name)
			}
		}
	}
}
