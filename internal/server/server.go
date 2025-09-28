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

	// Forecast API endpoint
	mux.HandleFunc("/api/forecast", h.handleForecast)

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
	Scenarios []string      `json:"scenarios"`
	Rows      []forecastRow `json:"rows"`
	CSV       string        `json:"csv"`
	Warnings  []string      `json:"warnings,omitempty"`
	Duration  string        `json:"duration"`
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

	cfg, err := config.LoadConfigurationFromReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		h.respondError(w, http.StatusBadRequest, err.Error())
		return
	}

	warnings := cfg.ValidateConfiguration()
	if err := cfg.ParseDateLists(); err != nil {
		h.respondError(w, http.StatusBadRequest, fmt.Sprintf("failed to parse dates: %v", err))
		return
	}

	if err := cfg.ProcessLoans(h.logger); err != nil {
		h.respondError(w, http.StatusBadRequest, fmt.Sprintf("failed to process loans: %v", err))
		return
	}

	results, err := forecast.GetForecast(h.logger, *cfg)
	if err != nil {
		h.respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to compute forecast: %v", err))
		return
	}

	elapsed := time.Since(start)

	response := forecastResponse{
		Scenarios: extractScenarioNames(results),
		Rows:      buildRows(results),
		CSV:       output.CsvString(results),
		Warnings:  warnings,
		Duration:  elapsed.String(),
	}

	if h.logger != nil {
		h.logger.Info("forecast computed",
			zap.String("op", "server.handleForecast"),
			zap.Int("scenarios", len(response.Scenarios)),
			zap.Int("rows", len(response.Rows)),
			zap.Duration("duration", elapsed),
		)
	}

	h.writeJSON(w, http.StatusOK, response)
}

func (h *handler) respondError(w http.ResponseWriter, status int, msg string) {
	if h.logger != nil {
		h.logger.Error("forecast request failed",
			zap.String("op", "server.handleForecast"),
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
