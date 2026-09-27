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

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", name, err)
	}
}

func fakeLookupEnv(present map[string]bool) lookupEnvFunc {
	return func(key string) bool {
		return present[key]
	}
}

func checkVariable(t *testing.T, vars []VariableStatus, name string) VariableStatus {
	t.Helper()
	for _, v := range vars {
		if v.Name == name {
			return v
		}
	}
	t.Fatalf("no variable found for %q in %+v", name, vars)
	return VariableStatus{}
}

func checkContractSource(t *testing.T, sources []ContractSource, file string) ContractSource {
	t.Helper()
	for _, s := range sources {
		if s.File == file {
			return s
		}
	}
	t.Fatalf("no contract source found for %q in %+v", file, sources)
	return ContractSource{}
}

// --- parseEnvKeys ---

func TestParseEnvKeys(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		wantKeys     []string
		wantFindings []Finding
	}{
		{
			name:     "simple assignment",
			input:    "DATABASE_URL=postgres://example\n",
			wantKeys: []string{"DATABASE_URL"},
		},
		{
			name:     "quoted value",
			input:    `JWT_SECRET="abc123"` + "\n",
			wantKeys: []string{"JWT_SECRET"},
		},
		{
			name:     "empty value still counts as present",
			input:    "EMPTY=\n",
			wantKeys: []string{"EMPTY"},
		},
		{
			name:     "export prefix",
			input:    "export REDIS_URL=redis://example\n",
			wantKeys: []string{"REDIS_URL"},
		},
		{
			name:     "blank and comment lines are ignored",
			input:    "\n# a full-line comment\n   \nAPI_URL=https://example\n",
			wantKeys: []string{"API_URL"},
		},
		{
			name:         "missing equals sign is a finding",
			input:        "NOT_AN_ASSIGNMENT\n",
			wantFindings: []Finding{{Source: "src", Line: 1, Detail: "could not be parsed"}},
		},
		{
			name:         "invalid key characters is a finding",
			input:        "123BAD=value\n",
			wantFindings: []Finding{{Source: "src", Line: 1, Detail: "could not be parsed"}},
		},
		{
			name:     "duplicate key within a file is deduplicated",
			input:    "A=1\nA=2\n",
			wantKeys: []string{"A"},
		},
		{
			name:         "finding retains the correct line number",
			input:        "GOOD=1\nnot valid\nALSO_GOOD=2\n",
			wantKeys:     []string{"GOOD", "ALSO_GOOD"},
			wantFindings: []Finding{{Source: "src", Line: 2, Detail: "could not be parsed"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			keys, findings, err := parseEnvKeys("src", []byte(tt.input))
			if err != nil {
				t.Fatalf("parseEnvKeys() error = %v", err)
			}
			if !reflect.DeepEqual(keys, tt.wantKeys) {
				t.Errorf("keys = %v, want %v", keys, tt.wantKeys)
			}
			if !reflect.DeepEqual(findings, tt.wantFindings) {
				t.Errorf("findings = %v, want %v", findings, tt.wantFindings)
			}
		})
	}
}

func TestParseEnvKeys_LineTooLongIsAnExplicitError(t *testing.T) {
	// A single "line" (no newline) far larger than maxEnvLineBytes, so the
	// scanner fails outright rather than silently returning a partial key
	// list.
	oversized := strings.Repeat("A", maxEnvLineBytes+1)

	keys, findings, err := parseEnvKeys("src", []byte(oversized))
	if err == nil {
		t.Fatal("parseEnvKeys() error = nil, want an error for an oversized line")
	}
	if keys != nil || findings != nil {
		t.Errorf("keys = %v, findings = %v, want both nil on scanner failure", keys, findings)
	}
	if !strings.Contains(err.Error(), "src") {
		t.Errorf("error = %q, want it to identify the source file", err.Error())
	}
	if strings.Contains(err.Error(), "AAAA") {
		t.Errorf("error = %q, want it not to include the offending content", err.Error())
	}
}

// --- Analyze: contract sources ---

