package optimizer

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/iwvelando/finance-forecast/internal/config"
	"github.com/iwvelando/finance-forecast/internal/forecast"
	"github.com/iwvelando/finance-forecast/pkg/mathutil"
	"github.com/iwvelando/finance-forecast/pkg/optimization"
	"go.uber.org/zap"
)

type Runner struct {
	logger    *zap.Logger
	conf      *config.Configuration
	fixedTime time.Time
}

type eventTarget struct {
	scenarioIndex int
	eventIndex    int
	scenarioName  string
	event         *config.Event
	field         string
	minValue      float64
	maxValue      float64
	originalState fieldState
}

type evaluation struct {
	value        float64
	display      string
	minCash      float64
	floor        float64
	floorReached bool
}

func (e evaluation) feasible() bool {
	return e.floorReached && e.minCash >= e.floor
}

func (e evaluation) headroom() float64 {
	return e.minCash - e.floor
}

type fieldState struct {
	numeric float64
	display string
}

// Result summarizes optimizer adjustments keyed by scenario name.
type Result struct {
	Summaries map[string][]optimization.Summary
}

// Empty indicates whether any optimizer adjustments were produced.
func (r Result) Empty() bool {
	return len(r.Summaries) == 0
}

// Apply attaches optimizer summaries to the provided forecast results.
func (r Result) Apply(forecasts []forecast.Forecast) {
	if len(r.Summaries) == 0 {
		return
	}
	for i := range forecasts {
		summaries, ok := r.Summaries[forecasts[i].Name]
		if !ok {
			continue
		}
		metrics := forecasts[i].Metrics
		metrics.Optimizations = append(metrics.Optimizations, summaries...)
		forecasts[i].Metrics = metrics
	}
}

