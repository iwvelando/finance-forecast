#!/bin/bash

# Change to project root directory (go up two levels from test/scripts)
cd "$(dirname "$0")/../.."

echo "🧪 Running Finance Forecast Test Suite"
echo "======================================="

# Run tests with verbose output and capture results
echo "📦 Testing config package..."
if go test -v ../../config 2>&1 | tee ../logs/config_test_output.log; then
    echo "✅ Config package tests completed"
else
    echo "❌ Config package tests failed"
fi

echo ""
echo "📦 Testing forecast package..."
if go test -v ../../forecast 2>&1 | tee ../logs/forecast_test_output.log; then
    echo "✅ Forecast package tests completed"
else
    echo "❌ Forecast package tests failed"
fi

echo ""
echo "📦 Running integration tests..."
if go test -v -run "TestMainIntegrationBaseline|TestCSVOutputFormat|TestConfigurationValidation" ../.. 2>&1 | tee ../logs/integration_test_output.log; then
    echo "✅ Integration tests completed"
else
    echo "❌ Integration tests failed"
fi

echo ""
echo "📦 Running performance benchmarks..."
if go test -bench=. -run=^$ ../../... 2>&1 | tee ../logs/benchmark_output.log; then
    echo "✅ Performance benchmarks completed"
else
    echo "❌ Performance benchmarks failed"
fi

echo ""
echo "📊 Test Summary"
echo "==============="

# Count test results
TOTAL_TESTS=$(grep -h "RUN\|PASS\|FAIL" ../logs/*_test_output.log | grep -c "RUN")
PASSED_TESTS=$(grep -h "PASS" ../logs/*_test_output.log | grep -c "PASS")
FAILED_TESTS=$(grep -h "FAIL" ../logs/*_test_output.log | grep -c "FAIL")

echo "Total tests run: $TOTAL_TESTS"
echo "Passed: $PASSED_TESTS"
echo "Failed: $FAILED_TESTS"

if [ "$FAILED_TESTS" -eq 0 ]; then
    echo "🎉 All tests passed! Ready for refactoring."
    exit 0
else
    echo "⚠️  Some tests failed. Please fix before refactoring."
    exit 1
fi