func TestAnalyze_NoContractFiles(t *testing.T) {
	dir := t.TempDir()

	result, err := analyze(dir, os.ReadFile, fakeLookupEnv(nil))
	if err != nil {
		t.Fatalf("analyze() error = %v", err)
	}

	if len(result.ContractSources) != 0 {
		t.Errorf("ContractSources = %v, want none", result.ContractSources)
	}
	if len(result.Variables) != 0 {
		t.Errorf("Variables = %v, want none", result.Variables)
	}
}

func TestAnalyze_NoContract_DoesNotReadLocalFiles(t *testing.T) {
	dir := t.TempDir()

	var readCalls []string
	recordingRead := func(name string) ([]byte, error) {
		readCalls = append(readCalls, filepath.Base(name))
		return nil, os.ErrNotExist
	}

	result, err := analyze(dir, recordingRead, fakeLookupEnv(nil))
	if err != nil {
		t.Fatalf("analyze() error = %v", err)
	}
	if len(result.ContractSources) != 0 {
		t.Errorf("ContractSources = %v, want none", result.ContractSources)
	}

	want := []string{".env.example", ".env.sample", ".env.template"}
	if !reflect.DeepEqual(readCalls, want) {
		t.Errorf("read calls = %v, want exactly %v (never .env or .env.local)", readCalls, want)
	}
}

func TestAnalyze_NoContract_DoesNotQueryProcessEnv(t *testing.T) {
	dir := t.TempDir()

	called := false
	lookupEnv := func(key string) bool {
		called = true
		return false
	}

	if _, err := analyze(dir, os.ReadFile, lookupEnv); err != nil {
		t.Fatalf("analyze() error = %v", err)
	}
	if called {
		t.Error("process environment was queried even though no contract exists")
	}
}

func TestAnalyze_NoContract_UnreadableEnvFileDoesNotFail(t *testing.T) {
	dir := t.TempDir()

	readFile := func(name string) ([]byte, error) {
		if filepath.Base(name) == ".env" {
			return nil, errors.New("permission denied")
		}
		return nil, os.ErrNotExist
	}

	result, err := analyze(dir, readFile, fakeLookupEnv(nil))
	if err != nil {
		t.Fatalf("analyze() error = %v, want no error since .env is never read without a contract", err)
	}
	if len(result.ContractSources) != 0 {
		t.Errorf("ContractSources = %v, want none", result.ContractSources)
	}
}

func TestAnalyze_EnvExample(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "DATABASE_URL=x\nJWT_SECRET=x\n")

	result, err := analyze(dir, os.ReadFile, fakeLookupEnv(nil))
	if err != nil {
		t.Fatalf("analyze() error = %v", err)
	}

	src := checkContractSource(t, result.ContractSources, ".env.example")
	if src.VariableCount != 2 {
		t.Errorf("VariableCount = %d, want 2", src.VariableCount)
	}
	if len(result.Variables) != 2 {
		t.Fatalf("Variables = %v, want two entries", result.Variables)
	}
}

func TestAnalyze_EnvSample(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.sample", "API_URL=x\n")

	result, err := analyze(dir, os.ReadFile, fakeLookupEnv(nil))
	if err != nil {
		t.Fatalf("analyze() error = %v", err)
	}

	checkContractSource(t, result.ContractSources, ".env.sample")
	checkVariable(t, result.Variables, "API_URL")
}

func TestAnalyze_EnvTemplate(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.template", "AWS_REGION=x\n")

	result, err := analyze(dir, os.ReadFile, fakeLookupEnv(nil))
	if err != nil {
		t.Fatalf("analyze() error = %v", err)
	}

	checkContractSource(t, result.ContractSources, ".env.template")
	checkVariable(t, result.Variables, "AWS_REGION")
}

func TestAnalyze_MultipleContractFilesUnion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "A=x\nB=x\n")
	writeFile(t, dir, ".env.sample", "B=x\nC=x\n")

	result, err := analyze(dir, os.ReadFile, fakeLookupEnv(nil))
	if err != nil {
		t.Fatalf("analyze() error = %v", err)
	}

	if len(result.Variables) != 3 {
		t.Fatalf("Variables = %v, want three entries (A, B, C)", result.Variables)
	}

	b := checkVariable(t, result.Variables, "B")
	want := []string{".env.example", ".env.sample"}
	if !reflect.DeepEqual(b.DeclaredIn, want) {
		t.Errorf("B.DeclaredIn = %v, want %v", b.DeclaredIn, want)
	}
}

