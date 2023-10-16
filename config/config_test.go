// Package config defines the data structures related to configuration and
// includes functions for modifying the loading and parsing the config.
package config

import (
	"reflect"
	"testing"
	"time"
)

func TestLoadConfiguration(t *testing.T) {
	type args struct {
		configPath string
	}
	tests := []struct {
		name    string
		args    args
		want    *Configuration
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := LoadConfiguration(tt.args.configPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfiguration() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LoadConfiguration() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfiguration_ParseDateLists(t *testing.T) {
	type fields struct {
		Common    Common
		Scenarios []Scenario
	}
	tests := []struct {
		name    string
		fields  fields
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
			if err := conf.ParseDateLists(); (err != nil) != tt.wantErr {
				t.Errorf("Configuration.ParseDateLists() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEvent_ComputeAmount(t *testing.T) {
	type fields struct {
		Name      string
		Amount    float64
		StartDate string
		EndDate   string
		Frequency int
		DateList  []time.Time
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			event := &Event{
				Name:      tt.fields.Name,
				Amount:    tt.fields.Amount,
				StartDate: tt.fields.StartDate,
				EndDate:   tt.fields.EndDate,
				Frequency: tt.fields.Frequency,
				DateList:  tt.fields.DateList,
			}
			if err := event.ComputeAmount(); (err != nil) != tt.wantErr {
				t.Errorf("Event.ComputeAmount() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEvent_FormDateList(t *testing.T) {
	type fields struct {
		Name      string
		Amount    float64
		StartDate string
		EndDate   string
		Frequency int
		DateList  []time.Time
	}
	type args struct {
		conf Configuration
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
			event := &Event{
				Name:      tt.fields.Name,
				Amount:    tt.fields.Amount,
				StartDate: tt.fields.StartDate,
				EndDate:   tt.fields.EndDate,
				Frequency: tt.fields.Frequency,
				DateList:  tt.fields.DateList,
			}
			if err := event.FormDateList(tt.args.conf); (err != nil) != tt.wantErr {
				t.Errorf("Event.FormDateList() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRound(t *testing.T) {
	type args struct {
		val float64
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
			if got := Round(tt.args.val); got != tt.want {
				t.Errorf("Round() = %v, want %v", got, tt.want)
			}
		})
	}
}
