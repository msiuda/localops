package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/msiuda/localops/internal/doctor"
	"github.com/msiuda/localops/internal/environment"
	"github.com/msiuda/localops/internal/project"
	"github.com/msiuda/localops/internal/storage"
	"github.com/msiuda/localops/internal/validation"
)

func newTestStore(t *testing.T) *storage.Store {
	t.Helper()
	path := filepath.Join(t.TempDir(), "projects.json")
	return storage.New(path)
}

func testStoreFunc(store *storage.Store) storeFunc {
	return func() (*storage.Store, error) {
		return store, nil
	}
}

func failingStoreFunc() storeFunc {
	return func() (*storage.Store, error) {
		return nil, errors.New("store unavailable")
	}
}

func TestRun_ProjectAdd(t *testing.T) {
	store := newTestStore(t)
	projectDir := t.TempDir()
	var out bytes.Buffer

	if err := run([]string{"project", "add", projectDir}, testStoreFunc(store), &out); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if !strings.Contains(out.String(), filepath.Base(projectDir)) {
		t.Errorf("output = %q, want it to mention the added project", out.String())
	}

	projects, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(projects) != 1 {
		t.Fatalf("Load() = %v, want one project", projects)
	}
	if projects[0].Path != mustAbs(t, projectDir) {
		t.Errorf("projects[0].Path = %q, want %q", projects[0].Path, mustAbs(t, projectDir))
	}
}

func TestRun_ProjectAdd_Duplicate(t *testing.T) {
	store := newTestStore(t)
	projectDir := t.TempDir()
	var out bytes.Buffer

	if err := run([]string{"project", "add", projectDir}, testStoreFunc(store), &out); err != nil {
		t.Fatalf("first run() error = %v", err)
	}
	if err := run([]string{"project", "add", projectDir}, testStoreFunc(store), &out); err == nil {
		t.Fatal("second run() error = nil, want an error for a duplicate path")
	}
}

func TestRun_ProjectList(t *testing.T) {
	store := newTestStore(t)
	projectDir := t.TempDir()
	var addOut bytes.Buffer
	if err := run([]string{"project", "add", projectDir}, testStoreFunc(store), &addOut); err != nil {
		t.Fatalf("run() add error = %v", err)
	}

	var listOut bytes.Buffer
	if err := run([]string{"project", "list"}, testStoreFunc(store), &listOut); err != nil {
		t.Fatalf("run() list error = %v", err)
	}

	if !strings.Contains(listOut.String(), filepath.Base(projectDir)) {
		t.Errorf("output = %q, want it to list the added project", listOut.String())
	}
}

func TestRun_ProjectList_Empty(t *testing.T) {
	store := newTestStore(t)
	var out bytes.Buffer

	if err := run([]string{"project", "list"}, testStoreFunc(store), &out); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if !strings.Contains(out.String(), "No projects registered.") {
		t.Errorf("output = %q, want the empty-list message", out.String())
	}
}

func TestRun_ProjectInspect(t *testing.T) {
	projectDir := t.TempDir()
	goModPath := filepath.Join(projectDir, "go.mod")
	if err := os.WriteFile(goModPath, []byte("module github.com/example/foo\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var out bytes.Buffer
	if err := run([]string{"project", "inspect", projectDir}, failingStoreFunc(), &out); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	got := out.String()
	for _, want := range []string{
		filepath.Base(projectDir),
		"Git repository: false",
		"Go module: true",
		"github.com/example/foo",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output = %q, want it to contain %q", got, want)
		}
	}
}

func TestRun_Doctor_PlainProject(t *testing.T) {
	projectDir := t.TempDir()

	var out bytes.Buffer
	if err := run([]string{"doctor", projectDir}, failingStoreFunc(), &out); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "Doctor: ") {
		t.Errorf("output = %q, want it to contain a Doctor header", got)
	}
	if !strings.Contains(got, "No required executables detected.") {
		t.Errorf("output = %q, want it to report no required executables", got)
	}
}

func TestRun_Doctor_InvalidPath(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")

	var out bytes.Buffer
	if err := run([]string{"doctor", missing}, failingStoreFunc(), &out); err == nil {
		t.Fatal("run() error = nil, want an error for a missing project path")
	}
}

