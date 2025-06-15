package config

import (
	"testing"
	"time"

	"github.com/iwvelando/finance-forecast/pkg/loans"
)

func TestToLoansConfig(t *testing.T) {
	tests := []struct {
		name     string
		loan     *Loan
		expected *loans.LoanConfig
		wantNil  bool
	}{
		{
			name: "Valid loan conversion",
			loan: &Loan{
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
			},
			expected: &loans.LoanConfig{
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
				ExtraPrincipalPayments:  []loans.Event{},
				AmortizationSchedule:    make(map[string]loans.Payment),
			},
			wantNil: false,
		},
		{
			name:     "Nil loan",
			loan:     nil,
			expected: nil,
			wantNil:  true,
		},
		{
			name: "Loan with extra principal payments",
			loan: &Loan{
				Name:      "Test Loan with Extra",
				StartDate: "2025-01",
				Principal: 100000,
				ExtraPrincipalPayments: []Event{
					{
						Name:      "Extra Payment",
						Amount:    1000,
						StartDate: "2025-06",
						EndDate:   "2025-12",
						Frequency: 3,
						DateList: []time.Time{
							time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC),
							time.Date(2025, 9, 1, 0, 0, 0, 0, time.UTC),
							time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC),
						},
					},
				},
				AmortizationSchedule: make(map[string]Payment),
			},
			expected: &loans.LoanConfig{
				Name:      "Test Loan with Extra",
				StartDate: "2025-01",
				Principal: 100000,
				ExtraPrincipalPayments: []loans.Event{
					{
						Name:      "Extra Payment",
						Amount:    1000,
						StartDate: "2025-06",
						EndDate:   "2025-12",
						Frequency: 3,
						DateList:  []string{"2025-06", "2025-09", "2025-12"},
					},
				},
				AmortizationSchedule: make(map[string]loans.Payment),
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.loan.ToLoansConfig()

			if tt.wantNil {
				if result != nil {
					t.Errorf("ToLoansConfig() expected nil, got %v", result)
				}
				return
			}

			if result == nil {
				t.Errorf("ToLoansConfig() returned nil unexpectedly")
				return
			}

			// Check basic fields
			if tt.expected != nil {
				if result.Name != tt.expected.Name {
					t.Errorf("ToLoansConfig().Name = %s, expected %s", result.Name, tt.expected.Name)
				}
				if result.Principal != tt.expected.Principal {
					t.Errorf("ToLoansConfig().Principal = %f, expected %f", result.Principal, tt.expected.Principal)
				}
				if result.InterestRate != tt.expected.InterestRate {
					t.Errorf("ToLoansConfig().InterestRate = %f, expected %f", result.InterestRate, tt.expected.InterestRate)
				}
			}

			// Check that AmortizationSchedule is initialized
			if result.AmortizationSchedule == nil {
				t.Errorf("ToLoansConfig().AmortizationSchedule is nil")
			}
		})
	}
}

func TestFromLoansPayment(t *testing.T) {
	loanPayment := loans.Payment{
		Payment:            1500.0,
		Principal:          800.0,
		Interest:           700.0,
		RemainingPrincipal: 99200.0,
		RefundableEscrow:   500.0,
	}

	result := FromLoansPayment(loanPayment)

	if result.Payment != loanPayment.Payment {
		t.Errorf("FromLoansPayment().Payment = %f, expected %f", result.Payment, loanPayment.Payment)
	}
	if result.Principal != loanPayment.Principal {
		t.Errorf("FromLoansPayment().Principal = %f, expected %f", result.Principal, loanPayment.Principal)
	}
	if result.Interest != loanPayment.Interest {
		t.Errorf("FromLoansPayment().Interest = %f, expected %f", result.Interest, loanPayment.Interest)
	}
	if result.RemainingPrincipal != loanPayment.RemainingPrincipal {
		t.Errorf("FromLoansPayment().RemainingPrincipal = %f, expected %f", result.RemainingPrincipal, loanPayment.RemainingPrincipal)
	}
	if result.RefundableEscrow != loanPayment.RefundableEscrow {
		t.Errorf("FromLoansPayment().RefundableEscrow = %f, expected %f", result.RefundableEscrow, loanPayment.RefundableEscrow)
	}
}

