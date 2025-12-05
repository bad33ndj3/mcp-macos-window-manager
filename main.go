// main.go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os/exec"
	"strconv"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ---------- Shared helpers ----------

func runAppleScript(ctx context.Context, script string) (string, error) {
	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("osascript error: %w (output: %s)", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func parseCSVInts(s string, n int) ([]int, error) {
	s = strings.TrimSpace(s)
	parts := strings.Split(s, ",")

	// Filter out empty parts after trimming
	var nonEmptyParts []string
	for _, p := range parts {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			nonEmptyParts = append(nonEmptyParts, trimmed)
		}
	}

	if len(nonEmptyParts) != n {
		return nil, fmt.Errorf("expected %d comma-separated values, got %d (%q)", n, len(nonEmptyParts), s)
	}
	out := make([]int, n)
	for i, p := range nonEmptyParts {
		v, err := strconv.Atoi(p)
		if err != nil {
			return nil, fmt.Errorf("invalid int at position %d: %q (%w)", i, p, err)
		}
		out[i] = v
	}
	return out, nil
}

func runCommand(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command error: %w (output: %s)", err, strings.TrimSpace(string(out)))
	}
	return strings.TrimSpace(string(out)), nil
}

func parseWindowRecord(record string) (appName, windowTitle string, x, y, width, height int, err error) {
	parts := strings.Split(record, "|")
	if len(parts) != 6 {
		return "", "", 0, 0, 0, 0, fmt.Errorf("expected 6 pipe-separated values, got %d (%q)", len(parts), record)
	}
	appName = strings.TrimSpace(parts[0])
	windowTitle = strings.TrimSpace(parts[1])
	x, err = strconv.Atoi(strings.TrimSpace(parts[2]))
	if err != nil {
		return "", "", 0, 0, 0, 0, fmt.Errorf("invalid x coordinate: %w", err)
	}
	y, err = strconv.Atoi(strings.TrimSpace(parts[3]))
	if err != nil {
		return "", "", 0, 0, 0, 0, fmt.Errorf("invalid y coordinate: %w", err)
	}
	width, err = strconv.Atoi(strings.TrimSpace(parts[4]))
	if err != nil {
		return "", "", 0, 0, 0, 0, fmt.Errorf("invalid width: %w", err)
	}
	height, err = strconv.Atoi(strings.TrimSpace(parts[5]))
	if err != nil {
		return "", "", 0, 0, 0, 0, fmt.Errorf("invalid height: %w", err)
	}
	return appName, windowTitle, x, y, width, height, nil
}

// ---------- Tool 1: Move + resize app window ----------

type MoveResizeArgs struct {
	// Example: "Google Chrome", "Visual Studio Code", "Safari"
	AppName string `json:"appName" jsonschema:"Name of the application, e.g. 'Google Chrome'"`
	// Pixel coordinates relative to the top-left of the main display / desktop space.
	X int `json:"x" jsonschema:"X position in pixels"`
	Y int `json:"y" jsonschema:"Y position in pixels"`
	// Window size in pixels.
	Width  int `json:"width" jsonschema:"Window width in pixels"`
	Height int `json:"height" jsonschema:"Window height in pixels"`
}

func MoveResizeApp(ctx context.Context, req *mcp.CallToolRequest, args MoveResizeArgs) (*mcp.CallToolResult, any, error) {
	if args.AppName == "" {
		return nil, nil, fmt.Errorf("appName is required")
	}
	if args.Width <= 0 || args.Height <= 0 {
		return nil, nil, fmt.Errorf("width and height must be > 0")
	}

	script := fmt.Sprintf(`
tell application "System Events"
	if not (exists application process "%[1]s") then
		error "Application '%[1]s' is not running."
	end if
	tell application process "%[1]s"
		set frontmost to true
		if (count of windows) is 0 then
			error "Application '%[1]s' has no windows."
		end if
		tell window 1
			set position to {%[2]d, %[3]d}
			set size to {%[4]d, %[5]d}
		end tell
	end tell
end tell
`, args.AppName, args.X, args.Y, args.Width, args.Height)

	if _, err := runAppleScript(ctx, script); err != nil {
		return nil, nil, err
	}

	text := fmt.Sprintf("Moved '%s' to (%d,%d) with size %dx%d", args.AppName, args.X, args.Y, args.Width, args.Height)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}, nil, nil
}

