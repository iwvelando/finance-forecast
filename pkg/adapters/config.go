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

// ConfigInvestmentAdapter wraps config.Investment to implement finance.Investment
type ConfigInvestmentAdapter struct {
	investment            config.Investment
	contributionSchedule  map[string]float64
	withdrawalSchedule    map[string]float64
	withdrawalPercentages map[string]float64
	fromCash              bool
}

// newConfigInvestmentAdapter constructs an adapter for the provided investment
func newConfigInvestmentAdapter(investment config.Investment) ConfigInvestmentAdapter {
	adapter := ConfigInvestmentAdapter{
		investment:            investment,
		contributionSchedule:  make(map[string]float64),
		withdrawalSchedule:    make(map[string]float64),
		withdrawalPercentages: make(map[string]float64),
		fromCash:              investment.ContributionsFromCash,
	}

	for _, contribution := range investment.Contributions {
		for _, date := range contribution.DateList {
			key := date.Format(config.DateTimeLayout)
			adapter.contributionSchedule[key] += contribution.Amount
		}
	}

	for _, withdrawal := range investment.Withdrawals {
		for _, date := range withdrawal.DateList {
			key := date.Format(config.DateTimeLayout)
			if withdrawal.Percentage != 0 {
				adapter.withdrawalPercentages[key] += withdrawal.Percentage
			} else {
				adapter.withdrawalSchedule[key] += withdrawal.Amount
			}
		}
	}

	return adapter
}

// GetName returns the investment name
func (a ConfigInvestmentAdapter) GetName() string {
	return a.investment.Name
}

// GetStartingValue returns the investment starting value
func (a ConfigInvestmentAdapter) GetStartingValue() float64 {
	return a.investment.StartingValue
}

// GetAnnualReturnRate returns the annual return rate percentage
func (a ConfigInvestmentAdapter) GetAnnualReturnRate() float64 {
	return a.investment.AnnualReturnRate
}

// GetTaxRate returns the tax rate percentage applied to gains
func (a ConfigInvestmentAdapter) GetTaxRate() float64 {
	return a.investment.TaxRate
}

// GetContributionForDate returns the total contribution scheduled for the provided date
func (a ConfigInvestmentAdapter) GetContributionForDate(date string) float64 {
	return a.contributionSchedule[date]
}

// GetWithdrawalForDate returns the total withdrawal scheduled for the provided date
func (a ConfigInvestmentAdapter) GetWithdrawalForDate(date string) float64 {
	return a.withdrawalSchedule[date]
}

// GetWithdrawalPercentageForDate returns the total withdrawal percentage scheduled for the provided date
func (a ConfigInvestmentAdapter) GetWithdrawalPercentageForDate(date string) float64 {
	return a.withdrawalPercentages[date]
}

// ContributionsFromCash indicates whether contributions reduce monthly income
func (a ConfigInvestmentAdapter) ContributionsFromCash() bool {
	return a.fromCash
}

// InvestmentsToFinanceInvestments converts config.Investment slices to finance.Investment slices
func InvestmentsToFinanceInvestments(investments []config.Investment) []finance.Investment {
	if investments == nil {
		return nil
	}

	var financeInvestments []finance.Investment
	for _, inv := range investments {
		financeInvestments = append(financeInvestments, newConfigInvestmentAdapter(inv))
	}
	return financeInvestments
}