func TestUpdateFromLoansConfig(t *testing.T) {
	tests := []struct {
		name       string
		loan       *Loan
		loanConfig *loans.LoanConfig
		expectNoop bool
	}{
		{
			name: "Valid update",
			loan: &Loan{
				Name:                 "Test Loan",
				AmortizationSchedule: make(map[string]Payment),
			},
			loanConfig: &loans.LoanConfig{
				AmortizationSchedule: map[string]loans.Payment{
					"2025-01": {
						Payment:            1500.0,
						Principal:          800.0,
						Interest:           700.0,
						RemainingPrincipal: 99200.0,
						RefundableEscrow:   500.0,
					},
				},
			},
			expectNoop: false,
		},
		{
			name:       "Nil loan",
			loan:       nil,
			loanConfig: &loans.LoanConfig{},
			expectNoop: true,
		},
		{
			name: "Nil loan config",
			loan: &Loan{
				AmortizationSchedule: make(map[string]Payment),
			},
			loanConfig: nil,
			expectNoop: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			originalSchedule := make(map[string]Payment)
			if tt.loan != nil && tt.loan.AmortizationSchedule != nil {
				for k, v := range tt.loan.AmortizationSchedule {
					originalSchedule[k] = v
				}
			}

			tt.loan.UpdateFromLoansConfig(tt.loanConfig)

			if tt.expectNoop {
				// For nil cases, nothing should change
				if tt.loan != nil {
					for k, v := range originalSchedule {
						if scheduleV, exists := tt.loan.AmortizationSchedule[k]; !exists || scheduleV != v {
							t.Errorf("UpdateFromLoansConfig() modified schedule when it shouldn't have")
						}
					}
				}
				return
			}

			// Check that the schedule was updated
			if len(tt.loan.AmortizationSchedule) != len(tt.loanConfig.AmortizationSchedule) {
				t.Errorf("UpdateFromLoansConfig() schedule length = %d, expected %d",
					len(tt.loan.AmortizationSchedule), len(tt.loanConfig.AmortizationSchedule))
			}

			for date, expectedPayment := range tt.loanConfig.AmortizationSchedule {
				if payment, exists := tt.loan.AmortizationSchedule[date]; !exists {
					t.Errorf("UpdateFromLoansConfig() missing date %s", date)
				} else if payment.Payment != expectedPayment.Payment {
					t.Errorf("UpdateFromLoansConfig() payment amount = %f, expected %f",
						payment.Payment, expectedPayment.Payment)
				}
			}
		})
	}
}

func TestSyncScheduleWithLoansConfig(t *testing.T) {
	loan := &Loan{
		Name: "Test Loan",
		AmortizationSchedule: map[string]Payment{
			"2024-12": {Payment: 1400.0, Principal: 750.0, Interest: 650.0, RemainingPrincipal: 99250.0},
			"2025-01": {Payment: 1500.0, Principal: 800.0, Interest: 700.0, RemainingPrincipal: 99200.0},
			"2025-02": {Payment: 1500.0, Principal: 810.0, Interest: 690.0, RemainingPrincipal: 98390.0},
		},
	}

	loanConfig := &loans.LoanConfig{
		AmortizationSchedule: map[string]loans.Payment{
			"2025-01": {Payment: 1500.0, Principal: 800.0, Interest: 700.0, RemainingPrincipal: 99200.0},
			// Note: 2024-12 and 2025-02 are intentionally missing (simulating early payoff)
		},
	}

	loan.SyncScheduleWithLoansConfig(loanConfig)

	// Check that only 2025-01 remains
	if len(loan.AmortizationSchedule) != 1 {
		t.Errorf("SyncScheduleWithLoansConfig() schedule length = %d, expected 1", len(loan.AmortizationSchedule))
	}

	if _, exists := loan.AmortizationSchedule["2025-01"]; !exists {
		t.Errorf("SyncScheduleWithLoansConfig() missing expected date 2025-01")
	}

	if _, exists := loan.AmortizationSchedule["2024-12"]; exists {
		t.Errorf("SyncScheduleWithLoansConfig() should have removed date 2024-12")
	}

	if _, exists := loan.AmortizationSchedule["2025-02"]; exists {
		t.Errorf("SyncScheduleWithLoansConfig() should have removed date 2025-02")
	}
}

func TestSyncScheduleWithLoansConfigNilCases(t *testing.T) {
	// Test with nil loan
	var nilLoan *Loan
	loanConfig := &loans.LoanConfig{
		AmortizationSchedule: map[string]loans.Payment{
			"2025-01": {Payment: 1500.0},
		},
	}

	// Should not panic
	nilLoan.SyncScheduleWithLoansConfig(loanConfig)

	// Test with nil loan config
	loan := &Loan{
		Name: "Test",
		AmortizationSchedule: map[string]Payment{
			"2025-01": {Payment: 1500.0},
		},
	}
	originalLen := len(loan.AmortizationSchedule)

	loan.SyncScheduleWithLoansConfig(nil)

	// Should not modify the original schedule
	if len(loan.AmortizationSchedule) != originalLen {
		t.Errorf("SyncScheduleWithLoansConfig(nil) modified schedule length")
	}
}
