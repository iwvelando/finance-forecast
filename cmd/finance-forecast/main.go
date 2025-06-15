package main

import (
	"flag"
	"fmt"

	"github.com/iwvelando/finance-forecast/internal/config"
	"github.com/iwvelando/finance-forecast/internal/forecast"
	"github.com/iwvelando/finance-forecast/pkg/constants"
	"github.com/iwvelando/finance-forecast/pkg/output"
	"github.com/iwvelando/finance-forecast/pkg/validation"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// initializeLogger creates a zap logger based on configuration and CLI override
func initializeLogger(loggingConfig config.LoggingConfig, logLevelOverride string) (*zap.Logger, error) {
	// Determine log level (CLI override takes precedence)
	level := loggingConfig.Level
	if logLevelOverride != "" {
		level = logLevelOverride
	}
	if level == "" {
		level = "info" // Default to info level
	}

	// Parse log level
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zapcore.DebugLevel
	case "info":
		zapLevel = zapcore.InfoLevel
	case "warn", "warning":
		zapLevel = zapcore.WarnLevel
	case "error":
		zapLevel = zapcore.ErrorLevel
	default:
		return nil, fmt.Errorf("invalid log level: %s", level)
	}

	// Determine output format
	format := loggingConfig.Format
	if format == "" {
		format = "json" // Default to JSON for production
	}

	// Configure encoder
	var config zap.Config
	switch format {
	case "console":
		config = zap.NewDevelopmentConfig()
		config.Level = zap.NewAtomicLevelAt(zapLevel)
	case "json":
		config = zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zapLevel)
	default:
		return nil, fmt.Errorf("invalid log format: %s", format)
	}

	// Configure output file if specified
	if loggingConfig.OutputFile != "" {
		config.OutputPaths = []string{loggingConfig.OutputFile}
		config.ErrorOutputPaths = []string{loggingConfig.OutputFile}
	}

	return config.Build()
}

func main() {
	// Process command line flags first to get config location
	configLocation := flag.String("config", constants.DefaultConfigFile, "path to configuration file")
	outputFormat := flag.String("output-format", constants.OutputFormatPretty, "type of output: pretty, csv")
	logLevel := flag.String("log-level", "", "log level override (debug, info, warn, error)")
	flag.Parse()

	// Load the config file to get logging configuration
	conf, err := config.LoadConfiguration(*configLocation)
	if err != nil {
		fmt.Printf("{\"op\": \"main\", \"level\": \"fatal\", \"msg\": \"failed to load configuration at %s\", \"error\": \"%v\"}\n", *configLocation, err)
		return
	}

	// Initialize logging based on config and CLI override
	logger, err := initializeLogger(conf.Logging, *logLevel)
	if err != nil {
		fmt.Printf("{\"op\": \"main\", \"level\": \"fatal\", \"msg\": \"failed to initialize logger\", \"error\": \"%v\"}\n", err)
		return
	}
	defer func() {
		_ = logger.Sync()
	}()

	err = validation.ValidateOutputFormat(*outputFormat)
	if err != nil {
		logger.Fatal(err.Error(),
			zap.String("op", "main"),
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
	case constants.OutputFormatPretty:
		output.PrettyFormat(results)
	case constants.OutputFormatCSV:
		output.CsvFormat(results)
	}

}
