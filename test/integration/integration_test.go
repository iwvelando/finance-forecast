package integration

import (
	"encoding/csv"
	"io"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/iwvelando/finance-forecast/internal/config"
	"github.com/iwvelando/finance-forecast/internal/forecast"
	"github.com/iwvelando/finance-forecast/pkg/output"
	"github.com/iwvelando/finance-forecast/pkg/testutil"
	"go.uber.org/zap"
)

// TestDeterministicComponentBaselines verifies that the deterministic test configurations
// remain stable and that the combined configuration equals the sum of its components.
func TestDeterministicComponentBaselines(t *testing.T) {
	type scenarioExpectation struct {
		name   string
		total  float64
		liquid float64
	}

	type baselineCase struct {
		name         string
		configPath   string
		expectations []scenarioExpectation
		accumulate   bool
	}

	cases := []baselineCase{
		{
			name:       "cash flows",
			configPath: "../test_cash_flows_config.yaml",
			expectations: []scenarioExpectation{
				{name: "deterministic cash flow baseline", total: 17150.00, liquid: 17150.00},
			},
			accumulate: true,
		},
		{
			name:       "single loan",
			configPath: "../test_single_loan_config.yaml",
			expectations: []scenarioExpectation{
				{name: "deterministic loan baseline", total: 9672.03, liquid: 9672.03},
			},
			accumulate: true,
		},
		{
			name:       "pretax investment",
			configPath: "../test_pretax_investment_config.yaml",
			expectations: []scenarioExpectation{
				{name: "deterministic pretax investment baseline", total: 17889.48, liquid: 3000.00},
			},
			accumulate: true,
		},
		{
			name:       "tax-free investment",
			configPath: "../test_aftertax_taxfree_config.yaml",
			expectations: []scenarioExpectation{
				{name: "deterministic tax-free investment baseline", total: 11426.02, liquid: 2218.78},
			},
			accumulate: true,
		},
		{
			name:       "taxable investment",
			configPath: "../test_aftertax_taxed_config.yaml",
			expectations: []scenarioExpectation{
				{name: "deterministic taxable investment baseline", total: 16467.96, liquid: 5900.00},
			},
			accumulate: true,
		},
		{
			name:       "combined",
			configPath: "../test_combined_config.yaml",
			expectations: []scenarioExpectation{
				{name: "deterministic combined baseline", total: 72605.48, liquid: 37940.81},
			},
			accumulate: false,
		},
	}

	const tolerance = 0.02
	fixedTime := time.Date(2024, 12, 15, 0, 0, 0, 0, time.UTC)
	logger := zap.NewNop()

	totalSum := 0.0
	liquidSum := 0.0

	for _, c := range cases {
		conf, err := config.LoadConfiguration(c.configPath)
		if err != nil {
			t.Fatalf("%s: LoadConfiguration() error = %v", c.name, err)
		}

		if err := conf.ParseDateListsWithFixedTime(fixedTime); err != nil {
			t.Fatalf("%s: ParseDateListsWithFixedTime() error = %v", c.name, err)
		}

		if err := conf.ProcessLoans(logger); err != nil {
			t.Fatalf("%s: ProcessLoans() error = %v", c.name, err)
		}

		results, err := forecast.GetForecastWithFixedTime(logger, *conf, fixedTime)
		if err != nil {
			t.Fatalf("%s: GetForecastWithFixedTime() error = %v", c.name, err)
		}

		if len(results) != len(c.expectations) {
			t.Fatalf("%s: expected %d scenarios, got %d", c.name, len(c.expectations), len(results))
		}

		finalMonth := conf.Common.DeathDate

		for _, expect := range c.expectations {
			scenario := testutil.FindScenario(results, expect.name)
			if scenario == nil {
				t.Fatalf("%s: scenario %q not found", c.name, expect.name)
			}

			totalVal, ok := scenario.Data[finalMonth]
			if !ok {
				t.Fatalf("%s: scenario %q missing total for %s", c.name, expect.name, finalMonth)
			}

			liquidVal, ok := scenario.Liquid[finalMonth]
			if !ok {
				t.Fatalf("%s: scenario %q missing liquid for %s", c.name, expect.name, finalMonth)
			}

			if math.Abs(totalVal-expect.total) > tolerance {
				t.Errorf("%s: scenario %q total mismatch for %s: expected %.2f, got %.2f",
					c.name, expect.name, finalMonth, expect.total, totalVal)
			}

			if math.Abs(liquidVal-expect.liquid) > tolerance {
				t.Errorf("%s: scenario %q liquid mismatch for %s: expected %.2f, got %.2f",
					c.name, expect.name, finalMonth, expect.liquid, liquidVal)
			}

			if c.accumulate {
				totalSum += totalVal
				liquidSum += liquidVal
			} else {
				if math.Abs(totalVal-totalSum) > tolerance {
					t.Errorf("%s: combined total %.2f differs from component sum %.2f", c.name, totalVal, totalSum)
				}
				if math.Abs(liquidVal-liquidSum) > tolerance {
					t.Errorf("%s: combined liquid %.2f differs from component sum %.2f", c.name, liquidVal, liquidSum)
				}
			}
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

	// Use a fixed time for deterministic testing
	fixedTime := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	// Parse date lists with fixed time
	err = conf.ParseDateListsWithFixedTime(fixedTime)
	if err != nil {
		t.Fatalf("ParseDateLists() error = %v", err)
	}

	err = conf.ProcessLoans(logger)
	if err != nil {
		t.Fatalf("ProcessLoans() error = %v", err)
	}

	// Get forecast with fixed time
	results, err := forecast.GetForecastWithFixedTime(logger, *conf, fixedTime)
	if err != nil {
		t.Fatalf("GetForecast() error = %v", err)
	}

	// Verify the results contain expected scenarios
	if len(results) != 3 {
		t.Errorf("Expected 3 scenarios in results, got %d", len(results))
	}

	// Verify we can read our baseline CSV file
	baselineFile, err := os.Open("../baseline/baseline_output.csv")
	if err != nil {
		t.Fatalf("Could not open baseline CSV file: %v", err)
	}
	defer func() {
		_ = baselineFile.Close()
	}()

	reader := csv.NewReader(baselineFile)
	headerRecord, err := reader.Read()
	if err != nil {
		t.Fatalf("Could not read CSV header: %v", err)
	}

	expectedHeaderParts := []string{
		"date",
		"liquid (current path)",
		"total (current path)",
		"notes (current path)",
		"liquid (new home purchase)",
		"total (new home purchase)",
		"notes (new home purchase)",
		"liquid (new home purchase with extra principal payments)",
		"total (new home purchase with extra principal payments)",
		"notes (new home purchase with extra principal payments)",
	}

	if len(headerRecord) != len(expectedHeaderParts) {
		t.Fatalf("CSV header should have %d fields, got %d", len(expectedHeaderParts), len(headerRecord))
	}

	for i, part := range expectedHeaderParts {
		if headerRecord[i] != part {
			t.Errorf("CSV header mismatch at column %d: expected %q, got %q", i, part, headerRecord[i])
		}
	}

	for lineCount := 0; lineCount < 5; lineCount++ {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Errorf("Error reading baseline CSV: %v", err)
			break
		}

		if len(record) != len(expectedHeaderParts) {
			t.Errorf("CSV record should have %d fields, got %d: %v", len(expectedHeaderParts), len(record), record)
			continue
		}

		if !strings.HasPrefix(record[0], "20") {
			t.Errorf("CSV date should start with year prefix: %s", record[0])
		}
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

	// Use a fixed time for deterministic testing
	fixedTime := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	// Parse date lists with fixed time
	err = conf.ParseDateListsWithFixedTime(fixedTime)
	if err != nil {
		t.Fatalf("ParseDateLists() error = %v", err)
	}

	err = conf.ProcessLoans(logger)
	if err != nil {
		t.Fatalf("ProcessLoans() error = %v", err)
	}

	results, err := forecast.GetForecastWithFixedTime(logger, *conf, fixedTime)
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

	// Use a fixed time for deterministic testing
	fixedTime := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	// Parse date lists with fixed time
	err = conf.ParseDateListsWithFixedTime(fixedTime)
	if err != nil {
		t.Fatalf("ParseDateLists() error = %v", err)
	}

	err = conf.ProcessLoans(logger)
	if err != nil {
		t.Fatalf("ProcessLoans() error = %v", err)
	}

	results, err := forecast.GetForecastWithFixedTime(logger, *conf, fixedTime)
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
				Events: []config.Event{{
					Name:      "Conservative Investment",
					Amount:    -1000,
					StartDate: "2025-01",
					EndDate:   "2029-12",
					Frequency: 1,
				},
					{
						Name:      "Return on Conservative Investment",
						Amount:    1500, // Positive return that outweighs the negative investment
						StartDate: "2025-01",
						EndDate:   "2029-12",
						Frequency: 12, // Annual return
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

	// Use a fixed time for deterministic testing
	fixedTime := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	// Process the configuration with fixed time
	// Set up date lists manually for testing
	for i, scenario := range conf.Scenarios {
		for j := range scenario.Events {
			err := conf.Scenarios[i].Events[j].FormDateListWithFixedTime(*conf, fixedTime)
			if err != nil {
				t.Fatalf("FormDateListWithFixedTime() error = %v", err)
			}
		}
	}
	for i := range conf.Common.Events {
		err := conf.Common.Events[i].FormDateListWithFixedTime(*conf, fixedTime)
		if err != nil {
			t.Fatalf("FormDateListWithFixedTime() error = %v", err)
		}
	}

	err := conf.ProcessLoans(logger)
	if err != nil {
		t.Fatalf("ProcessLoans() error = %v", err)
	}

	results, err := forecast.GetForecastWithFixedTime(logger, *conf, fixedTime)
	if err != nil {
		t.Fatalf("GetForecast() error = %v", err)
	}

	// Validate results
	if len(results) != 2 {
		t.Errorf("Expected 2 scenario results, got %d", len(results))
	}

	// Conservative scenario should have higher end balance than aggressive
	// (since aggressive invests more money each month - negative amounts are investments)
	conservativeResult := testutil.FindScenario(results, "Conservative")
	aggressiveResult := testutil.FindScenario(results, "Aggressive")

	if conservativeResult == nil || aggressiveResult == nil {
		t.Fatalf("Could not find expected scenarios in results")
	}

	// Compare end values
	conservativeEnd := conservativeResult.Data["2030-01"]
	aggressiveEnd := aggressiveResult.Data["2030-01"]

	// Since investments are negative amounts, the aggressive scenario with -2000 monthly
	// will end up with a lower balance than conservative with only -1000 monthly
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

	// Use a fixed time for deterministic testing
	fixedTime := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	// Test basic parsing with fixed time
	err = conf.ParseDateListsWithFixedTime(fixedTime)
	if err != nil {
		t.Fatalf("ParseDateLists failed: %v", err)
	}

	// Test loan processing
	err = conf.ProcessLoans(logger)
	if err != nil {
		t.Fatalf("ProcessLoans failed: %v", err)
	}

	// Test forecast generation with fixed time
	results, err := forecast.GetForecastWithFixedTime(logger, *conf, fixedTime)
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

	// Use a fixed time for deterministic testing
	fixedTime := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	// Run the same configuration multiple times
	var firstResults []forecast.Forecast

	for run := 0; run < 3; run++ {
		conf, err := config.LoadConfiguration("../test_config.yaml")
		if err != nil {
			t.Fatalf("LoadConfiguration failed on run %d: %v", run, err)
		}

		err = conf.ParseDateListsWithFixedTime(fixedTime)
		if err != nil {
			t.Fatalf("ParseDateLists failed on run %d: %v", run, err)
		}

		err = conf.ProcessLoans(logger)
		if err != nil {
			t.Fatalf("ProcessLoans failed on run %d: %v", run, err)
		}

		results, err := forecast.GetForecastWithFixedTime(logger, *conf, fixedTime)
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

	// Use a fixed time for deterministic testing
	fixedTime := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

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

			// Use fixed time for parsing date lists
			err = conf.ParseDateListsWithFixedTime(fixedTime)
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

			results, err := forecast.GetForecastWithFixedTime(logger, *conf, fixedTime)
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

// TestCSVBaselineConsistency ensures that our forecasting is consistent with the baseline data
func TestCSVBaselineConsistency(t *testing.T) {
	// Create a no-op logger to avoid debug output during testing
	logger := zap.NewNop()

	conf, err := config.LoadConfiguration("../test_config.yaml")
	if err != nil {
		t.Fatalf("LoadConfiguration() error = %v", err)
	}

	// Use a fixed time for deterministic testing - this MUST match the time used to generate the baseline
	fixedTime := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	// Parse date lists with fixed time
	err = conf.ParseDateListsWithFixedTime(fixedTime)
	if err != nil {
		t.Fatalf("ParseDateLists() error = %v", err)
	}

	err = conf.ProcessLoans(logger)
	if err != nil {
		t.Fatalf("ProcessLoans() error = %v", err)
	}

	results, err := forecast.GetForecastWithFixedTime(logger, *conf, fixedTime)
	if err != nil {
		t.Fatalf("GetForecast() error = %v", err)
	}

	// Read baseline data for comparison
	baselineFile, err := os.Open("../baseline/baseline_output.csv")
	if err != nil {
		t.Logf("Could not open baseline CSV file: %v", err)
		t.Logf("Skipping baseline comparison - this is expected if baseline hasn't been generated yet")
		return
	}
	defer func() {
		if err := baselineFile.Close(); err != nil {
			t.Logf("Failed to close baseline file: %v", err)
		}
	}()

	reader := csv.NewReader(baselineFile)
	if _, err := reader.Read(); err != nil {
		t.Fatalf("Could not read CSV header: %v", err)
	}

	// Create maps of date -> value for each scenario in the baseline
	baselineData := make(map[string]map[string]float64)
	scenarioNames := []string{
		"current path",
		"new home purchase",
		"new home purchase with extra principal payments",
	}

	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Errorf("Error reading baseline CSV: %v", err)
			return
		}
		if len(record) < 10 {
			continue
		}

		date := record[0]
		if baselineData[date] == nil {
			baselineData[date] = make(map[string]float64)
		}

		for idx, scenario := range scenarioNames {
			fieldIndex := 2 + idx*3
			if fieldIndex >= len(record) {
				continue
			}
			amountStr := record[fieldIndex]
			if amountStr == "" {
				continue
			}
			val, err := strconv.ParseFloat(amountStr, 64)
			if err != nil {
				t.Logf("Warning: could not parse amount '%s' for scenario %s at %s: %v", amountStr, scenario, date, err)
				continue
			}
			baselineData[date][scenario] = val
		}
	}

	// Compare generated results with baseline
	// Check a few key dates rather than every single one
	checkDates := []string{"2026-01", "2030-01", "2050-01", "2090-01"}
	for _, date := range checkDates {
		for _, result := range results {
			currentVal := result.Data[date]
			baselineVal, exists := baselineData[date][result.Name]

			if !exists {
				// This might be a new scenario or date not in baseline
				t.Logf("Date %s, scenario %s: No baseline data found", date, result.Name)
				continue
			}

			// Allow a small tolerance for floating point differences
			tolerance := 0.01
			diff := math.Abs(currentVal - baselineVal)
			if diff > tolerance {
				t.Errorf("Date %s, scenario %s: Current %.2f differs from baseline %.2f by %.2f",
					date, result.Name, currentVal, baselineVal, diff)
			}
		}
	}
}
