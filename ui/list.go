package ui

import (
	"claude-squad/log"
	"claude-squad/session"
	"errors"
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
)

const readyIcon = "* "
const pausedIcon = "|| "

var readyStyle = lipgloss.NewStyle().
	Foreground(lipgloss.AdaptiveColor{Light: "#51bd73", Dark: "#51bd73"})

var addedLinesStyle = lipgloss.NewStyle().
	Foreground(lipgloss.AdaptiveColor{Light: "#51bd73", Dark: "#51bd73"})

var removedLinesStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#de613e"))

var pausedStyle = lipgloss.NewStyle().
	Foreground(lipgloss.AdaptiveColor{Light: "#888888", Dark: "#888888"})

var titleStyle = lipgloss.NewStyle().
	Padding(1, 1, 0, 1).
	Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#dddddd"})

var listDescStyle = lipgloss.NewStyle().
	Padding(0, 1, 1, 1).
	Foreground(lipgloss.AdaptiveColor{Light: "#A49FA5", Dark: "#777777"})

var selectedTitleStyle = lipgloss.NewStyle().
	Padding(1, 1, 0, 1).
	Background(lipgloss.Color("#dde4f0")).
	Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#1a1a1a"})

var selectedDescStyle = lipgloss.NewStyle().
	Padding(0, 1, 1, 1).
	Background(lipgloss.Color("#dde4f0")).
	Foreground(lipgloss.AdaptiveColor{Light: "#1a1a1a", Dark: "#1a1a1a"})

var mainTitle = lipgloss.NewStyle().
	Background(lipgloss.Color("62")).
	Foreground(lipgloss.Color("230"))

var autoYesStyle = lipgloss.NewStyle().
	Background(lipgloss.Color("#dde4f0")).
	Foreground(lipgloss.Color("#1a1a1a"))

type List struct {
	items         []*session.Instance
	selectedIdx   int
	height, width int
	renderer      *InstanceRenderer
	autoyes       bool

	// map of repo name to number of instances using it. Used to display the repo name only if there are
	// multiple repos in play.
	repos map[string]int
	
	// Repository tabs component for managing multiple repositories
	repoTabs *RepoTabs
}

func NewList(spinner *spinner.Model, autoYes bool) *List {
	return &List{
		items:    []*session.Instance{},
		renderer: &InstanceRenderer{spinner: spinner},
		repos:    make(map[string]int),
		autoyes:  autoYes,
		repoTabs: NewRepoTabs(),
	}
}

// SetSize sets the height and width of the list.
func (l *List) SetSize(width, height int) {
	l.width = width
	l.height = height
	l.renderer.setWidth(width)
	l.repoTabs.SetWidth(width)
}

// SetSessionPreviewSize sets the height and width for the tmux sessions. This makes the stdout line have the correct
// width and height.
func (l *List) SetSessionPreviewSize(width, height int) (err error) {
	for i, item := range l.items {
		if !item.Started() || item.Paused() {
			continue
		}

		if innerErr := item.SetPreviewSize(width, height); innerErr != nil {
			err = errors.Join(
				err, fmt.Errorf("could not set preview size for instance %d: %v", i, innerErr))
		}
	}
	return
}

func (l *List) NumInstances() int {
	return len(l.items)
}

// InstanceRenderer handles rendering of session.Instance objects
type InstanceRenderer struct {
	spinner *spinner.Model
	width   int
}

func (r *InstanceRenderer) setWidth(width int) {
	r.width = AdjustPreviewWidth(width)
}

const branchIcon = ">"

