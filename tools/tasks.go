package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rgabriel/mcp-todoist/todoist"
)

// SearchTasksHandler creates a handler for searching/listing tasks.
func SearchTasksHandler(client todoist.API) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		params := url.Values{}

		if filter, ok := args["filter"].(string); ok && filter != "" {
			params.Set("filter", filter)
		}

		if projectID, ok := args["project_id"].(string); ok && projectID != "" {
			if err := ValidateID(projectID, "project_id"); err != nil {
				return mcp.NewToolResultError(err.Error()), nil
			}
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

		path := "/tasks"
		if len(params) > 0 {
			path += "?" + params.Encode()
		}

		respBody, err := client.Get(ctx, path)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to search tasks: %v", err)), nil
		}

		var tasks []map[string]interface{}
		if err := json.Unmarshal(respBody, &tasks); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse tasks: %v", err)), nil
		}

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

// GetTaskHandler creates a handler for getting a single task.
func GetTaskHandler(client todoist.API) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		taskID, ok := args["task_id"].(string)
		if !ok || taskID == "" {
			return mcp.NewToolResultError("task_id is required"), nil
		}
		if err := ValidateID(taskID, "task_id"); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		path := fmt.Sprintf("/tasks/%s", taskID)
		respBody, err := client.Get(ctx, path)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get task: %v", err)), nil
		}

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

// CreateTaskHandler creates a handler for creating a new task.
func CreateTaskHandler(client todoist.API) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		content, ok := args["content"].(string)
		if !ok || content == "" {
			return mcp.NewToolResultError("content is required"), nil
		}

		body := map[string]interface{}{
			"content": content,
		}

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

		respBody, err := client.Post(ctx, "/tasks", body)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create task: %v", err)), nil
		}

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

// UpdateTaskHandler creates a handler for updating a task.
func UpdateTaskHandler(client todoist.API) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		taskID, ok := args["task_id"].(string)
		if !ok || taskID == "" {
			return mcp.NewToolResultError("task_id is required"), nil
		}
		if err := ValidateID(taskID, "task_id"); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		body := map[string]interface{}{}

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

		path := fmt.Sprintf("/tasks/%s", taskID)
		respBody, err := client.Post(ctx, path, body)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to update task: %v", err)), nil
		}

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

// CompleteTaskHandler creates a handler for completing a task.
func CompleteTaskHandler(client todoist.API) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		taskID, ok := args["task_id"].(string)
		if !ok || taskID == "" {
			return mcp.NewToolResultError("task_id is required"), nil
		}
		if err := ValidateID(taskID, "task_id"); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		path := fmt.Sprintf("/tasks/%s/close", taskID)
		_, err := client.Post(ctx, path, nil)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to complete task: %v", err)), nil
		}

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

// UncompleteTaskHandler creates a handler for reopening a task.
func UncompleteTaskHandler(client todoist.API) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		taskID, ok := args["task_id"].(string)
		if !ok || taskID == "" {
			return mcp.NewToolResultError("task_id is required"), nil
		}
		if err := ValidateID(taskID, "task_id"); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		path := fmt.Sprintf("/tasks/%s/reopen", taskID)
		_, err := client.Post(ctx, path, nil)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to reopen task: %v", err)), nil
		}

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

// DeleteTaskHandler creates a handler for deleting a task.
func DeleteTaskHandler(client todoist.API) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		taskID, ok := args["task_id"].(string)
		if !ok || taskID == "" {
			return mcp.NewToolResultError("task_id is required"), nil
		}
		if err := ValidateID(taskID, "task_id"); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		path := fmt.Sprintf("/tasks/%s", taskID)
		err := client.Delete(ctx, path)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to delete task: %v", err)), nil
		}

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

