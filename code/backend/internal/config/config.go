// Package config reads the process environment once, at start-up. Every key
// the backend reads is listed in code/backend/.env.example.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config is everything the server needs from its environment. A story that
// reads a new key adds a field here and a line to .env.example.
type Config struct {
	// Port to listen on: $PORT, then $APP_PORT, then 8080.
	Port string
	// DatabaseURL is the Postgres DSN the runtime injects. It is never
	// assembled from separate DB_* parts and never logged.
	DatabaseURL string
	// CORSAllowedOrigins lists browser origins allowed to call the API with
	// credentials. Empty (the default) switches CORS off: in deployment the
	// browser reaches the API on its own origin under /api.
	CORSAllowedOrigins []string
}

// Load reads the real environment.
func Load() (Config, error) { return load(os.Getenv) }

func load(get func(string) string) (Config, error) {
	port := first(get("PORT"), get("APP_PORT"), "8080")
	if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
		return Config{}, fmt.Errorf("PORT %q is not a port number", port)
	}
	dsn := strings.TrimSpace(get("DATABASE_URL"))
	if dsn == "" {
		return Config{}, errors.New("DATABASE_URL is not set")
	}
	return Config{
		Port:               port,
		DatabaseURL:        dsn,
		CORSAllowedOrigins: list(get("CORS_ALLOWED_ORIGINS")),
	}, nil
}

// first returns the first value that is not blank.
func first(values ...string) string {
	for _, v := range values {
		if v = strings.TrimSpace(v); v != "" {
			return v
		}
	}
	return ""
}

// list splits a comma-separated setting, dropping blanks and trailing slashes.
func list(s string) []string {
	var out []string
	for _, part := range strings.Split(s, ",") {
		if part = strings.TrimRight(strings.TrimSpace(part), "/"); part != "" {
			out = append(out, part)
		}
	}
	return out
}
