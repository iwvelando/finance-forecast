package main

import (
	"flag"
	"fmt"
	"sort"
	"strings"

	"github.com/iwvelando/finance-forecast/internal/config"
	"github.com/iwvelando/finance-forecast/internal/forecast"
	"go.uber.org/zap"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

func main() {

	// Initialize logging.
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Println("{\"op\": \"main\", \"level\": \"fatal\", \"msg\": \"failed to initiate logger\"}")
		panic(err)
	}
	defer logger.Sync()

	// Process command line flags.
	configLocation := flag.String("config", "config.yaml", "path to configuration file")
	outputFormat := flag.String("output-format", "pretty", "type of output: pretty, csv")
	flag.Parse()

	if *outputFormat != "pretty" && *outputFormat != "csv" {
		logger.Fatal(fmt.Sprintf("expected output format of pretty or csv, got %s", *outputFormat),
			zap.String("op", "main"),
		)
	}

	// Load the config file based on path provided via CLI or the default.
	conf, err := config.LoadConfiguration(*configLocation)
	if err != nil {
		logger.Fatal(fmt.Sprintf("failed to load configuration at %s", *configLocation),
			zap.String("op", "main"),
			zap.Error(err),
		)
	}

	// Process the Event dates into time.Time.
	err = conf.ParseDateLists()
	if err != nil {
		logger.Fatal("failed to parse date lists",
			zap.String("op", "main"),
			zap.Error(err),
		)
	}

	// Process any stock-related events
	err = conf.ProcessStockEvents()
	if err != nil {
		logger.Fatal("failed to process stock events",
			zap.String("op", "main"),
			zap.Error(err),
		)
	}

	// Process the amortization schedules for all loans.
	err = conf.ProcessLoans(logger)
	if err != nil {
		logger.Fatal("failed to process loan amortization schedules",
			zap.String("op", "main"),
			zap.Error(err),
		)
	}

	// Run the simulation to get the Forecast.
	results, err := forecast.GetForecast(logger, *conf)
	if err != nil {
		logger.Fatal("failed to compute forecast",
			zap.String("op", "main"),
			zap.Error(err),
		)
	}

	// Handle output.
	if *outputFormat == "pretty" {
		PrettyFormat(results)
	} else if *outputFormat == "csv" {
		CsvFormat(results)
	}

}

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
			p.Printf("%s | $%.2f | %s\n", date, result.Data[date], strings.Join(result.Notes[date], ","))
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
