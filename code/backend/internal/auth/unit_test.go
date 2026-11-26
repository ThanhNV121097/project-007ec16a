package auth

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/argon2"
)

func TestNormalizeEmailAcceptsAStorableAddressAndLowerCasesIt(t *testing.T) {
	for in, want := range map[string]string{
		"dealer@example.com":       "dealer@example.com",
		" Dealer@Example.COM \n":   "dealer@example.com",
		"lan.tran+shop@mail.vn":    "lan.tran+shop@mail.vn",
		"a@b.c":                    "a@b.c",
	} {
		if got, ok := NormalizeEmail(in); !ok || got != want {
			t.Errorf("%q -> %q, %v; want %q", in, got, ok, want)
		}
	}
	for _, bad := range []string{
		"", "  ", "a@", "@b.c", "ab", "a@b", "a@.b.c", "a@b.c.", "a@@b.c", "a b@c.d", "a@b@c.d",
		"dôi@example.com", "dealer@examìple.com", "kK@example.com", "a\x00b@example.com",
		strings.Repeat("a", 190) + "@example.com",
	} {
		if got, ok := NormalizeEmail(bad); ok {
			t.Errorf("%q was accepted as %q", bad, got)
		}
	}
}

func TestDisplayNameIsTheStoredNameOrTheWordsOfTheEmail(t *testing.T) {
	for _, c := range []struct{ name, email, want string }{
		{"Minh Le", "minh@example.com", "Minh Le"},
		{"", "lan.tran@example.com", "Lan Tran"},
		{"  ", "mai_linh-pham@example.com", "Mai Linh Pham"},
		{"", "dealer@example.com", "Dealer"},
		{"", "..._@example.com", "..._@example.com"},
		{"", "đức.anh@example.com", "Đức Anh"},
	} {
		if got := DisplayName(c.name, c.email); got != c.want {
			t.Errorf("DisplayName(%q, %q) = %q, want %q", c.name, c.email, got, c.want)
		}
	}
	if got := (User{Email: "lan.tran@example.com"}).DisplayName(); got != "Lan Tran" {
		t.Errorf("User.DisplayName = %q", got)
	}
}

var newHash = regexp.MustCompile(`^\$argon2id\$v=19\$m=19456,t=2,p=1\$[A-Za-z0-9+/]{22}\$[A-Za-z0-9+/]{43}$`)

// SC-2: an argon2id string, not the password, a new salt for every password.
func TestHashPasswordIsArgon2idWithASaltPerPassword(t *testing.T) {
	a, err := HashPassword("same-password-1")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := HashPassword("same-password-1")
	if !newHash.MatchString(a) || !newHash.MatchString(b) {
		t.Fatalf("not the agreed encoding: %s", a)
	}
	if a == b {
		t.Error("two hashes of one password are equal: the salt is shared")
	}
	if strings.Contains(a, "same-password-1") {
		t.Error("the hash holds the password")
	}
	ctx := context.Background()
	if match, valid, err := check(ctx, a, "same-password-1"); !match || !valid || err != nil {
		t.Errorf("the right password: %v %v %v", match, valid, err)
	}
	if match, valid, _ := check(ctx, a, "same-password-2"); match || !valid {
		t.Errorf("a wrong password: %v %v", match, valid)
	}
}

// Verification reads m, t and p from the stored string, so a hash made with
// other costs (the test accounts' own) still verifies.
func TestVerificationReadsItsParametersFromTheString(t *testing.T) {
	salt := []byte("0123456789abcdef")
	key := argon2.IDKey([]byte("fixture-pw"), salt, 1, 8192, 2, 24)
	enc := fmt.Sprintf("$argon2id$v=19$m=8192,t=1,p=2$%s$%s", b64.EncodeToString(salt), b64.EncodeToString(key))
	if match, valid, _ := check(context.Background(), enc, "fixture-pw"); !match || !valid {
		t.Errorf("a hash with its own parameters: %v %v", match, valid)
	}
	if match, _, _ := check(context.Background(), enc, "other"); match {
		t.Error("a wrong password verified")
	}
}

