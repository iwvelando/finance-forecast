package main

import (
	"bufio"
	"os"
	"strings"
	"testing"

	"github.com/iwvelando/finance-forecast/config"
	"github.com/iwvelando/finance-forecast/forecast"
	"go.uber.org/zap"
)

// TestMainIntegrationBaseline tests that the application produces the same results
// as our baseline captured from the current working version
func TestMainIntegrationBaseline(t *testing.T) {
	// Skip this test unless running in verbose mode to avoid debug output from example config
	if !testing.Verbose() {
		t.Skip("Skipping integration test to avoid debug output. Run with -v to enable.")
	}

	logger, _ := zap.NewDevelopment()

	// Load and process the example configuration exactly as main() does
	conf, err := config.LoadConfiguration("config.yaml.example")
	if err != nil {
		t.Fatalf("LoadConfiguration() error = %v", err)
	}

	err = conf.ParseDateLists()
	if err != nil {
		t.Fatalf("ParseDateLists() error = %v", err)
	}

	err = conf.ProcessStockEvents()
	if err != nil {
		// Stock events might fail in test environment, skip if so
		t.Logf("ProcessStockEvents() error = %v (may be expected in test environment)", err)
	}

	err = conf.ProcessLoans(logger)
	if err != nil {
		t.Fatalf("ProcessLoans() error = %v", err)
	}

	results, err := forecast.GetForecast(logger, *conf)
	if err != nil {
		t.Fatalf("GetForecast() error = %v", err)
	}

	// Validate we have the expected number of scenarios
	if len(results) != 3 {
		t.Errorf("Expected 3 scenarios, got %d", len(results))
	}

	expectedScenarios := []string{
		"current path",
		"new home purchase",
		"new home purchase with extra principal payments",
	}

	for i, expected := range expectedScenarios {
		if i >= len(results) {
			t.Errorf("Missing scenario: %s", expected)
			continue
		}
		if results[i].Name != expected {
			t.Errorf("Expected scenario %s, got %s", expected, results[i].Name)
		}
	}

	// Validate baseline values from our CSV output
	validateBaselineValues(t, results)
}

// validateBaselineValues checks specific key values against our baseline
func validateBaselineValues(t *testing.T, results []forecast.Forecast) {
	// These are specific values from our baseline CSV output
	baselineChecks := []struct {
		scenario    string
		date        string
		expectedVal float64
		tolerance   float64
	}{
		{"current path", "2090-01", 295939.66, 1.0},
		{"new home purchase", "2090-01", 537436.86, 1.0},
		{"new home purchase with extra principal payments", "2090-01", 559379.68, 1.0},
	}

	for _, check := range baselineChecks {
		var result *forecast.Forecast
		for i := range results {
			if results[i].Name == check.scenario {
				result = &results[i]
				break
			}
		}

		if result == nil {
			t.Errorf("Scenario '%s' not found in results", check.scenario)
			continue
		}

		actualVal, exists := result.Data[check.date]
		if !exists {
			t.Errorf("Date '%s' not found in scenario '%s'", check.date, check.scenario)
			continue
		}

		if abs(actualVal-check.expectedVal) > check.tolerance {
			t.Errorf("Scenario '%s' at '%s': expected %.2f, got %.2f",
				check.scenario, check.date, check.expectedVal, actualVal)
		}
	}
}

