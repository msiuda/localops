// Package desktop adapts existing LocalOps core capabilities (storage,
// project inspection, Doctor, and Overview) for the desktop frontend.
//
// It is a thin presentation boundary: it never duplicates storage,
// inspection, Doctor, Overview, Validation, or Environment logic, and it
// never exposes secret or environment-variable values to the frontend.
package desktop

import (
	"fmt"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/msiuda/localops/internal/doctor"
	"github.com/msiuda/localops/internal/environment"
	"github.com/msiuda/localops/internal/overview"
	"github.com/msiuda/localops/internal/project"
	"github.com/msiuda/localops/internal/storage"
)

// Health is the UI-facing project health state, mirroring overview.Health.
type Health string

const (
	HealthHealthy     Health = "healthy"
	HealthIssues      Health = "issues"
	HealthUnavailable Health = "unavailable"
)

// Finding is a concise, UI-facing description of a single failed Doctor
// check.
type Finding struct {
	Tool   string `json:"tool"`
	Detail string `json:"detail"`
}

// ProjectCard is the UI-facing summary of a single registered project. It
// is a deliberately small DTO: only what a project card needs to render,
// never raw internal inspection/Doctor structures, and never any
// environment-variable value.
type ProjectCard struct {
	Name              string    `json:"name"`
	Path              string    `json:"path"`
	Health            Health    `json:"health"`
	Technologies      []string  `json:"technologies"`
	PackageManager    string    `json:"packageManager"`
	IssueCount        int       `json:"issueCount"`
	Findings          []Finding `json:"findings"`
	UnavailableReason string    `json:"unavailableReason"`
}

// ProjectsOverview is the UI-facing result of loading every registered
// project's Overview.
type ProjectsOverview struct {
	Projects []ProjectCard `json:"projects"`
}

// DoctorCheck is a single UI-facing Doctor check result, including passing
// checks (unlike Finding, which is only used for a card's concise failed-
// check preview).
type DoctorCheck struct {
	Tool      string `json:"tool"`
	Available bool   `json:"available"`
	Version   string `json:"version"`
	OK        bool   `json:"ok"`
	Note      string `json:"note"`
	Detail    string `json:"detail"`
}

// EnvContractSource describes one discovered Environment Contract source
// file. It never includes variable names or values, only a count.
type EnvContractSource struct {
	File          string `json:"file"`
	VariableCount int    `json:"variableCount"`
}

// EnvLocalSource describes one discovered local environment file. It never
// includes variable names or values, only a count.
type EnvLocalSource struct {
	File          string `json:"file"`
	VariableCount int    `json:"variableCount"`
}

// EnvVariable is the UI-facing status of a single declared environment
// variable. It is structurally incapable of carrying a value: every field
// is a name, a source filename, or a boolean/status.
type EnvVariable struct {
	Name       string   `json:"name"`
	DeclaredIn []string `json:"declaredIn"`
	Satisfied  bool     `json:"satisfied"`
	Source     string   `json:"source"`
}

// EnvFinding is a UI-facing Environment parsing finding: a source file and
// an optional line number, never the offending line's contents.
type EnvFinding struct {
	Source string `json:"source"`
	Line   int    `json:"line"`
	Detail string `json:"detail"`
}

// EnvironmentSummary is the UI-facing Environment Contract analysis for a
// single project. Like internal/environment.Result, it is structurally
// incapable of containing any environment variable's value.
//
// Error is set, with every other field left at its zero value, when
// Environment analysis itself failed (for example, a source file could not
// be read) — this is kept separate from the rest of ProjectDetail so an
// Environment failure does not hide otherwise-available project/Doctor
// information.
type EnvironmentSummary struct {
	HasContract     bool                `json:"hasContract"`
	ContractSources []EnvContractSource `json:"contractSources"`
	LocalSources    []EnvLocalSource    `json:"localSources"`
	Variables       []EnvVariable       `json:"variables"`
	Findings        []EnvFinding        `json:"findings"`
	MissingCount    int                 `json:"missingCount"`
	Error           string              `json:"error"`
}

