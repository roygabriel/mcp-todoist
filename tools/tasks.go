package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rgabriel/mcp-todoist/todoist"
)

// SearchTasksHandler creates a handler for searching/listing tasks
func SearchTasksHandler(client *todoist.Client) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		// Build query parameters
		params := url.Values{}
		
		if filter, ok := args["filter"].(string); ok && filter != "" {
			params.Set("filter", filter)
		}
		
		if projectID, ok := args["project_id"].(string); ok && projectID != "" {
			params.Set("project_id", projectID)
		}
		
		if label, ok := args["label"].(string); ok && label != "" {
			params.Set("label", label)
		}
		
		if ids, ok := args["ids"].([]interface{}); ok && len(ids) > 0 {
			idStrs := make([]string, 0, len(ids))
			for _, id := range ids {
				if idStr, ok := id.(string); ok {
					idStrs = append(idStrs, idStr)
				}
			}
			if len(idStrs) > 0 {
				params.Set("ids", strings.Join(idStrs, ","))
			}
		}

		// Build path with query parameters
		path := "/tasks"
		if len(params) > 0 {
			path += "?" + params.Encode()
		}

		// Fetch tasks
		respBody, err := client.Get(ctx, path)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to search tasks: %v", err)), nil
		}

		// Parse response
		var tasks []map[string]interface{}
		if err := json.Unmarshal(respBody, &tasks); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse tasks: %v", err)), nil
		}

		// Format response
		response := map[string]interface{}{
			"count": len(tasks),
			"tasks": tasks,
		}

		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// GetTaskHandler creates a handler for getting a single task
