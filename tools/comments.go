package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rgabriel/mcp-todoist/todoist"
)

// GetCommentsHandler creates a handler for getting comments
func GetCommentsHandler(client *todoist.Client) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		// Build query parameters
		params := url.Values{}
		hasFilter := false

		if taskID, ok := args["task_id"].(string); ok && taskID != "" {
			params.Set("task_id", taskID)
			hasFilter = true
		}

		if projectID, ok := args["project_id"].(string); ok && projectID != "" {
			params.Set("project_id", projectID)
			hasFilter = true
		}

		if !hasFilter {
			return mcp.NewToolResultError("either task_id or project_id is required"), nil
		}

		// Build path with query parameters
		path := "/comments?" + params.Encode()

		// Fetch comments
		respBody, err := client.Get(ctx, path)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to get comments: %v", err)), nil
		}

		// Parse response
		var comments []map[string]interface{}
		if err := json.Unmarshal(respBody, &comments); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse comments: %v", err)), nil
		}

		// Format response
		response := map[string]interface{}{
			"count":    len(comments),
			"comments": comments,
		}

		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// AddCommentHandler creates a handler for adding a new comment
func AddCommentHandler(client *todoist.Client) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
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

		// Add either task_id or project_id
		hasTarget := false
		if taskID, ok := args["task_id"].(string); ok && taskID != "" {
			body["task_id"] = taskID
			hasTarget = true
		}
		if projectID, ok := args["project_id"].(string); ok && projectID != "" {
			body["project_id"] = projectID
			hasTarget = true
		}

		if !hasTarget {
			return mcp.NewToolResultError("either task_id or project_id is required"), nil
		}

		// Create comment
		respBody, err := client.Post(ctx, "/comments", body)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to add comment: %v", err)), nil
		}

		// Parse response
		var comment map[string]interface{}
		if err := json.Unmarshal(respBody, &comment); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse response: %v", err)), nil
		}

		jsonData, err := json.MarshalIndent(comment, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// UpdateCommentHandler creates a handler for updating a comment
func UpdateCommentHandler(client *todoist.Client) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		// Extract and validate required parameters
		commentID, ok := args["comment_id"].(string)
		if !ok || commentID == "" {
			return mcp.NewToolResultError("comment_id is required"), nil
		}

		content, ok := args["content"].(string)
		if !ok || content == "" {
			return mcp.NewToolResultError("content is required"), nil
		}

		// Build request body
		body := map[string]interface{}{
			"content": content,
		}

		// Update comment
		path := fmt.Sprintf("/comments/%s", commentID)
		respBody, err := client.Post(ctx, path, body)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to update comment: %v", err)), nil
		}

		// Parse response
		var comment map[string]interface{}
		if err := json.Unmarshal(respBody, &comment); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse response: %v", err)), nil
		}

		jsonData, err := json.MarshalIndent(comment, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// DeleteCommentHandler creates a handler for deleting a comment
func DeleteCommentHandler(client *todoist.Client) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		commentID, ok := args["comment_id"].(string)
		if !ok || commentID == "" {
			return mcp.NewToolResultError("comment_id is required"), nil
		}

		// Delete comment
		path := fmt.Sprintf("/comments/%s", commentID)
		err := client.Delete(ctx, path)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to delete comment: %v", err)), nil
		}

		// Format response
		response := map[string]interface{}{
			"success":    true,
			"comment_id": commentID,
			"message":    "Comment deleted successfully",
		}

		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}
