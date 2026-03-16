package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/jrogala/vikunja-cli/client"
	"github.com/jrogala/vikunja-cli/config"
	"github.com/spf13/cobra"
)

var (
	taskListProject int64
	taskShowAll     bool
	taskShowDone    bool
	taskSortBy      string
	taskSearch      string
	taskAddProject  int64
	taskPriority    int
	taskDue         string
	taskDesc        string
	editTitle       string
	editDesc        string
	editPriority    int
	editDue         string
	editUndone      bool
)

func init() {
	rootCmd.AddCommand(taskCmd)
	taskCmd.AddCommand(taskListCmd)
	taskCmd.AddCommand(taskGetCmd)
	taskCmd.AddCommand(taskAddCmd)
	taskCmd.AddCommand(taskDoneCmd)
	taskCmd.AddCommand(taskEditCmd)
	taskCmd.AddCommand(taskDeleteCmd)

	taskListCmd.Flags().Int64VarP(&taskListProject, "project", "p", 0, "filter by project ID")
	taskListCmd.Flags().BoolVarP(&taskShowAll, "all", "a", false, "include completed tasks")
	taskListCmd.Flags().BoolVar(&taskShowDone, "done", false, "only completed tasks")
	taskListCmd.Flags().StringVarP(&taskSortBy, "sort", "s", "due_date", "sort field: due_date|priority|created|updated|title")
	taskListCmd.Flags().StringVar(&taskSearch, "search", "", "search by title")

	taskEditCmd.Flags().StringVar(&editTitle, "title", "", "new title")
	taskEditCmd.Flags().StringVar(&editDesc, "description", "", "new description")
	taskEditCmd.Flags().IntVar(&editPriority, "priority", -1, "new priority 0-4")
	taskEditCmd.Flags().StringVar(&editDue, "due-date", "", "new due date YYYY-MM-DD (use 'none' to clear)")
	taskEditCmd.Flags().BoolVar(&editUndone, "undone", false, "mark task as not done")

	taskAddCmd.Flags().Int64VarP(&taskAddProject, "project", "p", 1, "target project ID (default: Inbox)")
	taskAddCmd.Flags().IntVar(&taskPriority, "priority", 0, "0=none 1=low 2=medium 3=high 4=urgent")
	taskAddCmd.Flags().StringVar(&taskDue, "due-date", "", "due date YYYY-MM-DD")
	taskAddCmd.Flags().StringVar(&taskDesc, "description", "", "task description")
}

func newClient() *client.Client {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	return client.New(cfg)
}

var taskCmd = &cobra.Command{
	Use:     "task",
	Aliases: []string{"t", "tasks"},
	Short:   "Manage tasks",
}

var taskListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls", "l"},
	Short:   "List incomplete tasks. Use --all for all, --done for completed only.",
	RunE: func(cmd *cobra.Command, args []string) error {
		c := newClient()

		params := map[string]string{
			"sort_by":  taskSortBy,
			"order_by": "asc",
		}
		if taskSearch != "" {
			params["s"] = taskSearch
		}
		if !taskShowAll && !taskShowDone {
			params["filter"] = "done = false"
		} else if taskShowDone {
			params["filter"] = "done = true"
		}

		var tasks []client.Task
		var err error
		if taskListProject > 0 {
			tasks, err = c.GetProjectTasks(taskListProject, params)
		} else {
			tasks, err = c.GetTasks(params)
		}
		if err != nil {
			return err
		}

		if outputJSON {
			return printJSON(tasks)
		}

		if len(tasks) == 0 {
			fmt.Println("No tasks found.")
			return nil
		}

		printTasks(tasks)
		return nil
	},
}

var taskGetCmd = &cobra.Command{
	Use:   "get <id>",
	Short: "Get one task by ID. Returns all fields.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid task ID: %s", args[0])
		}
		c := newClient()
		task, err := c.GetTask(id)
		if err != nil {
			return err
		}
		if outputJSON {
			return printJSON(task)
		}
		printTaskDetail(task)
		return nil
	},
}

