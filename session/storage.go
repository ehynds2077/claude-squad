package session

import (
	"claude-squad/config"
	"encoding/json"
	"fmt"
	"time"
)

// InstanceData represents the serializable data of an Instance
type InstanceData struct {
	Title        string    `json:"title"`
	Path         string    `json:"path"`
	Branch       string    `json:"branch"`
	Status       Status    `json:"status"`
	Height       int       `json:"height"`
	Width        int       `json:"width"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	AutoYes      bool      `json:"auto_yes"`
	// RepositoryPath is the absolute path to the repository root this instance belongs to
	RepositoryPath string `json:"repository_path"`

	Program   string          `json:"program"`
	Worktree  GitWorktreeData `json:"worktree"`
	DiffStats DiffStatsData   `json:"diff_stats"`
}

// GitWorktreeData represents the serializable data of a GitWorktree
type GitWorktreeData struct {
	RepoPath      string `json:"repo_path"`
	WorktreePath  string `json:"worktree_path"`
	SessionName   string `json:"session_name"`
	BranchName    string `json:"branch_name"`
	BaseCommitSHA string `json:"base_commit_sha"`
}

// DiffStatsData represents the serializable data of a DiffStats
type DiffStatsData struct {
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
	Content string `json:"content"`
}

// Storage handles saving and loading instances using the state interface
type Storage struct {
	state config.StateManager
}

// NewStorage creates a new storage instance
func NewStorage(state config.StateManager) (*Storage, error) {
	return &Storage{
		state: state,
	}, nil
}

// SaveInstances saves the list of instances to disk
func (s *Storage) SaveInstances(instances []*Instance) error {
	// Convert instances to InstanceData
	data := make([]InstanceData, 0)
	for _, instance := range instances {
		if instance.Started() {
			data = append(data, instance.ToInstanceData())
		}
	}

	// Marshal to JSON
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal instances: %w", err)
	}

	return s.state.SaveInstances(jsonData)
}

// LoadInstances loads the list of instances from disk
func (s *Storage) LoadInstances() ([]*Instance, error) {
	jsonData := s.state.GetInstances()

	var instancesData []InstanceData
	if err := json.Unmarshal(jsonData, &instancesData); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instances: %w", err)
	}

	instances := make([]*Instance, len(instancesData))
	for i, data := range instancesData {
		instance, err := FromInstanceData(data)
		if err != nil {
			return nil, fmt.Errorf("failed to create instance %s: %w", data.Title, err)
		}
		instances[i] = instance
	}

	return instances, nil
}

// DeleteInstance removes an instance from storage
func (s *Storage) DeleteInstance(title string) error {
	instances, err := s.LoadInstances()
	if err != nil {
		return fmt.Errorf("failed to load instances: %w", err)
	}

	found := false
	newInstances := make([]*Instance, 0)
	for _, instance := range instances {
		data := instance.ToInstanceData()
		if data.Title != title {
			newInstances = append(newInstances, instance)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("instance not found: %s", title)
	}

	return s.SaveInstances(newInstances)
}

// UpdateInstance updates an existing instance in storage
func (s *Storage) UpdateInstance(instance *Instance) error {
	instances, err := s.LoadInstances()
	if err != nil {
		return fmt.Errorf("failed to load instances: %w", err)
	}

	data := instance.ToInstanceData()
	found := false
	for i, existing := range instances {
		existingData := existing.ToInstanceData()
		if existingData.Title == data.Title {
			instances[i] = instance
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("instance not found: %s", data.Title)
	}

	return s.SaveInstances(instances)
}

// DeleteAllInstances removes all stored instances
func (s *Storage) DeleteAllInstances() error {
	return s.state.DeleteAllInstances()
}

// Repository-aware instance operations

// LoadInstancesForRepository loads instances that belong to a specific repository
func (s *Storage) LoadInstancesForRepository(repositoryPath string) ([]*Instance, error) {
	allInstances, err := s.LoadInstances()
	if err != nil {
		return nil, err
	}
	
	var filteredInstances []*Instance
	for _, instance := range allInstances {
		data := instance.ToInstanceData()
		if data.RepositoryPath == repositoryPath {
			filteredInstances = append(filteredInstances, instance)
		}
	}
	
	return filteredInstances, nil
}

// GetInstanceCountByRepository returns the number of instances per repository
func (s *Storage) GetInstanceCountByRepository() (map[string]int, error) {
	allInstances, err := s.LoadInstances()
	if err != nil {
		return nil, err
	}
	
	counts := make(map[string]int)
	for _, instance := range allInstances {
		data := instance.ToInstanceData()
		if data.RepositoryPath != "" {
			counts[data.RepositoryPath]++
		}
	}
	
	return counts, nil
}

// UpdateInstanceCounts updates the instance counts for all repositories in state
func (s *Storage) UpdateInstanceCounts() error {
	counts, err := s.GetInstanceCountByRepository()
	if err != nil {
		return fmt.Errorf("failed to get instance counts: %w", err)
	}
	
	// Update each repository's instance count
	repos := s.state.GetRepositories()
	for _, repo := range repos {
		count := counts[repo.Path]
		if err := s.state.UpdateRepositoryInstanceCount(repo.Path, count); err != nil {
			return fmt.Errorf("failed to update instance count for %s: %w", repo.Path, err)
		}
	}
	
	return nil
}

// CleanupOrphanedInstances removes instances whose repositories no longer exist
func (s *Storage) CleanupOrphanedInstances() error {
	allInstances, err := s.LoadInstances()
	if err != nil {
		return fmt.Errorf("failed to load instances: %w", err)
	}
	
	repos := s.state.GetRepositories()
	repoMap := make(map[string]bool)
	for _, repo := range repos {
		repoMap[repo.Path] = true
	}
	
	var validInstances []*Instance
	orphanedCount := 0
	
	for _, instance := range allInstances {
		data := instance.ToInstanceData()
		// Keep instances that either have no repository association (legacy)
		// or whose repository still exists
		if data.RepositoryPath == "" || repoMap[data.RepositoryPath] {
			validInstances = append(validInstances, instance)
		} else {
			orphanedCount++
		}
	}
	
	if orphanedCount > 0 {
		if err := s.SaveInstances(validInstances); err != nil {
			return fmt.Errorf("failed to save cleaned instances: %w", err)
		}
	}
	
	return nil
}

// AssociateInstanceWithRepository associates an instance with a repository
// This updates the repository path for an existing instance
func (s *Storage) AssociateInstanceWithRepository(instanceTitle, repositoryPath string) error {
	allInstances, err := s.LoadInstances()
	if err != nil {
		return fmt.Errorf("failed to load instances: %w", err)
	}
	
	found := false
	for _, instance := range allInstances {
		data := instance.ToInstanceData()
		if data.Title == instanceTitle {
			// Update the instance's repository path
			instance.RepositoryPath = repositoryPath
			found = true
			break
		}
	}
	
	if !found {
		return fmt.Errorf("instance not found: %s", instanceTitle)
	}
	
	// Save updated instances
	return s.SaveInstances(allInstances)
}

// MigrateInstanceRepositoryPaths attempts to set repository paths for instances that don't have them
func (s *Storage) MigrateInstanceRepositoryPaths() error {
	allInstances, err := s.LoadInstances()
	if err != nil {
		return fmt.Errorf("failed to load instances: %w", err)
	}
	
	updated := false
	for _, instance := range allInstances {
		if instance.RepositoryPath == "" {
			// Try to determine repository path from instance path
			repoPath, err := s.DetermineRepositoryPath(instance.Path)
			if err == nil {
				instance.RepositoryPath = repoPath
				updated = true
			}
		}
	}
	
	if updated {
		return s.SaveInstances(allInstances)
	}
	
	return nil
}

// Helper function to determine repository path from instance path
func (s *Storage) DetermineRepositoryPath(instancePath string) (string, error) {
	return config.FindRepositoryForPath(instancePath)
}

// Repository management methods - delegate to state (direct cast needed)

// GetRepository returns a specific repository by path
func (s *Storage) GetRepository(path string) (*config.RepositoryData, error) {
	if state, ok := s.state.(*config.State); ok {
		return state.GetRepository(path)
	}
	return nil, fmt.Errorf("invalid state type")
}

// AddRepository adds a new repository to the state
func (s *Storage) AddRepository(repo config.RepositoryData) error {
	if state, ok := s.state.(*config.State); ok {
		return state.AddRepository(repo)
	}
	return fmt.Errorf("invalid state type")
}

// UpdateRepositoryLastAccessed updates the last accessed time for a repository
func (s *Storage) UpdateRepositoryLastAccessed(path string) error {
	if state, ok := s.state.(*config.State); ok {
		return state.UpdateRepositoryLastAccessed(path)
	}
	return fmt.Errorf("invalid state type")
}

// CleanupInvalidRepositories removes repositories that no longer exist or are invalid
func (s *Storage) CleanupInvalidRepositories() (int, error) {
	if state, ok := s.state.(*config.State); ok {
		return state.CleanupInvalidRepositories()
	}
	return 0, fmt.Errorf("invalid state type")
}
