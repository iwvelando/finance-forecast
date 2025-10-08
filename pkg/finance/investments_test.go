package finance

import (
	"math"
	"testing"

	"github.com/iwvelando/finance-forecast/pkg/constants"
	"go.uber.org/zap"
)

type stubInvestment struct {
	name          string
	startingValue float64
	annualRate    float64
	taxRate       float64
	withdrawalTax float64
	contributions map[string]float64
	withdrawals   map[string]float64
	withdrawalPct map[string]float64
	fromCash      bool
}

func (s stubInvestment) GetName() string {
	return s.name
}

func (s stubInvestment) GetStartingValue() float64 {
	return s.startingValue
}

func (s stubInvestment) GetAnnualReturnRate() float64 {
	return s.annualRate
}

func (s stubInvestment) GetTaxRate() float64 {
	return s.taxRate
}

func (s stubInvestment) GetWithdrawalTaxRate() float64 {
	return s.withdrawalTax
}

func (s stubInvestment) GetContributionForDate(date string) float64 {
	if s.contributions == nil {
		return 0
	}
	return s.contributions[date]
}

func (s stubInvestment) GetWithdrawalForDate(date string) float64 {
	if s.withdrawals == nil {
		return 0
	}
	return s.withdrawals[date]
}

func (s stubInvestment) GetWithdrawalPercentageForDate(date string) float64 {
	if s.withdrawalPct == nil {
		return 0
	}
	return s.withdrawalPct[date]
}

func (s stubInvestment) ContributionsFromCash() bool {
	return s.fromCash
}

func TestInvestmentProcessorInitializeStates(t *testing.T) {
	logger := zap.NewNop()
	processor := NewInvestmentProcessor(logger)

	investments := []Investment{
		stubInvestment{name: "A", startingValue: 500},
		stubInvestment{name: "B", startingValue: 250.5},
	}

	states := processor.InitializeStates(investments)
	if len(states) != 2 {
		t.Fatalf("expected 2 investment states, got %d", len(states))
	}

	if states["A"].CurrentValue != 500 {
		t.Errorf("state for A = %.2f, want 500.00", states["A"].CurrentValue)
	}

	if math.Abs(states["B"].CurrentValue-250.5) > 1e-9 {
		t.Errorf("state for B = %.2f, want 250.50", states["B"].CurrentValue)
	}
}

func TestInvestmentProcessorProcessInvestmentsForDate_BasicGrowth(t *testing.T) {
	logger := zap.NewNop()
	processor := NewInvestmentProcessor(logger)

	inv := stubInvestment{
		name:          "Core Fund",
		startingValue: 1000,
		annualRate:    12, // 1% monthly
		contributions: map[string]float64{"2025-07": 100},
	}

	investments := []Investment{inv}
	states := processor.InitializeStates(investments)

	totalChange, changes, err := processor.ProcessInvestmentsForDate("2025-07", investments, constants.DateTimeLayout, states)
	if err != nil {
		t.Fatalf("ProcessInvestmentsForDate returned error: %v", err)
	}

	if len(changes) != 1 {
		t.Fatalf("expected 1 investment change entry, got %d", len(changes))
	}

	change := changes[0]

	expectedNetChange := 111.0 // 100 contribution + 11 growth
	if math.Abs(totalChange-expectedNetChange) > 1e-9 {
		t.Errorf("totalChange = %.2f, want %.2f", totalChange, expectedNetChange)
	}

	if math.Abs(change.NetChange-expectedNetChange) > 1e-9 {
		t.Errorf("NetChange = %.2f, want %.2f", change.NetChange, expectedNetChange)
	}

	if math.Abs(change.Contribution-100) > 1e-9 {
		t.Errorf("Contribution = %.2f, want 100", change.Contribution)
	}

	if math.Abs(change.Growth-11) > 1e-9 {
		t.Errorf("Growth = %.2f, want 11", change.Growth)
	}

	if math.Abs(change.GrowthBeforeTax-11) > 1e-9 {
		t.Errorf("GrowthBeforeTax = %.2f, want 11", change.GrowthBeforeTax)
	}

	if math.Abs(states["Core Fund"].CurrentValue-1111) > 1e-9 {
		t.Errorf("updated state = %.2f, want 1111", states["Core Fund"].CurrentValue)
	}

	if change.ContributionFromCash {
		t.Errorf("ContributionFromCash expected false by default")
	}
}

