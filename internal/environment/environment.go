// Package environment analyzes a project's Environment Contract: which
// environment variables it declares it expects, which of those are
// currently satisfied by a local env file or the process environment, and
// which environment variables the project's own source code statically
// and recognizably uses.
//
// This package never represents, prints, persists, or otherwise exposes an
// environment variable's value. Only variable names, source filenames, and
// presence/absence information are ever handled.
package environment

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/msiuda/localops/internal/project"
	"github.com/msiuda/localops/internal/technology"
)

// ContractSource describes a discovered contract template file and how many
// distinct variable names it declares.
type ContractSource struct {
	File          string
	VariableCount int
}

// LocalSource describes a discovered local environment file and how many
// distinct variable names it declares.
type LocalSource struct {
	File          string
	VariableCount int
}

// VariableStatus is the Environment outcome for a single environment
// variable name — one that is declared by the Environment Contract,
// statically used by the project's source code, or both. It never carries
// the variable's value.
//
// Declared and Used are explicit facts rather than something a caller
// must infer from slice lengths, so the three independent correlation
// facts the Environment model exists to answer — declared, used,
// satisfied — are always available directly:
//
//   - Used + Declared + Satisfied: healthy.
//   - Used + Declared + !Satisfied: the declared variable is not locally
//     satisfied.
//   - Used + !Declared: the source uses a variable the contract never
//     declared.
//   - !Used + Declared: no supported static usage was found for a
//     declared variable — not proof that it is actually unused.
type VariableStatus struct {
	Name string
	// Declared reports whether any contract source declares Name.
	Declared bool
	// DeclaredIn lists the contract source file(s) that declare Name, in a
	// fixed, deterministic order. Empty when Declared is false.
	DeclaredIn []string
	// Used reports whether any supported static source-code usage of Name
	// was found.
	Used bool
	// UsedIn lists every detected usage location, ordered by file then
	// line. Empty when Used is false.
	UsedIn []Usage
	// Satisfied and Source are only ever meaningful when Declared is
	// true: local sources and the process environment are never consulted
	// for a variable the contract does not declare.
	Satisfied bool
	// Source is where Name was found to be satisfied: ".env.local", ".env",
	// or "process environment". It is empty when Satisfied is false.
	Source string
}

// Finding is a structured parsing issue in a contract or local source file.
// It never includes the offending line's contents, since that could expose
// a value.
type Finding struct {
	Source string
	Line   int
	Detail string
}

// Result is the Environment Contract outcome for a single project. It never
// contains any environment variable's value.
type Result struct {
	Path            string
	ContractSources []ContractSource
	LocalSources    []LocalSource
	Variables       []VariableStatus
	Findings        []Finding
}

// readFileFunc mirrors os.ReadFile's signature, allowing tests to
// substitute file reading instead of depending on the real filesystem.
type readFileFunc func(name string) ([]byte, error)

// lookupEnvFunc reports whether key is set in the process environment. It
// deliberately never returns the value itself, so a value can never flow
// into the Environment result even by accident.
type lookupEnvFunc func(key string) bool

// Analyze inspects the project at path for its Environment Contract: which
// environment variables it declares it expects (via .env.example,
// .env.sample, or .env.template), which of those are currently satisfied
// by a local env file (.env.local, .env) or the process environment, and
// which environment variables the project's source code statically and
// recognizably uses (supported Go and JavaScript/TypeScript forms only).
//
// Source-usage scanning runs even when no Environment Contract exists, so
// a project with no contract can still report "used but undeclared"
// variables; it never causes local env files or the process environment
// to be read when nothing is declared.
//
// An error is returned only when the project path itself cannot be
// resolved, or when a discovered contract/local source file cannot safely
// be read (for example, a permission error). A malformed line within a
// file that was read, or a source file that could not be scanned, is a
// Finding in the returned Result, not an error.
func Analyze(path string) (Result, error) {
	return analyze(path, os.ReadFile, lookupProcessEnv)
}

