# LocalOps

LocalOps is a local developer tool written in Go.

Its purpose is to help developers manage, inspect, diagnose, and eventually operate local software projects.

## Project context

Before implementing non-trivial changes, read the relevant project documentation:

- `docs/product.md`
- `docs/architecture.md`

Treat these documents as the source of truth for current product scope and architectural direction.

Do not implement future ideas from `docs/product.md` unless they are explicitly part of the current task.

## Project rules

Follow the applicable rules defined in:

- `.claude/rules/architecture.md`
- `.claude/rules/go.md`
- `.claude/rules/testing.md`
- `.claude/rules/workflow.md`

These rules are part of the project requirements.

## Current development approach

LocalOps is being developed incrementally.

Prefer:

- small, reviewable changes,
- simple implementations,
- explicit behavior,
- standard library solutions where reasonable,
- architecture driven by current requirements.

Avoid:

- speculative abstractions,
- premature extensibility,
- unnecessary layers,
- unrelated refactors,
- implementing future roadmap features early.

## Repository structure

The repository structure should evolve with the application.

Do not create directories, packages, or architectural layers unless the current implementation requires them.

The expected application entry point, once introduced, is:

```text
cmd/localops/
```

Application-specific code should generally remain internal to the module unless there is a concrete reason to expose it publicly.

## Development commands

Use the following commands when verifying Go changes:

```bash
gofmt
go test ./...
go vet ./...
go build ./...
```

Run only commands that are applicable to the current repository state.

Never report a command as successful unless it was actually executed successfully.

## Working with tasks

Before implementing a non-trivial task:

1. inspect the relevant code,
2. read relevant documentation,
3. understand existing patterns,
4. determine the smallest reasonable implementation.

If the requested change conflicts with current architecture or product documentation, identify the conflict before introducing a workaround.

During implementation, keep the task scope narrow.

After implementation:

1. review the full diff,
2. remove unnecessary complexity,
3. run applicable verification,
4. summarize what changed and any important assumptions.

## Git

Do not create commits unless explicitly requested.

Do not rewrite Git history or use destructive Git operations unless explicitly requested.
