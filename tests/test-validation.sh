#!/usr/bin/env bash
#
# Tests for prerequisite validation
#

test_git_available_check_passes() {
    # Should pass since git is available in test env
    bash -c "source '$PROJECT_ROOT/bin/wt' && check_git_available" >/dev/null 2>&1
}

test_git_config_check_passes() {
    # Should pass since test env sets git config
    bash -c "source '$PROJECT_ROOT/bin/wt' && check_git_config" >/dev/null 2>&1
}

test_git_config_check_fails_without_config() {
    # Create a clean environment without git config
    local temp_home
    temp_home="$(mktemp -d)"

    # Should fail when git config is missing
    ! HOME="$temp_home" bash -c "source '$PROJECT_ROOT/bin/wt' && check_git_config" 2>/dev/null

    rm -rf "$temp_home"
}

test_wt_command_runs_checks_first() {
    # Verify that checks run before parsing args
    local output
    output=$("$PROJECT_ROOT/bin/wt" 2>&1 || true)

    # If checks pass, we should get to usage message (no args provided)
    assert_true "[[ '$output' == *'Usage'* ]]" "Should show usage when no args provided"
}
