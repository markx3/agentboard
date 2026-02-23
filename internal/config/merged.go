package config

// Merged holds the final configuration with precedence: project > global > defaults.
type Merged struct {
	Username       string
	AgentDefault   string
	CopyFiles      []string
	InitScript     string
	ProjectName    string
	ThemeBorder    string
	ThemeText      string
	ThemeAccent    string
}

func Load() (*Merged, error) {
	global, err := LoadGlobal()
	if err != nil {
		return nil, err
	}

	project, err := LoadProject()
	if err != nil {
		return nil, err
	}

	agent := global.Agent.Default
	if project.Agent.Preferred != "" {
		agent = project.Agent.Preferred
	}

	return &Merged{
		Username:     global.User.GitHubUsername,
		AgentDefault: agent,
		CopyFiles:    project.Worktree.CopyFiles,
		InitScript:   project.Worktree.InitScript,
		ProjectName:  project.Project.Name,
		ThemeBorder:  global.Theme.Border,
		ThemeText:    global.Theme.Text,
		ThemeAccent:  global.Theme.Accent,
	}, nil
}
