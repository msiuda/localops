package environment

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func checkUsageVariable(t *testing.T, vars []VariableStatus, name string) VariableStatus {
	t.Helper()
	for _, v := range vars {
		if v.Name == name {
			return v
		}
	}
	t.Fatalf("no variable found for %q in %+v", name, vars)
	return VariableStatus{}
}

func mustMkdir(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, name), 0o755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", name, err)
	}
}

// --- usageScanner dispatch ---

func TestScannerFor(t *testing.T) {
	tests := []struct {
		path string
		want usageScanner
	}{
		{path: "src/app.ts", want: javascriptUsageScanner{}},
		{path: "src/app.go", want: goUsageScanner{}},
		{path: "README.md", want: nil},
	}

	for _, tt := range tests {
		got := scannerFor(tt.path)
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("scannerFor(%q) = %#v, want %#v", tt.path, got, tt.want)
		}
	}
}

// --- scanSourceUsages: traversal and built-in scanner dispatch ---

func TestScanSourceUsages_MultipleFiles(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, dir, "src")
	writeFile(t, dir, "src/db.ts", `export const url = process.env.DATABASE_URL;`)
	writeFile(t, filepath.Join(dir, "src"), "server.go", "package main\nimport \"os\"\nfunc main() { os.Getenv(\"PORT\") }\n")

	usages, findings, err := scanSourceUsages(dir, os.ReadFile, nil)
	if err != nil {
		t.Fatalf("scanSourceUsages() error = %v", err)
	}
	if len(findings) != 0 {
		t.Errorf("findings = %v, want none", findings)
	}

	dbURL, ok := usages["DATABASE_URL"]
	if !ok || len(dbURL) != 1 || dbURL[0].File != "src/db.ts" || dbURL[0].Line != 1 {
		t.Errorf("usages[DATABASE_URL] = %+v, want one usage at src/db.ts:1", dbURL)
	}
	port, ok := usages["PORT"]
	if !ok || len(port) != 1 || port[0].File != "src/server.go" || port[0].Line != 3 {
		t.Errorf("usages[PORT] = %+v, want one usage at src/server.go:3", port)
	}
}

func TestScanSourceUsages_IgnoresDependencyAndBuildDirectories(t *testing.T) {
	dir := t.TempDir()
	for _, ignored := range []string{".git", "node_modules", "vendor", "dist", "build", "coverage", ".next", "target"} {
		mustMkdir(t, dir, ignored)
		writeFile(t, filepath.Join(dir, ignored), "evil.ts", `process.env.SHOULD_NOT_APPEAR;`)
	}
	writeFile(t, dir, "real.ts", `process.env.SHOULD_APPEAR;`)

	usages, _, err := scanSourceUsages(dir, os.ReadFile, nil)
	if err != nil {
		t.Fatalf("scanSourceUsages() error = %v", err)
	}
	if _, found := usages["SHOULD_NOT_APPEAR"]; found {
		t.Error("usages contains a variable only referenced inside an ignored directory")
	}
	if _, found := usages["SHOULD_APPEAR"]; !found {
		t.Error("usages missing a variable referenced outside any ignored directory")
	}
}

func TestScanSourceUsages_IgnoresUnsupportedExtensions(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "README.md", `process.env.SHOULD_NOT_APPEAR`)
	writeFile(t, dir, "notes.txt", `process.env.SHOULD_NOT_APPEAR`)

	usages, _, err := scanSourceUsages(dir, os.ReadFile, nil)
	if err != nil {
		t.Fatalf("scanSourceUsages() error = %v", err)
	}
	if len(usages) != 0 {
		t.Errorf("usages = %v, want none for unsupported extensions", usages)
	}
}

func TestScanSourceUsages_DoesNotFollowSymlinks(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "real.ts", `process.env.REAL_ONLY;`)

	outsideDir := t.TempDir()
	writeFile(t, outsideDir, "outside.ts", `process.env.SHOULD_NOT_APPEAR;`)

	if err := os.Symlink(filepath.Join(outsideDir, "outside.ts"), filepath.Join(dir, "linked.ts")); err != nil {
		t.Skipf("symlink not supported in this environment: %v", err)
	}

	usages, _, err := scanSourceUsages(dir, os.ReadFile, nil)
	if err != nil {
		t.Fatalf("scanSourceUsages() error = %v", err)
	}
	if _, found := usages["SHOULD_NOT_APPEAR"]; found {
		t.Error("usages followed a symlinked file")
	}
	if _, found := usages["REAL_ONLY"]; !found {
		t.Error("usages missing the real, non-symlinked file's usage")
	}
}

func TestScanSourceUsages_OversizedFileIsAFindingNotAUsage(t *testing.T) {
	dir := t.TempDir()
	oversized := "process.env.FOO;" + strings.Repeat(" ", maxSourceFileBytes+1)
	writeFile(t, dir, "huge.ts", oversized)

	usages, findings, err := scanSourceUsages(dir, os.ReadFile, nil)
	if err != nil {
		t.Fatalf("scanSourceUsages() error = %v", err)
	}
	if _, found := usages["FOO"]; found {
		t.Error("usages contains a variable from a file that exceeds the scan size limit")
	}
	if len(findings) != 1 || findings[0].Source != "huge.ts" {
		t.Errorf("findings = %+v, want exactly one finding for huge.ts", findings)
	}
}

