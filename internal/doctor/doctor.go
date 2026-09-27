// Package doctor checks whether the local machine has the executables
// required by a project's detected technologies, and whether their
// versions are compatible with what the project declares.
package doctor

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/Masterminds/semver/v3"

	"github.com/msiuda/localops/internal/project"
)

// CheckResult reports the outcome of a single Doctor check: either an
// executable's availability and version, or a package manager consistency
// finding.
type CheckResult struct {
	Tool      string
	Available bool
	Version   string
	OK        bool
	Note      string
	Detail    string
}

// Report is the result of running Doctor checks against a project.
type Report struct {
	Path   string
	Checks []CheckResult
}

// lookPathFunc mirrors exec.LookPath's signature, allowing tests to
// substitute executable lookup instead of depending on the machine's PATH.
type lookPathFunc func(tool string) (string, error)

// runVersionFunc runs a tool's version command and returns its trimmed
// output, allowing tests to substitute command execution instead of running
// real tools.
type runVersionFunc func(tool string, args ...string) (string, error)

// Run checks whether the executables required by the technologies detected
// in insp are available on the local machine, retrieves their versions, and
// diagnoses whether the local Node.js environment (runtime and package
// manager) is compatible with what the project declares.
//
// Only executable lookup and each tool's own version command are ever run;
// no project scripts, builds, installs, or other commands are executed.
func Run(insp project.Inspection) Report {
	return run(insp, exec.LookPath, runVersionCommand)
}

// run implements Run, taking executable lookup and version retrieval as
// small function seams so tests can substitute them instead of depending on
// the machine's PATH or real tool installations.
func run(insp project.Inspection, lookPath lookPathFunc, runVersion runVersionFunc) Report {
	var checks []CheckResult

	if insp.IsGitRepository {
		checks = append(checks, checkTool("git", lookPath, runVersion))
	}
	if insp.HasGoMod {
		checks = append(checks, checkTool("go", lookPath, runVersion))
	}
	if insp.IsNodeProject {
		checks = append(checks, nodeChecks(insp, lookPath, runVersion)...)
	}

	return Report{
		Path:   insp.Path,
		Checks: checks,
	}
}

// nodeChecks runs the Node.js-related Doctor checks: the Node runtime
// itself (with its declared engines.node requirement, if any), consistency
// between package.json's declared package manager and the one detected from
// a lockfile, and the effective package manager's executable and version.
func nodeChecks(insp project.Inspection, lookPath lookPathFunc, runVersion runVersionFunc) []CheckResult {
	var checks []CheckResult

	nodeCheck := checkTool("node", lookPath, runVersion)
	if nodeCheck.OK && insp.NodeEngineNode != "" {
		if ok, msg := versionSatisfiesConstraint(nodeCheck.Version, insp.NodeEngineNode); ok {
			nodeCheck.Note = fmt.Sprintf("requires %s", insp.NodeEngineNode)
		} else {
			nodeCheck.OK = false
			nodeCheck.Detail = msg
		}
	}
	checks = append(checks, nodeCheck)

	manager, finding, hasFinding := resolvePackageManager(insp.NodePackageManager, insp.NodeDeclaredPackageManager)
	if hasFinding {
		checks = append(checks, finding)
	}
	if manager == "" {
		return checks
	}

	pmCheck := checkTool(manager, lookPath, runVersion)
	declaredVersion, hasDeclaredVersion := declaredVersionFor(manager, insp.NodeDeclaredPackageManager)

	switch {
	case pmCheck.OK && manager == "npm":
		pmCheck = applyNpmRequirements(pmCheck, declaredVersion, hasDeclaredVersion, insp.NodeEngineNpm)
	case pmCheck.OK && hasDeclaredVersion:
		if ok, msg := versionMatchesDeclared(pmCheck.Version, manager, declaredVersion); ok {
			pmCheck.Note = fmt.Sprintf("declared %s@%s", manager, declaredVersion)
		} else {
			pmCheck.OK = false
			pmCheck.Detail = msg
		}
	case !pmCheck.OK && hasDeclaredVersion:
		pmCheck.Detail = fmt.Sprintf("%s (project declares %s@%s)", pmCheck.Detail, manager, declaredVersion)
	}

	checks = append(checks, pmCheck)

	return checks
}

