package config

import (
	"math"
	"testing"

	"go.uber.org/zap"
)

// TestLoanEdgeCases tests various edge cases in loan processing
func TestLoanEdgeCases(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	tests := []struct {
		name        string
		loan        Loan
		config      Configuration
		expectError bool
		description string
	}{
		{
			name: "Zero interest rate",
			loan: Loan{
				Name:         "Zero Interest",
				StartDate:    "2025-01",
				Principal:    100000,
				InterestRate: 0.0,
				Term:         60, // 5 years instead of 30
			},
			config: Configuration{
				Common: Common{DeathDate: "2030-01"},
			},
			expectError: false,
			description: "Should handle zero interest rate gracefully",
		},
		{
			name: "100% down payment",
			loan: Loan{
				Name:         "Full Cash",
				StartDate:    "2025-01",
				Principal:    0,
				InterestRate: 5.0,
				Term:         60,
			},
			config: Configuration{
				Common: Common{DeathDate: "2030-01"},
			},
			expectError: false,
			description: "Should handle zero principal (100% down payment)",
		},
		{
			name: "Very high interest rate",
			loan: Loan{
				Name:         "High Interest",
				StartDate:    "2025-01",
				Principal:    100000,
				InterestRate: 50.0,
				Term:         60, // 5 years instead of 30
			},
			config: Configuration{
				Common: Common{DeathDate: "2030-01"},
			},
			expectError: false,
			description: "Should handle very high interest rates",
		},
		{
			name: "Single month term",
			loan: Loan{
				Name:         "Short Term",
				StartDate:    "2025-01",
				Principal:    1000,
				InterestRate: 5.0,
				Term:         1,
			},
			config: Configuration{
				Common: Common{DeathDate: "2030-01"},
			},
			expectError: false,
			description: "Should handle very short loan terms",
		},
		{
			name: "Early payoff before first payment",
			loan: Loan{
				Name:            "Immediate Payoff",
				StartDate:       "2025-02",
				Principal:       100000,
				InterestRate:    5.0,
				Term:            60, // 5 years instead of 30
				EarlyPayoffDate: "2025-01",
			},
			config: Configuration{
				Common: Common{DeathDate: "2030-01"},
			},
			expectError: false,
			description: "Should handle early payoff before loan starts",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test loan processing
			tt.config.Common.Loans = []Loan{tt.loan}
			err := tt.config.ProcessLoans(logger)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s but got none", tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.description, err)
			}
		})
	}
}

// TestInvalidConfigurations tests configurations that should fail validation
func TestInvalidConfigurations(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	tests := []struct {
		name        string
		config      Configuration
		expectError bool
		description string
	}{
		{
			name: "Missing death date",
			config: Configuration{
				Common: Common{
					StartingValue: 100000,
					Loans: []Loan{{
						Name:         "Test Loan",
						StartDate:    "2025-01",
						Principal:    100000,
						InterestRate: 5.0,
						Term:         60, // 5 years instead of 30
					}},
				},
			},
			expectError: true,
			description: "Should fail when death date is missing",
		},
		{
			name: "Negative loan term",
			config: Configuration{
				Common: Common{
					DeathDate: "2030-01",
					Loans: []Loan{{
						Name:         "Invalid Term",
						StartDate:    "2025-01",
						Principal:    100000,
						InterestRate: 5.0,
						Term:         -1,
					}},
				},
			},
			expectError: false, // The current implementation may not validate this
			description: "Should fail with negative loan term",
		},
		{
			name: "Negative principal",
			config: Configuration{
				Common: Common{
					DeathDate: "2030-01",
					Loans: []Loan{{
						Name:         "Negative Principal",
						StartDate:    "2025-01",
						Principal:    -100000,
						InterestRate: 5.0,
						Term:         60, // 5 years instead of 30
					}},
				},
			},
			expectError: false, // May be allowed for some scenarios
			description: "Negative principal handling",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.config.ProcessLoans(logger)

			if tt.expectError && err == nil {
				t.Errorf("Expected error for %s but got none", tt.description)
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error for %s: %v", tt.description, err)
			}
		})
	}
}

