package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestConfigIsAutoUpdateEnabled(t *testing.T) {
	t.Run("defaults to enabled when unset", func(t *testing.T) {
		cfg := Config{}
		if !cfg.IsAutoUpdateEnabled() {
			t.Fatalf("expected auto-update to be enabled by default")
		}
	})

	t.Run("disabled when flag is true", func(t *testing.T) {
		disabled := true
		cfg := Config{AutoUpdateDisabled: &disabled}
		if cfg.IsAutoUpdateEnabled() {
			t.Fatalf("expected auto-update to be disabled")
		}
	})
}

func TestUpsertGlobalAutoUpdateEnabled(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	if err := UpsertGlobalAutoUpdateEnabled(false); err != nil {
		t.Fatalf("unexpected error writing auto-update setting: %v", err)
	}

	cfg := mustReadGlobalConfig(t, tempHome)
	if cfg.AutoUpdateDisabled == nil || !*cfg.AutoUpdateDisabled {
		t.Fatalf("expected auto_update_disabled to be true after disabling auto-update")
	}
}

func TestUpsertGlobalConfigPreservesAutoUpdateSetting(t *testing.T) {
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)

	if err := UpsertGlobalAutoUpdateEnabled(false); err != nil {
		t.Fatalf("unexpected error writing auto-update setting: %v", err)
	}

	if err := UpsertGlobalConfig(Config{OrganizationId: "org_test"}); err != nil {
		t.Fatalf("unexpected error writing global config: %v", err)
	}

	cfg := mustReadGlobalConfig(t, tempHome)
	if cfg.AutoUpdateDisabled == nil || !*cfg.AutoUpdateDisabled {
		t.Fatalf("expected auto_update_disabled to remain true after other global config updates")
	}
	if cfg.OrganizationId != "org_test" {
		t.Fatalf("expected organization_id to be updated, got %q", cfg.OrganizationId)
	}
}

func mustReadGlobalConfig(t *testing.T, homeDir string) Config {
	t.Helper()

	configPath := filepath.Join(homeDir, ManifestConfigFile)
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read global config: %v", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("failed to decode global config JSON: %v", err)
	}

	return cfg
}
