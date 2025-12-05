# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a macOS-specific MCP (Model Context Protocol) server that provides window management capabilities through AppleScript automation. The server allows AI assistants to control application window positions and sizes on macOS.

## Architecture

**Single-file Go application**: The entire MCP server is implemented in `main.go` with no external modules or packages.

**Eight MCP tools** (3 original + 5 extended):

*Original tools:*
1. `move_resize_app` - Moves and resizes an application's frontmost window
2. `get_app_window_geometry` - Retrieves current window position and size for a specific app
3. `get_main_screen_bounds` - Gets the main desktop/screen dimensions

*Extended tools:*
4. `list_all_windows` - Lists all visible windows from all running applications with positions/sizes
5. `get_app_all_windows` - Gets all windows for a specific app (handles multi-window apps)
6. `move_resize_app_window` - Enhanced version that can target specific windows by index
7. `list_all_screens` - Lists all connected physical displays/monitors with bounds
8. `move_app_to_screen` - Convenience tool to move apps to specific screens with positioning presets

**AppleScript integration**: All window management operations are performed by executing AppleScript commands through `osascript`. The `runAppleScript` helper function handles script execution and error handling.

**System command integration**: Multi-monitor detection uses `system_profiler SPDisplaysDataType -json` via the `runCommand` helper, as pure AppleScript cannot reliably enumerate individual displays.

**MCP SDK**: Uses `github.com/modelcontextprotocol/go-sdk/mcp` for the MCP server implementation with stdio transport.

## Development Commands

**Initialize Go module** (if not already done):
```bash
go mod init github.com/yourusername/wm-mcp
go mod tidy
```

**Run the server**:
```bash
go run main.go
```

**Build executable**:
```bash
go build -o wm-mcp main.go
```

**Test AppleScript functionality manually**:
```bash
osascript -e 'tell application "System Events" to get name of every application process whose visible is true'
```

## macOS Permissions

This server requires macOS accessibility permissions to control other applications. Users must grant permission in System Preferences > Security & Privacy > Privacy > Accessibility.

## Key Implementation Details

**Coordinate system**: Uses macOS screen coordinates where (0,0) is at the top-left of the main display (the one with the menu bar). Multi-display setups have:
- Displays to the left: negative X coordinates
- Displays above: negative Y coordinates
- All displays form one continuous virtual coordinate space

**Window targeting**:
- Original tools target the frontmost window (window 1) of the specified application process
- Extended tools support multi-window apps by allowing window index specification
- Window indices are 1-based (1 = frontmost window)

**Multi-window support**: The `list_all_windows` and `get_app_all_windows` tools iterate through all windows of visible application processes to provide comprehensive window inventories.

**Multi-monitor detection**: Uses `system_profiler SPDisplaysDataType -json` to get accurate display information including resolutions, since pure AppleScript cannot reliably enumerate individual displays. Combines this with Finder desktop bounds to map the virtual coordinate space.

**Positioning presets**: The `move_app_to_screen` tool supports positioning presets:
- `center` - Center window on screen (50% width/height)
- `maximize` - Fill entire screen
- `left-half`, `right-half` - Left/right 50% of screen
- `top-half`, `bottom-half` - Top/bottom 50% of screen
- `custom` - User-specified position and size

**Error handling**: AppleScript errors are captured and returned with combined output for debugging. Common errors include application not running, application has no windows, or permission denied.

**Data parsing**:
- Window geometry data is returned as comma-separated integers from AppleScript and parsed using `parseCSVInts`
- Window lists use pipe-delimited records parsed by `parseWindowRecord`
- Display information is parsed from JSON output using `parseDisplaysJSON`
