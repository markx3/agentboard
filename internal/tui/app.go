package tui

import (
	"context"
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/marcosfelipeeipper/agentboard/internal/agent"
	"github.com/marcosfelipeeipper/agentboard/internal/board"
	"github.com/marcosfelipeeipper/agentboard/internal/db"
	"github.com/marcosfelipeeipper/agentboard/internal/tmux"
)

const (
	agentPollInterval = 2500 * time.Millisecond
	gracePeriod       = 5 * time.Second
)

type overlayType int

const (
	overlayNone overlayType = iota
	overlayForm
	overlayDetail
	overlayHelp
	overlayConfirm
	overlayPicker
)

// pendingRecon tracks a task whose agent window just died.
type pendingRecon struct {
	detectedAt        time.Time
	columnAtDetection db.TaskStatus
}

// pendingFocus tracks where to move the cursor after tasks reload.
type pendingFocus struct {
	taskID    string
	newStatus db.TaskStatus
}

type App struct {
	board        kanban
	service      board.Service
	overlay      overlayType
	form         taskForm
	detail       taskDetail
	picker       agentPicker
	notification *notification
	width        int
	height       int
	ready        bool
	// pendingSpawnTask holds a task awaiting skip-permissions confirmation.
	pendingSpawnTask *db.Task
	// availableRunners is cached at startup for agent detection.
	availableRunners []agent.AgentRunner
	// pendingRecons tracks tasks in the grace period after their agent window dies.
	pendingRecons map[string]pendingRecon
	// lastTasks caches the latest task list for reconciliation.
	lastTasks []db.Task
	// cursorFollow stores where to move the cursor after the next task reload.
	cursorFollow *pendingFocus
	// Server/tunnel status
	tunnelURL    string
	peerCount    int
	serverActive bool
}

// AppOption configures optional App behavior.
type AppOption func(*App)

// WithConnectAddr sets the remote server address for status bar display.
func WithConnectAddr(addr string) AppOption {
	return func(a *App) {
		a.tunnelURL = addr
		a.serverActive = true
	}
}

func NewApp(svc board.Service, opts ...AppOption) App {
	a := App{
		board:            newKanban(),
		service:          svc,
		form:             newTaskForm(),
		availableRunners: agent.AvailableRunners(),
		pendingRecons:    make(map[string]pendingRecon),
	}
	for _, opt := range opts {
		opt(&a)
	}
	return a
}

func (a App) Init() tea.Cmd {
	return tea.Batch(a.loadTasks(), a.scheduleAgentTick())
}

func (a App) loadTasks() tea.Cmd {
	return func() tea.Msg {
		tasks, err := a.service.ListTasks(context.Background())
		if err != nil {
			return errMsg{err}
		}
		return tasksLoadedMsg{tasks: tasks}
	}
}

