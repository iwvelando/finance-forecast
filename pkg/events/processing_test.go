package events

import (
	"testing"
	"time"
)

func TestEvent_FormDateList(t *testing.T) {
	tests := []struct {
		name        string
		event       Event
		deathDate   string
		expectCount int
		expectError bool
	}{
		{
			name: "Monthly event for 1 year",
			event: Event{
				Name:      "Monthly Income",
				StartDate: "2025-01",
				EndDate:   "2025-12",
				Frequency: 1,
			},
			deathDate:   "2030-01",
			expectCount: 12,
			expectError: false,
		},
		{
			name: "Quarterly event for 1 year",
			event: Event{
				Name:      "Quarterly Payment",
				StartDate: "2025-01",
				EndDate:   "2025-12",
				Frequency: 3,
			},
			deathDate:   "2030-01",
			expectCount: 4,
			expectError: false,
		},
		{
			name: "Annual event for 3 years",
			event: Event{
				Name:      "Annual Bonus",
				StartDate: "2025-01",
				EndDate:   "2027-01",
				Frequency: 12,
			},
			deathDate:   "2030-01",
			expectCount: 3,
			expectError: false,
		},
		{
			name: "One-time event",
			event: Event{
				Name:      "One Time Payment",
				StartDate: "2025-06",
				EndDate:   "2025-06",
				Frequency: 1,
			},
			deathDate:   "2030-01",
			expectCount: 1,
			expectError: false,
		},
		{
			name: "Event with no start date uses current time",
			event: Event{
				Name:      "Current Event",
				EndDate:   "2025-12",
				Frequency: 1,
			},
			deathDate:   "2030-01",
			expectCount: 0, // Can't predict exact count since it uses current time
			expectError: false,
		},
		{
			name: "Event with no end date uses death date",
			event: Event{
				Name:      "Until Death Event",
				StartDate: "2025-01",
				Frequency: 12,
			},
			deathDate:   "2027-01",
			expectCount: 3, // 2025, 2026, 2027
			expectError: false,
		},
		{
			name: "Event ending exactly at death date",
			event: Event{
				Name:      "Death Date Event",
				StartDate: "2025-01",
				EndDate:   "2026-01",
				Frequency: 12,
			},
			deathDate:   "2026-01",
			expectCount: 2, // 2025, 2026
			expectError: false,
		},
		{
			name: "Event with invalid start date",
			event: Event{
				Name:      "Invalid Event",
				StartDate: "invalid-date",
				EndDate:   "2025-12",
				Frequency: 1,
			},
			deathDate:   "2030-01",
			expectCount: 0,
			expectError: true,
		},
		{
			name: "Event with invalid end date",
			event: Event{
				Name:      "Invalid End Event",
				StartDate: "2025-01",
				EndDate:   "invalid-date",
				Frequency: 1,
			},
			deathDate:   "2030-01",
			expectCount: 0,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.FormDateList(tt.deathDate)

			if tt.expectError {
				if err == nil {
					t.Errorf("FormDateList() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("FormDateList() error = %v", err)
				return
			}

			if tt.expectCount > 0 && len(tt.event.DateList) != tt.expectCount {
				t.Errorf("FormDateList() expected %d dates, got %d", tt.expectCount, len(tt.event.DateList))
			}
		})
	}
}

func TestEvent_FormDateListWithEndDateModification(t *testing.T) {
	event := Event{
		Name:      "Test Event",
		StartDate: "2025-01",
		// No EndDate specified
		Frequency: 1,
	}
	deathDate := "2025-06"

	err := event.FormDateList(deathDate)
	if err != nil {
		t.Fatalf("FormDateList() error = %v", err)
	}

	// EndDate should be set to deathDate
	if event.EndDate != deathDate {
		t.Errorf("FormDateList() should set EndDate to %s, got %s", deathDate, event.EndDate)
	}
}

func TestProcessor_ParseDateLists(t *testing.T) {
	processor := NewProcessor()

	events := []*Event{
		{
			Name:      "Event 1",
			StartDate: "2025-01",
			EndDate:   "2025-03",
			Frequency: 1,
		},
		{
			Name:      "Event 2",
			StartDate: "2025-06",
			EndDate:   "2025-12",
			Frequency: 3,
		},
	}

	err := processor.ParseDateLists(events, "2030-01")
	if err != nil {
		t.Errorf("ParseDateLists() error = %v", err)
	}

	// Verify all events have date lists
	for i, event := range events {
		if len(event.DateList) == 0 {
			t.Errorf("Event %d has empty DateList", i)
		}
	}

	// Verify expected counts
	if len(events[0].DateList) != 3 { // Jan, Feb, Mar
		t.Errorf("Event 1 expected 3 dates, got %d", len(events[0].DateList))
	}

	if len(events[1].DateList) != 3 { // Jun, Sep, Dec
		t.Errorf("Event 2 expected 3 dates, got %d", len(events[1].DateList))
	}
}

func TestProcessor_ParseDateListsWithError(t *testing.T) {
	processor := NewProcessor()

	events := []*Event{
		{
			Name:      "Valid Event",
			StartDate: "2025-01",
			EndDate:   "2025-03",
			Frequency: 1,
		},
		{
			Name:      "Invalid Event",
			StartDate: "invalid-date",
			EndDate:   "2025-12",
			Frequency: 1,
		},
	}

	err := processor.ParseDateLists(events, "2030-01")
	if err == nil {
		t.Errorf("ParseDateLists() expected error for invalid date but got none")
	}
}

func TestNewProcessor(t *testing.T) {
	processor := NewProcessor()
	if processor == nil {
		t.Errorf("NewProcessor() returned nil")
	}
}

func TestEvent_DateListBoundaries(t *testing.T) {
	tests := []struct {
		name      string
		event     Event
		deathDate string
		checkFunc func(*testing.T, Event)
	}{
		{
			name: "Event dates should not exceed end date",
			event: Event{
				Name:      "Boundary Test",
				StartDate: "2025-01",
				EndDate:   "2025-06",
				Frequency: 1,
			},
			deathDate: "2030-01",
			checkFunc: func(t *testing.T, e Event) {
				endTime, _ := time.Parse("2006-01", e.EndDate)
				for _, date := range e.DateList {
					if date.After(endTime) {
						t.Errorf("Date %s is after EndDate %s", date.Format("2006-01"), e.EndDate)
					}
				}
			},
		},
		{
			name: "Event dates should not exceed death date",
			event: Event{
				Name:      "Long Event",
				StartDate: "2025-01",
				EndDate:   "2035-01", // After death date
				Frequency: 12,        // annual frequency
			},
			deathDate: "2030-01",
			checkFunc: func(t *testing.T, e Event) {
				// EndDate should be modified to death date
				if e.EndDate != "2030-01" {
					t.Errorf("EndDate should be modified to death date, got %s", e.EndDate)
				} else {
					t.Log("EndDate should be modified to death date")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.event.FormDateList(tt.deathDate)
			if err != nil {
				t.Fatalf("FormDateList() error = %v", err)
			}
			tt.checkFunc(t, tt.event)
		})
	}
}
