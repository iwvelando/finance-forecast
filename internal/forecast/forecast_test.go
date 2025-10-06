package forecast

import (
	"math"
	"strings"
	"testing"
	"time"

	"github.com/iwvelando/finance-forecast/internal/config"
	"github.com/iwvelando/finance-forecast/pkg/adapters"
	"github.com/iwvelando/finance-forecast/pkg/datetime"
	"github.com/iwvelando/finance-forecast/pkg/finance"
	"go.uber.org/zap"
)

func TestEventProcessing(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	forecastEngine := finance.NewForecastEngine(logger)

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
		datetime.MustParseTime(config.DateTimeLayout, "2025-06"),
		datetime.MustParseTime(config.DateTimeLayout, "2025-07"),
		datetime.MustParseTime(config.DateTimeLayout, "2025-08"),
	}
	events[1].DateList = []time.Time{
		datetime.MustParseTime(config.DateTimeLayout, "2025-06"),
		datetime.MustParseTime(config.DateTimeLayout, "2025-09"),
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
			// Convert events using the adapter
			financeEvents := adapters.EventsToFinanceEvents(events)

			amount, err := forecastEngine.ProcessMonthlyChanges(tt.date, financeEvents, nil, config.DateTimeLayout)
			if err != nil {
				t.Errorf("ProcessMonthlyChanges() error = %v", err)
			}
			if amount != tt.expected {
				t.Errorf("ProcessMonthlyChanges() = %.2f, expected %.2f", amount, tt.expected)
			}
		})
	}
}

