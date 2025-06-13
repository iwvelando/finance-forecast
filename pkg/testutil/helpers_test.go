package testutil

import (
	"fmt"
	"testing"

	"github.com/iwvelando/finance-forecast/internal/forecast"
)

func TestFindScenario(t *testing.T) {
	// Create test data
	results := []forecast.Forecast{
		{
			Name: "Scenario A",
			Data: map[string]float64{
				"2025-01": 1000.00,
			},
		},
		{
			Name: "Scenario B",
			Data: map[string]float64{
				"2025-01": 2000.00,
			},
		},
		{
			Name: "Another Scenario",
			Data: map[string]float64{
				"2025-01": 3000.00,
			},
		},
	}

	tests := []struct {
		name         string
		searchName   string
		expectFound  bool
		expectedData float64
	}{
		{
			name:         "Find existing scenario A",
			searchName:   "Scenario A",
			expectFound:  true,
			expectedData: 1000.00,
		},
		{
			name:         "Find existing scenario B",
			searchName:   "Scenario B",
			expectFound:  true,
			expectedData: 2000.00,
		},
		{
			name:         "Find scenario with longer name",
			searchName:   "Another Scenario",
			expectFound:  true,
			expectedData: 3000.00,
		},
		{
			name:        "Search for non-existent scenario",
			searchName:  "Non-existent",
			expectFound: false,
		},
		{
			name:        "Empty search name",
			searchName:  "",
			expectFound: false,
		},
		{
			name:        "Case sensitive search",
			searchName:  "scenario a", // lowercase
			expectFound: false,
		},
		{
			name:        "Partial name match",
			searchName:  "Scenario", // partial
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FindScenario(results, tt.searchName)

			if tt.expectFound {
				if result == nil {
					t.Errorf("FindScenario() expected to find scenario '%s' but got nil", tt.searchName)
					return
				}
				if result.Name != tt.searchName {
					t.Errorf("FindScenario() returned scenario with name '%s', expected '%s'",
						result.Name, tt.searchName)
				}
				if result.Data["2025-01"] != tt.expectedData {
					t.Errorf("FindScenario() returned scenario with data %v, expected %v",
						result.Data["2025-01"], tt.expectedData)
				}
			} else {
				if result != nil {
					t.Errorf("FindScenario() expected nil for scenario '%s' but got result with name '%s'",
						tt.searchName, result.Name)
				}
			}
		})
	}
}

func TestFindScenarioEmptyResults(t *testing.T) {
	// Test with empty results slice
	results := []forecast.Forecast{}

	result := FindScenario(results, "Any Scenario")
	if result != nil {
		t.Errorf("FindScenario() with empty results should return nil, got %v", result)
	}
}

func TestFindScenarioNilResults(t *testing.T) {
	// Test with nil results slice
	var results []forecast.Forecast = nil

	result := FindScenario(results, "Any Scenario")
	if result != nil {
		t.Errorf("FindScenario() with nil results should return nil, got %v", result)
	}
}

func TestFindScenarioReturnsPointer(t *testing.T) {
	// Test that FindScenario returns a pointer to the actual element
	results := []forecast.Forecast{
		{
			Name: "Test Scenario",
			Data: map[string]float64{
				"2025-01": 1000.00,
			},
		},
	}

	found := FindScenario(results, "Test Scenario")
	if found == nil {
		t.Fatalf("FindScenario() returned nil")
	}

	// Verify we get the same pointer
	if &results[0] != found {
		t.Errorf("FindScenario() should return pointer to original element")
	}

	// Modify through the returned pointer and verify original is modified
	found.Data["2025-02"] = 2000.00

	if results[0].Data["2025-02"] != 2000.00 {
		t.Errorf("Modifying through returned pointer should modify original")
	}
}

func TestFindScenarioWithDuplicateNames(t *testing.T) {
	// Test behavior with duplicate names (should return first match)
	results := []forecast.Forecast{
		{
			Name: "Duplicate",
			Data: map[string]float64{
				"2025-01": 1000.00,
			},
		},
		{
			Name: "Duplicate",
			Data: map[string]float64{
				"2025-01": 2000.00,
			},
		},
	}

	found := FindScenario(results, "Duplicate")
	if found == nil {
		t.Fatalf("FindScenario() returned nil")
	}

	// Should return the first match
	if found.Data["2025-01"] != 1000.00 {
		t.Errorf("FindScenario() should return first match, got data %v", found.Data["2025-01"])
	}

	// Verify it's actually the first element
	if &results[0] != found {
		t.Errorf("FindScenario() should return pointer to first matching element")
	}
}

func TestFindScenarioWithSpecialCharacters(t *testing.T) {
	// Test with scenario names containing special characters
	results := []forecast.Forecast{
		{
			Name: "Scenario with spaces",
			Data: map[string]float64{"2025-01": 1000.00},
		},
		{
			Name: "Scenario-with-hyphens",
			Data: map[string]float64{"2025-01": 2000.00},
		},
		{
			Name: "Scenario_with_underscores",
			Data: map[string]float64{"2025-01": 3000.00},
		},
		{
			Name: "Scenario (with parentheses)",
			Data: map[string]float64{"2025-01": 4000.00},
		},
		{
			Name: "Scenario #1",
			Data: map[string]float64{"2025-01": 5000.00},
		},
	}

	tests := []struct {
		name         string
		searchName   string
		expectedData float64
	}{
		{"Spaces", "Scenario with spaces", 1000.00},
		{"Hyphens", "Scenario-with-hyphens", 2000.00},
		{"Underscores", "Scenario_with_underscores", 3000.00},
		{"Parentheses", "Scenario (with parentheses)", 4000.00},
		{"Hash", "Scenario #1", 5000.00},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			found := FindScenario(results, tt.searchName)
			if found == nil {
				t.Errorf("FindScenario() should find scenario '%s'", tt.searchName)
				return
			}
			if found.Data["2025-01"] != tt.expectedData {
				t.Errorf("FindScenario() returned wrong data for '%s': got %v, expected %v",
					tt.searchName, found.Data["2025-01"], tt.expectedData)
			}
		})
	}
}

func TestFindScenarioPerformance(t *testing.T) {
	// Test with a reasonably large slice to ensure performance is acceptable
	const numScenarios = 1000
	results := make([]forecast.Forecast, numScenarios)

	for i := 0; i < numScenarios; i++ {
		results[i] = forecast.Forecast{
			Name: fmt.Sprintf("Scenario %d", i),
			Data: map[string]float64{"2025-01": float64(i * 100)},
		}
	}

	// Find scenario in the middle
	targetName := "Scenario 500"
	found := FindScenario(results, targetName)

	if found == nil {
		t.Errorf("FindScenario() should find '%s' in large slice", targetName)
		return
	}

	if found.Name != targetName {
		t.Errorf("FindScenario() returned wrong scenario: got '%s', expected '%s'",
			found.Name, targetName)
	}

	if found.Data["2025-01"] != 50000.00 {
		t.Errorf("FindScenario() returned wrong data: got %v, expected 50000.00",
			found.Data["2025-01"])
	}
}