// analyze implements Analyze, taking file reading and process-environment
// lookup as small function seams so tests can substitute them instead of
// depending on the real filesystem or the developer's real environment.
func analyze(path string, readFile readFileFunc, lookupEnv lookupEnvFunc) (Result, error) {
	proj, err := project.FromPath(path)
	if err != nil {
		return Result{}, err
	}

	// contractFileNames and localFileNames are kept local rather than
	// package-level state; localFileNames' order is also the deterministic
	// precedence used to resolve and display which source satisfies a
	// variable when more than one could (.env.local before .env).
	contractFileNames := []string{".env.example", ".env.sample", ".env.template"}
	localFileNames := []string{".env.local", ".env"}

	var contractSources []ContractSource
	var findings []Finding
	declaredIn := make(map[string][]string)

	for _, name := range contractFileNames {
		keys, fileFindings, existed, err := readEnvFile(proj.Path, name, readFile)
		if err != nil {
			return Result{}, err
		}
		if !existed {
			continue
		}

		contractSources = append(contractSources, ContractSource{File: name, VariableCount: len(keys)})
		findings = append(findings, fileFindings...)
		for _, key := range keys {
			declaredIn[key] = append(declaredIn[key], name)
		}
	}

	techResult, err := technology.Detect(proj.Path)
	if err != nil {
		return Result{}, err
	}
	detectedTech := make(map[technology.ID]bool, len(techResult.Detected))
	for _, d := range techResult.Detected {
		detectedTech[d.ID] = true
	}

	usedIn, usageFindings, err := scanSourceUsages(proj.Path, readFile, detectedTech)
	if err != nil {
		return Result{}, err
	}
	findings = append(findings, usageFindings...)

	var localSources []LocalSource
	localKeys := make(map[string]map[string]bool)

	// Local env files and the process environment are only ever consulted
	// when at least one variable is declared. This preserves the existing
	// guarantee that a project with no Environment Contract never reads
	// .env/.env.local or queries the process environment, even when
	// source-usage scanning did find statically used variables — both for
	// correctness and to avoid unnecessary access to files that may carry
	// secrets.
	if len(declaredIn) > 0 {
		for _, name := range localFileNames {
			keys, fileFindings, existed, err := readEnvFile(proj.Path, name, readFile)
			if err != nil {
				return Result{}, err
			}
			if !existed {
				continue
			}

			localSources = append(localSources, LocalSource{File: name, VariableCount: len(keys)})
			findings = append(findings, fileFindings...)

			set := make(map[string]bool, len(keys))
			for _, key := range keys {
				set[key] = true
			}
			localKeys[name] = set
		}
	}

	nameSet := make(map[string]bool, len(declaredIn)+len(usedIn))
	for name := range declaredIn {
		nameSet[name] = true
	}
	for name := range usedIn {
		nameSet[name] = true
	}

	names := make([]string, 0, len(nameSet))
	for name := range nameSet {
		names = append(names, name)
	}
	sort.Strings(names)

	variables := make([]VariableStatus, 0, len(names))
	for _, name := range names {
		declared := len(declaredIn[name]) > 0

		var satisfied bool
		var source string
		if declared {
			satisfied, source = resolveSource(name, localFileNames, localKeys, lookupEnv)
		}

		variables = append(variables, VariableStatus{
			Name:       name,
			Declared:   declared,
			DeclaredIn: append([]string(nil), declaredIn[name]...),
			Used:       len(usedIn[name]) > 0,
			UsedIn:     append([]Usage(nil), usedIn[name]...),
			Satisfied:  satisfied,
			Source:     source,
		})
	}

	return Result{
		Path:            proj.Path,
		ContractSources: contractSources,
		LocalSources:    localSources,
		Variables:       variables,
		Findings:        findings,
	}, nil
}

