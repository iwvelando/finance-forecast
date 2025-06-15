# Finance Forecast

A tool to simulate and forecast financial scenarios based on defined events and loans.

## Purpose

Finance Forecast helps you evaluate different financial scenarios by simulating events (income, spending, loans) over time.

## Usage

```
finance-forecast --config=./config.yaml --output-format=csv
```

### Options
- `--config`: Path to YAML config file (required)
- `--output-format`: `pretty` (default) or `csv`
- `--log-level`: Override logging level

## Key Concepts

### Simulation
- Processing starts the month after execution
- Initial value should account for current month

### Loans
- Compounded monthly
- Escrow handling:
  - Refunded when loan is paid early (except December)
  - Extrapolated to annual expense if asset not sold following maturity

## Logging Configuration

Configure in YAML:
```yaml
logging:
  level: info        # debug, info, warn, error
  format: console    # console or json
  output_file: path  # optional log file
```

Run with override:
```bash
./finance-forecast --log-level debug --config config.yaml
```
