package main

import (
	"bytes"
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

func TestRun_ProjectAdd(t *testing.T) {
	store := newTestStore(t)
	projectDir := t.TempDir()
	var out bytes.Buffer

	if err := run([]string{"project", "add", projectDir}, store, &out); err != nil {
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
	if err := run([]string{"project", "add", projectDir}, store, &addOut); err != nil {
		t.Fatalf("run() add error = %v", err)
	}

	var listOut bytes.Buffer
	if err := run([]string{"project", "list"}, store, &listOut); err != nil {
		t.Fatalf("run() list error = %v", err)
	}

	if !strings.Contains(listOut.String(), filepath.Base(projectDir)) {
		t.Errorf("output = %q, want it to list the added project", listOut.String())
	}
}

func TestRun_ProjectList_Empty(t *testing.T) {
	store := newTestStore(t)
	var out bytes.Buffer

	if err := run([]string{"project", "list"}, store, &out); err != nil {
		t.Fatalf("run() error = %v", err)
	}

	if !strings.Contains(out.String(), "No projects registered.") {
		t.Errorf("output = %q, want the empty-list message", out.String())
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newTestStore(t)
			var out bytes.Buffer

			if err := run(tt.args, store, &out); err == nil {
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
