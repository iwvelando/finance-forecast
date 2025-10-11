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
	"strconv"
	"strings"
	"time"

	"github.com/iwvelando/finance-forecast/internal/config"
	"github.com/iwvelando/finance-forecast/internal/forecast"
	"github.com/iwvelando/finance-forecast/internal/optimizer"
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
	version       string
}

type forecastOptions struct {
	Optimize bool
}

// NewHandler constructs the HTTP handler that serves the web UI and forecast API.
func NewHandler(logger *zap.Logger, maxUploadSize int64, version string) http.Handler {
	if logger == nil {
		logger = zap.NewNop()
	}

	if maxUploadSize <= 0 {
		maxUploadSize = constants.DefaultMaxUploadSizeBytes
	}

	trimmedVersion := strings.TrimSpace(version)
	if trimmedVersion == "" {
		trimmedVersion = "dev"
	}

	h := &handler{logger: logger, maxUploadSize: maxUploadSize, version: trimmedVersion}

	mux := http.NewServeMux()

	// Forecast API endpoint (file upload)
	mux.HandleFunc("/api/forecast", h.handleForecast)

	// Forecast API endpoint for editor-driven updates
	mux.HandleFunc("/api/editor/forecast", h.handleForecastEditor)

	// Config serialization endpoint for editor downloads
	mux.HandleFunc("/api/editor/export", h.handleConfigExport)

	// Version endpoint for UI metadata
	mux.HandleFunc("/api/version", h.handleVersion)

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
	Metrics    []scenarioMetrics      `json:"metrics,omitempty"`
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
	Liquid *float64 `json:"liquid,omitempty"`
	Total  *float64 `json:"total,omitempty"`
	Notes  []string `json:"notes,omitempty"`
}

type scenarioMetrics struct {
	EmergencyFund *emergencyFundMetric `json:"emergencyFund,omitempty"`
	Optimizations []optimizationMetric `json:"optimizations,omitempty"`
}

type optimizationMetric struct {
	TargetName  string   `json:"targetName"`
	Field       string   `json:"field"`
	Original    float64  `json:"original"`
	Value       float64  `json:"value"`
	Floor       float64  `json:"floor"`
	MinimumCash float64  `json:"minimumCash"`
	Headroom    float64  `json:"headroom"`
	Iterations  int      `json:"iterations"`
	Converged   bool     `json:"converged"`
	Notes       []string `json:"notes,omitempty"`
}

type emergencyFundMetric struct {
	TargetMonths           float64 `json:"targetMonths"`
	AverageMonthlyExpenses float64 `json:"averageMonthlyExpenses"`
	TargetAmount           float64 `json:"targetAmount"`
	InitialLiquid          float64 `json:"initialLiquid"`
	FundedMonths           float64 `json:"fundedMonths"`
	Shortfall              float64 `json:"shortfall"`
	Surplus                float64 `json:"surplus"`
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

	h.runForecast(w, configBytes, configMap, start, "server.handleForecast", forecastOptions{})
}

func (h *handler) handleVersion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{
		"version": h.version,
	})
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

	configPayload := payload
	if rawConfig, ok := payload["config"]; ok {
		cfgMap, ok := rawConfig.(map[string]interface{})
		if !ok {
			h.respondErrorWithOp(w, http.StatusBadRequest, "invalid config payload: expected object", "server.handleForecastEditor")
			return
		}
		configPayload = cfgMap
	}

	options := forecastOptions{}
	if rawOptions, ok := payload["options"]; ok {
		optsMap, ok := rawOptions.(map[string]interface{})
		if !ok {
			h.respondErrorWithOp(w, http.StatusBadRequest, "invalid options payload: expected object", "server.handleForecastEditor")
			return
		}
		if optimizeVal, ok := optsMap["optimize"]; ok {
			options.Optimize = coerceBool(optimizeVal)
		}
	}

	configBytes, err := yaml.Marshal(configPayload)
	if err != nil {
		h.respondErrorWithOp(w, http.StatusBadRequest, fmt.Sprintf("failed to encode configuration: %v", err), "server.handleForecastEditor")
		return
	}

	configMap, err := decodeYAMLToMap(configBytes)
	if err != nil {
		h.respondErrorWithOp(w, http.StatusBadRequest, fmt.Sprintf("failed to parse configuration: %v", err), "server.handleForecastEditor")
		return
	}

	h.runForecast(w, configBytes, configMap, start, "server.handleForecastEditor", options)
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

	yamlBytes, err := marshalOrderedConfigYAML(payload)
	if err != nil {
		h.respondErrorWithOp(w, http.StatusBadRequest, fmt.Sprintf("failed to encode configuration: %v", err), "server.handleConfigExport")
		return
	}

	h.writeJSON(w, http.StatusOK, map[string]string{
		"configYaml": string(yamlBytes),
	})
}

func marshalOrderedConfigYAML(payload map[string]interface{}) ([]byte, error) {
	items := make([]orderedItem, 0, len(payload))
	seen := make(map[string]struct{})

	for _, key := range []string{"logging", "output"} {
		if value, ok := payload[key]; ok {
			items = append(items, orderedItem{key: key, value: value})
			seen[key] = struct{}{}
		}
	}

	remainingKeys := make([]string, 0, len(payload))
	for key := range payload {
		if _, already := seen[key]; already {
			continue
		}
		remainingKeys = append(remainingKeys, key)
	}
	sort.Strings(remainingKeys)
	for _, key := range remainingKeys {
		items = append(items, orderedItem{key: key, value: payload[key]})
	}

	ordered := orderedConfig{items: items}
	return yaml.Marshal(ordered)
}