// scheduleAgentTick returns a Cmd that fires after the poll interval.
func (a App) scheduleAgentTick() tea.Cmd {
	return tea.Tick(agentPollInterval, func(time.Time) tea.Msg {
		return agentTickMsg{}
	})
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.board.SetSize(msg.Width, msg.Height)
		a.form.SetSize(msg.Width, msg.Height)
		if a.overlay == overlayDetail {
			a.detail.SetSize(msg.Width, msg.Height)
		}
		a.ready = true
		return a, nil

	case tasksLoadedMsg:
		a.lastTasks = msg.tasks
		a.board.LoadTasks(msg.tasks)
		// Follow cursor to the column where a task was just moved
		if a.cursorFollow != nil {
			a.board.FocusOnStatus(a.cursorFollow.newStatus)
			a.board.SelectTaskByID(a.cursorFollow.taskID)
			a.cursorFollow = nil
		}
		// Startup reconciliation: check for stale active states
		a.reconcileStaleOnStartup()
		return a, nil

	case taskCreatedMsg:
		a.overlay = overlayNone
		a.form.Reset()
		return a, tea.Batch(
			a.loadTasks(),
			a.notify(fmt.Sprintf("Created: %s", msg.task.Title)),
		)

	case taskMovedMsg:
		a.cursorFollow = &pendingFocus{taskID: msg.taskID, newStatus: msg.newStatus}
		cmds := []tea.Cmd{
			a.loadTasks(),
			a.notify(fmt.Sprintf("Moved to %s", msg.newStatus)),
		}
		// Auto-respawn agent if it was active (new column → new workflow)
		if msg.hadAgent {
			cmds = append(cmds, a.respawnAgent(msg.taskID, msg.newStatus))
		}
		return a, tea.Batch(cmds...)

	case taskDeletedMsg:
		return a, tea.Batch(
			a.loadTasks(),
			a.notify("Task deleted"),
		)

	case agentTickMsg:
		cmds := a.reconcileAgents()
		cmds = append(cmds, a.scheduleAgentTick(), a.loadTasks())
		return a, tea.Batch(cmds...)

	case errMsg:
		return a, a.notify(fmt.Sprintf("Error: %s", msg.err))

	case notifyMsg:
		a.notification = &notification{
			text:    msg.text,
			expires: time.Now().Add(3 * time.Second),
		}
		return a, scheduleNotificationClear(3 * time.Second)

	case spawnAfterConfirmMsg:
		// After skip-permissions confirmation, enter agent selection flow
		cmd := a.spawnAgent(msg.task)
		return a, cmd

	case agentSelectedMsg:
		a.overlay = overlayNone
		return a, a.spawnAgentWithRunner(msg.task, msg.runner)

	case agentSpawnedMsg:
		return a, tea.Batch(
			a.loadTasks(),
			a.notify("Agent spawned"),
		)

	case agentKilledMsg:
		return a, tea.Batch(
			a.loadTasks(),
			a.notify("Agent killed"),
		)

	case agentViewDoneMsg:
		return a, a.loadTasks()

	case serverStatusMsg:
		a.tunnelURL = msg.tunnelURL
		a.peerCount = msg.peerCount
		a.serverActive = msg.connected
		return a, nil

	case clearNotificationMsg:
		if a.notification != nil && time.Now().After(a.notification.expires) {
			a.notification = nil
		}
		return a, nil
	}

	// Route to overlay if active
	if a.overlay != overlayNone {
		return a.updateOverlay(msg)
	}

	return a.updateBoard(msg)
}

// reconcileStaleOnStartup checks all active tasks on first load.
// If an agent window is dead, start grace period immediately.
func (a *App) reconcileStaleOnStartup() {
	windows, _ := tmux.ListWindows()
	for _, task := range a.lastTasks {
		if task.AgentStatus != db.AgentActive {
			continue
		}
		windowName := agent.WindowName(task)
		if !windows[windowName] {
			if _, pending := a.pendingRecons[task.ID]; !pending {
				a.pendingRecons[task.ID] = pendingRecon{
					detectedAt:        time.Now(),
					columnAtDetection: task.Status,
				}
			}
		}
	}
}

