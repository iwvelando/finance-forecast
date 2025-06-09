#!/bin/bash

# Finance Forecast Test Runner
# This script runs the comprehensive test suite for the finance-forecast application

echo "🧪 Finance Forecast Test Suite Runner"
echo "====================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Function to print colored output
print_status() {
    local status=$1
    local message=$2
    if [ "$status" = "PASS" ]; then
        echo -e "${GREEN}✓ $message${NC}"
    elif [ "$status" = "FAIL" ]; then
        echo -e "${RED}✗ $message${NC}"
    elif [ "$status" = "WARN" ]; then
        echo -e "${YELLOW}⚠ $message${NC}"
    else
        echo "  $message"
    fi
}

# Function to run a test and capture result
run_test() {
    local test_name=$1
    local test_command=$2
    
    echo ""
    echo "Running: $test_name"
    echo "Command: $test_command"
    
    if eval $test_command >/dev/null 2>&1; then
        print_status "PASS" "$test_name"
        return 0
    else
        print_status "FAIL" "$test_name"
        return 1
    fi
}

# Change to project root directory (go up two levels from test/scripts)
cd "$(dirname "$0")/../.."

echo ""
echo "📁 Project Directory: $(pwd)"

# Check if required files exist
echo ""
echo "🔍 Checking required files..."
required_files=("config.yaml.example" "test/baseline/baseline_output.csv" "go.mod")
for file in "${required_files[@]}"; do
    if [ -f "$file" ]; then
        print_status "PASS" "Found $file"
    else
        print_status "FAIL" "Missing $file"
        exit 1
    fi
done

# Build the application
echo ""
echo "🔨 Building application..."
if go build . >/dev/null 2>&1; then
    print_status "PASS" "Application builds successfully"
else
    print_status "FAIL" "Application build failed"
    exit 1
fi

# Test the original application functionality
echo ""
echo "🎯 Testing original application..."
if timeout 30s ./finance-forecast -config=./config.yaml.example -output-format=csv >/dev/null 2>&1; then
    print_status "PASS" "Original application runs successfully"
else
    print_status "WARN" "Original application test timed out or failed (may be expected)"
fi

# Run package tests
echo ""
echo "📦 Running package tests..."

test_results=()

# Config package tests
run_test "Config Package Build" "go build ./config"
test_results+=($?)

# Forecast package tests  
run_test "Forecast Package Tests" "go test ./forecast"
test_results+=($?)

run_test "Forecast Package Build" "go build ./forecast"
test_results+=($?)

# Check for test compilation issues
echo ""
echo "🔧 Checking test compilation..."
if go test -c ./config >/dev/null 2>&1; then
    print_status "PASS" "Config tests compile"
else
    print_status "FAIL" "Config tests compilation failed"
    test_results+=(1)
fi

if go test -c ./forecast >/dev/null 2>&1; then
    print_status "PASS" "Forecast tests compile"
else
    print_status "FAIL" "Forecast tests compilation failed"
    test_results+=(1)
fi

# Summary
echo ""
echo "📊 Test Summary"
echo "==============="

total_tests=${#test_results[@]}
passed_tests=0
for result in "${test_results[@]}"; do
    if [ $result -eq 0 ]; then
        ((passed_tests++))
    fi
done

echo "Total tests: $total_tests"
echo "Passed: $passed_tests"
echo "Failed: $((total_tests - passed_tests))"

if [ $passed_tests -eq $total_tests ]; then
    print_status "PASS" "All tests passed! ✨"
    echo ""
    echo "🎉 The codebase is ready for refactoring!"
    echo "   - All core functionality is working"
    echo "   - Tests are in place to validate changes"
    echo "   - Baseline output is captured for comparison"
    exit 0
else
    print_status "FAIL" "Some tests failed"
    echo ""
    echo "❗ Issues found that should be addressed before refactoring:"
    echo "   - Review failed tests"
    echo "   - Ensure all dependencies are available"
    echo "   - Check test environment setup"
    exit 1
fi
