package config

import (
	"testing"

	"go.uber.org/zap"
)

func TestValidateConfigurationEdgeCases(t *testing.T) {
	logger := zap.NewNop()

	// Test configuration with edge cases
	conf := Configuration{
		Common: Common{
			DeathDate:     "2030-01",
			StartingValue: 10000,
			Events: []Event{
				{
					Name:      "Event After Death",
					Amount:    1000,
					StartDate: "2031-01",
					Frequency: 1,
				},
			},
			Loans: []Loan{
				{
					Name:         "Long Loan",
					StartDate:    "2025-01",
					Principal:    100000,
					InterestRate: 5.0,
					Term:         120, // 10 years, matures after death
				},
			},
		},
		Scenarios: []Scenario{
			{
				Name:   "Test",
				Active: true,
			},
		},
	}

	warnings := conf.ValidateConfiguration()

	// Verify we get appropriate warnings for edge cases
	if len(warnings) == 0 {
		t.Error("Expected validation warnings for edge cases but got none")
	}

	t.Logf("Found %d warnings:", len(warnings))
	for i, warning := range warnings {
		t.Logf("%d. %s", i+1, warning)
	}

	// Test that we can still process this configuration despite warnings
	err := conf.ParseDateLists()
	if err != nil {
		t.Errorf("ParseDateLists failed: %v", err)
	}

	err = conf.ProcessLoans(logger)
	if err != nil {
		t.Errorf("ProcessLoans failed: %v", err)
	}
}

func TestValidateConfigurationValid(t *testing.T) {
	// Test with a completely valid configuration
	conf := Configuration{
		Common: Common{
			DeathDate:     "2030-01",
			StartingValue: 10000,
			Events: []Event{
				{
					Name:      "Normal Event",
					Amount:    1000,
					StartDate: "2025-01",
					EndDate:   "2029-12",
					Frequency: 1,
				},
			},
			Loans: []Loan{
				{
					Name:         "Normal Loan",
					StartDate:    "2025-01",
					Principal:    100000,
					InterestRate: 5.0,
					Term:         60, // 5 years, completes before death
				},
			},
		},
		Scenarios: []Scenario{
			{
				Name:   "Valid Scenario",
				Active: true,
			},
		},
	}

	warnings := conf.ValidateConfiguration()

	// A well-formed configuration should have minimal or no warnings
	if len(warnings) > 2 {
		t.Errorf("Expected minimal warnings for valid config, got %d: %v", len(warnings), warnings)
	}
}
