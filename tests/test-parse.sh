#!/usr/bin/env bash
#
# Tests for argument parsing functions
#

source "$PROJECT_ROOT/lib/parse.sh"

# Test URL detection
test_is_url_https() {
    wt_is_url "https://github.com/user/repo"
}

test_is_url_git_at() {
    wt_is_url "git@github.com:user/repo.git"
}

test_is_url_http() {
    wt_is_url "http://example.com/repo"
}

test_is_not_url_simple_name() {
    ! wt_is_url "my-project"
}

test_is_not_url_path() {
    ! wt_is_url "/path/to/something"
}

test_is_not_url_relative_path() {
    ! wt_is_url "../something"
}

# Test URL to dirname conversion
test_url_to_dirname_https() {
    local result
    result="$(wt_url_to_dirname "https://github.com/user/repo")"
    assert_equals "repo" "$result"
}

test_url_to_dirname_https_with_git() {
    local result
    result="$(wt_url_to_dirname "https://github.com/user/repo.git")"
    assert_equals "repo" "$result"
}

test_url_to_dirname_git_at() {
    local result
    result="$(wt_url_to_dirname "git@github.com:user/repo.git")"
    assert_equals "repo" "$result"
}

test_url_to_dirname_trailing_slash() {
    local result
    result="$(wt_url_to_dirname "https://github.com/user/repo/")"
    assert_equals "" "$result"  # Edge case: trailing slash
}

# Test argument parsing
test_parse_args_local_project() {
    wt_parse_args "my-project"
    assert_equals "local" "$WT_MODE" &&
    assert_equals "" "$WT_REPO_URL" &&
    assert_equals "my-project" "$WT_DIR_NAME"
}

test_parse_args_remote_url() {
    wt_parse_args "https://github.com/user/repo"
    assert_equals "remote" "$WT_MODE" &&
    assert_equals "https://github.com/user/repo" "$WT_REPO_URL" &&
    assert_equals "repo" "$WT_DIR_NAME"
}

test_parse_args_remote_with_custom_name() {
    wt_parse_args "https://github.com/user/repo" "my-name"
    assert_equals "remote" "$WT_MODE" &&
    assert_equals "my-name" "$WT_DIR_NAME"
}

test_parse_args_git_at_url() {
    wt_parse_args "git@github.com:user/project.git"
    assert_equals "remote" "$WT_MODE" &&
    assert_equals "project" "$WT_DIR_NAME"
}

test_parse_args_no_args_fails() {
    ! wt_parse_args 2>/dev/null
}

test_parse_args_sets_target_path() {
    wt_parse_args "test-project"
    assert_equals "$HOME/wt/test-project" "$WT_TARGET_PATH"
}