func TestAnalyze_DuplicateKeysAcrossSourcesNotAnError(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "SHARED=x\n")
	writeFile(t, dir, ".env.sample", "SHARED=x\n")
	writeFile(t, dir, ".env.template", "SHARED=x\n")

	result, err := analyze(dir, os.ReadFile, fakeLookupEnv(nil))
	if err != nil {
		t.Fatalf("analyze() error = %v", err)
	}
	if len(result.Findings) != 0 {
		t.Errorf("Findings = %v, want none: duplicate declarations are not errors", result.Findings)
	}

	shared := checkVariable(t, result.Variables, "SHARED")
	want := []string{".env.example", ".env.sample", ".env.template"}
	if !reflect.DeepEqual(shared.DeclaredIn, want) {
		t.Errorf("SHARED.DeclaredIn = %v, want %v", shared.DeclaredIn, want)
	}
}

// --- Analyze: satisfaction ---

func TestAnalyze_EnvFileSatisfies(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "DATABASE_URL=x\n")
	writeFile(t, dir, ".env", "DATABASE_URL=postgres://real\n")

	result, err := analyze(dir, os.ReadFile, fakeLookupEnv(nil))
	if err != nil {
		t.Fatalf("analyze() error = %v", err)
	}

	got := checkVariable(t, result.Variables, "DATABASE_URL")
	if !got.Satisfied || got.Source != ".env" {
		t.Errorf("DATABASE_URL = %+v, want satisfied via .env", got)
	}
}

func TestAnalyze_EnvLocalFileSatisfies(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "API_URL=x\n")
	writeFile(t, dir, ".env.local", "API_URL=https://local\n")

	result, err := analyze(dir, os.ReadFile, fakeLookupEnv(nil))
	if err != nil {
		t.Fatalf("analyze() error = %v", err)
	}

	got := checkVariable(t, result.Variables, "API_URL")
	if !got.Satisfied || got.Source != ".env.local" {
		t.Errorf("API_URL = %+v, want satisfied via .env.local", got)
	}
}

func TestAnalyze_ProcessEnvSatisfies(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "HOME=x\n")

	result, err := analyze(dir, os.ReadFile, fakeLookupEnv(map[string]bool{"HOME": true}))
	if err != nil {
		t.Fatalf("analyze() error = %v", err)
	}

	got := checkVariable(t, result.Variables, "HOME")
	if !got.Satisfied || got.Source != "process environment" {
		t.Errorf("HOME = %+v, want satisfied via process environment", got)
	}
}

func TestAnalyze_MissingVariable(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "JWT_SECRET=x\n")

	result, err := analyze(dir, os.ReadFile, fakeLookupEnv(nil))
	if err != nil {
		t.Fatalf("analyze() error = %v", err)
	}

	got := checkVariable(t, result.Variables, "JWT_SECRET")
	if got.Satisfied {
		t.Error("Satisfied = true, want false")
	}
	if got.Source != "" {
		t.Errorf("Source = %q, want empty", got.Source)
	}
}

func TestAnalyze_SourcePrecedenceDeterministic(t *testing.T) {
	t.Run("env local wins over env", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, ".env.example", "KEY=x\n")
		writeFile(t, dir, ".env.local", "KEY=from-local\n")
		writeFile(t, dir, ".env", "KEY=from-env\n")

		result, err := analyze(dir, os.ReadFile, fakeLookupEnv(nil))
		if err != nil {
			t.Fatalf("analyze() error = %v", err)
		}

		got := checkVariable(t, result.Variables, "KEY")
		if got.Source != ".env.local" {
			t.Errorf("Source = %q, want %q", got.Source, ".env.local")
		}
	})

	t.Run("env wins over process environment", func(t *testing.T) {
		dir := t.TempDir()
		writeFile(t, dir, ".env.example", "KEY=x\n")
		writeFile(t, dir, ".env", "KEY=from-env\n")

		result, err := analyze(dir, os.ReadFile, fakeLookupEnv(map[string]bool{"KEY": true}))
		if err != nil {
			t.Fatalf("analyze() error = %v", err)
		}

		got := checkVariable(t, result.Variables, "KEY")
		if got.Source != ".env" {
			t.Errorf("Source = %q, want %q", got.Source, ".env")
		}
	})
}

