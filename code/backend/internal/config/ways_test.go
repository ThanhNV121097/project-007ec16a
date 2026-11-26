package config

import (
	"fmt"
	"strings"
	"testing"
)

func envOf(m map[string]string) func(string) string {
	base := map[string]string{"DATABASE_URL": "postgres://x"}
	for k, v := range m {
		base[k] = v
	}
	return func(k string) string { return base[k] }
}

var goodBank = map[string]string{
	"VIETQR_BANK_BIN": "970436", "VIETQR_BANK_NAME": "Vietcombank",
	"VIETQR_ACCOUNT_NUMBER": "0071001234567", "VIETQR_ACCOUNT_NAME": "NGUYEN VAN AN",
}

var goodMoMo = map[string]string{
	"MOMO_PARTNER_CODE": "MOMOTEST", "MOMO_ACCESS_KEY": "access-key-distinct", "MOMO_SECRET_KEY": "secret-key-distinct",
}

func with(base map[string]string, changes map[string]string) map[string]string {
	out := map[string]string{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range changes {
		out[k] = v
	}
	return out
}

// SC-36: none of the eight settings starts the server with nothing listed.
func TestNoneOfTheEightSettingsStillStartsAndSaysNothing(t *testing.T) {
	cfg, err := load(envOf(nil))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Bank.Set() || cfg.MoMo.Set() || len(cfg.Warnings) != 0 {
		t.Errorf("bank %v, momo %v, warnings %v", cfg.Bank.Set(), cfg.MoMo.Set(), cfg.Warnings)
	}
}

func TestTheBankAccountIsSetOnlyWhenAllFourAreValid(t *testing.T) {
	cfg, err := load(envOf(goodBank))
	if err != nil || !cfg.Bank.Set() || cfg.Bank.BIN != "970436" || cfg.Bank.AccountName != "NGUYEN VAN AN" {
		t.Fatalf("%v %+v", err, cfg.Bank)
	}
	for name, change := range map[string]map[string]string{
		"a blank bank name":       {"VIETQR_BANK_NAME": "  "},
		"a BIN of five digits":     {"VIETQR_BANK_BIN": "97043"},
		"a BIN with a letter":      {"VIETQR_BANK_BIN": "97043x"},
		"an account with a space":  {"VIETQR_ACCOUNT_NUMBER": "0071 0012"},
		"an account of 20 digits":  {"VIETQR_ACCOUNT_NUMBER": strings.Repeat("1", 20)},
		"a holder with a newline":  {"VIETQR_ACCOUNT_NAME": "AN\nVAN"},
		"a holder of 101 letters":  {"VIETQR_ACCOUNT_NAME": strings.Repeat("A", 101)},
		"a missing holder":         {"VIETQR_ACCOUNT_NAME": ""},
	} {
		cfg, err := load(envOf(with(goodBank, change)))
		if err != nil {
			t.Fatalf("%s stopped the server: %v", name, err)
		}
		if cfg.Bank.Set() || len(cfg.Warnings) == 0 {
			t.Errorf("%s: set %v, warnings %v", name, cfg.Bank.Set(), cfg.Warnings)
		}
		for _, w := range cfg.Warnings {
			if !strings.HasPrefix(w, "VIETQR_") || strings.Contains(w, "970436") || strings.Contains(w, "0071") {
				t.Errorf("%s: the warning %q must name the key and no value", name, w)
			}
		}
	}
}

func TestMoMoIsSetOnlyWithTheThreeKeysAndAValidAddress(t *testing.T) {
	cfg, err := load(envOf(goodMoMo))
	if err != nil || !cfg.MoMo.Set() || cfg.MoMo.APIURL != DefaultMoMoAPI {
		t.Fatalf("%v %+v", err, cfg.MoMo)
	}
	cfg, _ = load(envOf(with(goodMoMo, map[string]string{"MOMO_API_URL": " https://payment.momo.vn/ "})))
	if cfg.MoMo.APIURL != "https://payment.momo.vn" {
		t.Errorf("the address: %q", cfg.MoMo.APIURL)
	}
	for name, change := range map[string]map[string]string{
		"no secret key":          {"MOMO_SECRET_KEY": ""},
		"no access key":          {"MOMO_ACCESS_KEY": " "},
		"no partner code":        {"MOMO_PARTNER_CODE": ""},
		"an address with a query": {"MOMO_API_URL": "https://x.test/?a=1"},
		"an address that is not http": {"MOMO_API_URL": "ftp://x.test"},
		"an address with no host": {"MOMO_API_URL": "https://"},
	} {
		cfg, err := load(envOf(with(goodMoMo, change)))
		if err != nil {
			t.Fatalf("%s stopped the server: %v", name, err)
		}
		if cfg.MoMo.Set() || len(cfg.Warnings) == 0 {
			t.Errorf("%s: set %v, warnings %v", name, cfg.MoMo.Set(), cfg.Warnings)
		}
		for _, w := range cfg.Warnings {
			if strings.Contains(w, "distinct") || strings.Contains(w, "MOMOTEST") {
				t.Errorf("%s: the warning %q holds a value", name, w)
			}
		}
	}
}

// The two MoMo keys print as [redacted] however the config is printed.
func TestTheMoMoKeysAreRedactedWhenTheConfigIsPrinted(t *testing.T) {
	cfg, err := load(envOf(goodMoMo))
	if err != nil {
		t.Fatal(err)
	}
	for _, text := range []string{fmt.Sprintf("%v", cfg), fmt.Sprintf("%+v", cfg), fmt.Sprintf("%#v", cfg)} {
		if strings.Contains(text, "access-key-distinct") || strings.Contains(text, "secret-key-distinct") {
			t.Errorf("a key is in %q", text)
		}
	}
}
