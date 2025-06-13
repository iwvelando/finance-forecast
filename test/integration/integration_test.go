package integration

import (
	"bufio"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/iwvelando/finance-forecast/internal/config"
	"github.com/iwvelando/finance-forecast/internal/forecast"
	"github.com/iwvelando/finance-forecast/pkg/output"
	"github.com/iwvelando/finance-forecast/pkg/testutil"
	"go.uber.org/zap"
)

// TestMainIntegrationBaseline tests that the application produces the same results
// as our baseline captured from the current working version
func TestMainIntegrationBaseline(t *testing.T) {
	// Create a no-op logger to avoid debug output during testing
	logger := zap.NewNop()

	// Load and process the example configuration exactly as main() does
	conf, err := config.LoadConfiguration("../test_config.yaml")
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

		if math.Abs(actualVal-check.expectedVal) > check.tolerance {
			t.Errorf("Scenario '%s' at '%s': expected %.2f, got %.2f",
				check.scenario, check.date, check.expectedVal, actualVal)
		}
	}
}

// TestCSVOutputFormat tests that CSV output matches our baseline format
func TestCSVOutputFormat(t *testing.T) {
	// Create a no-op logger to avoid debug output during testing
	logger := zap.NewNop()

	conf, err := config.LoadConfiguration("../test_config.yaml")
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
	baselineFile, err := os.Open("../baseline/baseline_output.csv")
	if err != nil {
		t.Fatalf("Could not open baseline CSV file: %v", err)
	}
	defer func() {
		_ = baselineFile.Close()
	}()

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
	// Create a no-op logger to avoid debug output during testing
	logger := zap.NewNop()

	conf, err := config.LoadConfiguration("../test_config.yaml")
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

	// Test that PrettyFormat doesn't crash
	defer func() {
		if r := recover(); r != nil {
			t.Errorf("PrettyFormat() panicked: %v", r)
		}
	}()

	// Redirect stdout to /dev/null to suppress output
	originalStdout := os.Stdout
	devNull, err := os.OpenFile("/dev/null", os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("Failed to open /dev/null: %v", err)
	}
	os.Stdout = devNull

	// Call PrettyFormat with redirected stdout
	output.PrettyFormat(results)

	// Restore stdout and close /dev/null
	os.Stdout = originalStdout
	_ = devNull.Close()

	// We can't verify the content, but the test passes if there's no panic
	t.Log("PrettyFormat completed without panic")
}

