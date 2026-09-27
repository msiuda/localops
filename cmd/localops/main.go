// Command localops is the LocalOps CLI entry point.
package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/msiuda/localops/internal/doctor"
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
	case "help", "--help", "-h":
		fmt.Fprint(out, topLevelHelp)
		return nil
	case "project":
		return runProject(args[1:], getStore, out)
	case "doctor":
		return runDoctor(args[1:], out)
	default:
		return usageError()
	}
}

func runProject(args []string, getStore storeFunc, out io.Writer) error {
	if len(args) < 1 {
		return usageError()
	}

	switch args[0] {
	case "help", "--help", "-h":
		fmt.Fprint(out, projectHelp)
		return nil
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
	fmt.Fprintf(out, "Node.js project: %t\n", insp.IsNodeProject)
	if insp.NodePackageName != "" {
		fmt.Fprintf(out, "Node.js package name: %s\n", insp.NodePackageName)
	}
	if insp.NodePackageManager != "" {
		fmt.Fprintf(out, "Node.js package manager: %s\n", insp.NodePackageManager)
	}
	if insp.NodeEngineNode != "" {
		fmt.Fprintf(out, "Node.js version requirement: %s\n", insp.NodeEngineNode)
	}
	if insp.NodeEngineNpm != "" {
		fmt.Fprintf(out, "npm version requirement: %s\n", insp.NodeEngineNpm)
	}
	if insp.NodeDeclaredPackageManager != "" {
		fmt.Fprintf(out, "Declared package manager: %s\n", insp.NodeDeclaredPackageManager)
	}

	return nil
}

func runDoctor(args []string, out io.Writer) error {
	if len(args) == 1 && isHelpFlag(args[0]) {
		fmt.Fprint(out, doctorHelp)
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: localops doctor <path>")
	}

	insp, err := project.Inspect(args[0])
	if err != nil {
		return fmt.Errorf("doctor: %w", err)
	}

	report := doctor.Run(insp)

	fmt.Fprintf(out, "Doctor: %s\n\n", report.Path)

	if len(report.Checks) == 0 {
		fmt.Fprintln(out, "No required executables detected.")
		return nil
	}

	issues := 0
	for _, check := range report.Checks {
		if check.Available {
			fmt.Fprintf(out, "[OK] %s\n", check.Tool)
			continue
		}
		issues++
		fmt.Fprintf(out, "[FAIL] %s — %s\n", check.Tool, check.Detail)
	}

	fmt.Fprintln(out)
	switch issues {
	case 0:
		fmt.Fprintln(out, "No issues found")
	case 1:
		fmt.Fprintln(out, "1 issue found")
	default:
		fmt.Fprintf(out, "%d issues found\n", issues)
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

const topLevelHelp = `Usage:
  localops <command>

Commands:
  project   Manage and inspect local projects
  doctor    Diagnose a local project

Run 'localops project help' or 'localops doctor --help' for details.
`

const projectHelp = `Usage:
  localops project add <path>       Register a local project
  localops project list             List registered projects
  localops project inspect <path>   Inspect a project's filesystem metadata
`

const doctorHelp = `Usage:
  localops doctor <path>

Checks whether the local machine has the executables required by the
technologies detected in the project at <path> (Git, Go modules, Node.js
and its package manager).
`

// isHelpFlag reports whether arg requests help rather than naming a
// subcommand or path.
func isHelpFlag(arg string) bool {
	return arg == "help" || arg == "--help" || arg == "-h"
}

func usageError() error {
	return fmt.Errorf("usage: localops <project|doctor> ...")
}
