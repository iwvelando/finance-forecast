// Package forecast defines the data structures related to a given forecast and
// includes functions for computing the forecasts.
package forecast

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/iwvelando/finance-forecast/internal/config"
	"github.com/iwvelando/finance-forecast/pkg/adapters"
	"github.com/iwvelando/finance-forecast/pkg/datetime"
	"github.com/iwvelando/finance-forecast/pkg/finance"
	"go.uber.org/zap"
)

// Forecast holds all information related to a specific forecast.
type Forecast struct {
	Name    string
	Data    map[string]float64
	Liquid  map[string]float64
	Notes   map[string][]string
	Metrics ForecastMetrics
}

// ForecastMetrics aggregates supplementary scenario insights.
type ForecastMetrics struct {
	EmergencyFund *EmergencyFundRecommendation
}

// EmergencyFundRecommendation summarizes the emergency fund target for a scenario.
type EmergencyFundRecommendation struct {
	TargetMonths           float64
	AverageMonthlyExpenses float64
	TargetAmount           float64
	InitialLiquid          float64
	FundedMonths           float64
	Shortfall              float64
	Surplus                float64
}

// GetForecast processes the Forecasts for all Scenarios.
func GetForecast(logger *zap.Logger, conf config.Configuration) ([]Forecast, error) {
	// Use configured start date or current time
	var startTime time.Time
	if conf.StartDate != "" {
		var err error
		startTime, err = time.Parse(config.DateTimeLayout, conf.StartDate)
		if err != nil {
			return nil, fmt.Errorf("invalid startDate format '%s', expected YYYY-MM: %v", conf.StartDate, err)
		}
	} else {
		startTime = time.Now()
	}
	return GetForecastWithFixedTime(logger, conf, startTime)
}

