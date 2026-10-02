package desktop

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/msiuda/localops/internal/doctor"
	"github.com/msiuda/localops/internal/environment"
	"github.com/msiuda/localops/internal/overview"
	"github.com/msiuda/localops/internal/project"
	"github.com/msiuda/localops/internal/storage"
	"github.com/msiuda/localops/internal/technology"
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
			Technologies: technology.Result{
				Detected: []technology.Detected{
					{ID: technology.IDGo, Name: "Go", Kind: technology.KindLanguage},
					{ID: technology.IDNodeJS, Name: "Node.js", Kind: technology.KindRuntime},
				},
			},
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
	// Git is deliberately excluded from the compact technology summary; it
	// is repository metadata, not a Technology Intelligence fact.
	wantTechs := []string{"Go", "Node.js"}
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

func TestCompactTechnologySummary_PriorityOrderAndCap(t *testing.T) {
	insp := project.Inspection{
		IsGitRepository: true,
		Technologies: technology.Result{
			Detected: []technology.Detected{
				{ID: technology.IDTypeScript, Name: "TypeScript", Kind: technology.KindLanguage},
				{ID: technology.IDNodeJS, Name: "Node.js", Kind: technology.KindRuntime},
				{ID: technology.IDAngular, Name: "Angular", Kind: technology.KindFramework},
				{ID: technology.IDVue, Name: "Vue", Kind: technology.KindFramework},
			},
		},
	}

	names, more := compactTechnologySummary(insp)

	if len(names) != maxCompactTechnologies {
		t.Fatalf("names = %v, want exactly %d (the cap)", names, maxCompactTechnologies)
	}
	// Priority: language, then framework(s), then runtime/platform — not
	// Git (repository metadata, never part of this summary), and not the
	// detector's own language-first/alphabetical Detect() order.
	want := []string{"TypeScript", "Angular", "Vue"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("names = %v, want %v", names, want)
	}
	if !reflect.DeepEqual(more, []string{"Node.js"}) {
		t.Errorf("more = %v, want [Node.js] (bumped past the cap)", more)
	}
}

func TestCompactTechnologySummary_NextJSPreferredOverReact(t *testing.T) {
	insp := project.Inspection{
		Technologies: technology.Result{
			Detected: []technology.Detected{
				{ID: technology.IDTypeScript, Name: "TypeScript", Kind: technology.KindLanguage},
				{ID: technology.IDNodeJS, Name: "Node.js", Kind: technology.KindRuntime},
				{ID: technology.IDReact, Name: "React", Kind: technology.KindFramework},
				{ID: technology.IDNextJS, Name: "Next.js", Kind: technology.KindFramework},
			},
		},
	}

	names, more := compactTechnologySummary(insp)

	want := []string{"TypeScript", "Next.js", "Node.js"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("names = %v, want %v: Next.js is more characteristic than the React it is built on", names, want)
	}
	if len(more) != 0 {
		t.Errorf("more = %v, want none: React was suppressed by precedence, not bumped by the cap", more)
	}
}

func TestCompactTechnologySummary_NestJSPreferredOverExpress(t *testing.T) {
	insp := project.Inspection{
		Technologies: technology.Result{
			Detected: []technology.Detected{
				{ID: technology.IDTypeScript, Name: "TypeScript", Kind: technology.KindLanguage},
				{ID: technology.IDNodeJS, Name: "Node.js", Kind: technology.KindRuntime},
				{ID: technology.IDExpress, Name: "Express", Kind: technology.KindFramework},
				{ID: technology.IDNestJS, Name: "NestJS", Kind: technology.KindFramework},
			},
		},
	}

	names, _ := compactTechnologySummary(insp)

	want := []string{"TypeScript", "NestJS", "Node.js"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("names = %v, want %v: NestJS is more characteristic than the Express it wraps", names, want)
	}
}

func TestCompactTechnologySummary_PrecedenceDoesNotAffectFullDetectionOrGroups(t *testing.T) {
	insp := project.Inspection{
		Technologies: technology.Result{
			Detected: []technology.Detected{
				{ID: technology.IDTypeScript, Name: "TypeScript", Kind: technology.KindLanguage},
				{ID: technology.IDNextJS, Name: "Next.js", Kind: technology.KindFramework},
				{ID: technology.IDReact, Name: "React", Kind: technology.KindFramework},
			},
		},
	}

	// The compact summary filters React out in favor of Next.js...
	names, _ := compactTechnologySummary(insp)
	for _, n := range names {
		if n == "React" {
			t.Errorf("names = %v, want React suppressed in the compact summary", names)
		}
	}

	// ...but the full detection result and the grouped Overview picture
	// are untouched: both frameworks remain.
	if len(insp.Technologies.Detected) != 3 {
		t.Errorf("Technologies.Detected = %+v, want all 3 technologies still present", insp.Technologies.Detected)
	}
	groups := technologyGroups(insp)
	var frameworkNames []string
	for _, g := range groups {
		if g.Label == "Frameworks" {
			frameworkNames = g.Names
		}
	}
	want := []string{"Next.js", "React"}
	if !reflect.DeepEqual(frameworkNames, want) {
		t.Errorf("Frameworks group = %v, want %v (both still shown in the full picture)", frameworkNames, want)
	}
}

