// Package output provides utilities for formatting and displaying forecast results.
package output

import (
	"fmt"
	"sort"
	"strings"

	"github.com/iwvelando/finance-forecast/internal/forecast"
)

// formatCurrency formats a float64 as currency with commas
func formatCurrency(amount float64) string {
	// Format with 2 decimal places
	formatted := fmt.Sprintf("%.2f", amount)

	// Split into integer and decimal parts
	parts := strings.Split(formatted, ".")
	intPart := parts[0]
	decPart := parts[1]

	// Add commas to integer part
	if len(intPart) > 3 {
		var result strings.Builder
		for i, digit := range intPart {
			if i > 0 && (len(intPart)-i)%3 == 0 {
				result.WriteString(",")
			}
			result.WriteRune(digit)
		}
		intPart = result.String()
	}

	return intPart + "." + decPart
}

// PrettyFormat formats the forecast results in a human-readable format
func PrettyFormat(results []forecast.Forecast) {
	if len(results) == 0 {
		fmt.Println("No forecast results to display.")
		return
	}

	// Create a map to collect all dates across scenarios
	allDates := make(map[string]bool)
	for _, scenario := range results {
		for date := range scenario.Data {
			allDates[date] = true
		}
	}

	// Convert to sorted slice
	var dates []string
	for date := range allDates {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	// Format output with liquid and total columns
	for _, scenario := range results {
		fmt.Printf("--- Results for scenario %s ---\n", scenario.Name)
		fmt.Printf("Date    | Liquid Net Worth | Total Net Worth | Notes\n")
		fmt.Printf("____    | ________________ | _______________ | _____\n")

		for _, date := range dates {
			liquidDisplay := "—"
			if liquid, ok := scenario.Liquid[date]; ok {
				liquidDisplay = "$" + formatCurrency(liquid)
			}

			totalDisplay := "—"
			if total, ok := scenario.Data[date]; ok {
				totalDisplay = "$" + formatCurrency(total)
			}

			fmt.Printf("%s | %s | %s | ", date, liquidDisplay, totalDisplay)
			if notes, hasNotes := scenario.Notes[date]; hasNotes && len(notes) > 0 {
				fmt.Printf("%s", strings.Join(notes, ", "))
			}
			fmt.Println()
		}
		fmt.Println() // Extra blank line between scenarios
	}
}

// CsvFormat outputs in comma-separated value format.
func CsvFormat(results []forecast.Forecast) {
	lines := buildCsvLines(results)
	for _, line := range lines {
		fmt.Println(line)
	}
}

// CsvString converts the forecast results into a CSV string using the same format as CsvFormat.
func CsvString(results []forecast.Forecast) string {
	lines := buildCsvLines(results)
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

// buildCsvLines creates the ordered CSV lines shared by CsvFormat and CsvString.
func buildCsvLines(results []forecast.Forecast) []string {
	if len(results) == 0 {
		return []string{"Date,Scenario,Liquid,Total,Notes"}
	}

	allDates := make(map[string]bool)
	for _, scenario := range results {
		for date := range scenario.Data {
			allDates[date] = true
		}
	}

	var dates []string
	for date := range allDates {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	header := []string{"\"date\""}
	for _, scenario := range results {
		header = append(header, fmt.Sprintf("\"liquid (%s)\"", scenario.Name))
		header = append(header, fmt.Sprintf("\"total (%s)\"", scenario.Name))
		header = append(header, fmt.Sprintf("\"notes (%s)\"", scenario.Name))
	}

	lines := []string{strings.Join(header, ",")}

	for _, date := range dates {
		row := []string{fmt.Sprintf("\"%s\"", date)}
		for _, scenario := range results {
			if liquid, lOK := scenario.Liquid[date]; lOK {
				row = append(row, fmt.Sprintf("\"%.2f\"", liquid))
			} else {
				row = append(row, "\"\"")
			}

			if total, tOK := scenario.Data[date]; tOK {
				row = append(row, fmt.Sprintf("\"%.2f\"", total))
			} else {
				row = append(row, "\"\"")
			}

			if notes, hasNotes := scenario.Notes[date]; hasNotes && len(notes) > 0 {
				row = append(row, fmt.Sprintf("\"%s\"", strings.Join(notes, ",")))
			} else {
				row = append(row, "\"\"")
			}
		}
		lines = append(lines, strings.Join(row, ","))
	}

	return lines
}