func TestInvestmentProcessorProcessInvestmentsForDate_WithTaxesAndWithdrawal(t *testing.T) {
	logger := zap.NewNop()
	processor := NewInvestmentProcessor(logger)

	inv := stubInvestment{
		name:          "Taxed Fund",
		startingValue: 1000,
		annualRate:    12,
		taxRate:       25,
		withdrawals:   map[string]float64{"2025-07": 50},
	}

	investments := []Investment{inv}
	states := processor.InitializeStates(investments)

	totalChange, changes, err := processor.ProcessInvestmentsForDate("2025-07", investments, constants.DateTimeLayout, states)
	if err != nil {
		t.Fatalf("ProcessInvestmentsForDate returned error: %v", err)
	}

	change := changes[0]

	expectedGrowth := 7.5 // 1% monthly growth = 10, minus 25% tax = 7.5
	expectedTax := 2.5
	expectedNetChange := expectedGrowth - 50 // withdrawal after growth

	if math.Abs(change.Growth-expectedGrowth) > 1e-9 {
		t.Errorf("Growth = %.2f, want %.2f", change.Growth, expectedGrowth)
	}

	if math.Abs(change.GrowthBeforeTax-10) > 1e-9 {
		t.Errorf("GrowthBeforeTax = %.2f, want %.2f", change.GrowthBeforeTax, 10.0)
	}

	if math.Abs(change.Tax-expectedTax) > 1e-9 {
		t.Errorf("Tax = %.2f, want %.2f", change.Tax, expectedTax)
	}

	if math.Abs(change.Withdrawal-50) > 1e-9 {
		t.Errorf("Withdrawal = %.2f, want 50", change.Withdrawal)
	}

	if math.Abs(change.NetChange-expectedNetChange) > 1e-9 {
		t.Errorf("NetChange = %.2f, want %.2f", change.NetChange, expectedNetChange)
	}

	if math.Abs(totalChange-expectedNetChange) > 1e-9 {
		t.Errorf("totalChange = %.2f, want %.2f", totalChange, expectedNetChange)
	}

	expectedValue := 957.5
	if math.Abs(states["Taxed Fund"].CurrentValue-expectedValue) > 1e-9 {
		t.Errorf("state.CurrentValue = %.2f, want %.2f", states["Taxed Fund"].CurrentValue, expectedValue)
	}
}

func TestInvestmentProcessorProcessInvestmentsForDate_WithdrawalTaxApplied(t *testing.T) {
	logger := zap.NewNop()
	processor := NewInvestmentProcessor(logger)

	inv := stubInvestment{
		name:          "Taxable Brokerage",
		startingValue: 1000,
		annualRate:    12,
		withdrawalTax: 25,
		withdrawals:   map[string]float64{"2025-07": 100},
		contributions: nil,
		withdrawalPct: nil,
		fromCash:      false,
	}

	investments := []Investment{inv}
	states := processor.InitializeStates(investments)

	totalChange, changes, err := processor.ProcessInvestmentsForDate("2025-07", investments, constants.DateTimeLayout, states)
	if err != nil {
		t.Fatalf("ProcessInvestmentsForDate returned error: %v", err)
	}

	if len(changes) != 1 {
		t.Fatalf("expected 1 investment change entry, got %d", len(changes))
	}

	change := changes[0]

	expectedGrowth := 10.0
	if math.Abs(change.Growth-expectedGrowth) > 1e-9 {
		t.Errorf("Growth = %.2f, want %.2f", change.Growth, expectedGrowth)
	}

	if math.Abs(change.Withdrawal-100) > 1e-9 {
		t.Errorf("Withdrawal = %.2f, want 100", change.Withdrawal)
	}

	if math.Abs(change.WithdrawalFromGrowth-expectedGrowth) > 1e-9 {
		t.Errorf("WithdrawalFromGrowth = %.2f, want %.2f", change.WithdrawalFromGrowth, expectedGrowth)
	}

	expectedBasisWithdrawal := 90.0
	if math.Abs(change.WithdrawalFromBasis-expectedBasisWithdrawal) > 1e-9 {
		t.Errorf("WithdrawalFromBasis = %.2f, want %.2f", change.WithdrawalFromBasis, expectedBasisWithdrawal)
	}

	expectedWithdrawalTax := 2.5
	if math.Abs(change.WithdrawalTax-expectedWithdrawalTax) > 1e-9 {
		t.Errorf("WithdrawalTax = %.2f, want %.2f", change.WithdrawalTax, expectedWithdrawalTax)
	}

	expectedNetChange := expectedGrowth - change.Withdrawal
	if math.Abs(change.NetChange-expectedNetChange) > 1e-9 {
		t.Errorf("NetChange = %.2f, want %.2f", change.NetChange, expectedNetChange)
	}

	if math.Abs(totalChange-expectedNetChange) > 1e-9 {
		t.Errorf("totalChange = %.2f, want %.2f", totalChange, expectedNetChange)
	}

	state := states["Taxable Brokerage"]
	if state == nil {
		t.Fatalf("state for Taxable Brokerage not found")
	}

	if math.Abs(state.CurrentValue-910.0) > 1e-9 {
		t.Errorf("state.CurrentValue = %.2f, want 910.00", state.CurrentValue)
	}

	if math.Abs(state.PrincipalBalance-910.0) > 1e-9 {
		t.Errorf("state.PrincipalBalance = %.2f, want 910.00", state.PrincipalBalance)
	}

	if state.GrowthBalance != 0 {
		t.Errorf("state.GrowthBalance = %.2f, want 0", state.GrowthBalance)
	}
}

