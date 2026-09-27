package doctor

import (
	"errors"
	"testing"

	"github.com/msiuda/localops/internal/project"
)

func fakeLookPath(available map[string]string) lookPathFunc {
	return func(tool string) (string, error) {
		if path, ok := available[tool]; ok {
			return path, nil
		}
		return "", errors.New("not found")
	}
}

// fakeRunVersion returns a runVersionFunc that reports versions from the
// given map, fails for tools listed in failing, and records every call it
// receives into calls.
func fakeRunVersion(versions map[string]string, failing map[string]bool, calls *[]versionCall) runVersionFunc {
	return func(tool string, args ...string) (string, error) {
		if calls != nil {
			*calls = append(*calls, versionCall{tool: tool, args: append([]string(nil), args...)})
		}
		if failing[tool] {
			return "", errors.New("boom")
		}
		return versions[tool], nil
	}
}

type versionCall struct {
	tool string
	args []string
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

func callFor(t *testing.T, calls []versionCall, tool string) versionCall {
	t.Helper()
	for _, c := range calls {
		if c.tool == tool {
			return c
		}
	}
	t.Fatalf("no version call found for tool %q in %v", tool, calls)
	return versionCall{}
}

func argsEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestRun_PlainProject_NoRequiredTools(t *testing.T) {
	insp := project.Inspection{Path: "/tmp/plain"}

	report := run(insp, fakeLookPath(nil), fakeRunVersion(nil, nil, nil))

	if len(report.Checks) != 0 {
		t.Errorf("Checks = %v, want none", report.Checks)
	}
	if report.Path != "/tmp/plain" {
		t.Errorf("Path = %q, want %q", report.Path, "/tmp/plain")
	}
}

func TestRun_GitRepository(t *testing.T) {
	insp := project.Inspection{IsGitRepository: true}

	report := run(
		insp,
		fakeLookPath(map[string]string{"git": "/usr/bin/git"}),
		fakeRunVersion(map[string]string{"git": "git version 2.50.1"}, nil, nil),
	)

	if len(report.Checks) != 1 {
		t.Fatalf("Checks = %v, want exactly one check", report.Checks)
	}
	got := checkFor(t, report.Checks, "git")
	if !got.Available {
		t.Errorf("git Available = false, want true")
	}
	if got.Version != "git version 2.50.1" {
		t.Errorf("git Version = %q, want %q", got.Version, "git version 2.50.1")
	}
}

func TestRun_GoModule(t *testing.T) {
	insp := project.Inspection{HasGoMod: true}

	report := run(
		insp,
		fakeLookPath(map[string]string{"go": "/usr/local/go/bin/go"}),
		fakeRunVersion(map[string]string{"go": "go version go1.27.1 darwin/arm64"}, nil, nil),
	)

	if len(report.Checks) != 1 {
		t.Fatalf("Checks = %v, want exactly one check", report.Checks)
	}
	if got := checkFor(t, report.Checks, "go"); !got.Available {
		t.Errorf("go Available = false, want true")
	}
}

func TestRun_NodeProject(t *testing.T) {
	insp := project.Inspection{IsNodeProject: true}

	report := run(
		insp,
		fakeLookPath(map[string]string{"node": "/usr/bin/node"}),
		fakeRunVersion(map[string]string{"node": "v22.14.0"}, nil, nil),
	)

	if len(report.Checks) != 1 {
		t.Fatalf("Checks = %v, want exactly one check", report.Checks)
	}
	if got := checkFor(t, report.Checks, "node"); !got.Available {
		t.Errorf("node Available = false, want true")
	}
}

func TestRun_NodeProject_Pnpm(t *testing.T) {
	insp := project.Inspection{IsNodeProject: true, NodePackageManager: "pnpm"}

	report := run(
		insp,
		fakeLookPath(map[string]string{
			"node": "/usr/bin/node",
			"pnpm": "/usr/bin/pnpm",
		}),
		fakeRunVersion(map[string]string{"node": "v22.14.0", "pnpm": "10.9.2"}, nil, nil),
	)

	if len(report.Checks) != 2 {
		t.Fatalf("Checks = %v, want exactly two checks", report.Checks)
	}
	if got := checkFor(t, report.Checks, "pnpm"); !got.Available {
		t.Errorf("pnpm Available = false, want true")
	}
}

func TestRun_NodeProject_Yarn(t *testing.T) {
	insp := project.Inspection{IsNodeProject: true, NodePackageManager: "yarn"}

	report := run(
		insp,
		fakeLookPath(map[string]string{
			"node": "/usr/bin/node",
			"yarn": "/usr/bin/yarn",
		}),
		fakeRunVersion(map[string]string{"node": "v22.14.0", "yarn": "1.22.22"}, nil, nil),
	)

	if len(report.Checks) != 2 {
		t.Fatalf("Checks = %v, want exactly two checks", report.Checks)
	}
	if got := checkFor(t, report.Checks, "yarn"); !got.Available {
		t.Errorf("yarn Available = false, want true")
	}
}

func TestRun_NodeProject_Npm(t *testing.T) {
	insp := project.Inspection{IsNodeProject: true, NodePackageManager: "npm"}

	report := run(
		insp,
		fakeLookPath(map[string]string{
			"node": "/usr/bin/node",
			"npm":  "/usr/bin/npm",
		}),
		fakeRunVersion(map[string]string{"node": "v22.14.0", "npm": "10.9.2"}, nil, nil),
	)

	if len(report.Checks) != 2 {
		t.Fatalf("Checks = %v, want exactly two checks", report.Checks)
	}
	if got := checkFor(t, report.Checks, "npm"); !got.Available {
		t.Errorf("npm Available = false, want true")
	}
}

func TestRun_MissingExecutable(t *testing.T) {
	insp := project.Inspection{IsGitRepository: true}

	report := run(insp, fakeLookPath(nil), fakeRunVersion(nil, nil, nil))

	got := checkFor(t, report.Checks, "git")
	if got.Available {
		t.Error("Available = true, want false")
	}
	if got.Version != "" {
		t.Errorf("Version = %q, want empty for a missing executable", got.Version)
	}
	if got.Detail == "" {
		t.Error("Detail is empty, want a useful message")
	}
}

func TestRun_VersionCommandFails(t *testing.T) {
	insp := project.Inspection{IsGitRepository: true}

	report := run(
		insp,
		fakeLookPath(map[string]string{"git": "/usr/bin/git"}),
		fakeRunVersion(nil, map[string]bool{"git": true}, nil),
	)

	got := checkFor(t, report.Checks, "git")
	if !got.Available {
		t.Error("Available = false, want true for an executable that was found")
	}
	if got.Version != "" {
		t.Errorf("Version = %q, want empty when the version command fails", got.Version)
	}
	if got.Detail == "" {
		t.Error("Detail is empty, want a useful message describing the version failure")
	}
}

func TestRun_VersionCommandEmptyOutput(t *testing.T) {
	insp := project.Inspection{IsGitRepository: true}

	report := run(
		insp,
		fakeLookPath(map[string]string{"git": "/usr/bin/git"}),
		fakeRunVersion(map[string]string{"git": ""}, nil, nil),
	)

	got := checkFor(t, report.Checks, "git")
	if !got.Available {
		t.Error("Available = false, want true for an executable that was found")
	}
	if got.Version != "" {
		t.Errorf("Version = %q, want empty when the version command returns empty output", got.Version)
	}
	if got.Detail != "version command returned empty output" {
		t.Errorf("Detail = %q, want %q", got.Detail, "version command returned empty output")
	}
}

func TestRun_AllRequiredTools(t *testing.T) {
	insp := project.Inspection{
		IsGitRepository:    true,
		HasGoMod:           true,
		IsNodeProject:      true,
		NodePackageManager: "yarn",
	}

	report := run(
		insp,
		fakeLookPath(map[string]string{
			"git":  "/usr/bin/git",
			"go":   "/usr/local/go/bin/go",
			"node": "/usr/bin/node",
			"yarn": "/usr/bin/yarn",
		}),
		fakeRunVersion(map[string]string{
			"git":  "git version 2.50.1",
			"go":   "go version go1.27.1 darwin/arm64",
			"node": "v22.14.0",
			"yarn": "1.22.22",
		}, nil, nil),
	)

	if len(report.Checks) != 4 {
		t.Fatalf("Checks = %v, want exactly four checks", report.Checks)
	}
}

func TestRun_OneFailedVersionDoesNotStopOthers(t *testing.T) {
	insp := project.Inspection{
		IsGitRepository:    true,
		HasGoMod:           true,
		IsNodeProject:      true,
		NodePackageManager: "yarn",
	}

	report := run(
		insp,
		fakeLookPath(map[string]string{
			"git":  "/usr/bin/git",
			"go":   "/usr/local/go/bin/go",
			"node": "/usr/bin/node",
			"yarn": "/usr/bin/yarn",
		}),
		fakeRunVersion(
			map[string]string{
				"go":   "go version go1.27.1 darwin/arm64",
				"node": "v22.14.0",
				"yarn": "1.22.22",
			},
			map[string]bool{"git": true},
			nil,
		),
	)

	if len(report.Checks) != 4 {
		t.Fatalf("Checks = %v, want exactly four checks despite one version failure", report.Checks)
	}

	if got := checkFor(t, report.Checks, "git"); got.Version != "" || got.Detail == "" {
		t.Errorf("git check = %+v, want a failed version lookup", got)
	}
	for _, tool := range []string{"go", "node", "yarn"} {
		if got := checkFor(t, report.Checks, tool); got.Version == "" {
			t.Errorf("%s check = %+v, want a successful version lookup", tool, got)
		}
	}
}

func TestRun_VersionArgs(t *testing.T) {
	tests := []struct {
		tool string
		insp project.Inspection
		want []string
	}{
		{tool: "git", insp: project.Inspection{IsGitRepository: true}, want: []string{"--version"}},
		{tool: "go", insp: project.Inspection{HasGoMod: true}, want: []string{"version"}},
		{tool: "node", insp: project.Inspection{IsNodeProject: true}, want: []string{"--version"}},
		{
			tool: "npm",
			insp: project.Inspection{IsNodeProject: true, NodePackageManager: "npm"},
			want: []string{"--version"},
		},
		{
			tool: "pnpm",
			insp: project.Inspection{IsNodeProject: true, NodePackageManager: "pnpm"},
			want: []string{"--version"},
		},
		{
			tool: "yarn",
			insp: project.Inspection{IsNodeProject: true, NodePackageManager: "yarn"},
			want: []string{"--version"},
		},
	}

	available := map[string]string{
		"git": "/usr/bin/git", "go": "/usr/bin/go", "node": "/usr/bin/node",
		"npm": "/usr/bin/npm", "pnpm": "/usr/bin/pnpm", "yarn": "/usr/bin/yarn",
	}

	for _, tt := range tests {
		t.Run(tt.tool, func(t *testing.T) {
			var calls []versionCall
			run(tt.insp, fakeLookPath(available), fakeRunVersion(nil, nil, &calls))

			call := callFor(t, calls, tt.tool)
			if !argsEqual(call.args, tt.want) {
				t.Errorf("args for %s = %v, want %v", tt.tool, call.args, tt.want)
			}
		})
	}
}

func TestTrimVersionOutput(t *testing.T) {
	got := trimVersionOutput([]byte("  git version 2.50.1  \n\n"))
	want := "git version 2.50.1"
	if got != want {
		t.Errorf("trimVersionOutput() = %q, want %q", got, want)
	}
}
