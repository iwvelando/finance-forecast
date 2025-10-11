package optimizer

import (
	"math"
	"reflect"
	"testing"
	"time"

	"github.com/iwvelando/finance-forecast/internal/config"
	"github.com/iwvelando/finance-forecast/internal/forecast"
	formatutil "github.com/iwvelando/finance-forecast/pkg/format"
	"go.uber.org/zap"
)

func TestRunnerOptimizesEventForEmergencyFloor(t *testing.T) {
	t.Helper()

	conf := &config.Configuration{
		StartDate:       "2025-01",
		Recommendations: config.RecommendationsConfig{EmergencyFundMonths: 12},
		Common: config.Common{
			StartingValue: 10000,
			DeathDate:     "2030-12",
			Events: []config.Event{
				{
					Name:      "Expenses",
					Amount:    -1000,
					StartDate: "2025-01",
					Frequency: 1,
				},
			},
		},
		Scenarios: []config.Scenario{
			{
				Name:   "Baseline",
				Active: true,
				Events: []config.Event{
					{
						Name:      "Income",
						Amount:    2000,
						StartDate: "2025-01",
						EndDate:   "2025-12",
						Frequency: 1,
					},
					{
						Name:      "New Job",
						Amount:    2000,
						StartDate: "2026-01",
						Frequency: 1,
						Optimizer: &config.OptimizerConfig{
							Field:     config.OptimizerFieldAmount,
							Min:       floatPtr(0),
							Max:       floatPtr(2000),
							Tolerance: 1,
						},
					},
				},
			},
		},
	}

	startTime, err := time.Parse(config.DateTimeLayout, conf.StartDate)
	if err != nil {
		t.Fatalf("failed to parse start date: %v", err)
	}

	if err := conf.ParseDateListsWithFixedTime(startTime); err != nil {
		t.Fatalf("failed to parse date lists: %v", err)
	}

	if err := conf.ProcessLoans(zap.NewNop()); err != nil {
		t.Fatalf("failed to process loans: %v", err)
	}

	runner, err := NewRunner(zap.NewNop(), conf)
	if err != nil {
		t.Fatalf("failed to create optimizer runner: %v", err)
	}

	targets, err := runner.collectTargets()
	if err != nil {
		t.Fatalf("collect targets: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("expected one optimizer target, got %d", len(targets))
	}
	target := targets[0]

	baselineForecasts, err := forecast.GetForecastWithFixedTime(zap.NewNop(), *conf, startTime)
	if err != nil {
		t.Fatalf("baseline forecast failed: %v", err)
	}
	floor := 0.0
	for _, fc := range baselineForecasts {
		if fc.Name == target.scenarioName && fc.Metrics.EmergencyFund != nil {
			floor = fc.Metrics.EmergencyFund.TargetAmount
			break
		}
	}
	if floor == 0 {
		t.Fatalf("expected positive emergency fund floor")
	}

	result, err := runner.Run()
	if err != nil {
		t.Fatalf("optimizer run failed: %v", err)
	}

	optimized := conf.Scenarios[0].Events[1].Amount
	expected := 850.0
	if math.Abs(optimized-expected) > 10 {
		t.Fatalf("expected optimized amount near %.2f, got %.2f", expected, optimized)
	}

	summaries := result.Summaries["Baseline"]
	if len(summaries) != 1 {
		t.Fatalf("expected one optimization summary, got %d", len(summaries))
	}

	summary := summaries[0]
	if math.Abs(summary.Value-optimized) > 1 {
		t.Fatalf("summary value %.2f does not match optimized amount %.2f", summary.Value, optimized)
	}
	if math.Abs(summary.Floor-12000) > 1e-6 {
		t.Fatalf("unexpected floor %.2f", summary.Floor)
	}
	if !summary.Converged {
		t.Fatalf("expected converged optimization result")
	}

	forecasts, err := forecast.GetForecastWithFixedTime(zap.NewNop(), *conf, startTime)
	if err != nil {
		t.Fatalf("forecast run failed: %v", err)
	}

	result.Apply(forecasts)
	if len(forecasts) == 0 {
		t.Fatalf("expected forecasts after apply")
	}
	if len(forecasts[0].Metrics.Optimizations) != 1 {
		t.Fatalf("expected optimization summaries attached to forecast")
	}
}

func floatPtr(v float64) *float64 {
	return &v
}

func TestRunnerSetEventFieldValueRefreshesSchedule(t *testing.T) {
	conf := &config.Configuration{
		StartDate:       "2025-01",
		Recommendations: config.RecommendationsConfig{EmergencyFundMonths: 6},
		Common: config.Common{
			StartingValue: 5000,
			DeathDate:     "2027-12",
		},
		Scenarios: []config.Scenario{
			{
				Name:   "Schedule",
				Active: true,
				Events: []config.Event{
					{
						Name:      "Recurring Expense",
						Amount:    -500,
						StartDate: "2025-01",
						EndDate:   "2027-12",
						Frequency: 12,
					},
				},
			},
		},
	}

	startTime, err := time.Parse(config.DateTimeLayout, conf.StartDate)
	if err != nil {
		t.Fatalf("parse start date: %v", err)
	}

	if err := conf.ParseDateListsWithFixedTime(startTime); err != nil {
		t.Fatalf("parse date lists: %v", err)
	}

	runner, err := NewRunner(zap.NewNop(), conf)
	if err != nil {
		t.Fatalf("create runner: %v", err)
	}

	event := &conf.Scenarios[0].Events[0]
	originalDates := append([]time.Time(nil), event.DateList...)
	target := eventTarget{
		scenarioIndex: 0,
		eventIndex:    0,
		scenarioName:  conf.Scenarios[0].Name,
		event:         event,
		field:         config.OptimizerFieldFrequency,
		minValue:      1,
		maxValue:      12,
	}

	restore, state, err := runner.setEventFieldValue(target, 6)
	if err != nil {
		t.Fatalf("set event field value: %v", err)
	}
	if state.numeric != 6 {
		t.Fatalf("expected numeric state 6, got %.2f", state.numeric)
	}
	if event.Frequency != 6 {
		t.Fatalf("expected event frequency 6, got %d", event.Frequency)
	}
	if len(event.DateList) == len(originalDates) {
		t.Fatalf("expected schedule to refresh, lengths equal (%d)", len(event.DateList))
	}

	if restore != nil {
		restore()
	}
	if event.Frequency != 12 {
		t.Fatalf("expected frequency restored to 12, got %d", event.Frequency)
	}
	if !reflect.DeepEqual(event.DateList, originalDates) {
		t.Fatalf("expected date list to be restored")
	}
}

func TestRunnerEndDateOptimizerExtendsExpenseWhenFeasible(t *testing.T) {
	t.Helper()

	conf := &config.Configuration{
		StartDate:       "2025-01",
		Recommendations: config.RecommendationsConfig{EmergencyFundMonths: 12},
		Common: config.Common{
			StartingValue: 100000,
			DeathDate:     "2090-01",
		},
		Scenarios: []config.Scenario{
			{
				Name:   "Extend End Date",
				Active: true,
				Events: []config.Event{
					{
						Name:      "Income",
						Amount:    1000,
						StartDate: "2025-01",
						EndDate:   "2090-01",
						Frequency: 1,
					},
					{
						Name:      "Big Expense",
						Amount:    -10000,
						StartDate: "2030-01",
						EndDate:   "2080-01",
						Frequency: 12,
						Optimizer: &config.OptimizerConfig{
							Field:         config.OptimizerFieldEndDate,
							MinDate:       "2070-01",
							MaxDate:       "2089-01",
							Tolerance:     1,
							MaxIterations: 50,
						},
					},
				},
			},
		},
	}

	startTime, err := time.Parse(config.DateTimeLayout, conf.StartDate)
	if err != nil {
		t.Fatalf("failed to parse start date: %v", err)
	}

	if err := conf.ParseDateListsWithFixedTime(startTime); err != nil {
		t.Fatalf("failed to parse date lists: %v", err)
	}

	if err := conf.ProcessLoans(zap.NewNop()); err != nil {
		t.Fatalf("failed to process loans: %v", err)
	}

	runner, err := NewRunner(zap.NewNop(), conf)
	if err != nil {
		t.Fatalf("failed to create optimizer runner: %v", err)
	}

	result, err := runner.Run()
	if err != nil {
		t.Fatalf("optimizer run failed: %v", err)
	}

	event := conf.Scenarios[0].Events[1]
	if event.EndDate != "2089-01" {
		t.Fatalf("expected optimizer to extend end date to 2089-01, got %s", event.EndDate)
	}

	summaries := result.Summaries["Extend End Date"]
	if len(summaries) != 1 {
		t.Fatalf("expected one optimization summary, got %d", len(summaries))
	}

	summary := summaries[0]
	if !summary.Converged {
		t.Fatalf("expected converged summary")
	}
	if summary.ValueDisplay != "2089-01" {
		t.Fatalf("expected summary display 2089-01, got %s", summary.ValueDisplay)
	}
}

func TestRunnerStartDateOptimizerAdvancesExpenseWhenFeasible(t *testing.T) {
	t.Helper()

	conf := &config.Configuration{
		StartDate:       "2025-01",
		Recommendations: config.RecommendationsConfig{EmergencyFundMonths: 12},
		Common: config.Common{
			StartingValue: 100000,
			DeathDate:     "2090-01",
		},
		Scenarios: []config.Scenario{
			{
				Name:   "Advance Start Date",
				Active: true,
				Events: []config.Event{
					{
						Name:      "Income",
						Amount:    1000,
						StartDate: "2025-01",
						EndDate:   "2090-01",
						Frequency: 1,
					},
					{
						Name:      "Big Expense",
						Amount:    -10000,
						StartDate: "2030-01",
						EndDate:   "2080-01",
						Frequency: 12,
						Optimizer: &config.OptimizerConfig{
							Field:         config.OptimizerFieldStartDate,
							MinDate:       "2026-01",
							MaxDate:       "2040-01",
							Tolerance:     1,
							MaxIterations: 50,
						},
					},
				},
			},
		},
	}

	startTime, err := time.Parse(config.DateTimeLayout, conf.StartDate)
	if err != nil {
		t.Fatalf("failed to parse start date: %v", err)
	}

	if err := conf.ParseDateListsWithFixedTime(startTime); err != nil {
		t.Fatalf("failed to parse date lists: %v", err)
	}

	if err := conf.ProcessLoans(zap.NewNop()); err != nil {
		t.Fatalf("failed to process loans: %v", err)
	}

	runner, err := NewRunner(zap.NewNop(), conf)
	if err != nil {
		t.Fatalf("failed to create optimizer runner: %v", err)
	}

	result, err := runner.Run()
	if err != nil {
		t.Fatalf("optimizer run failed: %v", err)
	}

	event := conf.Scenarios[0].Events[1]
	if event.StartDate != "2026-01" {
		t.Fatalf("expected optimizer to advance start date to 2026-01, got %s", event.StartDate)
	}

	summaries := result.Summaries["Advance Start Date"]
	if len(summaries) != 1 {
		t.Fatalf("expected one optimization summary, got %d", len(summaries))
	}

	summary := summaries[0]
	if !summary.Converged {
		t.Fatalf("expected converged summary")
	}
	if summary.ValueDisplay != "2026-01" {
		t.Fatalf("expected summary display 2026-01, got %s", summary.ValueDisplay)
	}
}

func TestRunnerAmountOptimizerPrefersMinimumWhenHeadroomUnaffected(t *testing.T) {
	t.Helper()

	conf := &config.Configuration{
		StartDate:       "2025-01",
		Recommendations: config.RecommendationsConfig{EmergencyFundMonths: 12},
		Common: config.Common{
			StartingValue: 100000,
			DeathDate:     "2090-01",
		},
		Scenarios: []config.Scenario{
			{
				Name:   "Consume Expense",
				Active: true,
				Events: []config.Event{
					{
						Name:      "Income",
						Amount:    3000,
						StartDate: "2025-01",
						EndDate:   "2090-01",
						Frequency: 1,
					},
					{
						Name:      "Big Expense",
						Amount:    -10000,
						StartDate: "2028-01",
						EndDate:   "2080-01",
						Frequency: 12,
						Optimizer: &config.OptimizerConfig{
							Field:         config.OptimizerFieldAmount,
							Min:           floatPtr(-30000),
							Max:           floatPtr(-1000),
							Tolerance:     0.01,
							MaxIterations: 50,
						},
					},
				},
			},
		},
	}

	startTime, err := time.Parse(config.DateTimeLayout, conf.StartDate)
	if err != nil {
		t.Fatalf("failed to parse start date: %v", err)
	}

	if err := conf.ParseDateListsWithFixedTime(startTime); err != nil {
		t.Fatalf("failed to parse date lists: %v", err)
	}

	if err := conf.ProcessLoans(zap.NewNop()); err != nil {
		t.Fatalf("failed to process loans: %v", err)
	}

	runner, err := NewRunner(zap.NewNop(), conf)
	if err != nil {
		t.Fatalf("failed to create optimizer runner: %v", err)
	}

	result, err := runner.Run()
	if err != nil {
		t.Fatalf("optimizer run failed: %v", err)
	}

	event := conf.Scenarios[0].Events[1]
	if event.Amount != -30000 {
		t.Fatalf("expected optimizer to use minimum amount -30000, got %v", event.Amount)
	}

	summaries := result.Summaries["Consume Expense"]
	if len(summaries) != 1 {
		t.Fatalf("expected one optimization summary, got %d", len(summaries))
	}

	summary := summaries[0]
	if !summary.Converged {
		t.Fatalf("expected converged summary")
	}
	if summary.Value != -30000 {
		t.Fatalf("expected summary value -30000, got %v", summary.Value)
	}
	if summary.ValueDisplay != formatutil.Currency(-30000) {
		t.Fatalf("expected summary display %s, got %s", formatutil.Currency(-30000), summary.ValueDisplay)
	}
}

func TestRunnerFrequencyOptimizerPrefersMinimumWhenHeadroomUnaffected(t *testing.T) {
	t.Helper()

	conf := &config.Configuration{
		StartDate:       "2025-01",
		Recommendations: config.RecommendationsConfig{EmergencyFundMonths: 12},
		Common: config.Common{
			StartingValue: 26000,
			DeathDate:     "2090-01",
		},
		Scenarios: []config.Scenario{
			{
				Name:   "Reduce Frequency",
				Active: true,
				Events: []config.Event{
					{
						Name:      "Income",
						Amount:    4000,
						StartDate: "2025-01",
						EndDate:   "2090-01",
						Frequency: 1,
					},
					{
						Name:      "Living Expenses",
						Amount:    -2200,
						StartDate: "2025-01",
						EndDate:   "2090-01",
						Frequency: 1,
					},
					{
						Name:      "Big Expense",
						Amount:    -10000,
						StartDate: "2033-09",
						EndDate:   "2085-09",
						Frequency: 96,
						Optimizer: &config.OptimizerConfig{
							Field:         config.OptimizerFieldFrequency,
							Min:           floatPtr(24),
							Max:           floatPtr(120),
							Tolerance:     1,
							MaxIterations: 50,
						},
					},
				},
			},
		},
	}

	startTime, err := time.Parse(config.DateTimeLayout, conf.StartDate)
	if err != nil {
		t.Fatalf("failed to parse start date: %v", err)
	}

	if err := conf.ParseDateListsWithFixedTime(startTime); err != nil {
		t.Fatalf("failed to parse date lists: %v", err)
	}

	if err := conf.ProcessLoans(zap.NewNop()); err != nil {
		t.Fatalf("failed to process loans: %v", err)
	}

	runner, err := NewRunner(zap.NewNop(), conf)
	if err != nil {
		t.Fatalf("failed to create optimizer runner: %v", err)
	}

	result, err := runner.Run()
	if err != nil {
		t.Fatalf("optimizer run failed: %v", err)
	}

	event := conf.Scenarios[0].Events[2]
	if event.Frequency != 24 {
		t.Fatalf("expected optimizer to use minimum frequency 24, got %d", event.Frequency)
	}

	summaries := result.Summaries["Reduce Frequency"]
	if len(summaries) != 1 {
		t.Fatalf("expected one optimization summary, got %d", len(summaries))
	}

	summary := summaries[0]
	if !summary.Converged {
		t.Fatalf("expected converged summary")
	}
	if summary.Value != 24 {
		t.Fatalf("expected summary value 24, got %v", summary.Value)
	}
	if summary.ValueDisplay != "24" {
		t.Fatalf("expected summary display 24, got %s", summary.ValueDisplay)
	}
}

func TestRunnerHandlesInfeasibleUpperBound(t *testing.T) {
	t.Helper()

	conf := &config.Configuration{
		StartDate:       "2025-01",
		Recommendations: config.RecommendationsConfig{EmergencyFundMonths: 12},
		Common: config.Common{
			StartingValue: 10000,
			DeathDate:     "2026-12",
			Events: []config.Event{
				{
					Name:      "Expenses",
					Amount:    -4000,
					StartDate: "2025-01",
					Frequency: 1,
				},
			},
		},
		Scenarios: []config.Scenario{
			{
				Name:   "Infeasible",
				Active: true,
				Events: []config.Event{
					{
						Name:      "Base income",
						Amount:    1000,
						StartDate: "2025-01",
						Frequency: 1,
					},
					{
						Name:      "Supplement",
						Amount:    1500,
						StartDate: "2025-01",
						Frequency: 1,
						Optimizer: &config.OptimizerConfig{
							Field: config.OptimizerFieldAmount,
							Min:   floatPtr(0),
							Max:   floatPtr(2000),
						},
					},
				},
			},
		},
	}

	startTime, err := time.Parse(config.DateTimeLayout, conf.StartDate)
	if err != nil {
		t.Fatalf("failed to parse start date: %v", err)
	}

	if err := conf.ParseDateListsWithFixedTime(startTime); err != nil {
		t.Fatalf("failed to parse date lists: %v", err)
	}

	if err := conf.ProcessLoans(zap.NewNop()); err != nil {
		t.Fatalf("failed to process loans: %v", err)
	}

	runner, err := NewRunner(zap.NewNop(), conf)
	if err != nil {
		t.Fatalf("failed to create optimizer runner: %v", err)
	}

	targets, err := runner.collectTargets()
	if err != nil {
		t.Fatalf("collect targets: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("expected one optimizer target, got %d", len(targets))
	}
	target := targets[0]

	baselineForecasts, err := forecast.GetForecastWithFixedTime(zap.NewNop(), *conf, startTime)
	if err != nil {
		t.Fatalf("baseline forecast failed: %v", err)
	}
	floor := 0.0
	for _, fc := range baselineForecasts {
		if fc.Name == target.scenarioName && fc.Metrics.EmergencyFund != nil {
			floor = fc.Metrics.EmergencyFund.TargetAmount
			break
		}
	}
	if floor <= 0 {
		t.Fatalf("expected positive emergency fund floor")
	}

	result, err := runner.Run()
	if err != nil {
		t.Fatalf("optimizer run failed: %v", err)
	}

	summaries := result.Summaries["Infeasible"]
	if len(summaries) != 1 {
		t.Fatalf("expected one optimization summary, got %d", len(summaries))
	}

	summary := summaries[0]
	if summary.Converged {
		t.Fatalf("expected non-converged summary for infeasible bounds")
	}
	if summary.Value != 2000 {
		t.Fatalf("expected optimization value to use max bound 2000, got %.2f", summary.Value)
	}
	if len(summary.Notes) == 0 {
		t.Fatalf("expected summary to contain explanatory notes")
	}
	if summary.Headroom >= 0 {
		t.Fatalf("expected negative headroom when floor is not reached, got %.2f", summary.Headroom)
	}

	forecasts, err := forecast.GetForecastWithFixedTime(zap.NewNop(), *conf, startTime)
	if err != nil {
		t.Fatalf("forecast run failed: %v", err)
	}

	result.Apply(forecasts)
	if len(forecasts) == 0 {
		t.Fatalf("expected forecasts after apply")
	}
	if len(forecasts[0].Metrics.Optimizations) != 1 {
		t.Fatalf("expected optimization summaries attached to forecast")
	}
}

func TestRunnerStartDateFindsLatestFeasible(t *testing.T) {
	t.Helper()

	conf := &config.Configuration{
		StartDate:       "2025-01",
		Recommendations: config.RecommendationsConfig{EmergencyFundMonths: 6},
		Common: config.Common{
			StartingValue: 0,
			DeathDate:     "2026-12",
			Events: []config.Event{
				{
					Name:      "Core Expenses",
					Amount:    -1500,
					StartDate: "2025-01",
					EndDate:   "2026-12",
					Frequency: 1,
				},
			},
		},
		Scenarios: []config.Scenario{
			{
				Name:   "Start Date",
				Active: true,
				Events: []config.Event{
					{
						Name:      "Side Income",
						Amount:    3000,
						StartDate: "2025-06",
						EndDate:   "2026-12",
						Frequency: 1,
						Optimizer: &config.OptimizerConfig{
							Field:   config.OptimizerFieldStartDate,
							MinDate: "2025-01",
							MaxDate: "2025-12",
						},
					},
				},
			},
		},
	}

	startTime, err := time.Parse(config.DateTimeLayout, conf.StartDate)
	if err != nil {
		t.Fatalf("failed to parse start date: %v", err)
	}

	if err := conf.ParseDateListsWithFixedTime(startTime); err != nil {
		t.Fatalf("failed to parse date lists: %v", err)
	}

	if err := conf.ProcessLoans(zap.NewNop()); err != nil {
		t.Fatalf("failed to process loans: %v", err)
	}

	runner, err := NewRunner(zap.NewNop(), conf)
	if err != nil {
		t.Fatalf("failed to create optimizer runner: %v", err)
	}

	targets, err := runner.collectTargets()
	if err != nil {
		t.Fatalf("collect targets: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("expected one optimizer target, got %d", len(targets))
	}
	target := targets[0]

	baselineForecasts, err := forecast.GetForecastWithFixedTime(zap.NewNop(), *conf, startTime)
	if err != nil {
		t.Fatalf("baseline forecast failed: %v", err)
	}
	floor := 0.0
	for _, fc := range baselineForecasts {
		if fc.Name == target.scenarioName && fc.Metrics.EmergencyFund != nil {
			floor = fc.Metrics.EmergencyFund.TargetAmount
			break
		}
	}
	if floor <= 0 {
		t.Fatalf("expected positive emergency fund floor")
	}

	result, err := runner.Run()
	if err != nil {
		t.Fatalf("optimizer run failed: %v", err)
	}

	optimizedStart := conf.Scenarios[0].Events[0].StartDate
	if optimizedStart != "2025-10" {
		t.Fatalf("expected optimizer to choose latest feasible start date 2025-10, got %s", optimizedStart)
	}

	summaries := result.Summaries["Start Date"]
	if len(summaries) != 1 {
		t.Fatalf("expected one optimization summary, got %d", len(summaries))
	}

	summary := summaries[0]
	if !summary.Converged {
		t.Fatalf("expected converged summary for feasible lower bound")
	}
	if summary.Iterations == 0 {
		t.Fatalf("expected optimizer to iterate when searching interior values")
	}

	minIndex, err := monthIndexFromString("2025-10")
	if err != nil {
		t.Fatalf("failed to compute month index: %v", err)
	}
	if summary.Value != float64(minIndex) {
		t.Fatalf("expected summary value %.0f to match latest feasible month index, got %.0f", float64(minIndex), summary.Value)
	}
	if summary.ValueDisplay != "2025-10" {
		t.Fatalf("expected summary display 2025-10, got %s", summary.ValueDisplay)
	}
}

func TestRunnerAmountOptimizerHandlesInfrequentExpenses(t *testing.T) {
	t.Helper()

	conf := &config.Configuration{
		StartDate:       "2030-01",
		Recommendations: config.RecommendationsConfig{EmergencyFundMonths: 24},
		Common: config.Common{
			StartingValue: 1000,
			DeathDate:     "2070-12",
		},
		Scenarios: []config.Scenario{
			{
				Name:   "Infrequent Expense",
				Active: true,
				Events: []config.Event{
					{
						Name:      "Base Income",
						Amount:    4000,
						StartDate: "2030-01",
						Frequency: 1,
					},
					{
						Name:      "Monthly Expenses",
						Amount:    -3850,
						StartDate: "2030-01",
						Frequency: 1,
					},
					{
						Name:      "Insurance",
						Amount:    -200,
						StartDate: "2030-01",
						Frequency: 12,
					},
					{
						Name:      "Infrequent Purchase",
						Amount:    -6500,
						StartDate: "2033-09",
						EndDate:   "2068-01",
						Frequency: 96,
						Optimizer: &config.OptimizerConfig{
							Field: config.OptimizerFieldAmount,
							Min:   floatPtr(-1500),
							Max:   floatPtr(-250),
						},
					},
				},
			},
		},
	}

	startTime, err := time.Parse(config.DateTimeLayout, conf.StartDate)
	if err != nil {
		t.Fatalf("failed to parse start date: %v", err)
	}

	if err := conf.ParseDateListsWithFixedTime(startTime); err != nil {
		t.Fatalf("failed to parse date lists: %v", err)
	}

	if err := conf.ProcessLoans(zap.NewNop()); err != nil {
		t.Fatalf("failed to process loans: %v", err)
	}

	runner, err := NewRunner(zap.NewNop(), conf)
	if err != nil {
		t.Fatalf("failed to create optimizer runner: %v", err)
	}

	result, err := runner.Run()
	if err != nil {
		t.Fatalf("optimizer run failed: %v", err)
	}

	summaries := result.Summaries["Infrequent Expense"]
	if len(summaries) != 1 {
		t.Fatalf("expected one optimization summary, got %d", len(summaries))
	}

	summary := summaries[0]

	event := conf.Scenarios[0].Events[3]
	t.Logf("optimized amount: %.2f", event.Amount)
	if event.Amount <= -6500 {
		t.Fatalf("expected optimizer to reduce magnitude of infrequent expense, got %.2f (summary=%+v)", event.Amount, summary)
	}
	if !summary.Converged {
		t.Fatalf("expected converged summary")
	}
	if summary.Value <= -6500 {
		t.Fatalf("expected summary value greater than original amount, got %.2f (summary=%+v)", summary.Value, summary)
	}
	if summary.ValueDisplay == "-6500.00" {
		t.Fatalf("expected summary display to change, still %s", summary.ValueDisplay)
	}
}

func TestRunnerAmountOptimizerExpandsExpenseUntilFloor(t *testing.T) {
	t.Helper()

	conf := &config.Configuration{
		StartDate:       "2025-01",
		Recommendations: config.RecommendationsConfig{EmergencyFundMonths: 12},
		Common: config.Common{
			StartingValue: 100000,
			DeathDate:     "2090-01",
		},
		Scenarios: []config.Scenario{
			{
				Name:   "Minimal Change",
				Active: true,
				Events: []config.Event{
					{
						Name:      "Income",
						Amount:    1000,
						StartDate: "2025-01",
						EndDate:   "2090-01",
						Frequency: 1,
					},
					{
						Name:      "Big Expense",
						Amount:    -100000,
						StartDate: "2030-01",
						EndDate:   "2080-01",
						Frequency: 120,
						Optimizer: &config.OptimizerConfig{
							Field:         config.OptimizerFieldAmount,
							Min:           floatPtr(-120000),
							Max:           floatPtr(-20000),
							Tolerance:     1,
							MaxIterations: 50,
						},
					},
				},
			},
		},
	}

	startTime, err := time.Parse(config.DateTimeLayout, conf.StartDate)
	if err != nil {
		t.Fatalf("failed to parse start date: %v", err)
	}

	if err := conf.ParseDateListsWithFixedTime(startTime); err != nil {
		t.Fatalf("failed to parse date lists: %v", err)
	}

	if err := conf.ProcessLoans(zap.NewNop()); err != nil {
		t.Fatalf("failed to process loans: %v", err)
	}

	runner, err := NewRunner(zap.NewNop(), conf)
	if err != nil {
		t.Fatalf("failed to create optimizer runner: %v", err)
	}

	result, err := runner.Run()
	if err != nil {
		t.Fatalf("optimizer run failed: %v", err)
	}

	event := conf.Scenarios[0].Events[1]
	if event.Amount != -120000 {
		t.Fatalf("expected optimizer to choose minimum amount -120000, got %.2f", event.Amount)
	}

	summaries := result.Summaries["Minimal Change"]
	if len(summaries) != 1 {
		t.Fatalf("expected one optimization summary, got %d", len(summaries))
	}

	summary := summaries[0]
	if !summary.Converged {
		t.Fatalf("expected converged summary")
	}
	if summary.Value != -120000 {
		t.Fatalf("expected summary value to reach -120000, got %.2f", summary.Value)
	}
	if summary.ValueDisplay != "-$120,000.00" {
		t.Fatalf("expected summary display -$120,000.00, got %s", summary.ValueDisplay)
	}
}
