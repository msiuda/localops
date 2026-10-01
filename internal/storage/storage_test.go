package storage_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/msiuda/localops/internal/project"
	"github.com/msiuda/localops/internal/storage"
)

func TestStore_Load_MissingFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projects.json")
	store := storage.New(path)

	projects, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(projects) != 0 {
		t.Errorf("Load() = %v, want empty", projects)
	}
}

func TestStore_SaveAndLoad(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "projects.json")
	store := storage.New(path)

	want := []project.Project{
		{Name: "one", Path: "/tmp/one"},
		{Name: "two", Path: "/tmp/two"},
	}

	if err := store.Save(want); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	got, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(got) != len(want) {
		t.Fatalf("Load() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("Load()[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestStore_Save_FileFormat(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projects.json")
	store := storage.New(path)

	if err := store.Save([]project.Project{{Name: "one", Path: "/tmp/one"}}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile() error = %v", err)
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}

	projects, ok := raw["projects"].([]any)
	if !ok {
		t.Fatalf("raw[%q] = %v, want a projects array", "projects", raw["projects"])
	}
	entry, ok := projects[0].(map[string]any)
	if !ok {
		t.Fatalf("projects[0] = %v, want an object", projects[0])
	}
	if entry["name"] != "one" || entry["path"] != "/tmp/one" {
		t.Errorf("projects[0] = %v, want name/path fields", entry)
	}
}

func TestStore_Load_CorruptedFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projects.json")
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store := storage.New(path)
	if _, err := store.Load(); err == nil {
		t.Fatal("Load() error = nil, want error for corrupted storage")
	}
}

func TestStore_AddProject(t *testing.T) {
	projectDir := t.TempDir()
	store := storage.New(filepath.Join(t.TempDir(), "projects.json"))

	got, err := store.AddProject(projectDir)
	if err != nil {
		t.Fatalf("AddProject() error = %v", err)
	}

	wantPath, err := filepath.Abs(projectDir)
	if err != nil {
		t.Fatalf("Abs() error = %v", err)
	}
	if got.Path != wantPath {
		t.Errorf("Path = %q, want absolute path %q", got.Path, wantPath)
	}
	if got.Name != filepath.Base(wantPath) {
		t.Errorf("Name = %q, want %q", got.Name, filepath.Base(wantPath))
	}

	projects, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(projects) != 1 || projects[0].Path != wantPath {
		t.Fatalf("Load() = %v, want one project at %q", projects, wantPath)
	}
}

func TestStore_AddProject_Duplicate(t *testing.T) {
	projectDir := t.TempDir()
	store := storage.New(filepath.Join(t.TempDir(), "projects.json"))

	if _, err := store.AddProject(projectDir); err != nil {
		t.Fatalf("first AddProject() error = %v", err)
	}

	if _, err := store.AddProject(projectDir); err == nil {
		t.Fatal("second AddProject() error = nil, want an error for a duplicate path")
	}

	projects, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if len(projects) != 1 {
		t.Errorf("Load() = %v, want the duplicate attempt to leave storage unchanged", projects)
	}
}

func TestStore_AddProject_NonexistentPath(t *testing.T) {
	store := storage.New(filepath.Join(t.TempDir(), "projects.json"))
	missing := filepath.Join(t.TempDir(), "missing")

	if _, err := store.AddProject(missing); err == nil {
		t.Fatal("AddProject() error = nil, want an error for a nonexistent path")
	}
}

func TestStore_AddProject_NotADirectory(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store := storage.New(filepath.Join(t.TempDir(), "projects.json"))
	if _, err := store.AddProject(file); err == nil {
		t.Fatal("AddProject() error = nil, want an error for a non-directory path")
	}
}

func TestStore_AddProject_LoadFailureIsError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "projects.json")
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	store := storage.New(path)
	if _, err := store.AddProject(t.TempDir()); err == nil {
		t.Fatal("AddProject() error = nil, want an error when existing storage is corrupted")
	}
}
