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

// FormDateList handles the date to time.Time parsing for one given event.
func (event *Event) FormDateList(deathDate string) error {
	dateList := make([]time.Time, 1)
	var startDateT time.Time
	var err error

	// Unspecified startDate goes to the current time.
	if event.StartDate == "" {
		startDateT, err = time.Parse(datetime.DateTimeLayout, time.Now().Format(datetime.DateTimeLayout))
		if err != nil {
			return err
		}
	} else {
		startDateT, err = time.Parse(datetime.DateTimeLayout, event.StartDate)
		if err != nil {
			return err
		}
	}

	// Unspecified endDate goes to the deathDate.
	if event.EndDate == "" {
		event.EndDate = deathDate
	}
	endDateT, err := time.Parse(datetime.DateTimeLayout, event.EndDate)
	if err != nil {
		return err
	}

	// Identify all dates where an event takes place and aggregate them in
	// dateList.
	dateList[0] = startDateT
	for {
		nextDate := dateList[len(dateList)-1].AddDate(0, event.Frequency, 0)
		if nextDate.Equal(endDateT) {
			dateList = append(dateList, nextDate)
			break
		} else if nextDate.After(endDateT) {
			break
		} else {
			dateList = append(dateList, nextDate)
		}
	}
	event.DateList = dateList

	return nil
}
