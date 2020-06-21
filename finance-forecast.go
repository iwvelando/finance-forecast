package main

import (
	"flag"
	"fmt"
	"github.com/iwvelando/finance-forecast/config"
	"github.com/iwvelando/finance-forecast/forecast"
	"go.uber.org/zap"
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
	flag.Parse()

	// Load the config file based on path provided via CLI or the default
	conf, err := config.LoadConfiguration(*configLocation)
	if err != nil {
		logger.Fatal(fmt.Sprintf("failed to load configuration at %s", *configLocation),
			zap.String("op", "main"),
			zap.Error(err),
		)
	}

	//fmt.Println(conf)

	// Process the event dates
	*conf, err = config.ParseDateLists(*conf)
	if err != nil {
		logger.Fatal("failed to parse date lists",
			zap.String("op", "main"),
			zap.Error(err),
		)
	}

	//fmt.Println(conf)

	results, err := forecast.GetForecast(logger, *conf)
	if err != nil {
		logger.Fatal("failed to compute forecast",
			zap.String("op", "main"),
			zap.Error(err),
		)
	}

	PrettyFormat(results)

}

func PrettyFormat(results []forecast.Forecast) {
	for _, result := range results {
		fmt.Printf("--- Results for scenario %s ---\n", result.Name)
		fmt.Printf("Date    | Amount\n")
		fmt.Printf("____    | ______\n")
		dates := make([]string, len(result.Data))
		n := 0
		for date := range result.Data {
			dates[n] = date
			n++
		}
		sort.Strings(dates)
		for _, date := range dates {
			fmt.Printf("%s | %.2f\n", date, result.Data[date])
		}
		if len(results) > 1 {
			fmt.Printf("\n")
		}
	}
}