func TestRun_Overview_Empty(t *testing.T) {
	store := newTestStore(t)
	var out bytes.Buffer

	if err := run([]string{"overview"}, testStoreFunc(store), &out); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "No projects registered.") {
		t.Errorf("output = %q, want the empty-registry message", got)
	}
	if !strings.Contains(got, "localops project add <path>") {
		t.Errorf("output = %q, want a hint to register a project", got)
	}
}

func TestRun_Overview_HealthyAndUnavailable(t *testing.T) {
	store := newTestStore(t)

	healthyDir := t.TempDir()
	missingDir := filepath.Join(t.TempDir(), "missing")

	projects := []project.Project{
		{Name: filepath.Base(healthyDir), Path: healthyDir},
		{Name: "gone", Path: missingDir},
	}
	if err := store.Save(projects); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var out bytes.Buffer
	if err := run([]string{"overview"}, testStoreFunc(store), &out); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "[OK] "+filepath.Base(healthyDir)) {
		t.Errorf("output = %q, want the healthy project marked [OK]", got)
	}
	if !strings.Contains(got, "No issues") {
		t.Errorf("output = %q, want the healthy project to report no issues", got)
	}
	if !strings.Contains(got, "[UNAVAILABLE] gone") {
		t.Errorf("output = %q, want the missing project marked [UNAVAILABLE]", got)
	}
	if !strings.Contains(got, "2 projects: 1 healthy, 0 with issues, 1 unavailable") {
		t.Errorf("output = %q, want a matching summary line", got)
	}
}

func TestRun_Overview_TechnologySummaryUsesDoctorEffectiveManager(t *testing.T) {
	store := newTestStore(t)
	projectDir := t.TempDir()

	packageJSON := `{"name": "demo", "packageManager": "pnpm@10.4.1"}`
	if err := os.WriteFile(filepath.Join(projectDir, "package.json"), []byte(packageJSON), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "yarn.lock"), []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if err := store.Save([]project.Project{{Name: filepath.Base(projectDir), Path: projectDir}}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	var out bytes.Buffer
	if err := run([]string{"overview"}, testStoreFunc(store), &out); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	// package.json declares pnpm, but the lockfile is yarn.lock: Doctor
	// reconciles this to pnpm as the effective manager (and reports the
	// mismatch as an issue), so the summary must reflect pnpm, not yarn.
	got := out.String()
	if !strings.Contains(got, "Node.js · pnpm") {
		t.Errorf("output = %q, want the technology summary to show pnpm (Doctor's effective manager)", got)
	}
	if strings.Contains(got, "Node.js · yarn") {
		t.Errorf("output = %q, want it not to show yarn (the lockfile-detected manager) as the summary", got)
	}
}

func TestRun_Overview_LoadFailure(t *testing.T) {
	var out bytes.Buffer

	if err := run([]string{"overview"}, failingStoreFunc(), &out); err == nil {
		t.Fatal("run() error = nil, want an error when the store cannot be resolved")
	}
}

func TestRun_Validate_PlainProject(t *testing.T) {
	projectDir := t.TempDir()

	var out bytes.Buffer
	if err := run([]string{"validate", projectDir}, failingStoreFunc(), &out); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "LocalOps Validation") {
		t.Errorf("output = %q, want it to contain the Validation header", got)
	}
	if !strings.Contains(got, "No required executables detected.") {
		t.Errorf("output = %q, want it to report no required executables", got)
	}
	if !strings.Contains(got, "No validation checks detected.") {
		t.Errorf("output = %q, want it to report no validation checks", got)
	}
}

func TestRun_Validate_InvalidPath(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")

	var out bytes.Buffer
	if err := run([]string{"validate", missing}, failingStoreFunc(), &out); err == nil {
		t.Fatal("run() error = nil, want an error for a missing project path")
	}
}

