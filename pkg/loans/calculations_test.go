package loans

import (
	"math"
	"testing"

	"go.uber.org/zap"
)

func TestCalculateMonthlyPayment(t *testing.T) {
	tests := []struct {
		name               string
		principal          float64
		downPayment        float64
		annualInterestRate float64
		termMonths         int
		expectedRange      []float64 // [min, max] expected range
	}{
		{
			name:               "Standard 30-year mortgage",
			principal:          300000,
			downPayment:        60000, // 20%
			annualInterestRate: 6.0,
			termMonths:         360,
			expectedRange:      []float64{1400, 1500}, // Around $1439
		},
		{
			name:               "5-year car loan",
			principal:          25000,
			downPayment:        5000,
			annualInterestRate: 4.0,
			termMonths:         60,
			expectedRange:      []float64{360, 380}, // Around $368
		},
		{
			name:               "Zero interest loan",
			principal:          12000,
			downPayment:        2000,
			annualInterestRate: 0.0,
			termMonths:         60,
			expectedRange:      []float64{166, 167}, // Exactly $166.67
		},
		{
			name:               "100% down payment",
			principal:          50000,
			downPayment:        50000,
			annualInterestRate: 5.0,
			termMonths:         60,
			expectedRange:      []float64{0, 0}, // Should be 0
		},
		{
			name:               "High interest loan",
			principal:          10000,
			downPayment:        0,
			annualInterestRate: 18.0,
			termMonths:         36,
			expectedRange:      []float64{360, 380}, // Around $372
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateMonthlyPayment(tt.principal, tt.downPayment, tt.annualInterestRate, tt.termMonths)

			if result < tt.expectedRange[0] || result > tt.expectedRange[1] {
				t.Errorf("CalculateMonthlyPayment() = %.2f, expected range [%.2f, %.2f]",
					result, tt.expectedRange[0], tt.expectedRange[1])
			}
		})
	}
}

func TestCalculateInterestPayment(t *testing.T) {
	tests := []struct {
		name               string
		remainingPrincipal float64
		annualInterestRate float64
		expected           float64
	}{
		{
			name:               "Standard mortgage interest",
			remainingPrincipal: 200000,
			annualInterestRate: 6.0,
			expected:           1000.0, // 200000 * 0.06 / 12
		},
		{
			name:               "Car loan interest",
			remainingPrincipal: 15000,
			annualInterestRate: 4.5,
			expected:           56.25, // 15000 * 0.045 / 12
		},
		{
			name:               "Zero interest",
			remainingPrincipal: 10000,
			annualInterestRate: 0.0,
			expected:           0.0,
		},
		{
			name:               "High interest",
			remainingPrincipal: 5000,
			annualInterestRate: 24.0,
			expected:           100.0, // 5000 * 0.24 / 12
		},
		{
			name:               "Very small principal",
			remainingPrincipal: 100,
			annualInterestRate: 6.0,
			expected:           0.5, // 100 * 0.06 / 12
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateInterestPayment(tt.remainingPrincipal, tt.annualInterestRate)

			if math.Abs(result-tt.expected) > 0.01 {
				t.Errorf("CalculateInterestPayment() = %.2f, expected %.2f", result, tt.expected)
			}
		})
	}
}

func TestCheckEarlyPayoffThreshold(t *testing.T) {
	// Create mock amortization schedule
	schedule := map[string]Payment{
		"2025-05": {RemainingPrincipal: 44000}, // Previous month to 2025-06
		"2025-06": {RemainingPrincipal: 43000},
	}

	tests := []struct {
		name         string
		loanName     string
		startDate    string
		currentMonth string
		threshold    float64
		balance      float64
		expectNote   bool
	}{
		{
			name:         "Above threshold triggers note",
			loanName:     "Test Loan",
			startDate:    "2025-01",
			currentMonth: "2025-06",
			threshold:    5000,
			balance:      50000, // 50000 - 44000 = 6000 > 5000
			expectNote:   true,
		},
		{
			name:         "Below threshold no note",
			loanName:     "Test Loan",
			startDate:    "2025-01",
			currentMonth: "2025-06",
			threshold:    10000,
			balance:      50000, // 50000 - 44000 = 6000 < 10000
			expectNote:   false,
		},
		{
			name:         "Zero threshold no note",
			loanName:     "Test Loan",
			startDate:    "2025-01",
			currentMonth: "2025-06",
			threshold:    0,
			balance:      50000,
			expectNote:   false,
		},
		{
			name:         "Loan not started yet",
			loanName:     "Future Loan",
			startDate:    "2025-12",
			currentMonth: "2025-06",
			threshold:    5000,
			balance:      50000,
			expectNote:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note, err := CheckEarlyPayoffThreshold(
				tt.loanName,
				tt.startDate,
				tt.currentMonth,
				tt.threshold,
				schedule,
				tt.balance,
			)

			if err != nil {
				t.Errorf("CheckEarlyPayoffThreshold() error = %v", err)
				return
			}

			hasNote := note != ""
			if hasNote != tt.expectNote {
				t.Errorf("CheckEarlyPayoffThreshold() note = %t, expected %t", hasNote, tt.expectNote)
			}
		})
	}
}

