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
