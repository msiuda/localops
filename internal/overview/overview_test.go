package overview

import (
	"errors"
	"reflect"
	"testing"

	"github.com/msiuda/localops/internal/doctor"
	"github.com/msiuda/localops/internal/project"
)

// fakeInspect returns an inspectFunc that resolves canned inspection
// results and errors by project path.
func fakeInspect(results map[string]project.Inspection, errs map[string]error) inspectFunc {
	return func(path string) (project.Inspection, error) {
		if err, ok := errs[path]; ok {
			return project.Inspection{}, err
		}
		return results[path], nil
	}
}

// fakeRunDoctor returns a runDoctorFunc that resolves canned Doctor reports
// by inspection path.
func fakeRunDoctor(reports map[string]doctor.Report) runDoctorFunc {
	return func(insp project.Inspection) doctor.Report {
		return reports[insp.Path]
	}
}

func resultFor(t *testing.T, result Result, name string) ProjectResult {
	t.Helper()
	for _, pr := range result.Projects {
		if pr.Project.Name == name {
			return pr
		}
	}
	t.Fatalf("no result found for project %q in %+v", name, result.Projects)
	return ProjectResult{}
}

func TestBuild_Empty(t *testing.T) {
	result := build(nil, fakeInspect(nil, nil), fakeRunDoctor(nil))

	if len(result.Projects) != 0 {
		t.Errorf("Projects = %v, want none", result.Projects)
	}
}

func TestBuild_HealthyProject(t *testing.T) {
	p := project.Project{Name: "healthy", Path: "/projects/healthy"}
	insp := project.Inspection{Path: p.Path, IsGitRepository: true}
	report := doctor.Report{
		Path: p.Path,
		Checks: []doctor.CheckResult{
			{Tool: "git", Available: true, Version: "git version 2.50.1", OK: true},
		},
	}

	result := build(
		[]project.Project{p},
		fakeInspect(map[string]project.Inspection{p.Path: insp}, nil),
		fakeRunDoctor(map[string]doctor.Report{p.Path: report}),
	)

	got := resultFor(t, result, "healthy")
	if got.Health != HealthHealthy {
		t.Errorf("Health = %q, want %q", got.Health, HealthHealthy)
	}
	if got.Err != nil {
		t.Errorf("Err = %v, want nil", got.Err)
	}
}

func TestBuild_HealthyProjectWithNoChecks(t *testing.T) {
	p := project.Project{Name: "plain", Path: "/projects/plain"}
	insp := project.Inspection{Path: p.Path}
	report := doctor.Report{Path: p.Path}

	result := build(
		[]project.Project{p},
		fakeInspect(map[string]project.Inspection{p.Path: insp}, nil),
		fakeRunDoctor(map[string]doctor.Report{p.Path: report}),
	)

	got := resultFor(t, result, "plain")
	if got.Health != HealthHealthy {
		t.Errorf("Health = %q, want %q for a project with no required checks", got.Health, HealthHealthy)
	}
}

func TestBuild_ProjectWithIssues(t *testing.T) {
	p := project.Project{Name: "issues", Path: "/projects/issues"}
	insp := project.Inspection{Path: p.Path, IsNodeProject: true, NodePackageManager: "yarn"}
	report := doctor.Report{
		Path: p.Path,
		Checks: []doctor.CheckResult{
			{Tool: "node", Available: true, Version: "v22.14.0", OK: true},
			{Tool: "yarn", Available: false, Detail: "executable not found"},
		},
	}

	result := build(
		[]project.Project{p},
		fakeInspect(map[string]project.Inspection{p.Path: insp}, nil),
		fakeRunDoctor(map[string]doctor.Report{p.Path: report}),
	)

	got := resultFor(t, result, "issues")
	if got.Health != HealthIssues {
		t.Errorf("Health = %q, want %q", got.Health, HealthIssues)
	}
	if len(got.Report.Checks) != 2 {
		t.Errorf("Report.Checks = %v, want it preserved from Doctor", got.Report.Checks)
	}
}

func TestBuild_UnavailableProject(t *testing.T) {
	p := project.Project{Name: "gone", Path: "/projects/gone"}
	wantErr := errors.New("path does not exist")

	result := build(
		[]project.Project{p},
		fakeInspect(nil, map[string]error{p.Path: wantErr}),
		fakeRunDoctor(nil),
	)

	got := resultFor(t, result, "gone")
	if got.Health != HealthUnavailable {
		t.Errorf("Health = %q, want %q", got.Health, HealthUnavailable)
	}
	if got.Err == nil {
		t.Error("Err = nil, want the inspection error to be preserved")
	}
	if !reflect.DeepEqual(got.Inspection, project.Inspection{}) {
		t.Errorf("Inspection = %+v, want zero value for an unavailable project", got.Inspection)
	}
}

func TestBuild_OneUnavailableProjectDoesNotStopOthers(t *testing.T) {
	healthy := project.Project{Name: "healthy", Path: "/projects/healthy"}
	broken := project.Project{Name: "broken", Path: "/projects/broken"}
	issues := project.Project{Name: "issues", Path: "/projects/issues"}

	result := build(
		[]project.Project{healthy, broken, issues},
		fakeInspect(
			map[string]project.Inspection{
				healthy.Path: {Path: healthy.Path, IsGitRepository: true},
				issues.Path:  {Path: issues.Path, HasGoMod: true},
			},
			map[string]error{broken.Path: errors.New("no such file or directory")},
		),
		fakeRunDoctor(map[string]doctor.Report{
			healthy.Path: {Path: healthy.Path, Checks: []doctor.CheckResult{{Tool: "git", OK: true}}},
			issues.Path:  {Path: issues.Path, Checks: []doctor.CheckResult{{Tool: "go", OK: false, Detail: "executable not found"}}},
		}),
	)

	if len(result.Projects) != 3 {
		t.Fatalf("Projects = %v, want exactly three results", result.Projects)
	}
	if got := resultFor(t, result, "healthy"); got.Health != HealthHealthy {
		t.Errorf("healthy Health = %q, want %q", got.Health, HealthHealthy)
	}
	if got := resultFor(t, result, "broken"); got.Health != HealthUnavailable {
		t.Errorf("broken Health = %q, want %q", got.Health, HealthUnavailable)
	}
	if got := resultFor(t, result, "issues"); got.Health != HealthIssues {
		t.Errorf("issues Health = %q, want %q", got.Health, HealthIssues)
	}
}

func TestBuild_PreservesOrder(t *testing.T) {
	first := project.Project{Name: "first", Path: "/projects/first"}
	second := project.Project{Name: "second", Path: "/projects/second"}
	third := project.Project{Name: "third", Path: "/projects/third"}

	result := build(
		[]project.Project{first, second, third},
		fakeInspect(map[string]project.Inspection{
			first.Path:  {Path: first.Path},
			second.Path: {Path: second.Path},
			third.Path:  {Path: third.Path},
		}, nil),
		fakeRunDoctor(nil),
	)

	if len(result.Projects) != 3 {
		t.Fatalf("Projects = %v, want exactly three results", result.Projects)
	}
	wantOrder := []string{"first", "second", "third"}
	for i, want := range wantOrder {
		if got := result.Projects[i].Project.Name; got != want {
			t.Errorf("Projects[%d].Project.Name = %q, want %q", i, got, want)
		}
	}
}
