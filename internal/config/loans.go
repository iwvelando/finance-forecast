// Package config defines the data structures related to configuration and
// includes functions for modifying the loading and parsing the config.
package config

import (
	"fmt"

	"github.com/iwvelando/finance-forecast/pkg/datetime"
	"github.com/iwvelando/finance-forecast/pkg/loans"
	"go.uber.org/zap"
)

// Loan indicates a loan and its parameters.
type Loan struct {
	Name                    string             `yaml:"name,omitempty" mapstructure:"name"`
	StartDate               string             `yaml:"startDate,omitempty" mapstructure:"startDate"`
	Principal               float64            `yaml:"principal" mapstructure:"principal"`
	InterestRate            float64            `yaml:"interestRate" mapstructure:"interestRate"`
	Term                    int                `yaml:"term" mapstructure:"term"`
	DownPayment             float64            `yaml:"downPayment,omitempty" mapstructure:"downPayment"`
	Escrow                  float64            `yaml:"escrow,omitempty" mapstructure:"escrow"`
	MortgageInsurance       float64            `yaml:"mortgageInsurance,omitempty" mapstructure:"mortgageInsurance"`
	MortgageInsuranceCutoff float64            `yaml:"mortgageInsuranceCutoff,omitempty" mapstructure:"mortgageInsuranceCutoff"`
	EarlyPayoffThreshold    float64            `yaml:"earlyPayoffThreshold,omitempty" mapstructure:"earlyPayoffThreshold"`
	EarlyPayoffDate         string             `yaml:"earlyPayoffDate,omitempty" mapstructure:"earlyPayoffDate"`
	SellProperty            bool               `yaml:"sellProperty,omitempty" mapstructure:"sellProperty"`
	SellPrice               float64            `yaml:"sellPrice,omitempty" mapstructure:"sellPrice"`
	SellCostsNet            float64            `yaml:"sellCostsNet,omitempty" mapstructure:"sellCostsNet"`
	ExtraPrincipalPayments  []Event            `yaml:"extraPrincipalPayments,omitempty" mapstructure:"extraPrincipalPayments"`
	AmortizationSchedule    map[string]Payment `yaml:"amortizationSchedule,omitempty" mapstructure:"amortizationSchedule"`
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
	if conf == nil {
		return fmt.Errorf("configuration cannot be nil")
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	// First handle the processing for all Loans in Scenarios.
	for i, scenario := range conf.Scenarios {
		for j := range scenario.Loans {
			// Set default sell price if not specified
			if conf.Scenarios[i].Loans[j].SellPrice == 0 {
				conf.Scenarios[i].Loans[j].SellPrice = conf.Scenarios[i].Loans[j].Principal
			}

			err := conf.Scenarios[i].Loans[j].GetAmortizationSchedule(logger, *conf)
			if err != nil {
				return fmt.Errorf("failed to process loan %s in scenario %s: %w",
					conf.Scenarios[i].Loans[j].Name, scenario.Name, err)
			}
		}
	}

	// Next handle the processing for the Common Loans.
	for i := range conf.Common.Loans {
		// Set default sell price if not specified
		if conf.Common.Loans[i].SellPrice == 0 {
			conf.Common.Loans[i].SellPrice = conf.Common.Loans[i].Principal
		}

		err := conf.Common.Loans[i].GetAmortizationSchedule(logger, *conf)
		if err != nil {
			return fmt.Errorf("failed to process common loan %s: %w",
				conf.Common.Loans[i].Name, err)
		}
	}

	return nil
}

// GetAmortizationSchedule computes the amortization schedule for a given Loan.
func (loan *Loan) GetAmortizationSchedule(logger *zap.Logger, conf Configuration) error {
	if loan == nil {
		return fmt.Errorf("loan cannot be nil")
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	if loan.Name == "" {
		return fmt.Errorf("loan name cannot be empty")
	}

	// Convert config.Loan to loans.LoanConfig using helper
	loanConfig := loan.ToLoansConfig()
	if loanConfig == nil {
		return fmt.Errorf("failed to convert loan %s to loans config", loan.Name)
	}

	// Create generator and generate schedule
	generator := loans.NewAmortizationScheduleGenerator(logger)
	schedule, err := generator.GenerateSchedule(loanConfig, conf.Common.DeathDate)
	if err != nil {
		return fmt.Errorf("failed to generate amortization schedule for loan %s: %w", loan.Name, err)
	}

	// Convert the schedule back to our internal format using helper
	loan.AmortizationSchedule = make(map[string]Payment)
	for date, payment := range schedule {
		loan.AmortizationSchedule[date] = FromLoansPayment(payment)
	}

	return nil
}

// ExtraPrincipal returns an extra principal payment, if present, or 0
func (loan *Loan) ExtraPrincipal(logger *zap.Logger, date string) (float64, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

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
func (loan *Loan) CheckEarlyPayoffThreshold(currentMonth, deathDate string, balance float64) (string, error) {
	if loan == nil {
		return "", fmt.Errorf("loan cannot be nil")
	}
	if currentMonth == "" {
		return "", fmt.Errorf("currentMonth cannot be empty")
	}
	if deathDate == "" {
		return "", fmt.Errorf("deathDate cannot be empty")
	}

	// Convert config.Loan to loans.LoanConfig using helper
	loanConfig := loan.ToLoansConfig()
	if loanConfig == nil {
		return "", fmt.Errorf("failed to convert loan %s to loans config", loan.Name)
	}

	// Create generator
	generator := loans.NewAmortizationScheduleGenerator(nil) // Pass nil since the logger isn't used in the function

	// Use the generator to check early payoff threshold
	note, err := generator.CheckEarlyPayoffThresholdAndUpdate(loanConfig, currentMonth, deathDate, balance, loanConfig.AmortizationSchedule)
	if err != nil {
		return "", fmt.Errorf("failed to check early payoff threshold for loan %s: %w", loan.Name, err)
	}

	// If there was an early payoff, update the loan
	if note != "" {
		// Update the loan's early payoff threshold to prevent future checks
		loan.EarlyPayoffThreshold = 0

		// Synchronize the schedule using helper
		loan.SyncScheduleWithLoansConfig(loanConfig)
	}

	return note, nil
}