func TestCompactTechnologySummary_NoOverflowWhenWithinCap(t *testing.T) {
	insp := project.Inspection{
		Technologies: technology.Result{
			Detected: []technology.Detected{
				{ID: technology.IDGo, Name: "Go", Kind: technology.KindLanguage},
			},
		},
	}

	names, more := compactTechnologySummary(insp)

	if !reflect.DeepEqual(names, []string{"Go"}) {
		t.Errorf("names = %v, want [Go]", names)
	}
	if len(more) != 0 {
		t.Errorf("more = %v, want none", more)
	}
}

func TestCompactTechnologySummary_FrameworkIncludedWhenStronglyDetected(t *testing.T) {
	insp := project.Inspection{
		Technologies: technology.Result{
			Detected: []technology.Detected{
				{ID: technology.IDPHP, Name: "PHP", Kind: technology.KindLanguage},
				{ID: technology.IDLaravel, Name: "Laravel", Kind: technology.KindFramework},
			},
		},
	}

	names, _ := compactTechnologySummary(insp)

	if !reflect.DeepEqual(names, []string{"PHP", "Laravel"}) {
		t.Errorf("names = %v, want [PHP Laravel]", names)
	}
}

func TestTechnologyGroups_GroupedByKindOmittingEmpty(t *testing.T) {
	insp := project.Inspection{
		Technologies: technology.Result{
			Detected: []technology.Detected{
				{ID: technology.IDTypeScript, Name: "TypeScript", Kind: technology.KindLanguage},
				{ID: technology.IDNodeJS, Name: "Node.js", Kind: technology.KindRuntime},
				{ID: technology.IDNestJS, Name: "NestJS", Kind: technology.KindFramework},
			},
		},
	}

	groups := technologyGroups(insp)

	if len(groups) != 3 {
		t.Fatalf("groups = %+v, want 3 (no Platform group since nothing detected there)", groups)
	}
	if groups[0].Label != "Languages" || groups[0].Names[0] != "TypeScript" {
		t.Errorf("groups[0] = %+v, want Languages: [TypeScript]", groups[0])
	}
	if groups[2].Label != "Frameworks" || groups[2].Names[0] != "NestJS" {
		t.Errorf("groups[2] = %+v, want Frameworks: [NestJS]", groups[2])
	}
}

func TestToProjectDetail_EnvironmentSourceUsage(t *testing.T) {
	dir := t.TempDir()
	writeFile := func(name, contents string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(contents), 0o644); err != nil {
			t.Fatalf("WriteFile(%s) error = %v", name, err)
		}
	}
	writeFile(".env.example", "DATABASE_URL=\n")
	writeFile("db.ts", "export const url = process.env.DATABASE_URL;\nconst key = process.env.STRIPE_SECRET_KEY;\n")

	pr := overview.ProjectResult{
		Project: project.Project{Name: "demo", Path: dir},
		Health:  overview.HealthHealthy,
	}

	detail := toProjectDetail(pr)

	if detail.Environment.MissingCount != 1 {
		t.Errorf("MissingCount = %d, want 1 (DATABASE_URL declared but not locally satisfied)", detail.Environment.MissingCount)
	}
	if detail.Environment.UndeclaredCount != 1 {
		t.Errorf("UndeclaredCount = %d, want 1 (STRIPE_SECRET_KEY)", detail.Environment.UndeclaredCount)
	}
}

func TestToEnvironmentSummary_UsedDeclaredAndUndeclared(t *testing.T) {
	res := environment.Result{
		Path: "/projects/demo",
		Variables: []environment.VariableStatus{
			{
				Name:       "DATABASE_URL",
				Declared:   true,
				DeclaredIn: []string{".env.example"},
				Used:       true,
				UsedIn:     []environment.Usage{{File: "src/db.ts", Line: 14}},
				Satisfied:  true,
				Source:     ".env",
			},
			{
				Name:       "JWT_SECRET",
				Declared:   true,
				DeclaredIn: []string{".env.example"},
				Used:       false,
				Satisfied:  true,
				Source:     ".env",
			},
			{
				Name:   "STRIPE_SECRET_KEY",
				Used:   true,
				UsedIn: []environment.Usage{{File: "src/payments.ts", Line: 21}},
			},
		},
	}

	summary := toEnvironmentSummary(res, nil)

	if summary.MissingCount != 0 {
		t.Errorf("MissingCount = %d, want 0: an undeclared usage must never count as a missing declared variable", summary.MissingCount)
	}
	if summary.UndeclaredCount != 1 {
		t.Errorf("UndeclaredCount = %d, want 1", summary.UndeclaredCount)
	}

	var stripe EnvVariable
	for _, v := range summary.Variables {
		if v.Name == "STRIPE_SECRET_KEY" {
			stripe = v
		}
	}
	if stripe.Declared {
		t.Error("STRIPE_SECRET_KEY.Declared = true, want false")
	}
	if !stripe.Used || len(stripe.UsedIn) != 1 || stripe.UsedIn[0].File != "src/payments.ts" || stripe.UsedIn[0].Line != 21 {
		t.Errorf("STRIPE_SECRET_KEY usage = %+v, want one usage at src/payments.ts:21", stripe)
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
