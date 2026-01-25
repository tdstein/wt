#!/usr/bin/env bash
# Repository operations for wt

# Source parse functions if not already loaded
[[ -z "${WT_LIB_LOADED:-}" ]] && source "$(dirname "${BASH_SOURCE[0]}")/parse.sh"

WT_LIB_LOADED=1

# Check if target directory exists
# Returns 0 if exists, 1 if not
wt_target_exists() {
    [[ -d "$WT_TARGET_PATH" ]]
}

# Remove target directory
wt_remove_target() {
    rm -rf "$WT_TARGET_PATH"
}

# Create target directory
wt_create_target() {
    mkdir -p "$WT_TARGET_PATH"
}

# Initialize bare repository for local project
wt_init_local_bare() {
    git init --bare "$WT_TARGET_PATH/.bare" >/dev/null 2>&1 || return 1
    git -C "$WT_TARGET_PATH/.bare" symbolic-ref HEAD refs/heads/main
}

# Clone bare repository from remote
wt_clone_remote_bare() {
    git clone --bare "$WT_REPO_URL" "$WT_TARGET_PATH/.bare" || return 1
    git -C "$WT_TARGET_PATH/.bare" config remote.origin.fetch "+refs/heads/*:refs/remotes/origin/*"
}

# Create .git pointer file
wt_create_git_pointer() {
    echo "gitdir: ./.bare" > "$WT_TARGET_PATH/.git"
}

# Get default branch for remote repo
wt_get_remote_default_branch() {
    git -C "$WT_TARGET_PATH/.bare" remote show origin 2>/dev/null | \
        grep "HEAD branch" | \
        awk '{print $NF}'
}

# Create primary worktree for local repo
wt_create_local_worktree() {
    git -C "$WT_TARGET_PATH" worktree add -b main main >/dev/null 2>&1 || return 1
    git -C "$WT_TARGET_PATH/main" commit --allow-empty -m "Initial commit" >/dev/null 2>&1
}

# Create primary worktree for remote repo
wt_create_remote_worktree() {
    local branch="$1"
    git -C "$WT_TARGET_PATH" worktree add main "$branch" >/dev/null 2>&1
}

# List worktrees
wt_list_worktrees() {
    git -C "$WT_TARGET_PATH" worktree list
}

# Get sizes of .bare and main directories
wt_get_sizes() {
    du -sh "$WT_TARGET_PATH/.bare" "$WT_TARGET_PATH/main" 2>/dev/null
}

# Full setup for local project
wt_setup_local() {
    wt_create_target || return 1
    wt_init_local_bare || return 1
    wt_create_git_pointer || return 1
    wt_create_local_worktree || return 1
    return 0
}

# Full setup for remote project
wt_setup_remote() {
    local default_branch

    wt_create_target || return 1
    wt_clone_remote_bare || return 1
    wt_create_git_pointer || return 1

    default_branch="$(wt_get_remote_default_branch)"
    [[ -z "$default_branch" ]] && default_branch="main"

    wt_create_remote_worktree "$default_branch" || return 1
    return 0
}
