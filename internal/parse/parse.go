package parse

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ParseResult contains the parsed arguments
type ParseResult struct {
	Mode       string // "remote" or "local"
	RepoURL    string // URL if remote, empty if local
	DirName    string // Directory name
	TargetPath string // Full path: $HOME/wt/$DirName
}

// IsURL checks if the argument looks like a URL
// Supports both protocol URLs (http://, https://, git://)
// and SSH Git URLs (git@host:path)
func IsURL(arg string) bool {
	// Check for protocol URLs (http://, https://, git://, etc.)
	if strings.Contains(arg, "://") {
		return true
	}
	// Check for SSH Git URLs (git@github.com:...)
	if strings.HasPrefix(arg, "git@") {
		return true
	}
	return false
}

// URLToDirname extracts the directory name from a URL
// Examples:
//
//	https://github.com/user/repo.git -> repo
//	git@github.com:user/repo.git -> repo
//	https://github.com/user/repo -> repo
func URLToDirname(url string) string {
	var name string

	// For SSH URLs like git@github.com:user/repo.git
	// Split on : and take the part after the last colon
	if strings.Contains(url, ":") && strings.HasPrefix(url, "git@") {
		parts := strings.Split(url, ":")
		if len(parts) > 1 {
			url = parts[len(parts)-1] // Take part after last :
		}
	}

	// Get basename (last segment after /)
	name = filepath.Base(url)

	// Strip .git suffix
	name = strings.TrimSuffix(name, ".git")

	return name
}

// ParseArgs parses command line arguments and returns a ParseResult
// Accepts 1-2 arguments:
//   - If first arg is a URL: remote mode, optional second arg is custom directory name
//   - If first arg is not a URL: local mode, first arg is directory name
func ParseArgs(args []string) (*ParseResult, error) {
	if len(args) == 0 {
		return nil, fmt.Errorf("no arguments provided")
	}

	if len(args) > 2 {
		return nil, fmt.Errorf("too many arguments (expected 1-2, got %d)", len(args))
	}

	arg1 := args[0]
	var arg2 string
	if len(args) > 1 {
		arg2 = args[1]
	}

	result := &ParseResult{}

	if IsURL(arg1) {
		result.Mode = "remote"
		result.RepoURL = arg1

		if arg2 != "" {
			result.DirName = arg2
		} else {
			result.DirName = URLToDirname(arg1)
		}
	} else {
		result.Mode = "local"
		result.RepoURL = ""
		result.DirName = arg1
	}

	// Get HOME directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	result.TargetPath = filepath.Join(homeDir, "wt", result.DirName)

	return result, nil
}