var taskAddCmd = &cobra.Command{
	Use:   "add <title>",
	Short: "Create a task. Flags: -p project, --priority, --due-date, --description.",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		c := newClient()
		payload := map[string]any{
			"title": strings.Join(args, " "),
		}
		if taskPriority > 0 {
			payload["priority"] = taskPriority
		}
		if taskDue != "" {
			payload["due_date"] = taskDue + "T09:00:00Z"
		}
		if taskDesc != "" {
			payload["description"] = taskDesc
		}
		task, err := c.CreateTask(taskAddProject, payload)
		if err != nil {
			return err
		}
		if outputJSON {
			return printJSON(task)
		}
		fmt.Printf("Created task #%d: %s\n", task.ID, task.Title)
		return nil
	},
}

var taskDoneCmd = &cobra.Command{
	Use:   "done <id>",
	Short: "Mark task as complete by ID.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid task ID: %s", args[0])
		}
		c := newClient()
		task, err := c.CompleteTask(id)
		if err != nil {
			return err
		}
		if outputJSON {
			return printJSON(task)
		}
		fmt.Printf("Completed task #%d: %s\n", task.ID, task.Title)
		return nil
	},
}

var taskEditCmd = &cobra.Command{
	Use:   "edit <id>",
	Short: "Update a task. Flags: --title, --description, --priority, --due-date, --undone.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid task ID: %s", args[0])
		}
		updates := map[string]any{}
		if editTitle != "" {
			updates["title"] = editTitle
		}
		if editDesc != "" {
			updates["description"] = editDesc
		}
		if editPriority >= 0 {
			updates["priority"] = editPriority
		}
		if editDue == "none" {
			updates["due_date"] = "0001-01-01T00:00:00Z"
		} else if editDue != "" {
			updates["due_date"] = editDue + "T09:00:00Z"
		}
		if editUndone {
			updates["done"] = false
		}
		if len(updates) == 0 {
			return fmt.Errorf("no changes specified")
		}
		c := newClient()
		task, err := c.UpdateTask(id, updates)
		if err != nil {
			return err
		}
		if outputJSON {
			return printJSON(task)
		}
		fmt.Printf("Updated task #%d: %s\n", task.ID, task.Title)
		return nil
	},
}

var taskDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Permanently delete task by ID.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			return fmt.Errorf("invalid task ID: %s", args[0])
		}
		c := newClient()
		if err := c.DeleteTask(id); err != nil {
			return err
		}
		if outputJSON {
			return printJSON(map[string]any{"deleted": id})
		}
		fmt.Printf("Deleted task #%d\n", id)
		return nil
	},
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func priorityStr(p int) string {
	switch p {
	case 1:
		return "LOW"
	case 2:
		return "MED"
	case 3:
		return "HIGH"
	case 4:
		return "URGENT"
	default:
		return "-"
	}
}

func formatDue(due string) string {
	if due == "" || strings.HasPrefix(due, "0001") {
		return "-"
	}
	t, err := time.Parse(time.RFC3339, due)
	if err != nil {
		return due[:10]
	}
	now := time.Now()
	diff := time.Until(t)
	switch {
	case diff < -24*time.Hour:
		return t.Format("Jan 02") + " (overdue)"
	case diff < 0:
		return "today (overdue)"
	case diff < 24*time.Hour:
		return "today"
	case diff < 48*time.Hour:
		return "tomorrow"
	case t.Year() == now.Year():
		return t.Format("Jan 02")
	default:
		return t.Format("2006-01-02")
	}
}

func printTasks(tasks []client.Task) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tDONE\tPRIORITY\tDUE\tTITLE")
	for _, t := range tasks {
		done := " "
		if t.Done {
			done = "x"
		}
		fmt.Fprintf(w, "%d\t[%s]\t%s\t%s\t%s\n",
			t.ID, done, priorityStr(t.Priority), formatDue(t.DueDate), t.Title)
	}
	w.Flush()
}

func printTaskDetail(t *client.Task) {
	fmt.Printf("Task #%d\n", t.ID)
	fmt.Printf("  Title:       %s\n", t.Title)
	fmt.Printf("  Done:        %v\n", t.Done)
	fmt.Printf("  Priority:    %s\n", priorityStr(t.Priority))
	fmt.Printf("  Due:         %s\n", formatDue(t.DueDate))
	fmt.Printf("  Project:     %d\n", t.ProjectID)
	if t.Description != "" {
		fmt.Printf("  Description: %s\n", t.Description)
	}
}