func TestParsePHCRefusesWhatIsOutOfRangeOrDamaged(t *testing.T) {
	good := fmt.Sprintf("$argon2id$v=19$m=8192,t=1,p=2$%s$%s", b64.EncodeToString(make([]byte, 16)), b64.EncodeToString(make([]byte, 32)))
	if _, ok := parsePHC(good); !ok {
		t.Fatal("the control hash was refused")
	}
	salt, key := b64.EncodeToString(make([]byte, 16)), b64.EncodeToString(make([]byte, 32))
	for name, bad := range map[string]string{
		"empty":           "",
		"wrong algorithm": fmt.Sprintf("$argon2i$v=19$m=8192,t=1,p=2$%s$%s", salt, key),
		"old version":     fmt.Sprintf("$argon2id$v=16$m=8192,t=1,p=2$%s$%s", salt, key),
		"memory too big":  fmt.Sprintf("$argon2id$v=19$m=262145,t=1,p=2$%s$%s", salt, key),
		"time too big":    fmt.Sprintf("$argon2id$v=19$m=8192,t=11,p=2$%s$%s", salt, key),
		"lanes too many":  fmt.Sprintf("$argon2id$v=19$m=8192,t=1,p=17$%s$%s", salt, key),
		"zero lanes":      fmt.Sprintf("$argon2id$v=19$m=8192,t=1,p=0$%s$%s", salt, key),
		"short key":       fmt.Sprintf("$argon2id$v=19$m=8192,t=1,p=2$%s$%s", salt, b64.EncodeToString(make([]byte, 8))),
		"long key":        fmt.Sprintf("$argon2id$v=19$m=8192,t=1,p=2$%s$%s", salt, b64.EncodeToString(make([]byte, 65))),
		"memory too low":  fmt.Sprintf("$argon2id$v=19$m=8,t=1,p=2$%s$%s", salt, key),
		"not base64":      "$argon2id$v=19$m=8192,t=1,p=2$!!!$???",
		"missing part":    "$argon2id$v=19$m=8192,t=1,p=2$" + salt,
	} {
		if _, ok := parsePHC(bad); ok {
			t.Errorf("%s was accepted: %s", name, bad)
		}
	}
}

func TestTokensAre256BitsAndOnlyTheirHashIsKept(t *testing.T) {
	token, hash, err := newToken()
	if err != nil || len(token) != 43 || len(hash) != 32 {
		t.Fatalf("%q %d %v", token, len(hash), err)
	}
	back, ok := tokenHash(token)
	if !ok || !bytes.Equal(back, hash) {
		t.Error("the cookie value does not hash to the stored hash")
	}
	other, _, _ := newToken()
	if other == token {
		t.Error("two tokens are equal")
	}
	for _, bad := range []string{"", token[:42], token + "A", strings.Repeat("*", 43)} {
		if _, ok := tokenHash(bad); ok {
			t.Errorf("%q was taken for a token", bad)
		}
	}
}

// ---- SC-4, SC-5: a problem with the settings is logged and skipped ----

type neverOpened struct{ t *testing.T }

func (n neverOpened) fail() { n.t.Helper(); n.t.Error("the database was used for settings that cannot make an account") }
func (n neverOpened) Query(context.Context, string, ...any) (pgx.Rows, error) {
	n.fail()
	return nil, errors.New("unused")
}
func (n neverOpened) QueryRow(context.Context, string, ...any) pgx.Row { n.fail(); return nil }
func (n neverOpened) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	n.fail()
	return pgconn.CommandTag{}, errors.New("unused")
}
func (n neverOpened) Begin(context.Context) (pgx.Tx, error) {
	n.fail()
	return nil, errors.New("unused")
}

func TestEnsureAdminSkipsSettingsThatCannotMakeAnAccountAndLogsNoValue(t *testing.T) {
	for name, c := range map[string]struct{ email, password, line string }{
		"email unset":          {"", "Dealer-Pass1", "not set"},
		"password unset":       {"dealer@example.com", "", "not set"},
		"nine characters":      {"dealer@example.com", "Short-Pw9", "too short"},
		"nine multibyte":       {"dealer@example.com", "ééééééééé", "too short"},
		"a hundred and one":    {"dealer@example.com", strings.Repeat("p", 101), "too long"},
		"an address that fails": {"not-an-address", "Dealer-Pass1", "not an address"},
	} {
		var logs bytes.Buffer
		log := slog.New(slog.NewTextHandler(&logs, nil))
		if err := EnsureAdmin(context.Background(), neverOpened{t}, c.email, c.password, log); err != nil {
			t.Errorf("%s: %v", name, err)
		}
		if !strings.Contains(logs.String(), c.line) {
			t.Errorf("%s: the log lacks %q: %s", name, c.line, logs.String())
		}
		for _, secret := range []string{c.password, c.email} {
			if secret != "" && strings.Contains(logs.String(), secret) {
				t.Errorf("%s: the log holds %q", name, secret)
			}
		}
	}
}
