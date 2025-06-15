package configprocessor

import (
	"testing"
)

func TestNewProcessor(t *testing.T) {
	processor := NewProcessor()
	if processor == nil {
		t.Error("NewProcessor() returned nil")
	}
}

func TestProcessor_ValidateConfiguration(t *testing.T) {
	processor := NewProcessor()

	tests := []struct {
		name             string
		deathDate        string
		commonEvents     []EventInfo
		commonLoans      []LoanInfo
		scenarios        []ScenarioInfo
		expectedWarnings int
	}{
		{
			name:      "Valid configuration",
			deathDate: "2030-01",
			commonEvents: []EventInfo{
				{
					Name:      "Valid Event",
					StartDate: "2025-01",
					EndDate:   "2028-01",
				},
			},
			commonLoans: []LoanInfo{
				{
					Name:      "Valid Loan",
					StartDate: "2025-01",
					Term:      36, // 3 years
				},
			},
			scenarios: []ScenarioInfo{
				{
					Name:   "Valid Scenario",
					Active: true,
					Events: []EventInfo{
						{
							Name:      "Scenario Event",
							StartDate: "2025-06",
							EndDate:   "2027-01",
						},
					},
					Loans: []LoanInfo{
						{
							Name:      "Scenario Loan",
							StartDate: "2025-01",
							Term:      24, // 2 years
						},
					},
				},
			},
			expectedWarnings: 0,
		},
		{
			name:      "Configuration with warnings",
			deathDate: "2028-01",
			commonEvents: []EventInfo{
				{
					Name:      "Event After Death",
					StartDate: "2030-01",
					EndDate:   "2031-01",
				},
			},
			commonLoans: []LoanInfo{
				{
					Name:      "Long Loan",
					StartDate: "2025-01",
					Term:      60, // 5 years, extends past death
				},
			},
			scenarios: []ScenarioInfo{
				{
					Name:   "Active Scenario",
					Active: true,
					Events: []EventInfo{
						{
							Name:      "Late Event",
							StartDate: "2029-01",
							EndDate:   "2030-01",
						},
					},
					Loans: []LoanInfo{
						{
							Name:      "Late Loan",
							StartDate: "2026-01",
							Term:      60, // 5 years, past death
						},
					},
				},
				{
					Name:   "Inactive Scenario",
					Active: false,
					Events: []EventInfo{
						{
							Name:      "Should Be Ignored",
							StartDate: "2030-01",
							EndDate:   "2031-01",
						},
					},
					Loans: []LoanInfo{
						{
							Name:      "Also Ignored",
							StartDate: "2025-01",
							Term:      120, // 10 years, past death
						},
					},
				},
			},
			expectedWarnings: 4, // Event after death (2) + scenario event after death (2), no loan warnings
		},
		{
			name:             "Empty configuration",
			deathDate:        "2030-01",
			commonEvents:     []EventInfo{},
			commonLoans:      []LoanInfo{},
			scenarios:        []ScenarioInfo{},
			expectedWarnings: 0,
		},
		{
			name:      "Only inactive scenarios",
			deathDate: "2030-01",
			commonEvents: []EventInfo{
				{
					Name:      "Valid Event",
					StartDate: "2025-01",
					EndDate:   "2028-01",
				},
			},
			commonLoans: []LoanInfo{},
			scenarios: []ScenarioInfo{
				{
					Name:   "Inactive Scenario 1",
					Active: false,
					Events: []EventInfo{
						{
							Name:      "Should Be Ignored",
							StartDate: "2031-01",
							EndDate:   "2032-01",
						},
					},
				},
				{
					Name:   "Inactive Scenario 2",
					Active: false,
					Loans: []LoanInfo{
						{
							Name:      "Should Be Ignored",
							StartDate: "2025-01",
							Term:      120,
						},
					},
				},
			},
			expectedWarnings: 0, // Only common events count, inactive scenarios ignored
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := processor.ValidateConfiguration(
				tt.deathDate,
				tt.commonEvents,
				tt.commonLoans,
				tt.scenarios,
			)

			if len(warnings) != tt.expectedWarnings {
				t.Errorf("ValidateConfiguration() returned %d warnings, expected %d",
					len(warnings), tt.expectedWarnings)
				for i, warning := range warnings {
					t.Logf("Warning %d: %s", i+1, warning)
				}
			}
		})
	}
}

