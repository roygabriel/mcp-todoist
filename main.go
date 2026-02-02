package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/rgabriel/mcp-todoist/config"
	"github.com/rgabriel/mcp-todoist/todoist"
	"github.com/rgabriel/mcp-todoist/tools"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	// Create Todoist client
	todoistClient := todoist.NewClient(cfg.TodoistAPIToken)

	// Test connection
	ctx := context.Background()
	if err := todoistClient.TestConnection(ctx); err != nil {
		log.Fatalf("Failed to connect to Todoist API: %v", err)
	}

	// Create MCP server
	s := server.NewMCPServer(
		"Todoist Server",
		"1.0.0",
		server.WithToolCapabilities(false),
		server.WithRecovery(),
	)

	// Register task tools
	searchTasksTool := mcp.NewTool("search_tasks",
		mcp.WithDescription("Search and list tasks with optional filters (filter syntax, project_id, label, or ids)"),
		mcp.WithString("filter",
			mcp.Description("Todoist filter syntax (e.g., 'today', 'p1', 'overdue', '@label', '#project', 'today & p1')"),
		),
		mcp.WithString("project_id",
			mcp.Description("Filter tasks by project ID"),
		),
		mcp.WithString("label",
			mcp.Description("Filter tasks by label name"),
		),
		mcp.WithArray("ids",
			mcp.Description("Get specific tasks by IDs (comma-separated list)"),
		),
	)
	s.AddTool(searchTasksTool, tools.SearchTasksHandler(todoistClient))

	getTaskTool := mcp.NewTool("get_task",
		mcp.WithDescription("Get a single task by ID with full details"),
		mcp.WithString("task_id",
			mcp.Required(),
			mcp.Description("Task ID to retrieve"),
		),
	)
	s.AddTool(getTaskTool, tools.GetTaskHandler(todoistClient))

	createTaskTool := mcp.NewTool("create_task",
		mcp.WithDescription("Create a new task with optional due dates, priority, labels, and other properties"),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("Task title/content"),
		),
		mcp.WithString("description",
			mcp.Description("Task description (markdown supported)"),
		),
		mcp.WithString("project_id",
			mcp.Description("Project ID to add task to"),
		),
		mcp.WithString("section_id",
			mcp.Description("Section ID within project"),
		),
		mcp.WithString("parent_id",
			mcp.Description("Parent task ID (for sub-tasks)"),
		),
		mcp.WithNumber("order",
			mcp.Description("Task order"),
		),
		mcp.WithArray("labels",
			mcp.Description("Array of label names"),
		),
		mcp.WithNumber("priority",
			mcp.Description("Priority from 1 (normal) to 4 (urgent/p1)"),
		),
		mcp.WithString("due_string",
			mcp.Description("Natural language due date (e.g., 'tomorrow at 3pm', 'every monday')"),
		),
		mcp.WithString("due_date",
			mcp.Description("Due date in YYYY-MM-DD format"),
		),
		mcp.WithString("due_datetime",
			mcp.Description("Due date and time in RFC3339 format"),
		),
		mcp.WithString("assignee_id",
			mcp.Description("User ID to assign task to (for shared projects)"),
		),
		mcp.WithNumber("duration",
			mcp.Description("Task duration amount (requires duration_unit)"),
		),
		mcp.WithString("duration_unit",
			mcp.Description("Duration unit: 'minute' or 'day'"),
		),
		mcp.WithString("deadline_date",
			mcp.Description("Deadline date in YYYY-MM-DD format"),
		),
	)
	s.AddTool(createTaskTool, tools.CreateTaskHandler(todoistClient))

	updateTaskTool := mcp.NewTool("update_task",
		mcp.WithDescription("Update an existing task"),
		mcp.WithString("task_id",
			mcp.Required(),
			mcp.Description("Task ID to update"),
		),
		mcp.WithString("content",
			mcp.Description("New task title/content"),
		),
		mcp.WithString("description",
			mcp.Description("New task description"),
		),
		mcp.WithArray("labels",
			mcp.Description("New array of label names"),
		),
		mcp.WithNumber("priority",
			mcp.Description("New priority from 1 (normal) to 4 (urgent)"),
		),
		mcp.WithString("due_string",
			mcp.Description("New natural language due date"),
		),
		mcp.WithString("due_date",
			mcp.Description("New due date in YYYY-MM-DD format"),
		),
		mcp.WithString("due_datetime",
			mcp.Description("New due date and time in RFC3339 format"),
		),
		mcp.WithString("assignee_id",
			mcp.Description("New assignee user ID"),
		),
		mcp.WithNumber("duration",
			mcp.Description("New task duration amount"),
		),
		mcp.WithString("duration_unit",
			mcp.Description("New duration unit: 'minute' or 'day'"),
		),
		mcp.WithString("deadline_date",
			mcp.Description("New deadline date in YYYY-MM-DD format"),
		),
	)
	s.AddTool(updateTaskTool, tools.UpdateTaskHandler(todoistClient))

	completeTaskTool := mcp.NewTool("complete_task",
		mcp.WithDescription("Mark a task as completed"),
		mcp.WithString("task_id",
			mcp.Required(),
			mcp.Description("Task ID to complete"),
		),
	)
	s.AddTool(completeTaskTool, tools.CompleteTaskHandler(todoistClient))

	uncompleteTaskTool := mcp.NewTool("uncomplete_task",
		mcp.WithDescription("Reopen a completed task"),
		mcp.WithString("task_id",
			mcp.Required(),
			mcp.Description("Task ID to reopen"),
		),
	)
	s.AddTool(uncompleteTaskTool, tools.UncompleteTaskHandler(todoistClient))

	deleteTaskTool := mcp.NewTool("delete_task",
		mcp.WithDescription("Delete a task"),
		mcp.WithString("task_id",
			mcp.Required(),
			mcp.Description("Task ID to delete"),
		),
	)
	s.AddTool(deleteTaskTool, tools.DeleteTaskHandler(todoistClient))

	// Quick add task tool
	quickAddTaskTool := mcp.NewTool("quick_add_task",
		mcp.WithDescription("Quick add task using Todoist syntax: #project @label p1-p4 due date"),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("Task content with inline syntax: 'Buy milk #Shopping @groceries p1 tomorrow'"),
		),
	)
	s.AddTool(quickAddTaskTool, tools.QuickAddTaskHandler(todoistClient))

	// Get task stats tool
	getTaskStatsTool := mcp.NewTool("get_task_stats",
		mcp.WithDescription("Get aggregate statistics about tasks (by project, priority, today, overdue)"),
	)
	s.AddTool(getTaskStatsTool, tools.GetTaskStatsHandler(todoistClient))

	// Bulk complete tasks tool
	bulkCompleteTasksTool := mcp.NewTool("bulk_complete_tasks",
		mcp.WithDescription("Complete multiple tasks by IDs or filter string (respects rate limits)"),
		mcp.WithArray("task_ids",
			mcp.Description("Array of task IDs to complete"),
		),
		mcp.WithString("filter",
			mcp.Description("Todoist filter to select tasks to complete (e.g., 'today & p1')"),
		),
	)
	s.AddTool(bulkCompleteTasksTool, tools.BulkCompleteTasksHandler(todoistClient))

	// Register project tools
	listProjectsTool := mcp.NewTool("list_projects",
		mcp.WithDescription("List all projects"),
	)
	s.AddTool(listProjectsTool, tools.ListProjectsHandler(todoistClient))

	createProjectTool := mcp.NewTool("create_project",
		mcp.WithDescription("Create a new project"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Project name"),
		),
		mcp.WithString("parent_id",
			mcp.Description("Parent project ID (for sub-projects)"),
		),
		mcp.WithString("color",
			mcp.Description("Project color (e.g., 'red', 'blue', 'green')"),
		),
		mcp.WithBoolean("is_favorite",
			mcp.Description("Whether project is a favorite"),
		),
		mcp.WithString("view_style",
			mcp.Description("View style: 'list' or 'board'"),
		),
	)
	s.AddTool(createProjectTool, tools.CreateProjectHandler(todoistClient))

	getProjectTool := mcp.NewTool("get_project",
		mcp.WithDescription("Get a single project by ID"),
		mcp.WithString("project_id",
			mcp.Required(),
			mcp.Description("Project ID to retrieve"),
		),
	)
	s.AddTool(getProjectTool, tools.GetProjectHandler(todoistClient))

	updateProjectTool := mcp.NewTool("update_project",
		mcp.WithDescription("Update an existing project"),
		mcp.WithString("project_id",
			mcp.Required(),
			mcp.Description("Project ID to update"),
		),
		mcp.WithString("name",
			mcp.Description("New project name"),
		),
		mcp.WithString("color",
			mcp.Description("New project color"),
		),
		mcp.WithBoolean("is_favorite",
			mcp.Description("Whether project is a favorite"),
		),
		mcp.WithString("view_style",
			mcp.Description("New view style: 'list' or 'board'"),
		),
	)
	s.AddTool(updateProjectTool, tools.UpdateProjectHandler(todoistClient))

	deleteProjectTool := mcp.NewTool("delete_project",
		mcp.WithDescription("Delete a project"),
		mcp.WithString("project_id",
			mcp.Required(),
			mcp.Description("Project ID to delete"),
		),
	)
	s.AddTool(deleteProjectTool, tools.DeleteProjectHandler(todoistClient))

	// Register section tools
	listSectionsTool := mcp.NewTool("list_sections",
		mcp.WithDescription("List sections, optionally filtered by project"),
		mcp.WithString("project_id",
			mcp.Description("Filter sections by project ID"),
		),
	)
	s.AddTool(listSectionsTool, tools.ListSectionsHandler(todoistClient))

	createSectionTool := mcp.NewTool("create_section",
		mcp.WithDescription("Create a new section in a project"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Section name"),
		),
		mcp.WithString("project_id",
			mcp.Required(),
			mcp.Description("Project ID to create section in"),
		),
		mcp.WithNumber("order",
			mcp.Description("Section order"),
		),
	)
	s.AddTool(createSectionTool, tools.CreateSectionHandler(todoistClient))

	updateSectionTool := mcp.NewTool("update_section",
		mcp.WithDescription("Update a section name"),
		mcp.WithString("section_id",
			mcp.Required(),
			mcp.Description("Section ID to update"),
		),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("New section name"),
		),
	)
	s.AddTool(updateSectionTool, tools.UpdateSectionHandler(todoistClient))

	deleteSectionTool := mcp.NewTool("delete_section",
		mcp.WithDescription("Delete a section"),
		mcp.WithString("section_id",
			mcp.Required(),
			mcp.Description("Section ID to delete"),
		),
	)
	s.AddTool(deleteSectionTool, tools.DeleteSectionHandler(todoistClient))

	// Register label tools
	listLabelsTool := mcp.NewTool("list_labels",
		mcp.WithDescription("List all personal labels"),
	)
	s.AddTool(listLabelsTool, tools.ListLabelsHandler(todoistClient))

	createLabelTool := mcp.NewTool("create_label",
		mcp.WithDescription("Create a new personal label"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Label name"),
		),
		mcp.WithString("color",
			mcp.Description("Label color (e.g., 'red', 'blue', 'green')"),
		),
		mcp.WithNumber("order",
			mcp.Description("Label order"),
		),
		mcp.WithBoolean("is_favorite",
			mcp.Description("Whether label is a favorite"),
		),
	)
	s.AddTool(createLabelTool, tools.CreateLabelHandler(todoistClient))

	updateLabelTool := mcp.NewTool("update_label",
		mcp.WithDescription("Update a personal label"),
		mcp.WithString("label_id",
			mcp.Required(),
			mcp.Description("Label ID to update"),
		),
		mcp.WithString("name",
			mcp.Description("New label name"),
		),
		mcp.WithString("color",
			mcp.Description("New label color"),
		),
		mcp.WithNumber("order",
			mcp.Description("New label order"),
		),
		mcp.WithBoolean("is_favorite",
			mcp.Description("Whether label is a favorite"),
		),
	)
	s.AddTool(updateLabelTool, tools.UpdateLabelHandler(todoistClient))

	deleteLabelTool := mcp.NewTool("delete_label",
		mcp.WithDescription("Delete a personal label"),
		mcp.WithString("label_id",
			mcp.Required(),
			mcp.Description("Label ID to delete"),
		),
	)
	s.AddTool(deleteLabelTool, tools.DeleteLabelHandler(todoistClient))

	// Register comment tools
	getCommentsTool := mcp.NewTool("get_comments",
		mcp.WithDescription("Get comments for a task or project"),
		mcp.WithString("task_id",
			mcp.Description("Task ID to get comments for"),
		),
		mcp.WithString("project_id",
			mcp.Description("Project ID to get comments for"),
		),
	)
	s.AddTool(getCommentsTool, tools.GetCommentsHandler(todoistClient))

	addCommentTool := mcp.NewTool("add_comment",
		mcp.WithDescription("Add a comment to a task or project"),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("Comment content (markdown supported)"),
		),
		mcp.WithString("task_id",
			mcp.Description("Task ID to comment on"),
		),
		mcp.WithString("project_id",
			mcp.Description("Project ID to comment on"),
		),
	)
	s.AddTool(addCommentTool, tools.AddCommentHandler(todoistClient))

	updateCommentTool := mcp.NewTool("update_comment",
		mcp.WithDescription("Update a comment"),
		mcp.WithString("comment_id",
			mcp.Required(),
			mcp.Description("Comment ID to update"),
		),
		mcp.WithString("content",
			mcp.Required(),
			mcp.Description("New comment content"),
		),
	)
	s.AddTool(updateCommentTool, tools.UpdateCommentHandler(todoistClient))

	deleteCommentTool := mcp.NewTool("delete_comment",
		mcp.WithDescription("Delete a comment"),
		mcp.WithString("comment_id",
			mcp.Required(),
			mcp.Description("Comment ID to delete"),
		),
	)
	s.AddTool(deleteCommentTool, tools.DeleteCommentHandler(todoistClient))

	// Log startup
	fmt.Fprintf(os.Stderr, "Todoist MCP Server v1.0.0 starting...\n")
	fmt.Fprintf(os.Stderr, "Connected to Todoist API successfully\n")
	fmt.Fprintf(os.Stderr, "Rate limit: 450 requests per 15 minutes\n")
	fmt.Fprintf(os.Stderr, "Registered %d tools\n", 27)

	// Start the stdio server
	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