// QuickAddTaskHandler creates a handler for quick adding tasks with Todoist syntax.
func QuickAddTaskHandler(client todoist.API) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		content, ok := args["content"].(string)
		if !ok || content == "" {
			return mcp.NewToolResultError("content is required"), nil
		}

		// Parse project (#ProjectName)
		var projectID string
		projectRegex := regexp.MustCompile(`#(\w+)`)
		projectMatches := projectRegex.FindAllStringSubmatch(content, -1)
		if len(projectMatches) > 0 {
			projectName := projectMatches[0][1]

			respBody, err := client.Get(ctx, "/projects")
			if err == nil {
				var projects []map[string]interface{}
				if err := json.Unmarshal(respBody, &projects); err == nil {
					for _, proj := range projects {
						if name, ok := proj["name"].(string); ok {
							if strings.EqualFold(name, projectName) {
								if id, ok := proj["id"].(string); ok {
									projectID = id
									break
								}
							}
						}
					}
				}
			}
			content = projectRegex.ReplaceAllString(content, "")
		}

		// Parse labels (@label)
		var labels []string
		labelRegex := regexp.MustCompile(`@(\w+)`)
		labelMatches := labelRegex.FindAllStringSubmatch(content, -1)
		for _, match := range labelMatches {
			labels = append(labels, match[1])
		}
		content = labelRegex.ReplaceAllString(content, "")

		// Parse priority (p1-p4)
		var priority int
		priorityRegex := regexp.MustCompile(`\bp([1-4])\b`)
		priorityMatches := priorityRegex.FindStringSubmatch(content)
		if len(priorityMatches) > 0 {
			switch priorityMatches[1] {
			case "1":
				priority = 4
			case "2":
				priority = 3
			case "3":
				priority = 2
			case "4":
				priority = 1
			}
			content = priorityRegex.ReplaceAllString(content, "")
		}

		content = strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(content, " "))

		// Extract potential due date keywords
		var dueString string
		dateKeywords := []string{"tomorrow", "today", "tonight", "next week", "next month", "monday", "tuesday", "wednesday", "thursday", "friday", "saturday", "sunday", "jan", "feb", "mar", "apr", "may", "jun", "jul", "aug", "sep", "oct", "nov", "dec"}

		words := strings.Fields(content)
		dateStartIdx := -1
		for i := len(words) - 1; i >= 0; i-- {
			lowerWord := strings.ToLower(words[i])
			for _, keyword := range dateKeywords {
				if strings.Contains(lowerWord, keyword) {
					dateStartIdx = i
					break
				}
			}
			if dateStartIdx >= 0 {
				break
			}
		}

		if dateStartIdx >= 0 {
			dueString = strings.Join(words[dateStartIdx:], " ")
			content = strings.TrimSpace(strings.Join(words[:dateStartIdx], " "))
		}

		body := map[string]interface{}{
			"content": content,
		}

		if projectID != "" {
			body["project_id"] = projectID
		}
		if len(labels) > 0 {
			body["labels"] = labels
		}
		if priority > 0 {
			body["priority"] = priority
		}
		if dueString != "" {
			body["due_string"] = dueString
		}

		respBody, err := client.Post(ctx, "/tasks", body)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create task: %v", err)), nil
		}

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