// reconcileAgents checks agent windows and manages the grace period state machine.
// Returns commands for any DB updates that need to happen.
func (a *App) reconcileAgents() []tea.Cmd {
	var cmds []tea.Cmd
	ctx := context.Background()
	windows, _ := tmux.ListWindows()

	for _, task := range a.lastTasks {
		if task.AgentStatus != db.AgentActive {
			continue
		}

		windowName := agent.WindowName(task)

		if windows[windowName] {
			// Agent is running — clear any pending reconciliation
			delete(a.pendingRecons, task.ID)
			continue
		}

		// Window is dead
		pending, inGrace := a.pendingRecons[task.ID]
		if !inGrace {
			// Just detected — start grace period
			a.pendingRecons[task.ID] = pendingRecon{
				detectedAt:        time.Now(),
				columnAtDetection: task.Status,
			}
			continue
		}

		if time.Since(pending.detectedAt) < gracePeriod {
			// Still in grace period — wait
			continue
		}

		// Grace period elapsed — determine outcome
		delete(a.pendingRecons, task.ID)

		// Re-read task from DB (agent may have moved it during grace period)
		freshTask, err := a.service.GetTask(ctx, task.ID)
		if err != nil {
			continue
		}

		// Determine baseline: prefer AgentSpawnedStatus (set at spawn time),
		// fall back to columnAtDetection (set when window death was detected)
		baseline := db.TaskStatus(freshTask.AgentSpawnedStatus)
		if baseline == "" {
			baseline = pending.columnAtDetection
		}

		if freshTask.ResetRequested {
			// Agent wants fresh context — mark idle for respawn
			freshTask.ResetRequested = false
			freshTask.AgentStatus = db.AgentIdle
			freshTask.AgentStartedAt = ""
			freshTask.AgentSpawnedStatus = ""
			a.service.UpdateTask(ctx, freshTask)
			cmds = append(cmds, a.notify("Agent reset requested — ready for respawn"))
		} else if freshTask.Status != baseline {
			// Task moved to a new column — agent completed successfully
			freshTask.AgentStatus = db.AgentCompleted
			freshTask.AgentStartedAt = ""
			freshTask.AgentSpawnedStatus = ""
			a.service.UpdateTask(ctx, freshTask)
		} else {
			// Task still in same column — agent crashed/failed
			freshTask.AgentStatus = db.AgentError
			freshTask.AgentStartedAt = ""
			freshTask.AgentSpawnedStatus = ""
			a.service.UpdateTask(ctx, freshTask)
		}
	}

	return cmds
}

func (a App) updateOverlay(msg tea.Msg) (tea.Model, tea.Cmd) {
	// Confirm overlay handles its own keys (including esc)
	if a.overlay == overlayConfirm {
		return a.updateConfirm(msg)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Escape):
			a.overlay = overlayNone
			a.form.Reset()
			return a, nil
		}
	}

	switch a.overlay {
	case overlayForm:
		return a.updateForm(msg)
	case overlayDetail:
		return a.updateDetail(msg)
	case overlayPicker:
		return a.updatePicker(msg)
	case overlayHelp:
		if _, ok := msg.(tea.KeyMsg); ok {
			a.overlay = overlayNone
			return a, nil
		}
	}

	return a, nil
}

func (a App) updateConfirm(msg tea.Msg) (tea.Model, tea.Cmd) {
	keyMsg, ok := msg.(tea.KeyMsg)
	if !ok {
		return a, nil
	}

	switch keyMsg.String() {
	case "y":
		task := a.pendingSpawnTask
		a.pendingSpawnTask = nil
		a.overlay = overlayNone
		task.SkipPermissions = true
		return a, a.persistAndSpawn(*task)
	case "n":
		task := a.pendingSpawnTask
		a.pendingSpawnTask = nil
		a.overlay = overlayNone
		task.SkipPermissions = false
		return a, a.persistAndSpawn(*task)
	case "esc":
		a.pendingSpawnTask = nil
		a.overlay = overlayNone
		return a, nil
	}

	return a, nil
}

// persistAndSpawn saves the task's SkipPermissions then triggers the agent selection flow.
func (a *App) persistAndSpawn(task db.Task) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		if err := a.service.UpdateTask(ctx, &task); err != nil {
			return errMsg{fmt.Errorf("saving skip_permissions: %w", err)}
		}
		return spawnAfterConfirmMsg{task: task}
	}
}

func (a App) updatePicker(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	a.picker, cmd = a.picker.Update(msg)
	return a, cmd
}

func (a App) updateForm(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "enter" && a.form.focusTitle {
			// Tab to description if pressing enter on title
			a.form.titleInput.Blur()
			a.form.descInput.Focus()
			a.form.focusTitle = false
			return a, nil
		}
		if msg.String() == "ctrl+s" || (msg.String() == "enter" && !a.form.focusTitle) {
			title := a.form.Title()
			if title == "" {
				return a, a.notify("Title cannot be empty")
			}
			return a, a.createTask(title, a.form.Desc())
		}
	}

	var cmd tea.Cmd
	a.form, cmd = a.form.Update(msg)
	return a, cmd
}

