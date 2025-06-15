package validation

import (
	"testing"
)

func TestValidateDeathDate(t *testing.T) {
	tests := []struct {
		name        string
		loanName    string
		startDate   string
		deathDate   string
		termMonths  int
		expectWarn  bool
		expectError bool
	}{
		{
			name:        "Loan matures before death",
			loanName:    "Short Loan",
			startDate:   "2025-01",
			deathDate:   "2030-01",
			termMonths:  36, // 3 years
			expectWarn:  false,
			expectError: false,
		},
		{
			name:        "Loan matures after death",
			loanName:    "Long Loan",
			startDate:   "2025-01",
			deathDate:   "2028-01",
			termMonths:  60, // 5 years, extends to 2030
			expectWarn:  false,
			expectError: false,
		},
		{
			name:        "Loan matures exactly at death",
			loanName:    "Exact Loan",
			startDate:   "2025-01",
			deathDate:   "2030-01",
			termMonths:  60, // 5 years, exactly at death
			expectWarn:  false,
			expectError: false,
		},
		{
			name:        "Invalid start date",
			loanName:    "Invalid Loan",
			startDate:   "invalid-date",
			deathDate:   "2030-01",
			termMonths:  60,
			expectWarn:  false,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warning, err := ValidateDeathDate(tt.loanName, tt.startDate, tt.deathDate, tt.termMonths)

			if tt.expectError {
				if err == nil {
					t.Errorf("ValidateDeathDate() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ValidateDeathDate() unexpected error = %v", err)
				return
			}

			hasWarning := warning != ""
			if hasWarning != tt.expectWarn {
				t.Errorf("ValidateDeathDate() warning = %t, expected %t", hasWarning, tt.expectWarn)
			}

			if hasWarning {
				t.Logf("Warning: %s", warning)
			}
		})
	}
}

func TestValidateEventDates(t *testing.T) {
	tests := []struct {
		name            string
		eventName       string
		startDate       string
		endDate         string
		deathDate       string
		expectWarnCount int
	}{
		{
			name:            "Event entirely before death",
			eventName:       "Normal Event",
			startDate:       "2025-01",
			endDate:         "2028-01",
			deathDate:       "2030-01",
			expectWarnCount: 0,
		},
		{
			name:            "Event starts at death date",
			eventName:       "Death Start Event",
			startDate:       "2030-01",
			endDate:         "2030-06",
			deathDate:       "2030-01",
			expectWarnCount: 2, // Both start at death and end after death
		},
		{
			name:            "Event starts after death",
			eventName:       "After Death Event",
			startDate:       "2031-01",
			endDate:         "2032-01",
			deathDate:       "2030-01",
			expectWarnCount: 2, // Both start after death and end after death
		},
		{
			name:            "Event ends after death",
			eventName:       "Long Event",
			startDate:       "2025-01",
			endDate:         "2035-01",
			deathDate:       "2030-01",
			expectWarnCount: 1, // Only end after death
		},
		{
			name:            "Event with no end date",
			eventName:       "Indefinite Event",
			startDate:       "2025-01",
			endDate:         "",
			deathDate:       "2030-01",
			expectWarnCount: 0, // No end date warnings
		},
		{
			name:            "Event ending exactly at death",
			eventName:       "Exact End Event",
			startDate:       "2025-01",
			endDate:         "2030-01",
			deathDate:       "2030-01",
			expectWarnCount: 0, // Ending exactly at death is okay
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := ValidateEventDates(tt.eventName, tt.startDate, tt.endDate, tt.deathDate)

			if len(warnings) != tt.expectWarnCount {
				t.Errorf("ValidateEventDates() returned %d warnings, expected %d",
					len(warnings), tt.expectWarnCount)
			}

			for _, warning := range warnings {
				t.Logf("Warning: %s", warning)
			}
		})
	}
}

