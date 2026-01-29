package agent

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Info contains information about an agent worktree
type Info struct {
	Agent    string
	Branch   string
	Age      int64 // Age in seconds
	AgeHuman string
	Status   string
	Exists   bool
}

// List lists all agent worktrees
func (m *Manager) List() ([]Info, error) {
	metadataFiles, err := m.metadata.List()
	if err != nil {
		return nil, err
	}

	if len(metadataFiles) == 0 {
		return []Info{}, nil
	}

	var agents []Info
	for _, metadataFile := range metadataFiles {
		agent := strings.TrimSuffix(filepath.Base(metadataFile), ".json")

		metadata, err := m.metadata.Get(agent)
		if err != nil {
			continue
		}

		age, _ := m.metadata.Age(agent)
		ageHuman := AgeHuman(age)

		worktreePath := filepath.Join(m.targetPath, agent)
		exists := false
		if _, err := os.Stat(worktreePath); err == nil {
			exists = true
		}

		status := metadata.Status
		if !exists {
			status = "missing"
		}

		agents = append(agents, Info{
			Agent:    agent,
			Branch:   metadata.Branch,
			Age:      age,
			AgeHuman: ageHuman,
			Status:   status,
			Exists:   exists,
		})
	}

	return agents, nil
}

// AgeHuman returns the age formatted in human-readable form
func AgeHuman(ageSeconds int64) string {
	const (
		minute = 60
		hour   = 60 * minute
		day    = 24 * hour
	)

	switch {
	case ageSeconds < minute:
		return fmt.Sprintf("%ds", ageSeconds)
	case ageSeconds < hour:
		minutes := ageSeconds / minute
		return fmt.Sprintf("%dm", minutes)
	case ageSeconds < day:
		hours := ageSeconds / hour
		return fmt.Sprintf("%dh", hours)
	default:
		days := ageSeconds / day
		return fmt.Sprintf("%dd", days)
	}
}
