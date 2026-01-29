package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/tdstein/wt/internal/worktree"
)

// Context keys
type contextKey string

const (
	targetPathKey contextKey = "targetPath"
	taskIDKey     contextKey = "taskID"
)

// Colors for terminal output
const (
	colorReset  = "\033[0m"
	colorRed    = "\033[0;31m"
	colorGreen  = "\033[0;32m"
	colorYellow = "\033[0;33m"
	colorBlue   = "\033[0;34m"
)

// Logging functions
func logInfo(format string, args ...interface{}) {
	if isColorEnabled() {
		fmt.Printf("%s[INFO]%s %s\n", colorBlue, colorReset, fmt.Sprintf(format, args...))
	} else {
		fmt.Printf("[INFO] %s\n", fmt.Sprintf(format, args...))
	}
}

func logSuccess(format string, args ...interface{}) {
	if isColorEnabled() {
		fmt.Printf("%s[OK]%s %s\n", colorGreen, colorReset, fmt.Sprintf(format, args...))
	} else {
		fmt.Printf("[OK] %s\n", fmt.Sprintf(format, args...))
	}
}

func logWarn(format string, args ...interface{}) {
	if isColorEnabled() {
		fmt.Printf("%s[WARN]%s %s\n", colorYellow, colorReset, fmt.Sprintf(format, args...))
	} else {
		fmt.Printf("[WARN] %s\n", fmt.Sprintf(format, args...))
	}
}

func logError(format string, args ...interface{}) {
	if isColorEnabled() {
		fmt.Fprintf(os.Stderr, "%s[ERROR]%s %s\n", colorRed, colorReset, fmt.Sprintf(format, args...))
	} else {
		fmt.Fprintf(os.Stderr, "[ERROR] %s\n", fmt.Sprintf(format, args...))
	}
}

// isColorEnabled checks if terminal supports colors
func isColorEnabled() bool {
	// Check if stdout is a terminal
	fileInfo, err := os.Stdout.Stat()
	if err != nil {
		return false
	}

	// Check if it's a character device (terminal)
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// isInteractive checks if stdin is a terminal
func isInteractive() bool {
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// confirmRemove prompts user for confirmation to remove a directory
func confirmRemove(path string) bool {
	if isColorEnabled() {
		fmt.Printf("%sDirectory exists: %s%s\n", colorYellow, path, colorReset)
	} else {
		fmt.Printf("Directory exists: %s\n", path)
	}

	fmt.Print("Remove and recreate? [y/N] ")
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// findWtRoot searches for .bare directory in current or parent directories
func findWtRoot() (string, error) {
	return worktree.FindRoot()
}

// Context helpers
func withTargetPath(ctx context.Context, targetPath string) context.Context {
	return context.WithValue(ctx, targetPathKey, targetPath)
}

func getTargetPath(ctx context.Context) string {
	if targetPath, ok := ctx.Value(targetPathKey).(string); ok {
		return targetPath
	}
	return ""
}

// Context helpers for task ID
func withTaskID(ctx context.Context, taskID string) context.Context {
	return context.WithValue(ctx, taskIDKey, taskID)
}

func getTaskID(ctx context.Context) string {
	if taskID, ok := ctx.Value(taskIDKey).(string); ok {
		return taskID
	}
	return ""
}

// repeatString repeats a string n times
func repeatString(s string, n int) string {
	var result strings.Builder
	for i := 0; i < n; i++ {
		result.WriteString(s)
	}
	return result.String()
}
