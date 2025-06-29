# Claude Squad - Developer Notes

This file contains technical notes and troubleshooting guides for developers working on Claude Squad.

## Tmux Session Architecture

Claude Squad uses tmux to manage multiple AI coding sessions. Understanding the tmux session structure is crucial for debugging attachment issues.

### Session Structure

Each Claude Squad instance creates a tmux session with the following windows:

```
session_name/
├── Window 0 (main): Runs the AI assistant (Claude, Aider, etc.)
└── Window "terminal": Runs a plain shell (zsh) for user interaction
```

### Window Creation Process

1. **Main Window**: Created during `tmuxSession.Start()` with `tmux new-session -d -s session_name program`
2. **Terminal Window**: Created lazily during `tmuxSession.CaptureTerminalContent()` with `tmux new-window -n terminal zsh`

### Attachment Methods

- `Attach()` → `AttachToWindow("0")` → `tmux attach-session -t session_name:0` (main window)
- `AttachToTerminal()` → `AttachToWindow("terminal")` → `tmux attach-session -t session_name:terminal`

**Critical Note**: Always specify the window explicitly. Using `tmux attach-session -t session_name` without a window will attach to the **last active window**, not necessarily the main window.

## UI Tab to Tmux Window Mapping

The tabbed interface maps to tmux windows as follows:

```
UI Tab          → Tmux Window    → Content
─────────────────────────────────────────────
Preview Tab     → Window 0       → AI assistant interaction
Diff Tab        → Window 0       → AI assistant interaction  
Terminal Tab    → Window terminal → Shell interaction
```

### Attachment Logic

When pressing Enter to attach:

```go
if m.tabbedWindow.IsInTerminalTab() {
    ch, err = m.list.AttachToTerminal()  // → session_name:terminal
} else {
    ch, err = m.list.Attach()           // → session_name:0
}
```

## Common Issues and Troubleshooting

### Issue: Enter Always Attaches to Terminal Window

**Symptoms**: Regardless of which tab is active, pressing Enter always attaches to the terminal window.

**Root Cause**: The `Attach()` method was using `tmux attach-session -t session_name` without specifying a window. Tmux defaults to the last active window, which may be the terminal window after it's created.

**Solution**: Explicitly specify window 0 in the `Attach()` method:
```go
func (t *TmuxSession) Attach() (chan struct{}, error) {
    return t.AttachToWindow("0")  // Explicitly target window 0
}
```

### Issue: Terminal Window Doesn't Exist

**Symptoms**: Errors when trying to attach to terminal window or capture terminal content.

**Root Cause**: Terminal window is created lazily and may not exist yet.

**Solution**: The `AttachToTerminal()` method calls `CaptureTerminalContent()` first to ensure the window exists:
```go
func (i *Instance) AttachToTerminal() (chan struct{}, error) {
    // Ensure terminal window exists
    _, err := i.tmuxSession.CaptureTerminalContent()
    if err != nil {
        return nil, fmt.Errorf("failed to ensure terminal window exists: %w", err)
    }
    return i.tmuxSession.AttachToWindow("terminal")
}
```

### Debugging Window Issues

To debug tmux window issues:

```bash
# List all windows in a session
tmux list-windows -t session_name

# Show current active window
tmux display-message -t session_name -p "#{window_active} #{window_name} #{window_index}"

# Manually attach to specific window
tmux attach-session -t session_name:0        # Main window
tmux attach-session -t session_name:terminal # Terminal window
```

## Key Learnings

1. **Tmux Default Behavior**: When attaching without specifying a window, tmux uses the last active window, not the first window.

2. **Window Lifecycle**: The terminal window is created on-demand, which can affect which window becomes the "last active" one.

3. **Explicit Window Targeting**: Always specify the exact window you want to attach to avoid ambiguity.

4. **Tab State vs Window State**: The UI tab state and tmux window state are separate - ensure they stay synchronized.

## Development Guidelines

- Always test attachment behavior on both Preview and Terminal tabs
- When modifying tmux attachment logic, verify window targeting is explicit
- Consider window creation timing when debugging attachment issues
- Use window names ("terminal") or indices ("0") consistently throughout the codebase