func TestCalculateExtraPrincipal(t *testing.T) {
	events := []Event{
		{
			Name:     "Monthly Extra",
			Amount:   500,
			DateList: []string{"2025-01", "2025-02", "2025-03"},
		},
		{
			Name:     "One Time Extra",
			Amount:   5000,
			DateList: []string{"2025-02"},
		},
		{
			Name:     "Quarterly Extra",
			Amount:   1000,
			DateList: []string{"2025-03", "2025-06", "2025-09"},
		},
	}

	tests := []struct {
		name     string
		date     string
		expected float64
	}{
		{
			name:     "Date with monthly only",
			date:     "2025-01",
			expected: 500,
		},
		{
			name:     "Date with monthly and one-time",
			date:     "2025-02",
			expected: 5500, // 500 + 5000
		},
		{
			name:     "Date with monthly and quarterly",
			date:     "2025-03",
			expected: 1500, // 500 + 1000
		},
		{
			name:     "Date with quarterly only",
			date:     "2025-06",
			expected: 1000,
		},
		{
			name:     "Date with no extra payments",
			date:     "2025-04",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateExtraPrincipal(events, tt.date)

			if result != tt.expected {
				t.Errorf("CalculateExtraPrincipal() = %.2f, expected %.2f", result, tt.expected)
			}
		})
	}
}

func TestAmortizationScheduleGenerator_GenerateSchedule(t *testing.T) {
	logger := zap.NewNop()
	generator := NewAmortizationScheduleGenerator(logger)

	loan := &LoanConfig{
		Name:         "Test Loan",
		StartDate:    "2025-01",
		Principal:    100000,
		InterestRate: 6.0,
		Term:         60, // 5 years
		DownPayment:  20000,
		Escrow:       500,
	}

	schedule, err := generator.GenerateSchedule(loan, "2030-01")
	if err != nil {
		t.Fatalf("GenerateSchedule() error = %v", err)
	}

	// Verify basic properties
	if len(schedule) == 0 {
		t.Errorf("GenerateSchedule() produced empty schedule")
	}

	// Check first payment
	firstPayment, exists := schedule["2025-01"]
	if !exists {
		t.Errorf("GenerateSchedule() missing first payment")
	}

	// First payment should include down payment
	if firstPayment.Payment < 20000 {
		t.Errorf("First payment should include down payment of 20000, got %.2f", firstPayment.Payment)
	}

	// Check escrow tracking
	if firstPayment.RefundableEscrow != 500 {
		t.Errorf("First payment should have RefundableEscrow of 500, got %.2f", firstPayment.RefundableEscrow)
	}

	// Verify remaining principal decreases over time
	lastRemaining := math.MaxFloat64
	dates := []string{"2025-01", "2025-02", "2025-03", "2025-04", "2025-05"}

	for _, date := range dates {
		if payment, exists := schedule[date]; exists {
			if payment.RemainingPrincipal >= lastRemaining {
				t.Errorf("Remaining principal should decrease over time")
			}
			lastRemaining = payment.RemainingPrincipal
		}
	}
}

func TestAmortizationScheduleGenerator_WithEarlyPayoff(t *testing.T) {
	logger := zap.NewNop()
	generator := NewAmortizationScheduleGenerator(logger)

	loan := &LoanConfig{
		Name:            "Early Payoff Loan",
		StartDate:       "2025-01",
		Principal:       100000,
		InterestRate:    6.0,
		Term:            60,
		DownPayment:     20000,
		EarlyPayoffDate: "2025-06",
	}

	schedule, err := generator.GenerateSchedule(loan, "2030-01")
	if err != nil {
		t.Fatalf("GenerateSchedule() error = %v", err)
	}

	// Should have early payoff payment
	payoffPayment, exists := schedule["2025-06"]
	if !exists {
		t.Errorf("GenerateSchedule() missing early payoff payment")
	}

	// Early payoff should be a significant amount
	if payoffPayment.Payment < 50000 {
		t.Errorf("Early payoff payment seems too small: %.2f", payoffPayment.Payment)
	}
}

func TestNewAmortizationScheduleGenerator(t *testing.T) {
	logger := zap.NewNop()
	generator := NewAmortizationScheduleGenerator(logger)

	if generator == nil {
		t.Error("NewAmortizationScheduleGenerator() returned nil")
		return
	}

	if generator.logger != logger {
		t.Error("NewAmortizationScheduleGenerator() logger not set correctly")
	}
}

