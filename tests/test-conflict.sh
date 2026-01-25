#!/usr/bin/env bash
#
# Conflict detection and synchronization tests for wt
#

# Source the conflict library for unit testing
source "$PROJECT_ROOT/lib/conflict.sh"

# Helper to create a minimal wt structure
setup_conflict_test() {
	local name="${1:-test-conflict}"
	"$PROJECT_ROOT/bin/wt" "$name" >/dev/null 2>&1
	export WT_TARGET_PATH="$HOME/wt/$name"
	echo "$WT_TARGET_PATH"
}

# Test conflict detection with clean merge
test_conflict_check_clean() {
	local base
	base=$(setup_conflict_test)

	# Create a feature branch with non-conflicting change
	git -C "$base" worktree add feature -b feature-branch main >/dev/null 2>&1
	echo "feature work" > "$base/feature/feature.txt"
	git -C "$base/feature" add feature.txt
	git -C "$base/feature" commit -m "Feature work" >/dev/null 2>&1

	# Check for conflicts
	wt_conflict_check "main" "feature-branch"
}

# Test conflict detection with actual conflicts
test_conflict_check_with_conflicts() {
	local base
	base=$(setup_conflict_test)

	# Create conflicting changes
	echo "main version" > "$base/main/conflict.txt"
	git -C "$base/main" add conflict.txt
	git -C "$base/main" commit -m "Main change" >/dev/null 2>&1

	git -C "$base" worktree add feature -b feature-branch HEAD~1 >/dev/null 2>&1
	echo "feature version" > "$base/feature/conflict.txt"
	git -C "$base/feature" add conflict.txt
	git -C "$base/feature" commit -m "Feature change" >/dev/null 2>&1

	# Check for conflicts (should detect them)
	! wt_conflict_check "main" "feature-branch"
}

# Test conflict detection with multiple files
test_conflict_check_multiple_files() {
	local base
	base=$(setup_conflict_test)

	# Create changes in main
	echo "main v1" > "$base/main/file1.txt"
	echo "main v2" > "$base/main/file2.txt"
	git -C "$base/main" add file1.txt file2.txt
	git -C "$base/main" commit -m "Main changes" >/dev/null 2>&1

	# Create conflicting changes in feature
	git -C "$base" worktree add feature -b feature-branch HEAD~1 >/dev/null 2>&1
	echo "feature v1" > "$base/feature/file1.txt"
	echo "feature v2" > "$base/feature/file2.txt"
	git -C "$base/feature" add file1.txt file2.txt
	git -C "$base/feature" commit -m "Feature changes" >/dev/null 2>&1

	# Should detect conflicts
	! wt_conflict_check "main" "feature-branch"
}

# Test getting list of conflicting files
test_conflict_files() {
	local base
	base=$(setup_conflict_test)

	# Create conflicting change
	echo "main version" > "$base/main/conflict.txt"
	git -C "$base/main" add conflict.txt
	git -C "$base/main" commit -m "Main change" >/dev/null 2>&1

	git -C "$base" worktree add feature -b feature-branch HEAD~1 >/dev/null 2>&1
	echo "feature version" > "$base/feature/conflict.txt"
	git -C "$base/feature" add conflict.txt
	git -C "$base/feature" commit -m "Feature change" >/dev/null 2>&1

	local files
	files=$(wt_conflict_files "main" "feature-branch")

	assert_true "[[ '$files' == *'conflict.txt'* ]]" "Should list conflict.txt"
}

# Test divergence calculation - ahead only
test_conflict_divergence_ahead() {
	local base
	base=$(setup_conflict_test)

	# Create commits in feature branch
	git -C "$base" worktree add feature -b feature-branch main >/dev/null 2>&1
	echo "work1" > "$base/feature/work.txt"
	git -C "$base/feature" add work.txt
	git -C "$base/feature" commit -m "Work 1" >/dev/null 2>&1

	echo "work2" >> "$base/feature/work.txt"
	git -C "$base/feature" add work.txt
	git -C "$base/feature" commit -m "Work 2" >/dev/null 2>&1

	local divergence ahead behind
	divergence=$(wt_conflict_divergence "main" "feature-branch")
	read -r ahead behind <<< "$divergence"

	assert_equals "2" "$ahead" "Should be 2 commits ahead" &&
	assert_equals "0" "$behind" "Should be 0 commits behind"
}

# Test divergence calculation - behind only
test_conflict_divergence_behind() {
	local base
	base=$(setup_conflict_test)

	git -C "$base" worktree add feature -b feature-branch main >/dev/null 2>&1

	# Create commits in main
	echo "main work" > "$base/main/main.txt"
	git -C "$base/main" add main.txt
	git -C "$base/main" commit -m "Main work 1" >/dev/null 2>&1

	echo "more main work" >> "$base/main/main.txt"
	git -C "$base/main" add main.txt
	git -C "$base/main" commit -m "Main work 2" >/dev/null 2>&1

	local divergence ahead behind
	divergence=$(wt_conflict_divergence "main" "feature-branch")
	read -r ahead behind <<< "$divergence"

	assert_equals "0" "$ahead" "Should be 0 commits ahead" &&
	assert_equals "2" "$behind" "Should be 2 commits behind"
}

