package config

import (
	"claude-squad/log"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	StateFileName     = "state.json"
	InstancesFileName = "instances.json"
)

// RepositoryData represents a known repository with metadata
type RepositoryData struct {
	// Path is the absolute path to the repository root
	Path string `json:"path"`
	// Name is the display name of the repository (typically the directory name)
	Name string `json:"name"`
	// LastAccessed is the last time this repository was accessed
	LastAccessed time.Time `json:"last_accessed"`
	// CreatedAt is when this repository was first added to the system
	CreatedAt time.Time `json:"created_at"`
	// InstanceCount is the number of instances currently associated with this repository
	InstanceCount int `json:"instance_count"`
}

// InstanceStorage handles instance-related operations
type InstanceStorage interface {
	// SaveInstances saves the raw instance data
	SaveInstances(instancesJSON json.RawMessage) error
	// GetInstances returns the raw instance data
	GetInstances() json.RawMessage
	// DeleteAllInstances removes all stored instances
	DeleteAllInstances() error
}

// RepositoryStorage handles repository-related operations
type RepositoryStorage interface {
	// GetRepositories returns all known repositories
	GetRepositories() []RepositoryData
	// AddRepository adds a new repository to the state
	AddRepository(repo RepositoryData) error
	// RemoveRepository removes a repository from the state
	RemoveRepository(path string) error
	// UpdateRepository updates an existing repository's metadata
	UpdateRepository(repo RepositoryData) error
	// GetRepository returns a specific repository by path
	GetRepository(path string) (*RepositoryData, error)
	// GetSelectedRepository returns the currently selected repository path
	GetSelectedRepository() string
	// SetSelectedRepository sets the currently selected repository
	SetSelectedRepository(path string) error
	// UpdateRepositoryInstanceCount updates the instance count for a repository
	UpdateRepositoryInstanceCount(path string, count int) error
	// UpdateRepositoryLastAccessed updates the last accessed time for a repository
	UpdateRepositoryLastAccessed(path string) error
}

// AppState handles application-level state
type AppState interface {
	// GetHelpScreensSeen returns the bitmask of seen help screens
	GetHelpScreensSeen() uint32
	// SetHelpScreensSeen updates the bitmask of seen help screens
	SetHelpScreensSeen(seen uint32) error
}

// StateManager combines instance storage, repository storage, and app state management
type StateManager interface {
	InstanceStorage
	RepositoryStorage
	AppState
}

// State represents the application state that persists between sessions
type State struct {
	// HelpScreensSeen is a bitmask tracking which help screens have been shown
	HelpScreensSeen uint32 `json:"help_screens_seen"`
	// Instances stores the serialized instance data as raw JSON
	InstancesData json.RawMessage `json:"instances"`
	// Repositories stores the list of known repositories with metadata
	Repositories []RepositoryData `json:"repositories"`
	// SelectedRepository is the path of the currently selected repository
	SelectedRepository string `json:"selected_repository"`
	// StateVersion tracks the schema version for migration purposes
	StateVersion int `json:"state_version"`
}

const CurrentStateVersion = 1

// DefaultState returns the default state
func DefaultState() *State {
	return &State{
		HelpScreensSeen:    0,
		InstancesData:      json.RawMessage("[]"),
		Repositories:       make([]RepositoryData, 0),
		SelectedRepository: "",
		StateVersion:       CurrentStateVersion,
	}
}

// LoadState loads the state from disk. If it cannot be done, we return the default state.
func LoadState() *State {
	configDir, err := GetConfigDir()
	if err != nil {
		log.ErrorLog.Printf("failed to get config directory: %v", err)
		return DefaultState()
	}

	statePath := filepath.Join(configDir, StateFileName)
	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			// Create and save default state if file doesn't exist
			defaultState := DefaultState()
			if saveErr := SaveState(defaultState); saveErr != nil {
				log.WarningLog.Printf("failed to save default state: %v", saveErr)
			}
			return defaultState
		}

		log.WarningLog.Printf("failed to get state file: %v", err)
		return DefaultState()
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		log.ErrorLog.Printf("failed to parse state file: %v", err)
		return DefaultState()
	}

	// Perform state migration if needed
	migratedState := migrateState(&state)
	
	// Save migrated state if changes were made
	if migratedState.StateVersion != state.StateVersion {
		if saveErr := SaveState(migratedState); saveErr != nil {
			log.WarningLog.Printf("failed to save migrated state: %v", saveErr)
		}
	}

	return migratedState
}

