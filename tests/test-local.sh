#!/usr/bin/env bash
#
# Tests for local repository creation
#

source "$PROJECT_ROOT/lib/repo.sh"

# Test local repo initialization
test_setup_local_creates_structure() {
    wt_parse_args "test-project"
    wt_setup_local

    assert_dir_exists "$WT_TARGET_PATH/.bare" &&
    assert_file_exists "$WT_TARGET_PATH/.git" &&
    assert_dir_exists "$WT_TARGET_PATH/main"
}

test_setup_local_creates_valid_bare_repo() {
    wt_parse_args "test-project"
    wt_setup_local

    # Verify it's a valid git bare repo
    assert_true "[[ -d '$WT_TARGET_PATH/.bare/objects' ]]" &&
    assert_true "[[ -d '$WT_TARGET_PATH/.bare/refs' ]]" &&
    assert_file_exists "$WT_TARGET_PATH/.bare/HEAD"
}

test_setup_local_creates_git_pointer() {
    wt_parse_args "test-project"
    wt_setup_local

    assert_file_contains "$WT_TARGET_PATH/.git" "gitdir: ./.bare"
}

test_setup_local_main_has_commit() {
    wt_parse_args "test-project"
    wt_setup_local

    # Check that main worktree has at least one commit
    local commit_count
    commit_count=$(git -C "$WT_TARGET_PATH/main" rev-list --count HEAD 2>/dev/null || echo 0)
    assert_equals "1" "$commit_count" "Main should have initial commit"
}

test_setup_local_worktree_list() {
    wt_parse_args "test-project"
    wt_setup_local

    local output
    output=$(wt_list_worktrees)
    assert_true "[[ '$output' == *'.bare'* ]]" "Should list bare repo" &&
    assert_true "[[ '$output' == *'main'* ]]" "Should list main worktree"
}

test_target_exists_returns_true_when_exists() {
    wt_parse_args "test-project"
    mkdir -p "$WT_TARGET_PATH"
    wt_target_exists
}

test_target_exists_returns_false_when_not_exists() {
    wt_parse_args "nonexistent-project"
    ! wt_target_exists
}

test_remove_target_cleans_up() {
    wt_parse_args "test-project"
    mkdir -p "$WT_TARGET_PATH/some-dir"
    touch "$WT_TARGET_PATH/some-file"

    wt_remove_target

    ! [[ -d "$WT_TARGET_PATH" ]]
}

# Test adding additional worktrees
test_can_add_agent_worktree() {
    wt_parse_args "test-project"
    wt_setup_local

    # Add a new worktree like an agent would
    git -C "$WT_TARGET_PATH" worktree add agent-1 -b feature-1 main >/dev/null 2>&1

    assert_dir_exists "$WT_TARGET_PATH/agent-1" &&
    assert_true "[[ -f '$WT_TARGET_PATH/agent-1/.git' ]]"
}

test_worktrees_share_objects() {
    wt_parse_args "test-project"
    wt_setup_local

    # Add a worktree
    git -C "$WT_TARGET_PATH" worktree add agent-1 -b feature-1 main >/dev/null 2>&1

    # Create a commit in agent-1
    touch "$WT_TARGET_PATH/agent-1/test-file"
    git -C "$WT_TARGET_PATH/agent-1" add test-file
    git -C "$WT_TARGET_PATH/agent-1" commit -m "Test commit" >/dev/null 2>&1

    # The object should be in .bare, not in agent-1
    local obj_count_bare obj_count_main
    obj_count_bare=$(find "$WT_TARGET_PATH/.bare/objects" -type f | wc -l)

    # Objects should be shared in .bare
    assert_true "[[ $obj_count_bare -gt 0 ]]" "Objects should be in .bare"
}

test_multiple_worktrees_isolated() {
    wt_parse_args "test-project"
    wt_setup_local

    # Add two worktrees
    git -C "$WT_TARGET_PATH" worktree add agent-1 -b feature-1 main >/dev/null 2>&1
    git -C "$WT_TARGET_PATH" worktree add agent-2 -b feature-2 main >/dev/null 2>&1

    # Create different files in each
    echo "agent1" > "$WT_TARGET_PATH/agent-1/file1"
    echo "agent2" > "$WT_TARGET_PATH/agent-2/file2"

    # Files should not appear in other worktrees
    assert_false "[[ -f '$WT_TARGET_PATH/agent-2/file1' ]]" &&
    assert_false "[[ -f '$WT_TARGET_PATH/agent-1/file2' ]]" &&
    assert_false "[[ -f '$WT_TARGET_PATH/main/file1' ]]" &&
    assert_false "[[ -f '$WT_TARGET_PATH/main/file2' ]]"
}
