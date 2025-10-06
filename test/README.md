# Test Directory

This directory contains test artifacts for the finance-forecast project.

## Directory Structure

```
test/
├── README.md              # This file
├── test_aftertax_taxed_config.yaml       # Deterministic taxable brokerage baseline
├── test_aftertax_taxfree_config.yaml     # Deterministic tax-free (Roth) baseline
├── test_cash_flows_config.yaml           # Deterministic cash-flow baseline
├── test_combined_config.yaml             # Union of deterministic baseline components
├── test_config.yaml       # Dedicated legacy regression configuration
├── test_pretax_investment_config.yaml    # Deterministic pre-tax investment baseline
├── test_single_loan_config.yaml          # Deterministic single-loan baseline
├── baseline/              # Test baseline and reference files
├── docs/                  # Test documentation and reports
├── logs/                  # Test execution logs and output
└── integration/           # Integration tests
```

## Running Tests

### Using Makefile
```bash
# Run all tests
make test

# Run specific test types
make test-unit
make test-integration
make test-performance
```

## Test Configuration

The deterministic baseline files (`test_*_config.yaml`) provide isolated, verifiable scenarios that the integration tests use to guard core financial calculations. The legacy `test_config.yaml` is still used for regression coverage in other tests. Do not modify these files unless you also update the documented projections.

### Configuration Files
- **Deterministic Baselines**: Use the `test/test_*_config.yaml` files introduced for integration tests
- **Legacy Regression Tests**: Use `test/test_config.yaml`
- **Documentation/Examples**: Use `config.yaml.example`
- **Application Runtime**: User provides their own config file

## Test Outputs

- **Logs**: Generated in `logs/` directory when running tests with verbose output
- **Baseline Files**: Reference files in `baseline/` directory for comparison testing
