package config

import (
	"fmt"
	"github.com/spf13/viper"
	"time"
)

const DateTimeLayout = "2006-01"

// Configuration holds all configuration for finance-forecast
type Configuration struct {
	Common    Common
	Scenarios []Scenario
}

// Common holds the shared parameters and events between all scenarios
type Common struct {
	StartingValue float64
	DeathDate     string
	Events        []Event
}

// Scenario holds all events for a given scenario
type Scenario struct {
	Name   string
	Active bool
	Events []Event
}

// Event indicates a financial event
type Event struct {
	Name      string
	Amount    float64
	StartDate string
	EndDate   string
	Frequency int // months
	DateList  []time.Time
}

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

func ParseDateLists(conf Configuration) (Configuration, error) {
	for i, scenario := range conf.Scenarios {
		for j, event := range scenario.Events {
			dateList, err := FormDateList(event, conf)
			if err != nil {
				return conf, err
			}
			conf.Scenarios[i].Events[j].DateList = dateList
		}
	}
	for i, event := range conf.Common.Events {
		dateList, err := FormDateList(event, conf)
		if err != nil {
			return conf, err
		}
		conf.Common.Events[i].DateList = dateList
	}
	return conf, nil
}

func FormDateList(event Event, conf Configuration) ([]time.Time, error) {
	dateList := make([]time.Time, 1)
	var startDateT time.Time
	var err error
	if event.StartDate == "" {
		startDateT, err = time.Parse(DateTimeLayout, time.Now().Format(DateTimeLayout))
		if err != nil {
			return dateList, err
		}
	} else {
		startDateT, err = time.Parse(DateTimeLayout, event.StartDate)
		if err != nil {
			return dateList, err
		}
	}
	if event.EndDate == "" {
		event.EndDate = conf.Common.DeathDate
	}
	endDateT, err := time.Parse(DateTimeLayout, event.EndDate)
	if err != nil {
		return dateList, err
	}
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
	return dateList, nil
}