// ProjectDetail is the UI-facing detail view of a single registered
// project: its Overview health, full Doctor report, and Environment
// Contract analysis. It never carries a Validation preview or plan:
// Validation is not runnable from the desktop app yet, and LocalOps does
// not duplicate internal/validation's decisions in this package — when
// desktop Validation execution is added, it will compose the real
// internal/validation package instead.
type ProjectDetail struct {
	Name              string             `json:"name"`
	Path              string             `json:"path"`
	Health            Health             `json:"health"`
	UnavailableReason string             `json:"unavailableReason"`
	Technologies      []string           `json:"technologies"`
	PackageManager    string             `json:"packageManager"`
	IssueCount        int                `json:"issueCount"`
	DoctorChecks      []DoctorCheck      `json:"doctorChecks"`
	Environment       EnvironmentSummary `json:"environment"`
}

// Service is the Wails-facing desktop service. Its methods are bound to the
// frontend via generated TypeScript bindings.
type Service struct {
	store *storage.Store
}

// NewService creates a Service backed by store.
func NewService(store *storage.Store) *Service {
	return &Service{store: store}
}

// GetOverview loads every registered project (from the same storage the
// CLI uses) and returns a UI-facing summary of each project's Overview
// health.
func (s *Service) GetOverview() (ProjectsOverview, error) {
	projects, err := s.store.Load()
	if err != nil {
		return ProjectsOverview{}, fmt.Errorf("load registered projects: %w", err)
	}

	result := overview.Build(projects)

	cards := make([]ProjectCard, 0, len(result.Projects))
	for _, pr := range result.Projects {
		cards = append(cards, toProjectCard(pr))
	}

	return ProjectsOverview{Projects: cards}, nil
}

// AddProject registers path as a new LocalOps project, using the same
// shared storage.Store.AddProject operation the CLI's "project add" uses,
// so desktop and CLI registration behave identically.
func (s *Service) AddProject(path string) error {
	_, err := s.store.AddProject(path)
	return err
}

// PickProjectDirectory opens the native OS directory picker and returns the
// chosen directory's path. It returns an empty string with a nil error if
// the user cancels the picker; the caller must treat that as "do nothing,"
// not as a failure.
func (s *Service) PickProjectDirectory() (string, error) {
	return application.Get().Dialog.OpenFile().
		SetTitle("Select a Project Directory").
		CanChooseFiles(false).
		CanChooseDirectories(true).
		PromptForSingleSelection()
}

// GetProjectDetail loads the detail view for a single registered project,
// identified by its registered path. The path must belong to a project
// already in storage; GetProjectDetail never inspects an arbitrary
// unregistered path.
func (s *Service) GetProjectDetail(path string) (ProjectDetail, error) {
	projects, err := s.store.Load()
	if err != nil {
		return ProjectDetail{}, fmt.Errorf("load registered projects: %w", err)
	}

	proj, ok := findProjectByPath(projects, path)
	if !ok {
		return ProjectDetail{}, fmt.Errorf("project %q is not registered", path)
	}

	result := overview.Build([]project.Project{proj})
	return toProjectDetail(result.Projects[0]), nil
}

// findProjectByPath returns the registered project whose path exactly
// matches path, if any.
func findProjectByPath(projects []project.Project, path string) (project.Project, bool) {
	for _, p := range projects {
		if p.Path == path {
			return p, true
		}
	}
	return project.Project{}, false
}