func TestProcessor_ValidateConfigurationTypes(t *testing.T) {
	processor := NewProcessor()

	// Test type conversion and data flow
	events := []EventInfo{
		{
			Name:      "Test Event",
			StartDate: "2025-01",
			EndDate:   "2026-01",
		},
	}

	loans := []LoanInfo{
		{
			Name:      "Test Loan",
			StartDate: "2025-01",
			Term:      24,
		},
	}

	scenarios := []ScenarioInfo{
		{
			Name:   "Test Scenario",
			Active: true,
			Events: events,
			Loans:  loans,
		},
	}

	warnings := processor.ValidateConfiguration("2030-01", events, loans, scenarios)

	// Should not crash and should return a warnings slice (can be nil or empty)
	// This test is about ensuring the method doesn't panic and handles the types correctly
	_ = warnings // We don't care about the actual warnings, just that it doesn't crash
}

func TestEventInfo(t *testing.T) {
	// Test EventInfo struct
	event := EventInfo{
		Name:      "Test Event",
		StartDate: "2025-01",
		EndDate:   "2026-01",
	}

	if event.Name != "Test Event" {
		t.Errorf("EventInfo.Name = %s, expected Test Event", event.Name)
	}
	if event.StartDate != "2025-01" {
		t.Errorf("EventInfo.StartDate = %s, expected 2025-01", event.StartDate)
	}
	if event.EndDate != "2026-01" {
		t.Errorf("EventInfo.EndDate = %s, expected 2026-01", event.EndDate)
	}
}

func TestLoanInfo(t *testing.T) {
	// Test LoanInfo struct
	loan := LoanInfo{
		Name:      "Test Loan",
		StartDate: "2025-01",
		Term:      36,
	}

	if loan.Name != "Test Loan" {
		t.Errorf("LoanInfo.Name = %s, expected Test Loan", loan.Name)
	}
	if loan.StartDate != "2025-01" {
		t.Errorf("LoanInfo.StartDate = %s, expected 2025-01", loan.StartDate)
	}
	if loan.Term != 36 {
		t.Errorf("LoanInfo.Term = %d, expected 36", loan.Term)
	}
}

func TestScenarioInfo(t *testing.T) {
	// Test ScenarioInfo struct
	events := []EventInfo{{Name: "Event1"}}
	loans := []LoanInfo{{Name: "Loan1"}}

	scenario := ScenarioInfo{
		Name:   "Test Scenario",
		Active: true,
		Events: events,
		Loans:  loans,
	}

	if scenario.Name != "Test Scenario" {
		t.Errorf("ScenarioInfo.Name = %s, expected Test Scenario", scenario.Name)
	}
	if !scenario.Active {
		t.Errorf("ScenarioInfo.Active = %t, expected true", scenario.Active)
	}
	if len(scenario.Events) != 1 {
		t.Errorf("ScenarioInfo.Events length = %d, expected 1", len(scenario.Events))
	}
	if len(scenario.Loans) != 1 {
		t.Errorf("ScenarioInfo.Loans length = %d, expected 1", len(scenario.Loans))
	}
	if scenario.Events[0].Name != "Event1" {
		t.Errorf("ScenarioInfo.Events[0].Name = %s, expected Event1", scenario.Events[0].Name)
	}
	if scenario.Loans[0].Name != "Loan1" {
		t.Errorf("ScenarioInfo.Loans[0].Name = %s, expected Loan1", scenario.Loans[0].Name)
	}
}

func TestProcessor_ValidateConfigurationEdgeCases(t *testing.T) {
	processor := NewProcessor()

	// Test with nil slices
	warnings := processor.ValidateConfiguration("2030-01", nil, nil, nil)
	if len(warnings) != 0 {
		t.Error("ValidateConfiguration() with nil inputs should return empty warnings slice")
	}

	// Test with empty death date
	warnings = processor.ValidateConfiguration("", []EventInfo{}, []LoanInfo{}, []ScenarioInfo{})
	if len(warnings) != 0 {
		t.Error("ValidateConfiguration() with empty death date should return empty warnings slice")
	}

	// Test mixed active/inactive scenarios
	scenarios := []ScenarioInfo{
		{Name: "Active", Active: true, Events: []EventInfo{{StartDate: "2031-01"}}},
		{Name: "Inactive", Active: false, Events: []EventInfo{{StartDate: "2031-01"}}},
		{Name: "Active2", Active: true, Loans: []LoanInfo{{StartDate: "2025-01", Term: 120}}},
	}

	warnings = processor.ValidateConfiguration("2030-01", []EventInfo{}, []LoanInfo{}, scenarios)

	// Should get warnings from active scenarios only
	activeWarnings := 0
	for _, warning := range warnings {
		if warning != "" {
			activeWarnings++
		}
	}

	if activeWarnings == 0 {
		t.Error("Expected some warnings from active scenarios with problematic dates")
	}
}
