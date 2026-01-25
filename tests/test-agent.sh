#!/usr/bin/env bash
#
# Agent management tests for wt
#

# Helper to create a wt project for testing
setup_wt_project() {
	local name="${1:-test-project}"
	"$PROJECT_ROOT/bin/wt" "$name" >/dev/null 2>&1
	echo "$HOME/wt/$name"
}

# Test agent create command
test_agent_create() {
	local base
	base=$(setup_wt_project)

	cd "$base"
	"$PROJECT_ROOT/bin/wt" agent create alice 1234 main >/dev/null 2>&1

	assert_dir_exists "$base/alice" "Agent worktree should be created" &&
	assert_dir_exists "$base/.bare/worktree-metadata" "Metadata dir should be created" &&
	assert_file_exists "$base/.bare/worktree-metadata/alice.json" "Metadata file should be created"
}

# Test agent create with default base branch
test_agent_create_default_base() {
	local base
	base=$(setup_wt_project)

	cd "$base"
	"$PROJECT_ROOT/bin/wt" agent create bob 5678 >/dev/null 2>&1

	local branch
	branch=$(git -C "$base/bob" rev-parse --abbrev-ref HEAD 2>/dev/null)
	assert_equals "task/5678/bob" "$branch" "Should create branch with correct name"
}

# Test agent create validates agent name
test_agent_create_validates_name() {
	local base
	base=$(setup_wt_project)

	cd "$base"
	! "$PROJECT_ROOT/bin/wt" agent create "bad name" 1234 2>/dev/null
}

# Test agent create rejects duplicate agent
test_agent_create_rejects_duplicate() {
	local base
	base=$(setup_wt_project)

	cd "$base"
	"$PROJECT_ROOT/bin/wt" agent create alice 1234 >/dev/null 2>&1

	# Try to create again
	! "$PROJECT_ROOT/bin/wt" agent create alice 5678 2>/dev/null
}

# Test agent list shows agents
test_agent_list() {
	local base
	base=$(setup_wt_project)

	cd "$base"
	"$PROJECT_ROOT/bin/wt" agent create alice 1234 >/dev/null 2>&1
	"$PROJECT_ROOT/bin/wt" agent create bob 5678 >/dev/null 2>&1

	local output
	output=$("$PROJECT_ROOT/bin/wt" agent list 2>&1)

	assert_true "[[ '$output' == *'alice'* ]]" "Should show alice" &&
	assert_true "[[ '$output' == *'bob'* ]]" "Should show bob" &&
	assert_true "[[ '$output' == *'1234'* ]]" "Should show task 1234" &&
	assert_true "[[ '$output' == *'5678'* ]]" "Should show task 5678"
}

# Test agent list shows no agents
test_agent_list_empty() {
	local base
	base=$(setup_wt_project)

	cd "$base"
	local output
	output=$("$PROJECT_ROOT/bin/wt" agent list 2>&1)

	assert_true "[[ '$output' == *'No agent worktrees'* ]]"
}

# Test agent status dashboard
test_agent_status() {
	local base
	base=$(setup_wt_project)

	cd "$base"
	"$PROJECT_ROOT/bin/wt" agent create alice 1234 >/dev/null 2>&1

	local output
	output=$("$PROJECT_ROOT/bin/wt" agent status 2>&1)

	assert_true "[[ '$output' == *'Worktree Status Dashboard'* ]]" &&
	assert_true "[[ '$output' == *'Total worktrees: 1'* ]]" &&
	assert_true "[[ '$output' == *'Active worktrees: 1'* ]]"
}

# Test agent remove
test_agent_remove() {
	local base
	base=$(setup_wt_project)

	cd "$base"
	"$PROJECT_ROOT/bin/wt" agent create alice 1234 >/dev/null 2>&1
	"$PROJECT_ROOT/bin/wt" agent remove alice >/dev/null 2>&1

	assert_false "[[ -d '$base/alice' ]]" "Worktree should be removed" &&
	assert_false "[[ -f '$base/.bare/worktree-metadata/alice.json' ]]" "Metadata should be removed"
}

