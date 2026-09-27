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
