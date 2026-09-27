package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/msiuda/localops/internal/project"
)

func TestFromPath(t *testing.T) {
	dir := t.TempDir()
	sub := filepath.Join(dir, "myproject")
	if err := os.Mkdir(sub, 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	p, err := project.FromPath(sub)
	if err != nil {
		t.Fatalf("FromPath() error = %v", err)
	}

	wantPath, err := filepath.Abs(sub)
	if err != nil {
		t.Fatalf("filepath.Abs() error = %v", err)
	}

	if p.Path != wantPath {
		t.Errorf("Path = %q, want %q", p.Path, wantPath)
	}
	if p.Name != "myproject" {
		t.Errorf("Name = %q, want %q", p.Name, "myproject")
	}
}

func TestFromPath_NotExist(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing")

	if _, err := project.FromPath(missing); err == nil {
		t.Fatal("FromPath() error = nil, want error for missing path")
	}
}

func TestFromPath_NotDirectory(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("data"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := project.FromPath(file); err == nil {
		t.Fatal("FromPath() error = nil, want error for non-directory path")
	}
}
