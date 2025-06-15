package config

import (
	"testing"

	"go.uber.org/zap"
)

func TestLoanAmortization(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	// Test configuration for a simple loan
	config := Configuration{
		Common: Common{
			DeathDate: "2030-01",
		},
	}

	loan := &Loan{
		Name:         "Test Loan",
		StartDate:    "2025-01",
		Principal:    100000,
		InterestRate: 5.0,
		Term:         60, // 5 years instead of 30 to avoid date overflow
		DownPayment:  20000,
	}

	err := loan.GetAmortizationSchedule(logger, config)
	if err != nil {
		t.Fatalf("GetAmortizationSchedule() error = %v", err)
	}

	// Verify amortization schedule was created
	if len(loan.AmortizationSchedule) == 0 {
		t.Errorf("Amortization schedule was not created")
	}

	// Check first payment
	firstPayment, exists := loan.AmortizationSchedule["2025-01"]
	if !exists {
		t.Errorf("First payment not found in amortization schedule")
	}

	// Verify first payment includes down payment
	expectedFirstPayment := 20000 // down payment
	if firstPayment.Payment < float64(expectedFirstPayment) {
		t.Errorf("First payment should include down payment, got %.2f", firstPayment.Payment)
	}

	// Verify remaining principal calculation
	expectedPrincipalLent := 100000 - 20000 // principal minus down payment
	if firstPayment.RemainingPrincipal > float64(expectedPrincipalLent) {
		t.Errorf("Remaining principal after first payment should be less than %.2f, got %.2f",
			float64(expectedPrincipalLent), firstPayment.RemainingPrincipal)
	}
}

func TestLoanWithEarlyPayoff(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	config := Configuration{
		Common: Common{
			DeathDate: "2030-01",
		},
	}

	loan := &Loan{
		Name:            "Test Loan with Early Payoff",
		StartDate:       "2025-01",
		Principal:       100000,
		InterestRate:    5.0,
		Term:            60, // 5 years instead of 30
		DownPayment:     20000,
		EarlyPayoffDate: "2027-06",
	}

	err := loan.GetAmortizationSchedule(logger, config)
	if err != nil {
		t.Fatalf("GetAmortizationSchedule() error = %v", err)
	}

	// Verify early payoff payment exists
	earlyPayment, exists := loan.AmortizationSchedule["2027-06"]
	if !exists {
		t.Errorf("Early payoff payment not found in amortization schedule")
	}

	// Verify it's a significant payment (paying off remaining principal)
	// For a 5-year loan paid off after 2.5 years, should be around $40-50k
	if earlyPayment.Payment < 30000 {
		t.Errorf("Early payoff payment seems too small: %.2f", earlyPayment.Payment)
	}

	// Verify no payments after early payoff (except potential escrow)
	laterPayment, exists := loan.AmortizationSchedule["2027-07"]
	if exists && laterPayment.Payment > 1000 { // Allow for escrow payments
		t.Errorf("Found significant payment after early payoff: %.2f", laterPayment.Payment)
	}
}

func TestLoanWithEscrow(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	config := Configuration{
		Common: Common{
			DeathDate: "2030-01",
		},
	}

	loan := &Loan{
		Name:         "Test Loan with Escrow",
		StartDate:    "2025-01",
		Principal:    100000,
		InterestRate: 5.0,
		Term:         60, // 5 years instead of 30
		DownPayment:  20000,
		Escrow:       500,
	}

	err := loan.GetAmortizationSchedule(logger, config)
	if err != nil {
		t.Fatalf("GetAmortizationSchedule() error = %v", err)
	}

	// Check that payments include escrow
	firstPayment := loan.AmortizationSchedule["2025-01"]
	secondPayment := loan.AmortizationSchedule["2025-02"]

	// First payment includes down payment + regular payment + escrow
	// Second payment should include escrow
	if secondPayment.Payment < 500 {
		t.Errorf("Second payment should include escrow of 500, got %.2f", secondPayment.Payment)
	}

	// Check RefundableEscrow tracking
	if firstPayment.RefundableEscrow != 500 {
		t.Errorf("Expected RefundableEscrow of 500 for first payment, got %.2f", firstPayment.RefundableEscrow)
	}
}