// ---------- Tool 2: Get current window geometry for an app ----------

type GetWindowArgs struct {
	AppName string `json:"appName" jsonschema:"Name of the application, e.g. 'Google Chrome'"`
}

type WindowGeometry struct {
	AppName string `json:"appName" jsonschema:"Application name"`
	X       int    `json:"x" jsonschema:"X position in pixels"`
	Y       int    `json:"y" jsonschema:"Y position in pixels"`
	Width   int    `json:"width" jsonschema:"Window width in pixels"`
	Height  int    `json:"height" jsonschema:"Window height in pixels"`
}

func GetAppWindowGeometry(ctx context.Context, req *mcp.CallToolRequest, args GetWindowArgs) (*mcp.CallToolResult, WindowGeometry, error) {
	if args.AppName == "" {
		return nil, WindowGeometry{}, fmt.Errorf("appName is required")
	}

	script := fmt.Sprintf(`
tell application "System Events"
	if not (exists application process "%[1]s") then
		error "Application '%[1]s' is not running."
	end if
	tell application process "%[1]s"
		if (count of windows) is 0 then
			error "Application '%[1]s' has no windows."
		end if
		tell window 1
			set {xPos, yPos} to position
			set {w, h} to size
			return xPos & "," & yPos & "," & w & "," & h
		end tell
	end tell
end tell
`, args.AppName)

	out, err := runAppleScript(ctx, script)
	if err != nil {
		return nil, WindowGeometry{}, err
	}

	vals, err := parseCSVInts(out, 4)
	if err != nil {
		return nil, WindowGeometry{}, err
	}

	geom := WindowGeometry{
		AppName: args.AppName,
		X:       vals[0],
		Y:       vals[1],
		Width:   vals[2],
		Height:  vals[3],
	}

	text := fmt.Sprintf("Window '%s': pos=(%d,%d) size=%dx%d", geom.AppName, geom.X, geom.Y, geom.Width, geom.Height)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}, geom, nil
}

// ---------- Tool 3: Get main desktop (screen) bounds ----------
//
// Note: this returns the bounds of the Finder desktop window, which
// corresponds to the current Space. With multiple displays, coords can
// be a virtual desktop (e.g. negative X for left displays).

type ScreenBounds struct {
	Left   int `json:"left" jsonschema:"Left coordinate in pixels"`
	Top    int `json:"top" jsonschema:"Top coordinate in pixels"`
	Right  int `json:"right" jsonschema:"Right coordinate in pixels"`
	Bottom int `json:"bottom" jsonschema:"Bottom coordinate in pixels"`
	Width  int `json:"width" jsonschema:"Width in pixels (right-left)"`
	Height int `json:"height" jsonschema:"Height in pixels (bottom-top)"`
}

func GetMainScreenBounds(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, ScreenBounds, error) {
	// AppleScript: get bounds of Finder desktop window: {left, top, right, bottom}
	script := `
tell application "Finder"
	set b to bounds of window of desktop
	set {l, t, r, btm} to b
	return l & "," & t & "," & r & "," & btm
end tell
`
	out, err := runAppleScript(ctx, script)
	if err != nil {
		return nil, ScreenBounds{}, err
	}

	vals, err := parseCSVInts(out, 4)
	if err != nil {
		return nil, ScreenBounds{}, err
	}

	sb := ScreenBounds{
		Left:   vals[0],
		Top:    vals[1],
		Right:  vals[2],
		Bottom: vals[3],
		Width:  vals[2] - vals[0],
		Height: vals[3] - vals[1],
	}

	text := fmt.Sprintf("Main desktop bounds: left=%d top=%d right=%d bottom=%d width=%d height=%d",
		sb.Left, sb.Top, sb.Right, sb.Bottom, sb.Width, sb.Height)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}, sb, nil
}

