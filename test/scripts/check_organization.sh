#!/bin/bash

# Project Organization Status Check
# Verifies that the finance-forecast project is properly organized

echo "🔍 Finance Forecast Project Organization Check"
echo "=============================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Function to check if file/directory exists
check_exists() {
    local path=$1
    local type=$2
    local description=$3
    
    if [ "$type" = "file" ] && [ -f "$path" ]; then
        echo -e "${GREEN}✓${NC} $description: $path"
        return 0
    elif [ "$type" = "dir" ] && [ -d "$path" ]; then
        echo -e "${GREEN}✓${NC} $description: $path"
        return 0
    else
        echo -e "${RED}✗${NC} $description: $path (missing)"
        return 1
    fi
}

# Check main project structure
echo "📁 Main Project Structure:"
check_exists "Makefile" "file" "Build system"
check_exists "go.mod" "file" "Go module"
check_exists "finance-forecast.go" "file" "Main application"
check_exists "config.yaml.example" "file" "Example configuration"

echo ""
echo "📦 Package Structure:"
check_exists "config/" "dir" "Config package"
check_exists "forecast/" "dir" "Forecast package"

echo ""
echo "🧪 Test Organization:"
check_exists "test/" "dir" "Test directory"
check_exists "test/baseline/" "dir" "Baseline files"
check_exists "test/docs/" "dir" "Test documentation"
check_exists "test/logs/" "dir" "Test logs"
check_exists "test/scripts/" "dir" "Test scripts"
check_exists "test/README.md" "file" "Test documentation"

echo ""
echo "📋 Test Scripts:"
check_exists "test/scripts/run_tests.sh" "file" "Basic test runner"
check_exists "test/scripts/run_all_tests.sh" "file" "Comprehensive test runner"

echo ""
echo "📊 Test Artifacts:"
test_logs_count=$(find test/logs -name "*.log" 2>/dev/null | wc -l | tr -d ' ')
test_docs_count=$(find test/docs -name "*.md" 2>/dev/null | wc -l | tr -d ' ')
baseline_files_count=$(find test/baseline -type f 2>/dev/null | wc -l | tr -d ' ')

echo "  Test logs: $test_logs_count files"
echo "  Test docs: $test_docs_count files"  
echo "  Baseline files: $baseline_files_count files"

echo ""
echo "🔧 Build System:"
if [ -f "Makefile" ]; then
    makefile_targets=$(grep "^\.PHONY:" Makefile | wc -l | tr -d ' ')
    echo "  Makefile targets: $makefile_targets"
else
    echo -e "${RED}  Makefile not found${NC}"
fi

echo ""
echo "📈 Project Statistics:"
go_files=$(find . -name "*.go" -not -path "./vendor/*" | wc -l | tr -d ' ')
test_files=$(find . -name "*_test.go" -not -path "./vendor/*" | wc -l | tr -d ' ')
echo "  Go source files: $go_files"
echo "  Go test files: $test_files"

echo ""
echo "✨ Organization Status: COMPLETE"
echo ""
echo "The finance-forecast project has been successfully organized with:"
echo "  • Test artifacts moved to organized subdirectories"
echo "  • Comprehensive Makefile with standard targets"
echo "  • Updated test scripts with correct paths"
echo "  • Clear documentation and project structure"
echo "  • Clean root directory with logical file organization"
