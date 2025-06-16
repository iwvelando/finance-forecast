# Finance Forecast

## Purpose

Finance Forecast is a simple tool to read in a config file that outlines various financial events (i.e. anything related to income or spending, loans, etc) and associated dates and then run simulations on one or more scenarios to see how these turn out. The more specific the events are the more accurate the simulation should be. Output defaults to printing to the console in a pretty format, but CSV format for easy import to other software may also be specified.

The general goal is to help provide users with a reasonable best-guess guide for financial decisions. By expressing different choices in different scenarios and then running the simulation the user can see how things generally would out over the long-term.

## Usage

```
finance-forecast --config=config.yaml.example --output-format=csv
```

### Options
- `--config`: Path to YAML config file (required)
- `--output-format`: Override output format: `pretty` (default) or `csv`
- `--log-level`: Override logging level

## Key Concepts

### Simulation
- Processing starts from the configured `startDate` (YYYY-MM format) or current month if not specified
- Initial value should account for the month preceding the start date

### Loans
- Compounded monthly
- Escrow handling:
  - Refunded when loan is paid early (except December)
  - Extrapolated to annual expense if asset not sold following maturity

## Logging and Output Configuration

Configure in YAML:
```yaml
logging:
  level: info        # debug, info, warn, error
  format: console    # console or json
  outputFile: path   # optional log file

output:
  format: pretty     # pretty or csv
```

Run with overrides:
```bash
./finance-forecast --log-level debug --output-format csv --config config.yaml
```
