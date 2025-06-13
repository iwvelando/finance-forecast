// Package output provides utilities for formatting and displaying forecast results.
package output

import (
	"fmt"
	"sort"
	"strings"

	"github.com/iwvelando/finance-forecast/internal/forecast"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

// PrettyFormat outputs a human-readable rather than machine-readable table.
func PrettyFormat(results []forecast.Forecast) {
	p := message.NewPrinter(language.English)
	for _, result := range results {
		fmt.Printf("--- Results for scenario %s ---\n", result.Name)
		fmt.Printf("Date    | Amount        | Notes\n")
		fmt.Printf("____    | _____________ | _____\n")
		dates := make([]string, len(result.Data))
		n := 0
		for date := range result.Data {
			dates[n] = date
			n++
		}
		sort.Strings(dates)
		for _, date := range dates {
			_, _ = p.Printf("%s | $%.2f | %s\n", date, result.Data[date], strings.Join(result.Notes[date], ","))
		}
		if len(results) > 1 {
			fmt.Printf("\n")
		}
	}
}

// CsvFormat outputs in comma-separated value format.
func CsvFormat(results []forecast.Forecast) {
	// All results have the same timeline, so grab the dates from the first
	dates := make([]string, len(results[0].Data))
	n := 0
	for date := range results[0].Data {
		dates[n] = date
		n++
	}
	sort.Strings(dates)
	fmt.Printf(`"date"`)
	for _, result := range results {
		fmt.Printf(`,"amount (%s)","notes (%s)"`, result.Name, result.Name)
	}
	fmt.Printf("\n")
	for _, date := range dates {
		fmt.Printf(`"%s"`, date)
		for _, result := range results {
			fmt.Printf(`,"%.2f"`, result.Data[date])
			fmt.Printf(`,"%s"`, strings.Join(result.Notes[date], ","))
		}
		fmt.Printf("\n")
	}
}