func TestInvestmentProcessorProcessInvestmentsForDate_PercentageWithdrawal(t *testing.T) {
	logger := zap.NewNop()
	processor := NewInvestmentProcessor(logger)

	inv := stubInvestment{
		name:          "Safe Withdrawal",
		startingValue: 500000,
		annualRate:    6,
		withdrawalPct: map[string]float64{"2026-01": 4.0},
	}

	investments := []Investment{inv}
	states := processor.InitializeStates(investments)

	totalChange, changes, err := processor.ProcessInvestmentsForDate("2026-01", investments, constants.DateTimeLayout, states)
	if err != nil {
		t.Fatalf("ProcessInvestmentsForDate returned error: %v", err)
	}

	if len(changes) != 1 {
		t.Fatalf("expected 1 investment change entry, got %d", len(changes))
	}

	change := changes[0]

	monthlyRate := inv.annualRate / (constants.MonthsPerYear * 100.0)
	initial := inv.startingValue
	growth := initial * monthlyRate
	postGrowth := initial + growth
	expectedWithdrawal := postGrowth * (4.0 / 100.0)
	if math.Abs(change.Withdrawal-expectedWithdrawal) > 1e-6 {
		t.Fatalf("Withdrawal = %.2f, want %.2f", change.Withdrawal, expectedWithdrawal)
	}

	if math.Abs(change.WithdrawalPercentage-4.0) > 1e-9 {
		t.Fatalf("WithdrawalPercentage = %.2f, want 4.0", change.WithdrawalPercentage)
	}

	// After applying withdrawal, state should be reduced by withdrawal following growth
	expectedState := postGrowth - expectedWithdrawal

	if math.Abs(states["Safe Withdrawal"].CurrentValue-expectedState) > 1e-3 {
		t.Fatalf("state.CurrentValue = %.2f, want %.2f", states["Safe Withdrawal"].CurrentValue, expectedState)
	}

	if math.Abs(totalChange-(expectedState-initial)) > 1e-3 {
		t.Fatalf("totalChange = %.2f, want %.2f", totalChange, expectedState-initial)
	}
}

func TestInvestmentProcessorProcessInvestmentsForDate_FromCashFlag(t *testing.T) {
	logger := zap.NewNop()
	processor := NewInvestmentProcessor(logger)

	inv := stubInvestment{
		name:          "Traditional 401k",
		startingValue: 1000,
		annualRate:    6,
		contributions: map[string]float64{"2025-07": 200},
		fromCash:      true,
	}

	investments := []Investment{inv}
	states := processor.InitializeStates(investments)

	_, changes, err := processor.ProcessInvestmentsForDate("2025-07", investments, constants.DateTimeLayout, states)
	if err != nil {
		t.Fatalf("ProcessInvestmentsForDate returned error: %v", err)
	}

	if len(changes) != 1 {
		t.Fatalf("expected 1 investment change entry, got %d", len(changes))
	}

	if !changes[0].ContributionFromCash {
		t.Fatalf("expected ContributionFromCash to be true")
	}
}

func TestInvestmentProcessorProcessInvestmentsForDate_InvalidInput(t *testing.T) {
	logger := zap.NewNop()
	processor := NewInvestmentProcessor(logger)

	_, _, err := processor.ProcessInvestmentsForDate("2025-07", nil, "", nil)
	if err == nil {
		t.Fatalf("expected error for empty layout, got nil")
	}
}
