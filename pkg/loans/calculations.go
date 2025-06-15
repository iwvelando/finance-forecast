// Package loans provides common loan processing utilities.
package loans

import (
	"fmt"
	"math"

	"github.com/iwvelando/finance-forecast/pkg/constants"
	"github.com/iwvelando/finance-forecast/pkg/datetime"
	"github.com/iwvelando/finance-forecast/pkg/mathutil"
	"go.uber.org/zap"
)

// Payment holds the values for a given payment.
type Payment struct {
	Payment            float64
	Principal          float64
	Interest           float64
	RemainingPrincipal float64
	RefundableEscrow   float64
}

// CalculateMonthlyPayment calculates the monthly payment for a loan using the standard amortization formula.
func CalculateMonthlyPayment(principal, downPayment, annualInterestRate float64, termMonths int) float64 {
	if annualInterestRate == 0 {
		// For zero interest, simply divide the principal by term
		return (principal - downPayment) / float64(termMonths)
	}

	periodicInterestRate := annualInterestRate / (constants.PercentageMultiplier * constants.MonthsPerYear)
	power := math.Pow((1.00 + periodicInterestRate), float64(termMonths))
	discountFactor := (power - 1.00) / power
	return (principal - downPayment) * periodicInterestRate / discountFactor
}

// CalculateInterestPayment calculates the interest portion of a payment.
func CalculateInterestPayment(remainingPrincipal, annualInterestRate float64) float64 {
	return remainingPrincipal * annualInterestRate / (constants.PercentageMultiplier * constants.MonthsPerYear)
}

// CheckEarlyPayoffThreshold checks for whether or not it is time to payoff a
// loan early based on an optionally-configured threshold.
func CheckEarlyPayoffThreshold(logger *zap.Logger, loanName, startDate, currentMonth string, threshold float64,
	amortizationSchedule map[string]Payment, balance float64) (string, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	var note string

	started, err := datetime.DateBeforeDate(startDate, currentMonth)
	if err != nil {
		return note, err
	}

	if threshold > 0 && started {
		previousMonth, err := datetime.OffsetDate(currentMonth, datetime.DateTimeLayout, -1)
		if err != nil {
			return note, err
		}

		// Check if we have previous month data
		if previousPayment, exists := amortizationSchedule[previousMonth]; exists {
			cashSaved := balance - previousPayment.RemainingPrincipal
			if cashSaved > threshold {
				note = fmt.Sprintf("paying off asset %s for %.2f", loanName, previousPayment.RemainingPrincipal)
			}
		}
	}
	return note, nil
}

// Event represents an extra principal payment event
type Event struct {
	Name      string
	Amount    float64
	StartDate string
	EndDate   string
	Frequency int      // months
	DateList  []string // dates in YYYY-MM format
}

