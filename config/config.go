// Package config defines the data structures related to configuration and
// includes functions for modifying the loading and parsing the config.
package config

import (
	"fmt"
	"github.com/piquette/finance-go/quote"
	"github.com/spf13/viper"
	"math"
	"time"
)

// DateTimeLayout is the format expected in config files and is also the output
// date format.
const DateTimeLayout = "2006-01"

// Configuration holds all configuration for finance-forecast.
type Configuration struct {
	Common    Common
	Scenarios []Scenario
}

// Common holds the shared parameters, events, and loans between all scenarios.
type Common struct {
	StartingValue float64
	DeathDate     string
	Events        []Event
	Loans         []Loan
}

// Scenario holds all events and loans for a given scenario.
type Scenario struct {
	Name   string
	Active bool
	Events []Event
	Loans  []Loan
}

// Event indicates a financial event.
type Event struct {
	Name         string
	Amount       float64
	StartDate    string
	EndDate      string
	Frequency    int // months
	StockSymbol  string
	StockUnits   float64
	StockTaxRate float64
	DateList     []time.Time
}

// LoadConfiguration takes a file path as input and loads the YAML-formatted
// configuration there.
func LoadConfiguration(configPath string) (*Configuration, error) {
	viper.SetConfigFile(configPath)
	viper.AutomaticEnv()

	viper.SetConfigType("yml")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("Error reading config file, %s", err)
	}

	var configuration Configuration
	err := viper.Unmarshal(&configuration)
	if err != nil {
		return nil, fmt.Errorf("unable to decode into struct, %s", err)
	}

	return &configuration, nil
}

// ParseDateLists looks at every date provided in the configuration and
// parses it into a time.Time which is stored back into an Event.DateList.
func (conf *Configuration) ParseDateLists() error {
	// First handle the parsing for all Events in Scenarios.
	for i, scenario := range conf.Scenarios {
		for j := range scenario.Events {
			err := conf.Scenarios[i].Events[j].FormDateList(*conf)
			if err != nil {
				return err
			}
		}
		// Check for extra principal payments within loans.
		for j, loan := range scenario.Loans {
			for k := range loan.ExtraPrincipalPayments {
				err := conf.Scenarios[i].Loans[j].ExtraPrincipalPayments[k].FormDateList(*conf)
				if err != nil {
					return err
				}
			}
		}
	}

	// Next handle the parsing for the Common Events.
	for i := range conf.Common.Events {
		err := conf.Common.Events[i].FormDateList(*conf)
		if err != nil {
			return err
		}
	}

	// Check for extra principal payments for common loans.
	for i, loan := range conf.Common.Loans {
		for j := range loan.ExtraPrincipalPayments {
			err := conf.Common.Loans[i].ExtraPrincipalPayments[j].FormDateList(*conf)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// ProcessStockEvents determines the amount for any events declaring a stock symbol
func (conf *Configuration) ProcessStockEvents() error {
	// First handle the processing for all Events in Scenarios.
	for i, scenario := range conf.Scenarios {
		for j := range scenario.Events {
			err := conf.Scenarios[i].Events[j].ComputeAmount()
			if err != nil {
				return err
			}
		}
	}

	// Next handle the processing for the Common Events.
	for i := range conf.Common.Events {
		err := conf.Common.Events[i].ComputeAmount()
		if err != nil {
			return err
		}
	}

	return nil
}

// ComputeAmount determines Amount of Stock parameters have been set
func (event *Event) ComputeAmount() error {
	if event.StockSymbol == "" {
		return nil
	}

	price, err := quote.Get(event.StockSymbol)
	if err != nil {
		return err
	}

	event.Amount = event.StockUnits * (1 - event.StockTaxRate) * price.RegularMarketPrice

	return nil
}

// FormDateList handles the date to time.Time parsing for one given event.
func (event *Event) FormDateList(conf Configuration) error {
	dateList := make([]time.Time, 1)
	var startDateT time.Time
	var err error

	// Unspecified startDate goes to the current time.
	if event.StartDate == "" {
		startDateT, err = time.Parse(DateTimeLayout, time.Now().Format(DateTimeLayout))
		if err != nil {
			return err
		}
	} else {
		startDateT, err = time.Parse(DateTimeLayout, event.StartDate)
		if err != nil {
			return err
		}
	}

	// Unspecified endDate goes to the deathDate.
	if event.EndDate == "" {
		event.EndDate = conf.Common.DeathDate
	}
	endDateT, err := time.Parse(DateTimeLayout, event.EndDate)
	if err != nil {
		return err
	}

	// Identify all dates where an event takes place and aggregate them in
	// dateList.
	dateList[0] = startDateT
	for {
		nextDate := dateList[len(dateList)-1].AddDate(0, event.Frequency, 0)
		if nextDate.Equal(endDateT) {
			dateList = append(dateList, nextDate)
			break
		} else if nextDate.After(endDateT) {
			break
		} else {
			dateList = append(dateList, nextDate)
		}
	}
	event.DateList = dateList

	return nil
}

// Round rounds a value to two decimals, i.e. to represent real currency. Used
// for making logical comparisons.
func Round(val float64) float64 {
	return math.Round(val*100) / 100
}
