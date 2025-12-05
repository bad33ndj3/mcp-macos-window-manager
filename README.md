# mcp-macos-window-manager

A macOS Window Manager MCP Server

A Model Context Protocol (MCP) server that provides comprehensive window management capabilities for macOS through AppleScript automation. Allows AI assistants to control, move, and organize application windows across multiple displays.

## Features

### Window Management
- **List all windows** - Get all visible windows from all running applications
- **Get app windows** - Retrieve all windows for a specific application
- **Move & resize windows** - Precisely position windows by coordinates
- **Multi-window support** - Target specific windows by index for apps with multiple windows

### Multi-Monitor Support
- **List all displays** - Enumerate all connected physical monitors with their bounds
- **Screen detection** - Automatic detection using `system_profiler` with fallback support
- **Virtual desktop mapping** - Proper coordinate system handling for multi-display setups

### Convenience Features
- **Move to screen presets** - Quick positioning with presets:
  - `center` - Center window on screen (50% width/height)
  - `maximize` - Fill entire screen
  - `left-half`, `right-half` - Left/right 50% of screen
  - `top-half`, `bottom-half` - Top/bottom 50% of screen
  - `custom` - User-specified position and size

## MCP Tools

1. `move_resize_app` - Move and resize an application's frontmost window
2. `get_app_window_geometry` - Get position and size of an app's frontmost window
3. `get_main_screen_bounds` - Get the main desktop/screen dimensions
4. `list_all_windows` - List all visible windows from all running applications
5. `get_app_all_windows` - Get all windows for a specific application
6. `move_resize_app_window` - Move and resize a specific window by index
7. `list_all_screens` - List all connected physical displays/monitors
8. `move_app_to_screen` - Move app to specific screen with positioning presets

## Prerequisites

- macOS (tested on macOS Sonoma and later)
- Go 1.23+ (for building from source)
- macOS Accessibility permissions (required for window control)

## Installation

### Option 1: Build from Source

```bash
# Clone or download this repository
cd wm-mcp

# Initialize Go module and download dependencies
go mod init github.com/yourusername/wm-mcp
go mod tidy

# Build the executable
go build -o wm-mcp main.go

# The executable is now ready at ./wm-mcp
```

### Option 2: Use Pre-built Binary

If you have a pre-built binary, ensure it has execute permissions:

```bash
chmod +x wm-mcp
```

## macOS Permissions

This MCP server requires macOS Accessibility permissions to control other applications:

1. Go to **System Preferences** → **Security & Privacy** → **Privacy** → **Accessibility**
2. Click the lock icon to make changes
3. Add the terminal application you're using (e.g., Terminal.app, iTerm.app) or the AI client app
4. Ensure the checkbox is enabled

## Usage

### Running Standalone

```bash
# Run directly
./wm-mcp

# Or with Go
go run main.go
```

The server communicates via stdio using the Model Context Protocol.

## Integration with AI Tools

**Supported AI Tools:**
- Claude Desktop (Anthropic)
- Claude Code / Codex (VS Code, Cursor, etc.)
- Cursor
- Windsurf
- Codeium
- GitHub Copilot CLI
- Gemini (via Google AI Studio)
- Any MCP-compatible client

---

**Getting the absolute path:** Before configuring, get the full path to your executable:

```bash
# Navigate to your wm-mcp directory
cd /path/to/wm-mcp

# Get absolute path
pwd
# Output example: /Users/yourusername/code/wm-mcp

# Your full command path will be: /Users/yourusername/code/wm-mcp/wm-mcp
```

### Quick Installation (Recommended)

Some tools support adding MCP servers via command line:

#### Claude Desktop (with CLI support)

If your Claude Desktop version supports the CLI:

```bash
# Add the server
claude mcp add apple-window-manager /absolute/path/to/wm-mcp

# Or using the config command
claude mcp install /absolute/path/to/wm-mcp --name apple-window-manager
```

#### Cursor / Windsurf

```bash
# From your project directory
cursor mcp add /absolute/path/to/wm-mcp

# Or globally
cursor mcp add --global /absolute/path/to/wm-mcp
```

#### Codeium

```bash
codeium mcp add apple-window-manager /absolute/path/to/wm-mcp
```

---

### Manual Configuration

If your tool doesn't support quick commands, manually edit the configuration files:

#### Claude Desktop

**Config file:** `~/Library/Application Support/Claude/claude_desktop_config.json`

```json
{
  "mcpServers": {
    "apple-window-manager": {
      "command": "/absolute/path/to/wm-mcp",
      "args": []
    }
  }
}
```

