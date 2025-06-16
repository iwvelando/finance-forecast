package mathutil

import (
	"math"
	"testing"
)

func TestRound(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected float64
	}{
		{"Round up at midpoint", 1.235, 1.24},
		{"Round down below midpoint", 1.234, 1.23},
		{"No rounding needed", 1.23, 1.23},
		{"Large number", 12345.678, 12345.68},
		{"Negative number round up", -1.235, -1.24},
		{"Negative number round down", -1.234, -1.23},
		{"Zero", 0.0, 0.0},
		{"Very small positive", 0.001, 0.00},
		{"Very small negative", -0.001, 0.00},
		{"Exactly one cent", 0.01, 0.01},
		{"Nearly two cents", 0.019, 0.02},
		{"Large negative", -12345.678, -12345.68},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Round(tt.input)
			if math.Abs(result-tt.expected) > 0.001 {
				t.Errorf("Round(%v) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsZero(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected bool
	}{
		{"Exactly zero", 0.0, true},
		{"Very small positive", 0.001, true},
		{"Very small negative", -0.001, true},
		{"Just above tolerance", 0.02, false},
		{"Just below negative tolerance", -0.02, false},
		{"Exactly tolerance", 0.01, true},
		{"Exactly negative tolerance", -0.01, true},
		{"Large positive", 100.0, false},
		{"Large negative", -100.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsZero(tt.input)
			if result != tt.expected {
				t.Errorf("IsZero(%v) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsPositive(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected bool
	}{
		{"Large positive", 100.0, true},
		{"Small positive above tolerance", 0.02, true},
		{"Exactly tolerance", 0.01, false},
		{"Below tolerance", 0.001, false},
		{"Zero", 0.0, false},
		{"Negative", -1.0, false},
		{"Just above tolerance", 0.011, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPositive(tt.input)
			if result != tt.expected {
				t.Errorf("IsPositive(%v) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestIsNegative(t *testing.T) {
	tests := []struct {
		name     string
		input    float64
		expected bool
	}{
		{"Large negative", -100.0, true},
		{"Small negative below tolerance", -0.02, true},
		{"Exactly negative tolerance", -0.01, false},
		{"Above negative tolerance", -0.001, false},
		{"Zero", 0.0, false},
		{"Positive", 1.0, false},
		{"Just below negative tolerance", -0.011, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsNegative(tt.input)
			if result != tt.expected {
				t.Errorf("IsNegative(%v) = %v, expected %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestWithinTolerance(t *testing.T) {
	tests := []struct {
		name      string
		val1      float64
		val2      float64
		tolerance float64
		expected  bool
	}{
		{"Exactly equal", 1.0, 1.0, 0.1, true},
		{"Within tolerance", 1.0, 1.05, 0.1, true},
		{"Outside tolerance", 1.0, 1.15, 0.1, false},
		{"Negative values within tolerance", -1.0, -1.05, 0.1, true},
		{"Negative values outside tolerance", -1.0, -1.15, 0.1, false},
		{"Zero tolerance exact match", 1.0, 1.0, 0.0, true},
		{"Zero tolerance no match", 1.0, 1.001, 0.0, false},
		{"Large tolerance", 1.0, 5.0, 10.0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := WithinTolerance(tt.val1, tt.val2, tt.tolerance)
			if result != tt.expected {
				t.Errorf("WithinTolerance(%v, %v, %v) = %v, expected %v",
					tt.val1, tt.val2, tt.tolerance, result, tt.expected)
			}
		})
	}
}

func TestMin(t *testing.T) {
	tests := []struct {
		name     string
		a        float64
		b        float64
		expected float64
	}{
		{"First smaller", 1.0, 2.0, 1.0},
		{"Second smaller", 2.0, 1.0, 1.0},
		{"Equal values", 1.0, 1.0, 1.0},
		{"Negative numbers", -2.0, -1.0, -2.0},
		{"Mixed signs", -1.0, 1.0, -1.0},
		{"Zero and positive", 0.0, 1.0, 0.0},
		{"Zero and negative", 0.0, -1.0, -1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Min(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("Min(%v, %v) = %v, expected %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestMax(t *testing.T) {
	tests := []struct {
		name     string
		a        float64
		b        float64
		expected float64
	}{
		{"First larger", 2.0, 1.0, 2.0},
		{"Second larger", 1.0, 2.0, 2.0},
		{"Equal values", 1.0, 1.0, 1.0},
		{"Negative numbers", -2.0, -1.0, -1.0},
		{"Mixed signs", -1.0, 1.0, 1.0},
		{"Zero and positive", 0.0, 1.0, 1.0},
		{"Zero and negative", 0.0, -1.0, 0.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Max(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("Max(%v, %v) = %v, expected %v", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}

func TestCalculatePercentage(t *testing.T) {
	tests := []struct {
		name     string
		value    float64
		total    float64
		expected float64
	}{
		{"50% of 100", 50.0, 100.0, 50.0},
		{"25% of 200", 50.0, 200.0, 25.0},
		{"100% of value", 100.0, 100.0, 100.0},
		{"More than 100%", 150.0, 100.0, 150.0},
		{"Zero value", 0.0, 100.0, 0.0},
		{"Zero total", 50.0, 0.0, 0.0},
		{"Both zero", 0.0, 0.0, 0.0},
		{"Negative value", -50.0, 100.0, -50.0},
		{"Negative total", 50.0, -100.0, -50.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculatePercentage(tt.value, tt.total)
			if math.Abs(result-tt.expected) > 0.001 {
				t.Errorf("CalculatePercentage(%v, %v) = %v, expected %v",
					tt.value, tt.total, result, tt.expected)
			}
		})
	}
}

func TestApplyPercentage(t *testing.T) {
	tests := []struct {
		name       string
		value      float64
		percentage float64
		expected   float64
	}{
		{"50% of 100", 100.0, 50.0, 50.0},
		{"25% of 200", 200.0, 25.0, 50.0},
		{"100% of value", 100.0, 100.0, 100.0},
		{"150% of value", 100.0, 150.0, 150.0},
		{"0% of value", 100.0, 0.0, 0.0},
		{"Percentage of zero", 0.0, 50.0, 0.0},
		{"Negative percentage", 100.0, -50.0, -50.0},
		{"Negative value", -100.0, 50.0, -50.0},
		{"Small percentage", 100.0, 1.0, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ApplyPercentage(tt.value, tt.percentage)
			if math.Abs(result-tt.expected) > 0.001 {
				t.Errorf("ApplyPercentage(%v, %v) = %v, expected %v",
					tt.value, tt.percentage, result, tt.expected)
			}
		})
	}
}

// Test edge cases and boundary conditions
func TestRoundingEdgeCases(t *testing.T) {
	// Test very large numbers
	largeNum := 999999999.999
	result := Round(largeNum)
	expected := 1000000000.00
	if math.Abs(result-expected) > 0.001 {
		t.Errorf("Round of large number failed: got %v, expected %v", result, expected)
	}

	// Test very small numbers
	smallNum := 0.0001
	result = Round(smallNum)
	expected = 0.00
	if math.Abs(result-expected) > 0.001 {
		t.Errorf("Round of small number failed: got %v, expected %v", result, expected)
	}
}

func TestToleranceBoundaryConditions(t *testing.T) {
	tolerance := 0.01

	// Test exactly at tolerance boundary
	if !IsZero(tolerance) {
		t.Errorf("Value exactly at tolerance should be considered zero")
	}

	if !IsZero(-tolerance) {
		t.Errorf("Negative value exactly at tolerance should be considered zero")
	}

	// Test just outside tolerance
	if IsZero(tolerance + 0.001) {
		t.Errorf("Value just outside tolerance should not be considered zero")
	}

	if IsZero(-tolerance - 0.001) {
		t.Errorf("Negative value just outside tolerance should not be considered zero")
	}
}
