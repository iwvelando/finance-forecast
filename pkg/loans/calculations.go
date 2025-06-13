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
}

// AmortizationScheduleGenerator provides utilities for generating loan amortization schedules
type AmortizationScheduleGenerator struct {
	logger *zap.Logger
}

// NewAmortizationScheduleGenerator creates a new generator instance
func NewAmortizationScheduleGenerator(logger *zap.Logger) *AmortizationScheduleGenerator {
	return &AmortizationScheduleGenerator{logger: logger}
}

// GenerateSchedule creates a complete amortization schedule for a loan
func (g *AmortizationScheduleGenerator) GenerateSchedule(loan *LoanConfig, deathDate string) (map[string]Payment, error) {
	schedule := make(map[string]Payment)

	// Calculate basic loan parameters
	monthlyPayment := CalculateMonthlyPayment(loan.Principal, loan.DownPayment, loan.InterestRate, loan.Term)

	// Handle first payment
	var firstPayment Payment
	extraPrincipal := g.calculateExtraPrincipal(loan, loan.StartDate)
	firstPayment.Payment = monthlyPayment + loan.Escrow + loan.DownPayment + extraPrincipal
	firstPayment.Interest = CalculateInterestPayment(loan.Principal-loan.DownPayment, loan.InterestRate)
	firstPayment.Principal = monthlyPayment - firstPayment.Interest + extraPrincipal
	firstPayment.RemainingPrincipal = (loan.Principal - loan.DownPayment) - firstPayment.Principal
	firstPayment.RefundableEscrow = loan.Escrow
	schedule[loan.StartDate] = firstPayment

	// Generate remaining schedule
	previousMonth := loan.StartDate
	for month := 2; month <= loan.Term; month++ {
		currentMonth, err := datetime.OffsetDate(previousMonth, datetime.DateTimeLayout, 1)
		if err != nil {
			return nil, err
		}

		// Check death date
		if currentMonth == deathDate {
			break
		}

		pastDeath, err := datetime.DateBeforeDate(deathDate, currentMonth)
		if err != nil {
			return nil, err
		}
		if pastDeath {
			break
		}

		payment := g.generateMonthlyPayment(loan, schedule, previousMonth, currentMonth, monthlyPayment)
		schedule[currentMonth] = payment

		// Check if loan is paid off
		if mathutil.Round(payment.RemainingPrincipal) <= 0 {
			break
		}

		previousMonth = currentMonth
	}

	return schedule, nil
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

func (g *AmortizationScheduleGenerator) calculateExtraPrincipal(loan *LoanConfig, date string) float64 {
	amount := 0.00

	for _, event := range loan.ExtraPrincipalPayments {
		for _, eventDate := range event.DateList {
			if eventDate == date {
				amount += event.Amount
			}
		}
	}

	return amount
}

func (g *AmortizationScheduleGenerator) generateMonthlyPayment(loan *LoanConfig, schedule map[string]Payment,
	previousMonth, currentMonth string, monthlyPayment float64) Payment {

	var payment Payment
	previousPayment := schedule[previousMonth]

	// Handle escrow
	january, _ := datetime.CheckMonth(currentMonth, "01")
	if january {
		payment.RefundableEscrow = 0.00
	} else {
		payment.RefundableEscrow = previousPayment.RefundableEscrow + loan.Escrow
	}

	// Handle early payoff
	if loan.EarlyPayoffDate == currentMonth {
		if loan.SellProperty {
			payment.Payment = previousPayment.RemainingPrincipal - loan.SellPrice + loan.SellCostsNet
		} else {
			payment.Payment = previousPayment.RemainingPrincipal - payment.RefundableEscrow
		}
		return payment
	}

	// Regular payment calculation
	extraPrincipal := g.calculateExtraPrincipal(loan, currentMonth)
	payment.Payment = monthlyPayment + loan.Escrow + extraPrincipal
	payment.Interest = CalculateInterestPayment(previousPayment.RemainingPrincipal, loan.InterestRate)
	payment.Principal = monthlyPayment - payment.Interest + extraPrincipal
	payment.RemainingPrincipal = previousPayment.RemainingPrincipal - payment.Principal // Handle mortgage insurance
	if loan.MortgageInsuranceCutoff > 0 {
		loanToValue := payment.RemainingPrincipal / (loan.Principal - loan.DownPayment)
		if loanToValue*constants.PercentageMultiplier > loan.MortgageInsuranceCutoff {
			payment.Payment += loan.MortgageInsurance
		}
	}

	return payment
}
