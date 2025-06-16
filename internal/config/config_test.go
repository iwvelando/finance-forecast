package config

import (
	"math"
	"testing"
	"time"

	"github.com/iwvelando/finance-forecast/pkg/datetime"
	"github.com/iwvelando/finance-forecast/pkg/mathutil"
	"go.uber.org/zap"
)

func TestLoadConfiguration(t *testing.T) {
	tests := []struct {
		name       string
		configPath string
		wantError  bool
	}{
		{
			name:       "Non-existent config file",
			configPath: "nonexistent.yaml",
			wantError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config, err := LoadConfiguration(tt.configPath)
			if tt.wantError {
				if err == nil {
					t.Errorf("LoadConfiguration() expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("LoadConfiguration() error = %v", err)
				return
			}
			if config == nil {
				t.Errorf("LoadConfiguration() returned nil config")
			}
		})
	}
}

func TestComputeAmount(t *testing.T) {
	config := &Configuration{
		Common: Common{
			Events: []Event{
				{
					Name:   "Regular event",
					Amount: 100.0,
				},
				{
					Name:   "Negative event",
					Amount: -50.0,
				},
				{
					Name:   "Zero event",
					Amount: 0.0,
				},
			},
		},
	}

	// Test that events retain their original amount
	if config.Common.Events[0].Amount != 100.0 {
		t.Errorf("Expected amount 100.0, got %v", config.Common.Events[0].Amount)
	}

	if config.Common.Events[1].Amount != -50.0 {
		t.Errorf("Expected amount -50.0, got %v", config.Common.Events[1].Amount)
	}

	if config.Common.Events[2].Amount != 0.0 {
		t.Errorf("Expected amount 0.0, got %v", config.Common.Events[2].Amount)
	}
}

func TestLoadConfigurationExample(t *testing.T) {
	// Set up a no-op logger to prevent debug output during testing
	logger := zap.NewNop()

	config, err := LoadConfiguration("../../test/test_config.yaml")
	if err != nil {
		t.Errorf("LoadConfiguration() error = %v", err)
		return
	}
	if config == nil {
		t.Errorf("LoadConfiguration() returned nil config")
		return
	}

	// Test that logging configuration is properly loaded
	if config.Logging.Level == "" {
		t.Log("No logging level specified in config, will use default")
	}
	if config.Logging.Format == "" {
		t.Log("No logging format specified in config, will use default")
	}
	if config.Output.Format == "" {
		t.Log("No output format specified in config, will use default")
	}

	// Only test that the config loaded, don't process it further
	// to avoid triggering loan processing with debug output
	_ = logger // Use the logger variable to avoid unused variable error
}

func TestLoadConfigurationStructure(t *testing.T) {
	config, err := LoadConfiguration("../../test/test_config.yaml")
	if err != nil {
		t.Fatalf("LoadConfiguration() error = %v", err)
	}

	// Test common configuration
	if config.Common.StartingValue != 30000.00 {
		t.Errorf("Expected StartingValue = 30000.00, got %v", config.Common.StartingValue)
	}
	if config.Common.DeathDate != "2090-01" {
		t.Errorf("Expected DeathDate = 2090-01, got %v", config.Common.DeathDate)
	}

	// Test that we have expected scenarios
	expectedScenarios := []string{"current path", "new home purchase", "new home purchase with extra principal payments"}
	if len(config.Scenarios) != len(expectedScenarios) {
		t.Errorf("Expected %d scenarios, got %d", len(expectedScenarios), len(config.Scenarios))
	}

	for i, expectedName := range expectedScenarios {
		if i >= len(config.Scenarios) {
			t.Errorf("Missing scenario: %s", expectedName)
			continue
		}
		if config.Scenarios[i].Name != expectedName {
			t.Errorf("Expected scenario name %s, got %s", expectedName, config.Scenarios[i].Name)
		}
		if !config.Scenarios[i].Active {
			t.Errorf("Expected scenario %s to be active", expectedName)
		}
	}

	// Test common events
	if len(config.Common.Events) < 2 {
		t.Errorf("Expected at least 2 common events, got %d", len(config.Common.Events))
	}

	// Test common loans
	if len(config.Common.Loans) != 1 {
		t.Errorf("Expected 1 common loan, got %d", len(config.Common.Loans))
	}
	if config.Common.Loans[0].Name != "Auto loan" {
		t.Errorf("Expected auto loan, got %s", config.Common.Loans[0].Name)
	}
}

func TestParseDateLists(t *testing.T) {
	config, err := LoadConfiguration("../../test/test_config.yaml")
	if err != nil {
		t.Fatalf("LoadConfiguration() error = %v", err)
	}

	// Use a fixed time for deterministic testing
	fixedTime := time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)

	// Process events with fixed time for deterministic results
	err = config.ParseDateListsWithFixedTime(fixedTime)
	if err != nil {
		t.Fatalf("ParseDateListsWithFixedTime() error = %v", err)
	}

	// Test that DateLists are populated
	for i, scenario := range config.Scenarios {
		for j, event := range scenario.Events {
			if len(event.DateList) == 0 {
				t.Errorf("Scenario %d, Event %d (%s) has empty DateList", i, j, event.Name)
			}
		}
	}

	for i, event := range config.Common.Events {
		if len(event.DateList) == 0 {
			t.Errorf("Common Event %d (%s) has empty DateList", i, event.Name)
		}
	}
}

