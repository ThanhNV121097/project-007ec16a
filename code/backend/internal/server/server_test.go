package server

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

type fakeRow struct {
	value int
	err   error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	*(dest[0].(*int)) = r.value
	return nil
}

type fakeDB struct{ row fakeRow }

func (f fakeDB) QueryRow(context.Context, string, ...any) pgx.Row { return f.row }

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func get(h http.Handler, method, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(method, path, nil))
	return rec
}

func TestHealthIsOKOnlyWhenTheDatabaseAnswers(t *testing.T) {
	up := get(health(fakeDB{fakeRow{value: 1}}, quiet()), http.MethodGet, "/healthz")
	if up.Code != http.StatusOK || !strings.Contains(up.Body.String(), `"ok"`) {
		t.Errorf("healthy database: %d %s", up.Code, up.Body)
	}
	down := get(health(fakeDB{fakeRow{err: errors.New("connection refused")}}, quiet()), http.MethodGet, "/healthz")
	if down.Code != http.StatusServiceUnavailable {
		t.Errorf("failing database: %d, want 503", down.Code)
	}
	if strings.Contains(down.Body.String(), "connection refused") {
		t.Error("the database error reached the response body")
	}
	wrong := get(health(fakeDB{fakeRow{value: 0}}, quiet()), http.MethodGet, "/healthz")
	if wrong.Code != http.StatusServiceUnavailable {
		t.Errorf("an unexpected SELECT 1 answer: %d, want 503", wrong.Code)
	}
}

func TestUnknownRoutesAnswerTheJSONEnvelope(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", notFound)
	rec := get(mux, http.MethodGet, "/v1/nothing")
	if rec.Code != http.StatusNotFound || !strings.Contains(rec.Body.String(), `"not_found"`) {
		t.Errorf("got %d %s", rec.Code, rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Errorf("content type %q", ct)
	}
}
