// Package mathutil provides common mathematical utility functions.
package mathutil

import (
	"math"

	"github.com/iwvelando/finance-forecast/pkg/constants"
)

// Round rounds a value to two decimals, i.e. to represent real currency.
// Used for making logical comparisons.
func Round(val float64) float64 {
	return math.Round(val*constants.DecimalPrecision) / constants.DecimalPrecision
}

// IsZero checks if a value is effectively zero (within tolerance)
func IsZero(val float64) bool {
	return math.Abs(val) <= constants.CurrencyTolerance
}

// IsPositive checks if a value is positive (greater than tolerance)
func IsPositive(val float64) bool {
	return val > constants.CurrencyTolerance
}

// IsNegative checks if a value is negative (less than negative tolerance)
func IsNegative(val float64) bool {
	return val < -constants.CurrencyTolerance
}

// WithinTolerance checks if two values are within a specified tolerance
func WithinTolerance(val1, val2, tolerance float64) bool {
	return math.Abs(val1-val2) <= tolerance
}

// Min returns the minimum of two float64 values
func Min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// Max returns the maximum of two float64 values
func Max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// CalculatePercentage calculates what percentage value is of total
func CalculatePercentage(value, total float64) float64 {
	if total == 0 {
		return 0
	}
	return (value / total) * 100
}

// ApplyPercentage applies a percentage to a value
func ApplyPercentage(value, percentage float64) float64 {
	return value * (percentage / constants.PercentageMultiplier)
}
