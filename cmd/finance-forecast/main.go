package main

import (
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/iwvelando/finance-forecast/internal/config"
	"github.com/iwvelando/finance-forecast/internal/forecast"
	"github.com/iwvelando/finance-forecast/internal/server"
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
		// Ensure the directory exists
		if dir := filepath.Dir(loggingConfig.OutputFile); dir != "." {
			if err := os.MkdirAll(dir, 0755); err != nil {
				return nil, fmt.Errorf("failed to create log directory %s: %v", dir, err)
			}
		}

		// Test if we can create/write to the file
		if file, err := os.OpenFile(loggingConfig.OutputFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644); err != nil {
			return nil, fmt.Errorf("failed to open log file %s: %v", loggingConfig.OutputFile, err)
		} else {
			_ = file.Close()
		}

		config.OutputPaths = []string{loggingConfig.OutputFile}
		config.ErrorOutputPaths = []string{loggingConfig.OutputFile}
	}

	return config.Build()
}

func main() {
	// Process command line flags first to get config location
	configLocation := flag.String("config", constants.DefaultConfigFile, "path to configuration file")
	outputFormatFlag := flag.String("output-format", "", "type of output override: pretty, csv")
	logLevel := flag.String("log-level", "", "log level override (debug, info, warn, error)")
	serve := flag.Bool("serve", false, "start the web UI server")
	addr := flag.String("addr", "", "bind address for the web server (overrides server config)")
	maxUpload := flag.String("max-upload", "", "maximum upload size (e.g. 256K, 10M) overriding server config")
	serverConfigPath := flag.String("server-config", constants.DefaultServerConfigFile, "path to server configuration file")
	emergencyMonthsFlag := flag.String("emergency-months", "", "override emergency fund recommendation duration in months (e.g. 6). Set to 0 to disable recommendations.")
	flag.Parse()

	var emergencyMonthsOverride *float64
	if *emergencyMonthsFlag != "" {
		months, err := strconv.ParseFloat(*emergencyMonthsFlag, 64)
		if err != nil {
			fmt.Printf("{\"op\": \"main\", \"level\": \"fatal\", \"msg\": \"invalid value for --emergency-months\", \"value\": \"%s\", \"error\": \"%v\"}\n", *emergencyMonthsFlag, err)
			return
		}
		emergencyMonthsOverride = &months
	}

	if *serve {
		runServer(*addr, *maxUpload, *serverConfigPath, *configLocation, *logLevel)
		return
	}

	// Load the config file to get logging configuration
	conf, err := config.LoadConfiguration(*configLocation)
	if err != nil {
		fmt.Printf("{\"op\": \"main\", \"level\": \"fatal\", \"msg\": \"failed to load configuration at %s\", \"error\": \"%v\"}\n", *configLocation, err)
		return
	}
	if emergencyMonthsOverride != nil {
		conf.Recommendations.EmergencyFundMonths = *emergencyMonthsOverride
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

	// Determine output format (CLI override takes precedence over config)
	outputFormat := conf.Output.Format
	if *outputFormatFlag != "" {
		outputFormat = *outputFormatFlag
	}
	if outputFormat == "" {
		outputFormat = constants.OutputFormatPretty // Default to pretty format
	}

	err = validation.ValidateOutputFormat(outputFormat)
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
	switch outputFormat {
	case constants.OutputFormatPretty:
		output.PrettyFormat(results)
	case constants.OutputFormatCSV:
		output.CsvFormat(results)
	}

}

func runServer(addr string, maxUpload string, serverConfigPath string, configPath string, logLevel string) {
	var loggingConf config.LoggingConfig
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			cfg, err := config.LoadConfiguration(configPath)
			if err != nil {
				fmt.Printf("{\"op\": \"serve\", \"level\": \"fatal\", \"msg\": \"failed to load configuration at %s\", \"error\": \"%v\"}\n", configPath, err)
				return
			}
			loggingConf = cfg.Logging
		} else if !errors.Is(err, os.ErrNotExist) {
			fmt.Printf("{\"op\": \"serve\", \"level\": \"fatal\", \"msg\": \"unable to access configuration at %s\", \"error\": \"%v\"}\n", configPath, err)
			return
		}
	}

	srvCfg, err := server.LoadConfig(serverConfigPath)
	if err != nil {
		fmt.Printf("{\"op\": \"serve\", \"level\": \"fatal\", \"msg\": \"failed to load server configuration at %s\", \"error\": \"%v\"}\n", serverConfigPath, err)
		return
	}

	if addr != "" {
		srvCfg.Address = addr
	}
	if maxUpload != "" {
		size, err := server.ParseSize(maxUpload)
		if err != nil {
			fmt.Printf("{\"op\": \"serve\", \"level\": \"fatal\", \"msg\": \"invalid max-upload value\", \"value\": \"%s\", \"error\": \"%v\"}\n", maxUpload, err)
			return
		}
		srvCfg.SetUploadSizeBytes(size)
	}

	if srvCfg.Logging.Level != "" {
		loggingConf.Level = srvCfg.Logging.Level
	}
	if srvCfg.Logging.Format != "" {
		loggingConf.Format = srvCfg.Logging.Format
	}
	if srvCfg.Logging.OutputFile != "" {
		loggingConf.OutputFile = srvCfg.Logging.OutputFile
	}

	logger, err := initializeLogger(loggingConf, logLevel)
	if err != nil {
		fmt.Printf("{\"op\": \"serve\", \"level\": \"fatal\", \"msg\": \"failed to initialize logger\", \"error\": \"%v\"}\n", err)
		return
	}
	defer func() {
		_ = logger.Sync()
	}()

	logger.Info("starting finance-forecast web server",
		zap.String("addr", srvCfg.Address),
		zap.Int64("max_upload_bytes", srvCfg.UploadSizeBytes()),
	)

	handler := server.NewHandler(logger, srvCfg.UploadSizeBytes())
	if err := http.ListenAndServe(srvCfg.Address, handler); err != nil {
		logger.Fatal("server encountered an error",
			zap.String("op", "serve"),
			zap.Error(err),
		)
	}
}