// GetTaskStatsHandler creates a handler for getting task statistics.
func GetTaskStatsHandler(client todoist.API) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		tasksBody, err := client.Get(ctx, "/tasks")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to fetch tasks: %v", err)), nil
		}

		var tasks []map[string]interface{}
		if err := json.Unmarshal(tasksBody, &tasks); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse tasks: %v", err)), nil
		}

		projectsBody, err := client.Get(ctx, "/projects")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to fetch projects: %v", err)), nil
		}

		var projects []map[string]interface{}
		if err := json.Unmarshal(projectsBody, &projects); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse projects: %v", err)), nil
		}

		projectMap := make(map[string]string)
		for _, proj := range projects {
			if id, ok := proj["id"].(string); ok {
				if name, ok := proj["name"].(string); ok {
					projectMap[id] = name
				}
			}
		}

		stats := map[string]interface{}{
			"total_active": len(tasks),
			"today":        0,
			"overdue":      0,
			"by_priority": map[string]int{
				"p1": 0,
				"p2": 0,
				"p3": 0,
				"p4": 0,
			},
			"by_project": make(map[string]int),
		}

		today := time.Now().Format("2006-01-02")

		for _, task := range tasks {
			if priority, ok := task["priority"].(float64); ok {
				p := int(priority)
				switch p {
				case 4:
					stats["by_priority"].(map[string]int)["p1"]++
				case 3:
					stats["by_priority"].(map[string]int)["p2"]++
				case 2:
					stats["by_priority"].(map[string]int)["p3"]++
				case 1:
					stats["by_priority"].(map[string]int)["p4"]++
				}
			}

			if projectID, ok := task["project_id"].(string); ok {
				projectName := projectMap[projectID]
				if projectName == "" {
					projectName = "Unknown"
				}
				stats["by_project"].(map[string]int)[projectName]++
			}

			if due, ok := task["due"].(map[string]interface{}); ok {
				if dueDate, ok := due["date"].(string); ok {
					if dueDate == today {
						stats["today"] = stats["today"].(int) + 1
					} else if dueDate < today {
						stats["overdue"] = stats["overdue"].(int) + 1
					}
				}
			}
		}

		jsonData, err := json.MarshalIndent(stats, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// BulkCompleteTasksHandler creates a handler for completing multiple tasks.
func BulkCompleteTasksHandler(client todoist.API, syncClient todoist.SyncAPI) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		var taskIDs []string

		if filter, ok := args["filter"].(string); ok && filter != "" {
			params := url.Values{}
			params.Set("filter", filter)
			path := "/tasks?" + params.Encode()

			respBody, err := client.Get(ctx, path)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to fetch tasks with filter: %v", err)), nil
			}

			var tasks []map[string]interface{}
			if err := json.Unmarshal(respBody, &tasks); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to parse tasks: %v", err)), nil
			}

			for _, task := range tasks {
				if id, ok := task["id"].(string); ok {
					taskIDs = append(taskIDs, id)
				}
			}
		}

		if taskIDsParam, ok := args["task_ids"].([]interface{}); ok && len(taskIDsParam) > 0 {
			taskIDs = make([]string, 0, len(taskIDsParam))
			for _, id := range taskIDsParam {
				if idStr, ok := id.(string); ok {
					taskIDs = append(taskIDs, idStr)
				}
			}
		}

		if len(taskIDs) == 0 {
			return mcp.NewToolResultError("either task_ids or filter must be provided and match at least one task"), nil
		}

		var successCount int
		var failedTasks []string
		var usedBatching bool

		if len(taskIDs) > 5 {
			usedBatching = true

			commands := make([]todoist.Command, len(taskIDs))
			for i, taskID := range taskIDs {
				commands[i] = todoist.Command{
					Type: "item_close",
					UUID: todoist.GenerateUUID(),
					Args: map[string]interface{}{
						"id": taskID,
					},
				}
			}

			syncResp, err := syncClient.BatchCommands(ctx, commands)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to batch complete tasks: %v", err)), nil
			}

			for i, cmd := range commands {
				status := syncResp.SyncStatus[cmd.UUID]
				if statusStr, ok := status.(string); ok && statusStr == "ok" {
					successCount++
				} else {
					failedTasks = append(failedTasks, taskIDs[i])
				}
			}
		} else {
			usedBatching = false

			remaining := client.GetRemainingRequests()
			if remaining < len(taskIDs) {
				return mcp.NewToolResultError(fmt.Sprintf("insufficient rate limit capacity: need %d requests, have %d remaining in 15min window", len(taskIDs), remaining)), nil
			}

			for _, taskID := range taskIDs {
				path := fmt.Sprintf("/tasks/%s/close", taskID)
				_, err := client.Post(ctx, path, nil)
				if err != nil {
					failedTasks = append(failedTasks, taskID)
					continue
				}
				successCount++
			}
		}

		response := map[string]interface{}{
			"total_tasks":     len(taskIDs),
			"completed":       successCount,
			"failed":          len(failedTasks),
			"failed_task_ids": failedTasks,
			"used_batching":   usedBatching,
		}

		if len(failedTasks) == 0 {
			response["message"] = fmt.Sprintf("Successfully completed %d tasks", successCount)
		} else {
			response["message"] = fmt.Sprintf("Completed %d of %d tasks (%d failed)", successCount, len(taskIDs), len(failedTasks))
		}

		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// BatchCreateTasksHandler creates a handler for creating multiple tasks in one batch.
