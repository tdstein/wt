#!/usr/bin/env bash
# Agent management for wt worktrees

# Source dependencies
AGENT_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$AGENT_LIB_DIR/metadata.sh"
source "$AGENT_LIB_DIR/conflict.sh"

# Create a worktree for an agent
# Args: agent_name, task_id, [base_branch]
wt_agent_create() {
	local agent="$1"
	local task_id="$2"
	local base_branch="${3:-main}"
	local branch_name
	local worktree_path

	if [[ -z "$agent" ]] || [[ -z "$task_id" ]]; then
		echo "Error: Agent name and task ID required" >&2
		return 1
	fi

	# Validate agent name (alphanumeric, hyphens, underscores)
	if [[ ! "$agent" =~ ^[a-zA-Z0-9_-]+$ ]]; then
		echo "Error: Agent name must be alphanumeric with hyphens/underscores only" >&2
		return 1
	fi

	worktree_path="$WT_TARGET_PATH/$agent"

	# Check if worktree already exists
	if [[ -d "$worktree_path" ]]; then
		echo "Error: Worktree already exists: $worktree_path" >&2
		return 1
	fi

	# Create branch name: task/<task-id>/<agent-name>
	branch_name="task/${task_id}/${agent}"

	# Check if branch already exists
	if git -C "$WT_TARGET_PATH" rev-parse --verify "$branch_name" >/dev/null 2>&1; then
		echo "Error: Branch already exists: $branch_name" >&2
		return 1
	fi

	# Create worktree
	echo "Creating worktree: $worktree_path"
	echo "Branch: $branch_name"
	echo "Base: $base_branch"
	echo

	if ! git -C "$WT_TARGET_PATH" worktree add "$agent" -b "$branch_name" "$base_branch" 2>&1; then
		echo "Error: Failed to create worktree" >&2
		return 1
	fi

	# Create metadata
	wt_metadata_create "$agent" "$task_id" "$branch_name" "$base_branch"

	echo
	echo "✓ Worktree created successfully"
	echo
	echo "Next steps:"
	echo "  cd $worktree_path"
	echo "  # Make your changes"
	echo "  git add <files>"
	echo "  git commit -m \"Your commit message\""
	echo
	echo "Check status:"
	echo "  wt agent check $agent"

	return 0
}

# Remove an agent's worktree
# Args: agent_name, [--delete-branch]
wt_agent_remove() {
	local agent="$1"
	local delete_branch="${2:-}"
	local worktree_path
	local branch_name

	if [[ -z "$agent" ]]; then
		echo "Error: Agent name required" >&2
		return 1
	fi

	worktree_path="$WT_TARGET_PATH/$agent"

	# Check if worktree exists
	if [[ ! -d "$worktree_path" ]]; then
		echo "Error: Worktree does not exist: $worktree_path" >&2
		return 1
	fi

	# Get branch name from metadata if available
	if wt_metadata_exists "$agent"; then
		branch_name="$(wt_metadata_get "$agent" "branch")"
	else
		# Fallback: get current branch from worktree
		branch_name="$(git -C "$worktree_path" rev-parse --abbrev-ref HEAD 2>/dev/null || echo "")"
	fi

	echo "Removing worktree: $worktree_path"

	# Remove worktree
	if ! git -C "$WT_TARGET_PATH" worktree remove "$agent" 2>&1; then
		echo "Error: Failed to remove worktree" >&2
		echo "Try: git -C $WT_TARGET_PATH worktree remove --force $agent" >&2
		return 1
	fi

	# Remove metadata
	wt_metadata_remove "$agent"

	echo "✓ Worktree removed"

	# Delete branch if requested and branch name is known
	if [[ "$delete_branch" == "--delete-branch" ]] && [[ -n "$branch_name" ]]; then
		echo
		echo "Deleting branch: $branch_name"

		# Check if branch is merged
		if git -C "$WT_TARGET_PATH" branch --merged main | grep -q "$branch_name"; then
			if git -C "$WT_TARGET_PATH" branch -d "$branch_name" 2>&1; then
				echo "✓ Branch deleted"
			else
				echo "Error: Failed to delete branch" >&2
			fi
		else
			echo "Warning: Branch is not merged into main"
			echo "Use: git -C $WT_TARGET_PATH branch -D $branch_name (force delete)"
		fi
	fi

	return 0
}

# List all agent worktrees
wt_agent_list() {
	local metadata_files
	local agent
	local task_id
	local branch
	local age
	local age_human
	local status
	local has_worktrees=false

	echo "Agent Worktrees:"
	echo
	printf "%-20s %-10s %-30s %-10s %s\n" "AGENT" "TASK" "BRANCH" "AGE" "STATUS"
	printf "%-20s %-10s %-30s %-10s %s\n" "--------------------" "----------" "------------------------------" "----------" "------"

	# List from metadata
	metadata_files="$(wt_metadata_list)"

	if [[ -z "$metadata_files" ]]; then
		echo "No agent worktrees found"
		return 0
	fi

	while IFS= read -r metadata_file; do
		has_worktrees=true
		agent="$(basename "$metadata_file" .json)"
		task_id="$(wt_metadata_get "$agent" "task_id")"
		branch="$(wt_metadata_get "$agent" "branch")"
		age="$(wt_metadata_age "$agent")"
		age_human="$(wt_metadata_age_human "$age")"
		status="$(wt_metadata_get "$agent" "status")"

		# Check if worktree still exists
		if [[ ! -d "$WT_TARGET_PATH/$agent" ]]; then
			status="missing"
		fi

		printf "%-20s %-10s %-30s %-10s %s\n" "$agent" "$task_id" "$branch" "$age_human" "$status"
	done <<< "$metadata_files"

	echo
	echo "Total: $(echo "$metadata_files" | wc -l | tr -d ' ') agent(s)"
}