func TestScanSourceUsages_UnreadableFileIsAFindingNotAFailure(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "bad.ts", `process.env.FOO;`)
	writeFile(t, dir, "good.ts", `process.env.BAR;`)

	readFile := func(name string) ([]byte, error) {
		if filepath.Base(name) == "bad.ts" {
			return nil, errors.New("permission denied")
		}
		return os.ReadFile(name)
	}

	usages, findings, err := scanSourceUsages(dir, readFile, nil)
	if err != nil {
		t.Fatalf("scanSourceUsages() error = %v, want the scan to isolate the unreadable file", err)
	}
	if _, found := usages["BAR"]; !found {
		t.Error("usages missing the readable file's usage; one unreadable file should not hide the rest")
	}
	if len(findings) != 1 || findings[0].Source != "bad.ts" {
		t.Errorf("findings = %+v, want exactly one finding for bad.ts", findings)
	}
}

func TestScanSourceUsages_UnparseableGoFileIsAFindingNotAFailure(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "broken.go", "package main\nfunc main( {\n")
	writeFile(t, dir, "good.go", "package main\nimport \"os\"\nfunc main() { os.Getenv(\"BAR\") }\n")

	usages, findings, err := scanSourceUsages(dir, os.ReadFile, nil)
	if err != nil {
		t.Fatalf("scanSourceUsages() error = %v", err)
	}
	if _, found := usages["BAR"]; !found {
		t.Error("usages missing the valid file's usage")
	}
	if len(findings) != 1 || findings[0].Source != "broken.go" {
		t.Errorf("findings = %+v, want exactly one finding for broken.go", findings)
	}
}

// --- Correlation: Analyze integrating usage with the Environment Contract ---

func TestAnalyze_UsedDeclaredSatisfied(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "DATABASE_URL=x\n")
	writeFile(t, dir, ".env", "DATABASE_URL=postgres://real\n")
	writeFile(t, dir, "db.ts", `process.env.DATABASE_URL;`)

	result, err := analyze(dir, os.ReadFile, fakeLookupEnv(nil))
	if err != nil {
		t.Fatalf("analyze() error = %v", err)
	}

	v := checkUsageVariable(t, result.Variables, "DATABASE_URL")
	if !v.Declared || !v.Used || !v.Satisfied || v.Source != ".env" {
		t.Errorf("DATABASE_URL = %+v, want declared+used+satisfied via .env", v)
	}
	if len(v.UsedIn) != 1 || v.UsedIn[0].File != "db.ts" || v.UsedIn[0].Line != 1 {
		t.Errorf("DATABASE_URL.UsedIn = %+v, want one usage at db.ts:1", v.UsedIn)
	}
}

func TestAnalyze_UsedDeclaredMissing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "JWT_SECRET=x\n")
	writeFile(t, dir, "auth.ts", `process.env.JWT_SECRET;`)

	result, err := analyze(dir, os.ReadFile, fakeLookupEnv(nil))
	if err != nil {
		t.Fatalf("analyze() error = %v", err)
	}

	v := checkUsageVariable(t, result.Variables, "JWT_SECRET")
	if !v.Declared || !v.Used || v.Satisfied {
		t.Errorf("JWT_SECRET = %+v, want declared+used+missing", v)
	}
	if v.Source != "" {
		t.Errorf("JWT_SECRET.Source = %q, want empty", v.Source)
	}
}

func TestAnalyze_UsedUndeclared(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "DATABASE_URL=x\n")
	writeFile(t, dir, "stripe.ts", `process.env.STRIPE_SECRET_KEY;`)

	result, err := analyze(dir, os.ReadFile, fakeLookupEnv(nil))
	if err != nil {
		t.Fatalf("analyze() error = %v", err)
	}

	v := checkUsageVariable(t, result.Variables, "STRIPE_SECRET_KEY")
	if v.Declared {
		t.Error("STRIPE_SECRET_KEY.Declared = true, want false")
	}
	if !v.Used {
		t.Error("STRIPE_SECRET_KEY.Used = false, want true")
	}
	if v.Satisfied || v.Source != "" {
		t.Errorf("STRIPE_SECRET_KEY = %+v, want Satisfied=false and Source empty: satisfaction is never evaluated for an undeclared variable", v)
	}
	if len(v.DeclaredIn) != 0 {
		t.Errorf("STRIPE_SECRET_KEY.DeclaredIn = %v, want none", v.DeclaredIn)
	}
}