func BatchCreateTasksHandler(syncClient todoist.SyncAPI) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		tasksParam, ok := args["tasks"].([]interface{})
		if !ok || len(tasksParam) == 0 {
			return mcp.NewToolResultError("tasks array is required and must contain at least one task"), nil
		}

		commands := make([]todoist.Command, 0, len(tasksParam))
		tempIDs := make([]string, len(tasksParam))

		for i, taskParam := range tasksParam {
			taskMap, ok := taskParam.(map[string]interface{})
			if !ok {
				return mcp.NewToolResultError(fmt.Sprintf("task at index %d is not a valid object", i)), nil
			}

			content, ok := taskMap["content"].(string)
			if !ok || content == "" {
				return mcp.NewToolResultError(fmt.Sprintf("task at index %d missing required 'content' field", i)), nil
			}

			tempID := todoist.GenerateTempID()
			tempIDs[i] = tempID

			cmdArgs := map[string]interface{}{
				"content": content,
			}

			if description, ok := taskMap["description"].(string); ok && description != "" {
				cmdArgs["description"] = description
			}
			if projectID, ok := taskMap["project_id"].(string); ok && projectID != "" {
				cmdArgs["project_id"] = projectID
			}
			if sectionID, ok := taskMap["section_id"].(string); ok && sectionID != "" {
				cmdArgs["section_id"] = sectionID
			}
			if labels, ok := taskMap["labels"].([]interface{}); ok && len(labels) > 0 {
				labelStrs := make([]string, 0, len(labels))
				for _, l := range labels {
					if labelStr, ok := l.(string); ok {
						labelStrs = append(labelStrs, labelStr)
					}
				}
				if len(labelStrs) > 0 {
					cmdArgs["labels"] = labelStrs
				}
			}
			if priority, ok := taskMap["priority"].(float64); ok {
				p := int(priority)
				if p >= 1 && p <= 4 {
					cmdArgs["priority"] = p
				}
			}
			if dueString, ok := taskMap["due_string"].(string); ok && dueString != "" {
				cmdArgs["due_string"] = dueString
			}
			if dueDate, ok := taskMap["due_date"].(string); ok && dueDate != "" {
				cmdArgs["due_date"] = dueDate
			}

			if parentTempIDRef, ok := taskMap["parent_temp_id"].(string); ok && parentTempIDRef != "" {
				var parentIdx int
				if _, err := fmt.Sscanf(parentTempIDRef, "%d", &parentIdx); err == nil {
					if parentIdx >= 0 && parentIdx < len(tempIDs) && parentIdx < i {
						cmdArgs["parent_id"] = tempIDs[parentIdx]
					}
				} else {
					cmdArgs["parent_id"] = parentTempIDRef
				}
			} else if parentID, ok := taskMap["parent_id"].(string); ok && parentID != "" {
				cmdArgs["parent_id"] = parentID
			}

			commands = append(commands, todoist.Command{
				Type:   "item_add",
				UUID:   todoist.GenerateUUID(),
				TempID: tempID,
				Args:   cmdArgs,
			})
		}

		syncResp, err := syncClient.BatchCommands(ctx, commands)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to batch create tasks: %v", err)), nil
		}

		createdTasks := make([]map[string]interface{}, 0)
		failedIndices := make([]int, 0)

		for i, cmd := range commands {
			status := syncResp.SyncStatus[cmd.UUID]
			if statusStr, ok := status.(string); ok && statusStr == "ok" {
				taskInfo := map[string]interface{}{
					"index":   i,
					"temp_id": cmd.TempID,
				}
				if realID, ok := syncResp.TempIDMapping[cmd.TempID]; ok {
					taskInfo["id"] = realID
				}
				taskInfo["content"] = cmd.Args["content"]
				createdTasks = append(createdTasks, taskInfo)
			} else {
				failedIndices = append(failedIndices, i)
			}
		}

		response := map[string]interface{}{
			"total_tasks":     len(commands),
			"created":         len(createdTasks),
			"failed":          len(failedIndices),
			"failed_indices":  failedIndices,
			"created_tasks":   createdTasks,
			"temp_id_mapping": syncResp.TempIDMapping,
		}

		if len(failedIndices) == 0 {
			response["message"] = fmt.Sprintf("Successfully created %d tasks in a single batch", len(createdTasks))
		} else {
			response["message"] = fmt.Sprintf("Created %d of %d tasks (%d failed)", len(createdTasks), len(commands), len(failedIndices))
		}

		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// MoveTasksHandler creates a handler for moving multiple tasks to a different project.