func TestRenderValidation(t *testing.T) {
	result := validation.Result{
		Path: "/projects/demo",
		Doctor: doctor.Report{
			Checks: []doctor.CheckResult{
				{Tool: "node", Available: true, Version: "v22.14.0", OK: true},
			},
		},
		Checks: []validation.CheckResult{
			{Name: "lint", Command: "pnpm run lint", Status: validation.StatusPassed, Duration: 1800 * time.Millisecond},
			{
				Name:     "build",
				Command:  "pnpm run build",
				Status:   validation.StatusFailed,
				Duration: 7400 * time.Millisecond,
				Output:   "TypeScript compilation failed...",
			},
			{Name: "test", Command: "pnpm run test", Status: validation.StatusBlocked, Detail: "dependencies are not installed"},
		},
	}

	var out bytes.Buffer
	renderValidation(&out, result)

	got := out.String()
	for _, want := range []string{
		"Project: /projects/demo",
		"[OK] node — v22.14.0",
		"[OK] lint —",
		"[FAIL] build —",
		"TypeScript compilation failed...",
		"[BLOCKED] test — dependencies are not installed",
		"1 passed, 1 failed, 1 blocked",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output = %q, want it to contain %q", got, want)
		}
	}
}

func TestRun_Environment_NoContract(t *testing.T) {
	projectDir := t.TempDir()

	var out bytes.Buffer
	if err := run([]string{"environment", projectDir}, failingStoreFunc(), &out); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "No environment contract detected.") {
		t.Errorf("output = %q, want the no-contract message", got)
	}
	if !strings.Contains(got, ".env.example") || !strings.Contains(got, ".env.sample") || !strings.Contains(got, ".env.template") {
		t.Errorf("output = %q, want it to list the supported contract files", got)
	}
}

func TestRun_Environment_ContractAndLocalFiles(t *testing.T) {
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, ".env.example"), []byte("DATABASE_URL=x\nJWT_SECRET=x\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, ".env"), []byte("DATABASE_URL=postgres://real\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var out bytes.Buffer
	if err := run([]string{"environment", projectDir}, failingStoreFunc(), &out); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	got := out.String()
	for _, want := range []string{
		"LocalOps Environment",
		"Contract sources",
		".env.example — 2 variables",
		"Local sources",
		".env — 1 variable",
		"[OK] DATABASE_URL — .env",
		"[MISSING] JWT_SECRET",
		"1 satisfied, 1 missing",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("output = %q, want it to contain %q", got, want)
		}
	}
}

func TestRun_Environment_InvalidPath(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")

	var out bytes.Buffer
	if err := run([]string{"environment", missing}, failingStoreFunc(), &out); err == nil {
		t.Fatal("run() error = nil, want an error for a missing project path")
	}
}

func TestRun_Environment_SecretValuesNeverAppearInOutput(t *testing.T) {
	const secret = "SUPER_SECRET_VALUE_SHOULD_NEVER_APPEAR"
	projectDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projectDir, ".env.example"), []byte("API_KEY=x\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, ".env"), []byte("API_KEY="+secret+"\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	var out bytes.Buffer
	if err := run([]string{"environment", projectDir}, failingStoreFunc(), &out); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	got := out.String()
	if strings.Contains(got, secret) {
		t.Fatalf("output contains the secret value:\n%s", got)
	}
	if !strings.Contains(got, "[OK] API_KEY — .env") {
		t.Errorf("output = %q, want API_KEY marked satisfied via .env", got)
	}
}

func TestRenderEnvironment_ProcessEnvironmentSection(t *testing.T) {
	t.Run("hidden when unused", func(t *testing.T) {
		result := environment.Result{
			Path:            "/projects/demo",
			ContractSources: []environment.ContractSource{{File: ".env.example", VariableCount: 1}},
			LocalSources:    []environment.LocalSource{{File: ".env", VariableCount: 1}},
			Variables: []environment.VariableStatus{
				{Name: "DATABASE_URL", DeclaredIn: []string{".env.example"}, Satisfied: true, Source: ".env"},
			},
		}

		var out bytes.Buffer
		renderEnvironment(&out, result)

		if strings.Contains(out.String(), "process environment") {
			t.Errorf("output = %q, want no process environment line when it satisfied nothing", out.String())
		}
	})

	t.Run("shown when it satisfies a variable", func(t *testing.T) {
		result := environment.Result{
			Path:            "/projects/demo",
			ContractSources: []environment.ContractSource{{File: ".env.example", VariableCount: 1}},
			Variables: []environment.VariableStatus{
				{Name: "HOME", DeclaredIn: []string{".env.example"}, Satisfied: true, Source: "process environment"},
			},
		}

		var out bytes.Buffer
		renderEnvironment(&out, result)

		if !strings.Contains(out.String(), "process environment") {
			t.Errorf("output = %q, want a process environment line", out.String())
		}
	})
}

