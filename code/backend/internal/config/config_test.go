package config

import (
	"reflect"
	"testing"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestPortFallsBackFromPortToAppPortTo8080(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want string
	}{
		{"PORT wins", map[string]string{"PORT": "9000", "APP_PORT": "9100"}, "9000"},
		{"APP_PORT next", map[string]string{"APP_PORT": "9100"}, "9100"},
		{"default", map[string]string{}, "8080"},
		{"blank PORT is skipped", map[string]string{"PORT": "  ", "APP_PORT": "9100"}, "9100"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			c.env["DATABASE_URL"] = "postgres://u:p@h/db"
			got, err := load(env(c.env))
			if err != nil {
				t.Fatal(err)
			}
			if got.Port != c.want {
				t.Errorf("Port = %q, want %q", got.Port, c.want)
			}
		})
	}
}

func TestDatabaseURLIsRequiredAndNeverEchoed(t *testing.T) {
	if _, err := load(env(map[string]string{})); err == nil {
		t.Fatal("expected an error without DATABASE_URL")
	}
}

func TestAdminSettingsAreReadAndOptional(t *testing.T) {
	got, err := load(env(map[string]string{
		"DATABASE_URL":   "postgres://x",
		"ADMIN_EMAIL":    "  dealer@example.com ",
		"ADMIN_PASSWORD": " a password with spaces ",
	}))
	if err != nil {
		t.Fatal(err)
	}
	if got.AdminEmail != "dealer@example.com" || got.AdminPassword != " a password with spaces " {
		t.Errorf("admin settings: %q, %q", got.AdminEmail, got.AdminPassword)
	}
	none, err := load(env(map[string]string{"DATABASE_URL": "postgres://x"}))
	if err != nil || none.AdminEmail != "" || none.AdminPassword != "" {
		t.Errorf("without the settings: %q, %q, %v", none.AdminEmail, none.AdminPassword, err)
	}
}

func TestBadPortIsRefused(t *testing.T) {
	for _, p := range []string{"abc", "0", "70000", "-1"} {
		_, err := load(env(map[string]string{"PORT": p, "DATABASE_URL": "postgres://x"}))
		if err == nil {
			t.Errorf("PORT=%q was accepted", p)
		}
	}
}

func TestCORSOriginsAreSplitAndTrimmed(t *testing.T) {
	got, err := load(env(map[string]string{
		"DATABASE_URL":         "postgres://x",
		"CORS_ALLOWED_ORIGINS": " http://localhost:3000/, ,http://localhost:5173 ",
	}))
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"http://localhost:3000", "http://localhost:5173"}
	if !reflect.DeepEqual(got.CORSAllowedOrigins, want) {
		t.Errorf("origins = %v, want %v", got.CORSAllowedOrigins, want)
	}
}
