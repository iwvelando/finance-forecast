package forecast

import (
	"testing"
	"time"

	"github.com/iwvelando/finance-forecast/internal/config"
	"go.uber.org/zap"
)

func TestHandleEvents(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	// Create test events with date lists
	events := []config.Event{
		{
			Name:   "Monthly Income",
			Amount: 1000.0,
		},
		{
			Name:   "Quarterly Bill",
			Amount: -300.0,
		},
	}

	// Manually set date lists for testing
	events[0].DateList = []time.Time{
		mustParseTime("2025-06"),
		mustParseTime("2025-07"),
		mustParseTime("2025-08"),
	}
	events[1].DateList = []time.Time{
		mustParseTime("2025-06"),
		mustParseTime("2025-09"),
	}

	tests := []struct {
		name     string
		date     string
		expected float64
	}{
		{
			name:     "Date with both events",
			date:     "2025-06",
			expected: 700.0, // 1000 - 300
		},
		{
			name:     "Date with only monthly income",
			date:     "2025-07",
			expected: 1000.0,
		},
		{
			name:     "Date with no events",
			date:     "2025-05",
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, err := HandleEvents(logger, tt.date, events, config.DateTimeLayout)
			if err != nil {
				t.Errorf("HandleEvents() error = %v", err)
			}
			if amount != tt.expected {
				t.Errorf("HandleEvents() = %.2f, expected %.2f", amount, tt.expected)
			}
		})
	}
}

func TestHandleLoans(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	// Create test loans with amortization schedules
	loans := []config.Loan{
		{
			Name: "Test Loan 1",
			AmortizationSchedule: map[string]config.Payment{
				"2025-06": {Payment: 1500.0},
				"2025-07": {Payment: 1500.0},
			},
		},
		{
			Name: "Test Loan 2",
			AmortizationSchedule: map[string]config.Payment{
				"2025-06": {Payment: 800.0},
			},
		},
	}

	tests := []struct {
		name     string
		date     string
		expected float64
	}{
		{
			name:     "Date with both loan payments",
			date:     "2025-06",
			expected: -2300.0, // -(1500 + 800)
		},
		{
			name:     "Date with one loan payment",
			date:     "2025-07",
			expected: -1500.0, // -1500
		},
		{
			name:     "Date with no loan payments",
			date:     "2025-08",
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount := HandleLoans(logger, tt.date, loans)
			if amount != tt.expected {
				t.Errorf("HandleLoans() = %.2f, expected %.2f", amount, tt.expected)
			}
		})
	}
}

func TestGetForecast(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	// Create a simple test configuration
	conf := config.Configuration{
		Common: config.Common{
			StartingValue: 10000.0,
			DeathDate:     "2025-08",
			Events: []config.Event{
				{
					Name:   "Monthly Income",
					Amount: 1000.0,
				},
			},
			Loans: []config.Loan{},
		},
		Scenarios: []config.Scenario{
			{
				Name:   "Test Scenario",
				Active: true,
				Events: []config.Event{
					{
						Name:   "Scenario Income",
						Amount: 500.0,
					},
				},
				Loans: []config.Loan{},
			},
		},
	}

	// Set up date lists manually for testing
	conf.Common.Events[0].DateList = []time.Time{
		mustParseTime("2025-06"),
		mustParseTime("2025-07"),
		mustParseTime("2025-08"),
	}
	conf.Scenarios[0].Events[0].DateList = []time.Time{
		mustParseTime("2025-06"),
		mustParseTime("2025-07"),
	}

	results, err := GetForecast(logger, conf)
	if err != nil {
		t.Fatalf("GetForecast() error = %v", err)
	}

	// Verify we got one result for the active scenario
	if len(results) != 1 {
		t.Errorf("Expected 1 forecast result, got %d", len(results))
	}

	result := results[0]
	if result.Name != "Test Scenario" {
		t.Errorf("Expected scenario name 'Test Scenario', got '%s'", result.Name)
	}

	// Verify we have data points
	if len(result.Data) == 0 {
		t.Errorf("Expected forecast data, got empty map")
	}

	// Check starting value
	startDate := time.Now().Format(config.DateTimeLayout)
	if result.Data[startDate] != 10000.0 {
		t.Errorf("Expected starting value 10000.0, got %.2f", result.Data[startDate])
	}
}

func TestGetForecastInactiveScenario(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	conf := config.Configuration{
		Common: config.Common{
			StartingValue: 10000.0,
			DeathDate:     "2025-08",
		},
		Scenarios: []config.Scenario{
			{
				Name:   "Active Scenario",
				Active: true,
			},
			{
				Name:   "Inactive Scenario",
				Active: false,
			},
		},
	}

	results, err := GetForecast(logger, conf)
	if err != nil {
		t.Fatalf("GetForecast() error = %v", err)
	}

	// Should only get results for active scenarios
	if len(results) != 1 {
		t.Errorf("Expected 1 result for active scenario, got %d", len(results))
	}

	if results[0].Name != "Active Scenario" {
		t.Errorf("Expected 'Active Scenario', got '%s'", results[0].Name)
	}
}

// Helper function to parse time for testing
func mustParseTime(dateStr string) time.Time {
	t, err := time.Parse(config.DateTimeLayout, dateStr)
	if err != nil {
		panic(err)
	}
	return t
}

// Test with realistic data similar to the example config
func TestGetForecastRealistic(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	// Load and process the example configuration
	conf, err := config.LoadConfiguration("../../config.yaml.example")
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

	results, err := GetForecast(logger, *conf)
	if err != nil {
		t.Fatalf("GetForecast() error = %v", err)
	}

	// Should have 3 active scenarios
	if len(results) != 3 {
		t.Errorf("Expected 3 forecast results, got %d", len(results))
	}

	expectedScenarios := []string{"current path", "new home purchase", "new home purchase with extra principal payments"}
	for i, expected := range expectedScenarios {
		if i >= len(results) {
			t.Errorf("Missing scenario: %s", expected)
			continue
		}
		if results[i].Name != expected {
			t.Errorf("Expected scenario %s, got %s", expected, results[i].Name)
		}

		// Verify each scenario has data
		if len(results[i].Data) == 0 {
			t.Errorf("Scenario %s has no forecast data", expected)
		}

		// Verify starting value
		startDate := time.Now().Format(config.DateTimeLayout)
		if results[i].Data[startDate] != 30000.0 {
			t.Errorf("Scenario %s: expected starting value 30000.0, got %.2f",
				expected, results[i].Data[startDate])
		}
	}
}
