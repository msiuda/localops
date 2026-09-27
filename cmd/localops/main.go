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
	store, err := defaultStore()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if err := run(os.Args[1:], store, os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(args []string, store *storage.Store, out io.Writer) error {
	if len(args) < 1 {
		return usageError()
	}

	switch args[0] {
	case "project":
		return runProject(args[1:], store, out)
	default:
		return usageError()
	}
}

func runProject(args []string, store *storage.Store, out io.Writer) error {
	if len(args) < 1 {
		return usageError()
	}

	switch args[0] {
	case "add":
		return runProjectAdd(args[1:], store, out)
	case "list":
		return runProjectList(args[1:], store, out)
	default:
		return usageError()
	}
}

func runProjectAdd(args []string, store *storage.Store, out io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: localops project add <path>")
	}

	proj, err := project.FromPath(args[0])
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

func runProjectList(args []string, store *storage.Store, out io.Writer) error {
	if len(args) != 0 {
		return fmt.Errorf("usage: localops project list")
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

func defaultStore() (*storage.Store, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return nil, fmt.Errorf("locate user config directory: %w", err)
	}

	path := filepath.Join(configDir, "localops", "projects.json")
	return storage.New(path), nil
}

func usageError() error {
	return fmt.Errorf("usage: localops project <add|list> ...")
}
