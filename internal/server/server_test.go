package server

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/iwvelando/finance-forecast/pkg/constants"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

func TestHandleForecastSuccess(t *testing.T) {
	handler := NewHandler(zap.NewNop(), constants.DefaultMaxUploadSizeBytes)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	configPath := filepath.Join("..", "..", "test", "test_config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read test config: %v", err)
	}

	part, err := writer.CreateFormFile("file", "test_config.yaml")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("failed to write form data: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/forecast", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp forecastResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Scenarios) == 0 {
		t.Fatal("expected scenarios in response")
	}
	if len(resp.Rows) == 0 {
		t.Fatal("expected rows in response")
	}
	if resp.CSV == "" {
		t.Fatal("expected CSV data in response")
	}
	if resp.Duration == "" {
		t.Fatal("expected duration in response")
	}
	if resp.Config == nil {
		t.Fatal("expected config data in response")
	}
	if resp.ConfigYAML == "" {
		t.Fatal("expected config YAML in response")
	}
}

func TestHandleForecastEditorSuccess(t *testing.T) {
	handler := NewHandler(zap.NewNop(), constants.DefaultMaxUploadSizeBytes)

	configPath := filepath.Join("..", "..", "test", "test_config.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("failed to read test config: %v", err)
	}

	var payload map[string]interface{}
	if err := yaml.Unmarshal(data, &payload); err != nil {
		t.Fatalf("failed to unmarshal yaml: %v", err)
	}

	rr := performEditorJSON(t, handler, payload, "/api/editor/forecast")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp forecastResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Scenarios) == 0 {
		t.Fatal("expected scenarios in response")
	}
	if len(resp.Rows) == 0 {
		t.Fatal("expected rows in response")
	}
	if resp.Config == nil {
		t.Fatal("expected config data in response")
	}
	if resp.ConfigYAML == "" {
		t.Fatal("expected config YAML in response")
	}
}

func TestHandleConfigExport(t *testing.T) {
	handler := NewHandler(zap.NewNop(), constants.DefaultMaxUploadSizeBytes)

	payload := map[string]interface{}{
		"scenarios": []interface{}{
			map[string]interface{}{
				"name":   "sample",
				"active": true,
			},
		},
		"common": map[string]interface{}{
			"startingValue": 1500.0,
			"deathDate":     "2050-01",
		},
		"output": map[string]interface{}{
			"format": "pretty",
		},
		"logging": map[string]interface{}{
			"level":   "info",
			"enabled": true,
		},
	}

	rr := performEditorJSON(t, handler, payload, "/api/editor/export")

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	yamlStr := resp["configYaml"]
	if yamlStr == "" {
		t.Fatal("expected configYaml in response")
	}
	if !strings.Contains(yamlStr, "common:") {
		t.Fatalf("expected yaml to contain common section, got %q", yamlStr)
	}
	if !strings.Contains(yamlStr, "scenarios:") {
		t.Fatalf("expected yaml to contain scenarios section, got %q", yamlStr)
	}

	lines := strings.Split(strings.TrimRight(yamlStr, "\n"), "\n")
	orderedTop := make([]string, 0, 2)
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			continue
		}
		orderedTop = append(orderedTop, strings.TrimSpace(line))
		if len(orderedTop) == 2 {
			break
		}
	}

	if len(orderedTop) < 2 {
		t.Fatalf("expected at least two top-level keys in yaml, got %v", orderedTop)
	}
	if !strings.HasPrefix(orderedTop[0], "logging:") {
		t.Fatalf("expected logging to be first key, got %q", orderedTop[0])
	}
	if !strings.HasPrefix(orderedTop[1], "output:") {
		t.Fatalf("expected output to be second key, got %q", orderedTop[1])
	}
}

func TestHandleForecastMethodNotAllowed(t *testing.T) {
	handler := NewHandler(zap.NewNop(), constants.DefaultMaxUploadSizeBytes)

	req := httptest.NewRequest(http.MethodGet, "/api/forecast", nil)
	rr := httptest.NewRecorder()

	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", rr.Code)
	}
}

func TestHandleForecastUploadTooLarge(t *testing.T) {
	handler := NewHandler(zap.NewNop(), 64)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("file", "config.yaml")
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	if _, err := part.Write([]byte(strings.Repeat("a", 128))); err != nil {
		t.Fatalf("failed to write oversized payload: %v", err)
	}

	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/forecast", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status 413, got %d", rr.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if !strings.Contains(resp["error"], "upload exceeds limit") {
		t.Fatalf("expected upload limit error message, got %q", resp["error"])
	}
}

func TestHandleForecastMissingFile(t *testing.T) {
	handler := NewHandler(zap.NewNop(), constants.DefaultMaxUploadSizeBytes)

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/forecast", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if resp["error"] != "missing configuration file" {
		t.Fatalf("expected missing file error, got %q", resp["error"])
	}
}

func TestHandleForecastInvalidYAML(t *testing.T) {
	handler := NewHandler(zap.NewNop(), constants.DefaultMaxUploadSizeBytes)

	rr := performUpload(t, handler, "common: [", "config.yaml")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if !strings.Contains(resp["error"], "error reading config data") {
		t.Fatalf("expected parse error message, got %q", resp["error"])
	}
}

func TestHandleForecastDateParseFailure(t *testing.T) {
	handler := NewHandler(zap.NewNop(), constants.DefaultMaxUploadSizeBytes)

	configYAML := `
common:
  startingValue: 0
  deathDate: 2025-01
  events:
    - name: bad frequency
      amount: 10
      frequency: 0
scenarios:
  - name: sample
    active: true
`

	rr := performUpload(t, handler, configYAML, "config.yaml")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if !strings.Contains(resp["error"], "event frequency must be greater than zero") {
		t.Fatalf("expected frequency error, got %q", resp["error"])
	}
}

func TestHandleForecastProcessLoansFailure(t *testing.T) {
	handler := NewHandler(zap.NewNop(), constants.DefaultMaxUploadSizeBytes)

	configYAML := `
common:
  startingValue: 0
  deathDate: 2025-01
scenarios:
  - name: broken loan
    active: true
    loans:
      - {}
`

	rr := performUpload(t, handler, configYAML, "config.yaml")

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}

	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if !strings.Contains(resp["error"], "loan name cannot be empty") {
		t.Fatalf("expected loan error message, got %q", resp["error"])
	}
}

func TestStaticAssetsServed(t *testing.T) {
	handler := NewHandler(zap.NewNop(), constants.DefaultMaxUploadSizeBytes)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200 for index, got %d", rr.Code)
	}

	if !strings.Contains(rr.Body.String(), "Finance Forecast") {
		t.Fatalf("expected HTML body to contain title, got %q", rr.Body.String())
	}

	cssReq := httptest.NewRequest(http.MethodGet, "/styles.css", nil)
	cssRR := httptest.NewRecorder()
	handler.ServeHTTP(cssRR, cssReq)

	if cssRR.Code != http.StatusOK {
		t.Fatalf("expected status 200 for css, got %d", cssRR.Code)
	}
	if !strings.Contains(cssRR.Body.String(), ":root") {
		t.Fatalf("expected CSS body to contain styles, got %q", cssRR.Body.String())
	}
}

func performUpload(t *testing.T, handler http.Handler, content, filename string) *httptest.ResponseRecorder {
	t.Helper()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		t.Fatalf("failed to create form file: %v", err)
	}
	if _, err := part.Write([]byte(content)); err != nil {
		t.Fatalf("failed to write form data: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/forecast", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	return rr
}

func performEditorJSON(t *testing.T, handler http.Handler, payload map[string]interface{}, path string) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	return rr
}
