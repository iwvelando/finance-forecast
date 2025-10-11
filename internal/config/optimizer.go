package config

import (
	"fmt"
	"strings"
	"time"
)

const (
	OptimizerFieldAmount    = "amount"
	OptimizerFieldFrequency = "frequency"
	OptimizerFieldStartDate = "startDate"
	OptimizerFieldEndDate   = "endDate"

	OptimizerKindCashFloor       = "cash_floor"
	OptimizerTargetEmergencyFund = "emergencyFund"

	defaultToleranceAmount   = 0.01
	defaultToleranceDiscrete = 1
	defaultMaxIterations     = 50
)

// OptimizerConfig defines a single-parameter optimization directive.
type OptimizerConfig struct {
	Field         string   `yaml:"field,omitempty" mapstructure:"field"`
	Kind          string   `yaml:"kind,omitempty" mapstructure:"kind"`
	Target        string   `yaml:"target,omitempty" mapstructure:"target"`
	Min           *float64 `yaml:"min,omitempty" mapstructure:"min"`
	Max           *float64 `yaml:"max,omitempty" mapstructure:"max"`
	MinDate       string   `yaml:"minDate,omitempty" mapstructure:"minDate"`
	MaxDate       string   `yaml:"maxDate,omitempty" mapstructure:"maxDate"`
	Tolerance     float64  `yaml:"tolerance,omitempty" mapstructure:"tolerance"`
	MaxIterations int      `yaml:"maxIterations,omitempty" mapstructure:"maxIterations"`
}

// CanonicalOptimizerField returns the canonical identifier for an optimizer field.
func CanonicalOptimizerField(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return OptimizerFieldAmount
	}
	switch strings.ToLower(trimmed) {
	case "amount":
		return OptimizerFieldAmount
	case "frequency":
		return OptimizerFieldFrequency
	case "startdate", "start_date", "start-date":
		return OptimizerFieldStartDate
	case "enddate", "end_date", "end-date":
		return OptimizerFieldEndDate
	default:
		return strings.ToLower(trimmed)
	}
}

// Normalize ensures defaults and canonical values are applied before validation.
func (o *OptimizerConfig) Normalize() {
	if o == nil {
		return
	}
	o.Field = CanonicalOptimizerField(o.Field)
	if o.Field == "" {
		o.Field = OptimizerFieldAmount
	}

	o.Kind = strings.ToLower(strings.TrimSpace(o.Kind))
	if o.Kind == "" {
		o.Kind = OptimizerKindCashFloor
	}

	o.Target = strings.TrimSpace(o.Target)
	if o.Target == "" {
		o.Target = OptimizerTargetEmergencyFund
	}

	switch o.Field {
	case OptimizerFieldAmount:
		if o.Tolerance <= 0 {
			o.Tolerance = defaultToleranceAmount
		}
	case OptimizerFieldFrequency, OptimizerFieldStartDate, OptimizerFieldEndDate:
		if o.Tolerance <= 0 {
			o.Tolerance = defaultToleranceDiscrete
		}
	default:
		if o.Tolerance <= 0 {
			o.Tolerance = defaultToleranceAmount
		}
	}
	if o.MaxIterations <= 0 {
		o.MaxIterations = defaultMaxIterations
	}
}

// Validate returns an error when the optimizer configuration is unsupported.
func (o *OptimizerConfig) Validate() error {
	if o == nil {
		return fmt.Errorf("optimizer configuration cannot be nil")
	}

	o.Normalize()

	switch o.Field {
	case OptimizerFieldAmount, OptimizerFieldFrequency, OptimizerFieldStartDate, OptimizerFieldEndDate:
		// supported fields
	default:
		return fmt.Errorf("optimizer field %q is not supported", o.Field)
	}
	if o.Kind != OptimizerKindCashFloor {
		return fmt.Errorf("optimizer kind %q is not supported", o.Kind)
	}
	if o.Target != OptimizerTargetEmergencyFund {
		return fmt.Errorf("optimizer target %q is not supported", o.Target)
	}

	switch o.Field {
	case OptimizerFieldAmount:
		if o.Min == nil {
			return fmt.Errorf("optimizer requires a minimum bound")
		}
		if o.Max == nil {
			return fmt.Errorf("optimizer requires a maximum bound")
		}
		if *o.Min >= *o.Max {
			return fmt.Errorf("optimizer minimum %.2f must be less than maximum %.2f", *o.Min, *o.Max)
		}
	case OptimizerFieldFrequency:
		if o.Min == nil || o.Max == nil {
			return fmt.Errorf("optimizer requires integer bounds for frequency")
		}
		if *o.Min < 1 {
			return fmt.Errorf("optimizer frequency minimum %.0f must be at least 1", *o.Min)
		}
		if *o.Min >= *o.Max {
			return fmt.Errorf("optimizer frequency minimum %.0f must be less than maximum %.0f", *o.Min, *o.Max)
		}
	case OptimizerFieldStartDate, OptimizerFieldEndDate:
		if strings.TrimSpace(o.MinDate) == "" {
			return fmt.Errorf("optimizer %s requires a minimum date", o.Field)
		}
		if strings.TrimSpace(o.MaxDate) == "" {
			return fmt.Errorf("optimizer %s requires a maximum date", o.Field)
		}
		minIndex, err := parseMonthIndex(o.MinDate)
		if err != nil {
			return fmt.Errorf("optimizer %s minimum date %q is invalid: %w", o.Field, o.MinDate, err)
		}
		maxIndex, err := parseMonthIndex(o.MaxDate)
		if err != nil {
			return fmt.Errorf("optimizer %s maximum date %q is invalid: %w", o.Field, o.MaxDate, err)
		}
		if minIndex > maxIndex {
			return fmt.Errorf("optimizer %s minimum date %s must not be after maximum date %s", o.Field, o.MinDate, o.MaxDate)
		}
	default:
		return fmt.Errorf("optimizer field %q is not supported", o.Field)
	}

	return nil
}

func parseMonthIndex(value string) (int, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, fmt.Errorf("month value cannot be empty")
	}
	t, err := time.Parse(DateTimeLayout, value)
	if err != nil {
		return 0, err
	}
	return t.Year()*12 + int(t.Month()) - 1, nil
}
