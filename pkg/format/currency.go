package format

import (
	"fmt"
	"math"
	"strings"
)

// Currency returns a currency string with a dollar sign and thousands separators (e.g., "-$1,234.56").
func Currency(amount float64) string {
	formatted := formatPositiveCurrency(math.Abs(amount))
	if amount < 0 {
		return "-$" + formatted
	}
	return "$" + formatted
}

// NumericCurrency returns a currency string without a currency symbol but with separators (e.g., "-1,234.56").
func NumericCurrency(amount float64) string {
	sign := ""
	if amount < 0 {
		sign = "-"
	}
	formatted := formatPositiveCurrency(math.Abs(amount))
	return sign + formatted
}

func formatPositiveCurrency(value float64) string {
	formatted := fmt.Sprintf("%.2f", value)
	parts := strings.SplitN(formatted, ".", 2)
	intPart := parts[0]
	decPart := "00"
	if len(parts) == 2 {
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

	return intPart + "." + decPart
}
