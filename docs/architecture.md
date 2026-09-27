# LocalOps Architecture

## Purpose

This document describes the architecture of LocalOps.

It should reflect the architecture that exists or is currently required by the project.

Future ideas described in `product.md` must not introduce abstractions into the codebase before they are needed.

The architecture should evolve together with real product requirements.

---

## Current milestone

The current milestone is a CLI application capable of:

- registering a local project,
- listing registered projects,
- inspecting a project,
- detecting basic project metadata,
- persisting registered projects locally.

The architecture should remain intentionally small while supporting these capabilities.

---

## Architectural principles

### Prefer simple boundaries

Separate responsibilities when they are meaningfully different.

Do not create additional architectural layers purely to satisfy a pattern.

Avoid structures such as:

- repositories wrapping trivial file operations,
- service layers that only forward method calls,
- factories with a single implementation,
- interfaces with a single implementation and no consumer-side need,
- generic abstractions created only for hypothetical future functionality.

### Dependencies point inward toward behavior

CLI concerns should not contain core project inspection logic.

Core behavior should not depend on CLI presentation details.

Persistence details should not leak into project inspection logic.

### Interfaces are introduced at the consumer

Interfaces should only be introduced when they provide a concrete benefit.

Prefer concrete types by default.

When an interface is required, define it close to the code that consumes it rather than next to its implementation.

### Explicit behavior over hidden behavior

LocalOps should make important operations visible and understandable.

Avoid:

- implicit global state,
- hidden initialization,
- package-level mutable state,
- behavior triggered through side effects,
- configuration discovered through undocumented magic.

### Standard library first

Prefer the Go standard library when it provides a clear and maintainable solution.

External dependencies are acceptable when they provide meaningful value and reduce complexity.

Dependencies should not be introduced for functionality that can reasonably be implemented using the standard library.

---

## Initial structure

The exact directory structure may evolve.

For the first milestone, the project should remain close to conventional Go application structure.

A likely shape is:

```text
localops/
├── cmd/
│   └── localops/
│       └── main.go
│
├── internal/
│   ├── project/
│   ├── doctor/
│   ├── overview/
│   └── storage/
│
├── docs/
├── CLAUDE.md
└── go.mod
```

This structure is a direction, not a requirement.

Directories should only be created when they contain real code with a clear responsibility.

---

## Command layer

The CLI is an entry point into LocalOps.

Its responsibilities include:

- parsing user input,
- validating command arguments,
- invoking application behavior,
- presenting results,
- mapping failures to useful CLI output and exit codes.

The CLI should not contain:

- project detection logic,
- filesystem inspection rules,
- persistence implementation,
- business decisions unrelated to presentation.

The CLI should remain thin.

---

## Project domain

The `project` area represents local software projects known to LocalOps.

It owns concepts such as:

- project identity,
- project path,
- project metadata,
- project inspection,
- technology detection.

A project should initially contain only information that LocalOps actually uses.

Avoid building a large domain model before the required metadata is known.

A possible minimal representation may eventually contain:

```go
type Project struct {
    Name string
    Path string
}
```

Additional fields should be introduced when required by real behavior.

---

## Project inspection

Project inspection reads information from a local repository or directory and produces project metadata.

Inspection may eventually detect things such as:

- Git repository information,
- Go modules,
- Node package managers,
- build systems,
- configuration files,
- available commands.

Detection should be based on observable project files and tools.

Individual detectors should remain small and focused.

Do not create a plugin system for detectors during the initial milestone.

A simple collection of explicit detection functions is preferred until extensibility becomes an actual problem.

---

## Overview module

The `overview` package combines project registration, project inspection, and Doctor into a single structured view of every registered project's health.

Overview depends on `project` and `doctor`. Neither `project`, `doctor`, nor `storage` depends on Overview:

```text
overview
  ├── project
  └── doctor

CLI
  ├── storage
  └── overview
```

The CLI loads registered projects from `storage` and passes them into Overview; Overview itself never reads or writes LocalOps storage.

Overview evaluates each registered project independently. A project that cannot be inspected (for example, its path no longer exists) is reported as unavailable rather than aborting the rest.

Overview assigns each project one of three health states: healthy, issues, or unavailable. This is intentionally a small, explicit model, not a generic severity or diagnostics framework.

Overview composes `project.Inspect` and `doctor.Run` rather than duplicating their logic. Presentation (CLI output formatting) remains in the command layer, not in Overview.

---

## Persistence

Registered projects must survive between LocalOps executions.

For the first milestone, persistence should be local and simple.

The preferred initial solution is a single LocalOps data file stored in an appropriate user-specific application data/configuration directory.

The exact format may be decided during implementation.

Requirements:

- human-readable storage is preferred while the project is small,
- writes must not silently corrupt existing data,
- missing storage should be treated as an empty LocalOps state,
- invalid or corrupted storage must produce an explicit error,
- persistence behavior must be testable without touching the user's real LocalOps data.

A database should not be introduced unless file-based persistence becomes insufficient.

---

## Filesystem interaction

Filesystem access is a core part of LocalOps.

Code interacting with the filesystem should:

- use explicit paths,
- avoid changing the process working directory as hidden state,
- normalize or resolve paths when necessary,
- return useful errors with context,
- avoid modifying inspected projects unless a feature explicitly requires it.

Project inspection must be read-only during the initial milestone.

---

## Error handling

Errors are part of the application behavior.

Errors should:

- be returned rather than logged and swallowed,
- contain enough context to understand which operation failed,
- preserve underlying errors where useful,
- be translated into user-facing messages at the CLI boundary.

Avoid creating a complex custom error hierarchy without a concrete need.

Sentinel errors may be introduced for cases where callers need to distinguish specific expected conditions.

---

## Logging

The initial application does not require a complex logging framework.

Normal command results belong on standard output.

User-facing failures belong on standard error.

Internal diagnostic logging may be introduced later if there is a real need for verbose or debug modes.

---

## Configuration

The initial milestone should require little or no configuration.

Configuration must not be introduced merely because future features may need it.

When configuration becomes necessary, it should have:

- explicit precedence,
- documented defaults,
- predictable file locations,
- clear validation errors.

---

## Testing boundaries

Code containing behavior should be testable independently from the CLI process.

Tests should primarily cover:

- project registration behavior,
- persistence,
- project inspection,
- detection rules,
- error cases.

Prefer normal Go tests using the standard `testing` package.

Use temporary directories for filesystem-related tests.

Tests must not depend on the developer's machine state unless explicitly marked as integration tests.

---

## Dependency policy

Before adding an external dependency, consider:

1. Is the functionality already reasonably available in the standard library?
2. Does the dependency meaningfully reduce complexity?
3. Is it actively maintained?
4. Does it introduce a disproportionate dependency tree?
5. Would implementing the small required behavior directly be clearer?

Dependencies should solve concrete problems rather than anticipated ones.

---

## Architecture evolution

Architecture changes should follow product needs.

When introducing a new abstraction, there should be a concrete reason such as:

- multiple implementations exist,
- meaningful duplication exists,
- testing requires an explicit boundary,
- responsibilities have clearly diverged,
- current coupling makes a real feature difficult to implement.

"LocalOps may need this in the future" is not sufficient justification.

When the architecture changes significantly, this document must be updated together with the code.
