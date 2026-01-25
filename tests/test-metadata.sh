#!/usr/bin/env bash
#
# Metadata management unit tests for wt
#

# Source the metadata library for unit testing
source "$PROJECT_ROOT/lib/metadata.sh"

# Helper to create a minimal wt structure
setup_metadata_test() {
	local name="${1:-test-meta}"
	"$PROJECT_ROOT/bin/wt" "$name" >/dev/null 2>&1
	export WT_TARGET_PATH="$HOME/wt/$name"
	echo "$WT_TARGET_PATH"
}

# Test metadata directory creation
test_metadata_init() {
	local base
	base=$(setup_metadata_test)

	wt_metadata_init

	assert_dir_exists "$base/.bare/worktree-metadata"
}

# Test metadata file path generation
test_metadata_file_path() {
	local base
	base=$(setup_metadata_test)

	local path
	path=$(wt_metadata_file "alice")

	assert_equals "$base/.bare/worktree-metadata/alice.json" "$path"
}

# Test metadata creation
test_metadata_create() {
	local base
	base=$(setup_metadata_test)

	wt_metadata_create "alice" "1234" "task/1234/alice" "main"

	local metadata="$base/.bare/worktree-metadata/alice.json"
	assert_file_exists "$metadata" &&
	assert_file_contains "$metadata" '"agent": "alice"' &&
	assert_file_contains "$metadata" '"task_id": "1234"' &&
	assert_file_contains "$metadata" '"branch": "task/1234/alice"' &&
	assert_file_contains "$metadata" '"base_branch": "main"'
}

# Test metadata get
test_metadata_get() {
	local base
	base=$(setup_metadata_test)

	wt_metadata_create "alice" "1234" "task/1234/alice" "main"

	local agent task_id branch base_branch
	agent=$(wt_metadata_get "alice" "agent")
	task_id=$(wt_metadata_get "alice" "task_id")
	branch=$(wt_metadata_get "alice" "branch")
	base_branch=$(wt_metadata_get "alice" "base_branch")

	assert_equals "alice" "$agent" &&
	assert_equals "1234" "$task_id" &&
	assert_equals "task/1234/alice" "$branch" &&
	assert_equals "main" "$base_branch"
}

# Test metadata get returns error for nonexistent agent
test_metadata_get_nonexistent() {
	local base
	base=$(setup_metadata_test)

	! wt_metadata_get "nonexistent" "agent" 2>/dev/null
}

# Test metadata exists
test_metadata_exists() {
	local base
	base=$(setup_metadata_test)

	wt_metadata_create "alice" "1234" "task/1234/alice" "main"

	wt_metadata_exists "alice" &&
	! wt_metadata_exists "bob"
}

# Test metadata touch updates timestamp
test_metadata_touch() {
	local base
	base=$(setup_metadata_test)

	wt_metadata_create "alice" "1234" "task/1234/alice" "main"

	# Get initial timestamp
	local ts1
	ts1=$(wt_metadata_get "alice" "last_activity")

	# Wait a moment and touch
	sleep 1
	wt_metadata_touch "alice"

	local ts2
	ts2=$(wt_metadata_get "alice" "last_activity")

	assert_false "[[ '$ts1' == '$ts2' ]]" "Timestamp should be updated"
}

# Test metadata remove
test_metadata_remove() {
	local base
	base=$(setup_metadata_test)

	wt_metadata_create "alice" "1234" "task/1234/alice" "main"
	local metadata="$base/.bare/worktree-metadata/alice.json"

	assert_file_exists "$metadata"

	wt_metadata_remove "alice"

	assert_false "[[ -f '$metadata' ]]"
}

# Test metadata list
test_metadata_list() {
	local base
	base=$(setup_metadata_test)

	wt_metadata_create "alice" "1234" "task/1234/alice" "main"
	wt_metadata_create "bob" "5678" "task/5678/bob" "main"

	local output
	output=$(wt_metadata_list)

	assert_true "[[ '$output' == *'alice.json'* ]]" &&
	assert_true "[[ '$output' == *'bob.json'* ]]"
}

# Test metadata list empty
test_metadata_list_empty() {
	local base
	base=$(setup_metadata_test)

	local output
	output=$(wt_metadata_list)

	assert_equals "" "$output"
}

# Test metadata age calculation
test_metadata_age() {
	local base
	base=$(setup_metadata_test)

	wt_metadata_create "alice" "1234" "task/1234/alice" "main"

	sleep 2

	local age
	age=$(wt_metadata_age "alice")

	# Age should be at least 1 second
	assert_true "[[ '$age' -ge 1 ]]" "Age should be at least 1 second"
}

# Test metadata age human readable format
test_metadata_age_human() {
	assert_equals "30s" "$(wt_metadata_age_human 30)" &&
	assert_equals "5m" "$(wt_metadata_age_human 300)" &&
	assert_equals "2h" "$(wt_metadata_age_human 7200)" &&
	assert_equals "3d" "$(wt_metadata_age_human 259200)"
}

# Test metadata survives across multiple operations
test_metadata_persistence() {
	local base
	base=$(setup_metadata_test)

	wt_metadata_create "alice" "1234" "task/1234/alice" "main"
	wt_metadata_touch "alice"

	# Read after touch
	local agent
	agent=$(wt_metadata_get "alice" "agent")
	assert_equals "alice" "$agent"

	wt_metadata_touch "alice"

	# Read again after second touch
	agent=$(wt_metadata_get "alice" "agent")
	assert_equals "alice" "$agent"
}

# Test metadata with special characters in values
test_metadata_special_chars() {
	local base
	base=$(setup_metadata_test)

	wt_metadata_create "agent-1" "task-123" "feature/task-123/agent-1" "release/v1.0"

	local agent branch
	agent=$(wt_metadata_get "agent-1" "agent")
	branch=$(wt_metadata_get "agent-1" "branch")

	assert_equals "agent-1" "$agent" &&
	assert_equals "feature/task-123/agent-1" "$branch"
}

# Test metadata status field
test_metadata_status() {
	local base
	base=$(setup_metadata_test)

	wt_metadata_create "alice" "1234" "task/1234/alice" "main"

	local status
	status=$(wt_metadata_get "alice" "status")

	assert_equals "active" "$status"
}

# Test metadata created and last_activity timestamps
test_metadata_timestamps() {
	local base
	base=$(setup_metadata_test)

	wt_metadata_create "alice" "1234" "task/1234/alice" "main"

	local created last_activity
	created=$(wt_metadata_get "alice" "created")
	last_activity=$(wt_metadata_get "alice" "last_activity")

	# Both should be non-empty ISO 8601 timestamps
	assert_true "[[ -n '$created' ]]" "Created timestamp should exist" &&
	assert_true "[[ -n '$last_activity' ]]" "Last activity timestamp should exist" &&
	assert_true "[[ '$created' =~ [0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z ]]" "Created should be ISO 8601" &&
	assert_true "[[ '$last_activity' =~ [0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z ]]" "Last activity should be ISO 8601"
}
