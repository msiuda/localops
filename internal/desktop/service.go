// Package desktop adapts existing LocalOps core capabilities (storage,
// project inspection, Doctor, and Overview) for the desktop frontend.
//
// It is a thin presentation boundary: it never duplicates storage,
// inspection, Doctor, Overview, Validation, or Environment logic, and it
// never exposes secret or environment-variable values to the frontend.
package desktop

import (
	"fmt"
	"sort"

	"github.com/wailsapp/wails/v3/pkg/application"

	"github.com/msiuda/localops/internal/doctor"
	"github.com/msiuda/localops/internal/environment"
	"github.com/msiuda/localops/internal/overview"
	"github.com/msiuda/localops/internal/project"
	"github.com/msiuda/localops/internal/storage"
	"github.com/msiuda/localops/internal/technology"
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
	Name   string `json:"name"`
	Path   string `json:"path"`
	Health Health `json:"health"`
	// Technologies is the compact technology summary (see
	// compactTechnologySummary): capped, priority-ordered, Git excluded.
	// MoreTechnologies holds every name beyond that cap, for a frontend
	// "+N" overflow badge's tooltip — never a second badge row.
	Technologies      []string  `json:"technologies"`
	MoreTechnologies  []string  `json:"moreTechnologies"`
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

// EnvUsage is a single UI-facing source-code location where a variable was
// statically detected as used. It never carries the variable's value.
type EnvUsage struct {
	File string `json:"file"`
	Line int    `json:"line"`
}

// EnvVariable is the UI-facing status of a single environment variable —
// one that is declared by the Environment Contract, statically used by
// the project's source code, or both. It is structurally incapable of
// carrying a value: every field is a name, a source filename/line, or a
// boolean/status.
type EnvVariable struct {
	Name       string     `json:"name"`
	Declared   bool       `json:"declared"`
	DeclaredIn []string   `json:"declaredIn"`
	Used       bool       `json:"used"`
	UsedIn     []EnvUsage `json:"usedIn"`
	Satisfied  bool       `json:"satisfied"`
	Source     string     `json:"source"`
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
	// MissingCount counts only declared variables that are not locally
	// satisfied. UndeclaredCount counts only used variables the contract
	// never declared. They are deliberately separate: an undeclared usage
	// must never be presented as a missing declared variable.
	MissingCount    int    `json:"missingCount"`
	UndeclaredCount int    `json:"undeclaredCount"`
	Error           string `json:"error"`
}

// ProjectDetail is the UI-facing detail view of a single registered
// project: its Overview health, full Doctor report, and Environment
// Contract analysis. It never carries a Validation preview or plan:
// Validation is not runnable from the desktop app yet, and LocalOps does
// not duplicate internal/validation's decisions in this package — when
// desktop Validation execution is added, it will compose the real
// internal/validation package instead.
type ProjectDetail struct {
	Name              string `json:"name"`
	Path              string `json:"path"`
	Health            Health `json:"health"`
	UnavailableReason string `json:"unavailableReason"`
	// Technologies/MoreTechnologies are the same compact summary as
	// ProjectCard (see compactTechnologySummary), for the header.
	// TechnologyGroups is the separate, fuller grouped picture for
	// Overview's "Stack" section — the two are deliberately not the same
	// amount of information.
	Technologies     []string           `json:"technologies"`
	MoreTechnologies []string           `json:"moreTechnologies"`
	TechnologyGroups []TechnologyGroup  `json:"technologyGroups"`
	PackageManager   string             `json:"packageManager"`
	IssueCount       int                `json:"issueCount"`
	DoctorChecks     []DoctorCheck      `json:"doctorChecks"`
	Environment      EnvironmentSummary `json:"environment"`
}

// TechnologyGroup is one Technology Intelligence kind (e.g. "Languages",
// "Frameworks") and the detected technology names within it, for Project
// Detail's Overview "Stack" section. It never carries detection evidence —
// only display names — since that section shows the fuller grouped
// picture, not an evidence audit.
type TechnologyGroup struct {
	Label string   `json:"label"`
	Names []string `json:"names"`
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

	detail.Technologies, detail.MoreTechnologies = compactTechnologySummary(pr.Inspection)
	detail.TechnologyGroups = technologyGroups(pr.Inspection)
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
		var usedIn []EnvUsage
		for _, u := range v.UsedIn {
			usedIn = append(usedIn, EnvUsage{File: u.File, Line: u.Line})
		}

		summary.Variables = append(summary.Variables, EnvVariable{
			Name:       v.Name,
			Declared:   v.Declared,
			DeclaredIn: v.DeclaredIn,
			Used:       v.Used,
			UsedIn:     usedIn,
			Satisfied:  v.Satisfied,
			Source:     v.Source,
		})
		if v.Declared && !v.Satisfied {
			summary.MissingCount++
		}
		if v.Used && !v.Declared {
			summary.UndeclaredCount++
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

	card.Technologies, card.MoreTechnologies = compactTechnologySummary(pr.Inspection)
	card.PackageManager = effectivePackageManager(pr.Report)

	for _, check := range pr.Report.Checks {
		if !check.OK {
			card.IssueCount++
			card.Findings = append(card.Findings, Finding{Tool: check.Tool, Detail: check.Detail})
		}
	}

	return card
}

// maxCompactTechnologies caps the compact technology summary (ProjectRow,
// Project Detail header) at a fixed, small number of badges, so a project
// with many detected technologies never grows a row/header vertically.
// Anything beyond the cap is summarized by the caller as a single "+N"
// overflow, never a second badge row.
const maxCompactTechnologies = 3

// compactTechnologySummaryOrder ranks kinds for the compact summary: a
// primary language identifies the project fastest, a strongly-detected
// framework is the next most useful fact, and runtime/platform is
// supporting context — the opposite priority from technologyGroups, which
// exists to show the fuller picture, not a quick-scan identity. Git is
// deliberately absent: it is near-universal repository metadata, not a
// Technology Intelligence fact, and was never meant to compete with a
// project's actual stack for one of three badge slots.
var compactTechnologySummaryOrder = map[technology.Kind]int{
	technology.KindLanguage:  0,
	technology.KindFramework: 1,
	technology.KindRuntime:   2,
	technology.KindPlatform:  2,
}

// compactFrameworkPrecedence maps a framework ID to a more-characteristic
// framework ID that, when also detected in the same project, better
// identifies it for the compact summary — Next.js over the React it is
// built on, NestJS over the Express it wraps. This is presentation-only:
// internal/technology still reports every framework it found evidence
// for, and both technologyGroups and the full Detect result are
// unaffected — only the compact summary's selection is filtered.
var compactFrameworkPrecedence = map[technology.ID]technology.ID{
	technology.IDReact:   technology.IDNextJS,
	technology.IDExpress: technology.IDNestJS,
}

// compactTechnologySummary builds the single, backend-owned compact
// technology summary shared by ProjectCard and ProjectDetail — the only
// place technology priority/capping/precedence logic lives, so
// ProjectsPage, ProjectDetailPage, and Overview never each derive their
// own. names is capped to maxCompactTechnologies, in priority order; more
// holds every remaining detected name (for a frontend "+N" badge's
// tooltip), also in priority order, so the full picture is never silently
// discarded — it is simply not the compact view's job to show it
// (Overview's technologyGroups is).
func compactTechnologySummary(insp project.Inspection) (names []string, more []string) {
	detected := insp.Technologies.Detected

	present := make(map[technology.ID]bool, len(detected))
	for _, d := range detected {
		present[d.ID] = true
	}

	filtered := make([]technology.Detected, 0, len(detected))
	for _, d := range detected {
		if winner, ok := compactFrameworkPrecedence[d.ID]; ok && present[winner] {
			continue
		}
		filtered = append(filtered, d)
	}

	sort.SliceStable(filtered, func(i, j int) bool {
		return compactTechnologySummaryOrder[filtered[i].Kind] < compactTechnologySummaryOrder[filtered[j].Kind]
	})

	for i, d := range filtered {
		if i < maxCompactTechnologies {
			names = append(names, d.Name)
		} else {
			more = append(more, d.Name)
		}
	}
	return names, more
}

// technologyGroups builds the fuller, grouped Technology Intelligence
// picture (languages, runtime/platform, frameworks) for Project Detail's
// Overview "Stack" section. Each group is omitted when empty, so a project
// with no detected frameworks never shows an empty "Frameworks" group.
func technologyGroups(insp project.Inspection) []TechnologyGroup {
	kinds := []struct {
		label string
		kind  technology.Kind
	}{
		{"Languages", technology.KindLanguage},
		{"Runtime", technology.KindRuntime},
		{"Platform", technology.KindPlatform},
		{"Frameworks", technology.KindFramework},
	}

	var groups []TechnologyGroup
	for _, k := range kinds {
		var names []string
		for _, d := range insp.Technologies.Detected {
			if d.Kind == k.kind {
				names = append(names, d.Name)
			}
		}
		if len(names) == 0 {
			continue
		}
		groups = append(groups, TechnologyGroup{Label: k.label, Names: names})
	}
	return groups
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
