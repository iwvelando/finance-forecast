package main

import (
	"flag"
	"fmt"

	"github.com/iwvelando/finance-forecast/internal/config"
	"github.com/iwvelando/finance-forecast/internal/forecast"
	"github.com/iwvelando/finance-forecast/pkg/output"
	"github.com/iwvelando/finance-forecast/pkg/validation"
	"go.uber.org/zap"
)

func main() {

	// Initialize logging.
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Println("{\"op\": \"main\", \"level\": \"fatal\", \"msg\": \"failed to initiate logger\"}")
		panic(err)
	}
	defer func() {
		_ = logger.Sync()
	}()

	// Process command line flags.
	configLocation := flag.String("config", "config.yaml", "path to configuration file")
	outputFormat := flag.String("output-format", "pretty", "type of output: pretty, csv")
	flag.Parse()

	err = validation.ValidateOutputFormat(*outputFormat)
	if err != nil {
		logger.Fatal(err.Error(),
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

	// Validate configuration and display any warnings
	warnings := conf.ValidateConfiguration()
	for _, warning := range warnings {
		logger.Warn("Configuration warning: "+warning,
			zap.String("op", "main"),
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
	switch *outputFormat {
	case "pretty":
		output.PrettyFormat(results)
	case "csv":
		output.CsvFormat(results)
	}

}