type orderedConfig struct {
	items []orderedItem
}

type orderedItem struct {
	key   string
	value interface{}
}

func (o orderedConfig) MarshalYAML() (interface{}, error) {
	mapNode := &yaml.Node{
		Kind: yaml.MappingNode,
		Tag:  "!!map",
	}

	for _, item := range o.items {
		keyNode := &yaml.Node{
			Kind:  yaml.ScalarNode,
			Tag:   "!!str",
			Value: item.key,
		}
		valueNode := &yaml.Node{}
		if err := valueNode.Encode(item.value); err != nil {
			return nil, err
		}
		mapNode.Content = append(mapNode.Content, keyNode, valueNode)
	}

	return mapNode, nil
}

func (h *handler) runForecast(w http.ResponseWriter, configBytes []byte, configMap map[string]interface{}, start time.Time, op string, opts forecastOptions) {
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

	var optimizationResult *optimizer.Result
	if opts.Optimize {
		runner, err := optimizer.NewRunner(h.logger, cfg)
		if err != nil {
			h.respondErrorWithOp(w, http.StatusBadRequest, fmt.Sprintf("failed to initialize optimizer: %v", err), op)
			return
		}

		optimizationResult, err = runner.Run()
		if err != nil {
			h.respondErrorWithOp(w, http.StatusBadRequest, fmt.Sprintf("optimizer execution failed: %v", err), op)
			return
		}
	}

	results, err := forecast.GetForecast(h.logger, *cfg)
	if err != nil {
		h.respondErrorWithOp(w, http.StatusInternalServerError, fmt.Sprintf("failed to compute forecast: %v", err), op)
		return
	}

	if optimizationResult != nil && !optimizationResult.Empty() {
		optimizationResult.Apply(results)
	}

	if opts.Optimize {
		updatedBytes, err := yaml.Marshal(cfg)
		if err != nil {
			if h.logger != nil {
				h.logger.Warn("failed to marshal optimized configuration",
					zap.String("op", op),
					zap.Error(err),
				)
			}
		} else {
			configBytes = updatedBytes
			if updatedMap, mapErr := decodeYAMLToMap(updatedBytes); mapErr == nil {
				configMap = updatedMap
			} else if h.logger != nil {
				h.logger.Warn("failed to decode optimized configuration map",
					zap.String("op", op),
					zap.Error(mapErr),
				)
			}
		}
	}

	elapsed := time.Since(start)

	if configMap == nil {
		configMap = make(map[string]interface{})
	}

	response := forecastResponse{
		Scenarios:  extractScenarioNames(results),
		Rows:       buildRows(results),
		CSV:        output.CsvString(results),
		Metrics:    buildMetrics(results),
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
			liquidVal, liquidOK := scenario.Liquid[date]
			totalVal, totalOK := scenario.Data[date]
			notes := scenario.Notes[date]

			var liquidPtr *float64
			if liquidOK {
				v := liquidVal
				liquidPtr = &v
			}

			var totalPtr *float64
			if totalOK {
				v := totalVal
				totalPtr = &v
			}

			if liquidPtr != nil || totalPtr != nil || len(notes) > 0 {
				row.Values = append(row.Values, scenarioValue{
					Liquid: liquidPtr,
					Total:  totalPtr,
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

func buildMetrics(results []forecast.Forecast) []scenarioMetrics {
	if len(results) == 0 {
		return nil
	}

	metrics := make([]scenarioMetrics, 0, len(results))
	for _, scenario := range results {
		var scenarioMetric scenarioMetrics
		if ef := scenario.Metrics.EmergencyFund; ef != nil {
			scenarioMetric.EmergencyFund = &emergencyFundMetric{
				TargetMonths:           ef.TargetMonths,
				AverageMonthlyExpenses: ef.AverageMonthlyExpenses,
				TargetAmount:           ef.TargetAmount,
				InitialLiquid:          ef.InitialLiquid,
				FundedMonths:           ef.FundedMonths,
				Shortfall:              ef.Shortfall,
				Surplus:                ef.Surplus,
			}
		}
		if len(scenario.Metrics.Optimizations) > 0 {
			summaries := make([]optimizationMetric, 0, len(scenario.Metrics.Optimizations))
			for _, summary := range scenario.Metrics.Optimizations {
				summaries = append(summaries, optimizationMetric{
					TargetName:  summary.TargetName,
					Field:       summary.Field,
					Original:    summary.Original,
					Value:       summary.Value,
					Floor:       summary.Floor,
					MinimumCash: summary.MinimumCash,
					Headroom:    summary.Headroom,
					Iterations:  summary.Iterations,
					Converged:   summary.Converged,
					Notes:       append([]string(nil), summary.Notes...),
				})
			}
			scenarioMetric.Optimizations = summaries
		}
		metrics = append(metrics, scenarioMetric)
	}

	return metrics
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

func coerceBool(value interface{}) bool {
	switch v := value.(type) {
	case bool:
		return v
	case string:
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			return false
		}
		if parsed, err := strconv.ParseBool(trimmed); err == nil {
			return parsed
		}
	case float64:
		return v != 0
	case int:
		return v != 0
	case int64:
		return v != 0
	case json.Number:
		if parsed, err := strconv.ParseFloat(v.String(), 64); err == nil {
			return parsed != 0
		}
	}
	return false
}
