package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rgabriel/mcp-todoist/todoist"
)

// ListProjectsHandler creates a handler for listing all projects
func ListProjectsHandler(client *todoist.Client) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		// Fetch projects
		respBody, err := client.Get(ctx, "/projects")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list projects: %v", err)), nil
		}

		// Parse response
		var projects []map[string]interface{}
		if err := json.Unmarshal(respBody, &projects); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse projects: %v", err)), nil
		}

		// Format response
		response := map[string]interface{}{
			"count":    len(projects),
			"projects": projects,
		}

		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// CreateProjectHandler creates a handler for creating a new project
func CreateProjectHandler(client *todoist.Client) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		// Extract and validate required parameters
		name, ok := args["name"].(string)
		if !ok || name == "" {
			return mcp.NewToolResultError("name is required"), nil
		}

		// Build request body
		body := map[string]interface{}{
			"name": name,
		}

		// Add optional parameters
		if parentID, ok := args["parent_id"].(string); ok && parentID != "" {
			body["parent_id"] = parentID
		}
		if color, ok := args["color"].(string); ok && color != "" {
			body["color"] = color
		}
		if isFavorite, ok := args["is_favorite"].(bool); ok {
			body["is_favorite"] = isFavorite
		}
		if viewStyle, ok := args["view_style"].(string); ok && viewStyle != "" {
			body["view_style"] = viewStyle
		}

		// Create project
		respBody, err := client.Post(ctx, "/projects", body)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create project: %v", err)), nil
		}

		// Parse response
		var project map[string]interface{}
		if err := json.Unmarshal(respBody, &project); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse response: %v", err)), nil
		}

		jsonData, err := json.MarshalIndent(project, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// GetProjectHandler creates a handler for getting a single project
func GetProjectHandler(client *todoist.Client) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		projectID, ok := args["project_id"].(string)
		if !ok || projectID == "" {
			return mcp.NewToolResultError("project_id is required"), nil
		}

		// Fetch project
		path := fmt.Sprintf("/projects/%s", projectID)
		respBody, err := client.Get(ctx, path)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get project: %v", err)), nil
		}

		// Parse and format response
		var project map[string]interface{}
		if err := json.Unmarshal(respBody, &project); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse project: %v", err)), nil
		}

		jsonData, err := json.MarshalIndent(project, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// UpdateProjectHandler creates a handler for updating a project
func UpdateProjectHandler(client *todoist.Client) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		// Extract and validate required parameters
		projectID, ok := args["project_id"].(string)
		if !ok || projectID == "" {
			return mcp.NewToolResultError("project_id is required"), nil
		}

		// Build request body
		body := map[string]interface{}{}

		// Add optional parameters
		if name, ok := args["name"].(string); ok && name != "" {
			body["name"] = name
		}
		if color, ok := args["color"].(string); ok && color != "" {
			body["color"] = color
		}
		if isFavorite, ok := args["is_favorite"].(bool); ok {
			body["is_favorite"] = isFavorite
		}
		if viewStyle, ok := args["view_style"].(string); ok && viewStyle != "" {
			body["view_style"] = viewStyle
		}

		if len(body) == 0 {
			return mcp.NewToolResultError("at least one field to update must be provided"), nil
		}

		// Update project
		path := fmt.Sprintf("/projects/%s", projectID)
		respBody, err := client.Post(ctx, path, body)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to update project: %v", err)), nil
		}

		// Parse response
		var project map[string]interface{}
		if err := json.Unmarshal(respBody, &project); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse response: %v", err)), nil
		}

		jsonData, err := json.MarshalIndent(project, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// DeleteProjectHandler creates a handler for deleting a project
func DeleteProjectHandler(client *todoist.Client) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		projectID, ok := args["project_id"].(string)
		if !ok || projectID == "" {
			return mcp.NewToolResultError("project_id is required"), nil
		}

		// Delete project
		path := fmt.Sprintf("/projects/%s", projectID)
		err := client.Delete(ctx, path)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to delete project: %v", err)), nil
		}

		// Format response
		response := map[string]interface{}{
			"success":    true,
			"project_id": projectID,
			"message":    "Project deleted successfully",
		}

		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}