// LoanConfig represents loan configuration parameters
type LoanConfig struct {
	Name                    string
	StartDate               string
	Principal               float64
	InterestRate            float64
	Term                    int
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

// AmortizationScheduleGenerator provides utilities for generating loan amortization schedules
type AmortizationScheduleGenerator struct {
	logger *zap.Logger
}

// NewAmortizationScheduleGenerator creates a new generator instance
func NewAmortizationScheduleGenerator(logger *zap.Logger) *AmortizationScheduleGenerator {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &AmortizationScheduleGenerator{logger: logger}
}

// GenerateSchedule creates a complete amortization schedule for a loan
func (g *AmortizationScheduleGenerator) GenerateSchedule(loan *LoanConfig, deathDate string) (map[string]Payment, error) {
	schedule := make(map[string]Payment)

	// Calculate basic loan parameters
	monthlyPayment := CalculateMonthlyPayment(loan.Principal, loan.DownPayment, loan.InterestRate, loan.Term)

	// Handle first payment
	var firstPayment Payment
	extraPrincipal := 0.0
	for _, event := range loan.ExtraPrincipalPayments {
		for _, eventDate := range event.DateList {
			if eventDate == loan.StartDate {
				g.logger.Debug(fmt.Sprintf("%s: applying extra principal payment %.2f for loan %s",
					loan.StartDate, event.Amount, loan.Name),
					zap.String("op", "loans.GenerateSchedule"),
				)
				extraPrincipal += event.Amount
			}
		}
	}

	firstPayment.Payment = monthlyPayment + loan.Escrow + loan.DownPayment + extraPrincipal
	firstPayment.Interest = CalculateInterestPayment(loan.Principal-loan.DownPayment, loan.InterestRate)
	firstPayment.Principal = monthlyPayment - firstPayment.Interest + extraPrincipal
	firstPayment.RemainingPrincipal = (loan.Principal - loan.DownPayment) - firstPayment.Principal
	firstPayment.RefundableEscrow = loan.Escrow
	schedule[loan.StartDate] = firstPayment

	// Iterate over the remainder of the term.
	previousMonth := loan.StartDate
	currentMonth, err := datetime.OffsetDate(previousMonth, datetime.DateTimeLayout, 1)
	if err != nil {
		return nil, err
	}

	for month := 2; month <= loan.Term; month++ {
		// Check if we've reached or passed the death date
		if currentMonth == deathDate {
			g.logger.Debug(fmt.Sprintf("Loan %s reached death date %s, stopping payment generation",
				loan.Name, deathDate),
				zap.String("op", "loans.GenerateSchedule"),
			)
			break
		}

		// Check if we've passed the death date
		pastDeath, err := datetime.DateBeforeDate(deathDate, currentMonth)
		if err != nil {
			return nil, err
		}
		if pastDeath {
			g.logger.Debug(fmt.Sprintf("Loan %s passed death date %s at %s, stopping payment generation",
				loan.Name, deathDate, currentMonth),
				zap.String("op", "loans.GenerateSchedule"),
			)
			break
		}

		var currentPayment Payment

		// Calculate refundable escrow
		january, err := datetime.CheckMonth(currentMonth, "01")
		if err != nil {
			return nil, err
		}
		if january {
			currentPayment.RefundableEscrow = 0.00
		} else {
			currentPayment.RefundableEscrow = schedule[previousMonth].RefundableEscrow + loan.Escrow
		}

		if loan.EarlyPayoffDate == currentMonth {
			if loan.SellProperty {
				currentPayment.Payment = schedule[previousMonth].RemainingPrincipal - loan.SellPrice + loan.SellCostsNet
				g.logger.Debug(fmt.Sprintf("%s: paying off asset %s for %.2f and selling for %.2f with %.2f selling costs",
					loan.EarlyPayoffDate, loan.Name, schedule[previousMonth].RemainingPrincipal,
					loan.SellPrice, loan.SellCostsNet),
					zap.String("op", "loans.GenerateSchedule"),
				)
				schedule[currentMonth] = currentPayment
			} else {
				currentPayment.Payment = schedule[previousMonth].RemainingPrincipal - currentPayment.RefundableEscrow
				g.logger.Debug(fmt.Sprintf("%s: paying off asset %s for %.2f",
					loan.EarlyPayoffDate, loan.Name, schedule[previousMonth].RemainingPrincipal),
					zap.String("op", "loans.GenerateSchedule"),
				)
				schedule[currentMonth] = currentPayment
				// Since we paid off the loan but did not sell the asset we will
				// extrapolate the escrow to be paid on Decembers.
				for currentMonth != deathDate {
					december, err := datetime.CheckMonth(currentMonth, "12")
					if err != nil {
						return nil, err
					}
					if december {
						var escrowPayment Payment
						escrowPayment.Payment = loan.Escrow * 12
						schedule[currentMonth] = escrowPayment
					}
					currentMonth, err = datetime.OffsetDate(currentMonth, datetime.DateTimeLayout, 1)
					if err != nil {
						return nil, err
					}
				}
			}
			break
		} else {
			// Check for extra principal using the advanced calculation with overpayment prevention
			var loanEvents []Event
			loanEvents = append(loanEvents, loan.ExtraPrincipalPayments...)

			extraPrincipal, err := CalculateExtraPrincipalWithOverpaymentPrevention(
				g.logger, loanEvents, currentMonth, monthlyPayment,
				schedule[previousMonth].RemainingPrincipal, loan.InterestRate, loan.Name)
			if err != nil {
				return nil, err
			}

			currentPayment.Payment = monthlyPayment + loan.Escrow + extraPrincipal
			currentPayment.Interest = CalculateInterestPayment(schedule[previousMonth].RemainingPrincipal, loan.InterestRate)
			currentPayment.Principal = monthlyPayment - currentPayment.Interest + extraPrincipal

			if month == loan.Term || mathutil.Round(schedule[previousMonth].RemainingPrincipal-currentPayment.Principal) == 0 {
				// We will get machine error otherwise so just set to 0.
				currentPayment.RemainingPrincipal = 0.00
				december, err := datetime.CheckMonth(currentMonth, "12")
				if err != nil {
					return nil, err
				}
				if !december {
					// Incorporate the expected escrow refund; the RefundableEscrow value
					// tracks the refundable amount for early payoffs so we need to
					// reduce further by an escrow payment. Note that here we assume that
					// if a loan matures naturally then escrow will be applied that year
					// on december; this is not the assumption we use for early payoffs.
					currentPayment.Payment = currentPayment.Payment - currentPayment.RefundableEscrow - loan.Escrow
				}
			} else {
				currentPayment.RemainingPrincipal = schedule[previousMonth].RemainingPrincipal - currentPayment.Principal
			}
			if loan.MortgageInsuranceCutoff > 0 {
				if currentPayment.RemainingPrincipal/loan.Principal <= loan.MortgageInsuranceCutoff/100.0 {
					currentPayment.Payment -= loan.MortgageInsurance
				}
			}
			schedule[currentMonth] = currentPayment
			// Since the loan matured we will extrapolate the escrow to be paid on
			// Decembers.
			if month == loan.Term || mathutil.Round(schedule[previousMonth].RemainingPrincipal-currentPayment.Principal) == 0 {
				for currentMonth != deathDate {
					december, err := datetime.CheckMonth(currentMonth, "12")
					if err != nil {
						return nil, err
					}
					if december && loan.Escrow > 0 && month != loan.Term {
						var escrowPayment Payment
						escrowPayment.Payment = loan.Escrow * 12
						schedule[currentMonth] = escrowPayment
					}
					currentMonth, err = datetime.OffsetDate(currentMonth, datetime.DateTimeLayout, 1)
					if err != nil {
						return nil, err
					}
				}
				break
			}
		}
		previousMonth = currentMonth
		currentMonth, err = datetime.OffsetDate(currentMonth, datetime.DateTimeLayout, 1)
		if err != nil {
			return nil, err
		}
	}

	return schedule, nil
}

// CheckEarlyPayoffThresholdAndUpdate checks if a loan meets early payoff threshold and updates the schedule if needed
func (g *AmortizationScheduleGenerator) CheckEarlyPayoffThresholdAndUpdate(
	loan *LoanConfig,
	currentMonth string,
	deathDate string,
	balance float64,
	schedule map[string]Payment) (string, error) {

	var note string
	started, err := datetime.DateBeforeDate(loan.StartDate, currentMonth)
	if err != nil {
		return note, err
	}

	if loan.EarlyPayoffThreshold > 0 && started {
		previousMonth, err := datetime.OffsetDate(currentMonth, datetime.DateTimeLayout, -1)
		if err != nil {
			return note, err
		}
		if mathutil.Round(balance-schedule[previousMonth].RemainingPrincipal) >= loan.EarlyPayoffThreshold {
			g.logger.Debug(fmt.Sprintf("%s: based on threshold paying off asset %s for %.2f",
				currentMonth, loan.Name, schedule[previousMonth].RemainingPrincipal),
				zap.String("op", "loans.CheckEarlyPayoffThresholdAndUpdate"),
			)
			var finalPayment Payment
			if loan.SellProperty {
				finalPayment.Payment = schedule[previousMonth].RemainingPrincipal - loan.SellPrice + loan.SellCostsNet
				schedule[currentMonth] = finalPayment
				note = fmt.Sprintf("paying off asset %s for %.2f and selling for %.2f with %.2f selling costs",
					loan.Name, schedule[previousMonth].RemainingPrincipal, loan.SellPrice, loan.SellCostsNet)
				g.logger.Debug(fmt.Sprintf("%s: selling asset %s for %.2f with %.2f selling costs",
					currentMonth, loan.Name, loan.SellPrice, loan.SellCostsNet),
					zap.String("op", "loans.CheckEarlyPayoffThresholdAndUpdate"),
				)
			} else {
				note = fmt.Sprintf("paying off asset %s for %.2f", loan.Name, schedule[previousMonth].RemainingPrincipal)
				finalPayment.Payment = schedule[previousMonth].RemainingPrincipal - schedule[currentMonth].RefundableEscrow
				schedule[currentMonth] = finalPayment
			}

			// Modify the remainder of the amortization schedule to null out payments
			// and if we did not sell the property and did declare escrow then handle
			// converting that into equivalent annual payments.
			loan.EarlyPayoffThreshold = 0
			for currentMonth != deathDate {
				currentMonth, err = datetime.OffsetDate(currentMonth, datetime.DateTimeLayout, 1)
				if err != nil {
					return note, err
				}
				december, err := datetime.CheckMonth(currentMonth, "12")
				if err != nil {
					return note, err
				}
				if december && loan.Escrow > 0 {
					var escrowPayment Payment
					escrowPayment.Payment = loan.Escrow * 12
					schedule[currentMonth] = escrowPayment
				} else {
					delete(schedule, currentMonth)
				}
			}
		}
	}
	return note, nil
}

// CalculateExtraPrincipal calculates the total extra principal payment for a given date
func CalculateExtraPrincipal(extraPrincipalPayments []Event, date string) float64 {
	amount := 0.00

	for _, event := range extraPrincipalPayments {
		for _, eventDate := range event.DateList {
			if eventDate == date {
				amount += event.Amount
			}
		}
	}

	return amount
}

// CalculateExtraPrincipalWithLogging calculates the total extra principal payment for a given date and logs the result if > 0
func (g *AmortizationScheduleGenerator) CalculateExtraPrincipalWithLogging(extraPrincipalPayments []Event, date string, loanName string) float64 {
	amount := CalculateExtraPrincipal(extraPrincipalPayments, date)
	if amount > 0 {
		g.logger.Debug(fmt.Sprintf("%s: applying extra principal payment %.2f for loan %s", date, amount, loanName),
			zap.String("op", "loans.CalculateExtraPrincipalWithLogging"),
		)
	}
	return amount
}

// CalculateExtraPrincipalWithOverpaymentPrevention calculates extra principal with overpayment prevention
func CalculateExtraPrincipalWithOverpaymentPrevention(
	logger *zap.Logger, events []Event, date string, monthlyPayment, currentBalance, interestRate float64, loanName string,
) (float64, error) {
	totalExtra := CalculateExtraPrincipal(events, date)

	// Prevent overpayment by capping extra payment to current balance
	if totalExtra > currentBalance {
		logger.Debug("Capping extra principal payment to prevent overpayment",
			zap.String("date", date),
			zap.String("loan", loanName),
			zap.Float64("requested", totalExtra),
			zap.Float64("capped_to_balance", currentBalance))
		return currentBalance, nil
	}

	return totalExtra, nil
}

// calculateExtraPrincipal was replaced with direct use of CalculateExtraPrincipal

// generateMonthlyPayment was integrated directly into the GenerateSchedule method