func (a App) updateDetail(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.MoveRight):
			return a, a.moveTask(a.detail.task.ID, a.nextStatus(a.detail.task.Status))
		case key.Matches(msg, keys.MoveLeft):
			return a, a.moveTask(a.detail.task.ID, a.prevStatus(a.detail.task.Status))
		case key.Matches(msg, keys.SpawnAgent):
			if a.detail.task.AgentStatus == db.AgentActive {
				return a, a.notify("Agent already running")
			}
			t := a.detail.task
			a.pendingSpawnTask = &t
			a.overlay = overlayConfirm
			return a, nil
		case key.Matches(msg, keys.KillAgent):
			if a.detail.task.AgentStatus != db.AgentActive {
				return a, a.notify("No agent running")
			}
			return a, a.killAgent(a.detail.task)
		case key.Matches(msg, keys.ViewAgent):
			if a.detail.task.AgentStatus != db.AgentActive {
				return a, a.notify("No agent running")
			}
			return a, a.viewAgent(a.detail.task)
		case key.Matches(msg, keys.Delete):
			a.overlay = overlayNone
			return a, a.deleteTask(a.detail.task.ID)
		}
	}

	var cmd tea.Cmd
	a.detail, cmd = a.detail.Update(msg)
	return a, cmd
}

func (a App) updateBoard(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, keys.Quit):
			return a, tea.Quit
		case key.Matches(msg, keys.Left):
			a.board.FocusLeft()
			return a, nil
		case key.Matches(msg, keys.Right):
			a.board.FocusRight()
			return a, nil
		case key.Matches(msg, keys.New):
			a.overlay = overlayForm
			a.form.Reset()
			return a, a.form.titleInput.Focus()
		case key.Matches(msg, keys.Enter):
			if task := a.board.SelectedTask(); task != nil {
				if task.AgentStatus == db.AgentActive {
					return a, a.viewAgent(*task)
				}
				a.detail = newTaskDetail(*task)
				a.detail.SetSize(a.width, a.height)
				a.overlay = overlayDetail
			}
			return a, nil
		case key.Matches(msg, keys.MoveRight):
			if task := a.board.SelectedTask(); task != nil {
				return a, a.moveTask(task.ID, a.board.NextColumn())
			}
			return a, nil
		case key.Matches(msg, keys.MoveLeft):
			if task := a.board.SelectedTask(); task != nil {
				return a, a.moveTask(task.ID, a.board.PrevColumn())
			}
			return a, nil
		case key.Matches(msg, keys.SpawnAgent):
			if task := a.board.SelectedTask(); task != nil {
				if task.AgentStatus == db.AgentActive {
					return a, a.notify("Agent already running")
				}
				t := *task
				a.pendingSpawnTask = &t
				a.overlay = overlayConfirm
				return a, nil
			}
			return a, nil
		case key.Matches(msg, keys.KillAgent):
			if task := a.board.SelectedTask(); task != nil {
				if task.AgentStatus != db.AgentActive {
					return a, a.notify("No agent running")
				}
				return a, a.killAgent(*task)
			}
			return a, nil
		case key.Matches(msg, keys.ViewAgent):
			if task := a.board.SelectedTask(); task != nil {
				if task.AgentStatus != db.AgentActive {
					return a, a.notify("No agent running")
				}
				return a, a.viewAgent(*task)
			}
			return a, nil
		case key.Matches(msg, keys.Delete):
			if task := a.board.SelectedTask(); task != nil {
				return a, a.deleteTask(task.ID)
			}
			return a, nil
		case key.Matches(msg, keys.Help):
			a.overlay = overlayHelp
			return a, nil
		}
	}

	var cmd tea.Cmd
	a.board, cmd = a.board.Update(msg)
	return a, cmd
}

