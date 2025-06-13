package finance

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

// Mock implementations for testing
type mockEvent struct {
	name     string
	amount   float64
	dateList []time.Time
}

func (m mockEvent) GetName() string {
	return m.name
}

func (m mockEvent) GetAmount() float64 {
	return m.amount
}

func (m mockEvent) GetDateList() []time.Time {
	return m.dateList
}

type mockLoan struct {
	name     string
	schedule map[string]float64
}

func (m mockLoan) GetName() string {
	return m.name
}

func (m mockLoan) GetPaymentForDate(date string) (float64, bool) {
	payment, present := m.schedule[date]
	return payment, present
}

func TestEventProcessor_ProcessEventsForDate(t *testing.T) {
	logger := zap.NewNop()
	processor := NewEventProcessor(logger)

	// Create test events
	date1, _ := time.Parse("2006-01", "2025-06")
	date2, _ := time.Parse("2006-01", "2025-07")

	events := []EventWithDates{
		mockEvent{
			name:     "Income",
			amount:   1000.0,
			dateList: []time.Time{date1, date2},
		},
		mockEvent{
			name:     "Expense",
			amount:   -500.0,
			dateList: []time.Time{date1},
		},
		mockEvent{
			name:     "Other Income",
			amount:   200.0,
			dateList: []time.Time{date2},
		},
	}

	tests := []struct {
		name     string
		date     string
		expected float64
	}{
		{
			name:     "Date with multiple events",
			date:     "2025-06",
			expected: 500.0, // 1000 - 500
		},
		{
			name:     "Date with some events",
			date:     "2025-07",
			expected: 1200.0, // 1000 + 200
		},
		{
			name:     "Date with no events",
			date:     "2025-08",
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, err := processor.ProcessEventsForDate(tt.date, events, "2006-01")
			if err != nil {
				t.Errorf("ProcessEventsForDate() error = %v", err)
			}
			if amount != tt.expected {
				t.Errorf("ProcessEventsForDate() = %.2f, expected %.2f", amount, tt.expected)
			}
		})
	}
}

func TestEventProcessor_ProcessEventsForDateInvalidDate(t *testing.T) {
	logger := zap.NewNop()
	processor := NewEventProcessor(logger)

	events := []EventWithDates{}

	_, err := processor.ProcessEventsForDate("invalid-date", events, "2006-01")
	if err == nil {
		t.Errorf("ProcessEventsForDate() expected error for invalid date but got none")
	}
}

func TestLoanProcessor_ProcessLoansForDate(t *testing.T) {
	logger := zap.NewNop()
	processor := NewLoanProcessor(logger)

	loans := []LoanWithSchedule{
		mockLoan{
			name: "Mortgage",
			schedule: map[string]float64{
				"2025-06": 1500.0,
				"2025-07": 1500.0,
			},
		},
		mockLoan{
			name: "Car Loan",
			schedule: map[string]float64{
				"2025-06": 400.0,
				"2025-08": 400.0,
			},
		},
	}

	tests := []struct {
		name     string
		date     string
		expected float64
	}{
		{
			name:     "Date with multiple loan payments",
			date:     "2025-06",
			expected: -1900.0, // -(1500 + 400)
		},
		{
			name:     "Date with one loan payment",
			date:     "2025-07",
			expected: -1500.0, // -1500
		},
		{
			name:     "Date with different loan payment",
			date:     "2025-08",
			expected: -400.0, // -400
		},
		{
			name:     "Date with no loan payments",
			date:     "2025-09",
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount := processor.ProcessLoansForDate(tt.date, loans)
			if amount != tt.expected {
				t.Errorf("ProcessLoansForDate() = %.2f, expected %.2f", amount, tt.expected)
			}
		})
	}
}

