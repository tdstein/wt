#!/usr/bin/env bash
#
# Minimal test runner for wt
#
# Usage: ./test-runner.sh [test-file...]
#        ./test-runner.sh              # Run all tests
#

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Test counters
TESTS_RUN=0
TESTS_PASSED=0
TESTS_FAILED=0

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

# Create a temporary test directory
TEST_TMPDIR=""
setup_test_env() {
    TEST_TMPDIR="$(mktemp -d)"
    export HOME="$TEST_TMPDIR"
    mkdir -p "$HOME/wt"

    # Setup minimal git config for commits
    git config --global user.email "test@example.com"
    git config --global user.name "Test User"
    git config --global init.defaultBranch main
}

cleanup_test_env() {
    if [[ -n "$TEST_TMPDIR" && -d "$TEST_TMPDIR" ]]; then
        rm -rf "$TEST_TMPDIR"
    fi
}

# Test assertion functions
assert_equals() {
    local expected="$1"
    local actual="$2"
    local msg="${3:-}"

    if [[ "$expected" == "$actual" ]]; then
        return 0
    else
        echo "  Expected: '$expected'"
        echo "  Actual:   '$actual'"
        [[ -n "$msg" ]] && echo "  Message:  $msg"
        return 1
    fi
}

assert_true() {
    local condition="$1"
    local msg="${2:-}"

    if eval "$condition"; then
        return 0
    else
        echo "  Condition failed: $condition"
        [[ -n "$msg" ]] && echo "  Message: $msg"
        return 1
    fi
}

assert_false() {
    local condition="$1"
    local msg="${2:-}"

    if ! eval "$condition"; then
        return 0
    else
        echo "  Condition should be false: $condition"
        [[ -n "$msg" ]] && echo "  Message: $msg"
        return 1
    fi
}

assert_dir_exists() {
    local path="$1"
    if [[ -d "$path" ]]; then
        return 0
    else
        echo "  Directory not found: $path"
        return 1
    fi
}

assert_file_exists() {
    local path="$1"
    if [[ -f "$path" ]]; then
        return 0
    else
        echo "  File not found: $path"
        return 1
    fi
}

assert_file_contains() {
    local path="$1"
    local pattern="$2"
    if grep -q "$pattern" "$path" 2>/dev/null; then
        return 0
    else
        echo "  Pattern '$pattern' not found in: $path"
        return 1
    fi
}

# Run a single test function
run_test() {
    local test_name="$1"
    local test_func="$2"

    ((TESTS_RUN++))

    # Setup fresh environment for each test
    setup_test_env

    echo -n "  $test_name ... "

    local output
    local result=0
    output=$($test_func 2>&1) || result=$?

    cleanup_test_env

    if [[ $result -eq 0 ]]; then
        echo -e "${GREEN}PASS${NC}"
        ((TESTS_PASSED++))
    else
        echo -e "${RED}FAIL${NC}"
        if [[ -n "$output" ]]; then
            echo "$output" | sed 's/^/    /'
        fi
        ((TESTS_FAILED++))
    fi
}

# Run all test functions in a file
run_test_file() {
    local test_file="$1"
    local file_name
    file_name="$(basename "$test_file")"

    echo
    echo -e "${YELLOW}Running: $file_name${NC}"

    # Source the test file to get test functions
    source "$test_file"

    # Find all test_ functions
    local test_funcs
    test_funcs=$(declare -F | awk '{print $3}' | grep '^test_' || true)

    if [[ -z "$test_funcs" ]]; then
        echo "  No tests found"
        return
    fi

    for func in $test_funcs; do
        run_test "$func" "$func" || true
        # Unset the function after running
        unset -f "$func" 2>/dev/null || true
    done
}

# Main
main() {
    local test_files=("$@")

    # If no files specified, find all test files
    if [[ ${#test_files[@]} -eq 0 ]]; then
        while IFS= read -r file; do
            test_files+=("$file")
        done < <(find "$SCRIPT_DIR" -name 'test-*.sh' -not -name 'test-runner.sh' | sort)
    fi

    echo "================================"
    echo "wt test suite"
    echo "================================"

    # Export everything tests need
    export PROJECT_ROOT
    export -f assert_equals assert_true assert_false
    export -f assert_dir_exists assert_file_exists assert_file_contains

    for test_file in "${test_files[@]}"; do
        run_test_file "$test_file"
    done

    echo
    echo "================================"
    echo -e "Results: ${GREEN}$TESTS_PASSED passed${NC}, ${RED}$TESTS_FAILED failed${NC}, $TESTS_RUN total"
    echo "================================"

    [[ $TESTS_FAILED -eq 0 ]]
}

main "$@"
