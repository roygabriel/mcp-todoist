package todoist

import "context"

// API defines the interface for the Todoist REST API client.
type API interface {
	Get(ctx context.Context, path string) ([]byte, error)
	Post(ctx context.Context, path string, body interface{}) ([]byte, error)
	Delete(ctx context.Context, path string) error
	TestConnection(ctx context.Context) error
	GetRemainingRequests() int
}

// SyncAPI defines the interface for the Todoist Sync API client.
type SyncAPI interface {
	BatchCommands(ctx context.Context, commands []Command) (*SyncResponse, error)
	GetRemainingRequests() int
}