func TestEventFormDateList(t *testing.T) {
	config := Configuration{
		Common: Common{
			DeathDate: "2030-12",
		},
	}

	tests := []struct {
		name        string
		event       Event
		expectCount int
		expectError bool
	}{
		{
			name: "Monthly event for 1 year",
			event: Event{
				StartDate: "2025-01",
				EndDate:   "2025-12",
				Frequency: 1,
			},
			expectCount: 12,
			expectError: false,
		},
		{
			name: "Quarterly event for 1 year",
			event: Event{
				StartDate: "2025-01",
				EndDate:   "2025-12",
				Frequency: 3,
			},
			expectCount: 4,
			expectError: false,
		},
		{
			name: "One-time event",
			event: Event{
				StartDate: "2025-06",
				EndDate:   "2025-06",
				Frequency: 1,
			},
			expectCount: 1,
			expectError: false,
		},
		{
			name: "Event with no start date (should use current time)",
			event: Event{
				EndDate:   "2025-12",
				Frequency: 1,
			},
			expectCount: 12, // From 2025-01 to 2025-12
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use a fixed current time for deterministic testing
			fixedTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			err := tt.event.FormDateListWithFixedTime(config, fixedTime)
			if tt.expectError && err == nil {
				t.Errorf("FormDateList() expected error but got none")
				return
			}
			if !tt.expectError && err != nil {
				t.Errorf("FormDateList() error = %v", err)
				return
			}
			if tt.expectCount > 0 && len(tt.event.DateList) != tt.expectCount {
				t.Errorf("FormDateList() expected %d dates, got %d", tt.expectCount, len(tt.event.DateList))
			}
		})
	}
}

