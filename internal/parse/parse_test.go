package parse

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"https url", "https://github.com/user/repo.git", true},
		{"http url", "http://example.com/repo", true},
		{"git protocol", "git://host/repo.git", true},
		{"ssh git url", "git@github.com:user/repo.git", true},
		{"ssh with port", "git@example.com:2222/repo.git", true},
		{"simple name", "my-project", false},
		{"relative path", "./relative/path", false},
		{"absolute path", "/absolute/path", false},
		{"name with hyphen", "my-local-dir", false},
		{"name with underscore", "my_local_dir", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsURL(tt.input)
			if got != tt.want {
				t.Errorf("IsURL(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestURLToDirname(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"https with .git", "https://github.com/user/repo.git", "repo"},
		{"https without .git", "https://github.com/user/repo", "repo"},
		{"ssh git url", "git@github.com:user/repo.git", "repo"},
		{"ssh without .git", "git@github.com:user/repo", "repo"},
		{"git protocol", "git://host/path/to/repo", "repo"},
		{"nested path", "https://github.com/user/deeply/nested/repo.git", "repo"},
		{"hyphenated name", "https://github.com/user/my-project.git", "my-project"},
		{"underscored name", "git@github.com:user/my_project.git", "my_project"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := URLToDirname(tt.input)
			if got != tt.want {
				t.Errorf("URLToDirname(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestParseArgs(t *testing.T) {
	// Save and restore HOME
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)
	os.Setenv("HOME", "/home/testuser")

	tests := []struct {
		name      string
		args      []string
		wantMode  string
		wantURL   string
		wantDir   string
		wantPath  string
		wantError bool
	}{
		{
			name:      "remote url with auto dir",
			args:      []string{"https://github.com/user/repo.git"},
			wantMode:  "remote",
			wantURL:   "https://github.com/user/repo.git",
			wantDir:   "repo",
			wantPath:  "/home/testuser/wt/repo",
			wantError: false,
		},
		{
			name:      "remote url with custom dir",
			args:      []string{"https://github.com/user/repo.git", "mydir"},
			wantMode:  "remote",
			wantURL:   "https://github.com/user/repo.git",
			wantDir:   "mydir",
			wantPath:  "/home/testuser/wt/mydir",
			wantError: false,
		},
		{
			name:      "ssh url",
			args:      []string{"git@github.com:user/repo.git"},
			wantMode:  "remote",
			wantURL:   "git@github.com:user/repo.git",
			wantDir:   "repo",
			wantPath:  "/home/testuser/wt/repo",
			wantError: false,
		},
		{
			name:      "local project",
			args:      []string{"my-project"},
			wantMode:  "local",
			wantURL:   "",
			wantDir:   "my-project",
			wantPath:  "/home/testuser/wt/my-project",
			wantError: false,
		},
		{
			name:      "no arguments",
			args:      []string{},
			wantError: true,
		},
		{
			name:      "too many arguments",
			args:      []string{"arg1", "arg2", "arg3"},
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseArgs(tt.args)

			if tt.wantError {
				if err == nil {
					t.Errorf("ParseArgs(%v) expected error, got nil", tt.args)
				}
				return
			}

			if err != nil {
				t.Errorf("ParseArgs(%v) unexpected error: %v", tt.args, err)
				return
			}

			if got.Mode != tt.wantMode {
				t.Errorf("Mode = %q, want %q", got.Mode, tt.wantMode)
			}
			if got.RepoURL != tt.wantURL {
				t.Errorf("RepoURL = %q, want %q", got.RepoURL, tt.wantURL)
			}
			if got.DirName != tt.wantDir {
				t.Errorf("DirName = %q, want %q", got.DirName, tt.wantDir)
			}
			if got.TargetPath != tt.wantPath {
				t.Errorf("TargetPath = %q, want %q", got.TargetPath, tt.wantPath)
			}
		})
	}
}

func TestParseArgsRealHome(t *testing.T) {
	// Test with real HOME directory
	result, err := ParseArgs([]string{"test-project"})
	if err != nil {
		t.Fatalf("ParseArgs failed: %v", err)
	}

	homeDir, _ := os.UserHomeDir()
	expectedPath := filepath.Join(homeDir, "wt", "test-project")

	if result.TargetPath != expectedPath {
		t.Errorf("TargetPath = %q, want %q", result.TargetPath, expectedPath)
	}
}
