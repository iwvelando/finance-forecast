// Package config provides shared configuration processing utilities.
package config

import (
	"github.com/iwvelando/finance-forecast/pkg/validation"
)

// Processor handles configuration processing and validation
type Processor struct {
}

// NewProcessor creates a new configuration processor
func NewProcessor() *Processor {
	return &Processor{}
}

// ValidateConfiguration validates a configuration and returns warnings
func (p *Processor) ValidateConfiguration(commonDeathDate string, commonEvents []EventInfo, commonLoans []LoanInfo, scenarios []ScenarioInfo) []string {
	// Convert to validation types
	var validationEvents []validation.EventConfig
	for _, event := range commonEvents {
		validationEvents = append(validationEvents, validation.EventConfig{
			Name:      event.Name,
			StartDate: event.StartDate,
			EndDate:   event.EndDate,
		})
	}

	var validationLoans []validation.LoanConfig
	for _, loan := range commonLoans {
		validationLoans = append(validationLoans, validation.LoanConfig{
			Name:      loan.Name,
			StartDate: loan.StartDate,
			Term:      loan.Term,
		})
	}

	var validationScenarios []validation.ScenarioConfig
	for _, scenario := range scenarios {
		var scenarioEvents []validation.EventConfig
		for _, event := range scenario.Events {
			scenarioEvents = append(scenarioEvents, validation.EventConfig{
				Name:      event.Name,
				StartDate: event.StartDate,
				EndDate:   event.EndDate,
			})
		}

		var scenarioLoans []validation.LoanConfig
		for _, loan := range scenario.Loans {
			scenarioLoans = append(scenarioLoans, validation.LoanConfig{
				Name:      loan.Name,
				StartDate: loan.StartDate,
				Term:      loan.Term,
			})
		}

		validationScenarios = append(validationScenarios, validation.ScenarioConfig{
			Name:   scenario.Name,
			Active: scenario.Active,
			Events: scenarioEvents,
			Loans:  scenarioLoans,
		})
	}

	validator := validation.ConfigValidator{
		Common: validation.CommonConfig{
			DeathDate: commonDeathDate,
			Events:    validationEvents,
			Loans:     validationLoans,
		},
		Scenarios: validationScenarios,
	}

	return validator.ValidateAll()
}

// Event information for validation
type EventInfo struct {
	Name      string
	StartDate string
	EndDate   string
}

// Loan information for validation
type LoanInfo struct {
	Name      string
	StartDate string
	Term      int
}

// Scenario information for validation
type ScenarioInfo struct {
	Name   string
	Active bool
	Events []EventInfo
	Loans  []LoanInfo
}
