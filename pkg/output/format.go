// Package output provides utilities for formatting and displaying forecast results.
package output

import (
	"fmt"
	"sort"
	"strings"

	"github.com/iwvelando/finance-forecast/internal/forecast"
)

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

	// Format output
	for _, scenario := range results {
		fmt.Printf("\n=== %s ===\n", scenario.Name)
		for _, date := range dates {
			if balance, exists := scenario.Data[date]; exists {
				fmt.Printf("%s | $%.2f", date, balance)
				if notes, hasNotes := scenario.Notes[date]; hasNotes && len(notes) > 0 {
					fmt.Printf(" | %s", strings.Join(notes, ", "))
				}
				fmt.Println()
			}
		}
	}
}

// CsvFormat outputs in comma-separated value format.
func CsvFormat(results []forecast.Forecast) {
	if len(results) == 0 {
		fmt.Println("Date,Scenario,Amount,Notes")
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

	// Build header with scenario names
	header := []string{"\"date\""}
	for _, scenario := range results {
		header = append(header, fmt.Sprintf("\"amount (%s)\"", scenario.Name))
		header = append(header, fmt.Sprintf("\"notes (%s)\"", scenario.Name))
	}
	fmt.Println(strings.Join(header, ","))

	// Output data rows
	for _, date := range dates {
		row := []string{fmt.Sprintf("\"%s\"", date)}
		for _, scenario := range results {
			if balance, exists := scenario.Data[date]; exists {
				row = append(row, fmt.Sprintf("\"%.2f\"", balance))

				// Add notes
				if notes, hasNotes := scenario.Notes[date]; hasNotes && len(notes) > 0 {
					row = append(row, fmt.Sprintf("\"%s\"", strings.Join(notes, ",")))
				} else {
					row = append(row, "\"\"")
				}
			} else {
				row = append(row, "\"\"") // Empty amount
				row = append(row, "\"\"") // Empty notes
			}
		}
		fmt.Println(strings.Join(row, ","))
	}
}