func (r *InstanceRenderer) Render(i *session.Instance, idx int, selected bool, hasMultipleRepos bool) string {
	prefix := fmt.Sprintf(" %d. ", idx)
	if idx >= 10 {
		prefix = prefix[:len(prefix)-1]
	}
	titleS := selectedTitleStyle
	descS := selectedDescStyle
	if !selected {
		titleS = titleStyle
		descS = listDescStyle
	}

	// add spinner next to title if it's running
	var join string
	switch i.Status {
	case session.Running:
		join = fmt.Sprintf("%s ", r.spinner.View())
	case session.Ready:
		join = readyStyle.Render(readyIcon)
	case session.Paused:
		join = pausedStyle.Render(pausedIcon)
	default:
	}

	// Cut the title if it's too long
	titleText := i.Title
	widthAvail := r.width - 3 - len(prefix) - 1
	if widthAvail > 0 && widthAvail < len(titleText) && len(titleText) >= widthAvail-3 {
		titleText = titleText[:widthAvail-3] + "..."
	}
	title := titleS.Render(lipgloss.JoinHorizontal(
		lipgloss.Left,
		lipgloss.Place(r.width-3, 1, lipgloss.Left, lipgloss.Center, fmt.Sprintf("%s %s", prefix, titleText)),
		" ",
		join,
	))

	stat := i.GetDiffStats()

	var diff string
	var addedDiff, removedDiff string
	if stat == nil || stat.Error != nil || stat.IsEmpty() {
		// Don't show diff stats if there's an error or if they don't exist
		addedDiff = ""
		removedDiff = ""
		diff = ""
	} else {
		addedDiff = fmt.Sprintf("+%d", stat.Added)
		removedDiff = fmt.Sprintf("-%d ", stat.Removed)
		diff = lipgloss.JoinHorizontal(
			lipgloss.Center,
			addedLinesStyle.Background(descS.GetBackground()).Render(addedDiff),
			lipgloss.Style{}.Background(descS.GetBackground()).Foreground(descS.GetForeground()).Render(","),
			removedLinesStyle.Background(descS.GetBackground()).Render(removedDiff),
		)
	}

	remainingWidth := r.width
	remainingWidth -= len(prefix)
	remainingWidth -= len(branchIcon)

	diffWidth := len(addedDiff) + len(removedDiff)
	if diffWidth > 0 {
		diffWidth += 1
	}

	// Use fixed width for diff stats to avoid layout issues
	remainingWidth -= diffWidth

	branch := i.Branch
	if i.Started() && hasMultipleRepos {
		repoName, err := i.RepoName()
		if err != nil {
			log.ErrorLog.Printf("could not get repo name in instance renderer: %v", err)
		} else {
			branch += fmt.Sprintf(" (%s)", repoName)
		}
	}
	// Don't show branch if there's no space for it. Or show ellipsis if it's too long.
	if remainingWidth < 0 {
		branch = ""
	} else if remainingWidth < len(branch) {
		if remainingWidth < 3 {
			branch = ""
		} else {
			// We know the remainingWidth is at least 4 and branch is longer than that, so this is safe.
			branch = branch[:remainingWidth-3] + "..."
		}
	}
	remainingWidth -= len(branch)

	// Add spaces to fill the remaining width.
	spaces := ""
	if remainingWidth > 0 {
		spaces = strings.Repeat(" ", remainingWidth)
	}

	branchLine := fmt.Sprintf("%s %s-%s%s%s", strings.Repeat(" ", len(prefix)), branchIcon, branch, spaces, diff)

	// join title and subtitle
	text := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		descS.Render(branchLine),
	)

	return text
}

func (l *List) String() string {
	const titleText = " Instances "
	const autoYesText = " auto-yes "

	// Write the title.
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString("\n")

	// Render repository tabs if there are multiple repos
	if l.repoTabs.ShouldShowTabs() {
		tabsContent := l.repoTabs.Render()
		if tabsContent != "" {
			b.WriteString(tabsContent)
			b.WriteString("\n")
		}
	}

	// Write title line
	// add padding of 2 because the border on list items adds some extra characters
	titleWidth := AdjustPreviewWidth(l.width) + 2
	if !l.autoyes {
		b.WriteString(lipgloss.Place(
			titleWidth, 1, lipgloss.Left, lipgloss.Bottom, mainTitle.Render(titleText)))
	} else {
		title := lipgloss.Place(
			titleWidth/2, 1, lipgloss.Left, lipgloss.Bottom, mainTitle.Render(titleText))
		autoYes := lipgloss.Place(
			titleWidth-(titleWidth/2), 1, lipgloss.Right, lipgloss.Bottom, autoYesStyle.Render(autoYesText))
		b.WriteString(lipgloss.JoinHorizontal(
			lipgloss.Top, title, autoYes))
	}

	b.WriteString("\n")
	b.WriteString("\n")

	// Get filtered instances based on selected repository
	filteredItems := l.GetFilteredInstances()
	
	// Render the filtered list
	for i, item := range filteredItems {
		// Find the original index for selection highlighting
		originalIdx := -1
		for j, originalItem := range l.items {
			if originalItem == item {
				originalIdx = j
				break
			}
		}
		
		isSelected := originalIdx == l.selectedIdx
		b.WriteString(l.renderer.Render(item, i+1, isSelected, len(l.repos) > 1))
		if i != len(filteredItems)-1 {
			b.WriteString("\n\n")
		}
	}
	
	// Add empty lines at the end if we have space
	if len(filteredItems) == 0 && l.repoTabs.ShouldShowTabs() {
		b.WriteString("\n")
		b.WriteString(lipgloss.NewStyle().
			Foreground(lipgloss.AdaptiveColor{Light: "#999999", Dark: "#666666"}).
			Render("  No instances in this repository"))
	}
	
	return lipgloss.Place(l.width, l.height, lipgloss.Left, lipgloss.Top, b.String())
}

