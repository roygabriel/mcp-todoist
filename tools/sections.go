package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rgabriel/mcp-todoist/todoist"
)

// ListSectionsHandler creates a handler for listing sections
func ListSectionsHandler(client *todoist.Client) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		// Build query parameters
		params := url.Values{}
		if projectID, ok := args["project_id"].(string); ok && projectID != "" {
			params.Set("project_id", projectID)
		}

		// Build path with query parameters
		path := "/sections"
		if len(params) > 0 {
			path += "?" + params.Encode()
		}

		// Fetch sections
		respBody, err := client.Get(ctx, path)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list sections: %v", err)), nil
		}

		// Parse response
		var sections []map[string]interface{}
		if err := json.Unmarshal(respBody, &sections); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse sections: %v", err)), nil
		}

		// Format response
		response := map[string]interface{}{
			"count":    len(sections),
			"sections": sections,
		}

		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// CreateSectionHandler creates a handler for creating a new section
func CreateSectionHandler(client *todoist.Client) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		// Extract and validate required parameters
		name, ok := args["name"].(string)
		if !ok || name == "" {
			return mcp.NewToolResultError("name is required"), nil
		}

		projectID, ok := args["project_id"].(string)
		if !ok || projectID == "" {
			return mcp.NewToolResultError("project_id is required"), nil
		}

		// Build request body
		body := map[string]interface{}{
			"name":       name,
			"project_id": projectID,
		}

		// Add optional parameters
		if order, ok := args["order"].(float64); ok {
			body["order"] = int(order)
		}

		// Create section
		respBody, err := client.Post(ctx, "/sections", body)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create section: %v", err)), nil
		}

		// Parse response
		var section map[string]interface{}
		if err := json.Unmarshal(respBody, &section); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse response: %v", err)), nil
		}

		jsonData, err := json.MarshalIndent(section, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// UpdateSectionHandler creates a handler for updating a section
func UpdateSectionHandler(client *todoist.Client) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		// Extract and validate required parameters
		sectionID, ok := args["section_id"].(string)
		if !ok || sectionID == "" {
			return mcp.NewToolResultError("section_id is required"), nil
		}

		name, ok := args["name"].(string)
		if !ok || name == "" {
			return mcp.NewToolResultError("name is required"), nil
		}

		// Build request body
		body := map[string]interface{}{
			"name": name,
		}

		// Update section
		path := fmt.Sprintf("/sections/%s", sectionID)
		respBody, err := client.Post(ctx, path, body)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to update section: %v", err)), nil
		}

		// Parse response
		var section map[string]interface{}
		if err := json.Unmarshal(respBody, &section); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse response: %v", err)), nil
		}

		jsonData, err := json.MarshalIndent(section, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// DeleteSectionHandler creates a handler for deleting a section
func DeleteSectionHandler(client *todoist.Client) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		sectionID, ok := args["section_id"].(string)
		if !ok || sectionID == "" {
			return mcp.NewToolResultError("section_id is required"), nil
		}

		// Delete section
		path := fmt.Sprintf("/sections/%s", sectionID)
		err := client.Delete(ctx, path)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to delete section: %v", err)), nil
		}

		// Format response
		response := map[string]interface{}{
			"success":    true,
			"section_id": sectionID,
			"message":    "Section deleted successfully",
		}

		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}