// NewRunner constructs a Runner for the provided configuration.
func NewRunner(logger *zap.Logger, conf *config.Configuration) (*Runner, error) {
	if conf == nil {
		return nil, fmt.Errorf("configuration cannot be nil")
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	var fixedTime time.Time
	if conf.StartDate != "" {
		parsed, err := time.Parse(config.DateTimeLayout, conf.StartDate)
		if err != nil {
			return nil, fmt.Errorf("invalid start date %q: %w", conf.StartDate, err)
		}
		fixedTime = parsed
	} else {
		fixedTime = time.Now()
	}

	return &Runner{logger: logger, conf: conf, fixedTime: fixedTime}, nil
}

// Run executes all optimizer directives and mutates the configuration in place.
func (r *Runner) Run() (*Result, error) {
	targets, err := r.collectTargets()
	if err != nil {
		return nil, err
	}
	if len(targets) == 0 {
		return &Result{Summaries: make(map[string][]optimization.Summary)}, nil
	}

	baseline, err := forecast.GetForecastWithFixedTime(r.logger, *r.conf, r.fixedTime)
	if err != nil {
		return nil, fmt.Errorf("optimizer baseline forecast failed: %w", err)
	}

	floors := make(map[string]float64)
	for _, fc := range baseline {
		if fc.Metrics.EmergencyFund != nil {
			floors[fc.Name] = fc.Metrics.EmergencyFund.TargetAmount
		}
	}

	summaries := make(map[string][]optimization.Summary)

	for _, target := range targets {
		floor, ok := floors[target.scenarioName]
		if !ok {
			return nil, fmt.Errorf("optimizer: scenario %s missing emergency fund baseline", target.scenarioName)
		}
		if floor <= 0 {
			return nil, fmt.Errorf("optimizer: scenario %s requires a positive emergency fund target", target.scenarioName)
		}

		summary, err := r.optimizeEvent(target, floor)
		if err != nil {
			return nil, err
		}
		summaries[target.scenarioName] = append(summaries[target.scenarioName], summary)

		r.logger.Info("optimizer adjusted event field",
			zap.String("scenario", target.scenarioName),
			zap.String("event", target.event.Name),
			zap.String("field", target.field),
			zap.Float64("originalNumeric", target.originalState.numeric),
			zap.String("originalDisplay", target.originalState.display),
			zap.Float64("optimizedNumeric", summary.Value),
			zap.String("optimizedDisplay", summary.ValueDisplay),
			zap.Float64("floor", summary.Floor),
			zap.Float64("minCash", summary.MinimumCash),
			zap.Float64("headroom", summary.Headroom),
			zap.Int("iterations", summary.Iterations),
			zap.Bool("converged", summary.Converged),
		)
	}

	return &Result{Summaries: summaries}, nil
}

func (r *Runner) collectTargets() ([]eventTarget, error) {
	var targets []eventTarget

	for i := range r.conf.Scenarios {
		scenario := &r.conf.Scenarios[i]
		if !scenario.Active {
			continue
		}
		for j := range scenario.Events {
			event := &scenario.Events[j]
			if event.Optimizer == nil {
				continue
			}
			if err := event.Optimizer.Validate(); err != nil {
				return nil, fmt.Errorf("scenario %s event %s: %w", scenario.Name, event.Name, err)
			}
			field := config.CanonicalOptimizerField(event.Optimizer.Field)
			minValue, maxValue, err := boundsForField(event.Optimizer)
			if err != nil {
				return nil, fmt.Errorf("scenario %s event %s: %w", scenario.Name, event.Name, err)
			}
			state, err := getEventFieldState(event, field)
			if err != nil {
				return nil, fmt.Errorf("scenario %s event %s: %w", scenario.Name, event.Name, err)
			}
			targets = append(targets, eventTarget{
				scenarioIndex: i,
				eventIndex:    j,
				scenarioName:  scenario.Name,
				event:         event,
				field:         field,
				minValue:      minValue,
				maxValue:      maxValue,
				originalState: state,
			})
		}
	}

	for _, event := range r.conf.Common.Events {
		if event.Optimizer != nil {
			return nil, fmt.Errorf("optimizer directives on common events are not supported (event %s)", event.Name)
		}
	}

	return targets, nil
}

func (r *Runner) optimizeEvent(target eventTarget, floor float64) (optimization.Summary, error) {
	cfg := target.event.Optimizer
	if cfg == nil {
		return optimization.Summary{}, fmt.Errorf("optimizer configuration missing for event %s", target.event.Name)
	}

	minVal := target.minValue
	maxVal := target.maxValue

	lowerEval, err := r.evaluateTarget(target, minVal, floor)
	if err != nil {
		return optimization.Summary{}, err
	}
	upperEval, err := r.evaluateTarget(target, maxVal, floor)
	if err != nil {
		return optimization.Summary{}, err
	}

	searchLowerEval := lowerEval
	searchUpperEval := upperEval
	if !searchLowerEval.feasible() && !searchUpperEval.feasible() {
		chasedEval := upperEval
		if lowerEval.headroom() > upperEval.headroom() {
			chasedEval = lowerEval
		}
		note := fmt.Sprintf(
			"unable to satisfy minimum cash %s within bounds %s to %s",
			formatCurrency(floor),
			formatFieldDisplay(target.field, minVal),
			formatFieldDisplay(target.field, maxVal),
		)
		_, appliedState, err := r.setEventFieldValue(target, chasedEval.value)
		if err != nil {
			return optimization.Summary{}, err
		}
		summary := optimization.Summary{
			Scope:           "scenario",
			TargetName:      target.event.Name,
			Field:           cfg.Field,
			Original:        target.originalState.numeric,
			OriginalDisplay: target.originalState.display,
			Value:           appliedState.numeric,
			ValueDisplay:    appliedState.display,
			Floor:           floor,
			MinimumCash:     chasedEval.minCash,
			Headroom:        chasedEval.headroom(),
			Iterations:      0,
			Converged:       false,
			Notes:           []string{note},
		}
		return summary, nil
	}

	if searchLowerEval.feasible() && searchUpperEval.feasible() {
		preferredValue := clampValue(snapFieldValue(target.field, target.originalState.numeric), minVal, maxVal)
		preferredEval, err := r.evaluateTarget(target, preferredValue, floor)
		if err != nil {
			return optimization.Summary{}, err
		}
		bestEval := preferredEval
		bestChosen := bestEval.feasible()
		bestHeadroom := bestEval.headroom()
		bestDelta := math.Abs(bestEval.value - target.originalState.numeric)
		fieldKind := config.CanonicalOptimizerField(target.field)
		upperIsBetter := upperEval.headroom() >= lowerEval.headroom()
		headroomEpsilon := 1e-6
		deltaEpsilon := 1e-6
		valueEpsilon := 1e-6
		preferConsume := target.event.Amount < 0 && (fieldKind == config.OptimizerFieldAmount || fieldKind == config.OptimizerFieldFrequency || fieldKind == config.OptimizerFieldStartDate || fieldKind == config.OptimizerFieldEndDate)
		preferLowerValue := false
		preferHigherValue := false
		if preferConsume {
			switch fieldKind {
			case config.OptimizerFieldAmount, config.OptimizerFieldFrequency, config.OptimizerFieldStartDate:
				preferLowerValue = true
			case config.OptimizerFieldEndDate:
				preferHigherValue = true
			}
		}

		considerCandidate := func(eval evaluation) {
			if !eval.feasible() {
				return
			}
			headroom := eval.headroom()
			delta := math.Abs(eval.value - target.originalState.numeric)

			if !bestChosen {
				bestEval = eval
				bestHeadroom = headroom
				bestDelta = delta
				bestChosen = true
				return
			}

			if preferConsume {
				if headroom < bestHeadroom-headroomEpsilon {
					bestEval = eval
					bestHeadroom = headroom
					bestDelta = delta
					return
				}
				if math.Abs(headroom-bestHeadroom) <= headroomEpsilon {
					if preferLowerValue && eval.value < bestEval.value-valueEpsilon {
						bestEval = eval
						bestHeadroom = headroom
						bestDelta = delta
						return
					}
					if preferHigherValue && eval.value > bestEval.value+valueEpsilon {
						bestEval = eval
						bestHeadroom = headroom
						bestDelta = delta
						return
					}
					if preferLowerValue && eval.value > bestEval.value+valueEpsilon {
						return
					}
					if preferHigherValue && eval.value < bestEval.value-valueEpsilon {
						return
					}
					if delta < bestDelta-deltaEpsilon {
						bestEval = eval
						bestHeadroom = headroom
						bestDelta = delta
						return
					}
					if math.Abs(delta-bestDelta) <= deltaEpsilon {
						if preferLowerValue && eval.value < bestEval.value-valueEpsilon {
							bestEval = eval
							bestHeadroom = headroom
							bestDelta = delta
							return
						}
						if preferHigherValue && eval.value > bestEval.value+valueEpsilon {
							bestEval = eval
							bestHeadroom = headroom
							bestDelta = delta
							return
						}
					}
				}
				return
			}

			if headroom > bestHeadroom+headroomEpsilon {
				bestEval = eval
				bestHeadroom = headroom
				bestDelta = delta
				return
			}
			if math.Abs(headroom-bestHeadroom) <= headroomEpsilon {
				if delta < bestDelta-deltaEpsilon {
					bestEval = eval
					bestHeadroom = headroom
					bestDelta = delta
					return
				}
				if math.Abs(delta-bestDelta) <= deltaEpsilon {
					if upperIsBetter && eval.value > bestEval.value+valueEpsilon {
						bestEval = eval
						bestHeadroom = headroom
						bestDelta = delta
					} else if !upperIsBetter && eval.value < bestEval.value-valueEpsilon {
						bestEval = eval
						bestHeadroom = headroom
						bestDelta = delta
					}
				}
			}
		}

		considerCandidate(preferredEval)
		considerCandidate(lowerEval)
		considerCandidate(upperEval)

		if !bestChosen {
			if preferConsume {
				if upperEval.headroom() <= lowerEval.headroom() {
					bestEval = upperEval
				} else {
					bestEval = lowerEval
				}
			} else {
				if upperEval.headroom() >= lowerEval.headroom() {
					bestEval = upperEval
				} else {
					bestEval = lowerEval
				}
			}
			bestHeadroom = bestEval.headroom()
			bestDelta = math.Abs(bestEval.value - target.originalState.numeric)
		}

		_, appliedState, err := r.setEventFieldValue(target, bestEval.value)
		if err != nil {
			return optimization.Summary{}, err
		}
		summary := optimization.Summary{
			Scope:           "scenario",
			TargetName:      target.event.Name,
			Field:           cfg.Field,
			Original:        target.originalState.numeric,
			OriginalDisplay: target.originalState.display,
			Value:           appliedState.numeric,
			ValueDisplay:    appliedState.display,
			Floor:           floor,
			MinimumCash:     bestEval.minCash,
			Headroom:        bestEval.headroom(),
			Iterations:      0,
			Converged:       bestEval.feasible(),
		}
		if !bestEval.feasible() {
			note := fmt.Sprintf(
				"unable to satisfy minimum cash %s within bounds %s to %s",
				formatCurrency(floor),
				formatFieldDisplay(target.field, minVal),
				formatFieldDisplay(target.field, maxVal),
			)
			summary.Converged = false
			summary.Notes = []string{note}
		}
		return summary, nil
	}

	iterations := 0
	finalEval := searchUpperEval
	finalValue := searchUpperEval.value

	if searchLowerEval.feasible() && !searchUpperEval.feasible() {
		finalEval = searchLowerEval
		finalValue = searchLowerEval.value
		lower := searchLowerEval.value
		upper := searchUpperEval.value
		for iterations < cfg.MaxIterations && math.Abs(upper-lower) > cfg.Tolerance {
			mid := lower + (upper-lower)/2
			evalMid, err := r.evaluateTarget(target, mid, floor)
			if err != nil {
				return optimization.Summary{}, err
			}
			iterations++
			if evalMid.feasible() {
				finalEval = evalMid
				finalValue = evalMid.value
				if evalMid.value == lower {
					break
				}
				lower = evalMid.value
			} else {
				if evalMid.value == upper {
					break
				}
				upper = evalMid.value
			}
		}
	} else if !searchLowerEval.feasible() && searchUpperEval.feasible() {
		finalEval = searchUpperEval
		finalValue = searchUpperEval.value
		lower := searchLowerEval.value
		upper := searchUpperEval.value
		for iterations < cfg.MaxIterations && math.Abs(upper-lower) > cfg.Tolerance {
			mid := lower + (upper-lower)/2
			evalMid, err := r.evaluateTarget(target, mid, floor)
			if err != nil {
				return optimization.Summary{}, err
			}
			iterations++
			if evalMid.feasible() {
				finalEval = evalMid
				finalValue = evalMid.value
				if evalMid.value == upper {
					break
				}
				upper = evalMid.value
			} else {
				if evalMid.value == lower {
					break
				}
				lower = evalMid.value
			}
		}
	} else if searchLowerEval.feasible() {
		finalEval = searchLowerEval
		finalValue = searchLowerEval.value
	}

	if !finalEval.feasible() {
		note := fmt.Sprintf(
			"unable to satisfy minimum cash %s within bounds %s to %s",
			formatCurrency(floor),
			formatFieldDisplay(target.field, minVal),
			formatFieldDisplay(target.field, maxVal),
		)
		_, appliedState, err := r.setEventFieldValue(target, finalValue)
		if err != nil {
			return optimization.Summary{}, err
		}
		summary := optimization.Summary{
			Scope:           "scenario",
			TargetName:      target.event.Name,
			Field:           cfg.Field,
			Original:        target.originalState.numeric,
			OriginalDisplay: target.originalState.display,
			Value:           appliedState.numeric,
			ValueDisplay:    appliedState.display,
			Floor:           floor,
			MinimumCash:     finalEval.minCash,
			Headroom:        finalEval.headroom(),
			Iterations:      iterations,
			Converged:       false,
			Notes:           []string{note},
		}
		return summary, nil
	}

	converged := finalEval.feasible()

	_, appliedState, err := r.setEventFieldValue(target, finalValue)
	if err != nil {
		return optimization.Summary{}, err
	}

	summary := optimization.Summary{
		Scope:           "scenario",
		TargetName:      target.event.Name,
		Field:           cfg.Field,
		Original:        target.originalState.numeric,
		OriginalDisplay: target.originalState.display,
		Value:           appliedState.numeric,
		ValueDisplay:    appliedState.display,
		Floor:           floor,
		MinimumCash:     finalEval.minCash,
		Headroom:        finalEval.headroom(),
		Iterations:      iterations,
		Converged:       converged,
	}

	return summary, nil
}

func formatCurrency(amount float64) string {
	formatted := fmt.Sprintf("%.2f", math.Abs(amount))
	parts := strings.Split(formatted, ".")
	intPart := parts[0]
	decPart := "00"
	if len(parts) > 1 {
		decPart = parts[1]
	}

	if len(intPart) > 3 {
		var builder strings.Builder
		for i, digit := range intPart {
			if i > 0 && (len(intPart)-i)%3 == 0 {
				builder.WriteByte(',')
			}
			builder.WriteRune(digit)
		}
		intPart = builder.String()
	}

	if amount < 0 {
		return "-$" + intPart + "." + decPart
	}
	return "$" + intPart + "." + decPart
}

func (r *Runner) evaluateTarget(target eventTarget, amount float64, floor float64) (evaluation, error) {
	value := snapFieldValue(target.field, amount)
	value = clampValue(value, target.minValue, target.maxValue)
	restore, appliedState, err := r.setEventFieldValue(target, value)
	if err != nil {
		return evaluation{}, err
	}
	if restore != nil {
		defer restore()
	}

	forecasts, err := forecast.GetForecastWithFixedTime(r.logger, *r.conf, r.fixedTime)
	if err != nil {
		return evaluation{}, fmt.Errorf("optimizer forecast evaluation failed: %w", err)
	}

	var scenarioForecast *forecast.Forecast
	for i := range forecasts {
		if forecasts[i].Name == target.scenarioName {
			scenarioForecast = &forecasts[i]
			break
		}
	}
	if scenarioForecast == nil {
		return evaluation{}, fmt.Errorf("optimizer: forecast missing scenario %s", target.scenarioName)
	}

	minCash, reached := minCashAfterFloor(*scenarioForecast, floor)
	return evaluation{
		value:        appliedState.numeric,
		display:      appliedState.display,
		minCash:      minCash,
		floor:        floor,
		floorReached: reached,
	}, nil
}

func minCashAfterFloor(fc forecast.Forecast, floor float64) (float64, bool) {
	if len(fc.Liquid) == 0 {
		return 0, false
	}

	var dates []string
	for date := range fc.Liquid {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	floorReached := floor <= 0
	minCash := math.MaxFloat64

	for _, date := range dates {
		cash := fc.Liquid[date]
		if !floorReached {
			if cash >= floor {
				floorReached = true
				minCash = cash
			}
			continue
		}
		if cash < minCash {
			minCash = cash
		}
	}

	if !floorReached {
		return 0, false
	}

	if minCash == math.MaxFloat64 {
		minCash = 0
	}

	return minCash, true
}

func boundsForField(cfg *config.OptimizerConfig) (float64, float64, error) {
	if cfg == nil {
		return 0, 0, fmt.Errorf("optimizer configuration cannot be nil")
	}
	cfg.Normalize()
	field := config.CanonicalOptimizerField(cfg.Field)
	switch field {
	case config.OptimizerFieldAmount:
		if cfg.Min == nil || cfg.Max == nil {
			return 0, 0, fmt.Errorf("optimizer field %s requires numeric min and max values", config.OptimizerFieldAmount)
		}
		return *cfg.Min, *cfg.Max, nil
	case config.OptimizerFieldFrequency:
		if cfg.Min == nil || cfg.Max == nil {
			return 0, 0, fmt.Errorf("optimizer field %s requires numeric min and max values", config.OptimizerFieldFrequency)
		}
		return *cfg.Min, *cfg.Max, nil
	case config.OptimizerFieldStartDate:
		minIndex, err := monthIndexFromString(cfg.MinDate)
		if err != nil {
			return 0, 0, err
		}
		maxIndex, err := monthIndexFromString(cfg.MaxDate)
		if err != nil {
			return 0, 0, err
		}
		return float64(minIndex), float64(maxIndex), nil
	case config.OptimizerFieldEndDate:
		minIndex, err := monthIndexFromString(cfg.MinDate)
		if err != nil {
			return 0, 0, err
		}
		maxIndex, err := monthIndexFromString(cfg.MaxDate)
		if err != nil {
			return 0, 0, err
		}
		return float64(minIndex), float64(maxIndex), nil
	default:
		return 0, 0, fmt.Errorf("optimizer field %q is not supported", cfg.Field)
	}
}

func getEventFieldState(event *config.Event, field string) (fieldState, error) {
	normalized := config.CanonicalOptimizerField(field)
	switch normalized {
	case config.OptimizerFieldAmount:
		value := mathutil.Round(event.Amount)
		return fieldState{numeric: value, display: formatCurrency(value)}, nil
	case config.OptimizerFieldFrequency:
		if event.Frequency <= 0 {
			return fieldState{}, fmt.Errorf("event %s must have a positive frequency", event.Name)
		}
		return fieldState{numeric: float64(event.Frequency), display: fmt.Sprintf("%d", event.Frequency)}, nil
	case config.OptimizerFieldStartDate:
		if strings.TrimSpace(event.StartDate) == "" {
			return fieldState{}, fmt.Errorf("event %s requires a startDate to optimize", event.Name)
		}
		index, err := monthIndexFromString(event.StartDate)
		if err != nil {
			return fieldState{}, err
		}
		return fieldState{numeric: float64(index), display: event.StartDate}, nil
	case config.OptimizerFieldEndDate:
		if strings.TrimSpace(event.EndDate) == "" {
			return fieldState{}, fmt.Errorf("event %s requires an endDate to optimize", event.Name)
		}
		index, err := monthIndexFromString(event.EndDate)
		if err != nil {
			return fieldState{}, err
		}
		return fieldState{numeric: float64(index), display: event.EndDate}, nil
	default:
		return fieldState{}, fmt.Errorf("optimizer field %q is not supported", field)
	}
}

func (r *Runner) setEventFieldValue(target eventTarget, value float64) (func(), fieldState, error) {
	event := target.event
	if event == nil {
		return nil, fieldState{}, fmt.Errorf("event target cannot be nil")
	}

	normalized := config.CanonicalOptimizerField(target.field)
	var restore func()
	var state fieldState
	needSchedule := false

	switch normalized {
	case config.OptimizerFieldAmount:
		previous := event.Amount
		rounded := mathutil.Round(value)
		event.Amount = rounded
		restore = func() { event.Amount = previous }
		state = fieldState{numeric: rounded, display: formatCurrency(rounded)}
	case config.OptimizerFieldFrequency:
		previous := event.Frequency
		rounded := int(math.Round(value))
		if rounded < 1 {
			rounded = 1
		}
		event.Frequency = rounded
		restore = func() { event.Frequency = previous }
		state = fieldState{numeric: float64(rounded), display: fmt.Sprintf("%d", rounded)}
		needSchedule = true
	case config.OptimizerFieldStartDate:
		previous := event.StartDate
		index := int(math.Round(value))
		if index < 0 {
			index = 0
		}
		formatted := monthIndexToString(index)
		event.StartDate = formatted
		restore = func() { event.StartDate = previous }
		state = fieldState{numeric: float64(index), display: formatted}
		needSchedule = true
	case config.OptimizerFieldEndDate:
		previous := event.EndDate
		index := int(math.Round(value))
		if index < 0 {
			index = 0
		}
		formatted := monthIndexToString(index)
		event.EndDate = formatted
		restore = func() { event.EndDate = previous }
		state = fieldState{numeric: float64(index), display: formatted}
		needSchedule = true
	default:
		return nil, fieldState{}, fmt.Errorf("optimizer field %q is not supported", target.field)
	}

	if needSchedule {
		if err := event.FormDateListWithFixedTime(*r.conf, r.fixedTime); err != nil {
			restore()
			return nil, fieldState{}, err
		}
	}

	wrappedRestore := func() {
		restore()
		if needSchedule {
			if err := event.FormDateListWithFixedTime(*r.conf, r.fixedTime); err != nil && r.logger != nil {
				r.logger.Warn("failed to rebuild event schedule after optimizer restore",
					zap.String("scenario", target.scenarioName),
					zap.String("event", target.event.Name),
					zap.Error(err),
				)
			}
		}
	}

	return wrappedRestore, state, nil
}

func snapFieldValue(field string, value float64) float64 {
	switch config.CanonicalOptimizerField(field) {
	case config.OptimizerFieldAmount:
		return mathutil.Round(value)
	case config.OptimizerFieldFrequency, config.OptimizerFieldStartDate, config.OptimizerFieldEndDate:
		return math.Round(value)
	default:
		return value
	}
}

func clampValue(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func formatFieldDisplay(field string, value float64) string {
	switch config.CanonicalOptimizerField(field) {
	case config.OptimizerFieldAmount:
		return formatCurrency(mathutil.Round(value))
	case config.OptimizerFieldFrequency:
		return fmt.Sprintf("%d", int(math.Round(value)))
	case config.OptimizerFieldStartDate, config.OptimizerFieldEndDate:
		return monthIndexToString(int(math.Round(value)))
	default:
		return fmt.Sprintf("%.2f", value)
	}
}

func monthIndexFromString(value string) (int, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, fmt.Errorf("month value cannot be empty")
	}
	t, err := time.Parse(config.DateTimeLayout, trimmed)
	if err != nil {
		return 0, err
	}
	return t.Year()*12 + int(t.Month()) - 1, nil
}

func monthIndexToString(index int) string {
	if index < 0 {
		index = 0
	}
	year := index / 12
	month := index%12 + 1
	return fmt.Sprintf("%04d-%02d", year, month)
}
