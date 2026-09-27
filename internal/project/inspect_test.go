package project_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/msiuda/localops/internal/project"
)

func TestInspect_PlainDirectory(t *testing.T) {
	dir := t.TempDir()

	insp, err := project.Inspect(dir)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if insp.IsGitRepository {
		t.Error("IsGitRepository = true, want false")
	}
	if insp.HasGoMod {
		t.Error("HasGoMod = true, want false")
	}
	if insp.GoModulePath != "" {
		t.Errorf("GoModulePath = %q, want empty", insp.GoModulePath)
	}
}

func TestInspect_GitRepository(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	insp, err := project.Inspect(dir)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if !insp.IsGitRepository {
		t.Error("IsGitRepository = false, want true")
	}
}

func TestInspect_GitWorktreeFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".git"), []byte("gitdir: ../elsewhere\n"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	insp, err := project.Inspect(dir)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if !insp.IsGitRepository {
		t.Error("IsGitRepository = false, want true for a .git worktree file")
	}
}

func TestInspect_GoModule(t *testing.T) {
	dir := t.TempDir()
	content := "module github.com/example/foo\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	insp, err := project.Inspect(dir)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if !insp.HasGoMod {
		t.Error("HasGoMod = false, want true")
	}
	if insp.GoModulePath != "github.com/example/foo" {
		t.Errorf("GoModulePath = %q, want %q", insp.GoModulePath, "github.com/example/foo")
	}
}

func TestInspect_GoModule_QuotedPath(t *testing.T) {
	dir := t.TempDir()
	content := "module \"github.com/example/foo\"\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	insp, err := project.Inspect(dir)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if insp.GoModulePath != "github.com/example/foo" {
		t.Errorf("GoModulePath = %q, want %q", insp.GoModulePath, "github.com/example/foo")
	}
}

func TestInspect_GoModule_MissingModuleDirective(t *testing.T) {
	dir := t.TempDir()
	content := "go 1.21\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := project.Inspect(dir); err == nil {
		t.Fatal("Inspect() error = nil, want error for go.mod without a module directive")
	}
}

func TestInspect_GoModule_Malformed(t *testing.T) {
	dir := t.TempDir()
	content := "this is not a valid go.mod file {{{\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := project.Inspect(dir); err == nil {
		t.Fatal("Inspect() error = nil, want error for malformed go.mod")
	}
}

func TestInspect_NotDirectory(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file.txt")
	if err := os.WriteFile(file, []byte("data"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := project.Inspect(file); err == nil {
		t.Fatal("Inspect() error = nil, want error for non-directory path")
	}
}

func TestInspect_MissingPath(t *testing.T) {
	dir := t.TempDir()
	missing := filepath.Join(dir, "missing")

	if _, err := project.Inspect(missing); err == nil {
		t.Fatal("Inspect() error = nil, want error for missing path")
	}
}
