// Package output provides utilities for formatting and displaying forecast results.
package output

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/iwvelando/finance-forecast/internal/forecast"
	formatutil "github.com/iwvelando/finance-forecast/pkg/format"
	"github.com/iwvelando/finance-forecast/pkg/optimization"
)

func formatOptimizerValue(summary optimization.Summary, original bool) string {
	field := strings.ToLower(summary.Field)
	if original {
		if summary.OriginalDisplay != "" {
			return summary.OriginalDisplay
		}
	} else {
		if summary.ValueDisplay != "" {
			return summary.ValueDisplay
		}
	}

	var value float64
	if original {
		value = summary.Original
	} else {
		value = summary.Value
	}

	switch field {
	case "", "amount":
		return formatutil.Currency(value)
	case "frequency":
		return fmt.Sprintf("%d", int(math.Round(value)))
	case "startdate", "enddate":
		return formatMonthFromIndex(value)
	default:
		return fmt.Sprintf("%.2f", value)
	}
}

func formatMonthFromIndex(value float64) string {
	index := int(math.Round(value))
	if index < 0 {
		index = 0
	}
	year := index / 12
	month := index%12 + 1
	return fmt.Sprintf("%04d-%02d", year, month)
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
		printEmergencyFundSummary(scenario.Metrics.EmergencyFund)
		printOptimizationSummary(scenario.Metrics.Optimizations)
		fmt.Printf("Date    | Liquid Net Worth | Total Net Worth | Notes\n")
		fmt.Printf("____    | ________________ | _______________ | _____\n")

		for _, date := range dates {
			liquidDisplay := "—"
			if liquid, ok := scenario.Liquid[date]; ok {
				liquidDisplay = formatutil.Currency(liquid)
			}

			totalDisplay := "—"
			if total, ok := scenario.Data[date]; ok {
				totalDisplay = formatutil.Currency(total)
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

func printEmergencyFundSummary(ef *forecast.EmergencyFundRecommendation) {
	if ef == nil {
		return
	}
	formattedTarget := formatutil.Currency(ef.TargetAmount)
	formattedAverage := formatutil.Currency(ef.AverageMonthlyExpenses)
	line := fmt.Sprintf("Emergency fund target (%.1f months): %s", ef.TargetMonths, formattedTarget)
	line += fmt.Sprintf(" | Avg monthly expenses: %s", formattedAverage)
	if ef.FundedMonths > 0 {
		line += fmt.Sprintf(" | Starting coverage: %.1f months", ef.FundedMonths)
	}
	if ef.Shortfall > 0 {
		line += fmt.Sprintf(" | Shortfall: %s", formatutil.Currency(ef.Shortfall))
	} else if ef.Surplus > 0 {
		line += fmt.Sprintf(" | Surplus: %s", formatutil.Currency(ef.Surplus))
	}
	fmt.Println(line)
}

func printOptimizationSummary(summaries []optimization.Summary) {
	if len(summaries) == 0 {
		return
	}

	fmt.Println("Optimization adjustments:")
	for _, summary := range summaries {
		original := formatOptimizerValue(summary, true)
		value := formatOptimizerValue(summary, false)
		floor := formatutil.Currency(summary.Floor)
		minimum := formatutil.Currency(summary.MinimumCash)
		headroom := formatutil.Currency(summary.Headroom)
		status := "converged"
		if !summary.Converged {
			status = "not converged"
		}
		fmt.Printf(" - %s (%s): %s -> %s | floor %s | min cash %s | headroom %s | iterations %d (%s)\n",
			summary.TargetName,
			summary.Field,
			original,
			value,
			floor,
			minimum,
			headroom,
			summary.Iterations,
			status,
		)
		if len(summary.Notes) > 0 {
			fmt.Printf("   Notes: %s\n", strings.Join(summary.Notes, "; "))
		}
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