// toProjectDetail converts a core overview.ProjectResult, plus a fresh
// Environment analysis, into the small UI-facing ProjectDetail DTO. An
// Environment analysis failure is carried in Environment.Error rather than
// failing the whole detail, so the rest of the project's information
// remains available.
func toProjectDetail(pr overview.ProjectResult) ProjectDetail {
	detail := ProjectDetail{
		Name:   pr.Project.Name,
		Path:   pr.Project.Path,
		Health: Health(pr.Health),
	}

	if pr.Health == overview.HealthUnavailable {
		if pr.Err != nil {
			detail.UnavailableReason = pr.Err.Error()
		}
		return detail
	}

	detail.Technologies = technologies(pr.Inspection)
	detail.PackageManager = effectivePackageManager(pr.Report)

	for _, check := range pr.Report.Checks {
		detail.DoctorChecks = append(detail.DoctorChecks, toDoctorCheck(check))
		if !check.OK {
			detail.IssueCount++
		}
	}

	envResult, err := environment.Analyze(pr.Project.Path)
	detail.Environment = toEnvironmentSummary(envResult, err)

	return detail
}

// toDoctorCheck converts a core doctor.CheckResult into the UI-facing
// DoctorCheck DTO.
func toDoctorCheck(cr doctor.CheckResult) DoctorCheck {
	return DoctorCheck{
		Tool:      cr.Tool,
		Available: cr.Available,
		Version:   cr.Version,
		OK:        cr.OK,
		Note:      cr.Note,
		Detail:    cr.Detail,
	}
}

// toEnvironmentSummary converts a core environment.Result into the small
// UI-facing EnvironmentSummary DTO. It never copies a value: environment.
// Result itself has no field capable of holding one.
func toEnvironmentSummary(res environment.Result, err error) EnvironmentSummary {
	if err != nil {
		return EnvironmentSummary{Error: err.Error()}
	}

	summary := EnvironmentSummary{HasContract: len(res.ContractSources) > 0}

	for _, cs := range res.ContractSources {
		summary.ContractSources = append(summary.ContractSources, EnvContractSource{
			File:          cs.File,
			VariableCount: cs.VariableCount,
		})
	}
	for _, ls := range res.LocalSources {
		summary.LocalSources = append(summary.LocalSources, EnvLocalSource{
			File:          ls.File,
			VariableCount: ls.VariableCount,
		})
	}
	for _, v := range res.Variables {
		summary.Variables = append(summary.Variables, EnvVariable{
			Name:       v.Name,
			DeclaredIn: v.DeclaredIn,
			Satisfied:  v.Satisfied,
			Source:     v.Source,
		})
		if !v.Satisfied {
			summary.MissingCount++
		}
	}
	for _, f := range res.Findings {
		summary.Findings = append(summary.Findings, EnvFinding{
			Source: f.Source,
			Line:   f.Line,
			Detail: f.Detail,
		})
	}

	return summary
}

// toProjectCard converts a core overview.ProjectResult into the small
// UI-facing ProjectCard DTO.
func toProjectCard(pr overview.ProjectResult) ProjectCard {
	card := ProjectCard{
		Name:   pr.Project.Name,
		Path:   pr.Project.Path,
		Health: Health(pr.Health),
	}

	if pr.Health == overview.HealthUnavailable {
		if pr.Err != nil {
			card.UnavailableReason = pr.Err.Error()
		}
		return card
	}

	card.Technologies = technologies(pr.Inspection)
	card.PackageManager = effectivePackageManager(pr.Report)

	for _, check := range pr.Report.Checks {
		if !check.OK {
			card.IssueCount++
			card.Findings = append(card.Findings, Finding{Tool: check.Tool, Detail: check.Detail})
		}
	}

	return card
}

// technologies builds a short list of the technologies insp detected.
func technologies(insp project.Inspection) []string {
	var techs []string
	if insp.IsGitRepository {
		techs = append(techs, "Git")
	}
	if insp.HasGoMod {
		techs = append(techs, "Go")
	}
	if insp.IsNodeProject {
		techs = append(techs, "Node.js")
	}
	return techs
}

// effectivePackageManager returns the package manager Doctor actually
// evaluated, derived from report's checks rather than duplicating Doctor's
// own declared/lockfile reconciliation logic.
func effectivePackageManager(report doctor.Report) string {
	for _, check := range report.Checks {
		switch check.Tool {
		case "npm", "yarn", "pnpm":
			return check.Tool
		}
	}
	return ""
}