// Down selects the next item in the list.
func (l *List) Down() {
	filteredItems := l.GetFilteredInstances()
	if len(filteredItems) == 0 {
		return
	}
	
	// Ensure selection is valid first
	if l.selectedIdx >= len(l.items) {
		l.selectedIdx = 0
	}
	
	// Find current position in filtered list
	currentFilteredIdx := -1
	for i, item := range filteredItems {
		if l.items[l.selectedIdx] == item {
			currentFilteredIdx = i
			break
		}
	}
	
	// If current item is not in filtered list, select first filtered item
	if currentFilteredIdx == -1 {
		if len(filteredItems) > 0 {
			firstItem := filteredItems[0]
			for i, item := range l.items {
				if item == firstItem {
					l.selectedIdx = i
					break
				}
			}
		}
		return
	}
	
	// Move to next item in filtered list
	if currentFilteredIdx < len(filteredItems)-1 {
		nextItem := filteredItems[currentFilteredIdx+1]
		// Find original index of next item
		for i, item := range l.items {
			if item == nextItem {
				l.selectedIdx = i
				break
			}
		}
	}
}

// Kill selects the next item in the list.
func (l *List) Kill() {
	if len(l.items) == 0 {
		return
	}
	targetInstance := l.items[l.selectedIdx]

	// Kill the tmux session
	if err := targetInstance.Kill(); err != nil {
		log.ErrorLog.Printf("could not kill instance: %v", err)
	}

	// If you delete the last one in the list, select the previous one.
	if l.selectedIdx == len(l.items)-1 {
		defer l.Up()
	}

	// Unregister the repository path.
	gitWorktree, err := targetInstance.GetGitWorktree()
	if err != nil {
		log.ErrorLog.Printf("could not get git worktree: %v", err)
	} else if gitWorktree != nil {
		l.rmRepo(gitWorktree.GetRepoPath())
	}

	// Since there's items after this, the selectedIdx can stay the same.
	l.items = append(l.items[:l.selectedIdx], l.items[l.selectedIdx+1:]...)
}

func (l *List) Attach() (chan struct{}, error) {
	targetInstance := l.items[l.selectedIdx]
	return targetInstance.Attach()
}

func (l *List) AttachToTerminal() (chan struct{}, error) {
	targetInstance := l.items[l.selectedIdx]
	return targetInstance.AttachToTerminal()
}

// Up selects the prev item in the list.
func (l *List) Up() {
	filteredItems := l.GetFilteredInstances()
	if len(filteredItems) == 0 {
		return
	}
	
	// Ensure selection is valid first
	if l.selectedIdx >= len(l.items) {
		l.selectedIdx = 0
	}
	
	// Find current position in filtered list
	currentFilteredIdx := -1
	for i, item := range filteredItems {
		if l.items[l.selectedIdx] == item {
			currentFilteredIdx = i
			break
		}
	}
	
	// If current item is not in filtered list, select first filtered item
	if currentFilteredIdx == -1 {
		if len(filteredItems) > 0 {
			firstItem := filteredItems[0]
			for i, item := range l.items {
				if item == firstItem {
					l.selectedIdx = i
					break
				}
			}
		}
		return
	}
	
	// Move to previous item in filtered list
	if currentFilteredIdx > 0 {
		prevItem := filteredItems[currentFilteredIdx-1]
		// Find original index of previous item
		for i, item := range l.items {
			if item == prevItem {
				l.selectedIdx = i
				break
			}
		}
	}
}

func (l *List) addRepo(repo string) {
	if _, ok := l.repos[repo]; !ok {
		l.repos[repo] = 0
	}
	l.repos[repo]++
	
	// Update repository tabs
	l.repoTabs.AddRepo(repo)
	
	// Ensure valid selection after adding repo
	l.EnsureValidSelection()
}

