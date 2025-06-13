// Package forecast defines the data structures related to a given forecast and
// includes functions for computing the forecasts.
package forecast

import (
	"fmt"
	"time"

	"github.com/iwvelando/finance-forecast/internal/config"
	"github.com/iwvelando/finance-forecast/pkg/datetime"
	"github.com/iwvelando/finance-forecast/pkg/finance"
	"go.uber.org/zap"
)

// Forecast holds all information related to a specific forecast.
type Forecast struct {
	Name  string
	Data  map[string]float64
	Notes map[string][]string
}

// GetForecast processes the Forecasts for all Scenarios.
func GetForecast(logger *zap.Logger, conf config.Configuration) ([]Forecast, error) {
	var results []Forecast
	startDate := time.Now().Format(config.DateTimeLayout)
	for i, scenario := range conf.Scenarios {
		if !scenario.Active {
			logger.Debug(fmt.Sprintf("skipping scenario %s because it is inactive", scenario.Name),
				zap.String("op", "forecast.GetForecast"),
			)
			continue
		}

		// Loop through time until death and process events along the way.
		var result Forecast
		result.Name = scenario.Name
		result.Data = make(map[string]float64)
		result.Notes = make(map[string][]string)
		result.Data[startDate] = conf.Common.StartingValue
		previousDate := startDate
		for {
			date, err := datetime.OffsetDate(previousDate, config.DateTimeLayout, 1)
			if err != nil {
				return results, err
			}
			eventsAmount, err := HandleEvents(logger, date, scenario.Events, config.DateTimeLayout)
			if err != nil {
				return results, err
			}
			commonEventsAmount, err := HandleEvents(logger, date, conf.Common.Events, config.DateTimeLayout)
			if err != nil {
				return results, err
			}
			for j := range conf.Scenarios[i].Loans {
				note, err := conf.Scenarios[i].Loans[j].CheckEarlyPayoffThreshold(logger, date, conf.Common.DeathDate, result.Data[previousDate]+eventsAmount+commonEventsAmount)
				if err != nil {
					return results, err
				}
				if note != "" {
					result.Notes[date] = append(result.Notes[date], note)
				}
			}
			for j := range conf.Common.Loans {
				note, err := conf.Common.Loans[j].CheckEarlyPayoffThreshold(logger, date, conf.Common.DeathDate, result.Data[previousDate]+eventsAmount+commonEventsAmount)
				if err != nil {
					return results, err
				}
				if note != "" {
					result.Notes[date] = append(result.Notes[date], note)
				}
			}
			loansAmount := HandleLoans(logger, date, scenario.Loans)
			commonLoansAmount := HandleLoans(logger, date, conf.Common.Loans)
			result.Data[date] = result.Data[previousDate] + eventsAmount + commonEventsAmount + loansAmount + commonLoansAmount
			if date == conf.Common.DeathDate {
				break
			}
			previousDate = date
		}
		results = append(results, result)
	}

	return results, nil
}

// HandleEvents sums all amounts for Events that occur on the input date.
func HandleEvents(logger *zap.Logger, date string, events []config.Event, layout string) (float64, error) {
	// Convert config events to finance event interface
	var financeEvents []finance.EventWithDates
	for _, event := range events {
		financeEvents = append(financeEvents, configEventWrapper{event})
	}

	processor := finance.NewEventProcessor(logger)
	return processor.ProcessEventsForDate(date, financeEvents, layout)
}

// configEventWrapper wraps config.Event to implement finance.EventWithDates
type configEventWrapper struct {
	event config.Event
}

func (w configEventWrapper) GetName() string {
	return w.event.Name
}

func (w configEventWrapper) GetAmount() float64 {
	return w.event.Amount
}

func (w configEventWrapper) GetDateList() []time.Time {
	return w.event.DateList
}

// HandleLoans identifies any loan-based financial events that occur on the
// input date.
func HandleLoans(logger *zap.Logger, date string, loans []config.Loan) float64 {
	amount := 0.0
	for _, loan := range loans {
		if payment, present := loan.AmortizationSchedule[date]; present {
			logger.Debug(fmt.Sprintf("%s: loan %s is active for amount %.2f", date, loan.Name, payment.Payment),
				zap.String("op", "forecast.HandleLoans"),
			)
			amount -= payment.Payment
			continue
		}
	}
	return amount
}