# Test agent remove with branch deletion
test_agent_remove_with_branch() {
	local base
	base=$(setup_wt_project)

	cd "$base"
	"$PROJECT_ROOT/bin/wt" agent create alice 1234 >/dev/null 2>&1

	# Merge to main first so branch can be deleted
	git -C "$base/main" merge --no-edit task/1234/alice >/dev/null 2>&1

	"$PROJECT_ROOT/bin/wt" agent remove alice --delete-branch >/dev/null 2>&1

	# Verify branch is deleted
	! git -C "$base" rev-parse --verify task/1234/alice >/dev/null 2>&1
}

# Test agent check shows status
test_agent_check() {
	local base
	base=$(setup_wt_project)

	cd "$base"
	"$PROJECT_ROOT/bin/wt" agent create alice 1234 >/dev/null 2>&1

	local output
	output=$("$PROJECT_ROOT/bin/wt" agent check alice 2>&1)

	assert_true "[[ '$output' == *'Checking merge'* ]]" &&
	assert_true "[[ '$output' == *'Clean merge'* ]]"
}

# Test agent check detects uncommitted changes
test_agent_check_uncommitted_changes() {
	local base
	base=$(setup_wt_project)

	cd "$base"
	"$PROJECT_ROOT/bin/wt" agent create alice 1234 >/dev/null 2>&1

	# Make uncommitted change
	echo "test" > "$base/alice/test.txt"

	local output
	output=$("$PROJECT_ROOT/bin/wt" agent check alice 2>&1)

	assert_true "[[ '$output' == *'Uncommitted changes'* ]]"
}

# Test agent check detects conflicts
test_agent_check_detects_conflicts() {
	local base
	base=$(setup_wt_project)

	cd "$base"
	"$PROJECT_ROOT/bin/wt" agent create alice 1234 >/dev/null 2>&1

	# Create conflicting changes
	echo "main version" > "$base/main/conflict.txt"
	git -C "$base/main" add conflict.txt
	git -C "$base/main" commit -m "Add from main" >/dev/null 2>&1

	echo "alice version" > "$base/alice/conflict.txt"
	git -C "$base/alice" add conflict.txt
	git -C "$base/alice" commit -m "Add from alice" >/dev/null 2>&1

	local output
	output=$("$PROJECT_ROOT/bin/wt" agent check alice 2>&1)

	assert_true "[[ '$output' == *'Conflicts detected'* ]]" &&
	assert_true "[[ '$output' == *'conflict.txt'* ]]"
}

# Test agent check shows divergence
test_agent_check_shows_divergence() {
	local base
	base=$(setup_wt_project)

	cd "$base"
	"$PROJECT_ROOT/bin/wt" agent create alice 1234 >/dev/null 2>&1

	# Make changes in alice
	echo "test" > "$base/alice/file.txt"
	git -C "$base/alice" add file.txt
	git -C "$base/alice" commit -m "Alice work" >/dev/null 2>&1

	local output
	output=$("$PROJECT_ROOT/bin/wt" agent check alice 2>&1)

	assert_true "[[ '$output' == *'1 commits ahead'* ]]" &&
	assert_true "[[ '$output' == *'0 commits behind'* ]]"
}

# Test agent sync shows up to date
test_agent_sync_up_to_date() {
	local base
	base=$(setup_wt_project)

	cd "$base"
	"$PROJECT_ROOT/bin/wt" agent create alice 1234 >/dev/null 2>&1

	local output
	output=$("$PROJECT_ROOT/bin/wt" agent sync alice 2>&1)

	assert_true "[[ '$output' == *'Already up to date'* ]]"
}

# Test agent sync detects uncommitted changes
test_agent_sync_requires_clean_working_dir() {
	local base
	base=$(setup_wt_project)

	cd "$base"
	"$PROJECT_ROOT/bin/wt" agent create alice 1234 >/dev/null 2>&1

	# Make uncommitted change
	echo "test" > "$base/alice/test.txt"

	local output
	output=$("$PROJECT_ROOT/bin/wt" agent sync alice 2>&1 || true)

	assert_true "[[ '$output' == *'Uncommitted changes'* ]]"
}

# Test agent sync shows behind status
test_agent_sync_shows_behind() {
	local base
	base=$(setup_wt_project)

	cd "$base"
	"$PROJECT_ROOT/bin/wt" agent create alice 1234 >/dev/null 2>&1

	# Make change in main
	echo "main work" > "$base/main/mainfile.txt"
	git -C "$base/main" add mainfile.txt
	git -C "$base/main" commit -m "Main work" >/dev/null 2>&1

	local output
	output=$("$PROJECT_ROOT/bin/wt" agent sync alice 2>&1)

	assert_true "[[ '$output' == *'behind'* ]]"
}