func TestRenderEnvironment_Findings(t *testing.T) {
	result := environment.Result{
		Path:            "/projects/demo",
		ContractSources: []environment.ContractSource{{File: ".env.example", VariableCount: 1}},
		Variables: []environment.VariableStatus{
			{Name: "DATABASE_URL", DeclaredIn: []string{".env.example"}},
		},
		Findings: []environment.Finding{
			{Source: ".env.example", Line: 8, Detail: "could not be parsed"},
		},
	}

	var out bytes.Buffer
	renderEnvironment(&out, result)

	got := out.String()
	if !strings.Contains(got, "Findings") {
		t.Errorf("output = %q, want a Findings section", got)
	}
	if !strings.Contains(got, "- .env.example: line 8 could not be parsed") {
		t.Errorf("output = %q, want the finding line", got)
	}
}

func TestRun_Help(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "top-level --help", args: []string{"--help"}, want: topLevelHelp},
		{name: "top-level help", args: []string{"help"}, want: topLevelHelp},
		{name: "project --help", args: []string{"project", "--help"}, want: projectHelp},
		{name: "project help", args: []string{"project", "help"}, want: projectHelp},
		{name: "doctor --help", args: []string{"doctor", "--help"}, want: doctorHelp},
		{name: "doctor help", args: []string{"doctor", "help"}, want: doctorHelp},
		{name: "overview --help", args: []string{"overview", "--help"}, want: overviewHelp},
		{name: "overview help", args: []string{"overview", "help"}, want: overviewHelp},
		{name: "validate --help", args: []string{"validate", "--help"}, want: validateHelp},
		{name: "validate help", args: []string{"validate", "help"}, want: validateHelp},
		{name: "environment --help", args: []string{"environment", "--help"}, want: environmentHelp},
		{name: "environment help", args: []string{"environment", "help"}, want: environmentHelp},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer

			if err := run(tt.args, failingStoreFunc(), &out); err != nil {
				t.Fatalf("run(%v) error = %v", tt.args, err)
			}

			if out.String() != tt.want {
				t.Errorf("run(%v) output = %q, want %q", tt.args, out.String(), tt.want)
			}
		})
	}
}

func TestRun_InvalidArguments(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "no arguments", args: []string{}},
		{name: "unknown top-level command", args: []string{"bogus"}},
		{name: "missing project subcommand", args: []string{"project"}},
		{name: "unknown project subcommand", args: []string{"project", "bogus"}},
		{name: "add without path", args: []string{"project", "add"}},
		{name: "add with too many arguments", args: []string{"project", "add", "a", "b"}},
		{name: "list with unexpected argument", args: []string{"project", "list", "extra"}},
		{name: "inspect without path", args: []string{"project", "inspect"}},
		{name: "inspect with too many arguments", args: []string{"project", "inspect", "a", "b"}},
		{name: "inspect missing path", args: []string{"project", "inspect", filepath.Join(t.TempDir(), "missing")}},
		{name: "doctor without path", args: []string{"doctor"}},
		{name: "doctor with too many arguments", args: []string{"doctor", "a", "b"}},
		{name: "overview with unexpected argument", args: []string{"overview", "extra"}},
		{name: "validate without path", args: []string{"validate"}},
		{name: "validate with too many arguments", args: []string{"validate", "a", "b"}},
		{name: "environment without path", args: []string{"environment"}},
		{name: "environment with too many arguments", args: []string{"environment", "a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer

			if err := run(tt.args, failingStoreFunc(), &out); err == nil {
				t.Fatalf("run(%v) error = nil, want an error", tt.args)
			}
		})
	}
}

func mustAbs(t *testing.T, path string) string {
	t.Helper()
	abs, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}
	return abs
}
