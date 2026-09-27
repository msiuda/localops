package validation

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/msiuda/localops/internal/doctor"
	"github.com/msiuda/localops/internal/project"
)

type commandCall struct {
	dir  string
	name string
	args []string
}

type cmdResult struct {
	output string
	err    error
}

// fakeRunCommand returns a runCommandFunc that resolves canned output/error
// by "name arg1 arg2..." key, and records every call it receives into
// calls, so tests can assert on the working directory a command ran in.
//
// A command key that was not explicitly configured in results returns an
// error instead of silently succeeding, so tests must declare every
// command they expect Validation to execute and an unexpected command
// cannot accidentally look like a pass.
func fakeRunCommand(results map[string]cmdResult, calls *[]commandCall) runCommandFunc {
	return func(dir, name string, args ...string) (string, error) {
		key := strings.Join(append([]string{name}, args...), " ")
		if calls != nil {
			*calls = append(*calls, commandCall{dir: dir, name: name, args: append([]string(nil), args...)})
		}
		r, ok := results[key]
		if !ok {
			return "", fmt.Errorf("unexpected command: %s", key)
		}
		return r.output, r.err
	}
}

func fakeDirExists(exists bool, err error) dirExistsFunc {
	return func(path string) (bool, error) {
		return exists, err
	}
}

func fixedInspect(insp project.Inspection, err error) inspectFunc {
	return func(path string) (project.Inspection, error) {
		return insp, err
	}
}

func fixedRunDoctor(report doctor.Report) runDoctorFunc {
	return func(insp project.Inspection) doctor.Report {
		return report
	}
}

