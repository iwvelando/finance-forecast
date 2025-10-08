package finance

import (
	"fmt"
	"math"
	"time"

	"github.com/iwvelando/finance-forecast/pkg/constants"
	"go.uber.org/zap"
)

const percentDivisor = 100.0

func percentToDecimal(percent float64) float64 {
	return percent / percentDivisor
}

// Investment represents an investment account with contribution and withdrawal schedules.
type Investment interface {
	GetName() string
	GetStartingValue() float64
	GetAnnualReturnRate() float64
	GetTaxRate() float64
	GetWithdrawalTaxRate() float64
	GetContributionForDate(date string) float64
	GetWithdrawalForDate(date string) float64
	GetWithdrawalPercentageForDate(date string) float64
	ContributionsFromCash() bool
}

// InvestmentState tracks the running value of an investment across simulation months.
type InvestmentState struct {
	CurrentValue     float64
	PrincipalBalance float64
	GrowthBalance    float64
}

// InvestmentChange captures the computed deltas for a single investment in a given month.
type InvestmentChange struct {
	Name                 string
	Contribution         float64
	Withdrawal           float64
	WithdrawalPercentage float64
	WithdrawalTax        float64
	WithdrawalFromGrowth float64
	WithdrawalFromBasis  float64
	Growth               float64
	GrowthBeforeTax      float64
	Tax                  float64
	NetChange            float64
	ContributionFromCash bool
}

// InvestmentProcessor handles monthly investment computations.
type InvestmentProcessor struct {
	logger *zap.Logger
}

// NewInvestmentProcessor creates a processor for investment calculations.
func NewInvestmentProcessor(logger *zap.Logger) *InvestmentProcessor {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &InvestmentProcessor{logger: logger}
}

// InitializeStates creates initial investment states using the provided investments.
func (ip *InvestmentProcessor) InitializeStates(investments []Investment) map[string]*InvestmentState {
	states := make(map[string]*InvestmentState)
	for _, inv := range investments {
		if inv == nil {
			continue
		}
		states[inv.GetName()] = &InvestmentState{
			CurrentValue:     inv.GetStartingValue(),
			PrincipalBalance: inv.GetStartingValue(),
			GrowthBalance:    0,
		}
	}
	return states
}

// ProcessInvestmentsForDate processes all investments for a given date and returns the total change.
func (ip *InvestmentProcessor) ProcessInvestmentsForDate(date string, investments []Investment, layout string, states map[string]*InvestmentState) (float64, []InvestmentChange, error) {
	if layout == "" {
		return 0, nil, fmt.Errorf("layout cannot be empty")
	}
	if date == "" {
		return 0, nil, fmt.Errorf("date cannot be empty")
	}

	if _, err := time.Parse(layout, date); err != nil {
		return 0, nil, fmt.Errorf("failed to parse date %s with layout %s: %w", date, layout, err)
	}

	if len(investments) == 0 {
		return 0, nil, nil
	}

	if states == nil {
		states = ip.InitializeStates(investments)
	}

	totalChange := 0.0
	var changes []InvestmentChange

	for _, inv := range investments {
		if inv == nil {
			ip.logger.Warn("Skipping nil investment", zap.String("date", date))
			continue
		}

		state, ok := states[inv.GetName()]
		if !ok {
			state = &InvestmentState{
				CurrentValue:     inv.GetStartingValue(),
				PrincipalBalance: inv.GetStartingValue(),
			}
			states[inv.GetName()] = state
		}
		if state.PrincipalBalance == 0 && inv.GetStartingValue() != 0 && state.CurrentValue == inv.GetStartingValue() {
			state.PrincipalBalance = inv.GetStartingValue()
		}

		previousValue := state.CurrentValue

		contribution := inv.GetContributionForDate(date)
		if contribution != 0 {
			state.CurrentValue += contribution
			state.PrincipalBalance += contribution
			if state.PrincipalBalance < 0 {
				state.PrincipalBalance = 0
			}
		}

		monthlyRate := percentToDecimal(inv.GetAnnualReturnRate()) / constants.MonthsPerYear
		growthBeforeTax := state.CurrentValue * monthlyRate

		tax := 0.0
		// Taxes are applied monthly to the investment growth and are deducted immediately from the account balance.
		// This means that each month's growth is taxed before being added to the account, affecting compounding.
		if growthBeforeTax > 0 && inv.GetTaxRate() > 0 {
			tax = growthBeforeTax * percentToDecimal(inv.GetTaxRate())
		}

		afterTaxGrowth := growthBeforeTax - tax
		if afterTaxGrowth != 0 {
			state.CurrentValue += afterTaxGrowth
			state.GrowthBalance += afterTaxGrowth
			if state.GrowthBalance < 0 {
				deficit := -state.GrowthBalance
				state.GrowthBalance = 0
				state.PrincipalBalance -= deficit
				if state.PrincipalBalance < 0 {
					state.PrincipalBalance = 0
				}
			}
		}

		withdrawal := inv.GetWithdrawalForDate(date)
		withdrawalPercent := inv.GetWithdrawalPercentageForDate(date)
		if withdrawalPercent != 0 {
			percentAmount := state.CurrentValue * percentToDecimal(withdrawalPercent)
			withdrawal += percentAmount
		}
		if withdrawal > state.CurrentValue {
			withdrawal = state.CurrentValue
		}
		if withdrawal < 0 {
			withdrawal = 0
		}

		withdrawalFromGrowth := 0.0
		withdrawalFromBasis := 0.0
		if withdrawal != 0 {
			availableGrowth := state.GrowthBalance
			if availableGrowth < 0 {
				availableGrowth = 0
			}
			if availableGrowth > 0 {
				if availableGrowth >= withdrawal {
					withdrawalFromGrowth = withdrawal
				} else {
					withdrawalFromGrowth = availableGrowth
				}
			}
			withdrawalFromBasis = withdrawal - withdrawalFromGrowth

			state.GrowthBalance -= withdrawalFromGrowth
			if state.GrowthBalance < 0 {
				withdrawalFromBasis += -state.GrowthBalance
				state.GrowthBalance = 0
			}
			state.PrincipalBalance -= withdrawalFromBasis
			if state.PrincipalBalance < 0 {
				state.PrincipalBalance = 0
			}

			state.CurrentValue -= withdrawal
		}

		withdrawalTax := 0.0
		if withdrawalFromGrowth > 0 {
			taxRate := inv.GetWithdrawalTaxRate()
			if taxRate > 0 {
				withdrawalTax = withdrawalFromGrowth * percentToDecimal(taxRate)
			}
		}
		if withdrawal != 0 {
			state.CurrentValue = math.Max(state.CurrentValue, 0)
		}

		netChange := state.CurrentValue - previousValue
		totalChange += netChange

		changes = append(changes, InvestmentChange{
			Name:                 inv.GetName(),
			Contribution:         contribution,
			Withdrawal:           withdrawal,
			WithdrawalPercentage: withdrawalPercent,
			WithdrawalTax:        withdrawalTax,
			WithdrawalFromGrowth: withdrawalFromGrowth,
			WithdrawalFromBasis:  withdrawalFromBasis,
			Growth:               afterTaxGrowth,
			GrowthBeforeTax:      growthBeforeTax,
			Tax:                  tax,
			NetChange:            netChange,
			ContributionFromCash: inv.ContributionsFromCash(),
		})
	}

	return totalChange, changes, nil
}
