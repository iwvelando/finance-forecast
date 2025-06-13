// Package config defines the data structures related to configuration and
// includes functions for modifying the loading and parsing the config.
package config

import (
	"github.com/iwvelando/finance-forecast/pkg/datetime"
	"github.com/iwvelando/finance-forecast/pkg/loans"
	"go.uber.org/zap"
)

// Loan indicates a loan and its parameters.
type Loan struct {
	Name                    string
	StartDate               string
	Principal               float64
	InterestRate            float64
	Term                    int // months
	DownPayment             float64
	Escrow                  float64
	MortgageInsurance       float64
	MortgageInsuranceCutoff float64
	EarlyPayoffThreshold    float64
	EarlyPayoffDate         string
	SellProperty            bool
	SellPrice               float64
	SellCostsNet            float64
	ExtraPrincipalPayments  []Event
	AmortizationSchedule    map[string]Payment
}

// Payment holds the values for a given payment.
type Payment struct {
	Payment            float64
	Principal          float64
	Interest           float64
	RemainingPrincipal float64
	RefundableEscrow   float64
}

// ProcessLoans iterates through all loans and produces the amortization
// schedules.
func (conf *Configuration) ProcessLoans(logger *zap.Logger) error {
	// First handle the processing for all Loans in Scenarios.
	for i, scenario := range conf.Scenarios {
		for j := range scenario.Loans {
			if conf.Scenarios[i].Loans[j].SellPrice == 0 {
				conf.Scenarios[i].Loans[j].SellPrice = conf.Scenarios[i].Loans[j].Principal
			}
			err := conf.Scenarios[i].Loans[j].GetAmortizationSchedule(logger, *conf)
			if err != nil {
				return err
			}
		}
	}

	// Next handle the processing for the Common Loans.
	for i := range conf.Common.Loans {
		if conf.Common.Loans[i].SellPrice == 0 {
			conf.Common.Loans[i].SellPrice = conf.Common.Loans[i].Principal
		}
		err := conf.Common.Loans[i].GetAmortizationSchedule(logger, *conf)
		if err != nil {
			return err
		}
	}

	return nil
}

// GetAmortizationSchedule computes the amortization schedule for a given Loan.
func (loan *Loan) GetAmortizationSchedule(logger *zap.Logger, conf Configuration) error {
	// Convert config.Loan to loans.LoanConfig
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

	// Create generator and generate schedule
	generator := loans.NewAmortizationScheduleGenerator(logger)
	schedule, err := generator.GenerateSchedule(loanConfig, conf.Common.DeathDate)
	if err != nil {
		return err
	}

	// Convert the schedule back to our internal format
	loan.AmortizationSchedule = make(map[string]Payment)
	for date, payment := range schedule {
		loan.AmortizationSchedule[date] = Payment{
			Payment:            payment.Payment,
			Principal:          payment.Principal,
			Interest:           payment.Interest,
			RemainingPrincipal: payment.RemainingPrincipal,
			RefundableEscrow:   payment.RefundableEscrow,
		}
	}

	return nil
}

// ExtraPrincipal returns an extra principal payment, if present, or 0
func (loan *Loan) ExtraPrincipal(logger *zap.Logger, date string) (float64, error) {
	var loanEvents []loans.Event
	for _, event := range loan.ExtraPrincipalPayments {
		var dateList []string
		for _, eventDate := range event.DateList {
			dateList = append(dateList, eventDate.Format(datetime.DateTimeLayout))
		}
		loanEvents = append(loanEvents, loans.Event{
			Name:      event.Name,
			Amount:    event.Amount,
			StartDate: event.StartDate,
			EndDate:   event.EndDate,
			Frequency: event.Frequency,
			DateList:  dateList,
		})
	}

	generator := loans.NewAmortizationScheduleGenerator(logger)
	amount := generator.CalculateExtraPrincipalWithLogging(loanEvents, date, loan.Name)
	return amount, nil
}

// CheckEarlyPayoffThreshold checks for whether or not it is time to payoff a
// loan early based on an optionally-configured threshold.
func (loan *Loan) CheckEarlyPayoffThreshold(logger *zap.Logger, currentMonth, deathDate string, balance float64) (string, error) {
	// Convert config.Loan to loans.LoanConfig (just the fields needed for this operation)
	loanConfig := &loans.LoanConfig{
		Name:                 loan.Name,
		StartDate:            loan.StartDate,
		Principal:            loan.Principal,
		EarlyPayoffThreshold: loan.EarlyPayoffThreshold,
		SellProperty:         loan.SellProperty,
		SellPrice:            loan.SellPrice,
		SellCostsNet:         loan.SellCostsNet,
		Escrow:               loan.Escrow,
		AmortizationSchedule: make(map[string]loans.Payment),
	}

	// Convert the schedule to loans.Payment format
	for date, payment := range loan.AmortizationSchedule {
		loanConfig.AmortizationSchedule[date] = loans.Payment{
			Payment:            payment.Payment,
			Principal:          payment.Principal,
			Interest:           payment.Interest,
			RemainingPrincipal: payment.RemainingPrincipal,
			RefundableEscrow:   payment.RefundableEscrow,
		}
	}

	// Create generator
	generator := loans.NewAmortizationScheduleGenerator(logger)

	// Use the generator to check early payoff threshold
	note, err := generator.CheckEarlyPayoffThresholdAndUpdate(loanConfig, currentMonth, deathDate, balance, loanConfig.AmortizationSchedule)
	if err != nil {
		return "", err
	}

	// If there was an early payoff, copy the modified schedule back
	if note != "" {
		// Update the loan's early payoff threshold to prevent future checks
		loan.EarlyPayoffThreshold = 0

		// Copy the updated schedule back
		for date, payment := range loanConfig.AmortizationSchedule {
			loan.AmortizationSchedule[date] = Payment{
				Payment:            payment.Payment,
				Principal:          payment.Principal,
				Interest:           payment.Interest,
				RemainingPrincipal: payment.RemainingPrincipal,
				RefundableEscrow:   payment.RefundableEscrow,
			}
		}

		// Delete any dates that were removed from the loans schedule
		for date := range loan.AmortizationSchedule {
			if _, exists := loanConfig.AmortizationSchedule[date]; !exists {
				delete(loan.AmortizationSchedule, date)
			}
		}
	}

	return note, nil
}
