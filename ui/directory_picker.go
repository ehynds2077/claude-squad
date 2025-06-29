package ui

import (
	"claude-squad/session/git"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// DirectoryPicker wraps the filepicker to handle directory selection
type DirectoryPicker struct {
	filepicker   filepicker.Model
	width        int
	height       int
	selected     bool
	selectedPath string
	err          error
}

// DirectorySelectedMsg is sent when a directory is selected
type DirectorySelectedMsg struct {
	Path string
}

// DirectoryPickerCancelledMsg is sent when the directory picker is cancelled
type DirectoryPickerCancelledMsg struct{}

// NewDirectoryPicker creates a new directory picker
func NewDirectoryPicker() *DirectoryPicker {
	fp := filepicker.New()
	fp.AllowedTypes = []string{} // Empty means all file types, but we'll filter for directories
	fp.DirAllowed = true
	fp.FileAllowed = true // Need to show files to navigate, but we'll only allow selecting directories
	fp.ShowHidden = false
	
	// Set starting directory to home directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		homeDir = "."
	}
	fp.CurrentDirectory = homeDir
	
	// Custom key bindings
	fp.KeyMap = filepicker.KeyMap{
		GoToTop: key.NewBinding(
			key.WithKeys("g"),
			key.WithHelp("g", "top"),
		),
		GoToLast: key.NewBinding(
			key.WithKeys("G"),
			key.WithHelp("G", "bottom"),
		),
		Down: key.NewBinding(
			key.WithKeys("j", "down", "ctrl+n"),
			key.WithHelp("j", "down"),
		),
		Up: key.NewBinding(
			key.WithKeys("k", "up", "ctrl+p"),
			key.WithHelp("k", "up"),
		),
		PageDown: key.NewBinding(
			key.WithKeys("ctrl+f", "pgdown"),
			key.WithHelp("ctrl+f", "page down"),
		),
		PageUp: key.NewBinding(
			key.WithKeys("ctrl+b", "pgup"),
			key.WithHelp("ctrl+b", "page up"),
		),
		Back: key.NewBinding(
			key.WithKeys("h", "left", "backspace"),
			key.WithHelp("h", "back"),
		),
		Open: key.NewBinding(
			key.WithKeys("l", "right", "enter"),
			key.WithHelp("l", "open"),
		),
		Select: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "select"),
		),
	}
	
	return &DirectoryPicker{
		filepicker: fp,
		width:      80,
		height:     20,
	}
}

// SetSize sets the size of the directory picker
func (dp *DirectoryPicker) SetSize(width, height int) {
	dp.width = width
	dp.height = height
	dp.filepicker.Height = height - 4 // Leave space for border and instructions
}

// Init implements tea.Model
func (dp *DirectoryPicker) Init() tea.Cmd {
	return dp.filepicker.Init()
}

// Update implements tea.Model
func (dp *DirectoryPicker) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc", "q":
			// Cancel directory selection
			return dp, func() tea.Msg {
				return DirectoryPickerCancelledMsg{}
			}
		case "enter", " ":
			// Select current directory
			selectedPath := dp.filepicker.CurrentDirectory
			
			if selectedPath != "" {
				// Validate that it's a git repository
				if !dp.isGitRepository(selectedPath) {
					dp.err = fmt.Errorf("selected directory is not a git repository")
					return dp, nil
				}
				
				dp.selected = true
				dp.selectedPath = selectedPath
				return dp, func() tea.Msg {
					return DirectorySelectedMsg{Path: selectedPath}
				}
			}
		}
	}

	var cmd tea.Cmd
	dp.filepicker, cmd = dp.filepicker.Update(msg)
	
	// Check if a file/directory was selected via the filepicker
	if selected, _ := dp.filepicker.DidSelectFile(msg); selected {
		selectedPath := dp.filepicker.Path
		
		// Check if the selected path is a directory
		if info, err := os.Stat(selectedPath); err == nil {
			if info.IsDir() {
				// Validate that it's a git repository
				if !dp.isGitRepository(selectedPath) {
					dp.err = fmt.Errorf("selected directory is not a git repository")
					return dp, nil
				}
				
				dp.selected = true
				dp.selectedPath = selectedPath
				return dp, func() tea.Msg {
					return DirectorySelectedMsg{Path: selectedPath}
				}
			} else {
				// If it's a file, show error
				dp.err = fmt.Errorf("please select a directory, not a file")
				return dp, nil
			}
		} else {
			dp.err = fmt.Errorf("cannot access selected path: %v", err)
			return dp, nil
		}
	}
	
	return dp, cmd
}

// isGitRepository checks if the given path is a git repository
func (dp *DirectoryPicker) isGitRepository(path string) bool {
	return git.IsGitRepo(path)
}

// View implements tea.Model
func (dp *DirectoryPicker) View() string {
	var b strings.Builder
	
	// Title
	title := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#FAFAFA")).
		Background(lipgloss.Color("#7D56F4")).
		Padding(0, 1).
		Render("Select Directory")
	
	b.WriteString(title)
	b.WriteString("\n\n")
	
	// Error message if any
	if dp.err != nil {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF0000")).
			Bold(true)
		b.WriteString(errorStyle.Render(fmt.Sprintf("Error: %s", dp.err.Error())))
		b.WriteString("\n\n")
	}
	
	// Instructions
	instructions := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#626262")).
		Render("Navigate: j/k (up/down) | h/l (back/forward) | Enter/Space: select current dir | Cancel: esc/q")
	
	b.WriteString(instructions)
	b.WriteString("\n\n")
	
	// Current directory
	currentDir := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#04B575")).
		Bold(true).
		Render(fmt.Sprintf("Current: %s", dp.filepicker.CurrentDirectory))
	
	b.WriteString(currentDir)
	b.WriteString("\n\n")
	
	// File picker
	b.WriteString(dp.filepicker.View())
	
	// Border
	borderStyle := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#874BFD")).
		Padding(1)
	
	return lipgloss.Place(
		dp.width, dp.height,
		lipgloss.Center, lipgloss.Center,
		borderStyle.Render(b.String()),
	)
}

// SelectedPath returns the selected directory path
func (dp *DirectoryPicker) SelectedPath() string {
	return dp.selectedPath
}

// IsSelected returns true if a directory has been selected
func (dp *DirectoryPicker) IsSelected() bool {
	return dp.selected
}

// Reset resets the directory picker state
func (dp *DirectoryPicker) Reset() {
	dp.selected = false
	dp.selectedPath = ""
	dp.err = nil
}