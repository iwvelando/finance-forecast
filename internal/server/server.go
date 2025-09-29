package server

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/iwvelando/finance-forecast/internal/config"
	"github.com/iwvelando/finance-forecast/internal/forecast"
	"github.com/iwvelando/finance-forecast/pkg/constants"
	"github.com/iwvelando/finance-forecast/pkg/output"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

//go:embed static/*
var staticFiles embed.FS

type handler struct {
	logger        *zap.Logger
	maxUploadSize int64
}

// NewHandler constructs the HTTP handler that serves the web UI and forecast API.
func NewHandler(logger *zap.Logger, maxUploadSize int64) http.Handler {
	if logger == nil {
		logger = zap.NewNop()
	}

	if maxUploadSize <= 0 {
		maxUploadSize = constants.DefaultMaxUploadSizeBytes
	}

	h := &handler{logger: logger, maxUploadSize: maxUploadSize}

	mux := http.NewServeMux()

	// Forecast API endpoint (file upload)
	mux.HandleFunc("/api/forecast", h.handleForecast)

	// Forecast API endpoint for editor-driven updates
	mux.HandleFunc("/api/editor/forecast", h.handleForecastEditor)

	// Config serialization endpoint for editor downloads
	mux.HandleFunc("/api/editor/export", h.handleConfigExport)

	// Static assets (web UI)
	sub, err := fs.Sub(staticFiles, "static")
	if err != nil {
		panic(fmt.Sprintf("failed to prepare embedded static files: %v", err))
	}
	fileServer := http.FileServer(http.FS(sub))
	mux.Handle("/", fileServer)

	return mux
}

type forecastResponse struct {
	Scenarios  []string               `json:"scenarios"`
	Rows       []forecastRow          `json:"rows"`
	CSV        string                 `json:"csv"`
	Warnings   []string               `json:"warnings,omitempty"`
	Duration   string                 `json:"duration"`
	Config     map[string]interface{} `json:"config,omitempty"`
	ConfigYAML string                 `json:"configYaml,omitempty"`
}

type forecastRow struct {
	Date   string          `json:"date"`
	Values []scenarioValue `json:"values"`
}

type scenarioValue struct {
	Amount *float64 `json:"amount,omitempty"`
	Notes  []string `json:"notes,omitempty"`
}

func (h *handler) handleForecast(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	start := time.Now()
	if h.maxUploadSize > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, h.maxUploadSize)
	}
	if err := r.ParseMultipartForm(h.maxUploadSize); err != nil {
		var maxBytesErr *http.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			h.respondError(w, http.StatusRequestEntityTooLarge,
				fmt.Sprintf("upload exceeds limit of %d bytes", h.maxUploadSize))
			return
		}
		h.respondError(w, http.StatusBadRequest, fmt.Sprintf("failed to parse upload: %v", err))
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		h.respondError(w, http.StatusBadRequest, "missing configuration file")
		return
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && h.logger != nil {
			h.logger.Warn("failed to close uploaded file",
				zap.String("op", "server.handleForecast"),
				zap.Error(closeErr),
			)
		}
	}()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, file); err != nil {
		h.respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to read configuration: %v", err))
		return
	}

	configBytes := buf.Bytes()
	configMap, err := decodeYAMLToMap(configBytes)
	if err != nil {
		h.respondError(w, http.StatusBadRequest, fmt.Sprintf("error reading config data, %v", err))
		return
	}

	h.runForecast(w, configBytes, configMap, start, "server.handleForecast")
}

func (h *handler) handleForecastEditor(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	start := time.Now()

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.respondErrorWithOp(w, http.StatusBadRequest, fmt.Sprintf("failed to decode configuration: %v", err), "server.handleForecastEditor")
		return
	}
	if payload == nil {
		payload = make(map[string]interface{})
	}

	configBytes, err := yaml.Marshal(payload)
	if err != nil {
		h.respondErrorWithOp(w, http.StatusBadRequest, fmt.Sprintf("failed to encode configuration: %v", err), "server.handleForecastEditor")
		return
	}

	configMap, err := decodeYAMLToMap(configBytes)
	if err != nil {
		h.respondErrorWithOp(w, http.StatusBadRequest, fmt.Sprintf("failed to parse configuration: %v", err), "server.handleForecastEditor")
		return
	}

	h.runForecast(w, configBytes, configMap, start, "server.handleForecastEditor")
}

