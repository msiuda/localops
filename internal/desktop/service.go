// Package desktop adapts existing LocalOps core capabilities (storage,
// project inspection, Doctor, and Overview) for the desktop frontend.
//
// It is a thin presentation boundary: it never duplicates storage,
// inspection, Doctor, Overview, Validation, or Environment logic, and it
// never exposes secret or environment-variable values to the frontend.
package desktop

import (
	"fmt"

	"github.com/msiuda/localops/internal/doctor"
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
