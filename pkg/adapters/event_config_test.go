package adapters

import (
	"testing"

	"github.com/iwvelando/finance-forecast/internal/config"
	"github.com/iwvelando/finance-forecast/pkg/datetime"
)

func TestNewEventConfigAdapter(t *testing.T) {
	configEvent := &config.Event{
		Name:      "Test Event",
		Amount:    500.0,
		StartDate: "2025-01",
		EndDate:   "2025-12",
		Frequency: 3,
	}

	adapter := NewEventConfigAdapter(configEvent)

	if adapter == nil {
		t.Fatal("NewEventConfigAdapter() returned nil")
	}

	if adapter.ConfigEvent != configEvent {
		t.Error("ConfigEvent not set correctly")
	}

	if adapter.PkgEvent == nil {
		t.Error("PkgEvent not initialized")
	}

	// Verify PkgEvent is initialized with correct values
	if adapter.PkgEvent.Name != "Test Event" {
		t.Errorf("PkgEvent.Name = %s, expected 'Test Event'", adapter.PkgEvent.Name)
	}
	if adapter.PkgEvent.Amount != 500.0 {
		t.Errorf("PkgEvent.Amount = %f, expected 500.0", adapter.PkgEvent.Amount)
	}
	if adapter.PkgEvent.StartDate != "2025-01" {
		t.Errorf("PkgEvent.StartDate = %s, expected '2025-01'", adapter.PkgEvent.StartDate)
	}
	if adapter.PkgEvent.EndDate != "2025-12" {
		t.Errorf("PkgEvent.EndDate = %s, expected '2025-12'", adapter.PkgEvent.EndDate)
	}
	if adapter.PkgEvent.Frequency != 3 {
		t.Errorf("PkgEvent.Frequency = %d, expected 3", adapter.PkgEvent.Frequency)
	}
}

func TestEventConfigAdapter_FormDateList(t *testing.T) {
	configEvent := &config.Event{
		Name:      "Quarterly Event",
		Amount:    1000.0,
		StartDate: "2025-01",
		EndDate:   "2025-12",
		Frequency: 3, // Quarterly
	}

	adapter := NewEventConfigAdapter(configEvent)

	err := adapter.FormDateList("2030-01")
	if err != nil {
		t.Fatalf("FormDateList() error = %v", err)
	}

	// Check that ConfigEvent DateList was populated
	if len(configEvent.DateList) == 0 {
		t.Error("ConfigEvent.DateList should be populated after FormDateList()")
	}

	// For quarterly events from Jan to Dec, should have 4 dates: Jan, Apr, Jul, Oct
	expectedCount := 4
	if len(configEvent.DateList) != expectedCount {
		t.Errorf("ConfigEvent.DateList length = %d, expected %d", len(configEvent.DateList), expectedCount)
	}

	// Verify first date
	expectedFirstDate := datetime.MustParseTime(datetime.DateTimeLayout, "2025-01")
	if !configEvent.DateList[0].Equal(expectedFirstDate) {
		t.Errorf("First date = %v, expected %v", configEvent.DateList[0], expectedFirstDate)
	}

	// Verify dates are sorted
	for i := 1; i < len(configEvent.DateList); i++ {
		if configEvent.DateList[i].Before(configEvent.DateList[i-1]) {
			t.Errorf("Dates not sorted at index %d", i)
		}
	}
}

func TestEventConfigAdapter_FormDateListWithEndDateModification(t *testing.T) {
	configEvent := &config.Event{
		Name:      "No End Date Event",
		Amount:    500.0,
		StartDate: "2025-01",
		EndDate:   "", // No end date specified
		Frequency: 1,
	}

	adapter := NewEventConfigAdapter(configEvent)
	deathDate := "2026-06"

	err := adapter.FormDateList(deathDate)
	if err != nil {
		t.Fatalf("FormDateList() error = %v", err)
	}

	// EndDate should be set to deathDate
	if configEvent.EndDate != deathDate {
		t.Errorf("ConfigEvent.EndDate = %s, expected %s", configEvent.EndDate, deathDate)
	}

	// PkgEvent should also be updated
	if adapter.PkgEvent.EndDate != deathDate {
		t.Errorf("PkgEvent.EndDate = %s, expected %s", adapter.PkgEvent.EndDate, deathDate)
	}
}

