// Command localops is the LocalOps CLI entry point.
package main

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/msiuda/localops/internal/doctor"
	"github.com/msiuda/localops/internal/environment"
	"github.com/msiuda/localops/internal/overview"
	"github.com/msiuda/localops/internal/project"
	"github.com/msiuda/localops/internal/storage"
	"github.com/msiuda/localops/internal/validation"
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
	case "overview":
		return runOverview(args[1:], getStore, out)
	case "validate":
		return runValidate(args[1:], out)
	case "environment":
		return runEnvironment(args[1:], out)
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
		if check.OK {
			if check.Note != "" {
				fmt.Fprintf(out, "[OK] %s — %s (%s)\n", check.Tool, check.Version, check.Note)
			} else {
				fmt.Fprintf(out, "[OK] %s — %s\n", check.Tool, check.Version)
			}
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

func runOverview(args []string, getStore storeFunc, out io.Writer) error {
	if len(args) == 1 && isHelpFlag(args[0]) {
		fmt.Fprint(out, overviewHelp)
		return nil
	}
	if len(args) != 0 {
		return fmt.Errorf("usage: localops overview")
	}

	store, err := getStore()
	if err != nil {
		return fmt.Errorf("overview: %w", err)
	}

	projects, err := store.Load()
	if err != nil {
		return fmt.Errorf("overview: %w", err)
	}

	if len(projects) == 0 {
		fmt.Fprintln(out, "No projects registered.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Register one with:")
		fmt.Fprintln(out, "  localops project add <path>")
		return nil
	}

	result := overview.Build(projects)

	fmt.Fprintln(out, "LocalOps Overview")
	fmt.Fprintln(out)

	var healthy, withIssues, unavailable int
	for _, pr := range result.Projects {
		printOverviewProject(out, pr)
		fmt.Fprintln(out)

		switch pr.Health {
		case overview.HealthHealthy:
			healthy++
		case overview.HealthIssues:
			withIssues++
		case overview.HealthUnavailable:
			unavailable++
		}
	}

	fmt.Fprintf(out, "%d projects: %d healthy, %d with issues, %d unavailable\n",
		len(result.Projects), healthy, withIssues, unavailable)

	return nil
}

// printOverviewProject prints a single project's Overview result in the
// format described by "localops overview help".
func printOverviewProject(out io.Writer, pr overview.ProjectResult) {
	switch pr.Health {
	case overview.HealthHealthy:
		fmt.Fprintf(out, "[OK] %s\n", pr.Project.Name)
	case overview.HealthIssues:
		fmt.Fprintf(out, "[ISSUES] %s\n", pr.Project.Name)
	default:
		fmt.Fprintf(out, "[UNAVAILABLE] %s\n", pr.Project.Name)
	}
	fmt.Fprintf(out, "  %s\n", pr.Project.Path)

	if pr.Health == overview.HealthUnavailable {
		fmt.Fprintf(out, "  %s\n", pr.Err)
		return
	}

	fmt.Fprintf(out, "  %s\n", technologySummary(pr.Inspection, pr.Report))

	var failed []doctor.CheckResult
	for _, check := range pr.Report.Checks {
		if !check.OK {
			failed = append(failed, check)
		}
	}

	switch len(failed) {
	case 0:
		fmt.Fprintln(out, "  No issues")
	case 1:
		fmt.Fprintln(out, "  1 issue")
	default:
		fmt.Fprintf(out, "  %d issues\n", len(failed))
	}
	for _, check := range failed {
		fmt.Fprintf(out, "  - %s — %s\n", check.Tool, check.Detail)
	}
}

// technologySummary builds a short "Git · Go · Node.js · yarn"-style
// summary of the technologies insp detected. For the package manager, it
// reflects the manager Doctor actually evaluated (report), which may differ
// from the lockfile-detected one when package.json declares one instead.
func technologySummary(insp project.Inspection, report doctor.Report) string {
	var parts []string

	if insp.IsGitRepository {
		parts = append(parts, "Git")
	}
	if insp.HasGoMod {
		parts = append(parts, "Go")
	}
	if insp.IsNodeProject {
		parts = append(parts, "Node.js")
		if manager := effectivePackageManager(report); manager != "" {
			parts = append(parts, manager)
		}
	}

	if len(parts) == 0 {
		return "No detected technologies"
	}

	return strings.Join(parts, " · ")
}

// effectivePackageManager returns the package manager Doctor actually
// checked, derived from report's checks rather than duplicating Doctor's
// own declared/lockfile reconciliation logic.
func effectivePackageManager(report doctor.Report) string {
	for _, check := range report.Checks {
		switch check.Tool {
		case "npm", "yarn", "pnpm":
			return check.Tool
		}
	}
	return ""
}

func runValidate(args []string, out io.Writer) error {
	if len(args) == 1 && isHelpFlag(args[0]) {
		fmt.Fprint(out, validateHelp)
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: localops validate <path>")
	}

	result, err := validation.Validate(args[0])
	if err != nil {
		return fmt.Errorf("validate: %w", err)
	}

	renderValidation(out, result)
	return nil
}

// renderValidation prints result in the format described by
// "localops validate --help". It is a pure presentation step, kept
// separate from running Validation itself.
func renderValidation(out io.Writer, result validation.Result) {
	fmt.Fprintln(out, "LocalOps Validation")
	fmt.Fprintf(out, "Project: %s\n\n", result.Path)

	fmt.Fprintln(out, "Environment")
	if len(result.Doctor.Checks) == 0 {
		fmt.Fprintln(out, "No required executables detected.")
	}
	for _, check := range result.Doctor.Checks {
		if check.OK {
			if check.Note != "" {
				fmt.Fprintf(out, "[OK] %s — %s (%s)\n", check.Tool, check.Version, check.Note)
			} else {
				fmt.Fprintf(out, "[OK] %s — %s\n", check.Tool, check.Version)
			}
			continue
		}
		fmt.Fprintf(out, "[FAIL] %s — %s\n", check.Tool, check.Detail)
	}

	fmt.Fprintln(out)
	fmt.Fprintln(out, "Checks")

	if len(result.Checks) == 0 {
		fmt.Fprintln(out, "No validation checks detected.")
		return
	}

	var passed, failed, blocked int
	for _, check := range result.Checks {
		switch check.Status {
		case validation.StatusPassed:
			passed++
			fmt.Fprintf(out, "[OK] %s — %s\n", check.Name, check.Duration.Round(100*time.Millisecond))
		case validation.StatusFailed:
			failed++
			fmt.Fprintf(out, "[FAIL] %s — %s\n", check.Name, check.Duration.Round(100*time.Millisecond))
			if check.Output != "" {
				for _, line := range strings.Split(check.Output, "\n") {
					fmt.Fprintf(out, "  %s\n", line)
				}
			}
		case validation.StatusBlocked:
			blocked++
			fmt.Fprintf(out, "[BLOCKED] %s — %s\n", check.Name, check.Detail)
		}
	}

	fmt.Fprintln(out)
	fmt.Fprintf(out, "%d passed, %d failed, %d blocked\n", passed, failed, blocked)
}

func runEnvironment(args []string, out io.Writer) error {
	if len(args) == 1 && isHelpFlag(args[0]) {
		fmt.Fprint(out, environmentHelp)
		return nil
	}
	if len(args) != 1 {
		return fmt.Errorf("usage: localops environment <path>")
	}

	result, err := environment.Analyze(args[0])
	if err != nil {
		return fmt.Errorf("environment: %w", err)
	}

	renderEnvironment(out, result)
	return nil
}

// renderEnvironment prints result in the format described by
// "localops environment --help". It is a pure presentation step, kept
// separate from running the analysis itself. It never prints an
// environment variable's value, since Result never contains one.
func renderEnvironment(out io.Writer, result environment.Result) {
	fmt.Fprintln(out, "LocalOps Environment")
	fmt.Fprintf(out, "Project: %s\n\n", result.Path)

	if len(result.ContractSources) == 0 {
		fmt.Fprintln(out, "No environment contract detected.")
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Supported contract files:")
		fmt.Fprintln(out, "  .env.example")
		fmt.Fprintln(out, "  .env.sample")
		fmt.Fprintln(out, "  .env.template")
		return
	}

	fmt.Fprintln(out, "Contract sources")
	for _, s := range result.ContractSources {
		fmt.Fprintf(out, "  %s — %d %s\n", s.File, s.VariableCount, pluralizeVariables(s.VariableCount))
	}
	fmt.Fprintln(out)

	processEnvUsed := false
	for _, v := range result.Variables {
		if v.Source == "process environment" {
			processEnvUsed = true
			break
		}
	}

	fmt.Fprintln(out, "Local sources")
	for _, s := range result.LocalSources {
		fmt.Fprintf(out, "  %s — %d %s\n", s.File, s.VariableCount, pluralizeVariables(s.VariableCount))
	}
	if processEnvUsed {
		fmt.Fprintln(out, "  process environment")
	}
	fmt.Fprintln(out)

	fmt.Fprintln(out, "Variables")
	var satisfied, missing int
	for _, v := range result.Variables {
		if v.Satisfied {
			satisfied++
			fmt.Fprintf(out, "[OK] %s — %s\n", v.Name, v.Source)
			continue
		}
		missing++
		fmt.Fprintf(out, "[MISSING] %s\n", v.Name)
	}

	fmt.Fprintln(out)
	fmt.Fprintf(out, "%d satisfied, %d missing\n", satisfied, missing)

	if len(result.Findings) > 0 {
		fmt.Fprintln(out)
		fmt.Fprintln(out, "Findings")
		for _, f := range result.Findings {
			fmt.Fprintf(out, "- %s: line %d %s\n", f.Source, f.Line, f.Detail)
		}
	}
}

// pluralizeVariables returns "variable" or "variables" depending on n.
func pluralizeVariables(n int) string {
	if n == 1 {
		return "variable"
	}
	return "variables"
}

func defaultStore() (*storage.Store, error) {
	path, err := storage.DefaultPath()
	if err != nil {
		return nil, err
	}

	return storage.New(path), nil
}

const topLevelHelp = `Usage:
  localops <command>

Commands:
  project       Manage and inspect local projects
  doctor        Diagnose a local project
  overview      Show a health summary of all registered projects
  validate      Actively run a project's validation checks
  environment   Analyze a project's Environment Contract

Run 'localops project help', 'localops doctor --help',
'localops overview --help', 'localops validate --help', or
'localops environment --help' for details.
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
and its package manager), reports their versions, and for Node.js projects
flags Node/npm versions or a package manager that don't match what the
project declares (engines.node, engines.npm, and packageManager).
`

const overviewHelp = `Usage:
  localops overview

Loads every registered project, inspects it, runs Doctor against it, and
prints a concise summary of its health: healthy, has issues, or
unavailable (for example, if its registered path no longer exists).
`

const validateHelp = `Usage:
  localops validate <path>

Actively runs the project's validation checks:
  - for a Go module: 'go test ./...', 'go vet ./...', 'go build ./...',
  - for a Node.js project: its recognized package.json scripts (lint,
    typecheck, test, build), run through the project's effective package
    manager.

Unlike inspection and most of Doctor, this executes real project commands.
It never installs dependencies, modifies package manifests, or runs any
other script. A required tool that is unavailable or incompatible, or
missing Node.js dependencies, blocks only the checks that depend on it;
independent checks still run.
`

const environmentHelp = `Usage:
  localops environment <path>

Analyzes the project's Environment Contract: which environment variables
it declares it expects (via .env.example, .env.sample, or .env.template),
and which are currently satisfied by a local env file (.env.local, .env)
or the process environment.

This is read-only. It never creates, copies, or modifies any env file,
never injects variables into the process, and never prints, stores, or
otherwise exposes a variable's value — only variable names, source
filenames, and whether each is present are shown.
`

// isHelpFlag reports whether arg requests help rather than naming a
// subcommand or path.
func isHelpFlag(arg string) bool {
	return arg == "help" || arg == "--help" || arg == "-h"
}

func usageError() error {
	return fmt.Errorf("usage: localops <project|doctor|overview|validate|environment> ...")
}
