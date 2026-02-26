package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	boardpkg "github.com/markx3/agentboard/internal/board"
	"github.com/markx3/agentboard/internal/db"
)

var (
	taskFilterStatus   string
	taskFilterAssignee string
	taskFilterSearch   string
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

var taskCommentCmd = &cobra.Command{
	Use:   "comment <task-id>",
	Short: "Add a comment to a task",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskComment,
}

var taskBlockCmd = &cobra.Command{
	Use:   "block <task-id> <blocker-id>",
	Short: "Mark a task as blocked by another",
	Args:  cobra.ExactArgs(2),
	RunE:  runTaskBlock,
}

var taskUnblockCmd = &cobra.Command{
	Use:   "unblock <task-id> <blocker-id>",
	Short: "Remove a dependency",
	Args:  cobra.ExactArgs(2),
	RunE:  runTaskUnblock,
}

var taskSuggestCmd = &cobra.Command{
	Use:   "suggest <task-id>",
	Short: "Create a suggestion for a task",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskSuggest,
}

var taskProposeCmd = &cobra.Command{
	Use:   "propose",
	Short: "Propose a new task (creates a suggestion)",
	RunE:  runTaskPropose,
}

var taskSuggestionsCmd = &cobra.Command{
	Use:   "suggestions",
	Short: "List suggestions",
	RunE:  runTaskSuggestions,
}

var taskSuggestionCmd = &cobra.Command{
	Use:   "suggestion",
	Short: "Manage individual suggestions",
}

var taskSuggestionAcceptCmd = &cobra.Command{
	Use:   "accept <suggestion-id>",
	Short: "Accept a suggestion",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskSuggestionAccept,
}

var taskSuggestionDismissCmd = &cobra.Command{
	Use:   "dismiss <suggestion-id>",
	Short: "Dismiss a suggestion",
	Args:  cobra.ExactArgs(1),
	RunE:  runTaskSuggestionDismiss,
}

var (
	createTitle string
	createDesc  string
	claimUser   string
	noEnrich    bool

	// task update flags
	updateTitle            string
	updateDescription      string
	updateAssignee         string
	updateBranch           string
	updatePRUrl            string
	updateAddDep           string
	updateRemoveDep        string
	updateEnrichmentStatus string

	// task comment flags
	commentAuthor string
	commentBody   string

	// task suggest flags
	suggestAuthor  string
	suggestTitle   string
	suggestMessage string

	// task propose flags
	proposeTitle       string
	proposeDescription string
	proposeReason      string

	// task suggestions filter
	suggestionsStatus string
)

func init() {
	// Make --json a persistent flag on taskCmd so all subcommands inherit it
	taskCmd.PersistentFlags().BoolVar(&taskOutputJSON, "json", false, "output as JSON")

	taskListCmd.Flags().StringVar(&taskFilterStatus, "status", "", "filter by status")
	taskListCmd.Flags().StringVar(&taskFilterAssignee, "assignee", "", "filter by assignee")
	taskListCmd.Flags().StringVar(&taskFilterSearch, "search", "", "filter by title/description substring (case-insensitive)")

	taskCreateCmd.Flags().StringVar(&createTitle, "title", "", "task title (required)")
	taskCreateCmd.Flags().StringVar(&createDesc, "description", "", "task description")
	taskCreateCmd.Flags().BoolVar(&noEnrich, "no-enrich", false, "skip automatic enrichment")
	taskCreateCmd.MarkFlagRequired("title")

	taskClaimCmd.Flags().StringVar(&claimUser, "user", "", "username to claim as")

	// task update flags
	taskUpdateCmd.Flags().StringVar(&updateTitle, "title", "", "update task title")
	taskUpdateCmd.Flags().StringVar(&updateDescription, "description", "", "update task description")
	taskUpdateCmd.Flags().StringVar(&updateAssignee, "assignee", "", "update assignee")
	taskUpdateCmd.Flags().StringVar(&updateBranch, "branch", "", "update branch name")
	taskUpdateCmd.Flags().StringVar(&updatePRUrl, "pr-url", "", "update PR URL")
	taskUpdateCmd.Flags().StringVar(&updateAddDep, "add-dep", "", "add dependency (task ID prefix)")
	taskUpdateCmd.Flags().StringVar(&updateRemoveDep, "remove-dep", "", "remove dependency (task ID prefix)")
	taskUpdateCmd.Flags().StringVar(&updateEnrichmentStatus, "enrichment-status", "", "set enrichment status")

	// task comment flags
	taskCommentCmd.Flags().StringVar(&commentAuthor, "author", "", "comment author (required)")
	taskCommentCmd.Flags().StringVar(&commentBody, "body", "", "comment body (required)")
	taskCommentCmd.MarkFlagRequired("author")
	taskCommentCmd.MarkFlagRequired("body")

	// task suggest flags
	taskSuggestCmd.Flags().StringVar(&suggestAuthor, "author", "", "suggestion author")
	taskSuggestCmd.Flags().StringVar(&suggestTitle, "title", "", "suggestion title")
	taskSuggestCmd.Flags().StringVar(&suggestMessage, "message", "", "suggestion message (required)")
	taskSuggestCmd.MarkFlagRequired("message")

	// task propose flags
	taskProposeCmd.Flags().StringVar(&proposeTitle, "title", "", "proposed task title (required)")
	taskProposeCmd.Flags().StringVar(&proposeDescription, "description", "", "proposed task description")
	taskProposeCmd.Flags().StringVar(&proposeReason, "reason", "", "why this task is needed")
	taskProposeCmd.MarkFlagRequired("title")

	// task suggestions filter
	taskSuggestionsCmd.Flags().StringVar(&suggestionsStatus, "status", "pending", "filter by status (pending, accepted, dismissed)")

	// Build command tree
	taskSuggestionCmd.AddCommand(taskSuggestionAcceptCmd, taskSuggestionDismissCmd)
	taskCmd.AddCommand(
		taskListCmd, taskCreateCmd, taskMoveCmd, taskGetCmd, taskDeleteCmd,
		taskClaimCmd, taskUnclaimCmd, taskUpdateCmd, taskCommentCmd,
		taskBlockCmd, taskUnblockCmd,
		taskSuggestCmd, taskProposeCmd, taskSuggestionsCmd, taskSuggestionCmd,
	)
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

	if taskFilterSearch != "" {
		tasks = filterTasksBySearch(tasks, taskFilterSearch)
	}

	// Populate dependency data
	deps, depsErr := svc.ListAllDependencies(ctx)
	if depsErr == nil && deps != nil {
		for i := range tasks {
			if blockers, ok := deps[tasks[i].ID]; ok {
				tasks[i].BlockedBy = blockers
			}
		}
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

	// Set enrichment status
	if noEnrich {
		skipped := db.EnrichmentSkipped
		svc.UpdateTaskFields(context.Background(), task.ID, db.TaskFieldUpdate{
			EnrichmentStatus: &skipped,
		})
		task.EnrichmentStatus = skipped
	} else {
		pending := db.EnrichmentPending
		svc.UpdateTaskFields(context.Background(), task.ID, db.TaskFieldUpdate{
			EnrichmentStatus: &pending,
		})
		task.EnrichmentStatus = pending
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

	if taskOutputJSON && task != nil {
		return json.NewEncoder(os.Stdout).Encode(task)
	}

	fmt.Printf("Moved task to %s\n", newStatus)
	return nil
}

// taskGetResponse extends Task with dependencies and comments for JSON output.
type taskGetResponse struct {
	db.Task
	Dependencies []string     `json:"dependencies"`
	Comments     []db.Comment `json:"comments"`
}

func runTaskGet(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := openService()
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := context.Background()

	// Find by prefix
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

	// Populate dependency data (blocked-by)
	allDeps, depsErr := svc.ListAllDependencies(ctx)
	if depsErr == nil && allDeps != nil {
		if blockers, ok := allDeps[task.ID]; ok {
			task.BlockedBy = blockers
		}
	}

	if taskOutputJSON {
		// Include dependencies and comments in JSON output
		deps, _ := svc.ListDependencies(ctx, task.ID)
		if deps == nil {
			deps = []string{}
		}
		comments, _ := svc.ListComments(ctx, task.ID)
		if comments == nil {
			comments = []db.Comment{}
		}
		resp := taskGetResponse{
			Task:         *task,
			Dependencies: deps,
			Comments:     comments,
		}
		return json.NewEncoder(os.Stdout).Encode(resp)
	}

	fmt.Printf("ID:          %s\n", task.ID)
	fmt.Printf("Title:       %s\n", task.Title)
	fmt.Printf("Status:      %s\n", task.Status)
	fmt.Printf("Assignee:    %s\n", task.Assignee)
	fmt.Printf("Agent:       %s (%s)\n", task.AgentName, task.AgentStatus)
	fmt.Printf("Branch:      %s\n", task.BranchName)
	fmt.Printf("PR:          %s\n", task.PRUrl)
	if task.EnrichmentStatus != "" {
		fmt.Printf("Enrichment:  %s\n", task.EnrichmentStatus)
	}
	if len(task.BlockedBy) > 0 {
		var shortIDs []string
		for _, id := range task.BlockedBy {
			if len(id) > 8 {
				shortIDs = append(shortIDs, id[:8])
			} else {
				shortIDs = append(shortIDs, id)
			}
		}
		fmt.Printf("Blocked by:  %s\n", strings.Join(shortIDs, ", "))
	}
	fmt.Printf("Description: %s\n", task.Description)
	fmt.Printf("Created:     %s\n", task.CreatedAt.Format("2006-01-02 15:04"))

	// Show dependencies
	deps, _ := svc.ListDependencies(ctx, task.ID)
	if len(deps) > 0 {
		fmt.Printf("Depends on:  ")
		for i, d := range deps {
			if i > 0 {
				fmt.Printf(", ")
			}
			fmt.Printf("%s", d[:8])
		}
		fmt.Println()
	}

	// Show comments
	comments, _ := svc.ListComments(ctx, task.ID)
	if len(comments) > 0 {
		fmt.Println("\nComments:")
		for _, c := range comments {
			fmt.Printf("  [%s] %s: %s\n", c.CreatedAt.Format("15:04"), c.Author, c.Body)
		}
	}

	return nil
}

func runTaskDelete(cmd *cobra.Command, args []string) error {
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

	task, _ := svc.GetTask(ctx, fullID)

	if err := svc.DeleteTask(ctx, fullID); err != nil {
		return err
	}

	if taskOutputJSON && task != nil {
		return json.NewEncoder(os.Stdout).Encode(map[string]string{"deleted": fullID})
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

	ctx := context.Background()
	tasks, err := svc.ListTasks(ctx)
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

	if err := svc.ClaimTask(ctx, fullID, user); err != nil {
		return err
	}

	if taskOutputJSON {
		task, _ := svc.GetTask(ctx, fullID)
		if task != nil {
			return json.NewEncoder(os.Stdout).Encode(task)
		}
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

	ctx := context.Background()
	tasks, err := svc.ListTasks(ctx)
	if err != nil {
		return err
	}
	fullID := findByPrefix(tasks, args[0])
	if fullID == "" {
		return fmt.Errorf("task not found: %s", args[0])
	}

	if err := svc.UnclaimTask(ctx, fullID); err != nil {
		return err
	}

	if taskOutputJSON {
		task, _ := svc.GetTask(ctx, fullID)
		if task != nil {
			return json.NewEncoder(os.Stdout).Encode(task)
		}
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

	// Build partial update from explicitly-set flags
	var update db.TaskFieldUpdate
	if cmd.Flags().Changed("title") {
		if strings.TrimSpace(updateTitle) == "" {
			return fmt.Errorf("title cannot be empty")
		}
		update.Title = &updateTitle
	}
	if cmd.Flags().Changed("description") {
		update.Description = &updateDescription
	}
	if cmd.Flags().Changed("assignee") {
		update.Assignee = &updateAssignee
	}
	if cmd.Flags().Changed("branch") {
		update.BranchName = &updateBranch
	}
	if cmd.Flags().Changed("pr-url") {
		update.PRUrl = &updatePRUrl
	}
	if cmd.Flags().Changed("enrichment-status") {
		es := db.EnrichmentStatus(updateEnrichmentStatus)
		if !es.Valid() {
			return fmt.Errorf("invalid enrichment status: %s", updateEnrichmentStatus)
		}
		update.EnrichmentStatus = &es
	}

	if err := svc.UpdateTaskFields(ctx, fullID, update); err != nil {
		return err
	}

	// Handle dependency changes
	if cmd.Flags().Changed("add-dep") {
		depID := findByPrefix(tasks, updateAddDep)
		if depID == "" {
			return fmt.Errorf("dependency task not found: %s", updateAddDep)
		}
		if err := svc.AddDependency(ctx, fullID, depID); err != nil {
			return err
		}
	}
	if cmd.Flags().Changed("remove-dep") {
		depID := findByPrefix(tasks, updateRemoveDep)
		if depID == "" {
			return fmt.Errorf("dependency task not found: %s", updateRemoveDep)
		}
		if err := svc.RemoveDependency(ctx, fullID, depID); err != nil {
			return err
		}
	}

	task, _ := svc.GetTask(ctx, fullID)
	if taskOutputJSON && task != nil {
		return json.NewEncoder(os.Stdout).Encode(task)
	}

	fmt.Println("Task updated")
	return nil
}

func runTaskComment(cmd *cobra.Command, args []string) error {
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

	comment, err := svc.AddComment(ctx, fullID, commentAuthor, commentBody)
	if err != nil {
		return err
	}

	if taskOutputJSON {
		return json.NewEncoder(os.Stdout).Encode(comment)
	}

	fmt.Printf("Comment added by %s\n", comment.Author)
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

	if taskOutputJSON {
		return json.NewEncoder(os.Stdout).Encode(map[string]string{"task_id": taskID, "blocked_by": blockerID})
	}

	fmt.Printf("Task %s blocked by %s\n", taskID[:8], blockerID[:8])
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

	if taskOutputJSON {
		return json.NewEncoder(os.Stdout).Encode(map[string]string{"task_id": taskID, "unblocked": blockerID})
	}

	fmt.Printf("Dependency removed: %s no longer blocked by %s\n", taskID[:8], blockerID[:8])
	return nil
}

func runTaskSuggest(cmd *cobra.Command, args []string) error {
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

	sug, err := svc.CreateSuggestion(ctx, fullID, db.SuggestionHint, suggestAuthor, suggestTitle, suggestMessage)
	if err != nil {
		return err
	}

	if taskOutputJSON {
		return json.NewEncoder(os.Stdout).Encode(sug)
	}

	fmt.Printf("Suggestion created: %s\n", sug.ID[:8])
	return nil
}

func runTaskPropose(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := openService()
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := context.Background()
	message := proposeDescription
	if proposeReason != "" {
		message = proposeDescription + "\n\nReason: " + proposeReason
	}

	sug, err := svc.CreateSuggestion(ctx, "", db.SuggestionProposal, "", proposeTitle, message)
	if err != nil {
		return err
	}

	if taskOutputJSON {
		return json.NewEncoder(os.Stdout).Encode(sug)
	}

	fmt.Printf("Proposal created: %s (%s)\n", proposeTitle, sug.ID[:8])
	return nil
}

func runTaskSuggestions(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := openService()
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := context.Background()
	status := db.SuggestionStatus(suggestionsStatus)
	if !status.Valid() {
		return fmt.Errorf("invalid suggestion status: %s", suggestionsStatus)
	}

	suggestions, err := svc.ListSuggestions(ctx, status)
	if err != nil {
		return err
	}

	if taskOutputJSON {
		if suggestions == nil {
			suggestions = []db.Suggestion{}
		}
		return json.NewEncoder(os.Stdout).Encode(suggestions)
	}

	if len(suggestions) == 0 {
		fmt.Printf("No %s suggestions\n", status)
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTYPE\tTITLE\tTASK\tAUTHOR")
	for _, s := range suggestions {
		taskRef := ""
		if s.TaskID != "" && len(s.TaskID) >= 8 {
			taskRef = s.TaskID[:8]
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
			s.ID[:8], s.Type, s.Title, taskRef, s.Author)
	}
	return w.Flush()
}

func runTaskSuggestionAccept(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := openService()
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := context.Background()
	if err := svc.AcceptSuggestion(ctx, args[0]); err != nil {
		return err
	}

	if taskOutputJSON {
		sug, _ := svc.GetSuggestion(ctx, args[0])
		if sug != nil {
			return json.NewEncoder(os.Stdout).Encode(sug)
		}
	}

	fmt.Println("Suggestion accepted")
	return nil
}

func runTaskSuggestionDismiss(cmd *cobra.Command, args []string) error {
	svc, cleanup, err := openService()
	if err != nil {
		return err
	}
	defer cleanup()

	ctx := context.Background()
	if err := svc.DismissSuggestion(ctx, args[0]); err != nil {
		return err
	}

	if taskOutputJSON {
		sug, _ := svc.GetSuggestion(ctx, args[0])
		if sug != nil {
			return json.NewEncoder(os.Stdout).Encode(sug)
		}
	}

	fmt.Println("Suggestion dismissed")
	return nil
}

func filterTasksBySearch(tasks []db.Task, q string) []db.Task {
	q = strings.ToLower(q)
	var out []db.Task
	for _, t := range tasks {
		if strings.Contains(strings.ToLower(t.Title), q) ||
			strings.Contains(strings.ToLower(t.Description), q) {
			out = append(out, t)
		}
	}
	return out
}

func findByPrefix(tasks []db.Task, prefix string) string {
	for _, t := range tasks {
		if len(t.ID) >= len(prefix) && t.ID[:len(prefix)] == prefix {
			return t.ID
		}
	}
	return ""
}
