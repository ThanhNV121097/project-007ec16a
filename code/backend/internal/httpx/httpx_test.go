package httpx

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func quiet() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

func ok() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { JSON(w, http.StatusOK, map[string]string{"a": "b"}) })
}

func TestErrorWritesTheEnvelope(t *testing.T) {
	rec := httptest.NewRecorder()
	Error(rec, http.StatusUnprocessableEntity, "invalid", "Check the phone number", map[string]string{"phone": "Check the phone number"})
	if rec.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d", rec.Code)
	}
	if ct := rec.Header().Get("Content-Type"); ct != "application/json; charset=utf-8" {
		t.Errorf("content type %q", ct)
	}
	var body ErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body.Error != "invalid" || body.Errors["phone"] == "" {
		t.Errorf("unexpected body %+v", body)
	}
}

func TestRecoverAnswers500WithoutLeakingThePanic(t *testing.T) {
	h := Recover(quiet())(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("secret detail") }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d", rec.Code)
	}
	if b, _ := io.ReadAll(rec.Body); len(b) == 0 || strings.Contains(string(b), "secret detail") {
		t.Errorf("body %q leaks or is empty", b)
	}
}

func TestCORSIsOffWithoutOrigins(t *testing.T) {
	h := CORS(nil)(ok())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("CORS headers were set with no origin listed")
	}
}

func TestCORSAllowsOnlyListedOriginsAndAnswersPreflight(t *testing.T) {
	h := CORS([]string{"http://localhost:3000"})(ok())

	pre := httptest.NewRequest(http.MethodOptions, "/v1/products", nil)
	pre.Header.Set("Origin", "http://localhost:3000")
	pre.Header.Set("Access-Control-Request-Method", "POST")
	pre.Header.Set("Access-Control-Request-Headers", "content-type")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, pre)
	if rec.Code != http.StatusNoContent || rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" ||
		rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Errorf("preflight: %d %v", rec.Code, rec.Header())
	}

	other := httptest.NewRequest(http.MethodGet, "/v1/products", nil)
	other.Header.Set("Origin", "http://evil.example")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, other)
	if rec.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("an unlisted origin was allowed")
	}
	if rec.Code != http.StatusOK {
		t.Errorf("an unlisted origin must still reach the handler, got %d", rec.Code)
	}
}
