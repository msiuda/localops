# Testing

Tests should verify observable behavior and important failure cases.

## Principles

- Use Go's standard `testing` package by default.
- Prefer simple tests over elaborate testing infrastructure.
- Use table-driven tests when multiple cases naturally share the same behavior.
- Do not force every test into a table-driven structure.
- Test public or meaningful package behavior rather than private implementation details.
- Do not add mocks when a simpler test boundary is available.
- Do not add a mocking framework without a concrete need.

## Filesystem tests

- Use `t.TempDir()` for temporary filesystem state.
- Tests must not read from or write to the user's real LocalOps data.
- Tests must not depend on the developer's current working directory, installed projects, or home directory contents.

## Reliability

Tests should be:

- deterministic,
- isolated,
- repeatable,
- independent of execution order.

Unit tests should not require network access unless the test is explicitly intended as an integration test.

## Test integrity

Never:

- weaken an existing test merely to make it pass,
- delete a failing test without understanding why it fails,
- hard-code implementation behavior specifically for a test case,
- replace meaningful assertions with weaker ones to obtain a green test suite.

If a test appears incorrect, explain the issue rather than working around it.