func (a App) View() string {
	if !a.ready {
		return "Loading..."
	}

	boardView := a.board.View()
	statusBar := a.board.statusBar()

	// Append tunnel/connection info to the status bar
	if a.tunnelURL != "" {
		serverStatus := serverStatusBar(a.tunnelURL, a.peerCount, a.serverActive, a.width)
		statusBar = statusBar + serverStatus
	}

	if a.notification != nil {
		statusBar = notificationStyle.Render(a.notification.text)
	}

	help := helpStyle.Render(" h/l:columns  j/k:tasks  o:new  m/M:move  a:spawn agent  v:view  A:kill  enter:open/view  x:delete  ?:help  q:quit")

	mainView := lipgloss.JoinVertical(lipgloss.Left, boardView, statusBar, help)

	// Overlay rendering
	switch a.overlay {
	case overlayForm:
		return a.renderOverlay(mainView, a.form.View())
	case overlayDetail:
		return a.renderOverlay(mainView, a.detail.View())
	case overlayPicker:
		return a.renderOverlay(mainView, a.picker.View())
	case overlayHelp:
		return a.renderOverlay(mainView, a.helpView())
	case overlayConfirm:
		return a.renderOverlay(mainView, a.confirmView())
	}

	return mainView
}

func (a App) renderOverlay(bg, overlay string) string {
	return lipgloss.Place(
		a.width, a.height,
		lipgloss.Center, lipgloss.Center,
		overlay,
		lipgloss.WithWhitespaceChars(" "),
		lipgloss.WithWhitespaceForeground(lipgloss.Color("#333333")),
	)
}

func (a App) helpView() string {
	help := `Agentboard - Keyboard Shortcuts

Navigation:
  h / ←     Previous column
  l / →     Next column
  j / ↓     Next task
  k / ↑     Previous task

Actions:
  o         Create new task
  m         Move task right
  M         Move task left
  enter     Open task (view agent if active)
  x         Delete task
  a         Spawn agent (select if multiple available)
  v         View agent (split pane, Ctrl+q to close)
  A         Kill running agent

General:
  ?         Toggle help
  esc       Close overlay
  q         Quit

Press any key to close.`

	return overlayStyle.Width(a.width / 3).Render(help)
}

func (a App) confirmView() string {
	current := ""
	if a.pendingSpawnTask != nil && a.pendingSpawnTask.SkipPermissions {
		current = " [currently: yes]"
	}

	content := fmt.Sprintf(`Skip permissions?%s

Allow agent to run commands
without asking for approval.

  y - yes    n - no    esc - cancel`, current)

	return overlayStyle.Width(40).Render(content)
}

// Command helpers

func (a App) notify(text string) tea.Cmd {
	return func() tea.Msg {
		return notifyMsg{text: text}
	}
}

func (a App) createTask(title, description string) tea.Cmd {
	return func() tea.Msg {
		task, err := a.service.CreateTask(context.Background(), title, description)
		if err != nil {
			return errMsg{err}
		}
		return taskCreatedMsg{task: task}
	}
}

func (a App) moveTask(id string, newStatus db.TaskStatus) tea.Cmd {
	// Check if the task has an active agent before moving (for auto-respawn)
	hadAgent := false
	if task := a.board.SelectedTask(); task != nil && task.ID == id {
		hadAgent = task.AgentStatus == db.AgentActive
	} else if a.overlay == overlayDetail && a.detail.task.ID == id {
		hadAgent = a.detail.task.AgentStatus == db.AgentActive
	}

	return func() tea.Msg {
		ctx := context.Background()
		if err := a.service.MoveTask(ctx, id, newStatus); err != nil {
			return errMsg{err}
		}
		// When a task is moved via TUI and no agent window is alive,
		// reset agent status to idle (prevents stale completed/error states)
		if !hadAgent {
			task, err := a.service.GetTask(ctx, id)
			if err == nil && task.AgentStatus != db.AgentIdle {
				windowName := agent.WindowName(*task)
				if !tmux.IsWindowAlive(windowName) {
					task.AgentStatus = db.AgentIdle
					task.AgentStartedAt = ""
					task.AgentSpawnedStatus = ""
					task.ResetRequested = false
					a.service.UpdateTask(ctx, task)
				}
			}
		}
		return taskMovedMsg{taskID: id, newStatus: newStatus, hadAgent: hadAgent}
	}
}