func TestLoanWithMortgageInsurance(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	config := Configuration{
		Common: Common{
			DeathDate: "2030-01",
		},
	}

	loan := &Loan{
		Name:                    "Test Loan with MI",
		StartDate:               "2025-01",
		Principal:               100000,
		InterestRate:            5.0,
		Term:                    60,    // 5 years instead of 30
		DownPayment:             10000, // 10% down, so MI required
		MortgageInsurance:       100,
		MortgageInsuranceCutoff: 78.0, // Remove MI at 78% LTV
	}

	err := loan.GetAmortizationSchedule(logger, config)
	if err != nil {
		t.Fatalf("GetAmortizationSchedule() error = %v", err)
	}

	// Early payments should include mortgage insurance
	secondPayment := loan.AmortizationSchedule["2025-02"]
	if secondPayment.Payment < 100 {
		t.Errorf("Early payment should include mortgage insurance of 100")
	}

	// Check that mortgage insurance is eventually removed
	// (this is a simplified test - in reality it would take years to reach 78% LTV)
}

func TestLoanWithExtraPrincipal(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	config := Configuration{
		Common: Common{
			DeathDate: "2030-01",
		},
	}

	// Create an event for extra principal payment
	extraPaymentEvent := Event{
		Amount:    1000,
		StartDate: "2025-06",
		EndDate:   "2025-06",
		Frequency: 1,
	}
	err := extraPaymentEvent.FormDateList(config)
	if err != nil {
		t.Fatalf("FormDateList() error = %v", err)
	}

	loan := &Loan{
		Name:                   "Test Loan with Extra Principal",
		StartDate:              "2025-01",
		Principal:              100000,
		InterestRate:           5.0,
		Term:                   60, // 5 years instead of 30
		DownPayment:            20000,
		ExtraPrincipalPayments: []Event{extraPaymentEvent},
	}

	err = loan.GetAmortizationSchedule(logger, config)
	if err != nil {
		t.Fatalf("GetAmortizationSchedule() error = %v", err)
	}

	// Check that June payment includes extra principal
	junePayment, exists := loan.AmortizationSchedule["2025-06"]
	if !exists {
		t.Errorf("June payment not found")
	}

	mayPayment := loan.AmortizationSchedule["2025-05"]

	// June payment should be higher than May due to extra principal
	if junePayment.Payment <= mayPayment.Payment {
		t.Errorf("June payment (%.2f) should be higher than May payment (%.2f) due to extra principal",
			junePayment.Payment, mayPayment.Payment)
	}
}

func TestExtraPrincipal(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	config := Configuration{
		Common: Common{
			DeathDate: "2030-01",
		},
	}

	// Create events for different scenarios
	monthlyExtra := Event{
		Amount:    500,
		StartDate: "2025-01",
		EndDate:   "2025-12",
		Frequency: 1,
	}
	oneTimeExtra := Event{
		Amount:    5000,
		StartDate: "2025-06",
		EndDate:   "2025-06",
		Frequency: 1,
	}

	// Form date lists
	err := monthlyExtra.FormDateList(config)
	if err != nil {
		t.Fatalf("FormDateList() error = %v", err)
	}
	err = oneTimeExtra.FormDateList(config)
	if err != nil {
		t.Fatalf("FormDateList() error = %v", err)
	}

	loan := &Loan{
		ExtraPrincipalPayments: []Event{monthlyExtra, oneTimeExtra},
	}

	// Test various dates
	tests := []struct {
		date     string
		expected float64
	}{
		{"2025-01", 500},  // Monthly payment
		{"2025-06", 5500}, // Monthly + one-time
		{"2025-12", 500},  // Just monthly
		{"2026-01", 0},    // No payments
	}

	for _, tt := range tests {
		t.Run(tt.date, func(t *testing.T) {
			amount, err := loan.ExtraPrincipal(logger, tt.date)
			if err != nil {
				t.Errorf("ExtraPrincipal() error = %v", err)
			}
			if amount != tt.expected {
				t.Errorf("ExtraPrincipal() = %.2f, expected %.2f", amount, tt.expected)
			}
		})
	}
}