func TestCalculateExtraPrincipalWithOverpaymentPrevention(t *testing.T) {
	logger := zap.NewNop()

	events := []Event{
		{
			Name:     "Large Extra Payment",
			Amount:   50000, // More than remaining principal
			DateList: []string{"2025-06"},
		},
	}

	tests := []struct {
		name               string
		date               string
		monthlyPayment     float64
		remainingPrincipal float64
		interestRate       float64
		expectedMax        float64 // Maximum expected extra principal
		expectAdjustment   bool
	}{
		{
			name:               "Normal case - no overpayment",
			date:               "2025-01",
			monthlyPayment:     1000,
			remainingPrincipal: 80000,
			interestRate:       6.0,
			expectedMax:        50000, // Full amount should be allowed
			expectAdjustment:   false,
		},
		{
			name:               "Overpayment scenario",
			date:               "2025-06",
			monthlyPayment:     1000,
			remainingPrincipal: 10000, // Small remaining balance
			interestRate:       6.0,
			expectedMax:        10000, // Should be limited to remaining balance
			expectAdjustment:   true,
		},
		{
			name:               "No extra payment date",
			date:               "2025-12",
			monthlyPayment:     1000,
			remainingPrincipal: 50000,
			interestRate:       6.0,
			expectedMax:        0, // No payment on this date
			expectAdjustment:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CalculateExtraPrincipalWithOverpaymentPrevention(
				logger,
				events,
				tt.date,
				tt.monthlyPayment,
				tt.remainingPrincipal,
				tt.interestRate,
				"Test Loan",
			)

			if err != nil {
				t.Errorf("CalculateExtraPrincipalWithOverpaymentPrevention() error = %v", err)
				return
			}

			if result > tt.expectedMax {
				t.Errorf("CalculateExtraPrincipalWithOverpaymentPrevention() = %.2f, should not exceed %.2f",
					result, tt.expectedMax)
			}

			if result < 0 {
				t.Errorf("CalculateExtraPrincipalWithOverpaymentPrevention() = %.2f, should not be negative", result)
			}
		})
	}
}

func TestLoanConfigStruct(t *testing.T) {
	// Test that LoanConfig struct can be properly initialized
	loan := LoanConfig{
		Name:                    "Test Loan",
		StartDate:               "2025-01",
		Principal:               100000,
		InterestRate:            5.0,
		Term:                    360,
		DownPayment:             20000,
		Escrow:                  500,
		MortgageInsurance:       100,
		MortgageInsuranceCutoff: 78.0,
		EarlyPayoffThreshold:    5000,
		EarlyPayoffDate:         "2030-01",
		SellProperty:            true,
		SellPrice:               120000,
		SellCostsNet:            8000,
		ExtraPrincipalPayments:  []Event{},
		AmortizationSchedule:    make(map[string]Payment),
	}

	// Verify all fields are accessible
	if loan.Name != "Test Loan" {
		t.Errorf("LoanConfig.Name = %s, expected Test Loan", loan.Name)
	}
	if loan.Principal != 100000 {
		t.Errorf("LoanConfig.Principal = %.2f, expected 100000", loan.Principal)
	}
	if loan.MortgageInsuranceCutoff != 78.0 {
		t.Errorf("LoanConfig.MortgageInsuranceCutoff = %.1f, expected 78.0", loan.MortgageInsuranceCutoff)
	}
}

func TestPaymentStruct(t *testing.T) {
	// Test Payment struct
	payment := Payment{
		Payment:            1500.00,
		Principal:          800.00,
		Interest:           700.00,
		RemainingPrincipal: 95000.00,
		RefundableEscrow:   500.00,
	}

	// Verify all fields
	if payment.Payment != 1500.00 {
		t.Errorf("Payment.Payment = %.2f, expected 1500.00", payment.Payment)
	}
	if payment.Principal != 800.00 {
		t.Errorf("Payment.Principal = %.2f, expected 800.00", payment.Principal)
	}
	if payment.Interest != 700.00 {
		t.Errorf("Payment.Interest = %.2f, expected 700.00", payment.Interest)
	}
	if payment.RemainingPrincipal != 95000.00 {
		t.Errorf("Payment.RemainingPrincipal = %.2f, expected 95000.00", payment.RemainingPrincipal)
	}
	if payment.RefundableEscrow != 500.00 {
		t.Errorf("Payment.RefundableEscrow = %.2f, expected 500.00", payment.RefundableEscrow)
	}

	// Verify principal + interest approximately equals payment (allowing for rounding)
	calculatedPayment := payment.Principal + payment.Interest
	if math.Abs(calculatedPayment-payment.Payment) > 1.0 {
		t.Errorf("Principal + Interest = %.2f, Payment = %.2f, difference too large",
			calculatedPayment, payment.Payment)
	}
}