// ---------- Tool 4: List all windows from all apps ----------

type WindowInfo struct {
	AppName     string `json:"appName" jsonschema:"Application name"`
	WindowTitle string `json:"windowTitle" jsonschema:"Window title/name"`
	X           int    `json:"x" jsonschema:"X position in pixels"`
	Y           int    `json:"y" jsonschema:"Y position in pixels"`
	Width       int    `json:"width" jsonschema:"Window width in pixels"`
	Height      int    `json:"height" jsonschema:"Window height in pixels"`
}

type ListAllWindowsResult struct {
	Windows []WindowInfo `json:"windows" jsonschema:"List of all visible windows"`
	Count   int          `json:"count" jsonschema:"Total number of windows"`
}

func ListAllWindows(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, ListAllWindowsResult, error) {
	script := `
tell application "System Events"
	set windowList to {}
	repeat with proc in (application processes whose visible is true)
		set appName to name of proc
		try
			repeat with w in (windows of proc)
				try
					set {x, y} to position of w
					set {wWidth, wHeight} to size of w
					set windowTitle to name of w
					set end of windowList to appName & "|" & windowTitle & "|" & x & "|" & y & "|" & wWidth & "|" & wHeight
				end try
			end repeat
		end try
	end repeat
	set AppleScript's text item delimiters to ";"
	return windowList as text
end tell
`
	out, err := runAppleScript(ctx, script)
	if err != nil {
		return nil, ListAllWindowsResult{}, err
	}

	var windows []WindowInfo
	if strings.TrimSpace(out) != "" {
		records := strings.Split(out, ";")
		for _, record := range records {
			if strings.TrimSpace(record) == "" {
				continue
			}
			appName, windowTitle, x, y, width, height, err := parseWindowRecord(record)
			if err != nil {
				// Skip malformed records rather than failing completely
				continue
			}
			windows = append(windows, WindowInfo{
				AppName:     appName,
				WindowTitle: windowTitle,
				X:           x,
				Y:           y,
				Width:       width,
				Height:      height,
			})
		}
	}

	text := fmt.Sprintf("Found %d windows across all applications", len(windows))
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}, ListAllWindowsResult{
		Windows: windows,
		Count:   len(windows),
	}, nil
}

// ---------- Tool 5: Get all windows for a specific app ----------

type AppWindowInfo struct {
	Title  string `json:"title" jsonschema:"Window title"`
	Index  int    `json:"index" jsonschema:"Window index (1-based, 1 = frontmost)"`
	X      int    `json:"x" jsonschema:"X position in pixels"`
	Y      int    `json:"y" jsonschema:"Y position in pixels"`
	Width  int    `json:"width" jsonschema:"Window width in pixels"`
	Height int    `json:"height" jsonschema:"Window height in pixels"`
}

type GetAppAllWindowsResult struct {
	AppName string          `json:"appName" jsonschema:"Application name"`
	Windows []AppWindowInfo `json:"windows" jsonschema:"List of all windows for this app"`
	Count   int             `json:"count" jsonschema:"Total number of windows"`
}

