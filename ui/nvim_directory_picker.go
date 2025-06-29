package ui

import (
	"claude-squad/session/git"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// NvimDirectoryPicker uses neovim with Oil.nvim for directory selection
type NvimDirectoryPicker struct {
	selected     bool
	selectedPath string
	err          error
	tempFile     string
}

// NvimDirectorySelectedMsg is sent when a directory is selected via nvim
type NvimDirectorySelectedMsg struct {
	Path string
}

// NvimDirectoryPickerCancelledMsg is sent when nvim directory picker is cancelled
type NvimDirectoryPickerCancelledMsg struct{}

// NvimDirectoryPickerErrorMsg is sent when there's an error with nvim directory picker
type NvimDirectoryPickerErrorMsg struct {
	Error error
}

// NewNvimDirectoryPicker creates a new nvim-based directory picker
func NewNvimDirectoryPicker() *NvimDirectoryPicker {
	return &NvimDirectoryPicker{
		selected:     false,
		selectedPath: "",
		err:          nil,
		tempFile:     "",
	}
}

// LaunchDirectoryPicker launches the nvim directory picker
func (ndp *NvimDirectoryPicker) LaunchDirectoryPicker() tea.Cmd {
	// Create temporary file for communication
	tempFile, err := os.CreateTemp("", "claude_squad_dir_*")
	if err != nil {
		return func() tea.Msg {
			return NvimDirectoryPickerErrorMsg{Error: fmt.Errorf("failed to create temp file: %w", err)}
		}
	}
	tempFile.Close()
	ndp.tempFile = tempFile.Name()

	// Get the script path
	scriptPath := filepath.Join("scripts", "pick_directory.sh")
	
	// Check if script exists
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		// Try absolute path from executable directory
		execPath, err := os.Executable()
		if err != nil {
			return func() tea.Msg {
				return NvimDirectoryPickerErrorMsg{Error: fmt.Errorf("failed to get executable path: %w", err)}
			}
		}
		execDir := filepath.Dir(execPath)
		scriptPath = filepath.Join(execDir, "scripts", "pick_directory.sh")
		
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			return func() tea.Msg {
				return NvimDirectoryPickerErrorMsg{Error: fmt.Errorf("directory picker script not found: %s", scriptPath)}
			}
		}
	}

	// Create the command and use tea.ExecProcess
	cmd := exec.Command(scriptPath, ndp.tempFile)
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		if err != nil {
			// Clean up temp file on error
			if ndp.tempFile != "" {
				os.Remove(ndp.tempFile)
			}
			return NvimDirectoryPickerErrorMsg{Error: fmt.Errorf("directory picker failed: %w", err)}
		}

		// Read the result from temp file
		content, err := ioutil.ReadFile(ndp.tempFile)
		if err != nil {
			os.Remove(ndp.tempFile)
			return NvimDirectoryPickerErrorMsg{Error: fmt.Errorf("failed to read result: %w", err)}
		}

		// Clean up temp file
		os.Remove(ndp.tempFile)

		selectedPath := strings.TrimSpace(string(content))
		if selectedPath == "" {
			// Selection was cancelled
			return NvimDirectoryPickerCancelledMsg{}
		}

		// Validate that it's a git repository
		if !git.IsGitRepo(selectedPath) {
			return NvimDirectoryPickerErrorMsg{Error: fmt.Errorf("selected directory is not a git repository: %s", selectedPath)}
		}

		// Success
		return NvimDirectorySelectedMsg{Path: selectedPath}
	})
}

// Cleanup removes any temporary files
func (ndp *NvimDirectoryPicker) Cleanup() {
	if ndp.tempFile != "" {
		os.Remove(ndp.tempFile)
		ndp.tempFile = ""
	}
}

// IsNvimAvailable checks if neovim and Oil.nvim are available
func IsNvimAvailable() bool {
	// Check if nvim is in PATH
	_, err := exec.LookPath("nvim")
	if err != nil {
		return false
	}

	// Check if Oil.nvim is available (basic check)
	cmd := exec.Command("nvim", "--headless", "-c", "lua require('oil')", "-c", "qa")
	err = cmd.Run()
	return err == nil
}

// GetNvimSetupInstructions returns instructions for setting up Oil.nvim
func GetNvimSetupInstructions() string {
	return `To use the directory picker, you need:

1. Neovim installed and available in PATH
2. Oil.nvim plugin installed

To install Oil.nvim, add this to your Neovim config:

-- Using lazy.nvim
{
  'stevearc/oil.nvim',
  opts = {},
  dependencies = { "nvim-tree/nvim-web-devicons" },
}

-- Using packer.nvim
use {
  'stevearc/oil.nvim',
  config = function()
    require("oil").setup()
  end,
}

-- Using vim-plug
Plug 'stevearc/oil.nvim'

Then restart Neovim and run :Lazy sync (for lazy.nvim) or :PackerSync (for packer).

Visit https://github.com/stevearc/oil.nvim for more details.`
}