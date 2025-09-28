// Package constants provides shared constants for the finance-forecast application.
package constants

// DateTimeLayout is the format expected in config files and is also the output
// date format.
const DateTimeLayout = "2006-01"

// Financial constants
const (
	// MonthsPerYear is the number of months in a year
	MonthsPerYear = 12

	// DecimalPrecision is the precision for currency rounding (2 decimal places)
	DecimalPrecision = 100

	// DefaultFrequency is the default frequency for monthly events
	DefaultFrequency = 1

	// QuarterlyFrequency is the frequency for quarterly events
	QuarterlyFrequency = 3

	// AnnualFrequency is the frequency for annual events
	AnnualFrequency = 12
)

// Output format constants
const (
	// OutputFormatPretty is the human-readable output format
	OutputFormatPretty = "pretty"

	// OutputFormatCSV is the CSV output format
	OutputFormatCSV = "csv"
)

// Configuration file constants
const (
	// DefaultConfigFile is the default configuration file name
	DefaultConfigFile = "config.yaml"

	// ExampleConfigFile is the example configuration file name
	ExampleConfigFile = "config.yaml.example"

	// DefaultServerConfigFile is the default server configuration file name
	DefaultServerConfigFile = "server-config.yaml"
)

// Server configuration defaults
const (
	// DefaultServerAddress is the default HTTP listen address for the web UI
	DefaultServerAddress = ":8080"

	// DefaultMaxUploadSizeBytes is the default maximum upload size for YAML configs (256 KB)
	DefaultMaxUploadSizeBytes int64 = 256 * 1024
)

// Validation constants
const (
	// ToleranceForComparison is the tolerance for financial comparisons
	ToleranceForComparison = 1.0

	// DefaultMortgageInsuranceCutoff is the default LTV cutoff for mortgage insurance
	DefaultMortgageInsuranceCutoff = 78.0

	// CurrencyTolerance is the tolerance for currency comparisons (1 cent)
	CurrencyTolerance = 0.01

	// PercentageMultiplier is used for percentage conversions
	PercentageMultiplier = 100.0
)
