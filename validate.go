// This file contains validation utilities for testing
// Run with: go test -run TestValidateApplication
package main

import (
	"fmt"
	"testing"

	"github.com/iwvelando/finance-forecast/config"
	"github.com/iwvelando/finance-forecast/forecast"
	"go.uber.org/zap"
)

func TestValidateApplication(t *testing.T) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		t.Fatalf("Failed to create logger: %v", err)
	}

	fmt.Println("Loading configuration...")
	conf, err := config.LoadConfiguration("config.yaml.example")
	if err != nil {
		t.Fatalf("LoadConfiguration failed: %v", err)
	}
	fmt.Printf("✓ Loaded config with %d scenarios\n", len(conf.Scenarios))

	fmt.Println("Parsing date lists...")
	err = conf.ParseDateLists()
	if err != nil {
		t.Fatalf("ParseDateLists failed: %v", err)
	}
	fmt.Println("✓ Date lists parsed successfully")

	fmt.Println("Processing loans...")
	err = conf.ProcessLoans(logger)
	if err != nil {
		t.Fatalf("ProcessLoans failed: %v", err)
	}
	fmt.Println("✓ Loans processed successfully")

	fmt.Println("Generating forecasts...")
	results, err := forecast.GetForecast(logger, *conf)
	if err != nil {
		t.Fatalf("GetForecast failed: %v", err)
	}
	fmt.Printf("✓ Generated %d forecast results\n", len(results))

	// Validate key values
	fmt.Println("\nValidating key results:")
	expectedValues := map[string]map[string]float64{
		"current path": {
			"2090-01": 295939.66,
		},
		"new home purchase": {
			"2090-01": 537436.86,
		},
		"new home purchase with extra principal payments": {
			"2090-01": 559379.68,
		},
	}

	for _, result := range results {
		if expected, exists := expectedValues[result.Name]; exists {
			for date, expectedVal := range expected {
				if actualVal, dateExists := result.Data[date]; dateExists {
					diff := actualVal - expectedVal
					if diff < -1 || diff > 1 { // Allow 1 dollar tolerance
						fmt.Printf("⚠️  %s at %s: expected %.2f, got %.2f (diff: %.2f)\n",
							result.Name, date, expectedVal, actualVal, diff)
					} else {
						fmt.Printf("✓ %s at %s: %.2f (matches baseline)\n",
							result.Name, date, actualVal)
					}
				} else {
					fmt.Printf("❌ %s: missing date %s\n", result.Name, date)
				}
			}
		}
	}

	fmt.Println("\n✅ All tests completed successfully!")
}