// TestCsvFormat tests the CSV format function
func TestCsvFormat(t *testing.T) {
	// Create a no-op logger to avoid debug output during testing
	logger := zap.NewNop()

	conf, err := config.LoadConfiguration("../test_config.yaml")
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

	// Redirect stdout to /dev/null to suppress output
	originalStdout := os.Stdout
	devNull, err := os.OpenFile("/dev/null", os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("Failed to open /dev/null: %v", err)
	}
	os.Stdout = devNull

	// Call CsvFormat with redirected stdout
	output.CsvFormat(results)

	// Restore stdout and close /dev/null
	os.Stdout = originalStdout
	_ = devNull.Close()

	// We can't verify the content, but the test passes if there's no panic
	t.Log("CsvFormat completed without panic")
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
	conservativeResult := testutil.FindScenario(results, "Conservative")
	aggressiveResult := testutil.FindScenario(results, "Aggressive")

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

// TestBasicFunctionality tests basic functionality works
func TestBasicFunctionality(t *testing.T) {
	// Create a no-op logger to avoid debug output during testing
	logger := zap.NewNop()

	// Test basic config loading
	conf, err := config.LoadConfiguration("../test_config.yaml")
	if err != nil {
		t.Fatalf("LoadConfiguration failed: %v", err)
	}

	// Test basic parsing
	err = conf.ParseDateLists()
	if err != nil {
		t.Fatalf("ParseDateLists failed: %v", err)
	}

	// Test loan processing
	err = conf.ProcessLoans(logger)
	if err != nil {
		t.Fatalf("ProcessLoans failed: %v", err)
	}

	// Test forecast generation
	results, err := forecast.GetForecast(logger, *conf)
	if err != nil {
		t.Fatalf("GetForecast failed: %v", err)
	}

	if len(results) == 0 {
		t.Fatalf("Expected forecast results but got none")
	}

	t.Logf("Successfully generated %d forecast results", len(results))
}

// TestDataConsistency validates that multiple runs produce identical results
func TestDataConsistency(t *testing.T) {
	// Create a no-op logger to avoid debug output during testing
	logger := zap.NewNop()

	// Run the same configuration multiple times
	var firstResults []forecast.Forecast

	for run := 0; run < 3; run++ {
		conf, err := config.LoadConfiguration("../test_config.yaml")
		if err != nil {
			t.Fatalf("LoadConfiguration failed on run %d: %v", run, err)
		}

		err = conf.ParseDateLists()
		if err != nil {
			t.Fatalf("ParseDateLists failed on run %d: %v", run, err)
		}

		err = conf.ProcessLoans(logger)
		if err != nil {
			t.Fatalf("ProcessLoans failed on run %d: %v", run, err)
		}

		results, err := forecast.GetForecast(logger, *conf)
		if err != nil {
			t.Fatalf("GetForecast failed on run %d: %v", run, err)
		}

		if run == 0 {
			firstResults = results
			continue
		}

		// Compare with first run
		if len(results) != len(firstResults) {
			t.Errorf("Run %d: got %d results, expected %d", run, len(results), len(firstResults))
			continue
		}

		for i, result := range results {
			firstResult := firstResults[i]

			if result.Name != firstResult.Name {
				t.Errorf("Run %d, scenario %d: name mismatch %s != %s",
					run, i, result.Name, firstResult.Name)
			}

			if len(result.Data) != len(firstResult.Data) {
				t.Errorf("Run %d, scenario %d: data length mismatch %d != %d",
					run, i, len(result.Data), len(firstResult.Data))
				continue
			}

			// Check a few key data points
			checkDates := []string{"2090-01", "2050-01", "2030-01"}
			for _, date := range checkDates {
				val1, exists1 := result.Data[date]
				val2, exists2 := firstResult.Data[date]

				if exists1 != exists2 {
					t.Errorf("Run %d, scenario %d, date %s: existence mismatch", run, i, date)
					continue
				}

				if exists1 && exists2 {
					if math.Abs(val1-val2) > 0.01 {
						t.Errorf("Run %d, scenario %d, date %s: value mismatch %.2f != %.2f",
							run, i, date, val1, val2)
					}
				}
			}
		}
	}

	t.Log("Data consistency verified across multiple runs")
}

// TestConfigurationVariations tests different configuration variations
func TestConfigurationVariations(t *testing.T) {
	// Create a no-op logger to avoid debug output during testing
	logger := zap.NewNop()

	variations := []struct {
		name            string
		modifyConfig    func(*config.Configuration)
		expectError     bool
		expectScenarios int
	}{
		{
			name: "Baseline config",
			modifyConfig: func(c *config.Configuration) {
				// No changes
			},
			expectError:     false,
			expectScenarios: 3,
		},
		{
			name: "Shorter death date",
			modifyConfig: func(c *config.Configuration) {
				c.Common.DeathDate = "2055-01" // Must be after events and loans end (some go to 2050)
			},
			expectError:     false,
			expectScenarios: 3,
		},
		{
			name: "Higher starting value",
			modifyConfig: func(c *config.Configuration) {
				c.Common.StartingValue = 50000.0
			},
			expectError:     false,
			expectScenarios: 3,
		},
		{
			name: "Disable one scenario",
			modifyConfig: func(c *config.Configuration) {
				c.Scenarios[1].Active = false
			},
			expectError:     false,
			expectScenarios: 2,
		},
	}

	for _, variation := range variations {
		t.Run(variation.name, func(t *testing.T) {
			conf, err := config.LoadConfiguration("../test_config.yaml")
			if err != nil {
				t.Fatalf("LoadConfiguration failed: %v", err)
			}

			// Apply variation
			variation.modifyConfig(conf)

			err = conf.ParseDateLists()
			if variation.expectError && err == nil {
				t.Errorf("Expected error in ParseDateLists but got none")
				return
			}
			if !variation.expectError && err != nil {
				t.Errorf("Unexpected error in ParseDateLists: %v", err)
				return
			}

			if variation.expectError {
				return // Skip remaining tests for error cases
			}

			err = conf.ProcessLoans(logger)
			if err != nil {
				t.Errorf("ProcessLoans failed: %v", err)
				return
			}

			results, err := forecast.GetForecast(logger, *conf)
			if err != nil {
				t.Errorf("GetForecast failed: %v", err)
				return
			}

			if len(results) != variation.expectScenarios {
				t.Errorf("Expected %d scenarios, got %d", variation.expectScenarios, len(results))
			}
		})
	}
}