// GetForecastWithFixedTime generates forecasts using a fixed current time for deterministic testing
func GetForecastWithFixedTime(logger *zap.Logger, conf config.Configuration, fixedTime time.Time) ([]Forecast, error) {
	if logger == nil {
		logger = zap.NewNop()
	}

	var results []Forecast
	startDate := fixedTime.Format(config.DateTimeLayout)
	emergencyFundMonths := conf.EmergencyFundMonths()
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
		result.Liquid = make(map[string]float64)
		result.Notes = make(map[string][]string)
		previousDate := startDate
		// Create a forecast engine to process monthly changes
		forecastEngine := finance.NewForecastEngine(logger)

		scenarioEvents := adapters.EventsToFinanceEvents(scenario.Events)
		commonEvents := adapters.EventsToFinanceEvents(conf.Common.Events)
		scenarioLoans := adapters.LoansToFinanceLoans(scenario.Loans)
		commonLoans := adapters.LoansToFinanceLoans(conf.Common.Loans)
		scenarioInvestments := adapters.InvestmentsToFinanceInvestments(scenario.Investments)
		commonInvestments := adapters.InvestmentsToFinanceInvestments(conf.Common.Investments)

		scenarioInvestmentStates := initializeInvestmentStates(scenarioInvestments)
		commonInvestmentStates := initializeInvestmentStates(commonInvestments)

		cashBalance := conf.Common.StartingValue
		scenarioInvestmentTotal := sumInvestmentStartingValues(scenarioInvestments)
		commonInvestmentTotal := sumInvestmentStartingValues(commonInvestments)
		initialInvestmentBalance := scenarioInvestmentTotal + commonInvestmentTotal
		result.Liquid[startDate] = cashBalance
		result.Data[startDate] = cashBalance + initialInvestmentBalance

		monthsObserved := 0
		totalMonthlyExpenses := 0.0
		for {
			date, err := datetime.OffsetDate(previousDate, config.DateTimeLayout, 1)
			if err != nil {
				return results, err
			}

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

			// Process investments
			scenarioInvestmentChange, scenarioInvestmentDetails, scenarioInvestErr := forecastEngine.ProcessInvestments(date, scenarioInvestments, config.DateTimeLayout, scenarioInvestmentStates)
			if scenarioInvestErr != nil {
				return results, scenarioInvestErr
			}

			commonInvestmentChange, commonInvestmentDetails, commonInvestErr := forecastEngine.ProcessInvestments(date, commonInvestments, config.DateTimeLayout, commonInvestmentStates)
			if commonInvestErr != nil {
				return results, commonInvestErr
			}

			scenarioContributionOffset := sumIncomeReducingContributions(scenarioInvestmentDetails)
			commonContributionOffset := sumIncomeReducingContributions(commonInvestmentDetails)
			scenarioWithdrawalCash := sumWithdrawals(scenarioInvestmentDetails)
			commonWithdrawalCash := sumWithdrawals(commonInvestmentDetails)

			addInvestmentNotes(result.Notes, date, "scenario", scenarioInvestmentDetails)
			addInvestmentNotes(result.Notes, date, "common", commonInvestmentDetails)

			// Check for early payoff thresholds
			projectedBalance := result.Data[previousDate] + scenarioChanges + commonChanges - scenarioContributionOffset - commonContributionOffset + scenarioInvestmentChange + commonInvestmentChange

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

			cashDelta := scenarioChanges + commonChanges + scenarioLoansChanges + commonLoansChanges
			cashDelta -= scenarioContributionOffset + commonContributionOffset
			cashDelta += scenarioWithdrawalCash + commonWithdrawalCash
			cashBalance += cashDelta

			monthlyExpenses := calculateMonthlyExpenses(MonthlyExpenseInputs{
				ScenarioEvents:     scenarioChanges,
				CommonEvents:       commonChanges,
				ScenarioLoans:      scenarioLoansChanges,
				CommonLoans:        commonLoansChanges,
				OtherContributions: []float64{scenarioContributionOffset, commonContributionOffset},
			})
			totalMonthlyExpenses += monthlyExpenses
			monthsObserved++

			scenarioInvestmentTotal += scenarioInvestmentChange
			commonInvestmentTotal += commonInvestmentChange
			totalInvestments := scenarioInvestmentTotal + commonInvestmentTotal

			result.Liquid[date] = cashBalance
			result.Data[date] = cashBalance + totalInvestments
			if date == conf.Common.DeathDate {
				break
			}
			previousDate = date
		}

		if emergencyFundMonths > 0 {
			averageMonthlyExpenses := 0.0
			if monthsObserved > 0 {
				averageMonthlyExpenses = totalMonthlyExpenses / float64(monthsObserved)
			}
			targetAmount := averageMonthlyExpenses * emergencyFundMonths
			initialLiquid := result.Liquid[startDate]
			fundedMonths := 0.0
			if averageMonthlyExpenses > 0 {
				fundedMonths = initialLiquid / averageMonthlyExpenses
			}
			difference := initialLiquid - targetAmount
			shortfall := 0.0
			surplus := 0.0
			if difference >= 0 {
				surplus = difference
			} else {
				shortfall = math.Abs(difference)
			}

			result.Metrics.EmergencyFund = &EmergencyFundRecommendation{
				TargetMonths:           emergencyFundMonths,
				AverageMonthlyExpenses: averageMonthlyExpenses,
				TargetAmount:           targetAmount,
				InitialLiquid:          initialLiquid,
				FundedMonths:           fundedMonths,
				Shortfall:              shortfall,
				Surplus:                surplus,
			}
		}
		results = append(results, result)
	}

	return results, nil
}

func initializeInvestmentStates(investments []finance.Investment) map[string]*finance.InvestmentState {
	states := make(map[string]*finance.InvestmentState)
	for _, inv := range investments {
		if inv == nil {
			continue
		}
		states[inv.GetName()] = &finance.InvestmentState{
			CurrentValue:     inv.GetStartingValue(),
			PrincipalBalance: inv.GetStartingValue(),
		}
	}
	return states
}

