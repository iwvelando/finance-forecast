# Test Directory Organization

This directory contains all test-related artifacts for the finance-forecast project, organized into logical subdirectories for better maintainability.

## Directory Structure

```
test/
├── README.md              # This file
├── test_config.yaml       # Dedicated test configuration (DO NOT modify)
├── baseline/              # Test baseline and reference files
│   └── baseline_output.csv
├── docs/                  # Test documentation and reports
│   ├── TEST_DOCUMENTATION.md
│   ├── TEST_FIXES.md
│   ├── TEST_FIX_SUMMARY.md
│   ├── TEST_STATUS.md
│   ├── TEST_SUMMARY.md
│   └── TESTING_COMPLETE.md
├── logs/                  # Test execution logs and output
│   ├── benchmark_output.log
│   ├── config_test_output.log
│   ├── forecast_test_output.log
│   └── integration_test_output.log
└── scripts/               # Test execution scripts
    ├── run_all_tests.sh
    └── run_tests.sh
```

## Usage

### Running Tests

#### Using Makefile (Recommended)
```bash
# Run all tests
make test

# Run specific test types
make test-unit
make test-integration
make test-performance
make test-coverage

# Run tests with verbose output (logs to test/logs/)
make test-verbose
```

#### Using Test Scripts
```bash
# Run comprehensive test suite
./test/scripts/run_tests.sh

# Run all tests with verbose logging
./test/scripts/run_all_tests.sh
```

#### Manual Test Execution
```bash
# Unit tests
go test -v ./config ./forecast

# Integration tests
go test -v -run "^TestMain|^TestCSV|^TestConfiguration|^TestEndToEnd" .

# Performance benchmarks
go test -bench=. -run=^$ ./...
```

### Test Logs

Test execution logs are automatically generated in the `logs/` directory:

- `config_test_output.log` - Config package test results
- `forecast_test_output.log` - Forecast package test results  
- `integration_test_output.log` - Integration test results
- `benchmark_output.log` - Performance benchmark results

### Coverage Reports

When running `make test-coverage`, coverage reports are generated:
- `logs/coverage.out` - Coverage data file
- `logs/coverage.html` - HTML coverage report (open in browser)

### Baseline Files

The `baseline/` directory contains reference files for comparison testing:
- `baseline_output.csv` - Known good output for integration tests

## Test Configuration

### `test_config.yaml`
This is the dedicated configuration file used by all automated tests. It contains the same structure as `config.yaml.example` but is optimized for testing with predictable values. 

**Important**: Do not modify this file unless you understand the impact on test baseline values and are prepared to update all affected test expectations.

### Configuration File Usage
- **Tests**: Use `test/test_config.yaml`
- **Documentation/Examples**: Use `config.yaml.example`
- **Manual Testing**: Use `config.yaml.example` or create your own
- **Application Runtime**: User provides their own config file

## Test Organization Benefits

1. **Cleaner Repository**: Test artifacts are no longer scattered in the root directory
2. **Better Organization**: Logical grouping of related files
3. **Easier Maintenance**: Clear separation of concerns
4. **Improved Navigation**: Developers can quickly find relevant test resources
5. **Automated Cleanup**: Makefile targets handle log cleanup automatically

## Makefile Integration

The project Makefile includes targets that work with this organization:

- `make clean-logs` - Clean all test logs
- `make test-scripts` - Run the test scripts in this directory  
- `make organize-tests` - Show the current test organization
- `make status` - Display project status including test structure

## Adding New Tests

When adding new test files or scripts:

1. **Test Scripts**: Add to `scripts/` directory and make executable
2. **Baseline Data**: Add reference files to `baseline/` directory
3. **Documentation**: Update relevant files in `docs/` directory
4. **Logs**: Test execution logs will be automatically placed in `logs/`

## Notes

- All test scripts are designed to work from the project root directory
- Log files are automatically timestamped and can be safely deleted
- The test structure supports both manual execution and CI/CD integration
- Coverage reports provide detailed analysis of test effectiveness
