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
	// AdminEmail and AdminPassword are the owner's settings the admin account
	// is made from at start-up (auth.EnsureAdmin). Neither is required: the
	// server starts without them and nobody can sign in as admin. The email is
	// trimmed; the password is exactly as supplied. Neither is ever logged or
	// put in an error.
	AdminEmail    string
	AdminPassword string
	// VNPayTmnCode and VNPayHashSecret are the owner's VNPay keys (sandbox first),
	// read only from the environment. Neither is required: with one missing the
	// server starts and online payment is unavailable. The secret prints as
	// [redacted] however the config is printed.
	VNPayTmnCode    string
	VNPayHashSecret Secret
	// VNPayPaymentURL is VNPay's payment page: the sandbox unless the owner sets the
	// live one. It holds no query and no fragment.
	VNPayPaymentURL string
	// PublicBaseURL is the shop's own origin for VNPay's return address. Empty means
	// the origin is read from the request.
	PublicBaseURL string
	// MoMo and Bank are the owner's settings for the two extra ways to pay. Neither is
	// required and neither can stop the server: a way whose settings are missing or
	// malformed is simply not offered, and Warnings names each key at fault (by name,
	// never by value) for the server to log once.
	MoMo     MoMo
	Bank     Bank
	Warnings []string
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
	page, err := paymentPage(get("VNPAY_PAYMENT_URL"))
	if err != nil {
		return Config{}, err
	}
	origin, err := publicOrigin(get("PUBLIC_BASE_URL"))
	if err != nil {
		return Config{}, err
	}
	momo, momoWarnings := momoFrom(get)
	bank, bankWarnings := bankFrom(get)
	return Config{
		Port:               port,
		DatabaseURL:        dsn,
		CORSAllowedOrigins: list(get("CORS_ALLOWED_ORIGINS")),
		AdminEmail:         strings.TrimSpace(get("ADMIN_EMAIL")),
		AdminPassword:      get("ADMIN_PASSWORD"),
		VNPayTmnCode:       strings.TrimSpace(get("VNPAY_TMN_CODE")),
		VNPayHashSecret:    NewSecret(get("VNPAY_HASH_SECRET")),
		VNPayPaymentURL:    page,
		PublicBaseURL:      origin,
		MoMo:               momo,
		Bank:               bank,
		Warnings:           append(momoWarnings, bankWarnings...),
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