func GetTaskHandler(client *todoist.Client) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		taskID, ok := args["task_id"].(string)
		if !ok || taskID == "" {
			return mcp.NewToolResultError("task_id is required"), nil
		}

		// Fetch task
		path := fmt.Sprintf("/tasks/%s", taskID)
		respBody, err := client.Get(ctx, path)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get task: %v", err)), nil
		}

		// Parse and format response
		var task map[string]interface{}
		if err := json.Unmarshal(respBody, &task); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse task: %v", err)), nil
		}

		jsonData, err := json.MarshalIndent(task, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// CreateTaskHandler creates a handler for creating a new task
func CreateTaskHandler(client *todoist.Client) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		// Extract and validate required parameters
		content, ok := args["content"].(string)
		if !ok || content == "" {
			return mcp.NewToolResultError("content is required"), nil
		}

		// Build request body
		body := map[string]interface{}{
			"content": content,
		}

		// Add optional parameters
		if description, ok := args["description"].(string); ok && description != "" {
			body["description"] = description
		}
		if projectID, ok := args["project_id"].(string); ok && projectID != "" {
			body["project_id"] = projectID
		}
		if sectionID, ok := args["section_id"].(string); ok && sectionID != "" {
			body["section_id"] = sectionID
		}
		if parentID, ok := args["parent_id"].(string); ok && parentID != "" {
			body["parent_id"] = parentID
		}
		if order, ok := args["order"].(float64); ok {
			body["order"] = int(order)
		}
		if labels, ok := args["labels"].([]interface{}); ok && len(labels) > 0 {
			labelStrs := make([]string, 0, len(labels))
			for _, l := range labels {
				if labelStr, ok := l.(string); ok {
					labelStrs = append(labelStrs, labelStr)
				}
			}
			if len(labelStrs) > 0 {
				body["labels"] = labelStrs
			}
		}
		if priority, ok := args["priority"].(float64); ok {
			p := int(priority)
			if p < 1 || p > 4 {
				return mcp.NewToolResultError("priority must be between 1 (normal) and 4 (urgent)"), nil
			}
			body["priority"] = p
		}
		if dueString, ok := args["due_string"].(string); ok && dueString != "" {
			body["due_string"] = dueString
		}
		if dueDate, ok := args["due_date"].(string); ok && dueDate != "" {
			body["due_date"] = dueDate
		}
		if dueDatetime, ok := args["due_datetime"].(string); ok && dueDatetime != "" {
			body["due_datetime"] = dueDatetime
		}
		if assigneeID, ok := args["assignee_id"].(string); ok && assigneeID != "" {
			body["assignee_id"] = assigneeID
		}
		if duration, ok := args["duration"].(float64); ok {
			body["duration"] = int(duration)
		}
		if durationUnit, ok := args["duration_unit"].(string); ok && durationUnit != "" {
			body["duration_unit"] = durationUnit
		}
		if deadlineDate, ok := args["deadline_date"].(string); ok && deadlineDate != "" {
			body["deadline_date"] = deadlineDate
		}

		// Create task
		respBody, err := client.Post(ctx, "/tasks", body)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create task: %v", err)), nil
		}

		// Parse response
		var task map[string]interface{}
		if err := json.Unmarshal(respBody, &task); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse response: %v", err)), nil
		}

		jsonData, err := json.MarshalIndent(task, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// UpdateTaskHandler creates a handler for updating a task
func UpdateTaskHandler(client *todoist.Client) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		// Extract and validate required parameters
		taskID, ok := args["task_id"].(string)
		if !ok || taskID == "" {
			return mcp.NewToolResultError("task_id is required"), nil
		}

		// Build request body
		body := map[string]interface{}{}

		// Add optional parameters
		if content, ok := args["content"].(string); ok && content != "" {
			body["content"] = content
		}
		if description, ok := args["description"].(string); ok && description != "" {
			body["description"] = description
		}
		if labels, ok := args["labels"].([]interface{}); ok && len(labels) > 0 {
			labelStrs := make([]string, 0, len(labels))
			for _, l := range labels {
				if labelStr, ok := l.(string); ok {
					labelStrs = append(labelStrs, labelStr)
				}
			}
			if len(labelStrs) > 0 {
				body["labels"] = labelStrs
			}
		}
		if priority, ok := args["priority"].(float64); ok {
			p := int(priority)
			if p < 1 || p > 4 {
				return mcp.NewToolResultError("priority must be between 1 (normal) and 4 (urgent)"), nil
			}
			body["priority"] = p
		}
		if dueString, ok := args["due_string"].(string); ok && dueString != "" {
			body["due_string"] = dueString
		}
		if dueDate, ok := args["due_date"].(string); ok && dueDate != "" {
			body["due_date"] = dueDate
		}
		if dueDatetime, ok := args["due_datetime"].(string); ok && dueDatetime != "" {
			body["due_datetime"] = dueDatetime
		}
		if assigneeID, ok := args["assignee_id"].(string); ok && assigneeID != "" {
			body["assignee_id"] = assigneeID
		}
		if duration, ok := args["duration"].(float64); ok {
			body["duration"] = int(duration)
		}
		if durationUnit, ok := args["duration_unit"].(string); ok && durationUnit != "" {
			body["duration_unit"] = durationUnit
		}
		if deadlineDate, ok := args["deadline_date"].(string); ok && deadlineDate != "" {
			body["deadline_date"] = deadlineDate
		}

		if len(body) == 0 {
			return mcp.NewToolResultError("at least one field to update must be provided"), nil
		}

		// Update task
		path := fmt.Sprintf("/tasks/%s", taskID)
		respBody, err := client.Post(ctx, path, body)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to update task: %v", err)), nil
		}

		// Parse response
		var task map[string]interface{}
		if err := json.Unmarshal(respBody, &task); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse response: %v", err)), nil
		}

		jsonData, err := json.MarshalIndent(task, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// CompleteTaskHandler creates a handler for completing a task
func CompleteTaskHandler(client *todoist.Client) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		taskID, ok := args["task_id"].(string)
		if !ok || taskID == "" {
			return mcp.NewToolResultError("task_id is required"), nil
		}

		// Complete task
		path := fmt.Sprintf("/tasks/%s/close", taskID)
		_, err := client.Post(ctx, path, nil)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to complete task: %v", err)), nil
		}

		// Format response
		response := map[string]interface{}{
			"success": true,
			"task_id": taskID,
			"message": "Task completed successfully",
		}

		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// UncompleteTaskHandler creates a handler for reopening a task
func UncompleteTaskHandler(client *todoist.Client) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		taskID, ok := args["task_id"].(string)
		if !ok || taskID == "" {
			return mcp.NewToolResultError("task_id is required"), nil
		}

		// Reopen task
		path := fmt.Sprintf("/tasks/%s/reopen", taskID)
		_, err := client.Post(ctx, path, nil)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to reopen task: %v", err)), nil
		}

		// Format response
		response := map[string]interface{}{
			"success": true,
			"task_id": taskID,
			"message": "Task reopened successfully",
		}

		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// DeleteTaskHandler creates a handler for deleting a task
func DeleteTaskHandler(client *todoist.Client) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		taskID, ok := args["task_id"].(string)
		if !ok || taskID == "" {
			return mcp.NewToolResultError("task_id is required"), nil
		}

		// Delete task
		path := fmt.Sprintf("/tasks/%s", taskID)
		err := client.Delete(ctx, path)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to delete task: %v", err)), nil
		}

		// Format response
		response := map[string]interface{}{
			"success": true,
			"task_id": taskID,
			"message": "Task deleted successfully",
		}

		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}
