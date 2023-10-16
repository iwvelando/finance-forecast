// Package config defines the data structures related to configuration and
// includes functions for modifying the loading and parsing the config.
package config

import (
	"testing"

	"go.uber.org/zap"
)

func TestConfiguration_ProcessLoans(t *testing.T) {
	type fields struct {
		Common    Common
		Scenarios []Scenario
	}
	type args struct {
		logger *zap.Logger
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := &Configuration{
				Common:    tt.fields.Common,
				Scenarios: tt.fields.Scenarios,
			}
			if err := conf.ProcessLoans(tt.args.logger); (err != nil) != tt.wantErr {
				t.Errorf("Configuration.ProcessLoans() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoan_GetAmortizationSchedule(t *testing.T) {
	type fields struct {
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
	type args struct {
		logger *zap.Logger
		conf   Configuration
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loan := &Loan{
				Name:                    tt.fields.Name,
				StartDate:               tt.fields.StartDate,
				Principal:               tt.fields.Principal,
				InterestRate:            tt.fields.InterestRate,
				Term:                    tt.fields.Term,
				DownPayment:             tt.fields.DownPayment,
				Escrow:                  tt.fields.Escrow,
				MortgageInsurance:       tt.fields.MortgageInsurance,
				MortgageInsuranceCutoff: tt.fields.MortgageInsuranceCutoff,
				EarlyPayoffThreshold:    tt.fields.EarlyPayoffThreshold,
				EarlyPayoffDate:         tt.fields.EarlyPayoffDate,
				SellProperty:            tt.fields.SellProperty,
				SellPrice:               tt.fields.SellPrice,
				SellCostsNet:            tt.fields.SellCostsNet,
				ExtraPrincipalPayments:  tt.fields.ExtraPrincipalPayments,
				AmortizationSchedule:    tt.fields.AmortizationSchedule,
			}
			if err := loan.GetAmortizationSchedule(tt.args.logger, tt.args.conf); (err != nil) != tt.wantErr {
				t.Errorf("Loan.GetAmortizationSchedule() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoan_ExtraPrincipal(t *testing.T) {
	type fields struct {
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
	type args struct {
		logger *zap.Logger
		date   string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    float64
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loan := &Loan{
				Name:                    tt.fields.Name,
				StartDate:               tt.fields.StartDate,
				Principal:               tt.fields.Principal,
				InterestRate:            tt.fields.InterestRate,
				Term:                    tt.fields.Term,
				DownPayment:             tt.fields.DownPayment,
				Escrow:                  tt.fields.Escrow,
				MortgageInsurance:       tt.fields.MortgageInsurance,
				MortgageInsuranceCutoff: tt.fields.MortgageInsuranceCutoff,
				EarlyPayoffThreshold:    tt.fields.EarlyPayoffThreshold,
				EarlyPayoffDate:         tt.fields.EarlyPayoffDate,
				SellProperty:            tt.fields.SellProperty,
				SellPrice:               tt.fields.SellPrice,
				SellCostsNet:            tt.fields.SellCostsNet,
				ExtraPrincipalPayments:  tt.fields.ExtraPrincipalPayments,
				AmortizationSchedule:    tt.fields.AmortizationSchedule,
			}
			got, err := loan.ExtraPrincipal(tt.args.logger, tt.args.date)
			if (err != nil) != tt.wantErr {
				t.Errorf("Loan.ExtraPrincipal() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Loan.ExtraPrincipal() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoan_CheckEarlyPayoffThreshold(t *testing.T) {
	type fields struct {
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
	type args struct {
		logger       *zap.Logger
		currentMonth string
		deathDate    string
		balance      float64
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loan := &Loan{
				Name:                    tt.fields.Name,
				StartDate:               tt.fields.StartDate,
				Principal:               tt.fields.Principal,
				InterestRate:            tt.fields.InterestRate,
				Term:                    tt.fields.Term,
				DownPayment:             tt.fields.DownPayment,
				Escrow:                  tt.fields.Escrow,
				MortgageInsurance:       tt.fields.MortgageInsurance,
				MortgageInsuranceCutoff: tt.fields.MortgageInsuranceCutoff,
				EarlyPayoffThreshold:    tt.fields.EarlyPayoffThreshold,
				EarlyPayoffDate:         tt.fields.EarlyPayoffDate,
				SellProperty:            tt.fields.SellProperty,
				SellPrice:               tt.fields.SellPrice,
				SellCostsNet:            tt.fields.SellCostsNet,
				ExtraPrincipalPayments:  tt.fields.ExtraPrincipalPayments,
				AmortizationSchedule:    tt.fields.AmortizationSchedule,
			}
			got, err := loan.CheckEarlyPayoffThreshold(tt.args.logger, tt.args.currentMonth, tt.args.deathDate, tt.args.balance)
			if (err != nil) != tt.wantErr {
				t.Errorf("Loan.CheckEarlyPayoffThreshold() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Loan.CheckEarlyPayoffThreshold() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOffsetDate(t *testing.T) {
	type args struct {
		date   string
		layout string
		months int
	}
	tests := []struct {
		name    string
		args    args
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := OffsetDate(tt.args.date, tt.args.layout, tt.args.months)
			if (err != nil) != tt.wantErr {
				t.Errorf("OffsetDate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("OffsetDate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCheckMonth(t *testing.T) {
	type args struct {
		date  string
		month string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := CheckMonth(tt.args.date, tt.args.month)
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckMonth() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("CheckMonth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDateBeforeDate(t *testing.T) {
	type args struct {
		firstDate  string
		secondDate string
	}
	tests := []struct {
		name    string
		args    args
		want    bool
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := DateBeforeDate(tt.args.firstDate, tt.args.secondDate)
			if (err != nil) != tt.wantErr {
				t.Errorf("DateBeforeDate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("DateBeforeDate() = %v, want %v", got, tt.want)
			}
		})
	}
}
