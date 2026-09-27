package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/msiuda/localops/internal/storage"
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
