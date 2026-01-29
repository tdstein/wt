package agent

// Status returns dashboard information about all agents
type Status struct {
	TotalCount  int
	ActiveCount int
	Agents      []Info
}

// GetStatus returns status dashboard information
func (m *Manager) GetStatus() (*Status, error) {
	agents, err := m.List()
	if err != nil {
		return nil, err
	}

	activeCount := 0
	for _, agent := range agents {
		if agent.Exists {
			activeCount++
		}
	}

	return &Status{
		TotalCount:  len(agents),
		ActiveCount: activeCount,
		Agents:      agents,
	}, nil
}
