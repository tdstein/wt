package hooks

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed templates/*
var templatesFS embed.FS

// Install installs Claude Code hooks to the specified directory
func Install(targetDir string) error {
	// Validate target directory
	if targetDir == "" {
		return fmt.Errorf("target directory cannot be empty")
	}

	// Create .claude directory structure
	claudeDir := filepath.Join(targetDir, ".claude")
	hooksDir := filepath.Join(claudeDir, "hooks")

	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude/hooks directory: %w", err)
	}

	// Walk through embedded templates and write them to target
	err := fs.WalkDir(templatesFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		// Read embedded file
		content, err := templatesFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}

		// Determine target path
		relPath, err := filepath.Rel("templates", path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(claudeDir, relPath)

		// Check if file exists
		if _, err := os.Stat(targetPath); err == nil {
			return fmt.Errorf("file already exists: %s (use --force to overwrite)", targetPath)
		}

		// Write file
		if err := os.WriteFile(targetPath, content, 0755); err != nil {
			return fmt.Errorf("failed to write file %s: %w", targetPath, err)
		}

		fmt.Printf("Installed: %s\n", targetPath)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to install hooks: %w", err)
	}

	fmt.Printf("\n✓ Claude Code hooks installed successfully\n")
	fmt.Printf("  Configuration: %s\n", filepath.Join(claudeDir, "settings.json"))
	fmt.Printf("  Hook scripts: %s\n", hooksDir)
	return nil
}

// InstallWithForce installs hooks, overwriting existing files
func InstallWithForce(targetDir string) error {
	claudeDir := filepath.Join(targetDir, ".claude")
	hooksDir := filepath.Join(claudeDir, "hooks")

	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("failed to create .claude/hooks directory: %w", err)
	}

	err := fs.WalkDir(templatesFS, "templates", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}

		content, err := templatesFS.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read embedded file %s: %w", path, err)
		}

		relPath, err := filepath.Rel("templates", path)
		if err != nil {
			return err
		}

		targetPath := filepath.Join(claudeDir, relPath)

		if err := os.WriteFile(targetPath, content, 0755); err != nil {
			return fmt.Errorf("failed to write file %s: %w", targetPath, err)
		}

		fmt.Printf("Installed: %s\n", targetPath)
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to install hooks: %w", err)
	}

	fmt.Printf("\n✓ Claude Code hooks installed successfully\n")
	fmt.Printf("  Configuration: %s\n", filepath.Join(claudeDir, "settings.json"))
	fmt.Printf("  Hook scripts: %s\n", hooksDir)
	return nil
}

// PrintConfig outputs the settings.json configuration to stdout
func PrintConfig() error {
	content, err := templatesFS.ReadFile("templates/settings.json")
	if err != nil {
		return fmt.Errorf("failed to read settings.json: %w", err)
	}

	fmt.Print(string(content))
	return nil
}

// PrintScript outputs a specific hook script to stdout
func PrintScript(scriptName string) error {
	scriptPath := filepath.Join("templates/hooks", scriptName)
	content, err := templatesFS.ReadFile(scriptPath)
	if err != nil {
		return fmt.Errorf("failed to read script %s: %w", scriptName, err)
	}

	fmt.Print(string(content))
	return nil
}

// ListScripts lists all available hook scripts
func ListScripts() ([]string, error) {
	var scripts []string

	entries, err := templatesFS.ReadDir("templates/hooks")
	if err != nil {
		return nil, fmt.Errorf("failed to read hooks directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() && filepath.Ext(entry.Name()) == ".sh" {
			scripts = append(scripts, entry.Name())
		}
	}

	return scripts, nil
}
