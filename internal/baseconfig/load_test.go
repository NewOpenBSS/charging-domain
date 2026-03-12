package baseconfig

import (
	"os"
	"path/filepath"
	"testing"
)

type TestConfig struct {
	Name    string `yaml:"name"`
	Version int    `yaml:"version"`
}

func TestLoadConfig(t *testing.T) {
	// Setup temporary directory for test config files
	tmpDir, err := os.MkdirTemp("", "baseconfig-test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	validConfigFile := filepath.Join(tmpDir, "config.yaml")
	validYAML := "name: test-app\nversion: 1"
	if err := os.WriteFile(validConfigFile, []byte(validYAML), 0644); err != nil {
		t.Fatalf("failed to write valid config file: %v", err)
	}

	invalidYAMLFile := filepath.Join(tmpDir, "invalid.yaml")
	invalidYAML := "name: : invalid"
	if err := os.WriteFile(invalidYAMLFile, []byte(invalidYAML), 0644); err != nil {
		t.Fatalf("failed to write invalid config file: %v", err)
	}

	t.Run("valid config file", func(t *testing.T) {
		var cfg TestConfig
		err := LoadConfig(validConfigFile, &cfg)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if cfg.Name != "test-app" || cfg.Version != 1 {
			t.Errorf("unexpected config values: %+v", cfg)
		}
	})

	t.Run("non-existent file", func(t *testing.T) {
		var cfg TestConfig
		err := LoadConfig(filepath.Join(tmpDir, "non-existent.yaml"), &cfg)
		if err == nil {
			t.Error("expected error for non-existent file, got nil")
		}
	})

	t.Run("invalid YAML content", func(t *testing.T) {
		var cfg TestConfig
		err := LoadConfig(invalidYAMLFile, &cfg)
		if err == nil {
			t.Error("expected error for invalid YAML, got nil")
		}
	})

	t.Run("empty config path", func(t *testing.T) {
		var cfg TestConfig
		err := LoadConfig("", &cfg)
		if err == nil {
			t.Error("expected error for empty config path, got nil")
		}
	})

	t.Run("override with environment variable", func(t *testing.T) {
		envConfigFile := filepath.Join(tmpDir, "env-config.yaml")
		envYAML := "name: env-app\nversion: 2"
		if err := os.WriteFile(envConfigFile, []byte(envYAML), 0644); err != nil {
			t.Fatalf("failed to write env config file: %v", err)
		}

		os.Setenv("CONFIG_FILE", envConfigFile)
		defer os.Unsetenv("CONFIG_FILE")

		var cfg TestConfig
		// Pass an invalid file path as argument, but the env var should override it
		err := LoadConfig("ignored.yaml", &cfg)
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
		if cfg.Name != "env-app" || cfg.Version != 2 {
			t.Errorf("unexpected config values: %+v", cfg)
		}
	})
}
