#!/usr/bin/env bash
# Argument parsing functions for wt

# Check if argument looks like a URL
# Returns 0 (true) if it's a URL, 1 (false) otherwise
wt_is_url() {
    local arg="$1"
    [[ "$arg" == *"://"* ]] || [[ "$arg" == git@* ]]
}

# Extract directory name from URL
# Examples:
#   https://github.com/user/repo.git -> repo
#   git@github.com:user/repo.git -> repo
#   https://github.com/user/repo -> repo
wt_url_to_dirname() {
    local url="$1"
    local name

    # Get the last path segment
    name="${url##*/}"

    # Strip .git suffix if present
    name="${name%.git}"

    echo "$name"
}

# Parse arguments and set variables
# Sets: WT_MODE, WT_REPO_URL, WT_DIR_NAME, WT_TARGET_PATH
# Returns 0 on success, 1 on error
wt_parse_args() {
    local arg1="${1:-}"
    local arg2="${2:-}"

    if [[ -z "$arg1" ]]; then
        echo "Error: No arguments provided" >&2
        return 1
    fi

    if wt_is_url "$arg1"; then
        WT_MODE="remote"
        WT_REPO_URL="$arg1"
        if [[ -n "$arg2" ]]; then
            WT_DIR_NAME="$arg2"
        else
            WT_DIR_NAME="$(wt_url_to_dirname "$arg1")"
        fi
    else
        WT_MODE="local"
        WT_REPO_URL=""
        WT_DIR_NAME="$arg1"
    fi

    WT_TARGET_PATH="$HOME/wt/$WT_DIR_NAME"

    return 0
}
