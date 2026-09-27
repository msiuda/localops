// Package validation actively runs a project's validation commands (Go's
// test/vet/build, and a Node.js project's recognized package.json scripts)
// after checking, via project inspection and Doctor, that the required
// tools are actually usable on the local machine.
//
// Unlike project inspection and most of Doctor, Validate executes real
// project commands. Only the specific commands documented here are ever
// run: no dependency installation, package manifest changes, deploys, dev
// servers, migrations, seed commands, or arbitrary scripts.
package validation

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/msiuda/localops/internal/doctor"
	"github.com/msiuda/localops/internal/project"
)

// Status is the outcome of a single validation check.
type Status string

const (
	// StatusPassed means the command ran and exited successfully.
	StatusPassed Status = "passed"
	// StatusFailed means the command ran but exited with an error.
	StatusFailed Status = "failed"
	// StatusBlocked means the command was not attempted because a
	// required tool is unavailable or incompatible, or because its
	// dependencies are not installed.
	StatusBlocked Status = "blocked"
)

// CheckResult is the outcome of a single validation check.
type CheckResult struct {
	// Name identifies the check, e.g. "lint" or "go test".
	Name string
	// Command is a human-readable description of what was (or would have
	// been) executed, e.g. "pnpm run lint" or "go test ./...".
	Command string
	Status  Status
	// Duration is zero for a blocked check, since nothing was executed.
	Duration time.Duration
	// Output is the command's captured, truncated combined stdout/stderr.
	// It is only populated for a failed check.
	Output string
	// Detail explains why a check is blocked. It is empty otherwise.
	Detail string
}

// Result is the outcome of validating a single project.
type Result struct {
	Path       string
	Inspection project.Inspection
	Doctor     doctor.Report
	Checks     []CheckResult
}

// maxOutputLines caps how many lines of a failed command's captured output
// are kept, so a runaway command cannot flood the terminal. It is a simple,
// deterministic limit rather than a configurable logging system.
const maxOutputLines = 40

// inspectFunc mirrors project.Inspect's signature, allowing tests to
// substitute inspection instead of depending on real filesystem contents.
type inspectFunc func(path string) (project.Inspection, error)

// runDoctorFunc mirrors doctor.Run's signature, allowing tests to
// substitute Doctor instead of depending on real installed tools.
type runDoctorFunc func(insp project.Inspection) doctor.Report

// runCommandFunc runs name with args in dir and returns its combined,
// untruncated stdout/stderr. Tests substitute this instead of executing
// real project commands.
type runCommandFunc func(dir, name string, args ...string) (string, error)

// dirExistsFunc reports whether path exists and is a directory. Tests
// substitute this instead of depending on the real filesystem.
type dirExistsFunc func(path string) (bool, error)

// Validate inspects the project at path, runs Doctor against it, and then
// actively runs its validation checks (Go's test/vet/build and/or a
// Node.js project's recognized package.json scripts).
//
// An error is returned only when the project itself cannot be inspected.
// A check that fails, or that is blocked because a required tool is
// unavailable or incompatible, is represented in the returned Result rather
// than as an error.
func Validate(path string) (Result, error) {
	return validate(path, project.Inspect, doctor.Run, runCommand, dirExists)
}

// validate implements Validate, taking inspection, Doctor, command
// execution, and directory existence as small function seams so tests can
// substitute them.
func validate(
	path string,
	inspect inspectFunc,
	runDoctor runDoctorFunc,
	runCmd runCommandFunc,
	dirExists dirExistsFunc,
) (Result, error) {
	insp, err := inspect(path)
	if err != nil {
		return Result{}, err
	}

	report := runDoctor(insp)

	var checks []CheckResult
	if insp.HasGoMod {
		checks = append(checks, goChecks(insp, report, runCmd)...)
	}
	if insp.IsNodeProject {
		checks = append(checks, nodeChecks(insp, report, runCmd, dirExists)...)
	}

	return Result{
		Path:       insp.Path,
		Inspection: insp,
		Doctor:     report,
		Checks:     checks,
	}, nil
}

// goChecks runs go test, go vet, and go build for a Go module, each as an
// independent check. If Doctor found the go executable unavailable or
// incompatible, all three are reported as blocked instead of attempted.
func goChecks(insp project.Inspection, report doctor.Report, runCmd runCommandFunc) []CheckResult {
	if blocked, detail := doctorBlocks(report, "go"); blocked {
		return []CheckResult{
			blockedCheck("go test", "go test ./...", detail),
			blockedCheck("go vet", "go vet ./...", detail),
			blockedCheck("go build", "go build ./...", detail),
		}
	}

	return []CheckResult{
		runCheck("go test", "go", []string{"test", "./..."}, insp.Path, runCmd),
		runCheck("go vet", "go", []string{"vet", "./..."}, insp.Path, runCmd),
		runCheck("go build", "go", []string{"build", "./..."}, insp.Path, runCmd),
	}
}