func GetAppAllWindows(ctx context.Context, req *mcp.CallToolRequest, args GetWindowArgs) (*mcp.CallToolResult, GetAppAllWindowsResult, error) {
	if args.AppName == "" {
		return nil, GetAppAllWindowsResult{}, fmt.Errorf("appName is required")
	}

	script := fmt.Sprintf(`
tell application "System Events"
	if not (exists application process "%[1]s") then
		error "Application '%[1]s' is not running."
	end if
	tell application process "%[1]s"
		if (count of windows) is 0 then
			error "Application '%[1]s' has no windows."
		end if
		set windowData to {}
		repeat with w in windows
			try
				set {x, y} to position of w
				set {wWidth, wHeight} to size of w
				set windowTitle to name of w
				set end of windowData to windowTitle & "|" & x & "|" & y & "|" & wWidth & "|" & wHeight
			end try
		end repeat
		set AppleScript's text item delimiters to ";"
		return windowData as text
	end tell
end tell
`, args.AppName)

	out, err := runAppleScript(ctx, script)
	if err != nil {
		return nil, GetAppAllWindowsResult{}, err
	}

	var windows []AppWindowInfo
	if strings.TrimSpace(out) != "" {
		records := strings.Split(out, ";")
		for idx, record := range records {
			if strings.TrimSpace(record) == "" {
				continue
			}
			parts := strings.Split(record, "|")
			if len(parts) != 5 {
				continue
			}
			title := strings.TrimSpace(parts[0])
			x, _ := strconv.Atoi(strings.TrimSpace(parts[1]))
			y, _ := strconv.Atoi(strings.TrimSpace(parts[2]))
			width, _ := strconv.Atoi(strings.TrimSpace(parts[3]))
			height, _ := strconv.Atoi(strings.TrimSpace(parts[4]))

			windows = append(windows, AppWindowInfo{
				Title:  title,
				Index:  idx + 1, // 1-based index
				X:      x,
				Y:      y,
				Width:  width,
				Height: height,
			})
		}
	}

	text := fmt.Sprintf("Application '%s' has %d window(s)", args.AppName, len(windows))
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}, GetAppAllWindowsResult{
		AppName: args.AppName,
		Windows: windows,
		Count:   len(windows),
	}, nil
}

// ---------- Tool 6: Move + resize specific app window by index ----------

type MoveResizeWindowArgs struct {
	AppName     string `json:"appName" jsonschema:"Name of the application"`
	WindowIndex int    `json:"windowIndex" jsonschema:"Window index (1-based, 1 = frontmost window)"`
	X           int    `json:"x" jsonschema:"X position in pixels"`
	Y           int    `json:"y" jsonschema:"Y position in pixels"`
	Width       int    `json:"width" jsonschema:"Window width in pixels"`
	Height      int    `json:"height" jsonschema:"Window height in pixels"`
}

func MoveResizeAppWindow(ctx context.Context, req *mcp.CallToolRequest, args MoveResizeWindowArgs) (*mcp.CallToolResult, any, error) {
	if args.AppName == "" {
		return nil, nil, fmt.Errorf("appName is required")
	}
	if args.WindowIndex < 1 {
		return nil, nil, fmt.Errorf("windowIndex must be >= 1")
	}
	if args.Width <= 0 || args.Height <= 0 {
		return nil, nil, fmt.Errorf("width and height must be > 0")
	}

	script := fmt.Sprintf(`
tell application "System Events"
	if not (exists application process "%[1]s") then
		error "Application '%[1]s' is not running."
	end if
	tell application process "%[1]s"
		set frontmost to true
		if (count of windows) < %[2]d then
			error "Application '%[1]s' does not have window %[2]d."
		end if
		tell window %[2]d
			set position to {%[3]d, %[4]d}
			set size to {%[5]d, %[6]d}
		end tell
	end tell
end tell
`, args.AppName, args.WindowIndex, args.X, args.Y, args.Width, args.Height)

	if _, err := runAppleScript(ctx, script); err != nil {
		return nil, nil, err
	}

	text := fmt.Sprintf("Moved '%s' window %d to (%d,%d) with size %dx%d", args.AppName, args.WindowIndex, args.X, args.Y, args.Width, args.Height)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}, nil, nil
}

// ---------- Tool 7: List all screens / displays ----------

