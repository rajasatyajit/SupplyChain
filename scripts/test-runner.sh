#!/bin/bash

# Test Runner Script for SupplyChain Microservice
# This script runs comprehensive tests and generates reports

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REPORTS_DIR="reports"
COVERAGE_DIR="coverage"
TEST_TIMEOUT="30s"
INTEGRATION_TEST_TIMEOUT="60s"

# Create directories
mkdir -p "$REPORTS_DIR" "$COVERAGE_DIR"

# Functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

run_unit_tests() {
    log_info "Running unit tests..."
    
    go test -v -timeout="$TEST_TIMEOUT" \
        -coverprofile="$COVERAGE_DIR/unit-coverage.out" \
        -covermode=atomic \
        -json \
        ./... > "$REPORTS_DIR/unit-tests.json" 2>&1
    
    # Generate coverage reports
    go tool cover -html="$COVERAGE_DIR/unit-coverage.out" -o "$COVERAGE_DIR/unit-coverage.html"
    go tool cover -func="$COVERAGE_DIR/unit-coverage.out" > "$REPORTS_DIR/unit-coverage.txt"
    
    # Extract coverage percentage
    COVERAGE=$(go tool cover -func="$COVERAGE_DIR/unit-coverage.out" | grep total | awk '{print $3}')
    log_success "Unit tests completed. Coverage: $COVERAGE"
}

run_integration_tests() {
    log_info "Running integration tests..."
    
    if [ -d "test/integration" ]; then
        go test -v -timeout="$INTEGRATION_TEST_TIMEOUT" \
            -tags=integration \
            -json \
            ./test/integration/... > "$REPORTS_DIR/integration-tests.json" 2>&1
        log_success "Integration tests completed"
    else
        log_warning "No integration tests found"
    fi
}

run_race_tests() {
    log_info "Running race condition tests..."
    
    go test -race -timeout="$TEST_TIMEOUT" \
        -json \
        ./... > "$REPORTS_DIR/race-tests.json" 2>&1
    
    log_success "Race condition tests completed"
}

run_benchmarks() {
    log_info "Running benchmarks..."
    
    go test -bench=. -benchmem -benchtime=5s \
        ./... > "$REPORTS_DIR/benchmarks.txt" 2>&1
    
    log_success "Benchmarks completed"
}

run_fuzz_tests() {
    log_info "Checking for fuzz tests..."
    
    # Look for fuzz tests (Go 1.18+)
    if grep -r "func Fuzz" . --include="*.go" > /dev/null 2>&1; then
        log_info "Running fuzz tests..."
        go test -fuzz=. -fuzztime=30s ./... > "$REPORTS_DIR/fuzz-tests.txt" 2>&1 || true
        log_success "Fuzz tests completed"
    else
        log_info "No fuzz tests found"
    fi
}

check_test_coverage() {
    log_info "Analyzing test coverage..."
    
    if [ -f "$COVERAGE_DIR/unit-coverage.out" ]; then
        # Extract coverage percentage
        COVERAGE=$(go tool cover -func="$COVERAGE_DIR/unit-coverage.out" | grep total | awk '{print $3}' | sed 's/%//')
        
        # Set coverage threshold
        THRESHOLD=80
        
        if (( $(echo "$COVERAGE >= $THRESHOLD" | bc -l) )); then
            log_success "Coverage $COVERAGE% meets threshold of $THRESHOLD%"
        else
            log_warning "Coverage $COVERAGE% below threshold of $THRESHOLD%"
            
            # Show uncovered functions
            log_info "Functions with low coverage:"
            go tool cover -func="$COVERAGE_DIR/unit-coverage.out" | awk '$3 < 80 {print $1 ": " $3}' | head -10
        fi
    fi
}

generate_test_report() {
    log_info "Generating test report..."
    
    cat > "$REPORTS_DIR/test-summary.md" << EOF
# Test Report

Generated: $(date)

## Summary

- **Unit Tests**: $(grep -c '"Action":"pass"' "$REPORTS_DIR/unit-tests.json" 2>/dev/null || echo "N/A") passed
- **Integration Tests**: $(grep -c '"Action":"pass"' "$REPORTS_DIR/integration-tests.json" 2>/dev/null || echo "N/A") passed
- **Race Tests**: $(grep -c '"Action":"pass"' "$REPORTS_DIR/race-tests.json" 2>/dev/null || echo "N/A") passed

## Coverage

$(cat "$REPORTS_DIR/unit-coverage.txt" 2>/dev/null || echo "Coverage report not available")

## Files

- Unit Test Results: \`$REPORTS_DIR/unit-tests.json\`
- Integration Test Results: \`$REPORTS_DIR/integration-tests.json\`
- Coverage Report: \`$COVERAGE_DIR/unit-coverage.html\`
- Benchmark Results: \`$REPORTS_DIR/benchmarks.txt\`

EOF

    log_success "Test report generated: $REPORTS_DIR/test-summary.md"
}

cleanup_old_reports() {
    log_info "Cleaning up old reports..."
    
    # Remove reports older than 7 days
    find "$REPORTS_DIR" -name "*.json" -mtime +7 -delete 2>/dev/null || true
    find "$COVERAGE_DIR" -name "*.out" -mtime +7 -delete 2>/dev/null || true
    
    log_success "Cleanup completed"
}

main() {
    log_info "Starting comprehensive test suite..."
    
    # Check if Go is available
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        exit 1
    fi
    
    # Check if we're in a Go module
    if [ ! -f "go.mod" ]; then
        log_error "Not in a Go module directory"
        exit 1
    fi
    
    # Parse command line arguments
    RUN_UNIT=true
    RUN_INTEGRATION=true
    RUN_RACE=true
    RUN_BENCHMARKS=true
    RUN_FUZZ=false
    
    while [[ $# -gt 0 ]]; do
        case $1 in
            --unit-only)
                RUN_INTEGRATION=false
                RUN_RACE=false
                RUN_BENCHMARKS=false
                shift
                ;;
            --no-benchmarks)
                RUN_BENCHMARKS=false
                shift
                ;;
            --with-fuzz)
                RUN_FUZZ=true
                shift
                ;;
            --help)
                echo "Usage: $0 [options]"
                echo "Options:"
                echo "  --unit-only     Run only unit tests"
                echo "  --no-benchmarks Skip benchmark tests"
                echo "  --with-fuzz     Include fuzz tests"
                echo "  --help          Show this help"
                exit 0
                ;;
            *)
                log_error "Unknown option: $1"
                exit 1
                ;;
        esac
    done
    
    # Cleanup old reports
    cleanup_old_reports
    
    # Run tests based on configuration
    if [ "$RUN_UNIT" = true ]; then
        run_unit_tests
    fi
    
    if [ "$RUN_INTEGRATION" = true ]; then
        run_integration_tests
    fi
    
    if [ "$RUN_RACE" = true ]; then
        run_race_tests
    fi
    
    if [ "$RUN_BENCHMARKS" = true ]; then
        run_benchmarks
    fi
    
    if [ "$RUN_FUZZ" = true ]; then
        run_fuzz_tests
    fi
    
    # Analyze results
    check_test_coverage
    generate_test_report
    
    log_success "All tests completed successfully!"
    log_info "Reports available in: $REPORTS_DIR/"
    log_info "Coverage reports available in: $COVERAGE_DIR/"
}

# Run main function
main "$@"