# Test divergence calculation - both ahead and behind
test_conflict_divergence_both() {
	local base
	base=$(setup_conflict_test)

	git -C "$base" worktree add feature -b feature-branch main >/dev/null 2>&1

	# Commit in feature
	echo "feature work" > "$base/feature/feature.txt"
	git -C "$base/feature" add feature.txt
	git -C "$base/feature" commit -m "Feature work" >/dev/null 2>&1

	# Commit in main
	echo "main work" > "$base/main/main.txt"
	git -C "$base/main" add main.txt
	git -C "$base/main" commit -m "Main work" >/dev/null 2>&1

	local divergence ahead behind
	divergence=$(wt_conflict_divergence "main" "feature-branch")
	read -r ahead behind <<< "$divergence"

	assert_equals "1" "$ahead" "Should be 1 commit ahead" &&
	assert_equals "1" "$behind" "Should be 1 commit behind"
}

# Test detecting uncommitted staged changes
test_conflict_has_changes_staged() {
	local base
	base=$(setup_conflict_test)

	git -C "$base" worktree add feature -b feature-branch main >/dev/null 2>&1

	# Stage a change
	echo "staged work" > "$base/feature/staged.txt"
	git -C "$base/feature" add staged.txt

	wt_conflict_has_changes "$base/feature"
}

# Test detecting uncommitted unstaged changes
test_conflict_has_changes_unstaged() {
	local base
	base=$(setup_conflict_test)

	git -C "$base" worktree add feature -b feature-branch main >/dev/null 2>&1

	# Create a file and commit it first
	echo "initial" > "$base/feature/file.txt"
	git -C "$base/feature" add file.txt
	git -C "$base/feature" commit -m "Initial" >/dev/null 2>&1

	# Make unstaged change
	echo "modified" > "$base/feature/file.txt"

	wt_conflict_has_changes "$base/feature"
}

# Test detecting untracked files
test_conflict_has_changes_untracked() {
	local base
	base=$(setup_conflict_test)

	git -C "$base" worktree add feature -b feature-branch main >/dev/null 2>&1

	# Create untracked file
	echo "untracked" > "$base/feature/untracked.txt"

	wt_conflict_has_changes "$base/feature"
}

# Test clean working directory returns false
test_conflict_has_changes_clean() {
	local base
	base=$(setup_conflict_test)

	git -C "$base" worktree add feature -b feature-branch main >/dev/null 2>&1

	! wt_conflict_has_changes "$base/feature"
}

# Test sync when already up to date
test_conflict_sync_up_to_date() {
	local base
	base=$(setup_conflict_test)

	git -C "$base" worktree add agent-test -b test-branch main >/dev/null 2>&1

	local output
	output=$(wt_conflict_sync "agent-test" "main" 2>&1)

	assert_true "[[ '$output' == *'Already up to date'* ]]"
}

# Test sync detects uncommitted changes
test_conflict_sync_requires_clean() {
	local base
	base=$(setup_conflict_test)

	git -C "$base" worktree add agent-test -b test-branch main >/dev/null 2>&1

	# Make uncommitted change
	echo "uncommitted" > "$base/agent-test/file.txt"

	local output
	output=$(wt_conflict_sync "agent-test" "main" 2>&1 || true)

	assert_true "[[ '$output' == *'Uncommitted changes'* ]]"
}

# Test sync shows behind status
test_conflict_sync_shows_behind() {
	local base
	base=$(setup_conflict_test)

	git -C "$base" worktree add agent-test -b test-branch main >/dev/null 2>&1

	# Make change in main
	echo "main work" > "$base/main/main.txt"
	git -C "$base/main" add main.txt
	git -C "$base/main" commit -m "Main work" >/dev/null 2>&1

	local output
	output=$(wt_conflict_sync "agent-test" "main" 2>&1)

	assert_true "[[ '$output' == *'behind'* ]]" &&
	assert_true "[[ '$output' == *'--auto-rebase'* ]]"
}

# Test sync with auto-rebase
test_conflict_sync_auto_rebase() {
	local base
	base=$(setup_conflict_test)

	git -C "$base" worktree add agent-test -b test-branch main >/dev/null 2>&1

	# Feature commits non-conflicting work
	echo "feature" > "$base/agent-test/feature.txt"
	git -C "$base/agent-test" add feature.txt
	git -C "$base/agent-test" commit -m "Feature" >/dev/null 2>&1

	# Main makes a change
	echo "main work" > "$base/main/main.txt"
	git -C "$base/main" add main.txt
	git -C "$base/main" commit -m "Main work" >/dev/null 2>&1

	local output
	output=$(wt_conflict_sync "agent-test" "main" "--auto-rebase" 2>&1)

	assert_true "[[ '$output' == *'Rebase successful'* ]]"
}

