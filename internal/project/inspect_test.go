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

func TestInspect_NodeProject_Absent(t *testing.T) {
	dir := t.TempDir()

	insp, err := project.Inspect(dir)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if insp.IsNodeProject {
		t.Error("IsNodeProject = true, want false")
	}
	if insp.NodePackageName != "" {
		t.Errorf("NodePackageName = %q, want empty", insp.NodePackageName)
	}
	if insp.NodePackageManager != "" {
		t.Errorf("NodePackageManager = %q, want empty", insp.NodePackageManager)
	}
}

func TestInspect_NodeProject_PackageJSON(t *testing.T) {
	dir := t.TempDir()
	content := `{"name": "example-package", "version": "1.0.0"}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	insp, err := project.Inspect(dir)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if !insp.IsNodeProject {
		t.Error("IsNodeProject = false, want true")
	}
	if insp.NodePackageName != "example-package" {
		t.Errorf("NodePackageName = %q, want %q", insp.NodePackageName, "example-package")
	}
}

func TestInspect_NodeProject_MalformedPackageJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := project.Inspect(dir); err == nil {
		t.Fatal("Inspect() error = nil, want error for malformed package.json")
	}
}

func TestInspect_NodeProject_PackageJSONNotRegularFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "package.json"), 0o755); err != nil {
		t.Fatalf("Mkdir() error = %v", err)
	}

	if _, err := project.Inspect(dir); err == nil {
		t.Fatal("Inspect() error = nil, want error for package.json that is not a regular file")
	}
}

func TestInspect_NodePackageManager_Pnpm(t *testing.T) {
	dir := t.TempDir()
	writePackageJSON(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "pnpm-lock.yaml"), []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	insp, err := project.Inspect(dir)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if insp.NodePackageManager != "pnpm" {
		t.Errorf("NodePackageManager = %q, want %q", insp.NodePackageManager, "pnpm")
	}
}

func TestInspect_NodePackageManager_Yarn(t *testing.T) {
	dir := t.TempDir()
	writePackageJSON(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "yarn.lock"), []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	insp, err := project.Inspect(dir)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if insp.NodePackageManager != "yarn" {
		t.Errorf("NodePackageManager = %q, want %q", insp.NodePackageManager, "yarn")
	}
}

func TestInspect_NodePackageManager_Npm(t *testing.T) {
	dir := t.TempDir()
	writePackageJSON(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	insp, err := project.Inspect(dir)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if insp.NodePackageManager != "npm" {
		t.Errorf("NodePackageManager = %q, want %q", insp.NodePackageManager, "npm")
	}
}

func TestInspect_NodePackageManager_LockfileWithoutPackageJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	insp, err := project.Inspect(dir)
	if err != nil {
		t.Fatalf("Inspect() error = %v", err)
	}

	if insp.IsNodeProject {
		t.Error("IsNodeProject = true, want false for a lockfile without package.json")
	}
	if insp.NodePackageManager != "" {
		t.Errorf("NodePackageManager = %q, want empty for a lockfile without package.json", insp.NodePackageManager)
	}
}

func TestInspect_NodePackageManager_MultipleLockfiles(t *testing.T) {
	dir := t.TempDir()
	writePackageJSON(t, dir)
	if err := os.WriteFile(filepath.Join(dir, "yarn.lock"), []byte(""), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "package-lock.json"), []byte("{}"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	if _, err := project.Inspect(dir); err == nil {
		t.Fatal("Inspect() error = nil, want error for multiple lockfiles")
	}
}

func writePackageJSON(t *testing.T, dir string) {
	t.Helper()
	content := `{"name": "example-package"}`
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
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
