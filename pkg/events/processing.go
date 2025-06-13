// Package events provides common event processing utilities.
package events

import (
	"time"

	"github.com/iwvelando/finance-forecast/pkg/datetime"
)

// Event represents a financial event with date processing capabilities.
type Event struct {
	Name      string
	Amount    float64
	StartDate string
	EndDate   string
	Frequency int // months
	DateList  []time.Time
}

// Processor handles event processing operations
type Processor struct{}

// NewProcessor creates a new event processor
func NewProcessor() *Processor {
	return &Processor{}
}

// ProcessDateLists processes date lists for multiple events
func (p *Processor) ParseDateLists(events []*Event, deathDate string) error {
	for _, event := range events {
		if err := event.FormDateList(deathDate); err != nil {
			return err
		}
	}
	return nil
}

// FormDateList generates a list of dates for this event based on its frequency
// It takes only deathDate as parameter and uses current time internally
func (e *Event) FormDateList(deathDate string) error {
	// Use current time if start date is not specified
	currentTime := time.Now().Format("2006-01")
	startDate := e.StartDate
	if startDate == "" {
		startDate = currentTime
	}

	// Set default end date to death date if not specified
	endDate := e.EndDate
	if endDate == "" {
		endDate = deathDate
		e.EndDate = deathDate // Update the event's EndDate
	}

	// Check if event extends beyond death date and modify end date
	if beforeDeath, err := datetime.DateBeforeDate(deathDate, endDate); err == nil && beforeDeath {
		// Event extends beyond death date, truncate it
		endDate = deathDate
		e.EndDate = deathDate // Update the event's EndDate
	}

	// Generate date list based on frequency
	var dates []time.Time
	current := startDate

	for {
		// Parse current date
		currentTime, err := time.Parse("2006-01", current)
		if err != nil {
			return err
		}

		// Parse end date for comparison
		endTime, err := time.Parse("2006-01", endDate)
		if err != nil {
			return err
		}

		// Check if current date is beyond end date
		if currentTime.After(endTime) {
			break
		}

		dates = append(dates, currentTime)

		// Move to next date based on frequency
		if nextDate, err := datetime.OffsetDate(current, "2006-01", e.Frequency); err != nil {
			return err
		} else {
			current = nextDate
		}
	}

	e.DateList = dates
	return nil
}
