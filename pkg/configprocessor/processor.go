// Package configprocessor provides shared configuration processing utilities.
package configprocessor

// EventInfo represents event configuration information
type EventInfo struct {
	Name      string
	StartDate string
	EndDate   string
}

// LoanInfo represents loan configuration information
type LoanInfo struct {
	Name      string
	StartDate string
	Term      int
}

// ScenarioInfo represents scenario configuration information
type ScenarioInfo struct {
	Name   string
	Active bool
	Events []EventInfo
	Loans  []LoanInfo
}

// Processor handles configuration processing and validation
type Processor struct{}

// NewProcessor creates a new configuration processor
func NewProcessor() *Processor {
	return &Processor{}
}

// ValidateConfiguration validates the configuration and returns warnings
func (p *Processor) ValidateConfiguration(deathDate string, commonEvents []EventInfo, scenarios []ScenarioInfo) []string {
	var warnings []string

	// Basic validation - if no death date provided, return empty warnings
	if deathDate == "" {
		return warnings
	}

	// Validate common events
	for _, event := range commonEvents {
		if event.StartDate >= deathDate {
			warnings = append(warnings, "Event '"+event.Name+"' starts at or after death date ("+event.StartDate+" >= "+deathDate+")")
		}
		if event.EndDate != "" && event.EndDate > deathDate {
			warnings = append(warnings, "Event '"+event.Name+"' ends after death date ("+event.EndDate+" > "+deathDate+")")
		}
	}

	// Validate active scenarios
	for _, scenario := range scenarios {
		if !scenario.Active {
			continue // Skip inactive scenarios
		}

		// Validate scenario events
		for _, event := range scenario.Events {
			if event.StartDate >= deathDate {
				warnings = append(warnings, "Event 'Scenario '"+scenario.Name+"' event '"+event.Name+"'' starts at or after death date ("+event.StartDate+" >= "+deathDate+")")
			}
			if event.EndDate != "" && event.EndDate > deathDate {
				warnings = append(warnings, "Event 'Scenario '"+scenario.Name+"' event '"+event.Name+"'' ends after death date ("+event.EndDate+" > "+deathDate+")")
			}
		}
	}

	if len(warnings) == 0 {
		return nil
	}
	return warnings
}