// resolveSource determines whether name is satisfied, and by which source.
// Local env files are checked in localFileNames order (.env.local before
// .env) before falling back to the process environment, so the displayed
// source is deterministic even when a key is present in more than one.
func resolveSource(name string, localFileNames []string, localKeys map[string]map[string]bool, lookupEnv lookupEnvFunc) (satisfied bool, source string) {
	for _, fileName := range localFileNames {
		if localKeys[fileName][name] {
			return true, fileName
		}
	}
	if lookupEnv(name) {
		return true, "process environment"
	}
	return false, ""
}

// readEnvFile reads and parses the env-style file dir/name, if it exists.
func readEnvFile(dir, name string, readFile readFileFunc) (keys []string, findings []Finding, existed bool, err error) {
	data, err := readFile(filepath.Join(dir, name))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, false, nil
		}
		return nil, nil, false, fmt.Errorf("read %s: %w", name, err)
	}

	keys, findings, err = parseEnvKeys(name, data)
	if err != nil {
		return nil, nil, false, err
	}
	return keys, findings, true, nil
}

// maxEnvLineBytes bounds how long a single line in an env-style file may be
// before parsing fails explicitly. It is a fixed limit, not a configurable
// option, well above any realistic assignment line.
const maxEnvLineBytes = 1 << 20 // 1 MiB

// parseEnvKeys extracts the declared variable names from an env-style
// file's contents.
//
// Only the assignment key is ever identified; the value half of a line is
// never interpreted, returned, or retained. A malformed non-empty,
// non-comment line becomes a Finding identifying source and the line
// number, without including the line's contents, since that could expose a
// value. A scanning failure (for example, a line exceeding maxEnvLineBytes)
// becomes an explicit error naming only the source file, never returning a
// silently partial contract.
func parseEnvKeys(source string, data []byte) (keys []string, findings []Finding, err error) {
	seen := make(map[string]bool)

	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 0, 64*1024), maxEnvLineBytes)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		line = trimExportPrefix(line)

		idx := strings.Index(line, "=")
		if idx < 0 {
			findings = append(findings, Finding{Source: source, Line: lineNum, Detail: "could not be parsed"})
			continue
		}

		key := strings.TrimSpace(line[:idx])
		if !isValidEnvKey(key) {
			findings = append(findings, Finding{Source: source, Line: lineNum, Detail: "could not be parsed"})
			continue
		}

		if !seen[key] {
			seen[key] = true
			keys = append(keys, key)
		}
	}

	if scanErr := scanner.Err(); scanErr != nil {
		return nil, nil, fmt.Errorf("%s: could not be parsed: %w", source, scanErr)
	}

	return keys, findings, nil
}

// isValidEnvKey reports whether key is a conservative valid env-file
// assignment key: [A-Za-z_][A-Za-z0-9_]*.
func isValidEnvKey(key string) bool {
	if key == "" {
		return false
	}

	first := key[0]
	if !isEnvKeyLetter(first) {
		return false
	}

	for i := 1; i < len(key); i++ {
		c := key[i]
		if !isEnvKeyLetter(c) && !(c >= '0' && c <= '9') {
			return false
		}
	}

	return true
}

// isEnvKeyLetter reports whether c may appear as the first character (or
// any non-digit character) of a valid env-file assignment key.
func isEnvKeyLetter(c byte) bool {
	return c == '_' || (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
}

// trimExportPrefix removes a leading "export" keyword (followed by
// whitespace) from an env-file assignment line, if present.
func trimExportPrefix(line string) string {
	const prefix = "export"
	if !strings.HasPrefix(line, prefix) {
		return line
	}
	rest := line[len(prefix):]
	if rest == "" || (rest[0] != ' ' && rest[0] != '\t') {
		return line
	}
	return strings.TrimLeft(rest, " \t")
}

// lookupProcessEnv reports whether key is set in the process environment,
// without ever exposing its value.
func lookupProcessEnv(key string) bool {
	_, ok := os.LookupEnv(key)
	return ok
}
