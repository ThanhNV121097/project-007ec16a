// Package server builds the HTTP handler: the routes, wrapped in the
// middleware every request passes through.
//
// Routes carry NO /api prefix. The edge proxy strips it (deploy.yml,
// handle_path /api/*), so the browser's /api/v1/products reaches the server as
// /v1/products. A story adds its routes with one Register call below.
package server

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/admin"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/auth"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/catalogue"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/config"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/data"
	"github.com/ThanhNV121097/project-007ec16a/backend/internal/httpx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB is the slice of *pgxpool.Pool the health check needs.
type DB interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// New returns the whole API as one handler.
func New(cfg config.Config, pool *pgxpool.Pool, log *slog.Logger) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health(pool, log))
	// Feature modules mount here, one line each, e.g.
	//   catalogue.Register(mux, pool, log)
	svc := auth.New(pool, log)
	svc.Register(mux)
	admin.RegisterStaff(mux, pool, log)
	data.Register(mux, pool, log)
	catalogue.Register(mux, pool, log)
	mux.HandleFunc("/", notFound)

	// The chain, outermost first: Recover, AccessLog, CORS, Resolve (who is
	// signed in), the route recorder (for the access log), the mux.
	var h http.Handler = httpx.RecordRoute(mux)
	h = svc.Resolve()(h)
	h = httpx.CORS(cfg.CORSAllowedOrigins)(h)
	h = httpx.AccessLog(log)(h)
	h = httpx.Recover(log)(h)
	return h
}

// health answers 200 only while a SELECT 1 against the database works.
// Migrations have already succeeded by then: the server does not listen
// before they have.
func health(db DB, log *slog.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		var one int
		if err := db.QueryRow(ctx, "SELECT 1").Scan(&one); err != nil || one != 1 {
			log.Warn("health check failed", "error", err)
			httpx.Error(w, http.StatusServiceUnavailable, "unavailable", "The database cannot be reached.", nil)
			return
		}
		httpx.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
	}
}

func notFound(w http.ResponseWriter, _ *http.Request) {
	httpx.Error(w, http.StatusNotFound, "not_found", "Not found.", nil)
}