func sumInvestmentStartingValues(investments []finance.Investment) float64 {
	total := 0.0
	for _, inv := range investments {
		if inv == nil {
			continue
		}
		total += inv.GetStartingValue()
	}
	return total
}

func addInvestmentNotes(notes map[string][]string, date, scope string, changes []finance.InvestmentChange) {
	if len(changes) == 0 {
		return
	}

	for _, change := range changes {
		var parts []string
		if change.Contribution != 0 {
			label := "contribution"
			if change.ContributionFromCash {
				label = "contribution (reduces cash balance)"
			}
			parts = append(parts, fmt.Sprintf("%s %+0.2f", label, change.Contribution))
		}
		if change.Withdrawal != 0 || change.WithdrawalPercentage != 0 {
			note := fmt.Sprintf("withdrawal %+0.2f", change.Withdrawal)
			if change.WithdrawalPercentage != 0 {
				note = fmt.Sprintf("withdrawal (%0.2f%%) %+0.2f", change.WithdrawalPercentage, change.Withdrawal)
			}
			var breakdown []string
			if change.WithdrawalFromBasis != 0 {
				breakdown = append(breakdown, fmt.Sprintf("basis %+.2f", change.WithdrawalFromBasis))
			}
			if change.WithdrawalFromGrowth != 0 {
				breakdown = append(breakdown, fmt.Sprintf("growth %+.2f", change.WithdrawalFromGrowth))
			}
			if len(breakdown) > 0 {
				note = fmt.Sprintf("%s (%s)", note, strings.Join(breakdown, ", "))
			}
			parts = append(parts, note)
		}
		growthDisplay := change.GrowthBeforeTax
		if growthDisplay == 0 {
			growthDisplay = change.Growth
		}
		if growthDisplay != 0 {
			parts = append(parts, fmt.Sprintf("growth %+0.2f", growthDisplay))
		}
		if change.Tax != 0 {
			parts = append(parts, fmt.Sprintf("tax %.2f", change.Tax))
		}
		if change.WithdrawalTax != 0 {
			parts = append(parts, fmt.Sprintf("withdrawal tax %.2f", change.WithdrawalTax))
		}

		if len(parts) == 0 {
			continue
		}

		label := change.Name
		if scope != "" {
			if label != "" {
				label = fmt.Sprintf("%s %s", scope, label)
			} else {
				label = scope
			}
		}
		if label == "" {
			label = "investment"
		}

		notes[date] = append(notes[date], fmt.Sprintf("%s: %s", label, strings.Join(parts, ", ")))
	}
}

func sumIncomeReducingContributions(changes []finance.InvestmentChange) float64 {
	total := 0.0
	for _, change := range changes {
		if change.ContributionFromCash {
			total += change.Contribution
		}
	}
	return total
}

func sumWithdrawals(changes []finance.InvestmentChange) float64 {
	total := 0.0
	for _, change := range changes {
		// netCashReceived captures the actual cash that reaches the account after withdrawal taxes.
		netCashReceived := change.Withdrawal - change.WithdrawalTax
		total += netCashReceived
	}
	return total
}

// MonthlyExpenseInputs holds the inputs for calculateMonthlyExpenses.
type MonthlyExpenseInputs struct {
	ScenarioEvents     float64
	CommonEvents       float64
	ScenarioLoans      float64
	CommonLoans        float64
	OtherContributions []float64 // cash-reducing contributions, already positive
}

func calculateMonthlyExpenses(inputs MonthlyExpenseInputs) float64 {
	total := 0.0
	if inputs.ScenarioEvents < 0 {
		total += -inputs.ScenarioEvents
	}
	if inputs.CommonEvents < 0 {
		total += -inputs.CommonEvents
	}
	if inputs.ScenarioLoans < 0 {
		total += -inputs.ScenarioLoans
	}
	if inputs.CommonLoans < 0 {
		total += -inputs.CommonLoans
	}
	for _, contribution := range inputs.OtherContributions {
		if contribution > 0 {
			total += contribution
		}
	}
	return total
}