// applyNpmRequirements checks npm's installed version against whichever of
// its two possible requirements are declared: an exact version from
// package.json's "packageManager" field, and a range from "engines.npm".
// Both are required to pass when both are declared; the first violated
// requirement becomes the finding.
func applyNpmRequirements(check CheckResult, declaredVersion string, hasDeclaredVersion bool, enginesConstraint string) CheckResult {
	var notes []string

	if hasDeclaredVersion {
		if ok, msg := versionMatchesDeclared(check.Version, "npm", declaredVersion); ok {
			notes = append(notes, fmt.Sprintf("declared npm@%s", declaredVersion))
		} else {
			check.OK = false
			check.Detail = msg
			return check
		}
	}

	if enginesConstraint != "" {
		if ok, msg := versionSatisfiesConstraint(check.Version, enginesConstraint); ok {
			notes = append(notes, fmt.Sprintf("requires %s", enginesConstraint))
		} else {
			check.OK = false
			check.Detail = msg
			return check
		}
	}

	check.Note = strings.Join(notes, ", ")
	return check
}

// resolvePackageManager determines the effective Node.js package manager to
// check, reconciling the manager detected from a lockfile with the one
// declared by package.json's "packageManager" field.
//
// When the declared value is non-empty but unsupported or malformed, that
// is reported as a finding rather than silently ignored, falling back to
// the lockfile-detected manager (if any) as the effective manager so it can
// still be checked.
//
// When both a declared and a lockfile manager are present and disagree, the
// declared value is used as the effective manager, since it reflects an
// explicit developer decision, and a "package manager" mismatch finding is
// returned to surface the inconsistency rather than silently picking one.
func resolvePackageManager(lockfileManager, declaredPackageManager string) (manager string, finding CheckResult, hasFinding bool) {
	if declaredPackageManager != "" {
		if issue := packageManagerDeclarationIssue(declaredPackageManager); issue != "" {
			return lockfileManager, CheckResult{
				Tool:   "package manager",
				Detail: issue,
			}, true
		}
	}

	declaredName, _, declaredOK := parseDeclaredPackageManager(declaredPackageManager)

	if declaredOK && lockfileManager != "" && declaredName != lockfileManager {
		return declaredName, CheckResult{
			Tool:   "package manager",
			Detail: fmt.Sprintf("package.json declares %s but lockfile indicates %s", declaredName, lockfileManager),
		}, true
	}
	if declaredOK {
		return declaredName, CheckResult{}, false
	}
	return lockfileManager, CheckResult{}, false
}

// packageManagerDeclarationIssue returns a Doctor-facing explanation of why
// a non-empty package.json "packageManager" value is not one of the
// currently supported declarations (npm, yarn, or pnpm, each with an exact
// version), or "" if it parses as a supported declaration.
func packageManagerDeclarationIssue(declared string) string {
	name, version, found := strings.Cut(declared, "@")
	if !found || name == "" || version == "" {
		return fmt.Sprintf("invalid packageManager declaration %q", declared)
	}

	switch name {
	case "npm", "yarn", "pnpm":
		return ""
	default:
		return fmt.Sprintf("unsupported declaration %s", declared)
	}
}

// parseDeclaredPackageManager parses a package.json "packageManager" value
// such as "yarn@4.6.0" into its manager name and version. Only the
// currently recognized managers (npm, yarn, pnpm) are supported; any other
// value is reported as not recognized rather than guessed at.
func parseDeclaredPackageManager(declared string) (name, version string, ok bool) {
	name, version, found := strings.Cut(declared, "@")
	if !found || name == "" || version == "" {
		return "", "", false
	}

	switch name {
	case "npm", "yarn", "pnpm":
		return name, version, true
	default:
		return "", "", false
	}
}

