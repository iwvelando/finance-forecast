# Test Directory

This directory contains test artifacts for the finance-forecast project.

## Directory Structure

```
test/
├── README.md              # This file
├── test_config.yaml       # Dedicated test configuration
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

The `test_config.yaml` file is used by all automated tests. Do not modify this file unless you understand the impact on test baselines.

### Configuration Files
- **Tests**: Use `test/test_config.yaml`
- **Documentation/Examples**: Use `config.yaml.example`
- **Application Runtime**: User provides their own config file

## Test Outputs

- **Logs**: Generated in `logs/` directory when running tests with verbose output
- **Baseline Files**: Reference files in `baseline/` directory for comparison testing
