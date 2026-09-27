# Development workflow

Work as an engineer modifying an existing codebase, not as a code generator producing an isolated solution.

## Before implementation

Before changing code:

1. Read the relevant project documentation.
2. Inspect the files related to the requested change.
3. Understand the existing implementation and conventions.
4. Identify the smallest reasonable scope for the task.

Never speculate about code that has not been inspected.

For non-trivial work, briefly establish the implementation approach before editing.

If requirements are ambiguous but a reasonable assumption allows progress, state the assumption instead of silently inventing behavior.

## During implementation

Keep changes focused on the requested task.

Do not:

- perform unrelated refactors,
- rename unrelated code,
- reformat unrelated files,
- add features that were not requested,
- create abstractions solely for possible future use,
- add dependencies without a concrete benefit,
- rewrite working code simply because another style is possible.

Prefer modifying existing structures over introducing parallel implementations.

Temporary scripts or files created during investigation must be removed before finishing unless they are intentionally part of the solution.

## Dependencies

Before adding an external dependency:

1. Check whether the standard library reasonably solves the problem.
2. Confirm that the dependency materially simplifies the implementation.
3. Consider the maintenance and dependency-tree cost.

Do not add a dependency merely to avoid writing a small amount of straightforward Go.

## Verification

Before considering implementation complete:

1. Review the full diff.
2. Check for unnecessary complexity or unrelated changes.
3. Run `gofmt` on modified Go files.
4. Run:

   `go test ./...`

5. Run:

   `go vet ./...`

6. Run:

   `go build ./...`

Do not claim verification succeeded unless the command was actually executed successfully.

If a verification command cannot run, report why.

## Finishing a task

At the end, provide a concise summary containing:

- what changed,
- important design decisions,
- verification performed,
- any unresolved issue or assumption.

Do not create Git commits unless explicitly requested.

Do not use destructive Git operations such as `reset --hard`, forced checkout, or rewriting history unless explicitly requested.

## Documentation synchronization

When a task changes user-facing commands, behavior, setup, or currently supported capabilities, review the relevant README/project documentation and update it in the same change when it has become stale.

Do not update documentation speculatively for features that are not implemented.

## Local machine safety

Do not execute commands that create, modify, or delete files outside the repository unless explicitly required by the task or approved by the user.

When manually smoke-testing behavior that normally writes to user directories, isolate the environment using a temporary HOME or another explicit temporary location.

Do not use the developer's real application config, cache, or data directories for verification.
