#!/usr/bin/env bash
#
# Integration tests for wt CLI
#

# Test the full wt script
test_wt_local_project_e2e() {
    # Run wt script directly
    echo "n" | "$PROJECT_ROOT/bin/wt" "integration-test" >/dev/null 2>&1 || true
    "$PROJECT_ROOT/bin/wt" "integration-test" >/dev/null 2>&1

    assert_dir_exists "$HOME/wt/integration-test/.bare" &&
    assert_file_exists "$HOME/wt/integration-test/.git" &&
    assert_dir_exists "$HOME/wt/integration-test/main"
}

test_wt_rejects_existing_noninteractive() {
    # Create existing directory
    mkdir -p "$HOME/wt/existing-project"

    # Should fail in non-interactive mode
    ! "$PROJECT_ROOT/bin/wt" "existing-project" </dev/null 2>/dev/null
}

test_wt_usage_shows_help() {
    local output
    output=$("$PROJECT_ROOT/bin/wt" 2>&1 || true)

    assert_true "[[ '$output' == *'Usage'* ]]"
}

test_wt_creates_correct_gitdir() {
    "$PROJECT_ROOT/bin/wt" "gitdir-test" >/dev/null 2>&1

    local gitdir_content
    gitdir_content=$(cat "$HOME/wt/gitdir-test/.git")
    assert_equals "gitdir: ./.bare" "$gitdir_content"
}

test_wt_main_worktree_is_functional() {
    "$PROJECT_ROOT/bin/wt" "functional-test" >/dev/null 2>&1

    # Should be able to run git commands
    git -C "$HOME/wt/functional-test/main" status >/dev/null 2>&1 &&
    git -C "$HOME/wt/functional-test/main" log --oneline -1 >/dev/null 2>&1
}

test_wt_head_points_to_main() {
    "$PROJECT_ROOT/bin/wt" "head-test" >/dev/null 2>&1

    local head_ref
    head_ref=$(git -C "$HOME/wt/head-test/.bare" symbolic-ref HEAD)
    assert_equals "refs/heads/main" "$head_ref"
}

# Test worktree workflow simulation
test_parallel_agent_workflow() {
    "$PROJECT_ROOT/bin/wt" "parallel-test" >/dev/null 2>&1
    local base="$HOME/wt/parallel-test"

    # Simulate CEO creating worktrees for agents
    git -C "$base" worktree add agent-alice -b feature-auth main >/dev/null 2>&1
    git -C "$base" worktree add agent-bob -b feature-ui main >/dev/null 2>&1

    # Each agent works independently
    echo "auth code" > "$base/agent-alice/auth.txt"
    git -C "$base/agent-alice" add auth.txt
    git -C "$base/agent-alice" commit -m "Add auth" >/dev/null 2>&1

    echo "ui code" > "$base/agent-bob/ui.txt"
    git -C "$base/agent-bob" add ui.txt
    git -C "$base/agent-bob" commit -m "Add UI" >/dev/null 2>&1

    # Verify independent commits
    local alice_commits bob_commits
    alice_commits=$(git -C "$base/agent-alice" log --oneline | wc -l | tr -d ' ')
    bob_commits=$(git -C "$base/agent-bob" log --oneline | wc -l | tr -d ' ')

    assert_equals "2" "$alice_commits" "Alice should have 2 commits" &&
    assert_equals "2" "$bob_commits" "Bob should have 2 commits"
}

test_worktree_can_see_other_branches() {
    "$PROJECT_ROOT/bin/wt" "branch-test" >/dev/null 2>&1
    local base="$HOME/wt/branch-test"

    # Create agent worktree and make commits
    git -C "$base" worktree add agent-1 -b feature main >/dev/null 2>&1
    echo "test" > "$base/agent-1/file.txt"
    git -C "$base/agent-1" add file.txt
    git -C "$base/agent-1" commit -m "Feature commit" >/dev/null 2>&1

    # Main worktree should be able to see the feature branch
    local branches
    branches=$(git -C "$base/main" branch -a)
    assert_true "[[ '$branches' == *'feature'* ]]" "Main should see feature branch"
}

test_worktree_can_merge() {
    "$PROJECT_ROOT/bin/wt" "merge-test" >/dev/null 2>&1
    local base="$HOME/wt/merge-test"

    # Create and commit on feature branch
    git -C "$base" worktree add agent-1 -b feature main >/dev/null 2>&1
    echo "feature" > "$base/agent-1/feature.txt"
    git -C "$base/agent-1" add feature.txt
    git -C "$base/agent-1" commit -m "Add feature" >/dev/null 2>&1

    # Merge from main worktree
    git -C "$base/main" merge feature --no-edit >/dev/null 2>&1

    # File should now be in main
    assert_file_exists "$base/main/feature.txt"
}

test_worktree_prune_works() {
    "$PROJECT_ROOT/bin/wt" "prune-test" >/dev/null 2>&1
    local base="$HOME/wt/prune-test"

    # Create and then manually remove a worktree directory
    git -C "$base" worktree add temp-agent -b temp main >/dev/null 2>&1
    rm -rf "$base/temp-agent"

    # Prune should clean up the stale worktree
    git -C "$base" worktree prune

    # Should only show 2 worktrees now (bare and main)
    local wt_count
    wt_count=$(git -C "$base" worktree list | wc -l | tr -d ' ')
    assert_equals "2" "$wt_count" "Should have 2 worktrees after prune"
}