func TestConfigValidator_ValidateAll(t *testing.T) {
	tests := []struct {
		name            string
		validator       ConfigValidator
		expectWarnCount int
	}{
		{
			name: "Valid configuration",
			validator: ConfigValidator{
				Common: CommonConfig{
					DeathDate: "2030-01",
					Events: []EventConfig{
						{
							Name:      "Normal Event",
							StartDate: "2025-01",
							EndDate:   "2028-01",
						},
					},
					Loans: []LoanConfig{
						{
							Name:      "Normal Loan",
							StartDate: "2025-01",
							Term:      36, // 3 years
						},
					},
				},
				Scenarios: []ScenarioConfig{
					{
						Name:   "Active Scenario",
						Active: true,
						Events: []EventConfig{
							{
								Name:      "Scenario Event",
								StartDate: "2025-01",
								EndDate:   "2027-01",
							},
						},
						Loans: []LoanConfig{
							{
								Name:      "Scenario Loan",
								StartDate: "2025-01",
								Term:      24, // 2 years
							},
						},
					},
				},
			},
			expectWarnCount: 0,
		},
		{
			name: "Configuration with warnings",
			validator: ConfigValidator{
				Common: CommonConfig{
					DeathDate: "2028-01",
					Events: []EventConfig{
						{
							Name:      "Event After Death",
							StartDate: "2030-01",
							EndDate:   "2031-01",
						},
					},
					Loans: []LoanConfig{
						{
							Name:      "Long Loan",
							StartDate: "2025-01",
							Term:      60, // 5 years, extends past death
						},
					},
				},
				Scenarios: []ScenarioConfig{
					{
						Name:   "Active Scenario",
						Active: true,
						Events: []EventConfig{
							{
								Name:      "Late Event",
								StartDate: "2029-01",
								EndDate:   "2030-01",
							},
						},
					},
					{
						Name:   "Inactive Scenario",
						Active: false,
						Events: []EventConfig{
							{
								Name:      "Should Be Ignored",
								StartDate: "2030-01",
								EndDate:   "2031-01",
							},
						},
					},
				},
			},
			expectWarnCount: 4, // Event after death (2), scenario event after death (2), but inactive scenario ignored
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := tt.validator.ValidateAll()

			if len(warnings) != tt.expectWarnCount {
				t.Errorf("ValidateAll() returned %d warnings, expected %d",
					len(warnings), tt.expectWarnCount)
			}

			for i, warning := range warnings {
				t.Logf("Warning %d: %s", i+1, warning)
			}
		})
	}
}

func TestConfigValidator_InactiveScenarios(t *testing.T) {
	// Test that inactive scenarios are properly ignored
	validator := ConfigValidator{
		Common: CommonConfig{
			DeathDate: "2030-01",
		},
		Scenarios: []ScenarioConfig{
			{
				Name:   "Active Scenario",
				Active: true,
				Events: []EventConfig{
					{
						Name:      "Event After Death",
						StartDate: "2031-01",
						EndDate:   "2032-01",
					},
				},
				Loans: []LoanConfig{
					{
						Name:      "Long Loan",
						StartDate: "2025-01",
						Term:      120, // 10 years
					},
				},
			},
			{
				Name:   "Inactive Scenario",
				Active: false,
				Events: []EventConfig{
					{
						Name:      "Should Be Ignored",
						StartDate: "2031-01",
						EndDate:   "2032-01",
					},
				},
				Loans: []LoanConfig{
					{
						Name:      "Should Be Ignored",
						StartDate: "2025-01",
						Term:      120,
					},
				},
			},
		},
	}

	warnings := validator.ValidateAll()

	// Should only get warnings from active scenario
	// Active scenario: event after death (2 warnings) = 2 total
	expectedWarnings := 2
	if len(warnings) != expectedWarnings {
		t.Errorf("Expected %d warnings for active scenario only, got %d", expectedWarnings, len(warnings))
	}

	// Verify none of the warnings mention the inactive scenario
	for _, warning := range warnings {
		if warning == "Should Be Ignored" {
			t.Errorf("Found warning from inactive scenario: %s", warning)
		}
	}
}

func TestConfigValidator_EmptyConfiguration(t *testing.T) {
	// Test with minimal configuration
	validator := ConfigValidator{
		Common: CommonConfig{
			DeathDate: "2030-01",
		},
		Scenarios: []ScenarioConfig{},
	}

	warnings := validator.ValidateAll()

	// Should have no warnings for empty but valid configuration
	if len(warnings) != 0 {
		t.Errorf("Expected no warnings for empty configuration, got %d", len(warnings))
	}
}

func TestValidateEventDatesEdgeCases(t *testing.T) {
	// Test edge cases in event date validation
	tests := []struct {
		name      string
		startDate string
		endDate   string
		deathDate string
		expected  int
	}{
		{
			name:      "Start exactly at death, no end",
			startDate: "2030-01",
			endDate:   "",
			deathDate: "2030-01",
			expected:  1, // Start at death warning only
		},
		{
			name:      "Start before death, end exactly at death",
			startDate: "2029-01",
			endDate:   "2030-01",
			deathDate: "2030-01",
			expected:  0, // No warnings - ending at death is okay
		},
		{
			name:      "Start before death, end after death",
			startDate: "2029-01",
			endDate:   "2030-02",
			deathDate: "2030-01",
			expected:  1, // End after death warning only
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := ValidateEventDates("Test Event", tt.startDate, tt.endDate, tt.deathDate)
			if len(warnings) != tt.expected {
				t.Errorf("Expected %d warnings, got %d: %v", tt.expected, len(warnings), warnings)
			}
		})
	}
}
