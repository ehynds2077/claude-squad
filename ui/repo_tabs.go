package ui

import (
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RepoTabs manages the repository tab display and navigation
type RepoTabs struct {
	repos       []string // List of repository paths
	repoNames   []string // List of repository display names
	selectedIdx int      // Currently selected repository index
	width       int      // Available width for tabs
}

// Tab styling - consistent with main title styling in list.go
var (
	repoActiveTabStyle = lipgloss.NewStyle().
			Background(lipgloss.Color("62")).
			Foreground(lipgloss.Color("230")).
			Padding(0, 2).
			Bold(true)

	repoInactiveTabStyle = lipgloss.NewStyle().
				Background(lipgloss.AdaptiveColor{Light: "#e8e8e8", Dark: "#333333"}).
				Foreground(lipgloss.AdaptiveColor{Light: "#666666", Dark: "#999999"}).
				Padding(0, 2)

	repoTabSeparatorStyle = lipgloss.NewStyle().
				Foreground(lipgloss.AdaptiveColor{Light: "#cccccc", Dark: "#444444"})
	
	repoTabBarBackgroundStyle = lipgloss.NewStyle().
					Background(lipgloss.AdaptiveColor{Light: "#f5f5f5", Dark: "#1a1a1a"}).
					Padding(0, 0, 0, 0)
)

// NewRepoTabs creates a new repository tabs component
func NewRepoTabs() *RepoTabs {
	return &RepoTabs{
		repos:       []string{},
		repoNames:   []string{},
		selectedIdx: 0,
		width:       0,
	}
}

// SetWidth sets the available width for the tabs
func (rt *RepoTabs) SetWidth(width int) {
	rt.width = width
}

// AddRepo adds a repository to the tabs
func (rt *RepoTabs) AddRepo(repoPath string) {
	// Check if repo already exists
	for _, existing := range rt.repos {
		if existing == repoPath {
			return
		}
	}
	
	rt.repos = append(rt.repos, repoPath)
	rt.repoNames = append(rt.repoNames, rt.getRepoDisplayName(repoPath))
}

// RemoveRepo removes a repository from the tabs
func (rt *RepoTabs) RemoveRepo(repoPath string) {
	for i, repo := range rt.repos {
		if repo == repoPath {
			rt.repos = append(rt.repos[:i], rt.repos[i+1:]...)
			rt.repoNames = append(rt.repoNames[:i], rt.repoNames[i+1:]...)
			
			// Adjust selected index if necessary
			if rt.selectedIdx >= len(rt.repos) && len(rt.repos) > 0 {
				rt.selectedIdx = len(rt.repos) - 1
			} else if len(rt.repos) == 0 {
				rt.selectedIdx = 0
			}
			break
		}
	}
}

// GetSelectedRepo returns the currently selected repository path
func (rt *RepoTabs) GetSelectedRepo() string {
	if rt.selectedIdx >= 0 && rt.selectedIdx < len(rt.repos) {
		return rt.repos[rt.selectedIdx]
	}
	return ""
}

// GetSelectedRepoName returns the currently selected repository display name
func (rt *RepoTabs) GetSelectedRepoName() string {
	if rt.selectedIdx >= 0 && rt.selectedIdx < len(rt.repoNames) {
		return rt.repoNames[rt.selectedIdx]
	}
	return ""
}

// HasRepos returns true if there are any repositories in the tabs
func (rt *RepoTabs) HasRepos() bool {
	return len(rt.repos) > 0
}

// NumRepos returns the number of repositories
func (rt *RepoTabs) NumRepos() int {
	return len(rt.repos)
}

// SelectRepo sets the selected repository by path
func (rt *RepoTabs) SelectRepo(repoPath string) {
	for i, repo := range rt.repos {
		if repo == repoPath {
			rt.selectedIdx = i
			break
		}
	}
}

// NextRepo moves to the next repository tab (K key)
func (rt *RepoTabs) NextRepo() {
	if len(rt.repos) > 0 {
		rt.selectedIdx = (rt.selectedIdx + 1) % len(rt.repos)
	}
}

// PrevRepo moves to the previous repository tab (J key)
func (rt *RepoTabs) PrevRepo() {
	if len(rt.repos) > 0 {
		rt.selectedIdx = (rt.selectedIdx - 1 + len(rt.repos)) % len(rt.repos)
	}
}

// getRepoDisplayName extracts a display name from a repository path
func (rt *RepoTabs) getRepoDisplayName(repoPath string) string {
	if repoPath == "" {
		return "unknown"
	}
	
	// Use the last directory name as the display name
	return filepath.Base(repoPath)
}

// Render renders the repository tabs
func (rt *RepoTabs) Render() string {
	if len(rt.repos) == 0 {
		return ""
	}

	// If only one repo, don't show tabs
	if len(rt.repos) == 1 {
		return ""
	}

	var tabs []string
	availableWidth := rt.width - 8 // Leave margin for borders and spacing

	// Calculate maximum width per tab, accounting for separators
	separatorSpace := (len(rt.repos) - 1) * 3 // Space for separators (" | ")
	var maxTabWidth int
	if availableWidth <= separatorSpace {
		// If terminal is too narrow, just show first character of each repo
		maxTabWidth = 5
	} else {
		maxTabWidth = (availableWidth - separatorSpace) / len(rt.repos)
		if maxTabWidth < 8 { // Minimum tab width (allowing for reasonable names)
			maxTabWidth = 8
		}
	}

	for i, repoName := range rt.repoNames {
		// Truncate repo name if it's too long
		displayName := repoName
		maxNameLength := maxTabWidth - 4 // Account for padding
		if maxNameLength > 0 && len(displayName) > maxNameLength {
			if maxNameLength <= 3 {
				displayName = displayName[:maxNameLength] // Show as much as possible
			} else {
				displayName = displayName[:maxNameLength-3] + "..."
			}
		}

		if i == rt.selectedIdx {
			tabs = append(tabs, repoActiveTabStyle.Render(displayName))
		} else {
			tabs = append(tabs, repoInactiveTabStyle.Render(displayName))
		}
	}

	// Join tabs with separators
	tabsStr := strings.Join(tabs, repoTabSeparatorStyle.Render(" | "))

	// Create the full tab bar with background
	tabBar := lipgloss.Place(rt.width, 1, lipgloss.Center, lipgloss.Center, tabsStr)
	
	// Apply background to the entire width
	return repoTabBarBackgroundStyle.Width(rt.width).Render(tabBar)
}

// GetAllRepos returns all repository paths
func (rt *RepoTabs) GetAllRepos() []string {
	return rt.repos
}

// SetRepos sets the list of repositories (used for initialization)
func (rt *RepoTabs) SetRepos(repos []string) {
	rt.repos = make([]string, len(repos))
	rt.repoNames = make([]string, len(repos))
	
	copy(rt.repos, repos)
	for i, repo := range repos {
		rt.repoNames[i] = rt.getRepoDisplayName(repo)
	}
	
	// Reset selection to first repo
	if len(rt.repos) > 0 {
		rt.selectedIdx = 0
	}
}

// ShouldShowTabs returns true if tabs should be displayed (more than one repo)
func (rt *RepoTabs) ShouldShowTabs() bool {
	return len(rt.repos) > 1
}

// GetHeight returns the height this component will take when rendered
func (rt *RepoTabs) GetHeight() int {
	if rt.ShouldShowTabs() {
		return 1 // One line for the tabs
	}
	return 0
}

// SetSelectedIndex sets the selected tab index directly (bounds-checked)
func (rt *RepoTabs) SetSelectedIndex(idx int) {
	if idx >= 0 && idx < len(rt.repos) {
		rt.selectedIdx = idx
	}
}

// GetSelectedIndex returns the currently selected tab index
func (rt *RepoTabs) GetSelectedIndex() int {
	return rt.selectedIdx
}