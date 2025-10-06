package config

import (
	"fmt"
	"time"
)

// Investment describes an investment account with contributions and withdrawals.
type Investment struct {
	Name                      string  `yaml:"name,omitempty"`
	StartingValue             float64 `yaml:"startingValue,omitempty"`
	AnnualReturnRate          float64 `yaml:"annualReturnRate,omitempty"`
	TaxRate                   float64 `yaml:"taxRate,omitempty"`
	ContributionsReduceIncome bool    `yaml:"contributionsReduceIncome,omitempty"`
	Contributions             []Event `yaml:"contributions,omitempty"`
	Withdrawals               []Event `yaml:"withdrawals,omitempty"`
}

// FormDateListsWithFixedTime parses contribution and withdrawal date lists using the provided fixed time.
func (investment *Investment) FormDateListsWithFixedTime(conf Configuration, fixedTime time.Time) error {
	if investment == nil {
		return fmt.Errorf("investment cannot be nil")
	}

	for i := range investment.Contributions {
		if investment.Contributions[i].Frequency == 0 {
			investment.Contributions[i].Frequency = 1
		}

		if investment.Contributions[i].Percentage != 0 {
			return fmt.Errorf("investment %s contribution %d: percentage is not supported for contributions", investment.Name, i)
		}

		if err := investment.Contributions[i].FormDateListWithFixedTime(conf, fixedTime); err != nil {
			return fmt.Errorf("investment %s contribution %d: %w", investment.Name, i, err)
		}
	}

	hasAmountWithdrawals := false
	hasPercentWithdrawals := false
	for i := range investment.Withdrawals {
		if investment.Withdrawals[i].Frequency == 0 {
			investment.Withdrawals[i].Frequency = 1
		}

		amount := investment.Withdrawals[i].Amount
		percent := investment.Withdrawals[i].Percentage
		if amount != 0 && percent != 0 {
			return fmt.Errorf("investment %s withdrawal %d: specify either amount or percentage, not both", investment.Name, i)
		}
		if amount == 0 && percent == 0 {
			return fmt.Errorf("investment %s withdrawal %d: must specify amount or percentage", investment.Name, i)
		}
		if percent != 0 {
			hasPercentWithdrawals = true
		} else {
			hasAmountWithdrawals = true
		}

		if err := investment.Withdrawals[i].FormDateListWithFixedTime(conf, fixedTime); err != nil {
			return fmt.Errorf("investment %s withdrawal %d: %w", investment.Name, i, err)
		}
	}

	if hasAmountWithdrawals && hasPercentWithdrawals {
		return fmt.Errorf("investment %s: withdrawals cannot mix amount and percentage entries", investment.Name)
	}

	return nil
}
