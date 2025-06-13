// Package finance provides common financial calculation utilities.
package finance

import (
	"time"

	"go.uber.org/zap"
)

// EventProcessor handles financial event processing
type EventProcessor struct {
	logger *zap.Logger
}

// NewEventProcessor creates a new event processor
func NewEventProcessor(logger *zap.Logger) *EventProcessor {
	return &EventProcessor{logger: logger}
}

// ProcessEventsForDate processes all events for a specific date and returns the total amount
func (ep *EventProcessor) ProcessEventsForDate(date string, events []EventWithDates, layout string) (float64, error) {
	amount := 0.0
	dateT, err := time.Parse(layout, date)
	if err != nil {
		return amount, err
	}

	for _, event := range events {
		for _, eventDate := range event.GetDateList() {
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

// NewLoanProcessor creates a new loan processor
func NewLoanProcessor(logger *zap.Logger) *LoanProcessor {
	return &LoanProcessor{logger: logger}
}

// ProcessLoansForDate processes all loan payments for a specific date and returns the total amount
func (lp *LoanProcessor) ProcessLoansForDate(date string, loans []LoanWithSchedule) float64 {
	amount := 0.0
	for _, loan := range loans {
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
	eventProcessor *EventProcessor
	loanProcessor  *LoanProcessor
	logger         *zap.Logger
}

// NewForecastEngine creates a new forecast engine
func NewForecastEngine(logger *zap.Logger) *ForecastEngine {
	return &ForecastEngine{
		eventProcessor: NewEventProcessor(logger),
		loanProcessor:  NewLoanProcessor(logger),
		logger:         logger,
	}
}

// ProcessMonthlyChanges calculates the total financial changes for a given month
func (fe *ForecastEngine) ProcessMonthlyChanges(date string, events []EventWithDates, loans []LoanWithSchedule, layout string) (float64, error) {
	eventAmount, err := fe.eventProcessor.ProcessEventsForDate(date, events, layout)
	if err != nil {
		return 0, err
	}

	loanAmount := fe.loanProcessor.ProcessLoansForDate(date, loans)

	return eventAmount + loanAmount, nil
}