type DisplayInfo struct {
	Index  int    `json:"index" jsonschema:"Display index (0 = main display with menu bar)"`
	Name   string `json:"name" jsonschema:"Display name"`
	Left   int    `json:"left" jsonschema:"Left coordinate in pixels"`
	Top    int    `json:"top" jsonschema:"Top coordinate in pixels"`
	Right  int    `json:"right" jsonschema:"Right coordinate in pixels"`
	Bottom int    `json:"bottom" jsonschema:"Bottom coordinate in pixels"`
	Width  int    `json:"width" jsonschema:"Width in pixels"`
	Height int    `json:"height" jsonschema:"Height in pixels"`
	IsMain bool   `json:"isMain" jsonschema:"Whether this is the main display with menu bar"`
}

type ListAllScreensResult struct {
	Displays    []DisplayInfo `json:"displays" jsonschema:"List of all connected displays"`
	Count       int           `json:"count" jsonschema:"Total number of displays"`
	TotalWidth  int           `json:"totalWidth" jsonschema:"Total virtual desktop width"`
	TotalHeight int           `json:"totalHeight" jsonschema:"Total virtual desktop height"`
}

type systemProfilerDisplay struct {
	Resolution string `json:"_spdisplays_resolution"`
	Main       string `json:"spdisplays_main"`
	Name       string `json:"_name"`
}

type systemProfilerData struct {
	SPDisplaysDataType []struct {
		Name     string                  `json:"_name"`
		Displays []systemProfilerDisplay `json:"spdisplays_ndrvs"`
	} `json:"SPDisplaysDataType"`
}

func ListAllScreens(ctx context.Context, req *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, ListAllScreensResult, error) {
	// Get desktop bounds to determine total virtual space
	desktopScript := `
tell application "Finder"
	set b to bounds of window of desktop
	set {l, t, r, btm} to b
	return l & "," & t & "," & r & "," & btm
end tell
`
	desktopOut, err := runAppleScript(ctx, desktopScript)
	if err != nil {
		return nil, ListAllScreensResult{}, fmt.Errorf("failed to get desktop bounds: %w", err)
	}

	desktopVals, err := parseCSVInts(desktopOut, 4)
	if err != nil {
		return nil, ListAllScreensResult{}, fmt.Errorf("failed to parse desktop bounds: %w", err)
	}

	totalLeft := desktopVals[0]
	totalTop := desktopVals[1]
	totalRight := desktopVals[2]
	totalBottom := desktopVals[3]
	totalWidth := totalRight - totalLeft
	totalHeight := totalBottom - totalTop

	// Get display information from system_profiler
	profilerOut, err := runCommand(ctx, "system_profiler", "SPDisplaysDataType", "-json")
	if err != nil {
		// If system_profiler fails, fall back to single display
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Found 1 display (fallback): %dx%d", totalWidth, totalHeight)},
			},
		}, ListAllScreensResult{
			Displays: []DisplayInfo{
				{
					Index:  0,
					Name:   "Main Display",
					Left:   totalLeft,
					Top:    totalTop,
					Right:  totalRight,
					Bottom: totalBottom,
					Width:  totalWidth,
					Height: totalHeight,
					IsMain: true,
				},
			},
			Count:       1,
			TotalWidth:  totalWidth,
			TotalHeight: totalHeight,
		}, nil
	}

	var profilerData systemProfilerData
	if err := json.Unmarshal([]byte(profilerOut), &profilerData); err != nil {
		// If JSON parsing fails, fall back to single display
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: fmt.Sprintf("Found 1 display (fallback): %dx%d", totalWidth, totalHeight)},
			},
		}, ListAllScreensResult{
			Displays: []DisplayInfo{
				{
					Index:  0,
					Name:   "Main Display",
					Left:   totalLeft,
					Top:    totalTop,
					Right:  totalRight,
					Bottom: totalBottom,
					Width:  totalWidth,
					Height: totalHeight,
					IsMain: true,
				},
			},
			Count:       1,
			TotalWidth:  totalWidth,
			TotalHeight: totalHeight,
		}, nil
	}

	// Extract displays from system_profiler output
	var displays []DisplayInfo
	displayIndex := 0

	if len(profilerData.SPDisplaysDataType) > 0 {
		for _, gpu := range profilerData.SPDisplaysDataType {
			for _, display := range gpu.Displays {
				isMain := display.Main == "spdisplays_yes"

				// Parse resolution (e.g., "3840 x 2160")
				width := totalWidth
				height := totalHeight
				if display.Resolution != "" {
					resParts := strings.Fields(display.Resolution)
					if len(resParts) >= 3 {
						if w, err := strconv.Atoi(resParts[0]); err == nil {
							width = w
						}
						if h, err := strconv.Atoi(resParts[2]); err == nil {
							height = h
						}
					}
				}

				// For simplicity, assume horizontal layout left-to-right
				// Main display is at (0, 0)
				left := 0
				top := 0
				if isMain {
					// Main display at origin
					left = 0
					top = 0
				} else if len(displays) > 0 {
					// Place next to previous display (simple horizontal layout)
					lastDisplay := displays[len(displays)-1]
					left = lastDisplay.Right
					top = 0
				}

				displays = append(displays, DisplayInfo{
					Index:  displayIndex,
					Name:   display.Name,
					Left:   left,
					Top:    top,
					Right:  left + width,
					Bottom: top + height,
					Width:  width,
					Height: height,
					IsMain: isMain,
				})
				displayIndex++
			}
		}
	}

	// If no displays detected, use fallback
	if len(displays) == 0 {
		displays = []DisplayInfo{
			{
				Index:  0,
				Name:   "Main Display",
				Left:   totalLeft,
				Top:    totalTop,
				Right:  totalRight,
				Bottom: totalBottom,
				Width:  totalWidth,
				Height: totalHeight,
				IsMain: true,
			},
		}
	}

	text := fmt.Sprintf("Found %d display(s), total virtual desktop: %dx%d", len(displays), totalWidth, totalHeight)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}, ListAllScreensResult{
		Displays:    displays,
		Count:       len(displays),
		TotalWidth:  totalWidth,
		TotalHeight: totalHeight,
	}, nil
}

