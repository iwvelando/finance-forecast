// Package forecast defines the data structures related to a given forecast and
// includes functions for computing the forecasts.
package forecast

import (
	"reflect"
	"testing"

	"github.com/iwvelando/finance-forecast/config"
	"go.uber.org/zap"
)

func TestGetForecast(t *testing.T) {
	type args struct {
		logger *zap.Logger
		conf   config.Configuration
	}
	tests := []struct {
		name    string
		args    args
		want    []Forecast
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetForecast(tt.args.logger, tt.args.conf)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetForecast() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetForecast() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandleEvents(t *testing.T) {
	type args struct {
		logger *zap.Logger
		date   string
		events []config.Event
		layout string
	}
	tests := []struct {
		name    string
		args    args
		want    float64
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := HandleEvents(tt.args.logger, tt.args.date, tt.args.events, tt.args.layout)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleEvents() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("HandleEvents() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandleLoans(t *testing.T) {
	type args struct {
		logger *zap.Logger
		date   string
		loans  []config.Loan
	}
	tests := []struct {
		name string
		args args
		want float64
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HandleLoans(tt.args.logger, tt.args.date, tt.args.loans); got != tt.want {
				t.Errorf("HandleLoans() = %v, want %v", got, tt.want)
			}
		})
	}
}
