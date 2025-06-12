// Package config defines the data structures related to configuration and
// includes functions for modifying the loading and parsing the config.
package config

import (
	"fmt"
	"math"
	"time"

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
// This will also take into account optional configuration such as down
// payments, early payoff dates, and selling property on payoff (for example
// when trading in vehicles or upgrading homes). TODO this function is a mouth
// full and is a candidate for being revised.
func (loan *Loan) GetAmortizationSchedule(logger *zap.Logger, conf Configuration) error {
	// Compute periodic payment fundamentals.
	periodicInterestRate := loan.InterestRate / (100.0 * 12.0)
	power := math.Pow((1.00 + periodicInterestRate), float64(loan.Term))
	discountFactor := (power - 1.00) / power
	loanPayment := (loan.Principal - loan.DownPayment) * periodicInterestRate / discountFactor

	loan.AmortizationSchedule = make(map[string]Payment)

	// Handle the first month individually. TODO consider using a ghost point
	// to prevent having to treat this differently.
	var firstPayment Payment
	extraPrincipal, err := loan.ExtraPrincipal(logger, loan.StartDate)
	if err != nil {
		return err
	}
	firstPayment.Payment = loanPayment + loan.Escrow + loan.DownPayment + extraPrincipal
	firstPayment.Interest = (loan.Principal - loan.DownPayment) * loan.InterestRate / (100.0 * 12.0)
	firstPayment.Principal = loanPayment - firstPayment.Interest + extraPrincipal
	firstPayment.RemainingPrincipal = (loan.Principal - loan.DownPayment) - firstPayment.Principal
	firstPayment.RefundableEscrow = loan.Escrow
	loan.AmortizationSchedule[loan.StartDate] = firstPayment

	// Iterate over the remainder of the term.
	previousMonth := loan.StartDate
	currentMonth, err := OffsetDate(previousMonth, DateTimeLayout, 1)
	if err != nil {
		return err
	}

	for month := 2; month <= loan.Term; month++ {
		// Check if we've reached or passed the death date
		if currentMonth == conf.Common.DeathDate {
			logger.Debug(fmt.Sprintf("Loan %s reached death date %s, stopping payment generation", loan.Name, conf.Common.DeathDate),
				zap.String("op", "config.GetAmortizationSchedule"),
			)
			break
		}

		// Check if we've passed the death date
		pastDeath, err := DateBeforeDate(conf.Common.DeathDate, currentMonth)
		if err != nil {
			return err
		}
		if pastDeath {
			logger.Debug(fmt.Sprintf("Loan %s passed death date %s at %s, stopping payment generation", loan.Name, conf.Common.DeathDate, currentMonth),
				zap.String("op", "config.GetAmortizationSchedule"),
			)
			break
		}

		var currentPayment Payment

		// Calculate refundable escrow
		january, err := CheckMonth(currentMonth, "01")
		if err != nil {
			return err
		}
		if january {
			currentPayment.RefundableEscrow = 0.00
		} else {
			currentPayment.RefundableEscrow = loan.AmortizationSchedule[previousMonth].RefundableEscrow + loan.Escrow
		}

		if loan.EarlyPayoffDate == currentMonth {
			if loan.SellProperty {
				currentPayment.Payment = loan.AmortizationSchedule[previousMonth].RemainingPrincipal - loan.SellPrice + loan.SellCostsNet
				logger.Debug(fmt.Sprintf("%s: paying off asset %s for %.2f and selling for %.2f with %.2f selling costs", loan.EarlyPayoffDate, loan.Name, loan.AmortizationSchedule[previousMonth].RemainingPrincipal, loan.SellPrice, loan.SellCostsNet),
					zap.String("op", "config.GetAmortizationSchedule"),
				)
				loan.AmortizationSchedule[currentMonth] = currentPayment
			} else {
				currentPayment.Payment = loan.AmortizationSchedule[previousMonth].RemainingPrincipal - currentPayment.RefundableEscrow
				logger.Debug(fmt.Sprintf("%s: paying off asset %s for %.2f", loan.EarlyPayoffDate, loan.Name, loan.AmortizationSchedule[previousMonth].RemainingPrincipal),
					zap.String("op", "config.GetAmortizationSchedule"),
				)
				loan.AmortizationSchedule[currentMonth] = currentPayment
				// Since we paid off the loan but did not sell the asset we will
				// extrapolate the escrow to be paid on Decembers.
				for {
					if currentMonth == conf.Common.DeathDate {
						break
					}
					december, err := CheckMonth(currentMonth, "12")
					if err != nil {
						return err
					}
					if december {
						var escrowPayment Payment
						escrowPayment.Payment = loan.Escrow * 12
						loan.AmortizationSchedule[currentMonth] = escrowPayment
					}
					previousMonth = currentMonth
					currentMonth, err = OffsetDate(currentMonth, DateTimeLayout, 1)
					if err != nil {
						return err
					}
				}
			}
			break
		} else {
			// Check for extra principal
			extraPrincipal, err := loan.ExtraPrincipal(logger, currentMonth)
			if err != nil {
				return err
			}

			currentPayment.Payment = loanPayment + loan.Escrow + extraPrincipal
			currentPayment.Interest = loan.AmortizationSchedule[previousMonth].RemainingPrincipal * loan.InterestRate / (100.0 * 12.0)
			currentPayment.Principal = loanPayment - currentPayment.Interest + extraPrincipal

			// Ensure we do not overpay on extra principal
			if Round(currentPayment.Principal-loan.AmortizationSchedule[previousMonth].RemainingPrincipal) < extraPrincipal &&
				Round(currentPayment.Principal-loan.AmortizationSchedule[previousMonth].RemainingPrincipal) > 0 {
				// We could pay off the loan by paying a portion, but not all of, the
				// extra principal.
				extraPrincipal = currentPayment.Principal - loan.AmortizationSchedule[previousMonth].RemainingPrincipal
				currentPayment.Payment = loanPayment + loan.Escrow + extraPrincipal
				currentPayment.Interest = loan.AmortizationSchedule[previousMonth].RemainingPrincipal * loan.InterestRate / (100.0 * 12.0)
				logger.Debug(fmt.Sprintf("%s: adjusting extraPrincipal to %.2f to prevent overpayment for loan %s", currentMonth, extraPrincipal, loan.Name),
					zap.String("op", "config.GetAmortizationSchedule"),
				)
				currentPayment.Principal = loanPayment - currentPayment.Interest + extraPrincipal
			} else if Round(currentPayment.Principal-loan.AmortizationSchedule[previousMonth].RemainingPrincipal) > extraPrincipal {
				// In this case we should not be paying any extra principal; the
				// payment is actually liable to be reduced; adjust extraPrincipal to
				// be the appropriate non-positive value to make this adjustment.
				extraPrincipal = loan.AmortizationSchedule[previousMonth].RemainingPrincipal - (currentPayment.Principal - extraPrincipal)
				currentPayment.Payment = loanPayment + loan.Escrow + extraPrincipal
				currentPayment.Interest = loan.AmortizationSchedule[previousMonth].RemainingPrincipal * loan.InterestRate / (100.0 * 12.0)
				currentPayment.Principal = loanPayment - currentPayment.Interest + extraPrincipal
				logger.Debug(fmt.Sprintf("%s: adjusting extraPrincipal to %.2f to prevent overpayment for loan %s", currentMonth, extraPrincipal, loan.Name),
					zap.String("op", "config.GetAmortizationSchedule"),
				)
			}

			if month == loan.Term || Round(loan.AmortizationSchedule[previousMonth].RemainingPrincipal-currentPayment.Principal) == 0 {
				// We will get machine error otherwise so just set to 0.
				currentPayment.RemainingPrincipal = 0.00
				december, err := CheckMonth(currentMonth, "12")
				if err != nil {
					return err
				}
				if !december {
					// Incorporate the expected escrow refund; the RedunableEscrow value
					// tracks the refundable amount for early payoffs so we need to
					// reduce further by an escrow payment. Note that here we assume that
					// if a loan matures naturally then escrow will be applied that year
					// on december; this is not the assumption we use for early payoffs.
					currentPayment.Payment -= currentPayment.RefundableEscrow + loan.Escrow
				}
			} else {
				currentPayment.RemainingPrincipal = loan.AmortizationSchedule[previousMonth].RemainingPrincipal - currentPayment.Principal
			}
			if loan.MortgageInsuranceCutoff > 0 {
				if currentPayment.RemainingPrincipal/loan.Principal <= loan.MortgageInsuranceCutoff/100.0 {
					currentPayment.Payment -= loan.MortgageInsurance
				}
			}
			loan.AmortizationSchedule[currentMonth] = currentPayment
			// Since the loan matured we will extrapolate the escrow to be paid on
			// Decembers.
			if month == loan.Term || Round(loan.AmortizationSchedule[previousMonth].RemainingPrincipal-currentPayment.Principal) == 0 {
				for {
					if currentMonth == conf.Common.DeathDate {
						break
					}
					december, err := CheckMonth(currentMonth, "12")
					if err != nil {
						return err
					}
					if december && loan.Escrow > 0 && month != loan.Term {
						var escrowPayment Payment
						escrowPayment.Payment = loan.Escrow * 12
						loan.AmortizationSchedule[currentMonth] = escrowPayment
					}
					previousMonth = currentMonth
					currentMonth, err = OffsetDate(currentMonth, DateTimeLayout, 1)
					month = 0
					if err != nil {
						return err
					}
				}
				break
			}
		}
		previousMonth = currentMonth
		currentMonth, err = OffsetDate(currentMonth, DateTimeLayout, 1)
		if err != nil {
			return err
		}
	}

	return nil
}

// ExtraPrincipal returns an extra principal payment, if present, or 0
func (loan *Loan) ExtraPrincipal(logger *zap.Logger, date string) (float64, error) {
	amount := 0.00
	dateT, err := time.Parse(DateTimeLayout, date)
	if err != nil {
		return amount, err
	}
	for _, event := range loan.ExtraPrincipalPayments {
		for _, paymentDate := range event.DateList {
			if dateT.Equal(paymentDate) {
				logger.Debug(fmt.Sprintf("%s: applying extra principal payment %.2f for loan %s", date, event.Amount, loan.Name),
					zap.String("op", "config.ExtraPrincipal"),
				)
				amount += event.Amount
				break
			}
		}
	}
	return amount, nil
}

// CheckEarlyPayoffThreshold checks for whether or not it is time to payoff a
// loan early based on an optionally-configured threshold. Note that escrow
// refunds are not factored into the threshold comparison because in reality
// those can take some time to process (even though the simulation acts as
// though an escrow refund is processed immediately).
func (loan *Loan) CheckEarlyPayoffThreshold(logger *zap.Logger, currentMonth string, deathDate string, balance float64) (string, error) {
	var note string
	started, err := DateBeforeDate(loan.StartDate, currentMonth)
	if err != nil {
		return note, err
	}
	if loan.EarlyPayoffThreshold > 0 && started {
		previousMonth, err := OffsetDate(currentMonth, DateTimeLayout, -1)
		if err != nil {
			return note, err
		}
		if Round(balance-loan.AmortizationSchedule[previousMonth].RemainingPrincipal) >= loan.EarlyPayoffThreshold {
			logger.Debug(fmt.Sprintf("%s: based on threshold paying off asset %s for %.2f", currentMonth, loan.Name, loan.AmortizationSchedule[previousMonth].RemainingPrincipal),
				zap.String("op", "config.CheckEarlyPayoffThreshold"),
			)
			var finalPayment Payment
			if loan.SellProperty {
				finalPayment.Payment = loan.AmortizationSchedule[previousMonth].RemainingPrincipal - loan.SellPrice + loan.SellCostsNet
				loan.AmortizationSchedule[currentMonth] = finalPayment
				note = fmt.Sprintf("paying off asset %s for %.2f and selling for %.2f with %.2f selling costs", loan.Name, loan.AmortizationSchedule[previousMonth].RemainingPrincipal, loan.SellPrice, loan.SellCostsNet)
				logger.Debug(fmt.Sprintf("%s: selling asset %s for %.2f with %.2f selling costs", currentMonth, loan.Name, loan.SellPrice, loan.SellCostsNet),
					zap.String("op", "config.CheckEarlyPayoffThreshold"),
				)
			} else {
				note = fmt.Sprintf("paying off asset %s for %.2f", loan.Name, loan.AmortizationSchedule[previousMonth].RemainingPrincipal)
				finalPayment.Payment = loan.AmortizationSchedule[previousMonth].RemainingPrincipal - loan.AmortizationSchedule[currentMonth].RefundableEscrow
				loan.AmortizationSchedule[currentMonth] = finalPayment
			}

			// Modify the remainder of the amortization schedule to null out payments
			// and if we did not sell the property and did declare escrow then handle
			// converting that into equivalent annual payments.
			loan.EarlyPayoffThreshold = 0
			for {
				if currentMonth == deathDate {
					break
				}
				previousMonth = currentMonth
				currentMonth, err = OffsetDate(currentMonth, DateTimeLayout, 1)
				if err != nil {
					return note, err
				}
				december, err := CheckMonth(currentMonth, "12")
				if err != nil {
					return note, err
				}
				if december && loan.Escrow > 0 {
					var escrowPayment Payment
					escrowPayment.Payment = loan.Escrow * 12
					loan.AmortizationSchedule[currentMonth] = escrowPayment
				} else {
					delete(loan.AmortizationSchedule, currentMonth)
				}
			}

		}
	}
	return note, nil
}

// OffsetDate returns the string-formatted date offset by the given number of
// months relative to the given date.
func OffsetDate(date, layout string, months int) (string, error) {
	t, err := time.Parse(layout, date)
	if err != nil {
		return date, err
	}
	return t.AddDate(0, months, 0).Format(layout), nil
}

// CheckMonth identifies whether a given date is in the month indicated by the
// numeric representation e.g. 01 = January and 12 = December.
func CheckMonth(date string, month string) (bool, error) {
	dateT, err := time.Parse(DateTimeLayout, date)
	if err != nil {
		return false, err
	}
	if dateT.Format("01") == month {
		return true, nil
	} else {
		return false, nil
	}
}

// DateBeforeDate returns true if firstDate is strictly before secondDate.
func DateBeforeDate(firstDate string, secondDate string) (bool, error) {
	firstDateT, err := time.Parse(DateTimeLayout, firstDate)
	if err != nil {
		return false, err
	}
	secondDateT, err := time.Parse(DateTimeLayout, secondDate)
	if err != nil {
		return false, err
	}
	return firstDateT.Before(secondDateT), nil
}
