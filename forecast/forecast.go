package forecast

import (
	"github.com/iwvelando/finance-forecast/config"
	"go.uber.org/zap"
	"time"
)

type Forecast struct {
	Name string
	Data map[string]float64
}

func GetForecast(logger *zap.Logger, conf config.Configuration) ([]Forecast, error) {
	var results []Forecast
	startDate := time.Now().Format(config.DateTimeLayout)
	for _, scenario := range conf.Scenarios {
		if !scenario.Active {
			continue
		}
		var result Forecast
		result.Name = scenario.Name
		result.Data = make(map[string]float64)
		result.Data[startDate] = scenario.StartingValue
		previousDate := startDate
		for {
			date, err := IncrementDate(previousDate, config.DateTimeLayout)
			if err != nil {
				return results, nil
			}
			eventsAmount, err := HandleEvents(date, scenario.Events, config.DateTimeLayout)
			if err != nil {
				return results, nil
			}
			result.Data[date] = result.Data[previousDate] + eventsAmount
			if date == scenario.DeathDate {
				break
			}
			previousDate = date
		}
		results = append(results, result)
	}

	return results, nil
}

func IncrementDate(previousDate string, layout string) (string, error) {
	t, err := time.Parse(layout, previousDate)
	if err != nil {
		return previousDate, err
	}
	nextDate := t.AddDate(0, 1, 0).Format(layout)
	return nextDate, nil
}

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
