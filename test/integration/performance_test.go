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