func TestForecastEngine_ProcessMonthlyChanges(t *testing.T) {
	logger := zap.NewNop()
	engine := NewForecastEngine(logger)

	// Create test data
	date1, _ := time.Parse("2006-01", "2025-06")

	events := []EventWithDates{
		mockEvent{
			name:     "Income",
			amount:   2000.0,
			dateList: []time.Time{date1},
		},
		mockEvent{
			name:     "Expense",
			amount:   -300.0,
			dateList: []time.Time{date1},
		},
	}

	loans := []LoanWithSchedule{
		mockLoan{
			name: "Mortgage",
			schedule: map[string]float64{
				"2025-06": 1200.0,
			},
		},
	}

	tests := []struct {
		name     string
		date     string
		expected float64
	}{
		{
			name:     "Combined events and loans",
			date:     "2025-06",
			expected: 500.0, // 2000 - 300 - 1200
		},
		{
			name:     "No events or loans",
			date:     "2025-07",
			expected: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			amount, err := engine.ProcessMonthlyChanges(tt.date, events, loans, "2006-01")
			if err != nil {
				t.Errorf("ProcessMonthlyChanges() error = %v", err)
			}
			if amount != tt.expected {
				t.Errorf("ProcessMonthlyChanges() = %.2f, expected %.2f", amount, tt.expected)
			}
		})
	}
}

func TestForecastEngine_ProcessMonthlyChangesInvalidDate(t *testing.T) {
	logger := zap.NewNop()
	engine := NewForecastEngine(logger)

	_, err := engine.ProcessMonthlyChanges("invalid-date", nil, nil, "2006-01")
	if err == nil {
		t.Errorf("ProcessMonthlyChanges() expected error for invalid date but got none")
	}
}

func TestNewEventProcessor(t *testing.T) {
	logger := zap.NewNop()
	processor := NewEventProcessor(logger)
	if processor == nil {
		t.Errorf("NewEventProcessor() returned nil")
		return
	}
	if processor.logger != logger {
		t.Errorf("NewEventProcessor() logger not set correctly")
	}
}

func TestNewLoanProcessor(t *testing.T) {
	logger := zap.NewNop()
	processor := NewLoanProcessor(logger)
	if processor == nil {
		t.Errorf("NewLoanProcessor() returned nil")
		return
	}
	if processor.logger != logger {
		t.Errorf("NewLoanProcessor() logger not set correctly")
	}
}

func TestNewForecastEngine(t *testing.T) {
	logger := zap.NewNop()
	engine := NewForecastEngine(logger)
	if engine == nil {
		t.Errorf("NewForecastEngine() returned nil")
		return
	}
	if engine.logger != logger {
		t.Errorf("NewForecastEngine() logger not set correctly")
	}
	if engine.eventProcessor == nil {
		t.Errorf("NewForecastEngine() eventProcessor not initialized")
	}
	if engine.loanProcessor == nil {
		t.Errorf("NewForecastEngine() loanProcessor not initialized")
	}
}

// Test edge cases
func TestEventProcessorEdgeCases(t *testing.T) {
	logger := zap.NewNop()
	processor := NewEventProcessor(logger)

	// Test with empty events slice
	amount, err := processor.ProcessEventsForDate("2025-06", []EventWithDates{}, "2006-01")
	if err != nil {
		t.Errorf("ProcessEventsForDate() with empty events error = %v", err)
	}
	if amount != 0.0 {
		t.Errorf("ProcessEventsForDate() with empty events = %.2f, expected 0.0", amount)
	}

	// Test with nil events
	amount, err = processor.ProcessEventsForDate("2025-06", nil, "2006-01")
	if err != nil {
		t.Errorf("ProcessEventsForDate() with nil events error = %v", err)
	}
	if amount != 0.0 {
		t.Errorf("ProcessEventsForDate() with nil events = %.2f, expected 0.0", amount)
	}
}

func TestLoanProcessorEdgeCases(t *testing.T) {
	logger := zap.NewNop()
	processor := NewLoanProcessor(logger)

	// Test with empty loans slice
	amount := processor.ProcessLoansForDate("2025-06", []LoanWithSchedule{})
	if amount != 0.0 {
		t.Errorf("ProcessLoansForDate() with empty loans = %.2f, expected 0.0", amount)
	}

	// Test with nil loans
	amount = processor.ProcessLoansForDate("2025-06", nil)
	if amount != 0.0 {
		t.Errorf("ProcessLoansForDate() with nil loans = %.2f, expected 0.0", amount)
	}
}

func TestForecastEngineWithNilInputs(t *testing.T) {
	logger := zap.NewNop()
	engine := NewForecastEngine(logger)

	// Test with nil events and loans
	amount, err := engine.ProcessMonthlyChanges("2025-06", nil, nil, "2006-01")
	if err != nil {
		t.Errorf("ProcessMonthlyChanges() with nil inputs error = %v", err)
	}
	if amount != 0.0 {
		t.Errorf("ProcessMonthlyChanges() with nil inputs = %.2f, expected 0.0", amount)
	}
}
