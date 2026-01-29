package prune

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/tdstein/wt/internal/agent"
)

// Options contains options for pruning stale worktrees
type Options struct {
	OlderThanDays int
	DryRun        bool
	Interactive   bool
}

// Result contains the result of a prune operation
type Result struct {
	StaleAgents []string
	Removed     []string
	Errors      map[string]error
}

// Pruner handles pruning stale agent worktrees
type Pruner struct {
	manager *agent.Manager
}

// NewPruner creates a new pruner
func NewPruner(targetPath string) *Pruner {
	return &Pruner{
		manager: agent.NewManager(targetPath),
	}
}

// Prune removes stale agent worktrees
func (p *Pruner) Prune(opts Options) (*Result, error) {
	if opts.OlderThanDays == 0 {
		opts.OlderThanDays = 7
	}

	olderThanSeconds := int64(opts.OlderThanDays * 86400)

	result := &Result{
		StaleAgents: []string{},
		Removed:     []string{},
		Errors:      make(map[string]error),
	}

	agents, err := p.manager.List()
	if err != nil {
		return result, err
	}

	for _, agentInfo := range agents {
		if agentInfo.Age > olderThanSeconds {
			result.StaleAgents = append(result.StaleAgents, agentInfo.Agent)

			if opts.DryRun {
				continue
			}

			shouldRemove := true
			if opts.Interactive {
				shouldRemove = promptYesNo(fmt.Sprintf("Remove %s (%s old)?", agentInfo.Agent, agentInfo.AgeHuman))
			}

			if shouldRemove {
				err := p.manager.Remove(agent.RemoveOptions{AgentName: agentInfo.Agent})
				if err != nil {
					result.Errors[agentInfo.Agent] = err
				} else {
					result.Removed = append(result.Removed, agentInfo.Agent)
				}
			}
		}
	}

	return result, nil
}

// Helper function to prompt for yes/no
func promptYesNo(prompt string) bool {
	fmt.Printf("%s [y/N] ", prompt)
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

// ParseOlderThan parses an --older-than argument (e.g., "7d", "14d")
func ParseOlderThan(arg string) (int, error) {
	arg = strings.TrimSuffix(arg, "d")
	days, err := strconv.Atoi(arg)
	if err != nil {
		return 0, fmt.Errorf("invalid --older-than value: %s", arg)
	}
	if days < 0 {
		return 0, fmt.Errorf("--older-than must be positive")
	}
	return days, nil
}
