// Package environment analyzes a project's Environment Contract: which
// environment variables it declares it expects, and which of those are
// currently satisfied by a local env file or the process environment.
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

// VariableStatus is the Environment Contract outcome for a single declared
// environment variable. It never carries the variable's value.
type VariableStatus struct {
	Name string
	// DeclaredIn lists the contract source file(s) that declare Name, in a
	// fixed, deterministic order.
	DeclaredIn []string
	Satisfied  bool
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
// .env.sample, or .env.template), and which of those are currently
// satisfied by a local env file (.env.local, .env) or the process
// environment.
//
// An error is returned only when the project path itself cannot be
// resolved, or when a discovered source file cannot safely be read (for
// example, a permission error). A malformed line within a file that was
// read is a Finding in the returned Result, not an error.
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

	// With no declared contract, there is nothing to check local sources
	// or the process environment for. Returning here means a project with
	// no contract never reads .env/.env.local or queries the process
	// environment at all, which also avoids any unnecessary access to
	// files that may carry secrets.
	if len(contractSources) == 0 {
		return Result{Path: proj.Path}, nil
	}

	var localSources []LocalSource
	localKeys := make(map[string]map[string]bool)

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

	names := make([]string, 0, len(declaredIn))
	for name := range declaredIn {
		names = append(names, name)
	}
	sort.Strings(names)

	variables := make([]VariableStatus, 0, len(names))
	for _, name := range names {
		satisfied, source := resolveSource(name, localFileNames, localKeys, lookupEnv)

		variables = append(variables, VariableStatus{
			Name:       name,
			DeclaredIn: append([]string(nil), declaredIn[name]...),
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
