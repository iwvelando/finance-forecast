package server

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/iwvelando/finance-forecast/internal/config"
	"github.com/iwvelando/finance-forecast/pkg/constants"
	"gopkg.in/yaml.v3"
)

// Config defines runtime parameters for the HTTP server.
type Config struct {
	Address         string               `yaml:"address"`
	MaxUploadSize   string               `yaml:"maxUploadSize"`
	Logging         config.LoggingConfig `yaml:"logging"`
	uploadSizeBytes int64
}

// LoadConfig loads the server configuration from YAML. If the file does not exist,
// defaults are returned without error.
func LoadConfig(path string) (*Config, error) {
	cfg := &Config{
		Address:         constants.DefaultServerAddress,
		MaxUploadSize:   fmt.Sprintf("%d", constants.DefaultMaxUploadSizeBytes),
		Logging:         config.LoggingConfig{},
		uploadSizeBytes: constants.DefaultMaxUploadSizeBytes,
	}

	if path == "" {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cfg, nil
		}
		return nil, fmt.Errorf("failed to read server config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse server config: %w", err)
	}

	if err := cfg.normalize(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// UploadSizeBytes returns the configured upload size in bytes.
func (c *Config) UploadSizeBytes() int64 {
	return c.uploadSizeBytes
}

// SetUploadSizeBytes overrides the configured upload size.
func (c *Config) SetUploadSizeBytes(size int64) {
	if size > 0 {
		c.uploadSizeBytes = size
		c.MaxUploadSize = fmt.Sprintf("%d", size)
	}
}

func (c *Config) normalize() error {
	if c.Address == "" {
		c.Address = constants.DefaultServerAddress
	}

	sizeStr := strings.TrimSpace(c.MaxUploadSize)
	if sizeStr == "" {
		c.uploadSizeBytes = constants.DefaultMaxUploadSizeBytes
		c.MaxUploadSize = fmt.Sprintf("%d", constants.DefaultMaxUploadSizeBytes)
		return nil
	}

	bytes, err := ParseSize(sizeStr)
	if err != nil {
		return err
	}
	if bytes <= 0 {
		bytes = constants.DefaultMaxUploadSizeBytes
	}
	c.uploadSizeBytes = bytes
	return nil
}

// ParseSize converts a human-friendly byte string (e.g., "256K", "10M") into bytes.
func ParseSize(value string) (int64, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return constants.DefaultMaxUploadSizeBytes, nil
	}

	upper := strings.ToUpper(trimmed)
	idx := len(upper)
	for idx > 0 && !unicode.IsDigit(rune(upper[idx-1])) {
		idx--
	}
	if idx == 0 {
		return 0, fmt.Errorf("invalid size: %s", value)
	}
	numPart := strings.TrimSpace(upper[:idx])
	unitPart := strings.TrimSpace(upper[idx:])

	if numPart == "" {
		return 0, fmt.Errorf("invalid size: %s", value)
	}

	n, err := strconv.ParseInt(numPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid size value %q: %w", value, err)
	}

	var multiplier int64
	switch unitPart {
	case "", "B":
		multiplier = 1
	case "K", "KB":
		multiplier = 1024
	case "M", "MB":
		multiplier = 1024 * 1024
	case "G", "GB":
		multiplier = 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unsupported size unit %q", unitPart)
	}

	result := n * multiplier
	if result < 0 {
		return 0, fmt.Errorf("size overflow for value %s", value)
	}
	return result, nil
}
