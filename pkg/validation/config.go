// Package validation provides configuration validation utilities.
package validation

import (
	"fmt"

	"github.com/iwvelando/finance-forecast/pkg/datetime"
)

// ValidateDeathDate validates loan start date format but doesn't generate warnings for loans maturing after death
func ValidateDeathDate(loanName, startDate, deathDate string, termMonths int) (string, error) {
	_, err := datetime.OffsetDate(startDate, datetime.DateTimeLayout, termMonths)
	if err != nil {
		return "", err
	}
	return "", nil
}

// ValidateEventDates checks if events are properly scheduled relative to death date
func ValidateEventDates(eventName, startDate, endDate, deathDate string) []string {
	var warnings []string

	if startDate >= deathDate {
		warnings = append(warnings, fmt.Sprintf("Event '%s' starts at or after death date (%s >= %s)",
			eventName, startDate, deathDate))
	}

	if endDate != "" && endDate > deathDate {
		warnings = append(warnings, fmt.Sprintf("Event '%s' ends after death date (%s > %s)",
			eventName, endDate, deathDate))
	}

	return warnings
}

// ValidateConfiguration performs comprehensive configuration validation
type ConfigValidator struct {
	Common    CommonConfig
	Scenarios []ScenarioConfig
}

type CommonConfig struct {
	DeathDate string
	Events    []EventConfig
	Loans     []LoanConfig
}

type ScenarioConfig struct {
	Name   string
	Active bool
	Events []EventConfig
	Loans  []LoanConfig
}

type EventConfig struct {
	Name      string
	StartDate string
	EndDate   string
}

type LoanConfig struct {
	Name      string
	StartDate string
	Term      int
}

// ValidateAll validates the entire configuration and returns warnings
func (cv *ConfigValidator) ValidateAll() []string {
	var warnings []string

	deathDate := cv.Common.DeathDate

	// Check common events for dates at or after death
	for _, event := range cv.Common.Events {
		eventWarnings := ValidateEventDates(event.Name, event.StartDate, event.EndDate, deathDate)
		warnings = append(warnings, eventWarnings...)
	}

	// Check scenario events for dates at or after death
	for _, scenario := range cv.Scenarios {
		if !scenario.Active {
			continue
		}
		for _, event := range scenario.Events {
			eventWarnings := ValidateEventDates(fmt.Sprintf("Scenario '%s' event '%s'", scenario.Name, event.Name), event.StartDate, event.EndDate, deathDate)
			warnings = append(warnings, eventWarnings...)
		}
	}

	// No validation needed for loan maturity dates

	return warnings
}
