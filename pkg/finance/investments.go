package finance

import (
	"fmt"
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
	GetContributionForDate(date string) float64
	GetWithdrawalForDate(date string) float64
	GetWithdrawalPercentageForDate(date string) float64
	ContributionsFromCash() bool
}

// InvestmentState tracks the running value of an investment across simulation months.
type InvestmentState struct {
	CurrentValue float64
}

// InvestmentChange captures the computed deltas for a single investment in a given month.
type InvestmentChange struct {
	Name                 string
	Contribution         float64
	Withdrawal           float64
	WithdrawalPercentage float64
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
		states[inv.GetName()] = &InvestmentState{CurrentValue: inv.GetStartingValue()}
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
			state = &InvestmentState{CurrentValue: inv.GetStartingValue()}
			states[inv.GetName()] = state
		}

		previousValue := state.CurrentValue

		contribution := inv.GetContributionForDate(date)
		if contribution != 0 {
			state.CurrentValue += contribution
		}

		monthlyRate := percentToDecimal(inv.GetAnnualReturnRate()) / constants.MonthsPerYear
		growthBeforeTax := state.CurrentValue * monthlyRate

		tax := 0.0
		if growthBeforeTax > 0 && inv.GetTaxRate() > 0 {
			tax = growthBeforeTax * percentToDecimal(inv.GetTaxRate())
		}

		afterTaxGrowth := growthBeforeTax - tax
		if afterTaxGrowth != 0 {
			state.CurrentValue += afterTaxGrowth
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
		if withdrawal != 0 {
			state.CurrentValue -= withdrawal
		}

		netChange := state.CurrentValue - previousValue
		totalChange += netChange

		changes = append(changes, InvestmentChange{
			Name:                 inv.GetName(),
			Contribution:         contribution,
			Withdrawal:           withdrawal,
			WithdrawalPercentage: withdrawalPercent,
			Growth:               afterTaxGrowth,
			GrowthBeforeTax:      growthBeforeTax,
			Tax:                  tax,
			NetChange:            netChange,
			ContributionFromCash: inv.ContributionsFromCash(),
		})
	}

	return totalChange, changes, nil
}