// ---------- Tool 8: Move app to specific screen with presets ----------

type MoveAppToScreenArgs struct {
	AppName     string `json:"appName" jsonschema:"Name of the application"`
	ScreenIndex int    `json:"screenIndex" jsonschema:"Target screen index (0 = main display)"`
	Position    string `json:"position" jsonschema:"Positioning preset: 'center', 'maximize', 'left-half', 'right-half', 'top-half', 'bottom-half', or 'custom'"`
	// For custom positioning:
	XOffset *int `json:"xOffset,omitempty" jsonschema:"X offset from screen left (pixels, for custom position)"`
	YOffset *int `json:"yOffset,omitempty" jsonschema:"Y offset from screen top (pixels, for custom position)"`
	Width   *int `json:"width,omitempty" jsonschema:"Window width (pixels, for custom position)"`
	Height  *int `json:"height,omitempty" jsonschema:"Window height (pixels, for custom position)"`
}

func calculateWindowBounds(screen DisplayInfo, position string, xOffset, yOffset, width, height *int) (x, y, w, h int, err error) {
	switch position {
	case "center":
		w = screen.Width / 2
		h = screen.Height / 2
		x = screen.Left + (screen.Width-w)/2
		y = screen.Top + (screen.Height-h)/2
	case "maximize":
		x = screen.Left
		y = screen.Top
		w = screen.Width
		h = screen.Height
	case "left-half":
		x = screen.Left
		y = screen.Top
		w = screen.Width / 2
		h = screen.Height
	case "right-half":
		x = screen.Left + screen.Width/2
		y = screen.Top
		w = screen.Width / 2
		h = screen.Height
	case "top-half":
		x = screen.Left
		y = screen.Top
		w = screen.Width
		h = screen.Height / 2
	case "bottom-half":
		x = screen.Left
		y = screen.Top + screen.Height/2
		w = screen.Width
		h = screen.Height / 2
	case "custom":
		if xOffset == nil || yOffset == nil || width == nil || height == nil {
			return 0, 0, 0, 0, fmt.Errorf("custom position requires xOffset, yOffset, width, and height")
		}
		x = screen.Left + *xOffset
		y = screen.Top + *yOffset
		w = *width
		h = *height
	default:
		return 0, 0, 0, 0, fmt.Errorf("invalid position preset: %q (valid: center, maximize, left-half, right-half, top-half, bottom-half, custom)", position)
	}
	return x, y, w, h, nil
}

