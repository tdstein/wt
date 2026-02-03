package git

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Command represents a git command builder
type Command struct {
	dir  string
	args []string
}

// CloneOptions specifies options for cloning a repository
type CloneOptions struct {
	Bare         bool // Create a bare repository
	Partial      bool // Use partial clone (--filter=blob:none)
	ShallowDepth int  // Clone depth (0 = full history)
	SingleBranch bool // Clone only the default branch (faster for repos with many branches)
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

// RunWithProgress executes the git command, streaming stderr to os.Stderr for progress feedback
func (c *Command) RunWithProgress() error {
	cmd := exec.Command("git", c.args...)
	if c.dir != "" {
		cmd.Dir = c.dir
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Always stream stderr to user (git progress goes here)
	if stderr.Len() > 0 {
		fmt.Fprint(os.Stderr, stderr.String())
	}

	if err != nil {
		return fmt.Errorf("git %s failed: %w\nstderr: %s",
			strings.Join(c.args, " "), err, stderr.String())
	}

	return nil
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

// Clone runs git clone with progress output
func Clone(url, dir string, bare bool) error {
	args := []string{"clone"}
	if bare {
		args = append(args, "--bare")
	}
	args = append(args, url, dir)

	return New(args...).RunWithProgress()
}

// CloneWithOptions runs git clone with additional options
func CloneWithOptions(url, dir string, opts CloneOptions) error {
	args := []string{"clone"}

	// Add optimization flags
	if opts.Partial {
		args = append(args, "--filter=blob:none")
	}
	if opts.ShallowDepth > 0 {
		args = append(args, fmt.Sprintf("--depth=%d", opts.ShallowDepth))
	}
	if opts.SingleBranch {
		args = append(args, "--single-branch")
	}

	// Add bare flag
	if opts.Bare {
		args = append(args, "--bare")
	}

	args = append(args, url, dir)

	return New(args...).RunWithProgress()
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

// SetUpstream sets the upstream branch for the current branch
func SetUpstream(dir, branch, remote, remoteBranch string) error {
	return New("branch", "--set-upstream-to="+remote+"/"+remoteBranch, branch).WithDir(dir).RunSilent()
}

// Fetch fetches from a remote with progress output
func Fetch(dir, remote string) error {
	return New("fetch", remote).WithDir(dir).RunWithProgress()
}

// FetchBranch fetches a single branch from a remote
func FetchBranch(dir, remote, branch string) error {
	// Fetch the specific branch into refs/remotes/origin/<branch>
	refspec := fmt.Sprintf("+refs/heads/%s:refs/remotes/%s/%s", branch, remote, branch)
	return New("fetch", remote, refspec).WithDir(dir).RunWithProgress()
}
