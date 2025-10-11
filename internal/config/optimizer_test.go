package config

import "testing"

func TestCanonicalOptimizerField(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty defaults to amount", input: "", expected: OptimizerFieldAmount},
		{name: "amount casing", input: "Amount", expected: OptimizerFieldAmount},
		{name: "frequency", input: "FREQUENCY", expected: OptimizerFieldFrequency},
		{name: "start date variations", input: "start_date", expected: OptimizerFieldStartDate},
		{name: "end date variations", input: "END-DATE", expected: OptimizerFieldEndDate},
		{name: "unknown lowered", input: "Custom", expected: "custom"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			actual := CanonicalOptimizerField(tc.input)
			if actual != tc.expected {
				t.Fatalf("expected %q, got %q", tc.expected, actual)
			}
		})
	}
}

func TestOptimizerConfigNormalizeCanonicalField(t *testing.T) {
	testCases := []struct {
		name      string
		field     string
		tolerance float64
		expected  string
		discrete  bool
	}{
		{name: "amount default", field: "", expected: OptimizerFieldAmount, discrete: false},
		{name: "start date", field: "StartDate", expected: OptimizerFieldStartDate, discrete: true},
		{name: "end date", field: "enddate", expected: OptimizerFieldEndDate, discrete: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := &OptimizerConfig{
				Field:   tc.field,
				Min:     floatPtr(0),
				Max:     floatPtr(1),
				MinDate: "2025-01",
				MaxDate: "2025-12",
			}
			cfg.Normalize()

			if cfg.Field != tc.expected {
				t.Fatalf("expected field %q, got %q", tc.expected, cfg.Field)
			}

			if tc.discrete {
				expected := float64(defaultToleranceDiscrete)
				if cfg.Tolerance != expected {
					t.Fatalf("expected discrete tolerance %.2f, got %.2f", expected, cfg.Tolerance)
				}
			} else {
				expected := defaultToleranceAmount
				if cfg.Tolerance != expected {
					t.Fatalf("expected amount tolerance %.2f, got %.2f", expected, cfg.Tolerance)
				}
			}
		})
	}
}

func floatPtr(value float64) *float64 {
	return &value
}
