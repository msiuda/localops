// Package overview combines project registration, project inspection, and
// Doctor into a single structured view of every registered project's
// health.
package overview

import (
	"github.com/msiuda/localops/internal/doctor"
	"github.com/msiuda/localops/internal/project"
)

// Health is the overall state Overview assigns to a single project.
type Health string

const (
	// HealthHealthy means inspection and Doctor both ran and every Doctor
	// check passed.
	HealthHealthy Health = "healthy"
	// HealthIssues means inspection and Doctor both ran but at least one
	// Doctor check did not pass.
	HealthIssues Health = "issues"
	// HealthUnavailable means the registered project could not be
	// inspected, for example because its path no longer exists.
	HealthUnavailable Health = "unavailable"
)

// ProjectResult is the Overview outcome for a single registered project.
type ProjectResult struct {
	Project    project.Project
	Inspection project.Inspection
	Report     doctor.Report
	Health     Health
	// Err explains why the project is HealthUnavailable. It is nil for
	// every other Health value.
	Err error
}

// Result is the Overview outcome for every registered project.
type Result struct {
	Projects []ProjectResult
}

// inspectFunc mirrors project.Inspect's signature, allowing tests to
// substitute inspection instead of depending on real filesystem contents.
type inspectFunc func(path string) (project.Inspection, error)

// runDoctorFunc mirrors doctor.Run's signature, allowing tests to
// substitute Doctor instead of depending on real installed tools.
type runDoctorFunc func(insp project.Inspection) doctor.Report

// Build inspects and diagnoses every project in projects, in order,
// producing a structured Overview result.
//
// A project that cannot be inspected is reported as unavailable rather
// than aborting the rest; only a failure to load the registered projects
// themselves (handled by the caller) prevents Build from running at all.
func Build(projects []project.Project) Result {
	return build(projects, project.Inspect, doctor.Run)
}

// build implements Build, taking project inspection and Doctor as small
// function seams so tests can substitute them.
func build(projects []project.Project, inspect inspectFunc, runDoctor runDoctorFunc) Result {
	results := make([]ProjectResult, 0, len(projects))
	for _, p := range projects {
		results = append(results, evaluateProject(p, inspect, runDoctor))
	}

	return Result{Projects: results}
}

// evaluateProject inspects and diagnoses a single project.
func evaluateProject(p project.Project, inspect inspectFunc, runDoctor runDoctorFunc) ProjectResult {
	insp, err := inspect(p.Path)
	if err != nil {
		return ProjectResult{
			Project: p,
			Health:  HealthUnavailable,
			Err:     err,
		}
	}

	report := runDoctor(insp)

	health := HealthHealthy
	for _, check := range report.Checks {
		if !check.OK {
			health = HealthIssues
			break
		}
	}

	return ProjectResult{
		Project:    p,
		Inspection: insp,
		Report:     report,
		Health:     health,
	}
}