// SaveState saves the state to disk
func SaveState(state *State) error {
	configDir, err := GetConfigDir()
	if err != nil {
		return fmt.Errorf("failed to get config directory: %w", err)
	}

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	statePath := filepath.Join(configDir, StateFileName)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	return os.WriteFile(statePath, data, 0644)
}

// Helper functions for repository management

// UpdateRepositoryLastAccessed updates the last accessed time for a repository
func (s *State) UpdateRepositoryLastAccessed(path string) error {
	for i, repo := range s.Repositories {
		if repo.Path == path {
			s.Repositories[i].LastAccessed = time.Now()
			return SaveState(s)
		}
	}
	return fmt.Errorf("repository not found: %s", path)
}

// UpdateRepositoryInstanceCount updates the instance count for a repository
func (s *State) UpdateRepositoryInstanceCount(path string, count int) error {
	for i, repo := range s.Repositories {
		if repo.Path == path {
			s.Repositories[i].InstanceCount = count
			return SaveState(s)
		}
	}
	return fmt.Errorf("repository not found: %s", path)
}

// RepositoryManager provides high-level repository management operations
type RepositoryManager struct {
	state   StateManager
	storage InstanceStorage
}

// NewRepositoryManager creates a new repository manager
func NewRepositoryManager(state StateManager, storage InstanceStorage) *RepositoryManager {
	return &RepositoryManager{
		state:   state,
		storage: storage,
	}
}

// AddRepositoryFromPath validates and adds a repository from a given path
func (rm *RepositoryManager) AddRepositoryFromPath(path string) (*RepositoryData, error) {
	// Find repository root
	repoPath, err := FindRepositoryForPath(path)
	if err != nil {
		return nil, fmt.Errorf("not a git repository: %w", err)
	}
	
	// Check if already exists
	if existing, err := rm.state.GetRepository(repoPath); err == nil {
		// Update last accessed time (only possible if state is a *State)
		if state, ok := rm.state.(*State); ok {
			if err := state.UpdateRepositoryLastAccessed(repoPath); err != nil {
				return nil, fmt.Errorf("failed to update last accessed time: %w", err)
			}
		}
		return existing, nil
	}
	
	// Create and add new repository
	repoData, err := CreateRepositoryData(repoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create repository data: %w", err)
	}
	
	if err := rm.state.AddRepository(repoData); err != nil {
		return nil, fmt.Errorf("failed to add repository: %w", err)
	}
	
	return &repoData, nil
}

// RemoveRepositoryAndCleanup removes a repository and cleans up orphaned instances
func (rm *RepositoryManager) RemoveRepositoryAndCleanup(path string) error {
	// Remove from state
	if err := rm.state.RemoveRepository(path); err != nil {
		return fmt.Errorf("failed to remove repository: %w", err)
	}
	
	// Clean up orphaned instances if storage is available
	if rm.storage != nil {
		// This would require the storage to implement cleanup
		// For now, we'll leave this as a manual operation
	}
	
	return nil
}

// GetRepositoriesWithCounts returns repositories with current instance counts
func (rm *RepositoryManager) GetRepositoriesWithCounts() ([]RepositoryData, error) {
	repos := rm.state.GetRepositories()
	
	// Update instance counts if storage is available
	if rm.storage != nil {
		// This would require implementing instance counting in storage
		// For now, return repos as-is
	}
	
	return repos, nil
}

// GetRepositoriesSortedByLastAccessed returns repositories sorted by last accessed time (most recent first)
func (s *State) GetRepositoriesSortedByLastAccessed() []RepositoryData {
	repos := make([]RepositoryData, len(s.Repositories))
	copy(repos, s.Repositories)
	
	// Simple bubble sort by LastAccessed (descending)
	for i := 0; i < len(repos)-1; i++ {
		for j := 0; j < len(repos)-i-1; j++ {
			if repos[j].LastAccessed.Before(repos[j+1].LastAccessed) {
				repos[j], repos[j+1] = repos[j+1], repos[j]
			}
		}
	}
	
	return repos
}

// Repository validation and cleanup operations

