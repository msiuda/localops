// Package storage persists LocalOps state to a local file.
package storage

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/msiuda/localops/internal/project"
)

// Store persists projects as JSON at an explicit file path.
type Store struct {
	path string
}

// New returns a Store that reads and writes the file at path.
func New(path string) *Store {
	return &Store{path: path}
}

// DefaultPath returns the default LocalOps storage file location, in an
// appropriate user-specific application configuration directory. Every
// LocalOps entry point (the CLI and the desktop application) uses this same
// path, so they share one registered-project registry.
func DefaultPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locate user config directory: %w", err)
	}

	return filepath.Join(configDir, "localops", "projects.json"), nil
}

// fileState is the on-disk representation of the storage file.
type fileState struct {
	Projects []project.Project `json:"projects"`
}

// Load returns the projects currently in storage.
//
// A missing storage file is treated as an empty project list. Invalid or
// corrupted storage returns an explicit error.
func (s *Store) Load() ([]project.Project, error) {
	data, err := os.ReadFile(s.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read storage %q: %w", s.path, err)
	}

	var state fileState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("parse storage %q: %w", s.path, err)
	}

	return state.Projects, nil
}

// Save writes projects to storage, replacing any existing content.
//
// The write goes to a temporary file that is then renamed into place, which
// reduces (but does not eliminate) the risk of a failure mid-write leaving
// existing storage corrupted.
func (s *Store) Save(projects []project.Project) error {
	data, err := json.MarshalIndent(fileState{Projects: projects}, "", "  ")
	if err != nil {
		return fmt.Errorf("encode projects: %w", err)
	}

	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("create storage directory %q: %w", dir, err)
	}

	tmp, err := os.CreateTemp(dir, ".projects-*.json")
	if err != nil {
		return fmt.Errorf("create temp storage file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp storage file: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp storage file: %w", err)
	}

	if err := os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("replace storage file %q: %w", s.path, err)
	}

	return nil
}
