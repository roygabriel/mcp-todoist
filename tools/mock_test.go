package tools

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/rgabriel/mcp-todoist/todoist"
)

// MockAPI implements todoist.API for testing.
type MockAPI struct {
	GetFn                  func(ctx context.Context, path string) ([]byte, error)
	PostFn                 func(ctx context.Context, path string, body interface{}) ([]byte, error)
	DeleteFn               func(ctx context.Context, path string) error
	TestConnectionFn       func(ctx context.Context) error
	GetRemainingRequestsFn func() int
}

func (m *MockAPI) Get(ctx context.Context, path string) ([]byte, error) {
	if m.GetFn != nil {
		return m.GetFn(ctx, path)
	}
	return nil, fmt.Errorf("Get not configured")
}

func (m *MockAPI) Post(ctx context.Context, path string, body interface{}) ([]byte, error) {
	if m.PostFn != nil {
		return m.PostFn(ctx, path, body)
	}
	return nil, fmt.Errorf("Post not configured")
}

func (m *MockAPI) Delete(ctx context.Context, path string) error {
	if m.DeleteFn != nil {
		return m.DeleteFn(ctx, path)
	}
	return fmt.Errorf("Delete not configured")
}

func (m *MockAPI) TestConnection(ctx context.Context) error {
	if m.TestConnectionFn != nil {
		return m.TestConnectionFn(ctx)
	}
	return nil
}

func (m *MockAPI) GetRemainingRequests() int {
	if m.GetRemainingRequestsFn != nil {
		return m.GetRemainingRequestsFn()
	}
	return 450
}

// MockSyncAPI implements todoist.SyncAPI for testing.
type MockSyncAPI struct {
	BatchCommandsFn        func(ctx context.Context, commands []todoist.Command) (*todoist.SyncResponse, error)
	GetRemainingRequestsFn func() int
}

func (m *MockSyncAPI) BatchCommands(ctx context.Context, commands []todoist.Command) (*todoist.SyncResponse, error) {
	if m.BatchCommandsFn != nil {
		return m.BatchCommandsFn(ctx, commands)
	}
	return nil, fmt.Errorf("BatchCommands not configured")
}

func (m *MockSyncAPI) GetRemainingRequests() int {
	if m.GetRemainingRequestsFn != nil {
		return m.GetRemainingRequestsFn()
	}
	return 450
}

// makeReq creates a CallToolRequest with the given arguments.
func makeReq(args map[string]interface{}) mcp.CallToolRequest {
	return mcp.CallToolRequest{
		Params: mcp.CallToolParams{
			Arguments: args,
		},
	}
}

// resultText extracts the text from a CallToolResult.
func resultText(r *mcp.CallToolResult) string {
	if r != nil && len(r.Content) > 0 {
		if tc, ok := r.Content[0].(mcp.TextContent); ok {
			return tc.Text
		}
	}
	return ""
}
