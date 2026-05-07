package config_test

import (
	"os"
	"strings"
	"testing"

	"github.com/zamibd/MPanel/config"
)

func TestGetDBDSN_Defaults(t *testing.T) {
	// Clear all POSTGRES_* env vars so defaults kick in.
	vars := []string{
		"POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER",
		"POSTGRES_PASSWORD", "POSTGRES_DB", "POSTGRES_SSLMODE",
	}
	for _, v := range vars {
		os.Unsetenv(v)
	}

	dsn := config.GetDBDSN()

	checks := map[string]string{
		"host":    "postgres",
		"port":    "5432",
		"user":    "mpanel",
		"dbname":  "mpanel",
		"sslmode": "disable",
	}
	for key, want := range checks {
		if !strings.Contains(dsn, key+"="+want) {
			t.Errorf("GetDBDSN() missing %s=%s in DSN: %q", key, want, dsn)
		}
	}
}

func TestGetDBDSN_EnvOverrides(t *testing.T) {
	os.Setenv("POSTGRES_HOST", "db.example.com")
	os.Setenv("POSTGRES_PORT", "5433")
	os.Setenv("POSTGRES_USER", "myuser")
	os.Setenv("POSTGRES_PASSWORD", "secret123")
	os.Setenv("POSTGRES_DB", "mydb")
	os.Setenv("POSTGRES_SSLMODE", "require")
	defer func() {
		for _, v := range []string{
			"POSTGRES_HOST", "POSTGRES_PORT", "POSTGRES_USER",
			"POSTGRES_PASSWORD", "POSTGRES_DB", "POSTGRES_SSLMODE",
		} {
			os.Unsetenv(v)
		}
	}()

	dsn := config.GetDBDSN()

	checks := map[string]string{
		"host":     "db.example.com",
		"port":     "5433",
		"user":     "myuser",
		"password": "secret123",
		"dbname":   "mydb",
		"sslmode":  "require",
	}
	for key, want := range checks {
		if !strings.Contains(dsn, key+"="+want) {
			t.Errorf("GetDBDSN() missing %s=%s in DSN: %q", key, want, dsn)
		}
	}
}

func TestGetDBPath_EqualsDSN(t *testing.T) {
	os.Setenv("POSTGRES_HOST", "testhost")
	defer os.Unsetenv("POSTGRES_HOST")

	if config.GetDBPath() != config.GetDBDSN() {
		t.Errorf("GetDBPath() should be an alias for GetDBDSN()")
	}
}

func TestGetVersion_NotEmpty(t *testing.T) {
	v := config.GetVersion()
	if v == "" {
		t.Error("GetVersion() returned empty string")
	}
}

func TestGetName_NotEmpty(t *testing.T) {
	n := config.GetName()
	if n == "" {
		t.Error("GetName() returned empty string")
	}
}

func TestIsDebug_Default(t *testing.T) {
	os.Unsetenv("SUI_DEBUG")
	if config.IsDebug() {
		t.Error("IsDebug() should be false when SUI_DEBUG is unset")
	}
}

func TestIsDebug_True(t *testing.T) {
	os.Setenv("SUI_DEBUG", "true")
	defer os.Unsetenv("SUI_DEBUG")
	if !config.IsDebug() {
		t.Error("IsDebug() should be true when SUI_DEBUG=true")
	}
}