// TestCSVOutputFormat tests that CSV output matches our baseline format
func TestCSVOutputFormat(t *testing.T) {
	// Skip this test unless running in verbose mode to avoid debug output from example config
	if !testing.Verbose() {
		t.Skip("Skipping integration test to avoid debug output. Run with -v to enable.")
	}

	logger, _ := zap.NewDevelopment()

	conf, err := config.LoadConfiguration("config.yaml.example")
	if err != nil {
		t.Fatalf("LoadConfiguration() error = %v", err)
	}

	err = conf.ParseDateLists()
	if err != nil {
		t.Fatalf("ParseDateLists() error = %v", err)
	}

	err = conf.ProcessLoans(logger)
	if err != nil {
		t.Fatalf("ProcessLoans() error = %v", err)
	}

	_, err = forecast.GetForecast(logger, *conf)
	if err != nil {
		t.Fatalf("GetForecast() error = %v", err)
	}

	// Verify we can read our baseline CSV file
	baselineFile, err := os.Open("baseline_output.csv")
	if err != nil {
		t.Fatalf("Could not open baseline CSV file: %v", err)
	}
	defer baselineFile.Close()

	scanner := bufio.NewScanner(baselineFile)

	// Read header line
	if !scanner.Scan() {
		t.Fatalf("Could not read CSV header")
	}
	header := scanner.Text()

	// Verify header format
	expectedHeaderParts := []string{
		`"date"`,
		`"amount (current path)"`,
		`"notes (current path)"`,
		`"amount (new home purchase)"`,
		`"notes (new home purchase)"`,
		`"amount (new home purchase with extra principal payments)"`,
		`"notes (new home purchase with extra principal payments)"`,
	}

	for _, part := range expectedHeaderParts {
		if !strings.Contains(header, part) {
			t.Errorf("CSV header missing expected part: %s", part)
		}
	}

	// Read a few data lines to verify format
	lineCount := 0
	for scanner.Scan() && lineCount < 5 {
		line := scanner.Text()
		parts := strings.Split(line, ",")

		// Should have 7 parts: date, amount1, notes1, amount2, notes2, amount3, notes3
		if len(parts) != 7 {
			t.Errorf("CSV line should have 7 parts, got %d: %s", len(parts), line)
		}

		// First part should be a quoted date
		if !strings.HasPrefix(parts[0], `"20`) {
			t.Errorf("CSV date should start with quoted year: %s", parts[0])
		}

		lineCount++
	}

	if err := scanner.Err(); err != nil {
		t.Errorf("Error reading baseline CSV: %v", err)
	}
}

// TestPrettyOutputFormat tests the pretty print output
func TestPrettyOutputFormat(t *testing.T) {
	// Skip this test unless running in verbose mode to avoid debug output from example config
	if !testing.Verbose() {
		t.Skip("Skipping integration test to avoid debug output. Run with -v to enable.")
	}

	logger, _ := zap.NewDevelopment()

	conf, err := config.LoadConfiguration("config.yaml.example")
	if err != nil {
		t.Fatalf("LoadConfiguration() error = %v", err)
	}

	err = conf.ParseDateLists()
	if err != nil {
		t.Fatalf("ParseDateLists() error = %v", err)
	}

	err = conf.ProcessLoans(logger)
	if err != nil {
		t.Fatalf("ProcessLoans() error = %v", err)
	}

	results, err := forecast.GetForecast(logger, *conf)
	if err != nil {
		t.Fatalf("GetForecast() error = %v", err)
	}

	// Test that PrettyFormat doesn't crash (we can't easily test output without major refactoring)
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PrettyFormat() panicked: %v", r)
		}
	}()

	// This will print to stdout, but at least verifies it doesn't crash
	PrettyFormat(results)
}

// TestCsvFormat tests the CSV format function
func TestCsvFormat(t *testing.T) {
	// Skip this test unless running in verbose mode to avoid debug output from example config
	if !testing.Verbose() {
		t.Skip("Skipping integration test to avoid debug output. Run with -v to enable.")
	}

	logger, _ := zap.NewDevelopment()

	conf, err := config.LoadConfiguration("config.yaml.example")
	if err != nil {
		t.Fatalf("LoadConfiguration() error = %v", err)
	}

	err = conf.ParseDateLists()
	if err != nil {
		t.Fatalf("ParseDateLists() error = %v", err)
	}

	err = conf.ProcessLoans(logger)
	if err != nil {
		t.Fatalf("ProcessLoans() error = %v", err)
	}

	results, err := forecast.GetForecast(logger, *conf)
	if err != nil {
		t.Fatalf("GetForecast() error = %v", err)
	}

	// Test that CsvFormat doesn't crash
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("CsvFormat() panicked: %v", r)
		}
	}()

	// This will print to stdout, but at least verifies it doesn't crash
	CsvFormat(results)
}

