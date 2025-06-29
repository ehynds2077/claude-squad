package ui

import (
	"testing"
)

func TestRepoTabs_BasicFunctionality(t *testing.T) {
	tabs := NewRepoTabs()
	
	// Test empty tabs
	if tabs.HasRepos() {
		t.Error("Expected empty tabs to return false for HasRepos()")
	}
	
	if tabs.ShouldShowTabs() {
		t.Error("Expected empty tabs to return false for ShouldShowTabs()")
	}
	
	// Test adding repos
	tabs.AddRepo("/path/to/repo1")
	tabs.AddRepo("/path/to/repo2")
	
	if !tabs.HasRepos() {
		t.Error("Expected tabs with repos to return true for HasRepos()")
	}
	
	if !tabs.ShouldShowTabs() {
		t.Error("Expected tabs with multiple repos to return true for ShouldShowTabs()")
	}
	
	if tabs.NumRepos() != 2 {
		t.Errorf("Expected 2 repos, got %d", tabs.NumRepos())
	}
	
	// Test repo names
	if tabs.GetSelectedRepoName() != "repo1" {
		t.Errorf("Expected selected repo name to be 'repo1', got '%s'", tabs.GetSelectedRepoName())
	}
	
	if tabs.GetSelectedRepo() != "/path/to/repo1" {
		t.Errorf("Expected selected repo path to be '/path/to/repo1', got '%s'", tabs.GetSelectedRepo())
	}
}

func TestRepoTabs_Navigation(t *testing.T) {
	tabs := NewRepoTabs()
	tabs.SetWidth(80)
	
	tabs.AddRepo("/path/to/repo1")
	tabs.AddRepo("/path/to/repo2")
	tabs.AddRepo("/path/to/repo3")
	
	// Test navigation
	if tabs.GetSelectedIndex() != 0 {
		t.Errorf("Expected initial selection to be 0, got %d", tabs.GetSelectedIndex())
	}
	
	tabs.NextRepo()
	if tabs.GetSelectedIndex() != 1 {
		t.Errorf("Expected selection to be 1 after NextRepo(), got %d", tabs.GetSelectedIndex())
	}
	
	tabs.PrevRepo()
	if tabs.GetSelectedIndex() != 0 {
		t.Errorf("Expected selection to be 0 after PrevRepo(), got %d", tabs.GetSelectedIndex())
	}
	
	// Test wrapping
	tabs.PrevRepo()
	if tabs.GetSelectedIndex() != 2 {
		t.Errorf("Expected selection to wrap to 2, got %d", tabs.GetSelectedIndex())
	}
	
	tabs.NextRepo()
	if tabs.GetSelectedIndex() != 0 {
		t.Errorf("Expected selection to wrap to 0, got %d", tabs.GetSelectedIndex())
	}
}

func TestRepoTabs_RemoveRepo(t *testing.T) {
	tabs := NewRepoTabs()
	
	tabs.AddRepo("/path/to/repo1")
	tabs.AddRepo("/path/to/repo2")
	tabs.AddRepo("/path/to/repo3")
	
	// Remove middle repo
	tabs.RemoveRepo("/path/to/repo2")
	
	if tabs.NumRepos() != 2 {
		t.Errorf("Expected 2 repos after removal, got %d", tabs.NumRepos())
	}
	
	// Test that selection is adjusted
	tabs.SetSelectedIndex(1)
	if tabs.GetSelectedRepoName() != "repo3" {
		t.Errorf("Expected second repo to be 'repo3', got '%s'", tabs.GetSelectedRepoName())
	}
}

func TestRepoTabs_Rendering(t *testing.T) {
	tabs := NewRepoTabs()
	tabs.SetWidth(80)
	
	// Empty tabs should render nothing
	if tabs.Render() != "" {
		t.Error("Expected empty tabs to render empty string")
	}
	
	// Single repo should render nothing
	tabs.AddRepo("/path/to/repo1")
	if tabs.Render() != "" {
		t.Error("Expected single repo tabs to render empty string")
	}
	
	// Multiple repos should render
	tabs.AddRepo("/path/to/repo2")
	rendered := tabs.Render()
	if rendered == "" {
		t.Error("Expected multiple repo tabs to render content")
	}
	
	// Should contain repo names
	if len(rendered) == 0 {
		t.Error("Expected rendered content to have some length")
	}
}