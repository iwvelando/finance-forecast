// Package config defines the data structures related to configuration and
// includes functions for modifying the loading and parsing the config.
package config

import (
	"fmt"
	"time"

	"github.com/iwvelando/finance-forecast/pkg/configprocessor"
	"github.com/iwvelando/finance-forecast/pkg/constants"
	"github.com/iwvelando/finance-forecast/pkg/datetime"
	"github.com/spf13/viper"
)

// DateTimeLayout is the format expected in config files and is also the output
// date format.
const DateTimeLayout = constants.DateTimeLayout

// Configuration holds all configuration for finance-forecast.
type Configuration struct {
	Common    Common
	Scenarios []Scenario
	Logging   LoggingConfig `yaml:"logging,omitempty"`
	Output    OutputConfig  `yaml:"output,omitempty"`
}

// LoggingConfig holds logging configuration options
type LoggingConfig struct {
	Level      string `yaml:"level,omitempty"`      // debug, info, warn, error
	Format     string `yaml:"format,omitempty"`     // json, console
	OutputFile string `yaml:"outputFile,omitempty"` // optional file output
}

// OutputConfig holds output format configuration options
type OutputConfig struct {
	Format string `yaml:"format,omitempty"` // pretty, csv
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
	Name      string
	Amount    float64
	StartDate string
	EndDate   string
	Frequency int // months
	DateList  []time.Time
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
	return conf.ParseDateListsWithFixedTime(time.Now())
}

// ParseDateListsWithFixedTime parses all date lists in the configuration using a fixed time
func (conf *Configuration) ParseDateListsWithFixedTime(fixedTime time.Time) error {
	// First handle the parsing for all Events in Scenarios.
	for i, scenario := range conf.Scenarios {
		for j := range scenario.Events {
			err := conf.Scenarios[i].Events[j].FormDateListWithFixedTime(*conf, fixedTime)
			if err != nil {
				return err
			}
		}
		// Check for extra principal payments within loans.
		for j, loan := range scenario.Loans {
			for k := range loan.ExtraPrincipalPayments {
				err := conf.Scenarios[i].Loans[j].ExtraPrincipalPayments[k].FormDateListWithFixedTime(*conf, fixedTime)
				if err != nil {
					return err
				}
			}
		}
	}

	// Next handle the parsing for the Common Events.
	for i := range conf.Common.Events {
		err := conf.Common.Events[i].FormDateListWithFixedTime(*conf, fixedTime)
		if err != nil {
			return err
		}
	}

	// Check for extra principal payments for common loans.
	for i, loan := range conf.Common.Loans {
		for j := range loan.ExtraPrincipalPayments {
			err := conf.Common.Loans[i].ExtraPrincipalPayments[j].FormDateListWithFixedTime(*conf, fixedTime)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// FormDateList handles the date to time.Time parsing for one given event.
// This utilizes the datetime package for parsing and date manipulation.
func (event *Event) FormDateList(conf Configuration) error {
	return event.FormDateListWithFixedTime(conf, time.Now())
}

// FormDateListWithFixedTime handles the date to time.Time parsing for one given event
// with injectable fixed time for testing.
func (event *Event) FormDateListWithFixedTime(conf Configuration, fixedTime time.Time) error {
	dateList := make([]time.Time, 1)
	var startDateT time.Time
	var err error

	// Unspecified startDate goes to the fixed time.
	if event.StartDate == "" {
		// Use datetime package for consistent date handling
		startDateT = datetime.MustParseTime(DateTimeLayout, fixedTime.Format(DateTimeLayout))
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

	// Identify all dates where an event takes place and aggregate them in dateList.
	dateList[0] = startDateT

	// Using datetime pattern to generate sequence of dates
	for {
		// Calculate next event date based on frequency
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

// ValidateConfiguration performs general validation of the configuration and returns warnings
func (c *Configuration) ValidateConfiguration() []string {
	// Convert config structs to configprocessor format
	var commonEvents []configprocessor.EventInfo
	for _, event := range c.Common.Events {
		commonEvents = append(commonEvents, configprocessor.EventInfo{
			Name:      event.Name,
			StartDate: event.StartDate,
			EndDate:   event.EndDate,
		})
	}

	var scenarios []configprocessor.ScenarioInfo
	for _, scenario := range c.Scenarios {
		var scenarioEvents []configprocessor.EventInfo
		for _, event := range scenario.Events {
			scenarioEvents = append(scenarioEvents, configprocessor.EventInfo{
				Name:      event.Name,
				StartDate: event.StartDate,
				EndDate:   event.EndDate,
			})
		}

		scenarios = append(scenarios, configprocessor.ScenarioInfo{
			Name:   scenario.Name,
			Active: scenario.Active,
			Events: scenarioEvents,
		})
	}

	// Use the configprocessor for validation
	processor := configprocessor.NewProcessor()
	return processor.ValidateConfiguration(c.Common.DeathDate, commonEvents, scenarios)
}
