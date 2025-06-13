# Finance Forecast

## Purpose

Finance Forecast is meant to be a simple tool to read in a config file that outlines various financial events (i.e. anything related to income or spending, including loans) and associated dates and then run simulations on one or more scenarios to see how these turn out. The more specific the events are the more accurate the simulation should be. Output defaults to printing to the console on STDOUT in a pretty format, but CSV format for easy import to other software may also be specified.

The general goal is to help provide users with a reasonable best-guess guide for financial decisions. By expressing different choices in different scenarios and then running the simulation the user can see how things will generally play out over the long-term.

## Assumptions

### Simulation

* The simulation starts processing events the next calendar month after when this is run; in other words the starting value should be taking into account the current month.

### Loan Handling

Loans are currently assumed to be compounded monthly. It is a future work item to allow specifying the compounding period.

The simulation makes some assumptions regarding escrow handling for simplicity's sake; if your needs require more precise handling then you could handle escrow using events and not define it (or set it to 0) in the loan.

* If escrow is defined the simulation assumes that the accumulated escrow for that year will be refunded if:
  1. The loan is paid off early (but not on December)
  1. The loan matures (but not on December)
* If escrow is defined and the asset is _not_ sold off then the escrow will be extrapolated to an annual expense paid in December

## Usage

Run `finance-forecast --help` to print command-line flags which include specifying the YAML-formatted config file's location and the output format.

Sample usage:

```
finance-forecast -config=./config.yaml.example -output-format=csv
```

Output:

```
"date","amount (current path)","notes (current path)","amount (new home purchase)","notes (new home purchase)","amount (new home purchase with extra principal payments)","notes (new home purchase with extra principal payments)"
"2020-07","30000.00","","30000.00","","30000.00",""
"2020-08","29670.24","","29670.24","","29670.24",""
"2020-09","29340.47","","29340.47","","29340.47",""
...
"2030-10","22016.12","","122615.85","","83515.85",""
"2030-11","22117.31","","123673.33","","11167.85","paying off asset 5678 Street Address for 80658.67"
"2030-12","22218.51","","124730.82","","6278.52",""
"2031-01","22309.71","","125778.30","","8579.19",""
"2031-02","22410.91","","126835.79","","10889.86",""
"2031-03","22512.11","","127893.27","","13200.53",""
"2031-04","22603.31","","128940.75","","15501.20",""
"2031-05","22704.50","","8349.93","paying off asset 5678 Street Address for 125301.49","17811.87",""
"2031-06","22805.70","","10660.60","","20122.54",""
...
"2089-11","286955.01","","568270.01","","577731.95",""
"2089-12","281920.01","","562035.01","","571496.95",""
"2090-01","282875.01","","562990.01","","572451.95",""
```

## Future Work

* Implement tests
* Ability to declare categories for events and tabulate average spending and income based on category
* Consider methods for simple optimization by configuring supported values as within a range and optimize a cost function
* Handle inflation
* Allow specifying compounding period for loans
* Handle investment scenarios with reallocating asset distribution over time
* Produce charts of the different scenarios without having to import the CSV output in spreadsheet software
* A GUI might be interesting?

## Logging Configuration

Finance Forecast supports flexible logging configuration through the config file and command-line overrides.

### Configuration File

Add a `logging` section to your config file:

```yaml
logging:
  level: info        # Options: debug, info, warn, error (default: info)
  format: console    # Options: console (human-readable) or json (structured) (default: json)
  output_file: forecast.log  # Optional: log to file instead of stdout
```

### Command Line Override

You can override the log level at runtime:

```bash
# Override to debug level
./finance-forecast --log-level debug --config config.yaml

# Override to error level for quiet operation
./finance-forecast --log-level error --config config.yaml
```

### Log Levels

- **debug**: Detailed diagnostic information
- **info**: General operational messages (default)
- **warn**: Warning messages for potential issues
- **error**: Error messages for serious problems

### Log Formats

- **console**: Human-readable format for development
- **json**: Structured JSON format for production/monitoring

### Examples

Debug mode with console output:
```yaml
logging:
  level: debug
  format: console
```

Production mode with JSON logging to file:
```yaml
logging:
  level: info
  format: json
  output_file: /var/log/finance-forecast.log
```

### Testing Logging

Use the logging demo utility to test your configuration:

```bash
go run cmd/logging-demo/main.go --config config.yaml
go run cmd/logging-demo/main.go --config config.yaml --log-level debug
```

## Configuration