func TestLoanProcessing(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	forecastEngine := finance.NewForecastEngine(logger)

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
			// Convert loans using the adapter
			financeLoans := adapters.LoansToFinanceLoans(loans)

			amount, err := forecastEngine.ProcessMonthlyChanges(tt.date, nil, financeLoans, config.DateTimeLayout)
			if err != nil {
				t.Errorf("ProcessMonthlyChanges() error = %v", err)
			}
			if amount != tt.expected {
				t.Errorf("ProcessMonthlyChanges() = %.2f, expected %.2f", amount, tt.expected)
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
	fixedTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	conf.Common.Events[0].DateList = []time.Time{
		datetime.MustParseTime(config.DateTimeLayout, "2025-06"),
		datetime.MustParseTime(config.DateTimeLayout, "2025-07"),
		datetime.MustParseTime(config.DateTimeLayout, "2025-08"),
	}
	conf.Scenarios[0].Events[0].DateList = []time.Time{
		datetime.MustParseTime(config.DateTimeLayout, "2025-06"),
		datetime.MustParseTime(config.DateTimeLayout, "2025-07"),
	}

	results, err := GetForecastWithFixedTime(logger, conf, fixedTime)
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

	// Check starting value at the fixed time
	startDate := fixedTime.Format(config.DateTimeLayout)
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

	// Use a fixed time for deterministic testing
	fixedTime := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	results, err := GetForecastWithFixedTime(logger, conf, fixedTime)
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

// Test with realistic data similar to the example config
func TestGetForecastRealistic(t *testing.T) {
	// Use a no-op logger to suppress all debug output during testing
	logger := zap.NewNop()

	// Load and process the test configuration
	conf, err := config.LoadConfiguration("../../test/test_config.yaml")
	if err != nil {
		t.Fatalf("LoadConfiguration() error = %v", err)
	}

	// Use a fixed time for deterministic testing
	fixedTime := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	// Parse date lists with fixed time
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

	err = conf.ProcessLoans(logger)
	if err != nil {
		t.Fatalf("ProcessLoans() error = %v", err)
	}

	results, err := GetForecastWithFixedTime(logger, *conf, fixedTime)
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

		// Verify starting value using fixed date rather than time.Now()
		startDate := fixedTime.Format(config.DateTimeLayout)
		expectedStart := conf.Common.StartingValue
		for _, inv := range conf.Common.Investments {
			expectedStart += inv.StartingValue
		}
		for _, inv := range conf.Scenarios[i].Investments {
			expectedStart += inv.StartingValue
		}
		if math.Abs(results[i].Data[startDate]-expectedStart) > 1e-6 {
			t.Errorf("Scenario %s: expected starting value %.2f, got %.2f",
				expected, expectedStart, results[i].Data[startDate])
		}
	}
}

func TestGetForecastWithConfiguredStartDate(t *testing.T) {
	logger := zap.NewNop()

	// Create a test configuration with a specific start date
	conf := config.Configuration{
		StartDate: "2025-06",
		Common: config.Common{
			StartingValue: 15000.0,
			DeathDate:     "2025-12",
			Events: []config.Event{
				{
					Name:     "Test Income",
					Amount:   1000.0,
					DateList: []time.Time{datetime.MustParseTime(config.DateTimeLayout, "2025-07")},
				},
			},
		},
		Scenarios: []config.Scenario{
			{
				Name:   "Test Scenario",
				Active: true,
				Events: []config.Event{},
				Loans:  []config.Loan{},
			},
		},
	}

	results, err := GetForecast(logger, conf)
	if err != nil {
		t.Fatalf("GetForecast() error = %v", err)
	}

	// Should have 1 result
	if len(results) != 1 {
		t.Errorf("Expected 1 forecast result, got %d", len(results))
	}

	result := results[0]
	// Verify starting value is set at the configured start date
	if result.Data["2025-06"] != 15000.0 {
		t.Errorf("Expected starting value 15000.0 at configured start date, got %.2f", result.Data["2025-06"])
	}

	// Verify we have data for subsequent months
	if _, exists := result.Data["2025-07"]; !exists {
		t.Errorf("Expected data for month after start date")
	}
}

func TestGetForecastWithInvestments(t *testing.T) {
	logger := zap.NewNop()

	conf := config.Configuration{
		Common: config.Common{
			StartingValue: 1000,
			DeathDate:     "2025-08",
			Investments: []config.Investment{
				{
					Name:             "Common Fund",
					StartingValue:    500,
					AnnualReturnRate: 12,
				},
			},
		},
		Scenarios: []config.Scenario{
			{
				Name:   "Investment Scenario",
				Active: true,
				Investments: []config.Investment{
					{
						Name:             "Scenario Fund",
						StartingValue:    200,
						AnnualReturnRate: 12,
						Contributions: []config.Event{
							{Amount: 100},
						},
					},
				},
			},
		},
	}

	// Assign date lists for contributions
	conf.Scenarios[0].Investments[0].Contributions[0].DateList = []time.Time{
		datetime.MustParseTime(config.DateTimeLayout, "2025-07"),
		datetime.MustParseTime(config.DateTimeLayout, "2025-08"),
	}

	fixedTime := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	results, err := GetForecastWithFixedTime(logger, conf, fixedTime)
	if err != nil {
		t.Fatalf("GetForecastWithFixedTime() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 forecast result, got %d", len(results))
	}

	result := results[0]
	startDate := fixedTime.Format(config.DateTimeLayout)
	if val := result.Data[startDate]; math.Abs(val-1700) > 1e-9 {
		t.Fatalf("starting value = %.2f, want 1700", val)
	}

	firstMonth := "2025-07"
	expected := 1700 + 108.0 // 500 * 1% = 5 growth; scenario investment adds 100 contribution + 3 growth
	if val := result.Data[firstMonth]; math.Abs(val-expected) > 1e-6 {
		t.Errorf("balance for %s = %.2f, want %.2f", firstMonth, val, expected)
	}
}

func TestGetForecastWithInvestmentsContributionReducingIncome(t *testing.T) {
	logger := zap.NewNop()

	conf := config.Configuration{
		Common: config.Common{
			StartingValue: 1000,
			DeathDate:     "2025-08",
		},
		Scenarios: []config.Scenario{
			{
				Name:   "Investment Scenario",
				Active: true,
				Investments: []config.Investment{
					{
						Name:                  "Traditional 401k",
						StartingValue:         0,
						AnnualReturnRate:      0,
						ContributionsFromCash: true,
						Contributions: []config.Event{
							{Amount: 100},
						},
					},
				},
			},
		},
	}

	conf.Scenarios[0].Investments[0].Contributions[0].DateList = []time.Time{
		datetime.MustParseTime(config.DateTimeLayout, "2025-07"),
		datetime.MustParseTime(config.DateTimeLayout, "2025-08"),
	}

	fixedTime := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)

	results, err := GetForecastWithFixedTime(logger, conf, fixedTime)
	if err != nil {
		t.Fatalf("GetForecastWithFixedTime() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 forecast result, got %d", len(results))
	}

	result := results[0]
	startDate := fixedTime.Format(config.DateTimeLayout)
	if val := result.Data[startDate]; math.Abs(val-1000) > 1e-9 {
		t.Fatalf("starting value = %.2f, want 1000", val)
	}

	if val := result.Data["2025-07"]; math.Abs(val-1000) > 1e-9 {
		t.Errorf("balance for 2025-07 = %.2f, want 1000 (contributions offset by income)", val)
	}

	if val := result.Data["2025-08"]; math.Abs(val-1000) > 1e-9 {
		t.Errorf("balance for 2025-08 = %.2f, want 1000 (contributions offset by income)", val)
	}

	notes := result.Notes["2025-07"]
	found := false
	for _, note := range notes {
		if strings.Contains(note, "contribution (reduces cash balance) +100.00") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected note describing contribution sourced from income, got %v", notes)
	}
}

func TestInvestmentNotesShowGrossGrowth(t *testing.T) {
	logger := zap.NewNop()

	conf := config.Configuration{
		Common: config.Common{
			StartingValue: 0,
			DeathDate:     "2025-02",
		},
		Scenarios: []config.Scenario{
			{
				Name:   "Growth Scenario",
				Active: true,
				Investments: []config.Investment{
					{
						Name:             "Brokerage",
						StartingValue:    1000,
						AnnualReturnRate: 12,
						TaxRate:          10,
					},
				},
			},
		},
	}

	fixedTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	results, err := GetForecastWithFixedTime(logger, conf, fixedTime)
	if err != nil {
		t.Fatalf("GetForecastWithFixedTime() error = %v", err)
	}

	if len(results) != 1 {
		t.Fatalf("expected 1 forecast result, got %d", len(results))
	}

	notes := results[0].Notes["2025-02"]
	if len(notes) == 0 {
		t.Fatalf("expected notes for 2025-02, got none")
	}

	var sawGrowth, sawTax bool
	for _, note := range notes {
		if strings.Contains(note, "growth +10.00") {
			sawGrowth = true
		}
		if strings.Contains(note, "tax 1.00") {
			sawTax = true
		}
	}

	if !sawGrowth {
		t.Errorf("expected growth note to show gross amount, notes: %v", notes)
	}
	if !sawTax {
		t.Errorf("expected tax note to show withheld amount, notes: %v", notes)
	}
}
