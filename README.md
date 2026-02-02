# Todoist MCP Server

A Model Context Protocol (MCP) server that connects to Todoist using their official REST API v2. This server enables AI assistants like Claude to interact with your Todoist account - manage tasks, projects, sections, labels, and comments.

Built with Go using the official [mcp-go SDK](https://mcp-go.dev) and works on all operating systems (Linux, Windows, macOS).

## Features

- **Task Management** - Create, update, complete, reopen, delete, and search tasks with filters
- **Projects** - List, create, update, and delete projects with support for sub-projects
- **Sections** - Organize tasks within projects using sections
- **Labels** - Create and manage personal labels for task organization
- **Comments** - Add, update, and delete comments on tasks and projects
- **Natural Language Dates** - Use natural language for due dates ("tomorrow at 3pm", "every monday")
- **Advanced Filters** - Search tasks using Todoist's powerful filter syntax
- **Priority Management** - Set task priorities from p1 (urgent) to p4 (normal)
- **Rate Limiting** - Built-in rate limit tracking (450 requests per 15 minutes)
- **Cross-Platform** - Works on Linux, Windows, and macOS
- **Secure** - Uses API tokens (never your password)

## Prerequisites

- **Go 1.21 or higher** - [Install Go](https://go.dev/doc/install)
- **Todoist Account** - Free or premium account at [todoist.com](https://todoist.com)
- **API Token** - Personal API token from your Todoist account (see setup below)

## Getting Your Todoist API Token

To use this MCP server, you need a personal API token from Todoist:

1. Go to [Todoist Settings > Integrations](https://todoist.com/prefs/integrations)
2. Scroll down to the **Developer** section
3. Copy your **API token** (it looks like: `0123456789abcdef0123456789abcdef01234567`)
4. Save this token securely - you'll need it for configuration

**Important Notes:**
- Your API token provides full access to your Todoist account
- Never share your API token or commit it to version control
- You can regenerate your token at any time if it's compromised

## Installation

### From Source

```bash
# Clone the repository
git clone https://github.com/rgabriel/mcp-todoist.git
cd mcp-todoist

# Build the server
go build -o mcp-todoist

# Optional: Install to your PATH
go install
```

### Using go install

```bash
go install github.com/rgabriel/mcp-todoist@latest
```

## Configuration

Create a `.env` file in the same directory as the server executable (for local testing):

```bash
cp .env.example .env
```

Edit `.env` and add your API token:

```bash
TODOIST_API_TOKEN=your_api_token_here
```

**Environment Variables:**

- `TODOIST_API_TOKEN` (required) - Your Todoist API token from https://todoist.com/prefs/integrations

## Usage with Claude Desktop

Add this server to your Claude Desktop configuration file:

### macOS

Edit `~/Library/Application Support/Claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "todoist": {
      "command": "/path/to/mcp-todoist",
      "env": {
        "TODOIST_API_TOKEN": "your_api_token_here"
      }
    }
  }
}
```

### Windows

Edit `%APPDATA%\Claude\claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "todoist": {
      "command": "C:\\path\\to\\mcp-todoist.exe",
      "env": {
        "TODOIST_API_TOKEN": "your_api_token_here"
      }
    }
  }
}
```

### Linux

Edit `~/.config/claude/claude_desktop_config.json`:

```json
{
  "mcpServers": {
    "todoist": {
      "command": "/path/to/mcp-todoist",
      "env": {
        "TODOIST_API_TOKEN": "your_api_token_here"
      }
    }
  }
}
```

After adding the configuration, restart Claude Desktop.

## Available Tools

### Task Management

#### 1. search_tasks

Search and list tasks with optional filters.

**Parameters:**
- `filter` (optional) - Todoist filter syntax (see Filter Examples below)
- `project_id` (optional) - Filter by specific project ID
- `label` (optional) - Filter by label name
- `ids` (optional) - Array of task IDs to retrieve

**Example:**
```json
{
  "filter": "today & p1"
}
```

**Example Response:**
```json
{
  "count": 2,
  "tasks": [
    {
      "id": "7654321",
      "content": "Buy groceries",
      "description": "",
      "project_id": "2203306141",
      "priority": 4,
      "due": {
        "date": "2026-02-02",
        "string": "today"
      }
    }
  ]
}
```

#### 2. get_task

Get full details for a single task.

**Parameters:**
- `task_id` (required) - Task ID to retrieve

**Example:**
```json
{
  "task_id": "7654321"
}
```

#### 3. create_task

Create a new task.

**Parameters:**
- `content` (required) - Task title
- `description` (optional) - Task description (markdown supported)
- `project_id` (optional) - Project ID
- `section_id` (optional) - Section ID within project
- `parent_id` (optional) - Parent task ID (for sub-tasks)
- `order` (optional) - Task order
- `labels` (optional) - Array of label names
- `priority` (optional) - Priority from 1 (normal) to 4 (urgent/p1)
- `due_string` (optional) - Natural language due date
- `due_date` (optional) - Due date in YYYY-MM-DD format
- `due_datetime` (optional) - Due date and time in RFC3339 format
- `assignee_id` (optional) - User ID to assign (for shared projects)
- `duration` (optional) - Duration amount
- `duration_unit` (optional) - Duration unit: "minute" or "day"
- `deadline_date` (optional) - Deadline in YYYY-MM-DD format

**Example:**
```json
{
  "content": "Finish project proposal",
  "description": "Include budget and timeline",
  "priority": 4,
  "due_string": "tomorrow at 2pm",
  "labels": ["work", "urgent"]
}
```

#### 4. update_task

Update an existing task.

**Parameters:**
- `task_id` (required) - Task ID to update
- All other parameters from create_task (optional)

**Example:**
```json
{
  "task_id": "7654321",
  "content": "Updated task title",
  "priority": 3
}
```

#### 5. complete_task

Mark a task as completed.

**Parameters:**
- `task_id` (required) - Task ID to complete

**Example:**
```json
{
  "task_id": "7654321"
}
```

#### 6. uncomplete_task

Reopen a completed task.

**Parameters:**
- `task_id` (required) - Task ID to reopen

#### 7. delete_task

Delete a task permanently.

**Parameters:**
- `task_id` (required) - Task ID to delete

### Projects

#### 8. list_projects

List all projects.

**Parameters:** None

**Example Response:**
```json
{
  "count": 3,
  "projects": [
    {
      "id": "2203306141",
      "name": "Work",
      "color": "blue",
      "is_favorite": true,
      "view_style": "list"
    }
  ]
}
```

#### 9. create_project

Create a new project.

**Parameters:**
- `name` (required) - Project name
- `parent_id` (optional) - Parent project ID (for sub-projects)
- `color` (optional) - Project color (e.g., "red", "blue", "green")
- `is_favorite` (optional) - Whether project is a favorite
- `view_style` (optional) - View style: "list" or "board"

**Example:**
```json
{
  "name": "Personal Projects",
  "color": "green",
  "is_favorite": true,
  "view_style": "list"
}
```

#### 10. get_project

Get details for a single project.

**Parameters:**
- `project_id` (required) - Project ID to retrieve

#### 11. update_project

Update an existing project.

**Parameters:**
- `project_id` (required) - Project ID to update
- All other parameters from create_project (optional)

#### 12. delete_project

Delete a project and all its tasks.

**Parameters:**
- `project_id` (required) - Project ID to delete

### Sections

#### 13. list_sections

List sections, optionally filtered by project.

**Parameters:**
- `project_id` (optional) - Filter by project ID

#### 14. create_section

Create a new section in a project.

**Parameters:**
- `name` (required) - Section name
- `project_id` (required) - Project ID
- `order` (optional) - Section order

#### 15. update_section

Update a section name.

**Parameters:**
- `section_id` (required) - Section ID to update
- `name` (required) - New section name

#### 16. delete_section

Delete a section.

**Parameters:**
- `section_id` (required) - Section ID to delete

### Labels

#### 17. list_labels

List all personal labels.

**Parameters:** None

#### 18. create_label

Create a new personal label.

**Parameters:**
- `name` (required) - Label name
- `color` (optional) - Label color
- `order` (optional) - Label order
- `is_favorite` (optional) - Whether label is a favorite

#### 19. update_label

Update a personal label.

**Parameters:**
- `label_id` (required) - Label ID to update
- All other parameters from create_label (optional)

#### 20. delete_label

Delete a personal label.

**Parameters:**
- `label_id` (required) - Label ID to delete

### Comments

#### 21. get_comments

Get comments for a task or project.

**Parameters:**
- `task_id` (optional) - Task ID to get comments for
- `project_id` (optional) - Project ID to get comments for

Note: Either `task_id` or `project_id` is required.

#### 22. add_comment

Add a comment to a task or project.

**Parameters:**
- `content` (required) - Comment content (markdown supported)
- `task_id` (optional) - Task ID to comment on
- `project_id` (optional) - Project ID to comment on

Note: Either `task_id` or `project_id` is required.

#### 23. update_comment

Update a comment.

**Parameters:**
- `comment_id` (required) - Comment ID to update
- `content` (required) - New comment content

#### 24. delete_comment

Delete a comment.

**Parameters:**
- `comment_id` (required) - Comment ID to delete

## Todoist-Specific Features

### Natural Language Date Parsing

Use the `due_string` parameter with natural language:

- `"today"` - Due today
- `"tomorrow"` - Due tomorrow
- `"next monday"` - Due next Monday
- `"tomorrow at 3pm"` - Due tomorrow at 3:00 PM
- `"every day"` - Recurring daily
- `"every monday at 9am"` - Recurring weekly on Monday at 9:00 AM
- `"every 2 weeks"` - Recurring every 2 weeks
- `"jan 23"` - Due January 23rd

### Priority Mapping

Todoist uses priority levels 1-4:

- **Priority 4 (p1)** - Urgent (red flag)
- **Priority 3 (p2)** - High (orange flag)
- **Priority 2 (p3)** - Medium (yellow flag)
- **Priority 1 (p4)** - Normal (no flag) - default

When creating or updating tasks, use numbers 1-4. The API accepts both the numeric value and understands the p1-p4 notation in filter strings.

### Filter Syntax Examples

Use these filters with the `search_tasks` tool:

**Date-based filters:**
- `"today"` - Tasks due today
- `"tomorrow"` - Tasks due tomorrow
- `"overdue"` - Overdue tasks
- `"7 days"` - Tasks due in the next 7 days
- `"no date"` - Tasks without a due date

**Priority filters:**
- `"p1"` - Priority 1 (urgent) tasks
- `"p2"` - Priority 2 (high) tasks
- `"p3"` - Priority 3 (medium) tasks
- `"p4"` - Priority 4 (normal) tasks

**Label filters:**
- `"@shopping"` - Tasks with "shopping" label
- `"@work"` - Tasks with "work" label

**Project filters:**
- `"#Work"` - Tasks in "Work" project
- `"#Personal"` - Tasks in "Personal" project

**Combined filters:**
- `"today & p1"` - Urgent tasks due today
- `"overdue & @work"` - Overdue work tasks
- `"7 days & p1"` - Urgent tasks due in the next 7 days
- `"#Work & !assigned to: others"` - Work tasks assigned to me

**Search operators:**
- `"search: meeting"` - Tasks containing "meeting"
- `"created: today"` - Tasks created today
- `"assigned to: me"` - Tasks assigned to me

For more filter examples, see the [Todoist filters documentation](https://todoist.com/help/articles/introduction-to-filters-V98wIH).

## Rate Limiting

The Todoist API has a rate limit of **450 requests per 15 minutes**. This MCP server automatically tracks your requests and will return an error if you approach the limit.

**Tips to stay within limits:**
- Use filters to narrow down search results instead of fetching all tasks
- Batch operations when possible
- Avoid polling for updates frequently

If you hit the rate limit, wait for the 15-minute window to reset before making more requests.

## Development

### Running Locally

```bash
# Set environment variables
export TODOIST_API_TOKEN="your_api_token_here"

# Run the server
go run main.go
```

### Building

```bash
# Build for your current platform
go build -o mcp-todoist

# Build for specific platforms
GOOS=linux GOARCH=amd64 go build -o mcp-todoist-linux
GOOS=darwin GOARCH=arm64 go build -o mcp-todoist-macos
GOOS=windows GOARCH=amd64 go build -o mcp-todoist-windows.exe
```

### Testing with MCP Inspector

Use the [MCP Inspector](https://github.com/modelcontextprotocol/inspector) to test the server:

```bash
npx @modelcontextprotocol/inspector mcp-todoist
```

## Troubleshooting

### Authentication Failed

**Problem:** "authentication failed: invalid API token"

**Solutions:**
- Verify you're using the correct API token from https://todoist.com/prefs/integrations
- Check that you copied the entire token (no spaces or extra characters)
- Regenerate a new token if the old one was revoked
- Ensure the token is set correctly in your environment variables

### Rate Limit Exceeded

**Problem:** "rate limit exceeded: too many requests"

**Solutions:**
- Wait for the 15-minute window to reset
- The server tracks requests and shows current count
- Reduce the frequency of requests
- Use more specific filters to reduce the number of API calls

### Task/Project Not Found

**Problem:** "resource not found: the requested item doesn't exist"

**Solutions:**
- Verify the ID is correct (task IDs, project IDs, etc.)
- The item may have been deleted
- Use `list_projects` or `search_tasks` to find the correct ID
- Check that you have access to the item (for shared projects)

### Invalid Parameters

**Problem:** "priority must be between 1 (normal) and 4 (urgent)"

**Solutions:**
- Check parameter values match the expected format
- Priority must be 1-4 (integers)
- Dates should use YYYY-MM-DD format or natural language
- Arrays (like labels) should be properly formatted

### Network Timeouts

**Problem:** Connection timeouts or slow responses

**Solutions:**
- Check your internet connection
- Todoist API servers may be temporarily unavailable
- The server has a 30-second timeout - wait and retry
- Try accessing todoist.com in your browser to verify service status

## Architecture

The server consists of:

- **Todoist Client** (`todoist/client.go`) - HTTP client wrapper with rate limiting
- **Configuration** (`config/config.go`) - Environment variable loading and validation
- **Tool Handlers** (`tools/*.go`) - MCP tool implementations for each operation
- **Main Server** (`main.go`) - MCP server initialization and tool registration

## Dependencies

- [mcp-go](https://github.com/mark3labs/mcp-go) v0.43.2 - Official MCP Go SDK
- [godotenv](https://github.com/joho/godotenv) v1.5.1 - Environment variable loader
- Go standard library - `net/http`, `encoding/json`, `context`

## Security Considerations

- Never commit your `.env` file or API token to version control
- Your API token provides full access to your Todoist account
- You can regenerate your token at any time from https://todoist.com/prefs/integrations
- The server runs locally and doesn't send data to third parties
- All communication with Todoist uses HTTPS (TLS encryption)
- Store your token securely using environment variables or secret management

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## Support

For issues, questions, or feature requests, please open an issue on [GitHub](https://github.com/rgabriel/mcp-todoist/issues).

## Acknowledgments

- Built with the official [mcp-go SDK](https://mcp-go.dev)
- Integrates with [Todoist REST API v2](https://developer.todoist.com/rest/v2/)
- Follows the [Model Context Protocol](https://modelcontextprotocol.io) specification
