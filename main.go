package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rgabriel/mcp-todoist/config"
	"github.com/rgabriel/mcp-todoist/todoist"
	"github.com/rgabriel/mcp-todoist/tools"
)

var version = "dev"

func setupLogger() {
	level := slog.LevelInfo
	switch strings.ToUpper(os.Getenv("LOG_LEVEL")) {
	case "DEBUG":
		level = slog.LevelDebug
	case "WARN":
		level = slog.LevelWarn
	case "ERROR":
		level = slog.LevelError
	}
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	slog.SetDefault(slog.New(handler))
}

// timeoutMiddleware wraps every tool handler with a context deadline.
func timeoutMiddleware(d time.Duration) server.ToolHandlerMiddleware {
	return func(next server.ToolHandlerFunc) server.ToolHandlerFunc {
		return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
			ctx, cancel := context.WithTimeout(ctx, d)
			defer cancel()
			return next(ctx, req)
		}
	}
}

func main() {
	setupLogger()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration error", "error", err)
		os.Exit(1)
	}

	// Shared rate limiter for both REST and Sync clients
	rl := todoist.NewRateLimiter(15*time.Minute, 450)
	todoistClient := todoist.NewClient(cfg.TodoistAPIToken, rl)
	todoistSyncClient := todoist.NewSyncClient(cfg.TodoistAPIToken, rl)

	ctx := context.Background()
	if err := todoistClient.TestConnection(ctx); err != nil {
		slog.Error("failed to connect to Todoist API", "error", err)
		os.Exit(1)
	}

	s := server.NewMCPServer(
		"Todoist Server",
		version,
		server.WithToolCapabilities(false),
		server.WithRecovery(),
		server.WithToolHandlerMiddleware(timeoutMiddleware(30*time.Second)),
	)

	// ── Task tools ──────────────────────────────────────────────────────

	s.AddTool(mcp.NewTool("search_tasks",
		mcp.WithDescription("Search and list active tasks. Supports Todoist filter syntax, project filtering, label filtering, and fetching by IDs. Returns an array of task objects with id, content, description, project_id, priority, due, labels, and url. Use list_projects first to get valid project_id values."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("filter",
			mcp.Description("Todoist filter query (e.g., 'today', 'p1', 'overdue', '@label', '#project', 'today & p1'). See https://todoist.com/help/articles/introduction-to-filters-V98wIH"),
		),
		mcp.WithString("project_id",
			mcp.Description("Filter tasks by project ID. Use list_projects to discover valid IDs."),
		),
		mcp.WithString("label",
			mcp.Description("Filter tasks by label name. Use list_labels to discover valid names."),
		),
		mcp.WithArray("ids",
			mcp.Description("Fetch specific tasks by their IDs."),
		),
	), tools.SearchTasksHandler(todoistClient))

	s.AddTool(mcp.NewTool("get_task",
		mcp.WithDescription("Get a single task by ID with full details including content, description, project_id, section_id, priority (1-4), labels, due date, assignee, duration, and URL."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("task_id",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("Task ID to retrieve. Use search_tasks to find task IDs."),
		),
	), tools.GetTaskHandler(todoistClient))

	s.AddTool(mcp.NewTool("create_task",
		mcp.WithDescription("Create a new task. Returns the created task object with its assigned ID. Use list_projects and list_sections to get valid project_id/section_id values. Priority uses Todoist's internal scale: 1=normal, 4=urgent."),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("content",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("Task title/content."),
		),
		mcp.WithString("description",
			mcp.Description("Task description (supports markdown)."),
		),
		mcp.WithString("project_id",
			mcp.Description("Project ID to add task to. Use list_projects to find IDs."),
		),
		mcp.WithString("section_id",
			mcp.Description("Section ID within project. Use list_sections to find IDs."),
		),
		mcp.WithString("parent_id",
			mcp.Description("Parent task ID for creating sub-tasks."),
		),
		mcp.WithNumber("order",
			mcp.Description("Task order position."),
		),
		mcp.WithArray("labels",
			mcp.Description("Array of label names to apply."),
		),
		mcp.WithNumber("priority",
			mcp.Description("Priority: 1 (normal), 2, 3, or 4 (urgent/p1)."),
			mcp.Min(1),
			mcp.Max(4),
			mcp.DefaultNumber(1),
		),
		mcp.WithString("due_string",
			mcp.Description("Natural language due date (e.g., 'tomorrow at 3pm', 'every monday', 'next friday')."),
		),
		mcp.WithString("due_date",
			mcp.Description("Due date in YYYY-MM-DD format (e.g., '2025-12-31')."),
			mcp.Pattern(`^\d{4}-\d{2}-\d{2}$`),
		),
		mcp.WithString("due_datetime",
			mcp.Description("Due date and time in RFC 3339 format (e.g., '2025-12-31T14:00:00Z')."),
		),
		mcp.WithString("assignee_id",
			mcp.Description("User ID to assign task to (for shared projects)."),
		),
		mcp.WithNumber("duration",
			mcp.Description("Task duration amount. Requires duration_unit."),
			mcp.Min(1),
		),
		mcp.WithString("duration_unit",
			mcp.Description("Duration unit."),
			mcp.Enum("minute", "day"),
		),
		mcp.WithString("deadline_date",
			mcp.Description("Deadline date in YYYY-MM-DD format."),
			mcp.Pattern(`^\d{4}-\d{2}-\d{2}$`),
		),
	), tools.CreateTaskHandler(todoistClient))

	s.AddTool(mcp.NewTool("update_task",
		mcp.WithDescription("Update an existing task. Only provided fields are changed; omitted fields keep their current values. Returns the updated task object."),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("task_id",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("Task ID to update. Use search_tasks to find task IDs."),
		),
		mcp.WithString("content",
			mcp.Description("New task title/content."),
		),
		mcp.WithString("description",
			mcp.Description("New task description (supports markdown)."),
		),
		mcp.WithArray("labels",
			mcp.Description("New array of label names (replaces existing labels)."),
		),
		mcp.WithNumber("priority",
			mcp.Description("New priority: 1 (normal) to 4 (urgent)."),
			mcp.Min(1),
			mcp.Max(4),
		),
		mcp.WithString("due_string",
			mcp.Description("New natural language due date."),
		),
		mcp.WithString("due_date",
			mcp.Description("New due date in YYYY-MM-DD format."),
			mcp.Pattern(`^\d{4}-\d{2}-\d{2}$`),
		),
		mcp.WithString("due_datetime",
			mcp.Description("New due date and time in RFC 3339 format."),
		),
		mcp.WithString("assignee_id",
			mcp.Description("New assignee user ID."),
		),
		mcp.WithNumber("duration",
			mcp.Description("New task duration amount."),
			mcp.Min(1),
		),
		mcp.WithString("duration_unit",
			mcp.Description("New duration unit."),
			mcp.Enum("minute", "day"),
		),
		mcp.WithString("deadline_date",
			mcp.Description("New deadline date in YYYY-MM-DD format."),
			mcp.Pattern(`^\d{4}-\d{2}-\d{2}$`),
		),
	), tools.UpdateTaskHandler(todoistClient))

	s.AddTool(mcp.NewTool("complete_task",
		mcp.WithDescription("Mark a task as completed. For recurring tasks, this advances to the next occurrence. Returns success confirmation with the task_id."),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("task_id",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("Task ID to complete. Use search_tasks to find task IDs."),
		),
	), tools.CompleteTaskHandler(todoistClient))

	s.AddTool(mcp.NewTool("uncomplete_task",
		mcp.WithDescription("Reopen a previously completed task. Returns success confirmation with the task_id."),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("task_id",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("Task ID to reopen."),
		),
	), tools.UncompleteTaskHandler(todoistClient))

	s.AddTool(mcp.NewTool("delete_task",
		mcp.WithDescription("Permanently delete a task. This cannot be undone. Use complete_task instead if you want to mark it done. Returns success confirmation."),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("task_id",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("Task ID to delete. Use search_tasks to find task IDs."),
		),
	), tools.DeleteTaskHandler(todoistClient))

	s.AddTool(mcp.NewTool("quick_add_task",
		mcp.WithDescription("Quick-add a task using Todoist inline syntax. Parses #project, @label, p1-p4 priority, and date keywords from the content string. Example: 'Buy milk #Shopping @groceries p1 tomorrow'. Returns the created task."),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("content",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("Task content with inline syntax: #ProjectName @label p1-p4 and date keywords."),
		),
	), tools.QuickAddTaskHandler(todoistClient))

	s.AddTool(mcp.NewTool("get_task_stats",
		mcp.WithDescription("Get aggregate statistics about all active tasks. Returns total_active count, today count, overdue count, breakdown by_priority (p1-p4), and breakdown by_project (project name to count)."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	), tools.GetTaskStatsHandler(todoistClient))

	s.AddTool(mcp.NewTool("bulk_complete_tasks",
		mcp.WithDescription("Complete multiple tasks at once by IDs or filter. Uses Sync API batching for >5 tasks (single request) or REST API for <=5 tasks. Returns completed/failed counts and used_batching flag."),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithArray("task_ids",
			mcp.Description("Array of task IDs to complete. Overrides filter if both provided."),
		),
		mcp.WithString("filter",
			mcp.Description("Todoist filter to select tasks to complete (e.g., 'today & p1')."),
		),
	), tools.BulkCompleteTasksHandler(todoistClient, todoistSyncClient))

	s.AddTool(mcp.NewTool("batch_create_tasks",
		mcp.WithDescription("Create multiple tasks in a single Sync API request. Supports parent-child relationships via parent_temp_id (use array index of parent task). Returns created_tasks with real IDs and temp_id_mapping."),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithArray("tasks",
			mcp.Required(),
			mcp.Description("Array of task objects. Each must have 'content' (string). Optional: description, project_id, section_id, labels, priority (1-4), due_string, due_date, parent_id, parent_temp_id (index of parent in this array)."),
		),
	), tools.BatchCreateTasksHandler(todoistSyncClient))

	s.AddTool(mcp.NewTool("move_tasks",
		mcp.WithDescription("Move multiple tasks to a different project. Uses Sync API batching for >5 tasks. Provide either task_ids or a filter to select tasks. Returns moved/failed counts and destination project name."),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithArray("task_ids",
			mcp.Description("Array of task IDs to move. Overrides filter if both provided."),
		),
		mcp.WithString("filter",
			mcp.Description("Todoist filter to select tasks to move."),
		),
		mcp.WithString("to_project_id",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("Destination project ID. Use list_projects to find valid IDs."),
		),
	), tools.MoveTasksHandler(todoistClient, todoistSyncClient))

	// ── Project tools ───────────────────────────────────────────────────

	s.AddTool(mcp.NewTool("list_projects",
		mcp.WithDescription("List all projects. Returns each project's id, name, color, parent_id, order, is_favorite, is_inbox_project, is_team_inbox, and view_style. Use the id field as project_id in other tools."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	), tools.ListProjectsHandler(todoistClient))

	s.AddTool(mcp.NewTool("create_project",
		mcp.WithDescription("Create a new project. Returns the created project object with its assigned ID."),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("name",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("Project name."),
		),
		mcp.WithString("parent_id",
			mcp.Description("Parent project ID for creating sub-projects."),
		),
		mcp.WithString("color",
			mcp.Description("Project color."),
			mcp.Enum("berry_red", "red", "orange", "yellow", "olive_green", "lime_green", "green", "mint_green", "teal", "sky_blue", "light_blue", "blue", "grape", "violet", "lavender", "magenta", "salmon", "charcoal", "grey", "taupe"),
		),
		mcp.WithBoolean("is_favorite",
			mcp.Description("Whether project is a favorite."),
			mcp.DefaultBool(false),
		),
		mcp.WithString("view_style",
			mcp.Description("Project view style."),
			mcp.Enum("list", "board"),
			mcp.DefaultString("list"),
		),
	), tools.CreateProjectHandler(todoistClient))

	s.AddTool(mcp.NewTool("get_project",
		mcp.WithDescription("Get a single project by ID with full details including name, color, parent_id, order, is_favorite, and view_style."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("project_id",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("Project ID to retrieve. Use list_projects to find IDs."),
		),
	), tools.GetProjectHandler(todoistClient))

	s.AddTool(mcp.NewTool("update_project",
		mcp.WithDescription("Update an existing project. Only provided fields are changed. Returns the updated project object."),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("project_id",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("Project ID to update."),
		),
		mcp.WithString("name",
			mcp.Description("New project name."),
		),
		mcp.WithString("color",
			mcp.Description("New project color."),
			mcp.Enum("berry_red", "red", "orange", "yellow", "olive_green", "lime_green", "green", "mint_green", "teal", "sky_blue", "light_blue", "blue", "grape", "violet", "lavender", "magenta", "salmon", "charcoal", "grey", "taupe"),
		),
		mcp.WithBoolean("is_favorite",
			mcp.Description("Whether project is a favorite."),
		),
		mcp.WithString("view_style",
			mcp.Description("New view style."),
			mcp.Enum("list", "board"),
		),
	), tools.UpdateProjectHandler(todoistClient))

	s.AddTool(mcp.NewTool("delete_project",
		mcp.WithDescription("Permanently delete a project and all its tasks. This cannot be undone. Returns success confirmation."),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("project_id",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("Project ID to delete. Use list_projects to find IDs."),
		),
	), tools.DeleteProjectHandler(todoistClient))

	// ── Section tools ───────────────────────────────────────────────────

	s.AddTool(mcp.NewTool("list_sections",
		mcp.WithDescription("List sections, optionally filtered by project. Returns each section's id, name, project_id, and order. Use the id field as section_id in create_task."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("project_id",
			mcp.Description("Filter sections by project ID. Use list_projects to find IDs."),
		),
	), tools.ListSectionsHandler(todoistClient))

	s.AddTool(mcp.NewTool("create_section",
		mcp.WithDescription("Create a new section in a project. Returns the created section object."),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("name",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("Section name."),
		),
		mcp.WithString("project_id",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("Project ID to create section in. Use list_projects to find IDs."),
		),
		mcp.WithNumber("order",
			mcp.Description("Section order position."),
		),
	), tools.CreateSectionHandler(todoistClient))

	s.AddTool(mcp.NewTool("update_section",
		mcp.WithDescription("Rename a section. Returns the updated section object."),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("section_id",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("Section ID to update. Use list_sections to find IDs."),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("New section name."),
		),
	), tools.UpdateSectionHandler(todoistClient))

	s.AddTool(mcp.NewTool("delete_section",
		mcp.WithDescription("Permanently delete a section and move its tasks to the parent project. This cannot be undone. Returns success confirmation."),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("section_id",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("Section ID to delete. Use list_sections to find IDs."),
		),
	), tools.DeleteSectionHandler(todoistClient))

	// ── Label tools ─────────────────────────────────────────────────────

	s.AddTool(mcp.NewTool("list_labels",
		mcp.WithDescription("List all personal labels. Returns each label's id, name, color, order, and is_favorite. Use the name field in create_task's labels array."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
	), tools.ListLabelsHandler(todoistClient))

	s.AddTool(mcp.NewTool("create_label",
		mcp.WithDescription("Create a new personal label. Returns the created label object."),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("name",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("Label name."),
		),
		mcp.WithString("color",
			mcp.Description("Label color."),
			mcp.Enum("berry_red", "red", "orange", "yellow", "olive_green", "lime_green", "green", "mint_green", "teal", "sky_blue", "light_blue", "blue", "grape", "violet", "lavender", "magenta", "salmon", "charcoal", "grey", "taupe"),
		),
		mcp.WithNumber("order",
			mcp.Description("Label order position."),
		),
		mcp.WithBoolean("is_favorite",
			mcp.Description("Whether label is a favorite."),
			mcp.DefaultBool(false),
		),
	), tools.CreateLabelHandler(todoistClient))

	s.AddTool(mcp.NewTool("update_label",
		mcp.WithDescription("Update a personal label. Only provided fields are changed. Returns the updated label object."),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("label_id",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("Label ID to update. Use list_labels to find IDs."),
		),
		mcp.WithString("name",
			mcp.Description("New label name."),
		),
		mcp.WithString("color",
			mcp.Description("New label color."),
			mcp.Enum("berry_red", "red", "orange", "yellow", "olive_green", "lime_green", "green", "mint_green", "teal", "sky_blue", "light_blue", "blue", "grape", "violet", "lavender", "magenta", "salmon", "charcoal", "grey", "taupe"),
		),
		mcp.WithNumber("order",
			mcp.Description("New label order position."),
		),
		mcp.WithBoolean("is_favorite",
			mcp.Description("Whether label is a favorite."),
		),
	), tools.UpdateLabelHandler(todoistClient))

	s.AddTool(mcp.NewTool("delete_label",
		mcp.WithDescription("Permanently delete a personal label. Tasks with this label will have it removed. This cannot be undone. Returns success confirmation."),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("label_id",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("Label ID to delete. Use list_labels to find IDs."),
		),
	), tools.DeleteLabelHandler(todoistClient))

	// ── Comment tools ───────────────────────────────────────────────────

	s.AddTool(mcp.NewTool("get_comments",
		mcp.WithDescription("Get comments for a task or project. Provide either task_id or project_id. Returns an array of comment objects with id, content, posted_at, and attachment fields."),
		mcp.WithReadOnlyHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("task_id",
			mcp.Description("Task ID to get comments for. Use search_tasks to find IDs."),
		),
		mcp.WithString("project_id",
			mcp.Description("Project ID to get comments for. Use list_projects to find IDs."),
		),
	), tools.GetCommentsHandler(todoistClient))

	s.AddTool(mcp.NewTool("add_comment",
		mcp.WithDescription("Add a comment to a task or project. Provide content and either task_id or project_id. Returns the created comment object."),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(false),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("content",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("Comment content (supports markdown)."),
		),
		mcp.WithString("task_id",
			mcp.Description("Task ID to comment on."),
		),
		mcp.WithString("project_id",
			mcp.Description("Project ID to comment on."),
		),
	), tools.AddCommentHandler(todoistClient))

	s.AddTool(mcp.NewTool("update_comment",
		mcp.WithDescription("Update the content of an existing comment. Returns the updated comment object."),
		mcp.WithDestructiveHintAnnotation(false),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("comment_id",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("Comment ID to update."),
		),
		mcp.WithString("content",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("New comment content (supports markdown)."),
		),
	), tools.UpdateCommentHandler(todoistClient))

	s.AddTool(mcp.NewTool("delete_comment",
		mcp.WithDescription("Permanently delete a comment. This cannot be undone. Returns success confirmation."),
		mcp.WithDestructiveHintAnnotation(true),
		mcp.WithIdempotentHintAnnotation(true),
		mcp.WithOpenWorldHintAnnotation(true),
		mcp.WithString("comment_id",
			mcp.Required(),
			mcp.MinLength(1),
			mcp.Description("Comment ID to delete."),
		),
	), tools.DeleteCommentHandler(todoistClient))

	slog.Info("server starting",
		"version", version,
		"tools", 29,
		"rate_limit", "450/15min",
	)

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
