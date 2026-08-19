package config

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadReadsJSONConfig(t *testing.T) {
	clearConfigEnv(t)
	path := writeConfigFile(t, `{
		"server_address": "localhost:9090",
		"base_url": "http://short",
		"file_storage_path": "/tmp/urls.json",
		"database_dsn": "postgres://user:pass@localhost/db",
		"audit_file": "/tmp/audit.jsonl",
		"audit_url": "http://audit.local/events",
		"enable_https": true
	}`)
	flagSet := resetFlags(t, "-c", path)

	cfg, err := Load(flagSet)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.ServerAddr != "localhost:9090" {
		t.Fatalf("ServerAddr = %q, want %q", cfg.ServerAddr, "localhost:9090")
	}
	if cfg.BaseURL != "http://short" {
		t.Fatalf("BaseURL = %q, want %q", cfg.BaseURL, "http://short")
	}
	if cfg.FileStoragePath != "/tmp/urls.json" {
		t.Fatalf("FileStoragePath = %q, want %q", cfg.FileStoragePath, "/tmp/urls.json")
	}
	if cfg.DatabaseDSN != "postgres://user:pass@localhost/db" {
		t.Fatalf("DatabaseDSN = %q, want %q", cfg.DatabaseDSN, "postgres://user:pass@localhost/db")
	}
	if cfg.AuditFile != "/tmp/audit.jsonl" {
		t.Fatalf("AuditFile = %q, want %q", cfg.AuditFile, "/tmp/audit.jsonl")
	}
	if cfg.AuditURL != "http://audit.local/events" {
		t.Fatalf("AuditURL = %q, want %q", cfg.AuditURL, "http://audit.local/events")
	}
	if !cfg.EnableHTTPS {
		t.Fatal("EnableHTTPS = false, want true")
	}
}

func TestLoadGivesPriorityToFlagsAndEnvironment(t *testing.T) {
	clearConfigEnv(t)
	path := writeConfigFile(t, `{
		"server_address": "from-config:8080",
		"base_url": "http://from-config",
		"file_storage_path": "from-config.json",
		"database_dsn": "from-config-dsn",
		"enable_https": false
	}`)
	flagSet := resetFlags(t,
		"-c", path,
		"-a", "from-flag:8080",
		"-b", "http://from-flag",
		"-f", "from-flag.json",
		"-d", "from-flag-dsn",
		"-s",
	)
	t.Setenv("SERVER_ADDRESS", "from-env:8080")
	t.Setenv("ENABLE_HTTPS", "false")

	cfg, err := Load(flagSet)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.ServerAddr != "from-env:8080" {
		t.Fatalf("ServerAddr = %q, want env value", cfg.ServerAddr)
	}
	if cfg.BaseURL != "http://from-flag" {
		t.Fatalf("BaseURL = %q, want flag value", cfg.BaseURL)
	}
	if cfg.FileStoragePath != "from-flag.json" {
		t.Fatalf("FileStoragePath = %q, want flag value", cfg.FileStoragePath)
	}
	if cfg.DatabaseDSN != "from-flag-dsn" {
		t.Fatalf("DatabaseDSN = %q, want flag value", cfg.DatabaseDSN)
	}
	if cfg.EnableHTTPS {
		t.Fatal("EnableHTTPS = true, want env value false")
	}
}

func TestLoadReadsConfigPathFromLongFlagAndEnvironment(t *testing.T) {
	clearConfigEnv(t)
	flagPath := writeConfigFile(t, `{"server_address": "from-long-flag:8080"}`)
	envPath := writeConfigFile(t, `{"server_address": "from-env-config:8080"}`)

	t.Run("long flag", func(t *testing.T) {
		flagSet := resetFlags(t, "-config", flagPath)

		cfg, err := Load(flagSet)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.ServerAddr != "from-long-flag:8080" {
			t.Fatalf("ServerAddr = %q, want long flag config value", cfg.ServerAddr)
		}
	})

	t.Run("environment", func(t *testing.T) {
		flagSet := resetFlags(t, "-config", flagPath)
		t.Setenv("CONFIG", envPath)

		cfg, err := Load(flagSet)
		if err != nil {
			t.Fatalf("Load() error = %v", err)
		}

		if cfg.ServerAddr != "from-env-config:8080" {
			t.Fatalf("ServerAddr = %q, want env config value", cfg.ServerAddr)
		}
	})
}

func TestLoadReturnsConfigFileError(t *testing.T) {
	clearConfigEnv(t)
	flagSet := resetFlags(t, "-c", filepath.Join(t.TempDir(), "missing.json"))

	cfg, err := Load(flagSet)

	if err == nil {
		t.Fatal("Load() error = nil, want error")
	}
	if cfg != nil {
		t.Fatalf("Load() config = %#v, want nil", cfg)
	}
	if !strings.Contains(err.Error(), "load config file") {
		t.Fatalf("Load() error = %q, want config file context", err.Error())
	}
}

func resetFlags(t *testing.T, args ...string) *flag.FlagSet {
	t.Helper()

	os.Args = append([]string{"shortener"}, args...)
	return flag.NewFlagSet("shortener", flag.ContinueOnError)
}

func clearConfigEnv(t *testing.T) {
	t.Helper()

	envKeys := []string{
		"SERVER_ADDRESS",
		"BASE_URL",
		"FILE_STORAGE_PATH",
		"DATABASE_DSN",
		"AUDIT_FILE",
		"AUDIT_URL",
		"ENABLE_HTTPS",
		"TRUSTED_SUBNET",
		"CONFIG",
	}
	for _, key := range envKeys {
		value, ok := os.LookupEnv(key)
		if err := os.Unsetenv(key); err != nil {
			t.Fatalf("unset %s: %v", key, err)
		}
		t.Cleanup(func() {
			if ok {
				if err := os.Setenv(key, value); err != nil {
					t.Fatalf("restore %s: %v", key, err)
				}
				return
			}
			if err := os.Unsetenv(key); err != nil {
				t.Fatalf("restore unset %s: %v", key, err)
			}
		})
	}
}

func writeConfigFile(t *testing.T, content string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write config file: %v", err)
	}
	return path
}
