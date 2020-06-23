// Package forecast defines the data structures related to a given forecast and
// includes functions for computing the forecasts.
package forecast

import (
	"github.com/iwvelando/finance-forecast/config"
	"go.uber.org/zap"
	"time"
)

// Forecast holds all information related to a specific forecast.
type Forecast struct {
	Name string
	Data map[string]float64
}

// GetForecast processes the Forecasts for all Scenarios.
func GetForecast(logger *zap.Logger, conf config.Configuration) ([]Forecast, error) {
	var results []Forecast
	startDate := time.Now().Format(config.DateTimeLayout)
	for _, scenario := range conf.Scenarios {
		if !scenario.Active {
			continue
		}

		// Loop through time until death and process events along the way.
		var result Forecast
		result.Name = scenario.Name
		result.Data = make(map[string]float64)
		result.Data[startDate] = conf.Common.StartingValue
		previousDate := startDate
		for {
			date, err := IncrementDate(previousDate, config.DateTimeLayout)
			if err != nil {
				return results, err
			}
			eventsAmount, err := HandleEvents(date, scenario.Events, config.DateTimeLayout)
			if err != nil {
				return results, err
			}
			commonEventsAmount, err := HandleEvents(date, conf.Common.Events, config.DateTimeLayout)
			if err != nil {
				return results, err
			}
			result.Data[date] = result.Data[previousDate] + eventsAmount + commonEventsAmount
			if date == conf.Common.DeathDate {
				break
			}
			previousDate = date
		}
		results = append(results, result)
	}

	return results, nil
}

// IncrementDate returns the next string-formatted date following the input
// date; this is always a 1-month increment.
func IncrementDate(previousDate string, layout string) (string, error) {
	t, err := time.Parse(layout, previousDate)
	if err != nil {
		return previousDate, err
	}
	nextDate := t.AddDate(0, 1, 0).Format(layout)
	return nextDate, nil
}

// HandleEvents sums all amounts for Events that occur on the input date.
func HandleEvents(date string, events []config.Event, layout string) (float64, error) {
	amount := 0.0
	dateT, err := time.Parse(layout, date)
	if err != nil {
		return amount, err
	}
	for _, event := range events {
		for _, eventDate := range event.DateList {
			if dateT.Equal(eventDate) {
				amount += event.Amount
				break
			}
		}
	}
	return amount, nil
}
