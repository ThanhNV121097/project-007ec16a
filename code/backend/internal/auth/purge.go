package auth

import (
	"context"
	"log/slog"
	"time"

	"github.com/ThanhNV121097/project-007ec16a/backend/internal/ratelimit"
)

const purgeEvery = time.Hour

// RunPurge deletes the sessions that have expired and the rate-limit counters
// nobody needs, once at start and then every hour, until ctx is done. A failure
// is logged and the loop goes on. Run it in its own goroutine.
func RunPurge(ctx context.Context, db Querier, log *slog.Logger) {
	purgeOnce(ctx, db, log)
	ticker := time.NewTicker(purgeEvery)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			purgeOnce(ctx, db, log)
		}
	}
}

func purgeOnce(ctx context.Context, db Querier, log *slog.Logger) {
	step, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if _, err := db.Exec(step, `DELETE FROM sessions WHERE expires_at < now()`); err != nil {
		logFailure(log, "purging expired sessions failed", "purge", err)
	}
	if _, err := ratelimit.Purge(step, db); err != nil {
		logFailure(log, "purging rate limits failed", "purge", err)
	}
}