// declaredVersionFor returns the version package.json's "packageManager"
// field declares for manager, if it declares one for that specific manager.
func declaredVersionFor(manager, declaredPackageManager string) (string, bool) {
	name, version, ok := parseDeclaredPackageManager(declaredPackageManager)
	if !ok || name != manager {
		return "", false
	}
	return version, true
}

// checkTool reports whether tool is available on PATH and, if so, its
// reported version. A failure to retrieve the version is a finding, not an
// error, and does not prevent checking other tools.
func checkTool(tool string, lookPath lookPathFunc, runVersion runVersionFunc) CheckResult {
	if _, err := lookPath(tool); err != nil {
		return CheckResult{
			Tool:      tool,
			Available: false,
			Detail:    "executable not found",
		}
	}

	version, err := runVersion(tool, versionArgsFor(tool)...)
	if err != nil {
		return CheckResult{
			Tool:      tool,
			Available: true,
			Detail:    fmt.Sprintf("found executable but failed to read version: %v", err),
		}
	}
	if version == "" {
		return CheckResult{
			Tool:      tool,
			Available: true,
			Detail:    "version command returned empty output",
		}
	}

	return CheckResult{
		Tool:      tool,
		Available: true,
		Version:   version,
		OK:        true,
	}
}

// versionArgsFor returns the arguments used to retrieve tool's version.
func versionArgsFor(tool string) []string {
	switch tool {
	case "git":
		return []string{"--version"}
	case "go":
		return []string{"version"}
	case "node", "npm", "pnpm", "yarn":
		return []string{"--version"}
	default:
		return nil
	}
}

// runVersionCommand runs tool with args and returns its trimmed output.
func runVersionCommand(tool string, args ...string) (string, error) {
	out, err := exec.Command(tool, args...).Output()
	if err != nil {
		return "", err
	}
	return trimVersionOutput(out), nil
}

// trimVersionOutput trims a version command's raw output down to a single,
// display-ready string.
func trimVersionOutput(out []byte) string {
	return strings.TrimSpace(string(out))
}

// versionSatisfiesConstraint reports whether installedVersion (as reported
// by a tool's version command) satisfies constraint, using npm/Node-style
// semantic version constraint syntax. ok is false, with a human-readable
// message, when either value cannot be interpreted as expected.
func versionSatisfiesConstraint(installedVersion, constraint string) (ok bool, message string) {
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return false, fmt.Sprintf("invalid version requirement %q: %v", constraint, err)
	}

	v, err := semver.NewVersion(normalizeVersionForComparison(installedVersion))
	if err != nil {
		return false, fmt.Sprintf("installed version %q could not be parsed: %v", installedVersion, err)
	}

	if !c.Check(v) {
		return false, fmt.Sprintf("%s does not satisfy %s", installedVersion, constraint)
	}

	return true, ""
}

// versionMatchesDeclared reports whether installedVersion exactly matches
// the version package.json's "packageManager" field declares for manager.
func versionMatchesDeclared(installedVersion, manager, declaredVersion string) (ok bool, message string) {
	installed, err := semver.NewVersion(normalizeVersionForComparison(installedVersion))
	if err != nil {
		return false, fmt.Sprintf("installed version %q could not be parsed: %v", installedVersion, err)
	}

	declared, err := semver.NewVersion(normalizeVersionForComparison(declaredVersion))
	if err != nil {
		return false, fmt.Sprintf("declared version %q could not be parsed: %v", declaredVersion, err)
	}

	if !installed.Equal(declared) {
		return false, fmt.Sprintf("installed %s, project declares %s@%s", installedVersion, manager, declaredVersion)
	}

	return true, ""
}

// normalizeVersionForComparison strips a leading "v" (as printed by, for
// example, `node --version`) so version strings can be parsed as semantic
// versions for comparison. The raw value is left untouched for display.
func normalizeVersionForComparison(raw string) string {
	return strings.TrimPrefix(raw, "v")
}