**Setup steps:**
1. Replace `/absolute/path/to/wm-mcp` with full path (e.g., `/Users/yourusername/code/wm-mcp/wm-mcp`)
2. Restart Claude Desktop
3. Grant Accessibility permissions when prompted
4. Window management tools should now be available

#### Claude Code / Codex

**Config file:** `.claude/mcp.json` in workspace or global settings

For VS Code, Cursor, or other editors with Claude Code:

```json
{
  "mcpServers": {
    "apple-window-manager": {
      "command": "/absolute/path/to/wm-mcp",
      "args": []
    }
  }
}
```

**Global config location:**
- macOS: `~/.config/claude-code/mcp.json`
- VS Code: Add to workspace `.vscode/mcp.json`

#### Windsurf

**Config file:** `~/.windsurf/mcp_config.json`

```json
{
  "mcpServers": {
    "apple-window-manager": {
      "command": "/absolute/path/to/wm-mcp",
      "args": []
    }
  }
}
```

#### Cursor (Manual)

**Config file:** `~/.cursor/mcp.json`

```json
{
  "mcpServers": {
    "apple-window-manager": {
      "command": "/absolute/path/to/wm-mcp",
      "args": []
    }
  }
}
```

#### Codeium

**Config file:** `~/.codeium/mcp.json`

```json
{
  "mcpServers": {
    "apple-window-manager": {
      "command": "/absolute/path/to/wm-mcp",
      "args": []
    }
  }
}
```

#### Gemini (via Google AI Studio)

If using Gemini with MCP support through Google AI Studio or compatible client:

```json
{
  "servers": {
    "apple-window-manager": {
      "type": "stdio",
      "command": "/absolute/path/to/wm-mcp",
      "args": []
    }
  }
}
```

#### GitHub Copilot CLI

**Config file:** `~/.config/github-copilot/mcp.json`

```json
{
  "mcpServers": {
    "apple-window-manager": {
      "command": "/absolute/path/to/wm-mcp",
      "args": []
    }
  }
}
```

**Or via environment variable:**

```bash
export GITHUB_COPILOT_MCP_CONFIG='{"apple-window-manager":{"command":"/absolute/path/to/wm-mcp"}}'
```

**Or using Copilot CLI (if supported):**

```bash
gh copilot mcp add apple-window-manager /absolute/path/to/wm-mcp
```

## Example Usage with AI

Once configured, you can ask your AI assistant:

```
"List all my open windows"
"Move Chrome to the left half of my screen"
"Show me all my connected displays"
"Move Safari to my second monitor and maximize it"
"Get all windows for Finder"
"Move the second Chrome window to position 100,100 with size 800x600"
```

## Architecture

- **Single-file Go application** - All code in `main.go`
- **AppleScript integration** - Window operations via `osascript`
- **System commands** - Multi-monitor detection via `system_profiler`
- **MCP SDK** - Uses `github.com/modelcontextprotocol/go-sdk/mcp`
- **Stdio transport** - Communication via standard input/output

## Coordinate System

macOS uses a coordinate system where:
- **(0, 0)** is at the top-left of the main display (the one with the menu bar)
- **Displays to the left** have negative X coordinates
- **Displays above** have negative Y coordinates
- **All displays** form one continuous virtual coordinate space

## Development

```bash
# Run the server
go run main.go

# Build executable
go build -o wm-mcp main.go

# Test AppleScript functionality
osascript -e 'tell application "System Events" to get name of every application process whose visible is true'
```

## Troubleshooting

### "Permission denied" errors
- Grant Accessibility permissions to your terminal or AI client app
- Check System Preferences → Security & Privacy → Privacy → Accessibility

### "Application not running" errors
- Ensure the application name exactly matches the process name
- Use `list_all_windows` to see available application names

### "No displays detected"
- The server falls back to single display mode if `system_profiler` fails
- Check that `system_profiler SPDisplaysDataType -json` works in your terminal

### MCP server not appearing in AI client
- Verify the path to the executable is absolute, not relative
- Check the AI client logs for connection errors
- Restart the AI client after configuration changes
- Ensure the executable has proper permissions (`chmod +x`)

## Version

Current version: **0.3.0**

## License

MIT License - Feel free to use and modify as needed.

## Contributing

Contributions welcome! This is a personal project but open to improvements.

## Related Documentation

- [Model Context Protocol](https://modelcontextprotocol.io/)
- [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk)
- [AppleScript Language Guide](https://developer.apple.com/library/archive/documentation/AppleScript/Conceptual/AppleScriptLangGuide/)
