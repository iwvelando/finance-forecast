// Package validation provides configuration validation utilities.
package validation

import (
	"fmt"

	"github.com/iwvelando/finance-forecast/pkg/datetime"
)

// ValidateDeathDate checks if loans mature before the death date
func ValidateDeathDate(loanName, startDate, deathDate string, termMonths int) (string, error) {
	maturityDate, err := datetime.OffsetDate(startDate, datetime.DateTimeLayout, termMonths)
	if err != nil {
		return "", err
	}

	if maturityDate > deathDate {
		return fmt.Sprintf("Loan '%s' matures after death date (%s > %s) - loan will have outstanding balance",
			loanName, maturityDate, deathDate), nil
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

	// Check common loans for terms extending past death
	for _, loan := range cv.Common.Loans {
		warning, err := ValidateDeathDate(fmt.Sprintf("Common loan '%s'", loan.Name), loan.StartDate, deathDate, loan.Term)
		if err == nil && warning != "" {
			warnings = append(warnings, warning)
		}
	}

	// Check scenario loans for terms extending past death
	for _, scenario := range cv.Scenarios {
		if !scenario.Active {
			continue
		}
		for _, loan := range scenario.Loans {
			warning, err := ValidateDeathDate(fmt.Sprintf("Scenario '%s' loan '%s'", scenario.Name, loan.Name), loan.StartDate, deathDate, loan.Term)
			if err == nil && warning != "" {
				warnings = append(warnings, warning)
			}
		}
	}

	return warnings
}
