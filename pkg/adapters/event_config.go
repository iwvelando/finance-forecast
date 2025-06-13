// Package adapters provides adapter implementations between different package interfaces.
package adapters

import (
	"github.com/iwvelando/finance-forecast/internal/config"
	"github.com/iwvelando/finance-forecast/pkg/events"
)

// EventConfigAdapter adapts between config.Event and events.Event
type EventConfigAdapter struct {
	ConfigEvent *config.Event
	PkgEvent    *events.Event
}

// NewEventConfigAdapter creates a new adapter
func NewEventConfigAdapter(configEvent *config.Event) *EventConfigAdapter {
	pkgEvent := &events.Event{
		Name:      configEvent.Name,
		Amount:    configEvent.Amount,
		StartDate: configEvent.StartDate,
		EndDate:   configEvent.EndDate,
		Frequency: configEvent.Frequency,
		DateList:  configEvent.DateList,
	}

	return &EventConfigAdapter{
		ConfigEvent: configEvent,
		PkgEvent:    pkgEvent,
	}
}

// FormDateList adapts the pkg FormDateList method for config Event
func (adapter *EventConfigAdapter) FormDateList(deathDate string) error {
	// Sync any changes to the config event before processing
	adapter.PkgEvent.Name = adapter.ConfigEvent.Name
	adapter.PkgEvent.Amount = adapter.ConfigEvent.Amount
	adapter.PkgEvent.StartDate = adapter.ConfigEvent.StartDate
	adapter.PkgEvent.EndDate = adapter.ConfigEvent.EndDate
	adapter.PkgEvent.Frequency = adapter.ConfigEvent.Frequency

	// Call the pkg implementation
	err := adapter.PkgEvent.FormDateList(deathDate)
	if err != nil {
		return err
	}

	// Copy the date list back to the config event
	adapter.ConfigEvent.DateList = adapter.PkgEvent.DateList

	// Copy any modified end date back
	adapter.ConfigEvent.EndDate = adapter.PkgEvent.EndDate

	return nil
}
