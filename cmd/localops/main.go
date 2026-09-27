// Command localops is the LocalOps CLI entry point.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/msiuda/localops/internal/project"
	"github.com/msiuda/localops/internal/storage"
)

func main() {
	if err := run(os.Args[1:], defaultStore, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// storeFunc resolves the project store, only when a command actually needs
// persistence. It lets commands that don't touch storage, such as
// "project inspect", avoid resolving it altogether.
type storeFunc func() (*storage.Store, error)

func run(args []string, getStore storeFunc, out io.Writer) error {
	if len(args) < 1 {
		return usageError()
	}

	switch args[0] {
	case "project":
		return runProject(args[1:], getStore, out)
	default:
		return usageError()
	}
}

func runProject(args []string, getStore storeFunc, out io.Writer) error {
	if len(args) < 1 {
		return usageError()
	}

	switch args[0] {
	case "add":
		return runProjectAdd(args[1:], getStore, out)
	case "list":
		return runProjectList(args[1:], getStore, out)
	case "inspect":
		return runProjectInspect(args[1:], out)
	default:
		return usageError()
	}
}

func runProjectAdd(args []string, getStore storeFunc, out io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: localops project add <path>")
	}

	proj, err := project.FromPath(args[0])
	if err != nil {
		return fmt.Errorf("add project: %w", err)
	}

	store, err := getStore()
	if err != nil {
		return fmt.Errorf("add project: %w", err)
	}

	projects, err := store.Load()
	if err != nil {
		return fmt.Errorf("add project: %w", err)
	}

	projects = append(projects, proj)

	if err := store.Save(projects); err != nil {
		return fmt.Errorf("add project: %w", err)
	}

	fmt.Fprintf(out, "Added project %q at %s\n", proj.Name, proj.Path)
	return nil
}

func runProjectList(args []string, getStore storeFunc, out io.Writer) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: localops project list")
	}

	store, err := getStore()
	if err != nil {
		return fmt.Errorf("list projects: %w", err)
	}

	projects, err := store.Load()
	if err != nil {
		return fmt.Errorf("list projects: %w", err)
	}

	if len(projects) == 0 {
		fmt.Fprintln(out, "No projects registered.")
		return nil
	}

	for _, p := range projects {
		fmt.Fprintf(out, "%s\t%s\n", p.Name, p.Path)
	}

	return nil
}

func runProjectInspect(args []string, out io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: localops project inspect <path>")
	}

	insp, err := project.Inspect(args[0])
	if err != nil {
		return fmt.Errorf("inspect project: %w", err)
	}

	fmt.Fprintf(out, "Name: %s\n", insp.Name)
	fmt.Fprintf(out, "Path: %s\n", insp.Path)
	fmt.Fprintf(out, "Git repository: %t\n", insp.IsGitRepository)
	fmt.Fprintf(out, "Go module: %t\n", insp.HasGoMod)
	if insp.HasGoMod {
		fmt.Fprintf(out, "Go module path: %s\n", insp.GoModulePath)
	}

	return nil
}

func defaultStore() (*storage.Store, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("locate user config directory: %w", err)
	}

	path := filepath.Join(configDir, "localops", "projects.json")
	return storage.New(path), nil
}

func usageError() error {
	return fmt.Errorf("usage: localops project <add|list|inspect> ...")
}
