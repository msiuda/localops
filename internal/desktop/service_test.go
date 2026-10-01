package desktop

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/msiuda/localops/internal/doctor"
	"github.com/msiuda/localops/internal/environment"
	"github.com/msiuda/localops/internal/overview"
	"github.com/msiuda/localops/internal/project"
	"github.com/msiuda/localops/internal/storage"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	path := filepath.Join(t.TempDir(), "projects.json")
	return NewService(storage.New(path))
}

func cardFor(t *testing.T, cards []ProjectCard, name string) ProjectCard {
	t.Helper()
	for _, c := range cards {
		if c.Name == name {
			return c
		}
	}
	t.Fatalf("no card found for %q in %+v", name, cards)
	return ProjectCard{}
}

func TestGetOverview_Empty(t *testing.T) {
	svc := newTestService(t)

	got, err := svc.GetOverview()
	if err != nil {
		t.Fatalf("GetOverview() error = %v", err)
	}
	if len(got.Projects) != 0 {
		t.Errorf("Projects = %v, want none", got.Projects)
	}
}

func TestGetOverview_UnavailableProject(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")

	path := filepath.Join(t.TempDir(), "projects.json")
	store := storage.New(path)
	if err := store.Save([]project.Project{{Name: "gone", Path: missing}}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	svc := NewService(store)
	got, err := svc.GetOverview()
	if err != nil {
		t.Fatalf("GetOverview() error = %v", err)
	}

	card := cardFor(t, got.Projects, "gone")
	if card.Health != HealthUnavailable {
		t.Errorf("Health = %q, want %q", card.Health, HealthUnavailable)
	}
	if card.UnavailableReason == "" {
		t.Error("UnavailableReason is empty, want an explanation")
	}
	if len(card.Technologies) != 0 {
		t.Errorf("Technologies = %v, want none for an unavailable project", card.Technologies)
	}
}

func TestGetOverview_LoadFailureIsError(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "projects.json")
	if err := os.WriteFile(path, []byte("not valid json"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	svc := NewService(storage.New(path))
	if _, err := svc.GetOverview(); err == nil {
		t.Fatal("GetOverview() error = nil, want an error for corrupted storage")
	}
}

func TestToProjectCard_Healthy(t *testing.T) {
	pr := overview.ProjectResult{
		Project: project.Project{Name: "demo", Path: "/tmp/demo"},
		Health:  overview.HealthHealthy,
		Inspection: project.Inspection{
			IsGitRepository: true,
			HasGoMod:        true,
			IsNodeProject:   true,
		},
		Report: doctor.Report{
			Path: "/tmp/demo",
			Checks: []doctor.CheckResult{
				{Tool: "git", Available: true, OK: true},
				{Tool: "go", Available: true, OK: true},
				{Tool: "npm", Available: true, OK: true},
			},
		},
	}

	card := toProjectCard(pr)

	if card.Name != "demo" || card.Path != "/tmp/demo" {
		t.Errorf("Name/Path = %q/%q, want demo//tmp/demo", card.Name, card.Path)
	}
	if card.Health != HealthHealthy {
		t.Errorf("Health = %q, want %q", card.Health, HealthHealthy)
	}
	wantTechs := []string{"Git", "Go", "Node.js"}
	if len(card.Technologies) != len(wantTechs) {
		t.Fatalf("Technologies = %v, want %v", card.Technologies, wantTechs)
	}
	for i, tech := range wantTechs {
		if card.Technologies[i] != tech {
			t.Errorf("Technologies[%d] = %q, want %q", i, card.Technologies[i], tech)
		}
	}
	if card.PackageManager != "npm" {
		t.Errorf("PackageManager = %q, want npm", card.PackageManager)
	}
	if card.IssueCount != 0 {
		t.Errorf("IssueCount = %d, want 0", card.IssueCount)
	}
	if card.UnavailableReason != "" {
		t.Errorf("UnavailableReason = %q, want empty", card.UnavailableReason)
	}
}

func TestToProjectCard_Issues(t *testing.T) {
	pr := overview.ProjectResult{
		Project:    project.Project{Name: "demo", Path: "/tmp/demo"},
		Health:     overview.HealthIssues,
		Inspection: project.Inspection{HasGoMod: true},
		Report: doctor.Report{
			Checks: []doctor.CheckResult{
				{Tool: "go", Available: true, OK: true},
				{Tool: "git", Available: false, OK: false, Detail: "git not found on PATH"},
			},
		},
	}

	card := toProjectCard(pr)

	if card.Health != HealthIssues {
		t.Errorf("Health = %q, want %q", card.Health, HealthIssues)
	}
	if card.IssueCount != 1 {
		t.Errorf("IssueCount = %d, want 1", card.IssueCount)
	}
	if len(card.Findings) != 1 || card.Findings[0].Tool != "git" || card.Findings[0].Detail != "git not found on PATH" {
		t.Errorf("Findings = %+v, want a single git finding", card.Findings)
	}
}

func TestToProjectCard_Unavailable(t *testing.T) {
	pr := overview.ProjectResult{
		Project: project.Project{Name: "gone", Path: "/tmp/gone"},
		Health:  overview.HealthUnavailable,
		Err:     errors.New("path does not exist"),
	}

	card := toProjectCard(pr)

	if card.Health != HealthUnavailable {
		t.Errorf("Health = %q, want %q", card.Health, HealthUnavailable)
	}
	if card.UnavailableReason != "path does not exist" {
		t.Errorf("UnavailableReason = %q, want %q", card.UnavailableReason, "path does not exist")
	}
	if len(card.Technologies) != 0 {
		t.Errorf("Technologies = %v, want none", card.Technologies)
	}
}

func TestGetProjectDetail_NotRegistered(t *testing.T) {
	svc := newTestService(t)

	if _, err := svc.GetProjectDetail("/not/registered"); err == nil {
		t.Fatal("GetProjectDetail() error = nil, want an error for an unregistered path")
	}
}

func TestGetProjectDetail_Registered(t *testing.T) {
	// An empty directory: no .git, go.mod, or package.json, so Inspect
	// detects no technologies and Doctor runs zero checks (no real
	// executable is ever invoked) and Environment finds no contract files.
	dir := t.TempDir()

	path := filepath.Join(t.TempDir(), "projects.json")
	store := storage.New(path)
	if err := store.Save([]project.Project{{Name: filepath.Base(dir), Path: dir}}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	svc := NewService(store)
	detail, err := svc.GetProjectDetail(dir)
	if err != nil {
		t.Fatalf("GetProjectDetail() error = %v", err)
	}

	if detail.Health != HealthHealthy {
		t.Errorf("Health = %q, want %q", detail.Health, HealthHealthy)
	}
	if len(detail.DoctorChecks) != 0 {
		t.Errorf("DoctorChecks = %v, want none", detail.DoctorChecks)
	}
	if detail.Environment.HasContract {
		t.Error("Environment.HasContract = true, want false")
	}
	if detail.Environment.Error != "" {
		t.Errorf("Environment.Error = %q, want empty", detail.Environment.Error)
	}
}

func TestToProjectDetail_Unavailable(t *testing.T) {
	pr := overview.ProjectResult{
		Project: project.Project{Name: "gone", Path: "/tmp/gone"},
		Health:  overview.HealthUnavailable,
		Err:     errors.New("path does not exist"),
	}

	detail := toProjectDetail(pr)

	if detail.Health != HealthUnavailable {
		t.Errorf("Health = %q, want %q", detail.Health, HealthUnavailable)
	}
	if detail.UnavailableReason != "path does not exist" {
		t.Errorf("UnavailableReason = %q, want %q", detail.UnavailableReason, "path does not exist")
	}
	if detail.Environment.HasContract || detail.Environment.Error != "" {
		t.Errorf("Environment = %+v, want zero value (Environment must not be analyzed for an unavailable project)", detail.Environment)
	}
}

func TestToProjectDetail_WithEnvironmentContract(t *testing.T) {
	dir := t.TempDir()
	writeFile := func(name, contents string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", name, err)
		}
	}
	writeFile(".env.example", "FOO=\nBAZ=\n")
	writeFile(".env.local", "FOO=1\n")

	pr := overview.ProjectResult{
		Project: project.Project{Name: "demo", Path: dir},
		Health:  overview.HealthHealthy,
		Report: doctor.Report{
			Checks: []doctor.CheckResult{
				{Tool: "git", Available: true, OK: true, Version: "2.40.0"},
			},
		},
	}

	detail := toProjectDetail(pr)

	if detail.Environment.Error != "" {
		t.Fatalf("Environment.Error = %q, want empty", detail.Environment.Error)
	}
	if !detail.Environment.HasContract {
		t.Fatal("Environment.HasContract = false, want true")
	}
	if len(detail.Environment.ContractSources) != 1 || detail.Environment.ContractSources[0].VariableCount != 2 {
		t.Errorf("ContractSources = %+v, want one source declaring 2 variables", detail.Environment.ContractSources)
	}
	if detail.Environment.MissingCount != 1 {
		t.Errorf("MissingCount = %d, want 1 (only BAZ missing)", detail.Environment.MissingCount)
	}
	if len(detail.DoctorChecks) != 1 || detail.DoctorChecks[0].Tool != "git" || detail.DoctorChecks[0].Version != "2.40.0" {
		t.Errorf("DoctorChecks = %+v, want the single git check", detail.DoctorChecks)
	}
}

func TestToEnvironmentSummary_AnalysisError(t *testing.T) {
	summary := toEnvironmentSummary(environment.Result{}, errors.New("boom"))

	if summary.Error != "boom" {
		t.Errorf("Error = %q, want %q", summary.Error, "boom")
	}
	if summary.HasContract || len(summary.Variables) != 0 {
		t.Errorf("summary = %+v, want every other field left zero", summary)
	}
}

func TestAddProject(t *testing.T) {
	svc := newTestService(t)
	projectDir := t.TempDir()

	if err := svc.AddProject(projectDir); err != nil {
		t.Fatalf("AddProject() error = %v", err)
	}

	overview, err := svc.GetOverview()
	if err != nil {
		t.Fatalf("GetOverview() error = %v", err)
	}
	if len(overview.Projects) != 1 {
		t.Fatalf("Projects = %v, want the newly added project", overview.Projects)
	}
}

func TestAddProject_Duplicate(t *testing.T) {
	svc := newTestService(t)
	projectDir := t.TempDir()

	if err := svc.AddProject(projectDir); err != nil {
		t.Fatalf("first AddProject() error = %v", err)
	}
	if err := svc.AddProject(projectDir); err == nil {
		t.Fatal("second AddProject() error = nil, want an error for a duplicate path")
	}
}

func TestAddProject_NonexistentPath(t *testing.T) {
	svc := newTestService(t)

	if err := svc.AddProject(filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("AddProject() error = nil, want an error for a nonexistent path")
	}
}
