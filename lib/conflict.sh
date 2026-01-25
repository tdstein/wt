#!/usr/bin/env bash
# Conflict detection and branch synchronization for wt

# Check if a branch can merge cleanly into another branch
# Args: base_branch, feature_branch
# Returns: 0 if clean merge, 1 if conflicts
wt_conflict_check() {
	local base_branch="$1"
	local feature_branch="$2"
	local merge_result

	# Use git merge-tree to simulate merge without touching working directory
	merge_result="$(git -C "$WT_TARGET_PATH" merge-tree \
		"$(git -C "$WT_TARGET_PATH" merge-base "$base_branch" "$feature_branch")" \
		"$base_branch" \
		"$feature_branch" 2>&1)"

	# Check for conflict markers in merge-tree output
	if echo "$merge_result" | grep -q "^+<<<<<<< "; then
		return 1
	fi

	return 0
}

# Get list of conflicting files between branches
# Args: base_branch, feature_branch
wt_conflict_files() {
	local base_branch="$1"
	local feature_branch="$2"
	local merge_base
	local conflicts

	merge_base="$(git -C "$WT_TARGET_PATH" merge-base "$base_branch" "$feature_branch" 2>/dev/null)"

	if [[ -z "$merge_base" ]]; then
		return 1
	fi

	# Get files that differ between branches
	conflicts="$(git -C "$WT_TARGET_PATH" merge-tree "$merge_base" "$base_branch" "$feature_branch" 2>/dev/null | \
		grep -E "^\+\+\+|^\-\-\-" | \
		sed 's/^[+\-]\+[+\-]\+ [ab]\///' | \
		sort -u)"

	if [[ -n "$conflicts" ]]; then
		echo "$conflicts"
		return 0
	fi

	return 1
}

# Get divergence information (commits ahead/behind)
# Args: base_branch, feature_branch
# Outputs: "ahead behind" as two numbers
wt_conflict_divergence() {
	local base_branch="$1"
	local feature_branch="$2"
	local ahead behind

	ahead="$(git -C "$WT_TARGET_PATH" rev-list --count "${base_branch}..${feature_branch}" 2>/dev/null || echo 0)"
	behind="$(git -C "$WT_TARGET_PATH" rev-list --count "${feature_branch}..${base_branch}" 2>/dev/null || echo 0)"

	echo "$ahead $behind"
}

# Check if a worktree has uncommitted changes
# Args: worktree_path
wt_conflict_has_changes() {
	local worktree_path="$1"

	# Check for staged or unstaged changes
	if ! git -C "$worktree_path" diff-index --quiet HEAD -- 2>/dev/null; then
		return 0
	fi

	# Check for untracked files
	if [[ -n "$(git -C "$worktree_path" ls-files --others --exclude-standard 2>/dev/null)" ]]; then
		return 0
	fi

	return 1
}

# Sync a worktree with its base branch
# Args: agent_name, base_branch, [--auto-rebase]
wt_conflict_sync() {
	local agent="$1"
	local base_branch="$2"
	local auto_rebase="${3:-}"
	local worktree_path
	local current_branch
	local divergence
	local ahead behind

	worktree_path="$WT_TARGET_PATH/$agent"

	if [[ ! -d "$worktree_path" ]]; then
		return 1
	fi

	# Get current branch
	current_branch="$(git -C "$worktree_path" rev-parse --abbrev-ref HEAD 2>/dev/null)"

	if [[ -z "$current_branch" ]]; then
		return 1
	fi

	# Fetch latest from base
	git -C "$WT_TARGET_PATH" fetch origin "$base_branch" 2>/dev/null || true

	# Get divergence
	divergence="$(wt_conflict_divergence "$base_branch" "$current_branch")"
	read -r ahead behind <<< "$divergence"

	if [[ "$behind" -eq 0 ]]; then
		echo "Already up to date with $base_branch"
		return 0
	fi

	echo "Branch is $ahead commits ahead, $behind commits behind $base_branch"

	# Check for uncommitted changes
	if wt_conflict_has_changes "$worktree_path"; then
		echo "Error: Uncommitted changes detected. Commit or stash before syncing."
		return 1
	fi

	# Auto-rebase if requested
	if [[ "$auto_rebase" == "--auto-rebase" ]]; then
		echo "Rebasing onto $base_branch..."
		if git -C "$worktree_path" rebase "$base_branch" 2>&1; then
			echo "Rebase successful"
			return 0
		else
			echo "Error: Rebase failed. Resolve conflicts manually."
			git -C "$worktree_path" rebase --abort 2>/dev/null || true
			return 1
		fi
	else
		echo "Run with --auto-rebase to rebase onto $base_branch"
		return 0
	fi
}

# Pretty print merge check results
wt_conflict_check_report() {
	local agent="$1"
	local base_branch="$2"
	local feature_branch
	local worktree_path

	worktree_path="$WT_TARGET_PATH/$agent"
	feature_branch="$(git -C "$worktree_path" rev-parse --abbrev-ref HEAD 2>/dev/null)"

	if [[ -z "$feature_branch" ]]; then
		echo "Error: Could not determine branch for $agent"
		return 1
	fi

	echo "Checking merge: $feature_branch -> $base_branch"
	echo

	# Check for uncommitted changes
	if wt_conflict_has_changes "$worktree_path"; then
		echo "⚠️  Warning: Uncommitted changes detected"
		echo
	fi

	# Check divergence
	local divergence ahead behind
	divergence="$(wt_conflict_divergence "$base_branch" "$feature_branch")"
	read -r ahead behind <<< "$divergence"

	echo "Divergence: $ahead commits ahead, $behind commits behind"

	if [[ "$behind" -gt 0 ]]; then
		echo "⚠️  Branch is behind $base_branch. Consider rebasing."
	fi

	echo

	# Check for conflicts
	if wt_conflict_check "$base_branch" "$feature_branch"; then
		echo "✓ Clean merge - no conflicts detected"
		return 0
	else
		echo "✗ Conflicts detected"
		echo
		echo "Conflicting files:"
		wt_conflict_files "$base_branch" "$feature_branch" | sed 's/^/  - /'
		return 1
	fi
}
