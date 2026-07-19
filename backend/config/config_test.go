package config

import (
	"os"
	"testing"
)

func TestGetEnvCSV(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES_TEST", "nginx, 10.0.0.0/8")
	got := getEnvCSV("TRUSTED_PROXIES_TEST", []string{"fallback"})
	if len(got) != 2 || got[0] != "nginx" || got[1] != "10.0.0.0/8" {
		t.Fatalf("unexpected CSV parse: %#v", got)
	}

	_ = os.Unsetenv("TRUSTED_PROXIES_TEST")
	got = getEnvCSV("TRUSTED_PROXIES_TEST", []string{"fallback"})
	if len(got) != 1 || got[0] != "fallback" {
		t.Fatalf("expected fallback, got %#v", got)
	}
}

func TestGetEnvAsBool(t *testing.T) {
	t.Setenv("RUN_INPROCESS_CRONS_TEST", "false")
	if getEnvAsBool("RUN_INPROCESS_CRONS_TEST", true) {
		t.Fatal("expected false")
	}
	t.Setenv("RUN_INPROCESS_CRONS_TEST", "true")
	if !getEnvAsBool("RUN_INPROCESS_CRONS_TEST", false) {
		t.Fatal("expected true")
	}
}

func TestLoadConfigDevelopmentDefaults(t *testing.T) {
	t.Setenv("APP_ENV", "development")
	t.Setenv("JWT_ACCESS_SECRET", "")
	t.Setenv("JWT_REFRESH_SECRET", "")
	t.Setenv("RUN_INPROCESS_CRONS", "true")

	cfg := LoadConfig()
	if cfg.JWT.AccessSecret == "" || cfg.JWT.RefreshSecret == "" {
		t.Fatal("expected development JWT defaults")
	}
	if !cfg.Server.RunInProcessCrons {
		t.Fatal("expected RunInProcessCrons true")
	}
}