// nodeChecks runs each recognized package.json script (lint, typecheck,
// test, build) through the project's effective package manager, each as an
// independent check. If the Node runtime, the effective package manager,
// or its dependencies are not usable, every recognized script is reported
// as blocked instead of attempted.
func nodeChecks(insp project.Inspection, report doctor.Report, runCmd runCommandFunc, dirExists dirExistsFunc) []CheckResult {
	scripts := recognizedNodeScripts(insp)
	if len(scripts) == 0 {
		return nil
	}

	manager := effectivePackageManager(report)

	if blocked, detail := nodeBlocked(insp, report, manager, dirExists); blocked {
		checks := make([]CheckResult, 0, len(scripts))
		for _, script := range scripts {
			checks = append(checks, blockedCheck(script, nodeCommandDisplay(manager, script), detail))
		}
		return checks
	}

	checks := make([]CheckResult, 0, len(scripts))
	for _, script := range scripts {
		checks = append(checks, runCheck(script, manager, runScriptArgs(script), insp.Path, runCmd))
	}
	return checks
}

// nodeBlocked reports whether Node validation should be blocked, and why:
// an unusable Node runtime or effective package manager (including a
// package-manager declaration/lockfile mismatch), or dependencies that do
// not appear to be installed.
func nodeBlocked(insp project.Inspection, report doctor.Report, manager string, dirExists dirExistsFunc) (bool, string) {
	if blocked, detail := doctorBlocks(report, "node"); blocked {
		return true, detail
	}
	if manager == "" {
		return true, "no usable package manager detected"
	}
	if blocked, detail := doctorBlocks(report, manager); blocked {
		return true, detail
	}
	if blocked, detail := doctorBlocks(report, "package manager"); blocked {
		return true, detail
	}

	installed, err := dirExists(filepath.Join(insp.Path, "node_modules"))
	if err != nil {
		return true, fmt.Sprintf("could not check node_modules: %v", err)
	}
	if !installed {
		return true, "dependencies are not installed"
	}

	return false, ""
}

// recognizedNodeScripts returns the validation-oriented package.json
// scripts insp declares, in a fixed, deterministic order.
func recognizedNodeScripts(insp project.Inspection) []string {
	var scripts []string
	if insp.NodeScriptLint {
		scripts = append(scripts, "lint")
	}
	if insp.NodeScriptTypecheck {
		scripts = append(scripts, "typecheck")
	}
	if insp.NodeScriptTest {
		scripts = append(scripts, "test")
	}
	if insp.NodeScriptBuild {
		scripts = append(scripts, "build")
	}
	return scripts
}

// runScriptArgs returns the arguments used to run a package.json script
// through a package manager's "run" subcommand. npm, yarn, and pnpm all
// support the same "run <script>" form.
func runScriptArgs(script string) []string {
	return []string{"run", script}
}

// nodeCommandDisplay describes the command that would run script through
// manager, for display on a blocked check. It is empty when manager is
// unknown.
func nodeCommandDisplay(manager, script string) string {
	if manager == "" {
		return ""
	}
	return strings.Join(append([]string{manager}, runScriptArgs(script)...), " ")
}

// effectivePackageManager returns the package manager Doctor actually
// evaluated, derived from report's checks rather than duplicating Doctor's
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

// doctorBlocks reports whether tool's Doctor check, if any, did not pass,
// and a short explanation to use as the blocked reason.
func doctorBlocks(report doctor.Report, tool string) (bool, string) {
	for _, check := range report.Checks {
		if check.Tool == tool && !check.OK {
			if check.Detail != "" {
				return true, check.Detail
			}
			return true, fmt.Sprintf("%s is not usable", tool)
		}
	}
	return false, ""
}

// runCheck runs command with args in dir, timing the execution and
// capturing truncated output on failure.
func runCheck(name, command string, args []string, dir string, runCmd runCommandFunc) CheckResult {
	start := time.Now()
	out, err := runCmd(dir, command, args...)
	duration := time.Since(start)

	commandDisplay := strings.Join(append([]string{command}, args...), " ")

	if err != nil {
		return CheckResult{
			Name:     name,
			Command:  commandDisplay,
			Status:   StatusFailed,
			Duration: duration,
			Output:   truncateOutput(out),
		}
	}

	return CheckResult{
		Name:     name,
		Command:  commandDisplay,
		Status:   StatusPassed,
		Duration: duration,
	}
}

// blockedCheck builds a CheckResult for a check that was not attempted.
func blockedCheck(name, command, detail string) CheckResult {
	return CheckResult{
		Name:    name,
		Command: command,
		Status:  StatusBlocked,
		Detail:  detail,
	}
}

// truncateOutput keeps at most the first maxOutputLines lines of output,
// appending a marker when it had to cut something off.
func truncateOutput(output string) string {
	output = strings.TrimRight(output, "\n")
	if output == "" {
		return ""
	}

	lines := strings.Split(output, "\n")
	if len(lines) <= maxOutputLines {
		return output
	}

	return strings.Join(lines[:maxOutputLines], "\n") + "\n... (output truncated)"
}

// runCommand runs name with args in dir and returns its combined,
// untruncated stdout/stderr.
func runCommand(dir, name string, args ...string) (string, error) {
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

// dirExists reports whether path exists and is a directory.
func dirExists(path string) (bool, error) {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}
