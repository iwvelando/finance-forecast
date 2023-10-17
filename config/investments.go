// Package config defines the data structures related to configuration and
// includes functions for modifying the loading and parsing the config.
package config

import (
	"fmt"

	"go.uber.org/zap"
)

// Investment indicates a investment and its parameters.
type Investment struct {
	Name                    string
	DistributionDate        string
	ContributionEndDate     string
	CurrentValue            float64
	MonthlyContribution     float64
	ContributionsFromIncome bool
	GrowthRatePercent       float64
	GrowthTaxRatePercent    float64
	DistributionPercent     float64
	MonthsElapsed           int
}

// Process investments; returns negative values for contributions or positive values
// for distributions which are assumed to occur in December
func (investment *Investment) HandleInvestment(logger *zap.Logger, date string) (float64, error) {
	var payment float64
	if investment.ContributionsFromIncome {
		payment = -investment.MonthlyContribution
	} else {
		payment = 0.0
	}

	if investment.MonthsElapsed%12 == 0 {
		payment -= investment.CurrentValue*investment.GrowthRatePercent/100.0*investment.GrowthTaxRatePercent/100.0
		investment.CurrentValue *= 1.0 + investment.GrowthRatePercent/100.0
		logger.Debug(fmt.Sprintf("%s: investment %s value has grown to %.2f with tax liability of %.2f", date, investment.Name, investment.CurrentValue, investment.CurrentValue*investment.GrowthRatePercent/100.0*investment.GrowthTaxRatePercent/100.0),
			zap.String("op", "config.HandleInvestment"),
		)
	}
	investment.MonthsElapsed++

	december, err := CheckMonth(date, "12")
	if err != nil {
		return payment, err
	}

	var contributing bool
	if investment.MonthlyContribution > 0.0 {
		contributing, err = DateBeforeDate(date, investment.ContributionEndDate)
		if err != nil {
			return payment, err
		}
	} else {
		contributing = false
	}

	withdrawing, err := DateBeforeDate(investment.DistributionDate, date)
	if err != nil {
		return payment, err
	}
	if contributing {
		investment.CurrentValue += investment.MonthlyContribution
		logger.Debug(fmt.Sprintf("%s: investment %s received contribution of %.2f", date, investment.Name, investment.MonthlyContribution),
			zap.String("op", "config.HandleInvestment"),
		)
	} else {
		if withdrawing && december {
			payment = investment.CurrentValue * investment.DistributionPercent / 100.0
			investment.CurrentValue -= payment
			logger.Debug(fmt.Sprintf("%s: investment %s processed withdrawal of %.2f, %.2f remaining", date, investment.Name, payment, investment.CurrentValue),
				zap.String("op", "config.HandleInvestment"),
			)
		} else {
			payment = 0.0
		}
	}
	return payment, nil
}