func MoveAppToScreen(ctx context.Context, req *mcp.CallToolRequest, args MoveAppToScreenArgs) (*mcp.CallToolResult, any, error) {
	if args.AppName == "" {
		return nil, nil, fmt.Errorf("appName is required")
	}
	if args.Position == "" {
		return nil, nil, fmt.Errorf("position is required")
	}

	// Get all screens
	_, screensResult, err := ListAllScreens(ctx, req, struct{}{})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get screens: %w", err)
	}

	// Validate screen index
	if args.ScreenIndex < 0 || args.ScreenIndex >= len(screensResult.Displays) {
		return nil, nil, fmt.Errorf("invalid screen index %d (available: 0-%d)", args.ScreenIndex, len(screensResult.Displays)-1)
	}

	targetScreen := screensResult.Displays[args.ScreenIndex]

	// Calculate window bounds
	x, y, width, height, err := calculateWindowBounds(targetScreen, args.Position, args.XOffset, args.YOffset, args.Width, args.Height)
	if err != nil {
		return nil, nil, err
	}

	// Move the window using existing tool
	moveArgs := MoveResizeArgs{
		AppName: args.AppName,
		X:       x,
		Y:       y,
		Width:   width,
		Height:  height,
	}

	_, _, err = MoveResizeApp(ctx, req, moveArgs)
	if err != nil {
		return nil, nil, err
	}

	text := fmt.Sprintf("Moved '%s' to screen %d (%s) at position '%s': (%d,%d) %dx%d",
		args.AppName, args.ScreenIndex, targetScreen.Name, args.Position, x, y, width, height)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{Text: text},
		},
	}, nil, nil
}

// ---------- main: MCP server over stdio ----------

func main() {
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "apple-window-manager",
		Version: "0.3.0",
	}, nil)

	// Tool 1: move & resize
	mcp.AddTool(server, &mcp.Tool{
		Name:        "move_resize_app",
		Description: "Move and resize an application's frontmost window using AppleScript on macOS.",
	}, MoveResizeApp)

	// Tool 2: get window geometry
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_app_window_geometry",
		Description: "Get position and size of an application's frontmost window.",
	}, GetAppWindowGeometry)

	// Tool 3: get main screen / desktop bounds
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_main_screen_bounds",
		Description: "Get the bounds of the main desktop (Finder desktop window).",
	}, GetMainScreenBounds)

	// Tool 4: list all windows from all applications
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_all_windows",
		Description: "List all visible windows from all running applications with their positions and sizes.",
	}, ListAllWindows)

	// Tool 5: get all windows for a specific application
	mcp.AddTool(server, &mcp.Tool{
		Name:        "get_app_all_windows",
		Description: "Get all windows for a specific application (handles multi-window apps).",
	}, GetAppAllWindows)

	// Tool 6: move and resize specific window by index
	mcp.AddTool(server, &mcp.Tool{
		Name:        "move_resize_app_window",
		Description: "Move and resize a specific window by index for multi-window applications.",
	}, MoveResizeAppWindow)

	// Tool 7: list all screens / displays
	mcp.AddTool(server, &mcp.Tool{
		Name:        "list_all_screens",
		Description: "List all connected physical displays/monitors with their bounds and properties.",
	}, ListAllScreens)

	// Tool 8: move app to specific screen with positioning presets
	mcp.AddTool(server, &mcp.Tool{
		Name:        "move_app_to_screen",
		Description: "Convenience tool to move an application to a specific screen with positioning presets (center, maximize, left-half, right-half, etc.).",
	}, MoveAppToScreen)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("MCP server failed: %v", err)
	}
}
