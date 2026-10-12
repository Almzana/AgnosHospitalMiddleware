package config

import (
	"testing"
	"time"
)

func setEnv(t *testing.T, kv map[string]string) {
	t.Helper()
	for _, k := range []string{"PORT", "DATABASE_URL", "JWT_SECRET", "TOKEN_TTL", "HIS_TIMEOUT"} {
		t.Setenv(k, "")
	}
	for k, v := range kv {
		t.Setenv(k, v)
	}
}

func TestLoad_Defaults(t *testing.T) {
	setEnv(t, map[string]string{"DATABASE_URL": "postgres://x", "JWT_SECRET": "0123456789abcdef"})

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "8080" || cfg.TokenTTL != 24*time.Hour || cfg.HISTimeout != 5*time.Second {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
}

func TestLoad_Overrides(t *testing.T) {
	setEnv(t, map[string]string{
		"DATABASE_URL": "postgres://x", "JWT_SECRET": "0123456789abcdef",
		"PORT": "9000", "TOKEN_TTL": "30m", "HIS_TIMEOUT": "2s",
	})

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Port != "9000" || cfg.TokenTTL != 30*time.Minute || cfg.HISTimeout != 2*time.Second {
		t.Errorf("overrides not applied: %+v", cfg)
	}
}

func TestLoad_Invalid(t *testing.T) {
	tests := map[string]map[string]string{
		"missing database url": {"JWT_SECRET": "0123456789abcdef"},
		"missing jwt secret":   {"DATABASE_URL": "postgres://x"},
		"jwt secret too short": {"DATABASE_URL": "postgres://x", "JWT_SECRET": "short"},
		"bad token ttl":        {"DATABASE_URL": "postgres://x", "JWT_SECRET": "0123456789abcdef", "TOKEN_TTL": "soon"},
		"bad his timeout":      {"DATABASE_URL": "postgres://x", "JWT_SECRET": "0123456789abcdef", "HIS_TIMEOUT": "fast"},
	}
	for name, env := range tests {
		t.Run(name, func(t *testing.T) {
			setEnv(t, env)
			if _, err := Load(); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}
