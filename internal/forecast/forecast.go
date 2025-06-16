// Package forecast defines the data structures related to a given forecast and
// includes functions for computing the forecasts.
package forecast

import (
	"fmt"
	"time"

	"github.com/iwvelando/finance-forecast/internal/config"
	"github.com/iwvelando/finance-forecast/pkg/adapters"
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
	if logger == nil {
		logger = zap.NewNop()
	}

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
		// Create a forecast engine to process monthly changes
		forecastEngine := finance.NewForecastEngine(logger)

		for {
			date, err := datetime.OffsetDate(previousDate, config.DateTimeLayout, 1)
			if err != nil {
				return results, err
			}

			// Convert events and loans using adapters
			scenarioEvents := adapters.EventsToFinanceEvents(scenario.Events)
			commonEvents := adapters.EventsToFinanceEvents(conf.Common.Events)
			scenarioLoans := adapters.LoansToFinanceLoans(scenario.Loans)
			commonLoans := adapters.LoansToFinanceLoans(conf.Common.Loans)

			// Process scenario events
			scenarioChanges, scenarioErr := forecastEngine.ProcessMonthlyChanges(date, scenarioEvents, nil, config.DateTimeLayout)
			if scenarioErr != nil {
				return results, scenarioErr
			}

			// Process common events
			commonChanges, commonErr := forecastEngine.ProcessMonthlyChanges(date, commonEvents, nil, config.DateTimeLayout)
			if commonErr != nil {
				return results, commonErr
			}

			// Check for early payoff thresholds
			projectedBalance := result.Data[previousDate] + scenarioChanges + commonChanges

			for j := range conf.Scenarios[i].Loans {
				note, payoffErr := conf.Scenarios[i].Loans[j].CheckEarlyPayoffThreshold(date, conf.Common.DeathDate, projectedBalance)
				if payoffErr != nil {
					return results, payoffErr
				}
				if note != "" {
					result.Notes[date] = append(result.Notes[date], note)
				}
			}

			for j := range conf.Common.Loans {
				note, payoffErr := conf.Common.Loans[j].CheckEarlyPayoffThreshold(date, conf.Common.DeathDate, projectedBalance)
				if payoffErr != nil {
					return results, payoffErr
				}
				if note != "" {
					result.Notes[date] = append(result.Notes[date], note)
				}
			}

			// Process loan payments
			scenarioLoansChanges, scenarioLoansErr := forecastEngine.ProcessMonthlyChanges(date, nil, scenarioLoans, config.DateTimeLayout)
			if scenarioLoansErr != nil {
				return results, scenarioLoansErr
			}

			commonLoansChanges, commonLoansErr := forecastEngine.ProcessMonthlyChanges(date, nil, commonLoans, config.DateTimeLayout)
			if commonLoansErr != nil {
				return results, commonLoansErr
			}

			// Update the balance
			result.Data[date] = result.Data[previousDate] + scenarioChanges + commonChanges + scenarioLoansChanges + commonLoansChanges
			if date == conf.Common.DeathDate {
				break
			}
			previousDate = date
		}
		results = append(results, result)
	}

	return results, nil
}