func (h *handler) handleConfigExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		h.respondErrorWithOp(w, http.StatusBadRequest, fmt.Sprintf("failed to decode configuration: %v", err), "server.handleConfigExport")
		return
	}
	if payload == nil {
		payload = make(map[string]interface{})
	}

	yamlBytes, err := yaml.Marshal(payload)
	if err != nil {
		h.respondErrorWithOp(w, http.StatusBadRequest, fmt.Sprintf("failed to encode configuration: %v", err), "server.handleConfigExport")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{
		"configYaml": string(yamlBytes),
	})
}

func (h *handler) runForecast(w http.ResponseWriter, configBytes []byte, configMap map[string]interface{}, start time.Time, op string) {
	cfg, err := config.LoadConfigurationFromReader(bytes.NewReader(configBytes))
	if err != nil {
		h.respondErrorWithOp(w, http.StatusBadRequest, err.Error(), op)
		return
	}

	warnings := cfg.ValidateConfiguration()
	if err := cfg.ParseDateLists(); err != nil {
		h.respondErrorWithOp(w, http.StatusBadRequest, fmt.Sprintf("failed to parse dates: %v", err), op)
		return
	}

	if err := cfg.ProcessLoans(h.logger); err != nil {
		h.respondErrorWithOp(w, http.StatusBadRequest, fmt.Sprintf("failed to process loans: %v", err), op)
		return
	}

	results, err := forecast.GetForecast(h.logger, *cfg)
	if err != nil {
		h.respondErrorWithOp(w, http.StatusInternalServerError, fmt.Sprintf("failed to compute forecast: %v", err), op)
		return
	}

	elapsed := time.Since(start)

	if configMap == nil {
		configMap = make(map[string]interface{})
	}

	response := forecastResponse{
		Scenarios:  extractScenarioNames(results),
		Rows:       buildRows(results),
		CSV:        output.CsvString(results),
		Warnings:   warnings,
		Duration:   elapsed.String(),
		Config:     configMap,
		ConfigYAML: string(configBytes),
	}

	if h.logger != nil {
		h.logger.Info("forecast computed",
			zap.String("op", op),
			zap.Int("scenarios", len(response.Scenarios)),
			zap.Int("rows", len(response.Rows)),
			zap.Duration("duration", elapsed),
		)
	}

	h.writeJSON(w, http.StatusOK, response)
}

func decodeYAMLToMap(data []byte) (map[string]interface{}, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return make(map[string]interface{}), nil
	}

	var result map[string]interface{}
	if err := yaml.Unmarshal(trimmed, &result); err != nil {
		return nil, err
	}
	if result == nil {
		result = make(map[string]interface{})
	}
	return result, nil
}

func (h *handler) respondError(w http.ResponseWriter, status int, msg string) {
	h.respondErrorWithOp(w, status, msg, "server.handleForecast")
}

func (h *handler) respondErrorWithOp(w http.ResponseWriter, status int, msg string, op string) {
	if h.logger != nil {
		h.logger.Error("forecast request failed",
			zap.String("op", op),
			zap.Int("status", status),
			zap.String("error", msg),
		)
	}

	h.writeJSON(w, status, map[string]string{"error": msg})
}

func (h *handler) writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil && h.logger != nil {
		h.logger.Error("failed to write JSON response", zap.Error(err))
	}
}

func extractScenarioNames(results []forecast.Forecast) []string {
	names := make([]string, 0, len(results))
	for _, scenario := range results {
		names = append(names, scenario.Name)
	}
	return names
}

func buildRows(results []forecast.Forecast) []forecastRow {
	dateSet := make(map[string]struct{})
	for _, scenario := range results {
		for date := range scenario.Data {
			dateSet[date] = struct{}{}
		}
	}

	dates := make([]string, 0, len(dateSet))
	for date := range dateSet {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	rows := make([]forecastRow, 0, len(dates))
	for _, date := range dates {
		row := forecastRow{Date: date}
		for _, scenario := range results {
			if balance, ok := scenario.Data[date]; ok {
				value := balance
				notes := scenario.Notes[date]
				row.Values = append(row.Values, scenarioValue{
					Amount: &value,
					Notes:  normalizeNotes(notes),
				})
			} else {
				row.Values = append(row.Values, scenarioValue{})
			}
		}
		rows = append(rows, row)
	}

	return rows
}

func normalizeNotes(notes []string) []string {
	if len(notes) == 0 {
		return nil
	}

	filtered := make([]string, 0, len(notes))
	for _, note := range notes {
		if trimmed := strings.TrimSpace(note); trimmed != "" {
			filtered = append(filtered, trimmed)
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}