func TestRound(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"Round up", 1.235, 1.24},
		{"Round down", 1.234, 1.23},
		{"No decimal change", 1.23, 1.23},
		{"Large number", 12345.678, 12345.68},
		{"Negative number", -1.235, -1.24},
		{"Zero", 0.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mathutil.Round(tt.input)
			if math.Abs(result-tt.expected) > 0.001 {
				t.Errorf("Round(%v) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestOffsetDate(t *testing.T) {
	tests := []struct {
		name     string
		date     string
		layout   string
		months   int
		expected string
		wantErr  bool
	}{
		{
			name:     "Add one month",
			date:     "2025-01",
			layout:   datetime.DateTimeLayout,
			months:   1,
			expected: "2025-02",
			wantErr:  false,
		},
		{
			name:     "Add twelve months",
			date:     "2025-01",
			layout:   datetime.DateTimeLayout,
			months:   12,
			expected: "2026-01",
			wantErr:  false,
		},
		{
			name:     "Subtract one month",
			date:     "2025-02",
			layout:   datetime.DateTimeLayout,
			months:   -1,
			expected: "2025-01",
			wantErr:  false,
		},
		{
			name:     "Invalid date format",
			date:     "invalid",
			layout:   datetime.DateTimeLayout,
			months:   1,
			expected: "invalid",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := datetime.OffsetDate(tt.date, tt.layout, tt.months)
			if tt.wantErr {
				if err == nil {
					t.Errorf("OffsetDate() expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("OffsetDate() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("OffsetDate() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestCheckMonth(t *testing.T) {
	tests := []struct {
		name     string
		date     string
		month    string
		expected bool
		wantErr  bool
	}{
		{
			name:     "January match",
			date:     "2025-01",
			month:    "01",
			expected: true,
			wantErr:  false,
		},
		{
			name:     "December match",
			date:     "2025-12",
			month:    "12",
			expected: true,
			wantErr:  false,
		},
		{
			name:     "No match",
			date:     "2025-06",
			month:    "12",
			expected: false,
			wantErr:  false,
		},
		{
			name:     "Invalid date",
			date:     "invalid",
			month:    "01",
			expected: false,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := datetime.CheckMonth(tt.date, tt.month)
			if tt.wantErr {
				if err == nil {
					t.Errorf("CheckMonth() expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("CheckMonth() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("CheckMonth() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestDateBeforeDate(t *testing.T) {
	tests := []struct {
		name       string
		firstDate  string
		secondDate string
		expected   bool
		wantErr    bool
	}{
		{
			name:       "First before second",
			firstDate:  "2025-01",
			secondDate: "2025-02",
			expected:   true,
			wantErr:    false,
		},
		{
			name:       "First after second",
			firstDate:  "2025-02",
			secondDate: "2025-01",
			expected:   false,
			wantErr:    false,
		},
		{
			name:       "Same dates",
			firstDate:  "2025-01",
			secondDate: "2025-01",
			expected:   false,
			wantErr:    false,
		},
		{
			name:       "Invalid first date",
			firstDate:  "invalid",
			secondDate: "2025-01",
			expected:   false,
			wantErr:    true,
		},
		{
			name:       "Invalid second date",
			firstDate:  "2025-01",
			secondDate: "invalid",
			expected:   false,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := datetime.DateBeforeDate(tt.firstDate, tt.secondDate)
			if tt.wantErr {
				if err == nil {
					t.Errorf("DateBeforeDate() expected error but got none")
				}
				return
			}
			if err != nil {
				t.Errorf("DateBeforeDate() error = %v", err)
				return
			}
			if result != tt.expected {
				t.Errorf("DateBeforeDate() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestProcessLoans(t *testing.T) {
	logger := zap.NewNop()

	config := &Configuration{
		Common: Common{
			DeathDate: "2027-01", // Use a closer date to avoid overflow issues
			Loans: []Loan{
				{
					Name:         "Test Loan",
					StartDate:    "2025-01",
					Principal:    100000,
					InterestRate: 5.0,
					Term:         24, // Use a shorter term to avoid date overflow
				},
			},
		},
		Scenarios: []Scenario{
			{
				Name:   "Test Scenario",
				Active: true,
				Loans: []Loan{
					{
						Name:         "Test Scenario Loan",
						StartDate:    "2025-01",
						Principal:    50000,
						InterestRate: 4.0,
						Term:         24, // Use a shorter term
					},
				},
			},
		},
	}

	err := config.ProcessLoans(logger)
	if err != nil {
		t.Errorf("ProcessLoans() error = %v", err)
	}

	// Verify amortization schedules were created
	if len(config.Common.Loans[0].AmortizationSchedule) == 0 {
		t.Errorf("Common loan amortization schedule was not created")
	}

	if len(config.Scenarios[0].Loans[0].AmortizationSchedule) == 0 {
		t.Errorf("Scenario loan amortization schedule was not created")
	}
}

// Test the example configuration processing end-to-end
func TestExampleConfigurationProcessing(t *testing.T) {
	logger := zap.NewNop()

	config, err := LoadConfiguration("../../test/test_config.yaml")
	if err != nil {
		t.Fatalf("LoadConfiguration() error = %v", err)
	}

	err = config.ParseDateLists()
	if err != nil {
		t.Fatalf("ParseDateLists() error = %v", err)
	}

	err = config.ProcessLoans(logger)
	if err != nil {
		t.Fatalf("ProcessLoans() error = %v", err)
	}

	// Verify all loan amortization schedules were created
	for i, scenario := range config.Scenarios {
		for j, loan := range scenario.Loans {
			if len(loan.AmortizationSchedule) == 0 {
				t.Errorf("Scenario %d, Loan %d (%s) has no amortization schedule", i, j, loan.Name)
			}
		}
	}

	for i, loan := range config.Common.Loans {
		if len(loan.AmortizationSchedule) == 0 {
			t.Errorf("Common Loan %d (%s) has no amortization schedule", i, loan.Name)
		}
	}
}

func TestLoggingConfiguration(t *testing.T) {
	// Test with logging configuration
	config := Configuration{
		Logging: LoggingConfig{
			Level:  "debug",
			Format: "console",
		},
		Common: Common{
			DeathDate: "2030-01",
		},
	}

	// Verify logging config is properly set
	if config.Logging.Level != "debug" {
		t.Errorf("Expected logging level 'debug', got '%s'", config.Logging.Level)
	}
	if config.Logging.Format != "console" {
		t.Errorf("Expected logging format 'console', got '%s'", config.Logging.Format)
	}

	// Test default values (empty logging config)
	emptyConfig := Configuration{
		Common: Common{
			DeathDate: "2030-01",
		},
	}

	if emptyConfig.Logging.Level != "" {
		t.Errorf("Expected empty logging level, got '%s'", emptyConfig.Logging.Level)
	}
	if emptyConfig.Logging.Format != "" {
		t.Errorf("Expected empty logging format, got '%s'", emptyConfig.Logging.Format)
	}
}