func TestEventConfigAdapter_FormDateListSyncBehavior(t *testing.T) {
	// Test that changes to ConfigEvent are synced to PkgEvent before processing
	configEvent := &config.Event{
		Name:      "Original Name",
		Amount:    100.0,
		StartDate: "2025-01",
		EndDate:   "2025-06",
		Frequency: 1,
	}

	adapter := NewEventConfigAdapter(configEvent)

	// Modify ConfigEvent after adapter creation
	configEvent.Name = "Modified Name"
	configEvent.Amount = 200.0
	configEvent.StartDate = "2025-02"
	configEvent.EndDate = "2025-08"
	configEvent.Frequency = 2

	err := adapter.FormDateList("2030-01")
	if err != nil {
		t.Fatalf("FormDateList() error = %v", err)
	}

	// PkgEvent should reflect the modified values
	if adapter.PkgEvent.Name != "Modified Name" {
		t.Errorf("PkgEvent.Name = %s, expected 'Modified Name'", adapter.PkgEvent.Name)
	}
	if adapter.PkgEvent.Amount != 200.0 {
		t.Errorf("PkgEvent.Amount = %f, expected 200.0", adapter.PkgEvent.Amount)
	}
	if adapter.PkgEvent.StartDate != "2025-02" {
		t.Errorf("PkgEvent.StartDate = %s, expected '2025-02'", adapter.PkgEvent.StartDate)
	}
	if adapter.PkgEvent.EndDate != "2025-08" {
		t.Errorf("PkgEvent.EndDate = %s, expected '2025-08'", adapter.PkgEvent.EndDate)
	}
	if adapter.PkgEvent.Frequency != 2 {
		t.Errorf("PkgEvent.Frequency = %d, expected 2", adapter.PkgEvent.Frequency)
	}
}

func TestEventConfigAdapter_FormDateListError(t *testing.T) {
	configEvent := &config.Event{
		Name:      "Invalid Event",
		Amount:    100.0,
		StartDate: "invalid-date",
		EndDate:   "2025-12",
		Frequency: 1,
	}

	adapter := NewEventConfigAdapter(configEvent)

	err := adapter.FormDateList("2030-01")
	if err == nil {
		t.Error("FormDateList() expected error for invalid start date but got none")
	}
}

func TestEventConfigAdapter_FormDateListEmptyStartDate(t *testing.T) {
	configEvent := &config.Event{
		Name:      "No Start Date Event",
		Amount:    100.0,
		StartDate: "", // No start date - should use current time
		EndDate:   "2025-12",
		Frequency: 1,
	}

	adapter := NewEventConfigAdapter(configEvent)

	err := adapter.FormDateList("2030-01")
	if err != nil {
		t.Fatalf("FormDateList() error = %v", err)
	}

	// Should have at least one date (using current time as start)
	if len(configEvent.DateList) == 0 {
		t.Error("ConfigEvent.DateList should not be empty when using current time as start")
	}
}

func TestEventConfigAdapter_DateListBidirectionalSync(t *testing.T) {
	// Test that DateList is properly synced between ConfigEvent and PkgEvent
	configEvent := &config.Event{
		Name:      "Sync Test Event",
		Amount:    100.0,
		StartDate: "2025-01",
		EndDate:   "2025-03",
		Frequency: 1,
	}

	adapter := NewEventConfigAdapter(configEvent)

	// Initially both should have empty DateList
	if len(configEvent.DateList) != 0 {
		t.Error("ConfigEvent.DateList should initially be empty")
	}
	if len(adapter.PkgEvent.DateList) != 0 {
		t.Error("PkgEvent.DateList should initially be empty")
	}

	err := adapter.FormDateList("2030-01")
	if err != nil {
		t.Fatalf("FormDateList() error = %v", err)
	}

	// Both should now have the same DateList
	if len(configEvent.DateList) != len(adapter.PkgEvent.DateList) {
		t.Errorf("DateList lengths don't match: ConfigEvent=%d, PkgEvent=%d",
			len(configEvent.DateList), len(adapter.PkgEvent.DateList))
	}

	// Verify actual dates match
	for i, configDate := range configEvent.DateList {
		if i >= len(adapter.PkgEvent.DateList) {
			t.Fatalf("PkgEvent.DateList too short at index %d", i)
		}
		if !configDate.Equal(adapter.PkgEvent.DateList[i]) {
			t.Errorf("Date mismatch at index %d: ConfigEvent=%v, PkgEvent=%v",
				i, configDate, adapter.PkgEvent.DateList[i])
		}
	}
}
