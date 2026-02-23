package agent

import (
	"os/exec"
)

type Agent struct {
	Name    string
	Command string
	Args    []string
}

var knownAgents = []Agent{
	{Name: "claude", Command: "claude", Args: []string{}},
	{Name: "cursor", Command: "cursor", Args: []string{"--cli"}},
	{Name: "aider", Command: "aider", Args: []string{}},
	{Name: "codex", Command: "codex", Args: []string{}},
}

func Available() []Agent {
	var available []Agent
	for _, a := range knownAgents {
		if _, err := exec.LookPath(a.Command); err == nil {
			available = append(available, a)
		}
	}
	return available
}

func Find(name string) *Agent {
	for _, a := range knownAgents {
		if a.Name == name {
			if _, err := exec.LookPath(a.Command); err == nil {
				return &a
			}
			return nil
		}
	}
	return nil
}

func Default() *Agent {
	agents := Available()
	if len(agents) > 0 {
		return &agents[0]
	}
	return nil
}