# Check merge status for an agent
# Args: agent_name
wt_agent_check() {
	local agent="$1"
	local base_branch
	local worktree_path

	if [[ -z "$agent" ]]; then
		echo "Error: Agent name required" >&2
		return 1
	fi

	worktree_path="$WT_TARGET_PATH/$agent"

	if [[ ! -d "$worktree_path" ]]; then
		echo "Error: Worktree does not exist: $worktree_path" >&2
		return 1
	fi

	# Get base branch from metadata
	if wt_metadata_exists "$agent"; then
		base_branch="$(wt_metadata_get "$agent" "base_branch")"
	else
		base_branch="main"
	fi

	# Update last activity
	wt_metadata_touch "$agent" 2>/dev/null || true

	# Run conflict check report
	wt_conflict_check_report "$agent" "$base_branch"
}

# Sync agent worktree with base branch
# Args: agent_name, [--auto-rebase]
wt_agent_sync() {
	local agent="$1"
	local auto_rebase="${2:-}"
	local base_branch
	local worktree_path

	if [[ -z "$agent" ]]; then
		echo "Error: Agent name required" >&2
		return 1
	fi

	worktree_path="$WT_TARGET_PATH/$agent"

	if [[ ! -d "$worktree_path" ]]; then
		echo "Error: Worktree does not exist: $worktree_path" >&2
		return 1
	fi

	# Get base branch from metadata
	if wt_metadata_exists "$agent"; then
		base_branch="$(wt_metadata_get "$agent" "base_branch")"
	else
		base_branch="main"
	fi

	# Update last activity
	wt_metadata_touch "$agent" 2>/dev/null || true

	# Sync
	wt_conflict_sync "$agent" "$base_branch" "$auto_rebase"
}

# Prune stale agent worktrees
# Args: [--older-than=Nd] [--dry-run]
wt_agent_prune() {
	local older_than_days=7
	local dry_run=false
	local metadata_files
	local agent
	local age
	local age_days
	local stale_count=0

	# Parse arguments
	for arg in "$@"; do
		case $arg in
			--older-than=*)
				older_than_days="${arg#*=}"
				older_than_days="${older_than_days%d}"
				;;
			--dry-run)
				dry_run=true
				;;
		esac
	done

	local older_than_seconds=$((older_than_days * 86400))

	echo "Looking for stale worktrees (older than ${older_than_days} days)..."
	echo

	metadata_files="$(wt_metadata_list)"

	if [[ -z "$metadata_files" ]]; then
		echo "No agent worktrees found"
		return 0
	fi

	while IFS= read -r metadata_file; do
		agent="$(basename "$metadata_file" .json)"
		age="$(wt_metadata_age "$agent")"
		age_days=$((age / 86400))

		if [[ "$age" -gt "$older_than_seconds" ]]; then
			stale_count=$((stale_count + 1))
			echo "Stale: $agent (${age_days} days old)"

			if [[ "$dry_run" == false ]]; then
				echo -n "  Remove? [y/N] "
				read -r response
				if [[ "$response" =~ ^[Yy]$ ]]; then
					wt_agent_remove "$agent"
					echo
				fi
			fi
		fi
	done <<< "$metadata_files"

	if [[ "$stale_count" -eq 0 ]]; then
		echo "No stale worktrees found"
	elif [[ "$dry_run" == true ]]; then
		echo
		echo "Found $stale_count stale worktree(s)"
		echo "Run without --dry-run to remove them"
	fi
}

# Display agent status dashboard
wt_agent_status() {
	local total_count=0
	local active_count=0
	local metadata_files
	local agent
	local worktree_path

	echo "=== Worktree Status Dashboard ==="
	echo

	metadata_files="$(wt_metadata_list)"

	if [[ -n "$metadata_files" ]]; then
		total_count=$(echo "$metadata_files" | wc -l | tr -d ' ')

		while IFS= read -r metadata_file; do
			agent="$(basename "$metadata_file" .json)"
			worktree_path="$WT_TARGET_PATH/$agent"

			if [[ -d "$worktree_path" ]]; then
				active_count=$((active_count + 1))
			fi
		done <<< "$metadata_files"
	fi

	echo "Total worktrees: $total_count"
	echo "Active worktrees: $active_count"
	echo

	if [[ "$total_count" -gt 0 ]]; then
		wt_agent_list
	fi

	echo
	echo "Git worktree list:"
	git -C "$WT_TARGET_PATH" worktree list
}