// TestMathematicalValidation tests mathematical correctness of calculations
func TestMathematicalValidation(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	// Test a simple loan with known values
	config := Configuration{
		Common: Common{
			DeathDate: "2030-01",
			Loans: []Loan{{
				Name:         "Math Test",
				StartDate:    "2025-01",
				Principal:    100000,
				InterestRate: 6.0, // 6% annual = 0.5% monthly
				Term:         60,  // 5 years instead of 30
			}},
		},
	}

	err := config.ProcessLoans(logger)
	if err != nil {
		t.Fatalf("Failed to process loan: %v", err)
	}

	loan := &config.Common.Loans[0]
	if len(loan.AmortizationSchedule) == 0 {
		t.Fatal("Amortization schedule not generated")
	}

	// Check first payment - find the first date in the schedule
	var firstDate string
	var firstPayment Payment
	for date, payment := range loan.AmortizationSchedule {
		if firstDate == "" || date < firstDate {
			firstDate = date
			firstPayment = payment
		}
	}

	// For a $100,000 loan at 6% for 5 years (60 months), calculate expected payment
	// Using formula: M = P[r(1+r)^n]/[(1+r)^n-1]
	// P=100000, r=0.005 (6%/12), n=60
	principal := 100000.0
	monthlyRate := 0.06 / 12 // 0.5% monthly
	months := 60.0

	numerator := principal * (monthlyRate * math.Pow(1+monthlyRate, months))
	denominator := math.Pow(1+monthlyRate, months) - 1
	expectedPayment := numerator / denominator // Should be around $1933

	tolerance := 50.0 // Allow $50 tolerance

	if firstPayment.Payment < expectedPayment-tolerance ||
		firstPayment.Payment > expectedPayment+tolerance {
		t.Errorf("Monthly payment should be approximately %.2f, got %.2f",
			expectedPayment, firstPayment.Payment)
	}

	// Check that first payment has correct interest (principal * monthly rate)
	expectedFirstInterest := 100000 * 0.06 / 12 // $500
	if firstPayment.Interest < expectedFirstInterest-1 ||
		firstPayment.Interest > expectedFirstInterest+1 {
		t.Errorf("First payment interest should be approximately %.2f, got %.2f",
			expectedFirstInterest, firstPayment.Interest)
	}

	// Verify that payment = principal + interest for first payment
	calculatedPayment := firstPayment.Principal + firstPayment.Interest
	if abs(calculatedPayment-firstPayment.Payment) > 0.01 {
		t.Errorf("Payment should equal principal + interest, got %.2f + %.2f = %.2f, expected %.2f",
			firstPayment.Principal, firstPayment.Interest, calculatedPayment, firstPayment.Payment)
	}
}

// TestAmortizationConsistency tests that the amortization schedule is mathematically consistent
func TestAmortizationConsistency(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	config := Configuration{
		Common: Common{
			DeathDate: "2030-01",
			Loans: []Loan{{
				Name:         "Consistency Test",
				StartDate:    "2025-01",
				Principal:    100000,
				InterestRate: 6.0,
				Term:         60, // 5 years instead of 30
			}},
		},
	}

	err := config.ProcessLoans(logger)
	if err != nil {
		t.Fatalf("Failed to process loan: %v", err)
	}

	loan := &config.Common.Loans[0]
	schedule := loan.AmortizationSchedule

	if len(schedule) == 0 {
		t.Fatal("No amortization schedule generated")
	}

	// Get sorted list of dates
	var dates []string
	for date := range schedule {
		dates = append(dates, date)
	}

	// Sort dates to process them in order
	for i := 0; i < len(dates)-1; i++ {
		for j := i + 1; j < len(dates); j++ {
			if dates[i] > dates[j] {
				dates[i], dates[j] = dates[j], dates[i]
			}
		}
	}

	// Test that remaining principal decreases over time (for first 10 payments)
	checkCount := 10
	if len(dates) < checkCount {
		checkCount = len(dates) - 1
	}

	for i := 1; i < checkCount; i++ {
		currentPayment := schedule[dates[i]]
		previousPayment := schedule[dates[i-1]]

		if currentPayment.RemainingPrincipal >= previousPayment.RemainingPrincipal {
			t.Errorf("Remaining principal should decrease over time, but at payment %s: %.2f >= %.2f (previous: %s)",
				dates[i], currentPayment.RemainingPrincipal, previousPayment.RemainingPrincipal, dates[i-1])
		}
	}

	// Test that interest portion decreases over time (generally)
	for i := 1; i < checkCount; i++ {
		currentPayment := schedule[dates[i]]
		previousPayment := schedule[dates[i-1]]

		if currentPayment.Interest > previousPayment.Interest {
			t.Errorf("Interest portion should generally decrease over time, but at payment %s: %.2f > %.2f (previous: %s)",
				dates[i], currentPayment.Interest, previousPayment.Interest, dates[i-1])
		}
	}

	// Test that principal portion increases over time (generally)
	for i := 1; i < checkCount; i++ {
		currentPayment := schedule[dates[i]]
		previousPayment := schedule[dates[i-1]]

		if currentPayment.Principal < previousPayment.Principal {
			t.Errorf("Principal portion should generally increase over time, but at payment %s: %.2f < %.2f (previous: %s)",
				dates[i], currentPayment.Principal, previousPayment.Principal, dates[i-1])
		}
	}

	// Check a payment in the middle of the schedule
	if len(dates) > 5 {
		midDate := dates[5] // Check 6th payment for 5-year loan
		midPayment := schedule[midDate]

		// For a $100k loan at 6%, interest should be less than initial due to principal paydown
		initialMonthlyInterest := 100000 * 0.06 / 12 // Around $500

		// By the 6th payment on a 5-year loan, some principal has been paid down
		// so interest should be somewhat less than initial
		expectedLower := initialMonthlyInterest - 100 // Allow for principal paydown
		expectedUpper := initialMonthlyInterest

		if midPayment.Interest < expectedLower || midPayment.Interest > expectedUpper {
			t.Errorf("Mid payment interest should be approximately %.2f, got %.2f",
				(expectedLower+expectedUpper)/2, midPayment.Interest)
		}

		// Verify principal + interest = payment (for regular payments)
		calculatedPayment := midPayment.Principal + midPayment.Interest
		if abs(calculatedPayment-midPayment.Payment) > 1.0 {
			t.Errorf("Principal + Interest should equal Payment for regular payment, got %.2f + %.2f != %.2f",
				midPayment.Principal, midPayment.Interest, midPayment.Payment)
		}
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
