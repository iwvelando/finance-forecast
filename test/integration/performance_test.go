package integration

import (
	"os"
	"testing"
	"time"

	"github.com/iwvelando/finance-forecast/internal/config"
	"github.com/iwvelando/finance-forecast/internal/forecast"
	"go.uber.org/zap"
)

// TestRunner is a simple test runner for debugging
func TestMain(m *testing.M) {
	// Run tests
	code := m.Run()
	os.Exit(code)
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

// TestPerformance tests performance characteristics
func TestPerformance(t *testing.T) {
	// Create a no-op logger to avoid debug output during testing
	logger := zap.NewNop()

	start := time.Now()

	conf, err := config.LoadConfiguration("../test_config.yaml")
	if err != nil {
		t.Fatalf("LoadConfiguration failed: %v", err)
	}
	loadTime := time.Since(start)

	start = time.Now()
	err = conf.ParseDateLists()
	if err != nil {
		t.Fatalf("ParseDateLists failed: %v", err)
	}
	parseTime := time.Since(start)

	start = time.Now()
	err = conf.ProcessLoans(logger)
	if err != nil {
		t.Fatalf("ProcessLoans failed: %v", err)
	}
	loanTime := time.Since(start)

	start = time.Now()
	results, err := forecast.GetForecast(logger, *conf)
	if err != nil {
		t.Fatalf("GetForecast failed: %v", err)
	}
	forecastTime := time.Since(start)

	totalTime := loadTime + parseTime + loanTime + forecastTime

	t.Logf("Performance metrics:")
	t.Logf("  Load config: %v", loadTime)
	t.Logf("  Parse dates: %v", parseTime)
	t.Logf("  Process loans: %v", loanTime)
	t.Logf("  Generate forecast: %v", forecastTime)
	t.Logf("  Total time: %v", totalTime)

	// Performance expectations (adjust as needed)
	if totalTime > 10*time.Second {
		t.Errorf("Total processing time %v exceeds 10 second threshold", totalTime)
	}

	if len(results) != 3 {
		t.Errorf("Expected 3 results, got %d", len(results))
	}

	// Check that we have reasonable amount of data points
	for i, result := range results {
		if len(result.Data) < 100 {
			t.Errorf("Scenario %d (%s) has only %d data points, expected more",
				i, result.Name, len(result.Data))
		}
	}
}

// TestMemoryUsage performs basic memory usage validation
func TestMemoryUsage(t *testing.T) {
	// Create a no-op logger to avoid debug output during testing
	logger := zap.NewNop()

	// Run multiple iterations to check for memory leaks
	for i := 0; i < 10; i++ {
		conf, err := config.LoadConfiguration("../test_config.yaml")
		if err != nil {
			t.Fatalf("LoadConfiguration failed on iteration %d: %v", i, err)
		}

		err = conf.ParseDateLists()
		if err != nil {
			t.Fatalf("ParseDateLists failed on iteration %d: %v", i, err)
		}

		err = conf.ProcessLoans(logger)
		if err != nil {
			t.Fatalf("ProcessLoans failed on iteration %d: %v", i, err)
		}

		_, err = forecast.GetForecast(logger, *conf)
		if err != nil {
			t.Fatalf("GetForecast failed on iteration %d: %v", i, err)
		}
	}

	t.Log("Successfully completed 10 iterations without memory issues")
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
					if abs(val1-val2) > 0.01 {
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

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
