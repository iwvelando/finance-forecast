// Package config defines the data structures related to configuration and
// includes functions for modifying the loading and parsing the config.
package config

import (
	"fmt"
	"time"

	"github.com/iwvelando/finance-forecast/pkg/config"
	"github.com/iwvelando/finance-forecast/pkg/constants"
	"github.com/iwvelando/finance-forecast/pkg/events"
	"github.com/spf13/viper"
)

// DateTimeLayout is the format expected in config files and is also the output
// date format.
const DateTimeLayout = constants.DateTimeLayout

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
		return nil, fmt.Errorf("error reading config file, %s", err)
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
	processor := events.NewProcessor()

	// Convert config events to events.Event format for scenarios
	for i, scenario := range conf.Scenarios {
		var eventPtrs []*events.Event
		for j := range scenario.Events {
			eventPtr := &events.Event{
				Name:         conf.Scenarios[i].Events[j].Name,
				Amount:       conf.Scenarios[i].Events[j].Amount,
				StartDate:    conf.Scenarios[i].Events[j].StartDate,
				EndDate:      conf.Scenarios[i].Events[j].EndDate,
				Frequency:    conf.Scenarios[i].Events[j].Frequency,
				StockSymbol:  conf.Scenarios[i].Events[j].StockSymbol,
				StockUnits:   conf.Scenarios[i].Events[j].StockUnits,
				StockTaxRate: conf.Scenarios[i].Events[j].StockTaxRate,
				DateList:     conf.Scenarios[i].Events[j].DateList,
			}
			eventPtrs = append(eventPtrs, eventPtr)
		}

		err := processor.ProcessStockEvents(eventPtrs)
		if err != nil {
			return err
		}

		// Copy back the computed amounts
		for j, eventPtr := range eventPtrs {
			conf.Scenarios[i].Events[j].Amount = eventPtr.Amount
		}
	}

	// Convert config events to events.Event format for common events
	var commonEventPtrs []*events.Event
	for i := range conf.Common.Events {
		eventPtr := &events.Event{
			Name:         conf.Common.Events[i].Name,
			Amount:       conf.Common.Events[i].Amount,
			StartDate:    conf.Common.Events[i].StartDate,
			EndDate:      conf.Common.Events[i].EndDate,
			Frequency:    conf.Common.Events[i].Frequency,
			StockSymbol:  conf.Common.Events[i].StockSymbol,
			StockUnits:   conf.Common.Events[i].StockUnits,
			StockTaxRate: conf.Common.Events[i].StockTaxRate,
			DateList:     conf.Common.Events[i].DateList,
		}
		commonEventPtrs = append(commonEventPtrs, eventPtr)
	}

	err := processor.ProcessStockEvents(commonEventPtrs)
	if err != nil {
		return err
	}

	// Copy back the computed amounts
	for i, eventPtr := range commonEventPtrs {
		conf.Common.Events[i].Amount = eventPtr.Amount
	}

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

// ValidateConfiguration checks for edge cases and returns warnings
// This function identifies potential issues without failing the configuration
func (conf *Configuration) ValidateConfiguration() []string {
	processor := config.NewProcessor()

	// Convert common events
	var commonEvents []config.EventInfo
	for _, event := range conf.Common.Events {
		commonEvents = append(commonEvents, config.EventInfo{
			Name:      event.Name,
			StartDate: event.StartDate,
			EndDate:   event.EndDate,
		})
	}

	// Convert common loans
	var commonLoans []config.LoanInfo
	for _, loan := range conf.Common.Loans {
		commonLoans = append(commonLoans, config.LoanInfo{
			Name:      loan.Name,
			StartDate: loan.StartDate,
			Term:      loan.Term,
		})
	}

	// Convert scenarios
	var scenarios []config.ScenarioInfo
	for _, scenario := range conf.Scenarios {
		var scenarioEvents []config.EventInfo
		for _, event := range scenario.Events {
			scenarioEvents = append(scenarioEvents, config.EventInfo{
				Name:      event.Name,
				StartDate: event.StartDate,
				EndDate:   event.EndDate,
			})
		}

		var scenarioLoans []config.LoanInfo
		for _, loan := range scenario.Loans {
			scenarioLoans = append(scenarioLoans, config.LoanInfo{
				Name:      loan.Name,
				StartDate: loan.StartDate,
				Term:      loan.Term,
			})
		}

		scenarios = append(scenarios, config.ScenarioInfo{
			Name:   scenario.Name,
			Active: scenario.Active,
			Events: scenarioEvents,
			Loans:  scenarioLoans,
		})
	}

	return processor.ValidateConfiguration(conf.Common.DeathDate, commonEvents, commonLoans, scenarios)
}
