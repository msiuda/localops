// Package project represents local software projects known to LocalOps.
package project

import (
	"fmt"
	"os"
	"path/filepath"
)

// Project is a local software project registered with LocalOps.
type Project struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

// FromPath builds a Project from a filesystem path.
//
// The path must refer to an existing directory. The resulting Project has
// its Path resolved to a clean absolute path and its Name derived from the
// directory's base name.
func FromPath(path string) (Project, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return Project{}, fmt.Errorf("resolve path %q: %w", path, err)
	}

	info, err := os.Stat(abs)
	if err != nil {
		return Project{}, fmt.Errorf("access path %q: %w", abs, err)
	}
	if !info.IsDir() {
		return Project{}, fmt.Errorf("path %q is not a directory", abs)
	}

	return Project{
		Name: filepath.Base(abs),
		Path: abs,
	}, nil
}