func TestAnalyze_DeclaredNoSupportedUsageFound(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "LEGACY_FEATURE=x\n")

	result, err := analyze(dir, os.ReadFile, fakeLookupEnv(nil))
	if err != nil {
		t.Fatalf("analyze() error = %v", err)
	}

	v := checkUsageVariable(t, result.Variables, "LEGACY_FEATURE")
	if !v.Declared || v.Used {
		t.Errorf("LEGACY_FEATURE = %+v, want declared and not used", v)
	}
	if len(v.UsedIn) != 0 {
		t.Errorf("LEGACY_FEATURE.UsedIn = %v, want none", v.UsedIn)
	}
}

func TestAnalyze_NoContractButUsageDetected(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "db.ts", `process.env.DATABASE_URL;`)

	result, err := analyze(dir, os.ReadFile, fakeLookupEnv(nil))
	if err != nil {
		t.Fatalf("analyze() error = %v", err)
	}

	if len(result.ContractSources) != 0 {
		t.Errorf("ContractSources = %v, want none", result.ContractSources)
	}
	v := checkUsageVariable(t, result.Variables, "DATABASE_URL")
	if v.Declared {
		t.Error("DATABASE_URL.Declared = true, want false")
	}
	if !v.Used {
		t.Error("DATABASE_URL.Used = false, want true")
	}
}

func TestAnalyze_NoContractUsageDoesNotReadLocalFilesOrProcessEnv(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "db.ts", `process.env.DATABASE_URL;`)

	var readCalls []string
	recordingRead := func(name string) ([]byte, error) {
		readCalls = append(readCalls, filepath.Base(name))
		return os.ReadFile(name)
	}

	called := false
	lookupEnv := func(key string) bool {
		called = true
		return false
	}

	if _, err := analyze(dir, recordingRead, lookupEnv); err != nil {
		t.Fatalf("analyze() error = %v", err)
	}
	if called {
		t.Error("process environment was queried even though no contract exists")
	}
	for _, name := range readCalls {
		if name == ".env" || name == ".env.local" {
			t.Errorf("local file %q was read even though no contract exists", name)
		}
	}
}

func TestAnalyze_ContractPresentNoUsages(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "A=x\n")

	result, err := analyze(dir, os.ReadFile, fakeLookupEnv(nil))
	if err != nil {
		t.Fatalf("analyze() error = %v", err)
	}
	if len(result.Variables) != 1 {
		t.Fatalf("Variables = %v, want exactly one (A)", result.Variables)
	}
	if result.Variables[0].Used {
		t.Error("A.Used = true, want false")
	}
}

func TestAnalyze_DeterministicOrderingAcrossDeclaredAndUsed(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "ZEBRA=x\nMANGO=x\n")
	writeFile(t, dir, "app.ts", "process.env.APPLE;\nprocess.env.MANGO;\n")

	result, err := analyze(dir, os.ReadFile, fakeLookupEnv(nil))
	if err != nil {
		t.Fatalf("analyze() error = %v", err)
	}

	var names []string
	for _, v := range result.Variables {
		names = append(names, v.Name)
	}
	want := []string{"APPLE", "MANGO", "ZEBRA"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("variable order = %v, want %v", names, want)
	}
}

func TestAnalyze_LaravelEnvFormOnlyRecognizedWhenLaravelDetected(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "composer.json", `{"require":{"laravel/framework":"^10.0"}}`)
	writeFile(t, dir, "database.php", `$v = env("DATABASE_URL");`)

	result, err := analyze(dir, os.ReadFile, fakeLookupEnv(nil))
	if err != nil {
		t.Fatalf("analyze() error = %v", err)
	}

	v := checkUsageVariable(t, result.Variables, "DATABASE_URL")
	if !v.Used {
		t.Errorf("DATABASE_URL = %+v, want Used=true since Laravel is detected", v)
	}
}

func TestAnalyze_BarePHPEnvCallNotRecognizedWithoutLaravel(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "composer.json", `{"require":{"some/other-package":"^1.0"}}`)
	writeFile(t, dir, "database.php", `$v = env("DATABASE_URL");`)

	result, err := analyze(dir, os.ReadFile, fakeLookupEnv(nil))
	if err != nil {
		t.Fatalf("analyze() error = %v", err)
	}

	for _, v := range result.Variables {
		if v.Name == "DATABASE_URL" {
			t.Errorf("DATABASE_URL unexpectedly reported as used without Laravel detected: %+v", v)
		}
	}
}

// --- Security: usage scanning never carries values ---

func TestAnalyze_UsageScanNeverExposesSecretValues(t *testing.T) {
	const secret = "SUPER_SECRET_VALUE_SHOULD_NEVER_APPEAR"
	dir := t.TempDir()
	writeFile(t, dir, ".env", "API_KEY="+secret+"\n")
	writeFile(t, dir, "client.ts", `process.env.API_KEY;`)

	result, err := analyze(dir, os.ReadFile, fakeLookupEnv(nil))
	if err != nil {
		t.Fatalf("analyze() error = %v", err)
	}

	dump := fmt.Sprintf("%#v", result)
	if strings.Contains(dump, secret) {
		t.Fatalf("Result contains the secret value:\n%s", dump)
	}
}
