// Package adapters provides adapter implementations between different package interfaces.
package adapters

import (
	"time"

	"github.com/iwvelando/finance-forecast/internal/config"
	"github.com/iwvelando/finance-forecast/pkg/finance"
)

// ConfigEventAdapter wraps config.Event to implement finance.EventWithDates
type ConfigEventAdapter struct {
	Event config.Event
}

// GetName returns the event name
func (w ConfigEventAdapter) GetName() string {
	return w.Event.Name
}

// GetAmount returns the event amount
func (w ConfigEventAdapter) GetAmount() float64 {
	return w.Event.Amount
}

// GetDateList returns the event date list
func (w ConfigEventAdapter) GetDateList() []time.Time {
	return w.Event.DateList
}

// EventsToFinanceEvents converts config.Event slices to finance.EventWithDates slices
func EventsToFinanceEvents(events []config.Event) []finance.EventWithDates {
	if events == nil {
		return nil
	}

	var financeEvents []finance.EventWithDates
	for _, event := range events {
		financeEvents = append(financeEvents, ConfigEventAdapter{Event: event})
	}
	return financeEvents
}

// ConfigLoanAdapter wraps config.Loan to implement finance.LoanWithSchedule
type ConfigLoanAdapter struct {
	Loan config.Loan
}

// GetName returns the loan name
func (w ConfigLoanAdapter) GetName() string {
	return w.Loan.Name
}

// GetPaymentForDate returns the loan payment for a given date
func (w ConfigLoanAdapter) GetPaymentForDate(date string) (float64, bool) {
	payment, present := w.Loan.AmortizationSchedule[date]
	return payment.Payment, present
}

// LoansToFinanceLoans converts config.Loan slices to finance.LoanWithSchedule slices
func LoansToFinanceLoans(loans []config.Loan) []finance.LoanWithSchedule {
	if loans == nil {
		return nil
	}

	var financeLoans []finance.LoanWithSchedule
	for _, loan := range loans {
		financeLoans = append(financeLoans, ConfigLoanAdapter{Loan: loan})
	}
	return financeLoans
}
