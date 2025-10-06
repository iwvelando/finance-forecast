// Package finance provides common financial calculation utilities.
package finance

import (
	"fmt"
	"time"

	"go.uber.org/zap"
)

// EventProcessor handles financial event processing
type EventProcessor struct {
	logger *zap.Logger
}

// NewEventProcessor creates a new event processor with the given logger.
// If logger is nil, it will use a no-op logger to prevent panics.
func NewEventProcessor(logger *zap.Logger) *EventProcessor {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &EventProcessor{logger: logger}
}

// ProcessEventsForDate processes all events for a specific date and returns the total amount
func (ep *EventProcessor) ProcessEventsForDate(date string, events []EventWithDates, layout string) (float64, error) {
	if date == "" {
		return 0.0, fmt.Errorf("date cannot be empty")
	}
	if layout == "" {
		return 0.0, fmt.Errorf("layout cannot be empty")
	}

	amount := 0.0
	dateT, err := time.Parse(layout, date)
	if err != nil {
		return amount, fmt.Errorf("failed to parse date %s with layout %s: %w", date, layout, err)
	}

	for _, event := range events {
		if event == nil {
			ep.logger.Warn("Skipping nil event")
			continue
		}

		eventDates := event.GetDateList()
		if eventDates == nil {
			ep.logger.Warn("Event has nil date list", zap.String("event", event.GetName()))
			continue
		}

		for _, eventDate := range eventDates {
			if dateT.Equal(eventDate) {
				ep.logger.Debug("Event active",
					zap.String("date", date),
					zap.String("event", event.GetName()),
					zap.Float64("amount", event.GetAmount()),
				)
				amount += event.GetAmount()
				break
			}
		}
	}
	return amount, nil
}

// LoanProcessor handles loan payment processing
type LoanProcessor struct {
	logger *zap.Logger
}

// NewLoanProcessor creates a new loan processor with the given logger.
// If logger is nil, it will use a no-op logger to prevent panics.
func NewLoanProcessor(logger *zap.Logger) *LoanProcessor {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &LoanProcessor{logger: logger}
}

// ProcessLoansForDate processes all loan payments for a specific date and returns the total amount
func (lp *LoanProcessor) ProcessLoansForDate(date string, loans []LoanWithSchedule) float64 {
	if date == "" {
		lp.logger.Warn("ProcessLoansForDate called with empty date")
		return 0.0
	}

	amount := 0.0
	for _, loan := range loans {
		if loan == nil {
			lp.logger.Warn("Skipping nil loan")
			continue
		}

		if payment, present := loan.GetPaymentForDate(date); present {
			lp.logger.Debug("Loan payment active",
				zap.String("date", date),
				zap.String("loan", loan.GetName()),
				zap.Float64("amount", payment),
			)
			amount -= payment
		}
	}
	return amount
}

// EventWithDates interface for events that have date lists
type EventWithDates interface {
	GetName() string
	GetAmount() float64
	GetDateList() []time.Time
}

// LoanWithSchedule interface for loans that have amortization schedules
type LoanWithSchedule interface {
	GetName() string
	GetPaymentForDate(date string) (float64, bool)
}

// ForecastEngine coordinates the overall forecasting process
type ForecastEngine struct {
	eventProcessor      *EventProcessor
	loanProcessor       *LoanProcessor
	investmentProcessor *InvestmentProcessor
	logger              *zap.Logger
}

// NewForecastEngine creates a new forecast engine
func NewForecastEngine(logger *zap.Logger) *ForecastEngine {
	if logger == nil {
		// Create a no-op logger if none provided
		logger = zap.NewNop()
	}

	return &ForecastEngine{
		eventProcessor:      NewEventProcessor(logger),
		loanProcessor:       NewLoanProcessor(logger),
		investmentProcessor: NewInvestmentProcessor(logger),
		logger:              logger,
	}
}

// ProcessMonthlyChanges calculates the total financial changes for a given month
func (fe *ForecastEngine) ProcessMonthlyChanges(date string, events []EventWithDates, loans []LoanWithSchedule, layout string) (float64, error) {
	if fe.eventProcessor == nil || fe.loanProcessor == nil {
		return 0, fmt.Errorf("forecast engine not properly initialized")
	}

	eventAmount, err := fe.eventProcessor.ProcessEventsForDate(date, events, layout)
	if err != nil {
		return 0, fmt.Errorf("failed to process events for date %s: %w", date, err)
	}

	loanAmount := fe.loanProcessor.ProcessLoansForDate(date, loans)

	return eventAmount + loanAmount, nil
}

// ProcessInvestments processes investments for a specific date and returns the total change and per-investment details.
func (fe *ForecastEngine) ProcessInvestments(date string, investments []Investment, layout string, states map[string]*InvestmentState) (float64, []InvestmentChange, error) {
	if fe.investmentProcessor == nil {
		return 0, nil, fmt.Errorf("investment processor not initialized")
	}
	return fe.investmentProcessor.ProcessInvestmentsForDate(date, investments, layout, states)
}
