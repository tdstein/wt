#!/usr/bin/env bash
# Metadata management for wt agent worktrees

# Get metadata directory path
wt_metadata_dir() {
	echo "$WT_TARGET_PATH/.bare/worktree-metadata"
}

# Get metadata file path for an agent
wt_metadata_file() {
	local agent="$1"
	echo "$(wt_metadata_dir)/${agent}.json"
}

# Ensure metadata directory exists
wt_metadata_init() {
	local metadata_dir
	metadata_dir="$(wt_metadata_dir)"
	mkdir -p "$metadata_dir"
}

# Create metadata for an agent worktree
# Args: agent_name, task_id, branch, base_branch
wt_metadata_create() {
	local agent="$1"
	local task_id="$2"
	local branch="$3"
	local base_branch="$4"
	local timestamp

	timestamp="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

	wt_metadata_init

	cat > "$(wt_metadata_file "$agent")" <<EOF
{
  "agent": "$agent",
  "task_id": "$task_id",
  "branch": "$branch",
  "base_branch": "$base_branch",
  "created": "$timestamp",
  "last_activity": "$timestamp",
  "status": "active"
}
EOF
}

# Update last activity timestamp for an agent
wt_metadata_touch() {
	local agent="$1"
	local metadata_file
	local timestamp

	metadata_file="$(wt_metadata_file "$agent")"

	if [[ ! -f "$metadata_file" ]]; then
		return 1
	fi

	timestamp="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

	# Use simple sed to update last_activity (portable across macOS and Linux)
	if [[ "$OSTYPE" == "darwin"* ]]; then
		sed -i '' "s/\"last_activity\": \"[^\"]*\"/\"last_activity\": \"$timestamp\"/" "$metadata_file"
	else
		sed -i "s/\"last_activity\": \"[^\"]*\"/\"last_activity\": \"$timestamp\"/" "$metadata_file"
	fi
}

# Get metadata value for an agent
# Args: agent_name, field
wt_metadata_get() {
	local agent="$1"
	local field="$2"
	local metadata_file

	metadata_file="$(wt_metadata_file "$agent")"

	if [[ ! -f "$metadata_file" ]]; then
		return 1
	fi

	# Simple JSON extraction (works for single-line values)
	grep "\"$field\":" "$metadata_file" | sed 's/.*: "\([^"]*\)".*/\1/' | tr -d ' '
}

# Remove metadata for an agent
wt_metadata_remove() {
	local agent="$1"
	local metadata_file

	metadata_file="$(wt_metadata_file "$agent")"

	if [[ -f "$metadata_file" ]]; then
		rm -f "$metadata_file"
	fi
}

# List all agent metadata files
wt_metadata_list() {
	local metadata_dir
	metadata_dir="$(wt_metadata_dir)"

	if [[ ! -d "$metadata_dir" ]]; then
		return 0
	fi

	find "$metadata_dir" -name "*.json" -type f 2>/dev/null | sort
}

# Check if agent has metadata
wt_metadata_exists() {
	local agent="$1"
	[[ -f "$(wt_metadata_file "$agent")" ]]
}

# Get age of worktree in seconds (since last activity)
wt_metadata_age() {
	local agent="$1"
	local last_activity
	local current_time
	local age

	last_activity="$(wt_metadata_get "$agent" "last_activity")"

	if [[ -z "$last_activity" ]]; then
		echo "0"
		return 1
	fi

	# Convert ISO 8601 to epoch (portable)
	if [[ "$OSTYPE" == "darwin"* ]]; then
		current_time="$(date -u +%s)"
		last_activity_epoch="$(date -j -f "%Y-%m-%dT%H:%M:%SZ" "$last_activity" +%s 2>/dev/null || echo 0)"
	else
		current_time="$(date -u +%s)"
		last_activity_epoch="$(date -d "$last_activity" +%s 2>/dev/null || echo 0)"
	fi

	age=$((current_time - last_activity_epoch))
	echo "$age"
}

# Format age in human-readable format
wt_metadata_age_human() {
	local age="$1"
	local days hours minutes

	if [[ "$age" -lt 60 ]]; then
		echo "${age}s"
	elif [[ "$age" -lt 3600 ]]; then
		minutes=$((age / 60))
		echo "${minutes}m"
	elif [[ "$age" -lt 86400 ]]; then
		hours=$((age / 3600))
		echo "${hours}h"
	else
		days=$((age / 86400))
		echo "${days}d"
	fi
}