func checkFor(t *testing.T, checks []CheckResult, name string) CheckResult {
	t.Helper()
	for _, c := range checks {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("no check found for %q in %+v", name, checks)
	return CheckResult{}
}

func TestValidate_PlainProject_NoChecks(t *testing.T) {
	insp := project.Inspection{Path: "/projects/plain"}

	result, err := validate(
		"/projects/plain",
		fixedInspect(insp, nil),
		fixedRunDoctor(doctor.Report{Path: insp.Path}),
		fakeRunCommand(nil, nil),
		fakeDirExists(true, nil),
	)
	if err != nil {
		t.Fatalf("validate() error = %v", err)
	}

	if len(result.Checks) != 0 {
		t.Errorf("Checks = %v, want none", result.Checks)
	}
}

func TestValidate_NodeProject_RecognizedScripts(t *testing.T) {
	insp := project.Inspection{
		Path:               "/projects/node",
		IsNodeProject:      true,
		NodePackageManager: "pnpm",
		NodeScriptLint:     true,
		NodeScriptTest:     true,
	}
	report := doctor.Report{
		Checks: []doctor.CheckResult{
			{Tool: "node", OK: true, Version: "v22.14.0"},
			{Tool: "pnpm", OK: true, Version: "10.4.1"},
		},
	}

	result, err := validate(
		insp.Path,
		fixedInspect(insp, nil),
		fixedRunDoctor(report),
		fakeRunCommand(map[string]cmdResult{
			"pnpm run lint": {output: ""},
			"pnpm run test": {output: ""},
		}, nil),
		fakeDirExists(true, nil),
	)
	if err != nil {
		t.Fatalf("validate() error = %v", err)
	}

	if len(result.Checks) != 2 {
		t.Fatalf("Checks = %v, want exactly two checks", result.Checks)
	}
	lint := checkFor(t, result.Checks, "lint")
	if lint.Status != StatusPassed {
		t.Errorf("lint Status = %q, want %q", lint.Status, StatusPassed)
	}
	if lint.Command != "pnpm run lint" {
		t.Errorf("lint Command = %q, want %q", lint.Command, "pnpm run lint")
	}
}

func TestValidate_NodeProject_UnrelatedScriptsIgnored(t *testing.T) {
	insp := project.Inspection{
		Path:               "/projects/node",
		IsNodeProject:      true,
		NodePackageManager: "npm",
		// No NodeScript* fields set: package.json declares only
		// unrelated scripts such as "start" or "dev".
	}
	report := doctor.Report{
		Checks: []doctor.CheckResult{
			{Tool: "node", OK: true},
			{Tool: "npm", OK: true},
		},
	}

	result, err := validate(
		insp.Path,
		fixedInspect(insp, nil),
		fixedRunDoctor(report),
		fakeRunCommand(nil, nil),
		fakeDirExists(true, nil),
	)
	if err != nil {
		t.Fatalf("validate() error = %v", err)
	}

	if len(result.Checks) != 0 {
		t.Errorf("Checks = %v, want none for a project with no recognized scripts", result.Checks)
	}
}

func TestValidate_NodeProject_MissingNodeModulesBlocks(t *testing.T) {
	insp := project.Inspection{
		Path:               "/projects/node",
		IsNodeProject:      true,
		NodePackageManager: "pnpm",
		NodeScriptLint:     true,
	}
	report := doctor.Report{
		Checks: []doctor.CheckResult{
			{Tool: "node", OK: true},
			{Tool: "pnpm", OK: true},
		},
	}

	var calls []commandCall
	result, err := validate(
		insp.Path,
		fixedInspect(insp, nil),
		fixedRunDoctor(report),
		fakeRunCommand(nil, &calls),
		fakeDirExists(false, nil),
	)
	if err != nil {
		t.Fatalf("validate() error = %v", err)
	}

	got := checkFor(t, result.Checks, "lint")
	if got.Status != StatusBlocked {
		t.Errorf("Status = %q, want %q", got.Status, StatusBlocked)
	}
	if got.Detail != "dependencies are not installed" {
		t.Errorf("Detail = %q, want %q", got.Detail, "dependencies are not installed")
	}
	if len(calls) != 0 {
		t.Errorf("calls = %v, want no commands executed for a blocked check", calls)
	}
}

func TestValidate_NodeProject_NoEffectiveManagerBlocks(t *testing.T) {
	insp := project.Inspection{
		Path:           "/projects/node",
		IsNodeProject:  true,
		NodeScriptLint: true,
		// No NodePackageManager (no lockfile) and no declared
		// packageManager, so Doctor never evaluates any package manager
		// tool and there is nothing to run "lint" through.
	}
	report := doctor.Report{
		Checks: []doctor.CheckResult{
			{Tool: "node", OK: true},
		},
	}

	result, err := validate(
		insp.Path,
		fixedInspect(insp, nil),
		fixedRunDoctor(report),
		fakeRunCommand(nil, nil),
		fakeDirExists(true, nil),
	)
	if err != nil {
		t.Fatalf("validate() error = %v", err)
	}

	got := checkFor(t, result.Checks, "lint")
	if got.Status != StatusBlocked {
		t.Errorf("Status = %q, want %q", got.Status, StatusBlocked)
	}
	if got.Detail != "no usable package manager detected" {
		t.Errorf("Detail = %q, want %q", got.Detail, "no usable package manager detected")
	}
}

func TestValidate_NodeProject_MissingPackageManagerBlocks(t *testing.T) {
	insp := project.Inspection{
		Path:               "/projects/node",
		IsNodeProject:      true,
		NodePackageManager: "pnpm",
		NodeScriptLint:     true,
	}
	report := doctor.Report{
		Checks: []doctor.CheckResult{
			{Tool: "node", OK: true},
			{Tool: "pnpm", OK: false, Detail: "executable not found"},
		},
	}

	result, err := validate(
		insp.Path,
		fixedInspect(insp, nil),
		fixedRunDoctor(report),
		fakeRunCommand(nil, nil),
		fakeDirExists(true, nil),
	)
	if err != nil {
		t.Fatalf("validate() error = %v", err)
	}

	got := checkFor(t, result.Checks, "lint")
	if got.Status != StatusBlocked {
		t.Errorf("Status = %q, want %q", got.Status, StatusBlocked)
	}
	if got.Detail != "executable not found" {
		t.Errorf("Detail = %q, want %q", got.Detail, "executable not found")
	}
}

func TestValidate_NodeProject_Successful(t *testing.T) {
	insp := project.Inspection{
		Path:               "/projects/node",
		IsNodeProject:      true,
		NodePackageManager: "yarn",
		NodeScriptBuild:    true,
	}
	report := doctor.Report{
		Checks: []doctor.CheckResult{
			{Tool: "node", OK: true},
			{Tool: "yarn", OK: true},
		},
	}

	result, err := validate(
		insp.Path,
		fixedInspect(insp, nil),
		fixedRunDoctor(report),
		fakeRunCommand(map[string]cmdResult{"yarn run build": {output: "done"}}, nil),
		fakeDirExists(true, nil),
	)
	if err != nil {
		t.Fatalf("validate() error = %v", err)
	}

	got := checkFor(t, result.Checks, "build")
	if got.Status != StatusPassed {
		t.Errorf("Status = %q, want %q", got.Status, StatusPassed)
	}
	if got.Output != "" {
		t.Errorf("Output = %q, want empty on success", got.Output)
	}
}

func TestValidate_NodeProject_FailedWithOutput(t *testing.T) {
	insp := project.Inspection{
		Path:               "/projects/node",
		IsNodeProject:      true,
		NodePackageManager: "npm",
		NodeScriptBuild:    true,
	}
	report := doctor.Report{
		Checks: []doctor.CheckResult{
			{Tool: "node", OK: true},
			{Tool: "npm", OK: true},
		},
	}

	result, err := validate(
		insp.Path,
		fixedInspect(insp, nil),
		fixedRunDoctor(report),
		fakeRunCommand(map[string]cmdResult{
			"npm run build": {output: "TypeScript compilation failed...", err: errors.New("exit status 1")},
		}, nil),
		fakeDirExists(true, nil),
	)
	if err != nil {
		t.Fatalf("validate() error = %v", err)
	}

	got := checkFor(t, result.Checks, "build")
	if got.Status != StatusFailed {
		t.Errorf("Status = %q, want %q", got.Status, StatusFailed)
	}
	if !strings.Contains(got.Output, "TypeScript compilation failed") {
		t.Errorf("Output = %q, want it to contain the captured output", got.Output)
	}
}

func TestValidate_NodeProject_OneFailureDoesNotStopOthers(t *testing.T) {
	insp := project.Inspection{
		Path:               "/projects/node",
		IsNodeProject:      true,
		NodePackageManager: "pnpm",
		NodeScriptLint:     true,
		NodeScriptTest:     true,
	}
	report := doctor.Report{
		Checks: []doctor.CheckResult{
			{Tool: "node", OK: true},
			{Tool: "pnpm", OK: true},
		},
	}

	result, err := validate(
		insp.Path,
		fixedInspect(insp, nil),
		fixedRunDoctor(report),
		fakeRunCommand(map[string]cmdResult{
			"pnpm run lint": {err: errors.New("exit status 1")},
			"pnpm run test": {output: "ok"},
		}, nil),
		fakeDirExists(true, nil),
	)
	if err != nil {
		t.Fatalf("validate() error = %v", err)
	}

	if len(result.Checks) != 2 {
		t.Fatalf("Checks = %v, want exactly two checks", result.Checks)
	}
	if got := checkFor(t, result.Checks, "lint"); got.Status != StatusFailed {
		t.Errorf("lint Status = %q, want %q", got.Status, StatusFailed)
	}
	if got := checkFor(t, result.Checks, "test"); got.Status != StatusPassed {
		t.Errorf("test Status = %q, want %q", got.Status, StatusPassed)
	}
}

func TestValidate_GoProject_ProducesThreeChecks(t *testing.T) {
	insp := project.Inspection{Path: "/projects/go", HasGoMod: true}
	report := doctor.Report{Checks: []doctor.CheckResult{{Tool: "go", OK: true}}}

	result, err := validate(
		insp.Path,
		fixedInspect(insp, nil),
		fixedRunDoctor(report),
		fakeRunCommand(map[string]cmdResult{
			"go test ./...":  {},
			"go vet ./...":   {},
			"go build ./...": {},
		}, nil),
		fakeDirExists(true, nil),
	)
	if err != nil {
		t.Fatalf("validate() error = %v", err)
	}

	if len(result.Checks) != 3 {
		t.Fatalf("Checks = %v, want exactly three checks", result.Checks)
	}
	for _, name := range []string{"go test", "go vet", "go build"} {
		checkFor(t, result.Checks, name)
	}
}

func TestValidate_GoProject_Successful(t *testing.T) {
	insp := project.Inspection{Path: "/projects/go", HasGoMod: true}
	report := doctor.Report{Checks: []doctor.CheckResult{{Tool: "go", OK: true}}}

	result, err := validate(
		insp.Path,
		fixedInspect(insp, nil),
		fixedRunDoctor(report),
		fakeRunCommand(map[string]cmdResult{
			"go test ./...":  {},
			"go vet ./...":   {},
			"go build ./...": {},
		}, nil),
		fakeDirExists(true, nil),
	)
	if err != nil {
		t.Fatalf("validate() error = %v", err)
	}

	for _, name := range []string{"go test", "go vet", "go build"} {
		if got := checkFor(t, result.Checks, name); got.Status != StatusPassed {
			t.Errorf("%s Status = %q, want %q", name, got.Status, StatusPassed)
		}
	}
}

func TestValidate_GoProject_OneFailureDoesNotStopOthers(t *testing.T) {
	insp := project.Inspection{Path: "/projects/go", HasGoMod: true}
	report := doctor.Report{Checks: []doctor.CheckResult{{Tool: "go", OK: true}}}

	result, err := validate(
		insp.Path,
		fixedInspect(insp, nil),
		fixedRunDoctor(report),
		fakeRunCommand(map[string]cmdResult{
			"go test ./...":  {output: "--- FAIL: TestX", err: errors.New("exit status 1")},
			"go vet ./...":   {},
			"go build ./...": {},
		}, nil),
		fakeDirExists(true, nil),
	)
	if err != nil {
		t.Fatalf("validate() error = %v", err)
	}

	if got := checkFor(t, result.Checks, "go test"); got.Status != StatusFailed {
		t.Errorf("go test Status = %q, want %q", got.Status, StatusFailed)
	}
	if got := checkFor(t, result.Checks, "go vet"); got.Status != StatusPassed {
		t.Errorf("go vet Status = %q, want %q", got.Status, StatusPassed)
	}
	if got := checkFor(t, result.Checks, "go build"); got.Status != StatusPassed {
		t.Errorf("go build Status = %q, want %q", got.Status, StatusPassed)
	}
}

func TestValidate_HybridProject(t *testing.T) {
	insp := project.Inspection{
		Path:               "/projects/hybrid",
		HasGoMod:           true,
		IsNodeProject:      true,
		NodePackageManager: "yarn",
		NodeScriptBuild:    true,
	}
	report := doctor.Report{
		Checks: []doctor.CheckResult{
			{Tool: "go", OK: true},
			{Tool: "node", OK: true},
			{Tool: "yarn", OK: true},
		},
	}

	result, err := validate(
		insp.Path,
		fixedInspect(insp, nil),
		fixedRunDoctor(report),
		fakeRunCommand(map[string]cmdResult{
			"go test ./...":  {},
			"go vet ./...":   {},
			"go build ./...": {},
			"yarn run build": {},
		}, nil),
		fakeDirExists(true, nil),
	)
	if err != nil {
		t.Fatalf("validate() error = %v", err)
	}

	if len(result.Checks) != 4 {
		t.Fatalf("Checks = %v, want exactly four checks (three Go, one Node)", result.Checks)
	}
	for _, name := range []string{"go test", "go vet", "go build", "build"} {
		checkFor(t, result.Checks, name)
	}
}

func TestValidate_ExecutionUsesProjectWorkingDirectory(t *testing.T) {
	insp := project.Inspection{
		Path:               "/projects/hybrid",
		HasGoMod:           true,
		IsNodeProject:      true,
		NodePackageManager: "pnpm",
		NodeScriptLint:     true,
	}
	report := doctor.Report{
		Checks: []doctor.CheckResult{
			{Tool: "go", OK: true},
			{Tool: "node", OK: true},
			{Tool: "pnpm", OK: true},
		},
	}

	var calls []commandCall
	_, err := validate(
		insp.Path,
		fixedInspect(insp, nil),
		fixedRunDoctor(report),
		fakeRunCommand(map[string]cmdResult{
			"go test ./...":  {},
			"go vet ./...":   {},
			"go build ./...": {},
			"pnpm run lint":  {},
		}, &calls),
		fakeDirExists(true, nil),
	)
	if err != nil {
		t.Fatalf("validate() error = %v", err)
	}

	if len(calls) != 4 {
		t.Fatalf("calls = %v, want exactly four executed commands", calls)
	}
	for _, call := range calls {
		if call.dir != insp.Path {
			t.Errorf("command %q ran in dir %q, want %q", call.name, call.dir, insp.Path)
		}
	}
}

func TestValidate_ExecutesOnlyFixedCommandSurface(t *testing.T) {
	insp := project.Inspection{
		Path:                "/projects/hybrid",
		HasGoMod:            true,
		IsNodeProject:       true,
		NodePackageManager:  "pnpm",
		NodeScriptLint:      true,
		NodeScriptTypecheck: true,
		NodeScriptTest:      true,
		NodeScriptBuild:     true,
	}
	report := doctor.Report{
		Checks: []doctor.CheckResult{
			{Tool: "go", OK: true},
			{Tool: "node", OK: true},
			{Tool: "pnpm", OK: true},
		},
	}

	var calls []commandCall
	_, err := validate(
		insp.Path,
		fixedInspect(insp, nil),
		fixedRunDoctor(report),
		fakeRunCommand(map[string]cmdResult{
			"go test ./...":      {},
			"go vet ./...":       {},
			"go build ./...":     {},
			"pnpm run lint":      {},
			"pnpm run typecheck": {},
			"pnpm run test":      {},
			"pnpm run build":     {},
		}, &calls),
		fakeDirExists(true, nil),
	)
	if err != nil {
		t.Fatalf("validate() error = %v", err)
	}

	var executed []string
	for _, c := range calls {
		executed = append(executed, strings.Join(append([]string{c.name}, c.args...), " "))
	}
	sort.Strings(executed)

	want := []string{
		"go build ./...",
		"go test ./...",
		"go vet ./...",
		"pnpm run build",
		"pnpm run lint",
		"pnpm run test",
		"pnpm run typecheck",
	}

	if len(executed) != len(want) {
		t.Fatalf("executed commands = %v, want exactly %v", executed, want)
	}
	for i := range want {
		if executed[i] != want[i] {
			t.Fatalf("executed commands = %v, want exactly %v", executed, want)
		}
	}
}

func TestValidate_InvalidProjectPath(t *testing.T) {
	wantErr := errors.New("path does not exist")

	_, err := validate(
		"/projects/missing",
		fixedInspect(project.Inspection{}, wantErr),
		fixedRunDoctor(doctor.Report{}),
		fakeRunCommand(nil, nil),
		fakeDirExists(true, nil),
	)
	if err == nil {
		t.Fatal("validate() error = nil, want an error for an uninspectable project")
	}
}

func TestTruncateOutput(t *testing.T) {
	t.Run("under limit is unchanged", func(t *testing.T) {
		input := "line1\nline2\nline3\n"
		got := truncateOutput(input)
		want := "line1\nline2\nline3"
		if got != want {
			t.Errorf("truncateOutput() = %q, want %q", got, want)
		}
	})

	t.Run("over limit is truncated to maxOutputLines with a marker", func(t *testing.T) {
		lines := make([]string, maxOutputLines+10)
		for i := range lines {
			lines[i] = "line"
		}
		input := strings.Join(lines, "\n")

		got := truncateOutput(input)

		gotLines := strings.Split(got, "\n")
		if len(gotLines) != maxOutputLines+1 {
			t.Fatalf("got %d lines, want %d (maxOutputLines + marker)", len(gotLines), maxOutputLines+1)
		}
		if gotLines[len(gotLines)-1] != "... (output truncated)" {
			t.Errorf("last line = %q, want the truncation marker", gotLines[len(gotLines)-1])
		}
	})

	t.Run("empty output stays empty", func(t *testing.T) {
		if got := truncateOutput(""); got != "" {
			t.Errorf("truncateOutput(\"\") = %q, want empty", got)
		}
	})
}