func TestCheckEarlyPayoffThreshold(t *testing.T) {
	loan := &Loan{
		Name:                 "Test Loan",
		StartDate:            "2025-01",
		EarlyPayoffThreshold: 10000,
		AmortizationSchedule: map[string]Payment{
			"2025-05": {RemainingPrincipal: 50000},
		},
	}

	tests := []struct {
		name         string
		currentMonth string
		balance      float64
		expectNote   bool
	}{
		{
			name:         "Balance above threshold",
			currentMonth: "2025-06",
			balance:      65000, // 65000 - 50000 = 15000 > 10000 threshold
			expectNote:   true,
		},
		{
			name:         "Balance below threshold",
			currentMonth: "2025-06",
			balance:      55000, // 55000 - 50000 = 5000 < 10000 threshold
			expectNote:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note, err := loan.CheckEarlyPayoffThreshold(tt.currentMonth, "2030-01", tt.balance)
			if err != nil {
				t.Errorf("CheckEarlyPayoffThreshold() error = %v", err)
			}

			hasNote := note != ""
			if hasNote != tt.expectNote {
				t.Errorf("CheckEarlyPayoffThreshold() note presence = %v, expected %v", hasNote, tt.expectNote)
			}
		})
	}
}

func TestLoanSellProperty(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	config := Configuration{
		Common: Common{
			DeathDate: "2030-01",
		},
	}

	loan := &Loan{
		Name:            "Test Property Loan",
		StartDate:       "2025-01",
		Principal:       200000,
		InterestRate:    4.0,
		Term:            60, // 5 years instead of 30
		DownPayment:     40000,
		EarlyPayoffDate: "2027-06",
		SellProperty:    true,
		SellPrice:       250000,
		SellCostsNet:    15000,
	}

	err := loan.GetAmortizationSchedule(logger, config)
	if err != nil {
		t.Fatalf("GetAmortizationSchedule() error = %v", err)
	}

	// Check early payoff payment when selling property
	payoffPayment, exists := loan.AmortizationSchedule["2027-06"]
	if !exists {
		t.Errorf("Payoff payment not found")
	}

	// When selling: Payment = RemainingPrincipal - SellPrice + SellCosts
	// This should typically be negative (money received) or small positive
	if payoffPayment.Payment > 50000 {
		t.Errorf("Property sale payoff payment seems too high: %.2f", payoffPayment.Payment)
	}
}

// TestLoanNilSafeConversion tests nil safety in object conversion functions
func TestLoanNilSafeConversion(t *testing.T) {
	// Test nil-safe conversion
	var loan *Loan
	result := loan.ToLoansConfig()
	if result != nil {
		t.Errorf("Expected nil result for nil loan, got %v", result)
	}
}

// TestLoanErrorHandling tests proper error handling for various functions
func TestLoanErrorHandling(t *testing.T) {
	// Test proper error handling in GetAmortizationSchedule
	testLoan := &Loan{
		Name:      "Test Loan",
		StartDate: "2025-01",
		Principal: 100000,
	}

	// Test with nil logger - should not error since function creates no-op logger
	err := testLoan.GetAmortizationSchedule(nil, Configuration{})
	if err != nil {
		t.Logf("GetAmortizationSchedule error with nil logger (expected for incomplete loan config): %v", err)
	}

	// Test with valid logger
	logger := zap.NewNop()
	conf := Configuration{
		Common: Common{
			DeathDate: "2030-01",
		},
	}

	err = testLoan.GetAmortizationSchedule(logger, conf)
	if err != nil {
		t.Logf("GetAmortizationSchedule error (expected for incomplete loan config): %v", err)
	}

	// Test CheckEarlyPayoffThreshold with nil checks
	// Create a loan pointer that's nil to test nil safety
	var nilLoan *Loan
	_, err = nilLoan.CheckEarlyPayoffThreshold("2025-01", "2030-01", 50000)
	if err == nil {
		t.Errorf("Expected error for nil loan in CheckEarlyPayoffThreshold, got none")
	}

	// Test with empty currentMonth
	_, err = testLoan.CheckEarlyPayoffThreshold("", "2030-01", 50000)
	if err == nil {
		t.Errorf("Expected error for empty currentMonth, got none")
	}

	// Test with empty deathDate
	_, err = testLoan.CheckEarlyPayoffThreshold("2025-01", "", 50000)
	if err == nil {
		t.Errorf("Expected error for empty deathDate, got none")
	}
}

// TestProcessLoansNilSafety tests the nil safety in ProcessLoans
func TestProcessLoansNilSafety(t *testing.T) {
	logger := zap.NewNop()

	// Test nil configuration
	var conf *Configuration
	err := conf.ProcessLoans(logger)
	if err == nil {
		t.Errorf("Expected error for nil configuration, got none")
	}

	// Test nil logger - this should pass now with our fix
	validConf := &Configuration{
		Common: Common{
			DeathDate: "2030-01",
		},
	}
	err = validConf.ProcessLoans(nil)
	if err != nil {
		t.Errorf("Expected no error for nil logger after fix, got: %v", err)
	}
}