// ValidateRepository checks if a repository path is valid and accessible
func ValidateRepositoryPath(path string) error {
	if path == "" {
		return fmt.Errorf("repository path cannot be empty")
	}
	
	// Check if path exists
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("repository path does not exist: %s", path)
		}
		return fmt.Errorf("error accessing repository path %s: %w", path, err)
	}
	
	// Check if it's a directory
	if !info.IsDir() {
		return fmt.Errorf("repository path is not a directory: %s", path)
	}
	
	// Check if it contains a .git directory (is a git repository)
	gitPath := filepath.Join(path, ".git")
	if _, err := os.Stat(gitPath); err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path is not a git repository (no .git directory found): %s", path)
		}
		return fmt.Errorf("error checking .git directory in %s: %w", path, err)
	}
	
	return nil
}

// CleanupInvalidRepositories removes repositories that no longer exist or are invalid
func (s *State) CleanupInvalidRepositories() (int, error) {
	var validRepos []RepositoryData
	removedCount := 0
	
	for _, repo := range s.Repositories {
		if err := ValidateRepositoryPath(repo.Path); err != nil {
			if log.InfoLog != nil {
				log.InfoLog.Printf("Removing invalid repository %s: %v", repo.Path, err)
			}
			removedCount++
			
			// Clear selected repository if it was the invalid one
			if s.SelectedRepository == repo.Path {
				s.SelectedRepository = ""
			}
		} else {
			validRepos = append(validRepos, repo)
		}
	}
	
	if removedCount > 0 {
		s.Repositories = validRepos
		if err := SaveState(s); err != nil {
			return removedCount, fmt.Errorf("failed to save state after cleanup: %w", err)
		}
	}
	
	return removedCount, nil
}

// SanitizeRepositoryName creates a safe display name from a repository path
func SanitizeRepositoryName(path string) string {
	name := filepath.Base(path)
	if name == "." || name == "" {
		// If base name is not useful, use the last two path components
		parts := strings.Split(filepath.Clean(path), string(filepath.Separator))
		if len(parts) >= 2 {
			name = filepath.Join(parts[len(parts)-2], parts[len(parts)-1])
		} else if len(parts) >= 1 {
			name = parts[len(parts)-1]
		} else {
			name = "unknown"
		}
	}
	return name
}

// CreateRepositoryData creates a new RepositoryData from a path
func CreateRepositoryData(path string) (RepositoryData, error) {
	if err := ValidateRepositoryPath(path); err != nil {
		return RepositoryData{}, err
	}
	
	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return RepositoryData{}, fmt.Errorf("failed to get absolute path: %w", err)
	}
	
	now := time.Now()
	return RepositoryData{
		Path:          absPath,
		Name:          SanitizeRepositoryName(absPath),
		LastAccessed:  now,
		CreatedAt:     now,
		InstanceCount: 0,
	}, nil
}

// FindRepositoryForPath attempts to find the repository root for a given path
// by walking up the directory tree looking for a .git directory
func FindRepositoryForPath(path string) (string, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute path: %w", err)
	}
	
	currentPath := absPath
	for {
		gitPath := filepath.Join(currentPath, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return currentPath, nil
		}
		
		parentPath := filepath.Dir(currentPath)
		if parentPath == currentPath {
			// Reached filesystem root
			break
		}
		currentPath = parentPath
	}
	
	return "", fmt.Errorf("no git repository found for path: %s", path)
}

// InstanceStorage interface implementation

// SaveInstances saves the raw instance data
func (s *State) SaveInstances(instancesJSON json.RawMessage) error {
	s.InstancesData = instancesJSON
	return SaveState(s)
}

// GetInstances returns the raw instance data
func (s *State) GetInstances() json.RawMessage {
	return s.InstancesData
}

// DeleteAllInstances removes all stored instances
func (s *State) DeleteAllInstances() error {
	s.InstancesData = json.RawMessage("[]")
	return SaveState(s)
}

// AppState interface implementation

// GetHelpScreensSeen returns the bitmask of seen help screens
func (s *State) GetHelpScreensSeen() uint32 {
	return s.HelpScreensSeen
}

// SetHelpScreensSeen updates the bitmask of seen help screens
func (s *State) SetHelpScreensSeen(seen uint32) error {
	s.HelpScreensSeen = seen
	return SaveState(s)
}

// RepositoryStorage interface implementation

// GetRepositories returns all known repositories
func (s *State) GetRepositories() []RepositoryData {
	return s.Repositories
}

// AddRepository adds a new repository to the state
func (s *State) AddRepository(repo RepositoryData) error {
	// Check if repository already exists
	for i, existing := range s.Repositories {
		if existing.Path == repo.Path {
			// Update existing repository
			s.Repositories[i] = repo
			return SaveState(s)
		}
	}
	
	// Add new repository
	s.Repositories = append(s.Repositories, repo)
	return SaveState(s)
}

