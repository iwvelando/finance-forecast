package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/iwvelando/finance-forecast/pkg/constants"
)

func TestLoadConfigDefaultsWhenMissing(t *testing.T) {
	cfg, err := LoadConfig(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Address == "" {
		t.Fatalf("expected default address, got empty")
	}
	if cfg.UploadSizeBytes() <= 0 {
		t.Fatalf("expected positive default max upload size, got %d", cfg.UploadSizeBytes())
	}
	if cfg.Logging.Level != "" || cfg.Logging.Format != "" || cfg.Logging.OutputFile != "" {
		t.Fatalf("expected empty logging defaults, got %+v", cfg.Logging)
	}
}

func TestLoadConfigOverrides(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "server-config.yaml")

	contents := []byte(`address: 127.0.0.1:9000
maxUploadSize: 2M
logging:
  level: debug
  format: console
  outputFile: /tmp/server.log
`)
	if err := os.WriteFile(path, contents, 0600); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	if cfg.Address != "127.0.0.1:9000" {
		t.Fatalf("expected address override, got %s", cfg.Address)
	}
	if cfg.UploadSizeBytes() != 2*1024*1024 {
		t.Fatalf("expected max upload override, got %d", cfg.UploadSizeBytes())
	}
	if cfg.Logging.Level != "debug" {
		t.Fatalf("expected logging level debug, got %s", cfg.Logging.Level)
	}
	if cfg.Logging.Format != "console" {
		t.Fatalf("expected logging format console, got %s", cfg.Logging.Format)
	}
	if cfg.Logging.OutputFile != "/tmp/server.log" {
		t.Fatalf("expected logging outputFile /tmp/server.log, got %s", cfg.Logging.OutputFile)
	}
}

func TestLoadConfigInvalidYaml(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")

	if err := os.WriteFile(path, []byte("maxUploadSize: invalid"), 0600); err != nil {
		t.Fatalf("failed to write temp config: %v", err)
	}

	if _, err := LoadConfig(path); err == nil {
		t.Fatal("expected error for invalid YAML but got nil")
	}
}

func TestParseSize(t *testing.T) {
	tests := map[string]int64{
		"":          constants.DefaultMaxUploadSizeBytes,
		"1024":      1024,
		"512b":      512,
		"256K":      256 * 1024,
		"1m":        1024 * 1024,
		"3MB":       3 * 1024 * 1024,
		"2G":        2 * 1024 * 1024 * 1024,
		"  4096   ": 4096,
	}

	for input, expected := range tests {
		got, err := ParseSize(input)
		if err != nil {
			t.Fatalf("parseSize(%q) returned error: %v", input, err)
		}
		if got != expected {
			t.Fatalf("parseSize(%q) = %d, expected %d", input, got, expected)
		}
	}

	if _, err := ParseSize("1TB"); err == nil {
		t.Fatal("expected error for unsupported unit")
	}
	if _, err := ParseSize("abc"); err == nil {
		t.Fatal("expected error for invalid number")
	}
}
