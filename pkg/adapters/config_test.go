package adapters

import (
	"testing"
	"time"

	"github.com/iwvelando/finance-forecast/internal/config"
)

func TestEventAdapter(t *testing.T) {
	event := config.Event{
		Name:     "Test Event",
		Amount:   100.0,
		DateList: []time.Time{time.Now()},
	}
	
	adapter := ConfigEventAdapter{Event: event}
	
	if adapter.GetName() != "Test Event" {
		t.Errorf("Expected name 'Test Event', got %s", adapter.GetName())
	}
	
	if adapter.GetAmount() != 100.0 {
		t.Errorf("Expected amount 100.0, got %f", adapter.GetAmount())
	}
}

func TestEventsToFinanceEvents(t *testing.T) {
	events := []config.Event{
		{
			Name:     "Test Event",
			Amount:   100.0,
			DateList: []time.Time{time.Now()},
		},
	}
	
	financeEvents := EventsToFinanceEvents(events)
	
	if len(financeEvents) != 1 {
		t.Errorf("Expected 1 finance event, got %d", len(financeEvents))
	}
}