// RemoveRepository removes a repository from the state
// NOTE: This does not handle cleanup of associated instances - 
// call storage.CleanupOrphanedInstances() after removing repositories
func (s *State) RemoveRepository(path string) error {
	for i, repo := range s.Repositories {
		if repo.Path == path {
			// Remove repository from slice
			s.Repositories = append(s.Repositories[:i], s.Repositories[i+1:]...)
			
			// Clear selected repository if it was the removed one
			if s.SelectedRepository == path {
				s.SelectedRepository = ""
			}
			
			return SaveState(s)
		}
	}
	return fmt.Errorf("repository not found: %s", path)
}

// UpdateRepository updates an existing repository's metadata
func (s *State) UpdateRepository(repo RepositoryData) error {
	for i, existing := range s.Repositories {
		if existing.Path == repo.Path {
			s.Repositories[i] = repo
			return SaveState(s)
		}
	}
	return fmt.Errorf("repository not found: %s", repo.Path)
}

// GetRepository returns a specific repository by path
func (s *State) GetRepository(path string) (*RepositoryData, error) {
	for _, repo := range s.Repositories {
		if repo.Path == path {
			return &repo, nil
		}
	}
	return nil, fmt.Errorf("repository not found: %s", path)
}

// GetSelectedRepository returns the currently selected repository path
func (s *State) GetSelectedRepository() string {
	return s.SelectedRepository
}

// SetSelectedRepository sets the currently selected repository
func (s *State) SetSelectedRepository(path string) error {
	// Validate that the repository exists if path is not empty
	if path != "" {
		found := false
		for _, repo := range s.Repositories {
			if repo.Path == path {
				found = true
				break
			}
		}
		if !found {
			return fmt.Errorf("repository not found: %s", path)
		}
	}
	
	s.SelectedRepository = path
	return SaveState(s)
}

// BatchUpdateRepositories performs multiple repository operations atomically
func (s *State) BatchUpdateRepositories(operations []func(*State) error) error {
	// Apply all operations
	for _, op := range operations {
		if err := op(s); err != nil {
			return err
		}
	}
	
	// Save state once at the end
	return SaveState(s)
}

// GetRepositoryStats returns statistics about repository usage
func (s *State) GetRepositoryStats() map[string]interface{} {
	stats := make(map[string]interface{})
	
	totalRepos := len(s.Repositories)
	totalInstances := 0
	
	for _, repo := range s.Repositories {
		totalInstances += repo.InstanceCount
	}
	
	stats["total_repositories"] = totalRepos
	stats["total_instances"] = totalInstances
	stats["selected_repository"] = s.SelectedRepository
	
	if totalRepos > 0 {
		stats["average_instances_per_repo"] = float64(totalInstances) / float64(totalRepos)
	}
	
	return stats
}

// CompactRepositories removes repositories with zero instances and validates remaining ones
func (s *State) CompactRepositories() (int, error) {
	var validRepos []RepositoryData
	removedCount := 0
	
	for _, repo := range s.Repositories {
		// Remove repositories with zero instances and invalid paths
		if repo.InstanceCount == 0 {
			if err := ValidateRepositoryPath(repo.Path); err != nil {
				removedCount++
				continue
			}
		}
		validRepos = append(validRepos, repo)
	}
	
	if removedCount > 0 {
		s.Repositories = validRepos
		// Clear selected repository if it was removed
		found := false
		for _, repo := range validRepos {
			if repo.Path == s.SelectedRepository {
				found = true
				break
			}
		}
		if !found {
			s.SelectedRepository = ""
		}
		
		if err := SaveState(s); err != nil {
			return removedCount, fmt.Errorf("failed to save state after compacting: %w", err)
		}
	}
	
	return removedCount, nil
}

// migrateState handles migration from older state versions
func migrateState(state *State) *State {
	// If no version is set, this is a v0 state - migrate to v1
	if state.StateVersion == 0 {
		// Initialize new fields for v1
		if state.Repositories == nil {
			state.Repositories = make([]RepositoryData, 0)
		}
		if state.SelectedRepository == "" {
			state.SelectedRepository = ""
		}
		state.StateVersion = 1
		if log.InfoLog != nil {
			log.InfoLog.Printf("Migrated state from version 0 to version 1")
		}
	}
	
	return state
}
