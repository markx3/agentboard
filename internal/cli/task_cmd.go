package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/tabwriter"

	"github.com/spf13/cobra"

	boardpkg "github.com/marcosfelipeeipper/agentboard/internal/board"
	"github.com/marcosfelipeeipper/agentboard/internal/db"
)

var (
	taskFilterStatus   string
	taskFilterAssignee string
	taskOutputJSON     bool
)

var taskCmd = &cobra.Command{
	Use:   "task",
	Short: "Manage tasks programmatically",
}

var taskListCmd = &cobra.Command{
	Use:   "list",
	Short: "List tasks",
	RunE:  runTaskList,
}

var taskCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new task",
	RunE:  runTaskCreate,
}

var taskMoveCmd = &cobra.Command{
	Use:   "move <task-id> <column>",
	Short: "Move a task to a column",
	Args:  cobra.ExactArgs(2),
	RunE:  runTaskMove,
}

var taskGetCmd = &cobra.Command{
	Use:   "get <task-id>",
	Short: "Get task details",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskGet,
}

var taskDeleteCmd = &cobra.Command{
	Use:   "delete <task-id>",
	Short: "Delete a task",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskDelete,
}

var taskClaimCmd = &cobra.Command{
	Use:   "claim <task-id>",
	Short: "Claim a task",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskClaim,
}

var taskUnclaimCmd = &cobra.Command{
	Use:   "unclaim <task-id>",
	Short: "Unclaim a task",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskUnclaim,
}

var taskUpdateCmd = &cobra.Command{
	Use:   "update <task-id>",
	Short: "Update task fields",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskUpdate,
}

var taskBlockCmd = &cobra.Command{
	Use:   "block <task-id> <blocker-id>",
	Short: "Mark a task as blocked by another task",
	Args:  cobra.ExactArgs(2),
	RunE:  runTaskBlock,
}

var taskUnblockCmd = &cobra.Command{
	Use:   "unblock <task-id> <blocker-id>",
	Short: "Remove a dependency between tasks",
	Args:  cobra.ExactArgs(2),
	RunE:  runTaskUnblock,
}

var (
	createTitle string
	createDesc  string
	claimUser   string
	updateTitle string
	updateDesc  string
	updateAssignee string
	updateBranch string
	updatePRUrl  string
)

func init() {
	taskListCmd.Flags().StringVar(&taskFilterStatus, "status", "", "filter by status")
	taskListCmd.Flags().StringVar(&taskFilterAssignee, "assignee", "", "filter by assignee")
	taskListCmd.Flags().BoolVar(&taskOutputJSON, "json", false, "output as JSON")

	taskCreateCmd.Flags().StringVar(&createTitle, "title", "", "task title (required)")
	taskCreateCmd.Flags().StringVar(&createDesc, "description", "", "task description")
	taskCreateCmd.MarkFlagRequired("title")

	taskGetCmd.Flags().BoolVar(&taskOutputJSON, "json", false, "output as JSON")

	taskClaimCmd.Flags().StringVar(&claimUser, "user", "", "username to claim as")

	taskUpdateCmd.Flags().StringVar(&updateTitle, "title", "", "new title")
	taskUpdateCmd.Flags().StringVar(&updateDesc, "description", "", "new description")
	taskUpdateCmd.Flags().StringVar(&updateAssignee, "assignee", "", "new assignee")
	taskUpdateCmd.Flags().StringVar(&updateBranch, "branch", "", "new branch name")
	taskUpdateCmd.Flags().StringVar(&updatePRUrl, "pr-url", "", "new PR URL")

	taskCmd.AddCommand(taskListCmd, taskCreateCmd, taskMoveCmd, taskGetCmd, taskDeleteCmd, taskClaimCmd, taskUnclaimCmd, taskUpdateCmd, taskBlockCmd, taskUnblockCmd)
	rootCmd.AddCommand(taskCmd)
}

func openService() (boardpkg.Service, func(), error) {
	dbPath := filepath.Join(".agentboard", "board.db")
	database, err := db.Open(dbPath)
	if err != nil {
		return nil, nil, fmt.Errorf("opening database: %w", err)
	}
	svc := boardpkg.NewLocalService(database)
	return svc, func() { database.Close() }, nil
}