func (a App) deleteTask(id string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		// Kill agent window before deleting
		task, err := a.service.GetTask(ctx, id)
		if err == nil && task.AgentStatus == db.AgentActive {
			windowName := agent.WindowName(*task)
			tmux.KillWindow(windowName)
		}
		if err := a.service.DeleteTask(ctx, id); err != nil {
			return errMsg{err}
		}
		return taskDeletedMsg{taskID: id}
	}
}

func (a App) nextStatus(current db.TaskStatus) db.TaskStatus {
	for i, s := range columnOrder {
		if s == current && i < len(columnOrder)-1 {
			return columnOrder[i+1]
		}
	}
	return current
}

func (a App) prevStatus(current db.TaskStatus) db.TaskStatus {
	for i, s := range columnOrder {
		if s == current && i > 0 {
			return columnOrder[i-1]
		}
	}
	return current
}

// Agent command helpers

// spawnAgent initiates agent spawning. If multiple agents are available, it shows the picker.
// If only one is available, it spawns directly. If none, it shows an error.
func (a *App) spawnAgent(task db.Task) tea.Cmd {
	runners := a.availableRunners
	switch len(runners) {
	case 0:
		return a.notify("No agent CLIs detected in PATH")
	case 1:
		return a.spawnAgentWithRunner(task, runners[0])
	default:
		a.picker = newAgentPicker(runners, task, a.width, a.height)
		a.overlay = overlayPicker
		return nil
	}
}

// spawnAgentWithRunner spawns a specific agent runner on a task.
func (a App) spawnAgentWithRunner(task db.Task, runner agent.AgentRunner) tea.Cmd {
	return func() tea.Msg {
		if err := agent.Spawn(context.Background(), a.service, task, runner); err != nil {
			return errMsg{fmt.Errorf("%s", err)}
		}
		return agentSpawnedMsg{taskID: task.ID}
	}
}

func (a App) viewAgent(task db.Task) tea.Cmd {
	winName := agent.WindowName(task)
	if tmux.InTmux() {
		// Split pane: agent on the right, TUI stays running
		return func() tea.Msg {
			if err := tmux.SplitView(winName); err != nil {
				return errMsg{fmt.Errorf("split view: %w", err)}
			}
			return nil
		}
	}
	// Not in tmux: full-screen attach, Ctrl+q to return to TUI
	c := tmux.AttachCmd(winName)
	return tea.ExecProcess(c, func(err error) tea.Msg {
		return agentViewDoneMsg{}
	})
}

func (a App) respawnAgent(taskID string, newStatus db.TaskStatus) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		// Fetch the updated task from DB
		task, err := a.service.GetTask(ctx, taskID)
		if err != nil {
			return errMsg{err}
		}

		// Guard: skip respawn if agent was already spawned for this column
		if task.AgentSpawnedStatus == string(task.Status) {
			return notifyMsg{text: "Agent already working on this column — skipping respawn"}
		}

		// Look up the runner the task was previously using
		runner := agent.GetRunner(task.AgentName)
		if runner == nil || !runner.Available() {
			// Agent no longer available — mark as error
			task.AgentStatus = db.AgentError
			task.AgentStartedAt = ""
			task.AgentSpawnedStatus = ""
			a.service.UpdateTask(ctx, task)
			return notifyMsg{text: fmt.Sprintf("Agent %q no longer available", task.AgentName)}
		}

		// Deactivate any active ralph loop so the new agent runs once without looping
		_ = agent.DeactivateRalphLoop(*task)

		// Spawn handles killing the old window and creating a new one
		if err := agent.Spawn(ctx, a.service, *task, runner); err != nil {
			return errMsg{fmt.Errorf("respawn agent: %w", err)}
		}
		return agentSpawnedMsg{taskID: taskID}
	}
}

func (a App) killAgent(task db.Task) tea.Cmd {
	return func() tea.Msg {
		if err := agent.Kill(context.Background(), a.service, task); err != nil {
			return errMsg{fmt.Errorf("%s", err)}
		}
		return agentKilledMsg{taskID: task.ID}
	}
}
