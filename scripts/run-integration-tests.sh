#!/bin/bash

# Integration Test Runner for Feature Flag Library
# This script runs comprehensive integration tests including Redis, PostgreSQL, and load tests

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
REDIS_URL="${REDIS_TEST_URL:-redis://localhost:6379/1}"
POSTGRES_URL="${POSTGRES_TEST_URL:-postgres://testuser:testpass@localhost:5432/testdb?sslmode=disable}"
RUN_LOAD_TESTS="${RUN_LOAD_TESTS:-false}"
VERBOSE="${VERBOSE:-false}"

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if a service is available
check_service() {
    local service=$1
    local url=$2
    local timeout=5

    case $service in
        "redis")
            if command -v redis-cli >/dev/null 2>&1; then
                if timeout $timeout redis-cli -u "$url" ping >/dev/null 2>&1; then
                    return 0
                fi
            fi
            ;;
        "postgres")
            if command -v psql >/dev/null 2>&1; then
                if timeout $timeout psql "$url" -c "SELECT 1;" >/dev/null 2>&1; then
                    return 0
                fi
            fi
            ;;
    esac
    return 1
}

# Function to run tests with proper environment setup
run_test_suite() {
    local test_name=$1
    local test_pattern=$2
    local required_env=$3

    print_status "Running $test_name..."

    # Set up environment variables
    export REDIS_TEST_URL="$REDIS_URL"
    export POSTGRES_TEST_URL="$POSTGRES_URL"

    # Build test command
    local cmd="go test -tags=integration"
    if [ "$VERBOSE" = "true" ]; then
        cmd="$cmd -v"
    fi
    cmd="$cmd -run '$test_pattern' ./featureflag"

    # Run the test
    if eval $cmd; then
        print_success "$test_name completed successfully"
        return 0
    else
        print_error "$test_name failed"
        return 1
    fi
}

# Main execution
main() {
    print_status "Starting Feature Flag Integration Tests"
    print_status "========================================"

    # Check if we're in the right directory
    if [ ! -f "go.mod" ]; then
        print_error "Please run this script from the project root directory"
        exit 1
    fi

    # Check Go installation
    if ! command -v go >/dev/null 2>&1; then
        print_error "Go is not installed or not in PATH"
        exit 1
    fi

    print_status "Go version: $(go version)"

    # Test configuration
    print_status "Test Configuration:"
    echo "  Redis URL: $REDIS_URL"
    echo "  PostgreSQL URL: $POSTGRES_URL"
    echo "  Load Tests: $RUN_LOAD_TESTS"
    echo "  Verbose: $VERBOSE"
    echo ""

    # Check service availability
    print_status "Checking service availability..."

    REDIS_AVAILABLE=false
    POSTGRES_AVAILABLE=false

    if check_service "redis" "$REDIS_URL"; then
        print_success "Redis is available"
        REDIS_AVAILABLE=true
    else
        print_warning "Redis is not available - Redis tests will be skipped"
        print_warning "To run Redis tests, ensure Redis is running and set REDIS_TEST_URL"
    fi

    if check_service "postgres" "$POSTGRES_URL"; then
        print_success "PostgreSQL is available"
        POSTGRES_AVAILABLE=true
    else
        print_warning "PostgreSQL is not available - PostgreSQL tests will be skipped"
        print_warning "To run PostgreSQL tests, ensure PostgreSQL is running and set POSTGRES_TEST_URL"
    fi

    echo ""

    # Run test suites
    local failed_tests=0

    # 1. Configuration Integration Tests
    if ! run_test_suite "Configuration Integration Tests" "TestConfigurationIntegration" ""; then
        ((failed_tests++))
    fi

    # 2. PostgreSQL Integration Tests
    if [ "$POSTGRES_AVAILABLE" = "true" ]; then
        if ! run_test_suite "PostgreSQL Integration Tests" "TestPostgresIntegration" "POSTGRES_TEST_URL"; then
            ((failed_tests++))
        fi
    else
        print_warning "Skipping PostgreSQL Integration Tests - service not available"
    fi

    # 3. Redis Integration Tests
    if [ "$REDIS_AVAILABLE" = "true" ]; then
        if ! run_test_suite "Redis Integration Tests" "TestRedisIntegration" "REDIS_TEST_URL"; then
            ((failed_tests++))
        fi
    else
        print_warning "Skipping Redis Integration Tests - service not available"
    fi

    # 4. End-to-End Integration Tests
    if ! run_test_suite "End-to-End Integration Tests" "TestEndToEndWorkflows" ""; then
        ((failed_tests++))
    fi

    # 5. Load Tests (optional)
    if [ "$RUN_LOAD_TESTS" = "true" ]; then
        print_status "Running Load Tests (this may take several minutes)..."
        if ! run_test_suite "Load Tests" "TestLoadScenarios|TestConcurrencyLimits|TestMemoryUsage|TestCachePerformance" ""; then
            ((failed_tests++))
        fi
    else
        print_warning "Skipping Load Tests - set RUN_LOAD_TESTS=true to enable"
    fi

    # Summary
    echo ""
    print_status "Integration Test Summary"
    print_status "========================"

    if [ $failed_tests -eq 0 ]; then
        print_success "All integration tests passed!"
        exit 0
    else
        print_error "$failed_tests test suite(s) failed"
        exit 1
    fi
}

# Help function
show_help() {
    echo "Feature Flag Integration Test Runner"
    echo ""
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -h, --help              Show this help message"
    echo "  -v, --verbose           Enable verbose test output"
    echo "  -l, --load-tests        Run load tests (may take several minutes)"
    echo "  --redis-url URL         Redis connection URL (default: redis://localhost:6379/1)"
    echo "  --postgres-url URL      PostgreSQL connection URL"
    echo ""
    echo "Environment Variables:"
    echo "  REDIS_TEST_URL          Redis connection URL for tests"
    echo "  POSTGRES_TEST_URL       PostgreSQL connection URL for tests"
    echo "  RUN_LOAD_TESTS          Set to 'true' to run load tests"
    echo "  VERBOSE                 Set to 'true' for verbose output"
    echo ""
    echo "Examples:"
    echo "  $0                                    # Run basic integration tests"
    echo "  $0 -v -l                             # Run all tests with verbose output"
    echo "  $0 --redis-url redis://localhost:6380/0  # Use custom Redis URL"
    echo ""
    echo "Prerequisites:"
    echo "  - Go 1.19 or later"
    echo "  - Redis server (for Redis tests)"
    echo "  - PostgreSQL server (for PostgreSQL tests)"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_help
            exit 0
            ;;
        -v|--verbose)
            VERBOSE=true
            shift
            ;;
        -l|--load-tests)
            RUN_LOAD_TESTS=true
            shift
            ;;
        --redis-url)
            REDIS_URL="$2"
            shift 2
            ;;
        --postgres-url)
            POSTGRES_URL="$2"
            shift 2
            ;;
        *)
            print_error "Unknown option: $1"
            show_help
            exit 1
            ;;
    esac
done

# Run main function
main