# Test agent prune finds no stale worktrees
test_agent_prune_none() {
	local base
	base=$(setup_wt_project)

	cd "$base"
	"$PROJECT_ROOT/bin/wt" agent create alice 1234 >/dev/null 2>&1

	local output
	output=$("$PROJECT_ROOT/bin/wt" agent prune --older-than=1d --dry-run 2>&1)

	assert_true "[[ '$output' == *'No stale worktrees'* ]]"
}

# Test metadata creation
test_metadata_structure() {
	local base
	base=$(setup_wt_project)

	cd "$base"
	"$PROJECT_ROOT/bin/wt" agent create alice 1234 main >/dev/null 2>&1

	local metadata="$base/.bare/worktree-metadata/alice.json"
	assert_file_contains "$metadata" '"agent": "alice"' &&
	assert_file_contains "$metadata" '"task_id": "1234"' &&
	assert_file_contains "$metadata" '"branch": "task/1234/alice"' &&
	assert_file_contains "$metadata" '"base_branch": "main"' &&
	assert_file_contains "$metadata" '"status": "active"'
}

# Test branch naming convention
test_branch_naming_convention() {
	local base
	base=$(setup_wt_project)

	cd "$base"
	"$PROJECT_ROOT/bin/wt" agent create alice 1234 >/dev/null 2>&1
	"$PROJECT_ROOT/bin/wt" agent create bob 5678 >/dev/null 2>&1

	local alice_branch bob_branch
	alice_branch=$(git -C "$base/alice" rev-parse --abbrev-ref HEAD)
	bob_branch=$(git -C "$base/bob" rev-parse --abbrev-ref HEAD)

	assert_equals "task/1234/alice" "$alice_branch" &&
	assert_equals "task/5678/bob" "$bob_branch"
}

# Test that agent commands require wt directory
test_agent_requires_wt_directory() {
	cd "$HOME"
	! "$PROJECT_ROOT/bin/wt" agent list 2>/dev/null
}

# Test that agent commands work from subdirectories
test_agent_works_from_subdirectory() {
	local base
	base=$(setup_wt_project)

	cd "$base"
	"$PROJECT_ROOT/bin/wt" agent create alice 1234 >/dev/null 2>&1

	# Run from main subdirectory
	cd "$base/main"
	local output
	output=$("$PROJECT_ROOT/bin/wt" agent list 2>&1)

	assert_true "[[ '$output' == *'alice'* ]]"
}

# Test multiple agents can coexist
test_multiple_agents() {
	local base
	base=$(setup_wt_project)

	cd "$base"
	"$PROJECT_ROOT/bin/wt" agent create alice 1 >/dev/null 2>&1
	"$PROJECT_ROOT/bin/wt" agent create bob 2 >/dev/null 2>&1
	"$PROJECT_ROOT/bin/wt" agent create charlie 3 >/dev/null 2>&1

	assert_dir_exists "$base/alice" &&
	assert_dir_exists "$base/bob" &&
	assert_dir_exists "$base/charlie" &&

	local output
	output=$("$PROJECT_ROOT/bin/wt" agent status 2>&1)
	assert_true "[[ '$output' == *'Total worktrees: 3'* ]]"
}

# Test agent work isolation
test_agent_work_isolation() {
	local base
	base=$(setup_wt_project)

	cd "$base"
	"$PROJECT_ROOT/bin/wt" agent create alice 1234 >/dev/null 2>&1
	"$PROJECT_ROOT/bin/wt" agent create bob 5678 >/dev/null 2>&1

	# Alice makes changes
	echo "alice work" > "$base/alice/alice.txt"
	git -C "$base/alice" add alice.txt
	git -C "$base/alice" commit -m "Alice work" >/dev/null 2>&1

	# Bob makes changes
	echo "bob work" > "$base/bob/bob.txt"
	git -C "$base/bob" add bob.txt
	git -C "$base/bob" commit -m "Bob work" >/dev/null 2>&1

	# Verify isolation - alice.txt should not exist in bob's worktree
	assert_false "[[ -f '$base/bob/alice.txt' ]]" "Bob should not see alice's file" &&
	assert_false "[[ -f '$base/alice/bob.txt' ]]" "Alice should not see bob's file"
}
