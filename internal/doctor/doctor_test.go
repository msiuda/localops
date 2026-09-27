package doctor

import (
	"errors"
	"testing"

	"github.com/msiuda/localops/internal/project"
)

func fakeLookPath(available map[string]string) func(string) (string, error) {
	return func(tool string) (string, error) {
		if path, ok := available[tool]; ok {
			return path, nil
		}
		return "", errors.New("not found")
	}
}

func checkFor(t *testing.T, checks []CheckResult, tool string) CheckResult {
	t.Helper()
	for _, c := range checks {
		if c.Tool == tool {
			return c
		}
	}
	t.Fatalf("no check found for tool %q in %v", tool, checks)
	return CheckResult{}
}

func TestRun_PlainProject_NoRequiredTools(t *testing.T) {
	insp := project.Inspection{Path: "/tmp/plain"}

	report := run(insp, fakeLookPath(nil))

	if len(report.Checks) != 0 {
		t.Errorf("Checks = %v, want none", report.Checks)
	}
	if report.Path != "/tmp/plain" {
		t.Errorf("Path = %q, want %q", report.Path, "/tmp/plain")
	}
}

func TestRun_GitRepository(t *testing.T) {
	insp := project.Inspection{IsGitRepository: true}

	report := run(insp, fakeLookPath(map[string]string{"git": "/usr/bin/git"}))

	if len(report.Checks) != 1 {
		t.Fatalf("Checks = %v, want exactly one check", report.Checks)
	}
	if got := checkFor(t, report.Checks, "git"); !got.Available {
		t.Errorf("git Available = false, want true")
	}
}

func TestRun_GoModule(t *testing.T) {
	insp := project.Inspection{HasGoMod: true}

	report := run(insp, fakeLookPath(map[string]string{"go": "/usr/local/go/bin/go"}))

	if len(report.Checks) != 1 {
		t.Fatalf("Checks = %v, want exactly one check", report.Checks)
	}
	if got := checkFor(t, report.Checks, "go"); !got.Available {
		t.Errorf("go Available = false, want true")
	}
}

func TestRun_NodeProject(t *testing.T) {
	insp := project.Inspection{IsNodeProject: true}

	report := run(insp, fakeLookPath(map[string]string{"node": "/usr/bin/node"}))

	if len(report.Checks) != 1 {
		t.Fatalf("Checks = %v, want exactly one check", report.Checks)
	}
	if got := checkFor(t, report.Checks, "node"); !got.Available {
		t.Errorf("node Available = false, want true")
	}
}

func TestRun_NodeProject_Pnpm(t *testing.T) {
	insp := project.Inspection{IsNodeProject: true, NodePackageManager: "pnpm"}

	report := run(insp, fakeLookPath(map[string]string{
		"node": "/usr/bin/node",
		"pnpm": "/usr/bin/pnpm",
	}))

	if len(report.Checks) != 2 {
		t.Fatalf("Checks = %v, want exactly two checks", report.Checks)
	}
	if got := checkFor(t, report.Checks, "pnpm"); !got.Available {
		t.Errorf("pnpm Available = false, want true")
	}
}

func TestRun_NodeProject_Yarn(t *testing.T) {
	insp := project.Inspection{IsNodeProject: true, NodePackageManager: "yarn"}

	report := run(insp, fakeLookPath(map[string]string{
		"node": "/usr/bin/node",
		"yarn": "/usr/bin/yarn",
	}))

	if len(report.Checks) != 2 {
		t.Fatalf("Checks = %v, want exactly two checks", report.Checks)
	}
	if got := checkFor(t, report.Checks, "yarn"); !got.Available {
		t.Errorf("yarn Available = false, want true")
	}
}

func TestRun_NodeProject_Npm(t *testing.T) {
	insp := project.Inspection{IsNodeProject: true, NodePackageManager: "npm"}

	report := run(insp, fakeLookPath(map[string]string{
		"node": "/usr/bin/node",
		"npm":  "/usr/bin/npm",
	}))

	if len(report.Checks) != 2 {
		t.Fatalf("Checks = %v, want exactly two checks", report.Checks)
	}
	if got := checkFor(t, report.Checks, "npm"); !got.Available {
		t.Errorf("npm Available = false, want true")
	}
}

func TestRun_MissingExecutable(t *testing.T) {
	insp := project.Inspection{IsGitRepository: true}

	report := run(insp, fakeLookPath(nil))

	got := checkFor(t, report.Checks, "git")
	if got.Available {
		t.Error("Available = true, want false")
	}
	if got.Detail == "" {
		t.Error("Detail is empty, want a useful message")
	}
}

func TestRun_AllRequiredTools(t *testing.T) {
	insp := project.Inspection{
		IsGitRepository:    true,
		HasGoMod:           true,
		IsNodeProject:      true,
		NodePackageManager: "yarn",
	}

	report := run(insp, fakeLookPath(map[string]string{
		"git":  "/usr/bin/git",
		"go":   "/usr/local/go/bin/go",
		"node": "/usr/bin/node",
		"yarn": "/usr/bin/yarn",
	}))

	if len(report.Checks) != 4 {
		t.Fatalf("Checks = %v, want exactly four checks", report.Checks)
	}
}