# Test sync auto-rebase detects conflicts
test_conflict_sync_rebase_fails_on_conflicts() {
	local base
	base=$(setup_conflict_test)

	# Create conflicting changes
	echo "main version" > "$base/main/conflict.txt"
	git -C "$base/main" add conflict.txt
	git -C "$base/main" commit -m "Main" >/dev/null 2>&1

	git -C "$base" worktree add agent-test -b test-branch HEAD~1 >/dev/null 2>&1
	echo "feature version" > "$base/agent-test/conflict.txt"
	git -C "$base/agent-test" add conflict.txt
	git -C "$base/agent-test" commit -m "Feature" >/dev/null 2>&1

	local output
	output=$(wt_conflict_sync "agent-test" "main" "--auto-rebase" 2>&1 || true)

	assert_true "[[ '$output' == *'Rebase failed'* ]]"
}

# Test conflict check report with clean merge
test_conflict_check_report_clean() {
	local base
	base=$(setup_conflict_test)

	git -C "$base" worktree add agent-test -b test-branch main >/dev/null 2>&1

	# Non-conflicting change
	echo "work" > "$base/agent-test/work.txt"
	git -C "$base/agent-test" add work.txt
	git -C "$base/agent-test" commit -m "Work" >/dev/null 2>&1

	local output
	output=$(wt_conflict_check_report "agent-test" "main" 2>&1)

	assert_true "[[ '$output' == *'Checking merge'* ]]" &&
	assert_true "[[ '$output' == *'Clean merge'* ]]" &&
	assert_true "[[ '$output' == *'1 commits ahead'* ]]"
}

# Test conflict check report detects uncommitted changes
test_conflict_check_report_uncommitted() {
	local base
	base=$(setup_conflict_test)

	git -C "$base" worktree add agent-test -b test-branch main >/dev/null 2>&1

	# Uncommitted change
	echo "uncommitted" > "$base/agent-test/file.txt"

	local output
	output=$(wt_conflict_check_report "agent-test" "main" 2>&1)

	assert_true "[[ '$output' == *'Uncommitted changes'* ]]"
}

# Test conflict check report detects conflicts
test_conflict_check_report_with_conflicts() {
	local base
	base=$(setup_conflict_test)

	# Create conflicting changes
	echo "main version" > "$base/main/conflict.txt"
	git -C "$base/main" add conflict.txt
	git -C "$base/main" commit -m "Main" >/dev/null 2>&1

	git -C "$base" worktree add agent-test -b test-branch HEAD~1 >/dev/null 2>&1
	echo "feature version" > "$base/agent-test/conflict.txt"
	git -C "$base/agent-test" add conflict.txt
	git -C "$base/agent-test" commit -m "Feature" >/dev/null 2>&1

	local output
	output=$(wt_conflict_check_report "agent-test" "main" 2>&1 || true)

	assert_true "[[ '$output' == *'Conflicts detected'* ]]" &&
	assert_true "[[ '$output' == *'conflict.txt'* ]]"
}

# Test conflict check report shows behind warning
test_conflict_check_report_behind() {
	local base
	base=$(setup_conflict_test)

	git -C "$base" worktree add agent-test -b test-branch main >/dev/null 2>&1

	# Main moves forward
	echo "main work" > "$base/main/main.txt"
	git -C "$base/main" add main.txt
	git -C "$base/main" commit -m "Main" >/dev/null 2>&1

	local output
	output=$(wt_conflict_check_report "agent-test" "main" 2>&1)

	assert_true "[[ '$output' == *'behind'* ]]" &&
	assert_true "[[ '$output' == *'Consider rebasing'* ]]"
}

# Test conflict detection with merge base calculation
test_conflict_merge_base() {
	local base
	base=$(setup_conflict_test)

	git -C "$base" worktree add feature -b feature-branch main >/dev/null 2>&1

	# Both branches make changes
	echo "main" > "$base/main/main.txt"
	git -C "$base/main" add main.txt
	git -C "$base/main" commit -m "Main" >/dev/null 2>&1

	echo "feature" > "$base/feature/feature.txt"
	git -C "$base/feature" add feature.txt
	git -C "$base/feature" commit -m "Feature" >/dev/null 2>&1

	# Should still merge cleanly (different files)
	wt_conflict_check "main" "feature-branch"
}

# Test conflict detection handles non-existent branches gracefully
test_conflict_check_invalid_branch() {
	local base
	base=$(setup_conflict_test)

	! wt_conflict_check "main" "nonexistent-branch" 2>/dev/null
}