func MoveTasksHandler(client todoist.API, syncClient todoist.SyncAPI) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		toProjectID, ok := args["to_project_id"].(string)
		if !ok || toProjectID == "" {
			return mcp.NewToolResultError("to_project_id is required"), nil
		}
		if err := ValidateID(toProjectID, "to_project_id"); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		var taskIDs []string

		if filter, ok := args["filter"].(string); ok && filter != "" {
			params := url.Values{}
			params.Set("filter", filter)
			path := "/tasks?" + params.Encode()

			respBody, err := client.Get(ctx, path)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to fetch tasks with filter: %v", err)), nil
			}

			var tasks []map[string]interface{}
			if err := json.Unmarshal(respBody, &tasks); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to parse tasks: %v", err)), nil
			}

			for _, task := range tasks {
				if id, ok := task["id"].(string); ok {
					taskIDs = append(taskIDs, id)
				}
			}
		}

		if taskIDsParam, ok := args["task_ids"].([]interface{}); ok && len(taskIDsParam) > 0 {
			taskIDs = make([]string, 0, len(taskIDsParam))
			for _, id := range taskIDsParam {
				if idStr, ok := id.(string); ok {
					taskIDs = append(taskIDs, idStr)
				}
			}
		}

		if len(taskIDs) == 0 {
			return mcp.NewToolResultError("either task_ids or filter must be provided and match at least one task"), nil
		}

		projectPath := fmt.Sprintf("/projects/%s", toProjectID)
		projectResp, err := client.Get(ctx, projectPath)
		var toProjectName string
		if err == nil {
			var project map[string]interface{}
			if json.Unmarshal(projectResp, &project) == nil {
				if name, ok := project["name"].(string); ok {
					toProjectName = name
				}
			}
		}
		if toProjectName == "" {
			toProjectName = toProjectID
		}

		var successCount int
		var failedTasks []string
		var usedBatching bool

		if len(taskIDs) > 5 {
			usedBatching = true

			commands := make([]todoist.Command, len(taskIDs))
			for i, taskID := range taskIDs {
				commands[i] = todoist.Command{
					Type: "item_update",
					UUID: todoist.GenerateUUID(),
					Args: map[string]interface{}{
						"id":         taskID,
						"project_id": toProjectID,
					},
				}
			}

			syncResp, err := syncClient.BatchCommands(ctx, commands)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("failed to batch move tasks: %v", err)), nil
			}

			for i, cmd := range commands {
				status := syncResp.SyncStatus[cmd.UUID]
				if statusStr, ok := status.(string); ok && statusStr == "ok" {
					successCount++
				} else {
					failedTasks = append(failedTasks, taskIDs[i])
				}
			}
		} else {
			usedBatching = false

			remaining := client.GetRemainingRequests()
			if remaining < len(taskIDs) {
				return mcp.NewToolResultError(fmt.Sprintf("insufficient rate limit capacity: need %d requests, have %d remaining in 15min window", len(taskIDs), remaining)), nil
			}

			for _, taskID := range taskIDs {
				path := fmt.Sprintf("/tasks/%s", taskID)
				body := map[string]interface{}{
					"project_id": toProjectID,
				}
				_, err := client.Post(ctx, path, body)
				if err != nil {
					failedTasks = append(failedTasks, taskID)
					continue
				}
				successCount++
			}
		}

		response := map[string]interface{}{
			"total_tasks":     len(taskIDs),
			"moved":           successCount,
			"failed":          len(failedTasks),
			"failed_task_ids": failedTasks,
			"to_project":      toProjectName,
			"used_batching":   usedBatching,
		}

		if len(failedTasks) == 0 {
			response["message"] = fmt.Sprintf("Successfully moved %d tasks to '%s'", successCount, toProjectName)
		} else {
			response["message"] = fmt.Sprintf("Moved %d of %d tasks to '%s' (%d failed)", successCount, len(taskIDs), toProjectName, len(failedTasks))
		}

		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}
