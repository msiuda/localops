package project

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/mod/modfile"

	"github.com/msiuda/localops/internal/technology"
)

// Inspection is the basic metadata LocalOps can determine about a project
// directory by reading its filesystem contents.
type Inspection struct {
	Name                       string
	Path                       string
	IsGitRepository            bool
	HasGoMod                   bool
	GoModulePath               string
	IsNodeProject              bool
	NodePackageName            string
	NodePackageManager         string
	NodeEngineNode             string
	NodeEngineNpm              string
	NodeDeclaredPackageManager string
	NodeScriptLint             bool
	NodeScriptTypecheck        bool
	NodeScriptTest             bool
	NodeScriptBuild            bool
	// Technologies is the structured Technology Intelligence detection
	// result for this project: every language, runtime, platform, and
	// framework LocalOps found evidence for, with that evidence. It is
	// additive to the fields above, which Doctor's existing per-ecosystem
	// checks continue to key off directly; nothing here replaces them.
	Technologies technology.Result
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

	isNodeProject, pkg, err := readNodePackageJSON(proj.Path)
	if err != nil {
		return Inspection{}, fmt.Errorf("inspect %q: %w", proj.Path, err)
	}

	var nodePackageManager string
	if isNodeProject {
		nodePackageManager, err = detectNodePackageManager(proj.Path)
		if err != nil {
			return Inspection{}, fmt.Errorf("inspect %q: %w", proj.Path, err)
		}
	}

	_, hasLintScript := pkg.Scripts["lint"]
	_, hasTypecheckScript := pkg.Scripts["typecheck"]
	_, hasTestScript := pkg.Scripts["test"]
	_, hasBuildScript := pkg.Scripts["build"]

	technologies, err := technology.Detect(proj.Path)
	if err != nil {
		return Inspection{}, fmt.Errorf("inspect %q: %w", proj.Path, err)
	}

	return Inspection{
		Name:                       proj.Name,
		Path:                       proj.Path,
		IsGitRepository:            isGitRepo,
		HasGoMod:                   hasGoMod,
		GoModulePath:               goModulePath,
		IsNodeProject:              isNodeProject,
		NodePackageName:            pkg.Name,
		NodePackageManager:         nodePackageManager,
		NodeEngineNode:             pkg.Engines.Node,
		NodeEngineNpm:              pkg.Engines.Npm,
		NodeDeclaredPackageManager: pkg.PackageManager,
		NodeScriptLint:             hasLintScript,
		NodeScriptTypecheck:        hasTypecheckScript,
		NodeScriptTest:             hasTestScript,
		NodeScriptBuild:            hasBuildScript,
		Technologies:               technologies,
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

// nodePackageJSON holds the package.json fields LocalOps reads during
// inspection.
type nodePackageJSON struct {
	Name    string `json:"name"`
	Engines struct {
		Node string `json:"node"`
		Npm  string `json:"npm"`
	} `json:"engines"`
	PackageManager string            `json:"packageManager"`
	Scripts        map[string]string `json:"scripts"`
}

// readNodePackageJSON reports whether dir contains a package.json file and,
// if so, the metadata declared by its top-level "name", "engines",
// "packageManager", and "scripts" fields.
//
// A package.json entry that is not a regular file, or that cannot be parsed
// as JSON, is treated as an error rather than silently ignored.
func readNodePackageJSON(dir string) (bool, nodePackageJSON, error) {
	path := filepath.Join(dir, "package.json")

	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nodePackageJSON{}, nil
	}
	if err != nil {
		return false, nodePackageJSON{}, fmt.Errorf("check package.json: %w", err)
	}
	if !info.Mode().IsRegular() {
		return false, nodePackageJSON{}, fmt.Errorf("package.json at %q is not a regular file", path)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return false, nodePackageJSON{}, fmt.Errorf("read package.json: %w", err)
	}

	var pkg nodePackageJSON
	if err := json.Unmarshal(data, &pkg); err != nil {
		return false, nodePackageJSON{}, fmt.Errorf("parse package.json: %w", err)
	}

	return true, pkg, nil
}

// detectNodePackageManager detects the Node.js package manager used by dir
// based on which supported lockfile is present.
//
// If more than one supported lockfile is present, the result is ambiguous
// and an error is returned instead of silently picking one.
func detectNodePackageManager(dir string) (string, error) {
	// lockfiles maps the lockfile that identifies each supported Node.js
	// package manager.
	lockfiles := []struct {
		file    string
		manager string
	}{
		{file: "pnpm-lock.yaml", manager: "pnpm"},
		{file: "yarn.lock", manager: "yarn"},
		{file: "package-lock.json", manager: "npm"},
	}

	var foundFiles []string
	var manager string

	for _, lf := range lockfiles {
		_, err := os.Stat(filepath.Join(dir, lf.file))
		if err == nil {
			foundFiles = append(foundFiles, lf.file)
			manager = lf.manager
			continue
		}
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("check %s: %w", lf.file, err)
		}
	}

	if len(foundFiles) > 1 {
		return "", fmt.Errorf("multiple package manager lockfiles found: %s", strings.Join(foundFiles, ", "))
	}

	return manager, nil
}
