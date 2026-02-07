package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rgabriel/mcp-todoist/todoist"
)

// ListLabelsHandler creates a handler for listing all personal labels.
func ListLabelsHandler(client todoist.API) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		respBody, err := client.Get(ctx, "/labels")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to list labels: %v", err)), nil
		}

		var labels []map[string]interface{}
		if err := json.Unmarshal(respBody, &labels); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse labels: %v", err)), nil
		}

		response := map[string]interface{}{
			"count":  len(labels),
			"labels": labels,
		}

		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// CreateLabelHandler creates a handler for creating a new label.
func CreateLabelHandler(client todoist.API) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		name, ok := args["name"].(string)
		if !ok || name == "" {
			return mcp.NewToolResultError("name is required"), nil
		}

		body := map[string]interface{}{
			"name": name,
		}

		if color, ok := args["color"].(string); ok && color != "" {
			body["color"] = color
		}
		if order, ok := args["order"].(float64); ok {
			body["order"] = int(order)
		}
		if isFavorite, ok := args["is_favorite"].(bool); ok {
			body["is_favorite"] = isFavorite
		}

		respBody, err := client.Post(ctx, "/labels", body)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to create label: %v", err)), nil
		}

		var label map[string]interface{}
		if err := json.Unmarshal(respBody, &label); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse response: %v", err)), nil
		}

		jsonData, err := json.MarshalIndent(label, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// UpdateLabelHandler creates a handler for updating a label.
func UpdateLabelHandler(client todoist.API) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		labelID, ok := args["label_id"].(string)
		if !ok || labelID == "" {
			return mcp.NewToolResultError("label_id is required"), nil
		}
		if err := ValidateID(labelID, "label_id"); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		body := map[string]interface{}{}

		if name, ok := args["name"].(string); ok && name != "" {
			body["name"] = name
		}
		if color, ok := args["color"].(string); ok && color != "" {
			body["color"] = color
		}
		if order, ok := args["order"].(float64); ok {
			body["order"] = int(order)
		}
		if isFavorite, ok := args["is_favorite"].(bool); ok {
			body["is_favorite"] = isFavorite
		}

		if len(body) == 0 {
			return mcp.NewToolResultError("at least one field to update must be provided"), nil
		}

		path := fmt.Sprintf("/labels/%s", labelID)
		respBody, err := client.Post(ctx, path, body)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to update label: %v", err)), nil
		}

		var label map[string]interface{}
		if err := json.Unmarshal(respBody, &label); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to parse response: %v", err)), nil
		}

		jsonData, err := json.MarshalIndent(label, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}

// DeleteLabelHandler creates a handler for deleting a label.
func DeleteLabelHandler(client todoist.API) func(context.Context, mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	return func(ctx context.Context, req mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := req.GetArguments()

		labelID, ok := args["label_id"].(string)
		if !ok || labelID == "" {
			return mcp.NewToolResultError("label_id is required"), nil
		}
		if err := ValidateID(labelID, "label_id"); err != nil {
			return mcp.NewToolResultError(err.Error()), nil
		}

		path := fmt.Sprintf("/labels/%s", labelID)
		err := client.Delete(ctx, path)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to delete label: %v", err)), nil
		}

		response := map[string]interface{}{
			"success":  true,
			"label_id": labelID,
			"message":  "Label deleted successfully",
		}

		jsonData, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("failed to format response: %v", err)), nil
		}

		return mcp.NewToolResultText(string(jsonData)), nil
	}
}