func runTaskList(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := openService()
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := context.Background()
	var tasks []db.Task

	if taskFilterStatus != "" {
		tasks, err = svc.ListTasksByStatus(ctx, db.TaskStatus(taskFilterStatus))
	} else {
		tasks, err = svc.ListTasks(ctx)
	}
	if err != nil {
		return err
	}

	if taskFilterAssignee != "" {
		var filtered []db.Task
		for _, t := range tasks {
			if t.Assignee == taskFilterAssignee {
				filtered = append(filtered, t)
			}
		}
		tasks = filtered
	}

	if taskOutputJSON {
		return json.NewEncoder(os.Stdout).Encode(tasks)
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tSTATUS\tASSIGNEE\tAGENT")
	for _, t := range tasks {
		agentCol := string(t.AgentStatus)
		if t.AgentName != "" {
			agentCol = fmt.Sprintf("%s (%s)", t.AgentName, t.AgentStatus)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			t.ID[:8], t.Title, t.Status, t.Assignee, agentCol)
	}
	return w.Flush()
}

func runTaskCreate(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := openService()
	if err != nil {
		return err
	}
	defer cleanup()

	task, err := svc.CreateTask(context.Background(), createTitle, createDesc)
	if err != nil {
		return err
	}

	if taskOutputJSON {
		return json.NewEncoder(os.Stdout).Encode(task)
	}

	fmt.Printf("Created task: %s (%s)\n", task.Title, task.ID[:8])
	return nil
}

func runTaskMove(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := openService()
	if err != nil {
		return err
	}
	defer cleanup()

	taskID := args[0]
	newStatus := db.TaskStatus(args[1])
	if !newStatus.Valid() {
		return fmt.Errorf("invalid status: %s (use: backlog, brainstorm, planning, in_progress, review, done)", args[1])
	}

	// Find task by prefix
	tasks, err := svc.ListTasks(context.Background())
	if err != nil {
		return err
	}
	fullID := findByPrefix(tasks, taskID)
	if fullID == "" {
		return fmt.Errorf("task not found: %s", taskID)
	}

	if err := svc.MoveTask(context.Background(), fullID, newStatus); err != nil {
		return err
	}

	// Update agent metadata so reconciliation has accurate baseline
	task, getErr := svc.GetTask(context.Background(), fullID)
	if getErr == nil && task.AgentStatus == db.AgentActive {
		task.AgentSpawnedStatus = string(newStatus)
		svc.UpdateTask(context.Background(), task)
	}

	fmt.Printf("Moved task to %s\n", newStatus)
	return nil
}

func runTaskGet(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := openService()
	if err != nil {
		return err
	}
	defer cleanup()

	// Find by prefix
	tasks, err := svc.ListTasks(context.Background())
	if err != nil {
		return err
	}
	fullID := findByPrefix(tasks, args[0])
	if fullID == "" {
		return fmt.Errorf("task not found: %s", args[0])
	}

	task, err := svc.GetTask(context.Background(), fullID)
	if err != nil {
		return err
	}

	if taskOutputJSON {
		return json.NewEncoder(os.Stdout).Encode(task)
	}

	fmt.Printf("ID:          %s\n", task.ID)
	fmt.Printf("Title:       %s\n", task.Title)
	fmt.Printf("Status:      %s\n", task.Status)
	fmt.Printf("Assignee:    %s\n", task.Assignee)
	fmt.Printf("Agent:       %s (%s)\n", task.AgentName, task.AgentStatus)
	fmt.Printf("Branch:      %s\n", task.BranchName)
	fmt.Printf("PR:          %s\n", task.PRUrl)
	fmt.Printf("Description: %s\n", task.Description)
	fmt.Printf("Created:     %s\n", task.CreatedAt.Format("2006-01-02 15:04"))
	return nil
}

func runTaskDelete(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := openService()
	if err != nil {
		return err
	}
	defer cleanup()

	tasks, err := svc.ListTasks(context.Background())
	if err != nil {
		return err
	}
	fullID := findByPrefix(tasks, args[0])
	if fullID == "" {
		return fmt.Errorf("task not found: %s", args[0])
	}

	if err := svc.DeleteTask(context.Background(), fullID); err != nil {
		return err
	}

	fmt.Println("Task deleted")
	return nil
}

func runTaskClaim(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := openService()
	if err != nil {
		return err
	}
	defer cleanup()

	tasks, err := svc.ListTasks(context.Background())
	if err != nil {
		return err
	}
	fullID := findByPrefix(tasks, args[0])
	if fullID == "" {
		return fmt.Errorf("task not found: %s", args[0])
	}

	user := claimUser
	if user == "" {
		user = "local"
	}

	if err := svc.ClaimTask(context.Background(), fullID, user); err != nil {
		return err
	}

	fmt.Printf("Task claimed by %s\n", user)
	return nil
}

func runTaskUnclaim(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := openService()
	if err != nil {
		return err
	}
	defer cleanup()

	tasks, err := svc.ListTasks(context.Background())
	if err != nil {
		return err
	}
	fullID := findByPrefix(tasks, args[0])
	if fullID == "" {
		return fmt.Errorf("task not found: %s", args[0])
	}

	if err := svc.UnclaimTask(context.Background(), fullID); err != nil {
		return err
	}

	fmt.Println("Task unclaimed")
	return nil
}

func runTaskUpdate(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := openService()
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := context.Background()
	tasks, err := svc.ListTasks(ctx)
	if err != nil {
		return err
	}
	fullID := findByPrefix(tasks, args[0])
	if fullID == "" {
		return fmt.Errorf("task not found: %s", args[0])
	}

	task, err := svc.GetTask(ctx, fullID)
	if err != nil {
		return err
	}

	changed := false
	if cmd.Flags().Changed("title") {
		task.Title = updateTitle
		changed = true
	}
	if cmd.Flags().Changed("description") {
		task.Description = updateDesc
		changed = true
	}
	if cmd.Flags().Changed("assignee") {
		task.Assignee = updateAssignee
		changed = true
	}
	if cmd.Flags().Changed("branch") {
		task.BranchName = updateBranch
		changed = true
	}
	if cmd.Flags().Changed("pr-url") {
		task.PRUrl = updatePRUrl
		changed = true
	}

	if !changed {
		return fmt.Errorf("no fields to update (use --title, --description, --assignee, --branch, --pr-url)")
	}

	if err := svc.UpdateTask(ctx, task); err != nil {
		return fmt.Errorf("updating task: %w", err)
	}

	fmt.Printf("Task %s updated\n", task.ID[:8])
	return nil
}

func runTaskBlock(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := openService()
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := context.Background()
	tasks, err := svc.ListTasks(ctx)
	if err != nil {
		return err
	}

	taskID := findByPrefix(tasks, args[0])
	if taskID == "" {
		return fmt.Errorf("task not found: %s", args[0])
	}
	blockerID := findByPrefix(tasks, args[1])
	if blockerID == "" {
		return fmt.Errorf("blocker task not found: %s", args[1])
	}

	if err := svc.AddDependency(ctx, taskID, blockerID); err != nil {
		return err
	}

	fmt.Printf("Task %s is now blocked by %s\n", args[0], args[1])
	return nil
}

func runTaskUnblock(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := openService()
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := context.Background()
	tasks, err := svc.ListTasks(ctx)
	if err != nil {
		return err
	}

	taskID := findByPrefix(tasks, args[0])
	if taskID == "" {
		return fmt.Errorf("task not found: %s", args[0])
	}
	blockerID := findByPrefix(tasks, args[1])
	if blockerID == "" {
		return fmt.Errorf("blocker task not found: %s", args[1])
	}

	if err := svc.RemoveDependency(ctx, taskID, blockerID); err != nil {
		return err
	}

	fmt.Printf("Dependency removed: %s no longer blocked by %s\n", args[0], args[1])
	return nil
}

func findByPrefix(tasks []db.Task, prefix string) string {
	for _, t := range tasks {
		if len(t.ID) >= len(prefix) && t.ID[:len(prefix)] == prefix {
			return t.ID
		}
	}
	return ""
}