func TestAnalyze_EmptyValueStillCountsAsPresent(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "EMPTY=x\n")
	writeFile(t, dir, ".env", "EMPTY=\n")

	result, err := analyze(dir, os.ReadFile, fakeLookupEnv(nil))
	if err != nil {
		t.Fatalf("analyze() error = %v", err)
	}

	got := checkVariable(t, result.Variables, "EMPTY")
	if !got.Satisfied || got.Source != ".env" {
		t.Errorf("EMPTY = %+v, want satisfied via .env despite an empty value", got)
	}
}

func TestAnalyze_MalformedLineProducesFinding(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "GOOD=x\nthis line has no equals sign\n")

	result, err := analyze(dir, os.ReadFile, fakeLookupEnv(nil))
	if err != nil {
		t.Fatalf("analyze() error = %v", err)
	}

	if len(result.Findings) != 1 {
		t.Fatalf("Findings = %v, want exactly one", result.Findings)
	}
	f := result.Findings[0]
	if f.Source != ".env.example" || f.Line != 2 || f.Detail != "could not be parsed" {
		t.Errorf("Finding = %+v, want {.env.example 2 could not be parsed}", f)
	}
}

func TestAnalyze_DeterministicAlphabeticalOrder(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "ZEBRA=x\nAPPLE=x\nMANGO=x\n")

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

func TestAnalyze_InvalidProjectPath(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")

	if _, err := analyze(missing, os.ReadFile, fakeLookupEnv(nil)); err == nil {
		t.Fatal("analyze() error = nil, want an error for a missing project path")
	}
}

func TestAnalyze_ReadFileErrorIsCommandError(t *testing.T) {
	dir := t.TempDir()

	failingRead := func(name string) ([]byte, error) {
		if strings.HasSuffix(name, ".env.example") {
			return nil, errors.New("permission denied")
		}
		return nil, os.ErrNotExist
	}

	if _, err := analyze(dir, failingRead, fakeLookupEnv(nil)); err == nil {
		t.Fatal("analyze() error = nil, want an error when a source file cannot be read")
	}
}

// --- Security: values must never appear anywhere ---

func TestAnalyze_SecretValuesNeverAppearInResult(t *testing.T) {
	const secret = "SUPER_SECRET_VALUE_SHOULD_NEVER_APPEAR"
	dir := t.TempDir()
	writeFile(t, dir, ".env.example", "API_KEY=placeholder\n")
	writeFile(t, dir, ".env", "API_KEY="+secret+"\n")

	result, err := analyze(dir, os.ReadFile, fakeLookupEnv(nil))
	if err != nil {
		t.Fatalf("analyze() error = %v", err)
	}

	dump := fmt.Sprintf("%#v", result)
	if strings.Contains(dump, secret) {
		t.Fatalf("Result contains the secret value:\n%s", dump)
	}

	got := checkVariable(t, result.Variables, "API_KEY")
	if !got.Satisfied || got.Source != ".env" {
		t.Errorf("API_KEY = %+v, want satisfied via .env", got)
	}
}

func TestAnalyze_SecretValuesNeverAppearInFindings(t *testing.T) {
	const secret = "SUPER_SECRET_VALUE_SHOULD_NEVER_APPEAR"
	dir := t.TempDir()
	// A malformed line (invalid key characters) whose content happens to
	// carry secret-looking text; the parser must never echo it.
	writeFile(t, dir, ".env.example", "SUPER-SECRET-KEY="+secret+"\n")

	result, err := analyze(dir, os.ReadFile, fakeLookupEnv(nil))
	if err != nil {
		t.Fatalf("analyze() error = %v", err)
	}

	if len(result.Findings) != 1 {
		t.Fatalf("Findings = %v, want exactly one", result.Findings)
	}
	for _, f := range result.Findings {
		if strings.Contains(f.Detail, secret) || strings.Contains(f.Source, secret) {
			t.Errorf("Finding leaked content: %+v", f)
		}
	}

	dump := fmt.Sprintf("%#v", result)
	if strings.Contains(dump, secret) {
		t.Fatalf("Result contains leaked content:\n%s", dump)
	}
}
