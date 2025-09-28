# Finance Forecast

## Purpose

Finance Forecast is a simple tool to read in a config file that outlines various financial events (i.e. anything related to income or spending, loans, etc) and associated dates and then run simulations on one or more scenarios to see how these turn out. The more specific the events are the more accurate the simulation should be. Output defaults to printing to the console in a pretty format, but CSV format for easy import to other software may also be specified.

The general goal is to help provide users with a reasonable best-guess guide for financial decisions. By expressing different choices in different scenarios and then running the simulation the user can see how things generally would out over the long-term.

## Usage

```
finance-forecast --config=config.yaml.example --output-format=csv
```

### Web UI Mode

Run the embedded web server to upload configurations through a browser:

```bash
finance-forecast --serve --addr :8080
```

When running in server mode:

- Visit `http://localhost:8080` (or your chosen address) to open the UI
- Upload a YAML configuration to run the simulation
- Review the rendered results table and download the generated CSV without touching the CLI
- Provide `--config` if you want to reuse logging settings from a file
- Adjust runtime settings (address, upload limits, logging) using `server-config.yaml` (copy from `server-config.yaml.example`). Upload limits accept human-friendly units like `256K`, `10M`, or `1G`. Configure structured logging with `logging.level`, `logging.format`, and `logging.outputFile`. CLI flags such as `--addr`, `--max-upload`, and `--server-config` override or choose the configuration file when needed, while `--log-level` still wins over file settings.

### Options
- `--config`: Path to YAML config file (required for CLI; optional for server logging defaults)
- `--output-format`: Override output format: `pretty` (default) or `csv`
- `--log-level`: Override logging level (takes precedence over config and server-config settings)
- `--serve`: Start the web UI server instead of running the CLI simulation
- `--addr`: Bind address for the web UI server (overrides server config)
- `--server-config`: Path to the server configuration file (default `server-config.yaml`)
- `--max-upload`: Maximum upload size in bytes for YAML configs (overrides server config)

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