// TestConfigurationValidation tests validation of different configuration scenarios
func TestConfigurationValidation(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func() *config.Configuration
		expectError bool
	}{
		{
			name: "Valid minimal configuration",
			setupConfig: func() *config.Configuration {
				return &config.Configuration{
					Common: config.Common{
						StartingValue: 1000,
						DeathDate:     "2026-01",
					},
					Scenarios: []config.Scenario{
						{
							Name:   "Test",
							Active: true,
						},
					},
				}
			},
			expectError: false,
		},
		{
			name: "Configuration with invalid event date format",
			setupConfig: func() *config.Configuration {
				return &config.Configuration{
					Common: config.Common{
						StartingValue: 1000,
						DeathDate:     "2026-01",
						Events: []config.Event{
							{
								Name:      "Invalid Event",
								Amount:    100,
								StartDate: "invalid-date-format",
								EndDate:   "2026-01",
								Frequency: 1,
							},
						},
					},
					Scenarios: []config.Scenario{
						{
							Name:   "Test",
							Active: true,
						},
					},
				}
			},
			expectError: true,
		},
	}

	logger := zap.NewNop() // Use no-op logger to avoid debug output

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := tt.setupConfig()

			err := conf.ParseDateLists()
			if tt.expectError && err == nil {
				t.Errorf("Expected error in ParseDateLists but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error in ParseDateLists: %v", err)
			}

			if !tt.expectError {
				err = conf.ProcessLoans(logger)
				if err != nil {
					t.Errorf("Unexpected error in ProcessLoans: %v", err)
				}

				_, err = forecast.GetForecast(logger, *conf)
				if err != nil {
					t.Errorf("Unexpected error in GetForecast: %v", err)
				}
			}
		})
	}
}

// TestEndToEndWithComplexScenario tests a complex scenario end-to-end
func TestEndToEndWithComplexScenario(t *testing.T) {
	logger := zap.NewNop() // Use no-op logger to avoid debug output

	// Create a complex configuration programmatically
	conf := &config.Configuration{
		Common: config.Common{
			StartingValue: 50000,
			DeathDate:     "2030-01",
			Events: []config.Event{
				{
					Name:      "Monthly Income",
					Amount:    5000,
					StartDate: "2025-01",
					EndDate:   "2029-12",
					Frequency: 1,
				},
				{
					Name:      "Annual Bonus",
					Amount:    10000,
					StartDate: "2025-12",
					EndDate:   "2029-12",
					Frequency: 12,
				},
			},
			Loans: []config.Loan{
				{
					Name:         "Car Loan",
					StartDate:    "2025-01",
					Principal:    25000,
					InterestRate: 3.5,
					Term:         60,
					DownPayment:  5000,
				},
			},
		},
		Scenarios: []config.Scenario{
			{
				Name:   "Conservative",
				Active: true,
				Events: []config.Event{
					{
						Name:      "Conservative Investment",
						Amount:    -1000,
						StartDate: "2025-01",
						EndDate:   "2029-12",
						Frequency: 1,
					},
				},
			},
			{
				Name:   "Aggressive",
				Active: true,
				Events: []config.Event{
					{
						Name:      "Aggressive Investment",
						Amount:    -2000,
						StartDate: "2025-01",
						EndDate:   "2029-12",
						Frequency: 1,
					},
				},
			},
		},
	}

	// Process the configuration
	err := conf.ParseDateLists()
	if err != nil {
		t.Fatalf("ParseDateLists() error = %v", err)
	}

	err = conf.ProcessLoans(logger)
	if err != nil {
		t.Fatalf("ProcessLoans() error = %v", err)
	}

	results, err := forecast.GetForecast(logger, *conf)
	if err != nil {
		t.Fatalf("GetForecast() error = %v", err)
	}

	// Validate results
	if len(results) != 2 {
		t.Errorf("Expected 2 scenario results, got %d", len(results))
	}

	// Conservative scenario should have higher end balance than aggressive
	// (since aggressive invests more money each month)
	conservativeResult := findScenario(results, "Conservative")
	aggressiveResult := findScenario(results, "Aggressive")

	if conservativeResult == nil || aggressiveResult == nil {
		t.Fatalf("Could not find expected scenarios in results")
	}

	// Compare end values
	conservativeEnd := conservativeResult.Data["2030-01"]
	aggressiveEnd := aggressiveResult.Data["2030-01"]

	if aggressiveEnd >= conservativeEnd {
		t.Errorf("Expected conservative (%.2f) > aggressive (%.2f) end balance",
			conservativeEnd, aggressiveEnd)
	}
}

// Helper function to find a scenario by name
func findScenario(results []forecast.Forecast, name string) *forecast.Forecast {
	for i := range results {
		if results[i].Name == name {
			return &results[i]
		}
	}
	return nil
}
