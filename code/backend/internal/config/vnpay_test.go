package config

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"testing"
)

const distinct = "s3cr3t-hash-value-7f3a"

func TestVNPaySettingsAreReadTrimmedWithTheSandboxAsDefault(t *testing.T) {
	got, err := load(env(map[string]string{
		"DATABASE_URL":      "postgres://x",
		"VNPAY_TMN_CODE":    "  TMN123 ",
		"VNPAY_HASH_SECRET": " " + distinct + " ",
		"PUBLIC_BASE_URL":   " https://shop.example.com/ ",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if got.VNPayTmnCode != "TMN123" || got.VNPayHashSecret.Reveal() != distinct || !got.VNPayHashSecret.IsSet() {
		t.Errorf("keys: %q, set=%v", got.VNPayTmnCode, got.VNPayHashSecret.IsSet())
	}
	if got.VNPayPaymentURL != DefaultVNPayPage || got.PublicBaseURL != "https://shop.example.com" {
		t.Errorf("page %q, origin %q", got.VNPayPaymentURL, got.PublicBaseURL)
	}
}

// A missing key never stops the server: it starts and online payment is unavailable.
func TestMissingVNPayKeysDoNotStopTheServer(t *testing.T) {
	got, err := load(env(map[string]string{"DATABASE_URL": "postgres://x"}))
	if err != nil {
		t.Fatal(err)
	}
	if got.VNPayTmnCode != "" || got.VNPayHashSecret.IsSet() || got.PublicBaseURL != "" {
		t.Errorf("got %+v", got)
	}
}

func TestMalformedAddressesStopTheServerAndNameTheKeyNotTheValue(t *testing.T) {
	for key, values := range map[string][]string{
		"VNPAY_PAYMENT_URL": {"ftp://pay.example.com/x", "https://", "https://pay.example.com/x?a=1", "https://pay.example.com/x#f", "pay.example.com/x", "https://u:p@pay.example.com/x"},
		"PUBLIC_BASE_URL":   {"shop.example.com", "https://shop.example.com/shop", "https://shop.example.com?x=1", "ftp://shop.example.com", "https://"},
	} {
		for _, v := range values {
			_, err := load(env(map[string]string{"DATABASE_URL": "postgres://x", key: v}))
			if err == nil {
				t.Errorf("%s=%q was accepted", key, v)
				continue
			}
			if !strings.Contains(err.Error(), key) || strings.Contains(err.Error(), v) {
				t.Errorf("%s=%q: error %q must name the key and not the value", key, v, err)
			}
		}
	}
}

func TestAHttpPaymentPageAndALocalOriginAreAccepted(t *testing.T) {
	got, err := load(env(map[string]string{
		"DATABASE_URL": "postgres://x", "VNPAY_PAYMENT_URL": "http://localhost:9000/pay", "PUBLIC_BASE_URL": "http://localhost:5173",
	}))
	if err != nil || got.VNPayPaymentURL != "http://localhost:9000/pay" || got.PublicBaseURL != "http://localhost:5173" {
		t.Fatalf("got %q %q %v", got.VNPayPaymentURL, got.PublicBaseURL, err)
	}
}

// The hash secret prints as [redacted] however the config is printed, logged or encoded.
func TestTheHashSecretIsNeverPrinted(t *testing.T) {
	cfg, err := load(env(map[string]string{"DATABASE_URL": "postgres://x", "VNPAY_HASH_SECRET": distinct}))
	if err != nil {
		t.Fatal(err)
	}
	var logged strings.Builder
	log := slog.New(slog.NewTextHandler(&logged, nil))
	log.Info("config", "secret", cfg.VNPayHashSecret, "all", cfg)
	encoded, _ := json.Marshal(cfg)
	for name, text := range map[string]string{
		"%v": fmt.Sprintf("%v", cfg), "%+v": fmt.Sprintf("%+v", cfg), "%#v": fmt.Sprintf("%#v", cfg),
		"%s": fmt.Sprintf("%s", cfg.VNPayHashSecret), "slog": logged.String(), "json": string(encoded),
	} {
		if strings.Contains(text, distinct) {
			t.Errorf("%s holds the secret: %s", name, text)
		}
	}
	if !strings.Contains(fmt.Sprintf("%+v", cfg.VNPayHashSecret), "[redacted]") {
		t.Error("the secret does not print as [redacted]")
	}
}
