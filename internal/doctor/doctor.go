// Package doctor checks whether the local machine has the executables
// required by a project's detected technologies.
package doctor

import (
	"fmt"
	"os/exec"

	"github.com/msiuda/localops/internal/project"
)

// CheckResult reports whether a single required executable is available on
// the local machine.
type CheckResult struct {
	Tool      string
	Available bool
	Detail    string
}

// Report is the result of running Doctor checks against a project.
type Report struct {
	Path   string
	Checks []CheckResult
}

// Run checks whether the executables required by the technologies detected
// in insp are available on the local machine.
//
// Detection is read-only: required executables are looked up on PATH, never
// executed.
func Run(insp project.Inspection) Report {
	return run(insp, exec.LookPath)
}

// run implements Run, taking the executable lookup as a small function seam
// so tests can substitute it instead of depending on the machine's PATH.
func run(insp project.Inspection, lookPath func(string) (string, error)) Report {
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
		checks = append(checks, checkTool(tool, lookPath))
	}

	return Report{
		Path:   insp.Path,
		Checks: checks,
	}
}

// checkTool reports whether tool is available on PATH.
func checkTool(tool string, lookPath func(string) (string, error)) CheckResult {
	path, err := lookPath(tool)
	if err != nil {
		return CheckResult{
			Tool:      tool,
			Available: false,
			Detail:    "executable not found",
		}
	}

	return CheckResult{
		Tool:      tool,
		Available: true,
		Detail:    fmt.Sprintf("found at %s", path),
	}
}
