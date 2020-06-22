package main

import (
	"flag"
	"fmt"
	"github.com/iwvelando/finance-forecast/config"
	"github.com/iwvelando/finance-forecast/forecast"
	"go.uber.org/zap"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"sort"
)

func main() {

	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Println("{\"op\": \"main\", \"level\": \"fatal\", \"msg\": \"failed to initiate logger\"}")
		panic(err)
	}
	defer logger.Sync()

	configLocation := flag.String("config", "config.yaml", "path to configuration file")
	outputFormat := flag.String("output-format", "pretty", "type of output: pretty, csv")
	flag.Parse()

	if *outputFormat != "pretty" && *outputFormat != "csv" {
		logger.Fatal(fmt.Sprintf("expected output format of pretty or csv, got %s", *outputFormat),
			zap.String("op", "main"),
		)
	}

	// Load the config file based on path provided via CLI or the default
	conf, err := config.LoadConfiguration(*configLocation)
	if err != nil {
		logger.Fatal(fmt.Sprintf("failed to load configuration at %s", *configLocation),
			zap.String("op", "main"),
			zap.Error(err),
		)
	}

	// Process the event dates
	*conf, err = config.ParseDateLists(*conf)
	if err != nil {
		logger.Fatal("failed to parse date lists",
			zap.String("op", "main"),
			zap.Error(err),
		)
	}

	results, err := forecast.GetForecast(logger, *conf)
	if err != nil {
		logger.Fatal("failed to compute forecast",
			zap.String("op", "main"),
			zap.Error(err),
		)
	}

	if *outputFormat == "pretty" {
		PrettyFormat(results)
	} else if *outputFormat == "csv" {
		CsvFormat(results)
	}

}

func PrettyFormat(results []forecast.Forecast) {
	p := message.NewPrinter(language.English)
	for _, result := range results {
		fmt.Printf("--- Results for scenario %s ---\n", result.Name)
		fmt.Printf("Date    | Amount\n")
		fmt.Printf("____    | _____________\n")
		dates := make([]string, len(result.Data))
		n := 0
		for date := range result.Data {
			dates[n] = date
			n++
		}
		sort.Strings(dates)
		for _, date := range dates {
			p.Printf("%s | $%.2f\n", date, result.Data[date])
		}
		if len(results) > 1 {
			fmt.Printf("\n")
		}
	}
}

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
		fmt.Printf(`,"amount (%s)"`, result.Name)
	}
	fmt.Printf("\n")
	for _, date := range dates {
		fmt.Printf(`"%s"`, date)
		for _, result := range results {
			fmt.Printf(`,"%.2f"`, result.Data[date])
		}
		fmt.Printf("\n")
	}
}
