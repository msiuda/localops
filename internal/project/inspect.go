package project

import (
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/mod/modfile"
)

// Inspection is the basic metadata LocalOps can determine about a project
// directory by reading its filesystem contents.
type Inspection struct {
	Name            string
	Path            string
	IsGitRepository bool
	HasGoMod        bool
	GoModulePath    string
}

// Inspect reads basic, read-only metadata about the project at path.
//
// The path does not need to be registered with LocalOps. Detection is based
// solely on filesystem contents; no external commands are executed.
func Inspect(path string) (Inspection, error) {
	proj, err := FromPath(path)
	if err != nil {
		return Inspection{}, err
	}

	isGitRepo, err := isGitRepository(proj.Path)
	if err != nil {
		return Inspection{}, fmt.Errorf("inspect %q: %w", proj.Path, err)
	}

	hasGoMod, err := hasGoModFile(proj.Path)
	if err != nil {
		return Inspection{}, fmt.Errorf("inspect %q: %w", proj.Path, err)
	}

	var goModulePath string
	if hasGoMod {
		goModulePath, err = readGoModulePath(filepath.Join(proj.Path, "go.mod"))
		if err != nil {
			return Inspection{}, fmt.Errorf("inspect %q: %w", proj.Path, err)
		}
	}

	return Inspection{
		Name:            proj.Name,
		Path:            proj.Path,
		IsGitRepository: isGitRepo,
		HasGoMod:        hasGoMod,
		GoModulePath:    goModulePath,
	}, nil
}

// isGitRepository reports whether dir contains a .git entry. Git repositories
// normally have a .git directory, while worktrees and submodules use a .git
// file, so either is accepted.
func isGitRepository(dir string) (bool, error) {
	_, err := os.Stat(filepath.Join(dir, ".git"))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("check git repository: %w", err)
}

// hasGoModFile reports whether dir contains a go.mod file.
func hasGoModFile(dir string) (bool, error) {
	_, err := os.Stat(filepath.Join(dir, "go.mod"))
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("check go.mod: %w", err)
}

// readGoModulePath reads the module path declared by the module directive
// in the go.mod file at path, using the canonical go.mod parser.
func readGoModulePath(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read go.mod: %w", err)
	}

	modFile, err := modfile.Parse(path, data, nil)
	if err != nil {
		return "", fmt.Errorf("parse go.mod: %w", err)
	}

	if modFile.Module == nil {
		return "", fmt.Errorf("parse go.mod: no module directive found")
	}

	return modFile.Module.Mod.Path, nil
}
