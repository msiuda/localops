---
paths:
  - "**/*.go"
---

# Go

Write idiomatic, straightforward Go.

## General

- Always format Go code with `gofmt`.
- Prefer the standard library when it provides a clear solution.
- Prefer concrete types by default.
- Introduce interfaces only when they solve a concrete problem for the consuming code.
- Define interfaces close to their consumers, not their implementations.
- Keep packages focused around a clear responsibility.
- Avoid generic `utils`, `helpers`, `common`, or similar dumping-ground packages.
- Keep exported API surface as small as practical.
- Do not introduce abstractions for hypothetical future requirements.

## Errors

- Return errors instead of swallowing them.
- Add useful operation context when propagating errors.
- Preserve wrapped errors with `%w` when callers may need the underlying error.
- Do not log an error and return the same error unless both actions are intentionally required.
- Do not create custom error types or sentinel errors unless callers need to distinguish the condition programmatically.

## State

- Avoid mutable package-level state.
- Prefer explicit dependency passing over hidden initialization.
- Avoid `init()` unless there is a concrete technical reason for it.

## Concurrency

- Do not introduce goroutines or channels unless concurrency provides a real benefit.
- Every goroutine must have clear ownership and lifecycle.
- Do not introduce concurrency merely because Go makes it easy.

## Readability

- Prefer obvious code over clever code.
- Keep functions focused, but do not split code into trivial one-line helpers without a reason.
- Comments should explain decisions or non-obvious behavior, not restate the code.