func (l *List) rmRepo(repo string) {
	if _, ok := l.repos[repo]; !ok {
		log.ErrorLog.Printf("repo %s not found", repo)
		return
	}
	l.repos[repo]--
	if l.repos[repo] == 0 {
		delete(l.repos, repo)
		// Remove from repository tabs
		l.repoTabs.RemoveRepo(repo)
		
		// Ensure valid selection after removing repo
		l.EnsureValidSelection()
	}
}

// AddInstance adds a new instance to the list. It returns a finalizer function that should be called when the instance
// is started. If the instance was restored from storage or is paused, you can call the finalizer immediately.
// When creating a new one and entering the name, you want to call the finalizer once the name is done.
func (l *List) AddInstance(instance *session.Instance) (finalize func()) {
	l.items = append(l.items, instance)
	// The finalizer registers the repo path once the instance is started.
	return func() {
		gitWorktree, err := instance.GetGitWorktree()
		if err != nil {
			log.ErrorLog.Printf("could not get git worktree: %v", err)
			return
		}
		if gitWorktree == nil {
			log.ErrorLog.Printf("git worktree is nil")
			return
		}

		l.addRepo(gitWorktree.GetRepoPath())
	}
}

// GetSelectedInstance returns the currently selected instance
func (l *List) GetSelectedInstance() *session.Instance {
	if len(l.items) == 0 {
		return nil
	}
	return l.items[l.selectedIdx]
}

// SetSelectedInstance sets the selected index. Noop if the index is out of bounds.
func (l *List) SetSelectedInstance(idx int) {
	if idx >= len(l.items) {
		return
	}
	l.selectedIdx = idx
}

// GetInstances returns all instances in the list
func (l *List) GetInstances() []*session.Instance {
	return l.items
}

// GetRepoTabs returns the repository tabs component
func (l *List) GetRepoTabs() *RepoTabs {
	return l.repoTabs
}

// GetFilteredInstances returns instances filtered by the currently selected repository
func (l *List) GetFilteredInstances() []*session.Instance {
	if !l.repoTabs.ShouldShowTabs() {
		return l.items
	}
	
	selectedRepo := l.repoTabs.GetSelectedRepo()
	if selectedRepo == "" {
		return l.items
	}
	
	var filtered []*session.Instance
	for _, instance := range l.items {
		if instance.Started() {
			// Get the git worktree to access the full repository path
			gitWorktree, err := instance.GetGitWorktree()
			if err != nil {
				log.ErrorLog.Printf("could not get git worktree for filtering: %v", err)
				continue
			}
			if gitWorktree != nil && gitWorktree.GetRepoPath() == selectedRepo {
				filtered = append(filtered, instance)
			}
		} else {
			// Include non-started instances as they don't have a repository yet
			filtered = append(filtered, instance)
		}
	}
	
	return filtered
}

// EnsureValidSelection ensures the current selection is visible in the filtered view
func (l *List) EnsureValidSelection() {
	filteredItems := l.GetFilteredInstances()
	if len(filteredItems) == 0 {
		return
	}
	
	// Bounds check
	if l.selectedIdx >= len(l.items) {
		l.selectedIdx = 0
	}
	
	// Check if current selection is in filtered items
	currentItem := l.items[l.selectedIdx]
	for _, item := range filteredItems {
		if item == currentItem {
			return // Current selection is valid
		}
	}
	
	// Current selection is not visible, select first filtered item
	firstItem := filteredItems[0]
	for i, item := range l.items {
		if item == firstItem {
			l.selectedIdx = i
			break
		}
	}
}

// NextRepo switches to the next repository tab
func (l *List) NextRepo() {
	if l.repoTabs.ShouldShowTabs() {
		l.repoTabs.NextRepo()
		l.EnsureValidSelection()
	}
}

// PrevRepo switches to the previous repository tab
func (l *List) PrevRepo() {
	if l.repoTabs.ShouldShowTabs() {
		l.repoTabs.PrevRepo()
		l.EnsureValidSelection()
	}
}

// GetCurrentRepoName returns the display name of the currently selected repository
func (l *List) GetCurrentRepoName() string {
	if l.repoTabs.ShouldShowTabs() {
		return l.repoTabs.GetSelectedRepoName()
	}
	return ""
}

// GetCurrentRepoPath returns the full path of the currently selected repository
func (l *List) GetCurrentRepoPath() string {
	if l.repoTabs.ShouldShowTabs() {
		return l.repoTabs.GetSelectedRepo()
	}
	return ""
}

// HasMultipleRepos returns true if there are multiple repositories being managed
func (l *List) HasMultipleRepos() bool {
	return l.repoTabs.ShouldShowTabs()
}
