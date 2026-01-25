package git

import (
	"bytes"
	"fmt"
	"os/exec"
	"strings"
)

// Command represents a git command builder
type Command struct {
	dir  string
	args []string
}

// New creates a new git command builder
func New(args ...string) *Command {
	return &Command{args: args}
}

// WithDir sets the working directory for the command
func (c *Command) WithDir(dir string) *Command {
	c.dir = dir
	return c
}

// Run executes the git command and returns the output
func (c *Command) Run() (string, error) {
	cmd := exec.Command("git", c.args...)
	if c.dir != "" {
		cmd.Dir = c.dir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("git %s failed: %w\nstderr: %s",
			strings.Join(c.args, " "), err, stderr.String())
	}

	return strings.TrimSpace(stdout.String()), nil
}

// RunSilent executes the git command, discarding output
func (c *Command) RunSilent() error {
	_, err := c.Run()
	return err
}

// Init runs git init --bare
func Init(dir string, bare bool) error {
	args := []string{"init"}
	if bare {
		args = append(args, "--bare")
	}
	args = append(args, dir)

	return New(args...).RunSilent()
}

// Clone runs git clone
func Clone(url, dir string, bare bool) error {
	args := []string{"clone"}
	if bare {
		args = append(args, "--bare")
	}
	args = append(args, url, dir)

	return New(args...).RunSilent()
}

// SymbolicRef sets a symbolic reference
func SymbolicRef(dir, name, ref string) error {
	return New("symbolic-ref", name, ref).WithDir(dir).RunSilent()
}

// Config sets a git configuration value
func Config(dir, key, value string) error {
	return New("config", key, value).WithDir(dir).RunSilent()
}

// RemoteShow gets information about a remote
func RemoteShow(dir, remote string) (string, error) {
	return New("remote", "show", remote).WithDir(dir).Run()
}

// WorktreeAdd creates a new worktree
func WorktreeAdd(dir, path, branch string, newBranch bool) error {
	args := []string{"worktree", "add"}
	if newBranch {
		args = append(args, "-b", branch, path)
	} else {
		args = append(args, path, branch)
	}

	return New(args...).WithDir(dir).RunSilent()
}

// WorktreeList lists all worktrees
func WorktreeList(dir string) (string, error) {
	return New("worktree", "list").WithDir(dir).Run()
}

// Commit creates a commit
func Commit(dir, message string, allowEmpty bool) error {
	args := []string{"commit", "-m", message}
	if allowEmpty {
		args = append(args, "--allow-empty")
	}

	return New(args...).WithDir(dir).RunSilent()
}
