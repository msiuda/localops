// Package doctor checks whether the local machine has the executables
// required by a project's detected technologies.
package doctor

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/msiuda/localops/internal/project"
)

// CheckResult reports whether a single required executable is available on
// the local machine and, if so, the version it reports.
type CheckResult struct {
	Tool      string
	Available bool
	Version   string
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
// in insp are available on the local machine and, for each available
// executable, retrieves its reported version.
//
// Only lookup and each tool's own version command are ever run; no project
// scripts, builds, or other commands are executed.
func Run(insp project.Inspection) Report {
	return run(insp, exec.LookPath, runVersionCommand)
}

// run implements Run, taking executable lookup and version retrieval as
// small function seams so tests can substitute them instead of depending on
// the machine's PATH or real tool installations.
func run(insp project.Inspection, lookPath lookPathFunc, runVersion runVersionFunc) Report {
	var tools []string

	if insp.IsGitRepository {
		tools = append(tools, "git")
	}
	if insp.HasGoMod {
		tools = append(tools, "go")
	}
	if insp.IsNodeProject {
		tools = append(tools, "node")

		switch insp.NodePackageManager {
		case "pnpm":
			tools = append(tools, "pnpm")
		case "yarn":
			tools = append(tools, "yarn")
		case "npm":
			tools = append(tools, "npm")
		}
	}

	checks := make([]CheckResult, 0, len(tools))
	for _, tool := range tools {
		checks = append(checks, checkTool(tool, lookPath, runVersion))
	}

	return Report{
		Path:   insp.Path,
		Checks: checks,
	}
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
