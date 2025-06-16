// Package config defines conversion utilities for configuration objects.
package config

import (
	"github.com/iwvelando/finance-forecast/pkg/datetime"
	"github.com/iwvelando/finance-forecast/pkg/loans"
)

// ToLoansConfig converts an internal config.Loan to a pkg/loans.LoanConfig
// This eliminates duplication in conversion logic
func (loan *Loan) ToLoansConfig() *loans.LoanConfig {
	if loan == nil {
		return nil
	}

	loanConfig := &loans.LoanConfig{
		Name:                    loan.Name,
		StartDate:               loan.StartDate,
		Principal:               loan.Principal,
		InterestRate:            loan.InterestRate,
		Term:                    loan.Term,
		DownPayment:             loan.DownPayment,
		Escrow:                  loan.Escrow,
		MortgageInsurance:       loan.MortgageInsurance,
		MortgageInsuranceCutoff: loan.MortgageInsuranceCutoff,
		EarlyPayoffThreshold:    loan.EarlyPayoffThreshold,
		EarlyPayoffDate:         loan.EarlyPayoffDate,
		SellProperty:            loan.SellProperty,
		SellPrice:               loan.SellPrice,
		SellCostsNet:            loan.SellCostsNet,
		AmortizationSchedule:    make(map[string]loans.Payment),
	}

	// Convert ExtraPrincipalPayments
	for _, event := range loan.ExtraPrincipalPayments {
		var dateList []string
		for _, eventDate := range event.DateList {
			dateList = append(dateList, eventDate.Format(datetime.DateTimeLayout))
		}
		loanConfig.ExtraPrincipalPayments = append(loanConfig.ExtraPrincipalPayments, loans.Event{
			Name:      event.Name,
			Amount:    event.Amount,
			StartDate: event.StartDate,
			EndDate:   event.EndDate,
			Frequency: event.Frequency,
			DateList:  dateList,
		})
	}

	// Convert AmortizationSchedule if it exists
	for date, payment := range loan.AmortizationSchedule {
		loanConfig.AmortizationSchedule[date] = loans.Payment{
			Payment:            payment.Payment,
			Principal:          payment.Principal,
			Interest:           payment.Interest,
			RemainingPrincipal: payment.RemainingPrincipal,
			RefundableEscrow:   payment.RefundableEscrow,
		}
	}

	return loanConfig
}

// FromLoansPayment converts a pkg/loans.Payment to internal config.Payment
func FromLoansPayment(payment loans.Payment) Payment {
	return Payment{
		Payment:            payment.Payment,
		Principal:          payment.Principal,
		Interest:           payment.Interest,
		RemainingPrincipal: payment.RemainingPrincipal,
		RefundableEscrow:   payment.RefundableEscrow,
	}
}

// UpdateFromLoansConfig updates the loan's amortization schedule from a loans.LoanConfig
func (loan *Loan) UpdateFromLoansConfig(loanConfig *loans.LoanConfig) {
	if loan == nil || loanConfig == nil {
		return
	}

	loan.AmortizationSchedule = make(map[string]Payment)
	for date, payment := range loanConfig.AmortizationSchedule {
		loan.AmortizationSchedule[date] = FromLoansPayment(payment)
	}
}

// SyncScheduleWithLoansConfig synchronizes the loan's schedule with the loans config,
// including deletion of dates that no longer exist
func (loan *Loan) SyncScheduleWithLoansConfig(loanConfig *loans.LoanConfig) {
	if loan == nil || loanConfig == nil {
		return
	}

	// Copy the updated schedule back
	for date, payment := range loanConfig.AmortizationSchedule {
		loan.AmortizationSchedule[date] = FromLoansPayment(payment)
	}

	// Delete any dates that were removed from the loans schedule
	for date := range loan.AmortizationSchedule {
		if _, exists := loanConfig.AmortizationSchedule[date]; !exists {
			delete(loan.AmortizationSchedule, date)
		}
	}
}
