# LocalOps

LocalOps is a local developer CLI tool for managing and inspecting local
software projects, with basic diagnostics for whether your machine has the
tools a project needs.

## What it does today

- **Register and list local projects** — keep a small local record of
  projects you work on.
- **Inspect a project's filesystem metadata**, based purely on the files
  already in the repository (no external commands are run):
  - Git repository detection (`.git` directory or worktree file),
  - Go module detection (presence of `go.mod` and its declared module path),
  - Node.js project detection (presence of `package.json` and its declared
    package name), including which package manager is in use, based on
    `pnpm-lock.yaml`, `yarn.lock`, or `package-lock.json`.
- **Doctor**: given a project's inspection results, checks whether the
  executables required by its detected technologies (`git`, `go`, `node`,
  and its package manager) are available on your `PATH`. For each available
  executable, Doctor runs only its version command (e.g. `git --version`,
  `go version`, `node --version`) and reports the raw version output.
  For Node.js projects, Doctor additionally:
  - checks the installed Node.js version against `package.json`'s
    `engines.node` requirement, when declared,
  - reconciles the package manager declared by `package.json`'s
    `packageManager` field with the one detected from a lockfile, flagging
    a mismatch, or an unsupported/malformed `packageManager` value, as a
    finding (falling back to the lockfile-detected manager, if any, so it
    can still be checked),
  - checks the effective package manager's installed version against the
    exact version `packageManager` declares, when present,
  - when npm is the effective package manager, checks its installed
    version against both the exact version `packageManager` declares and
    the `engines.npm` range, when either or both are present — both must
    be satisfied.

  Doctor does not compare Go versions, does not inspect `.nvmrc`,
  `.node-version`, Volta, or Corepack configuration, and does not run
  builds, installs, project scripts, or any other project commands.
- **Overview**: loads every registered project and, for each one
  independently, inspects it and runs Doctor against it, then reports its
  health as one of:
  - **healthy** — inspection and Doctor both ran and every Doctor check
    passed,
  - **issues** — inspection and Doctor both ran but at least one Doctor
    check did not pass (the failed checks are listed),
  - **unavailable** — the registered path could no longer be inspected
    (for example, it no longer exists). An unavailable project does not
    stop the rest of the overview from running.

Everything above is read-only with respect to the project being inspected.

## Commands

```text
localops --help
localops project add <path>
localops project list
localops project inspect <path>
localops doctor <path>
localops overview
```

Run `localops help`, `localops project help`, `localops doctor --help`, or
`localops overview --help` for details on any command.

## Running from the repository

```bash
go run ./cmd/localops project inspect .
go run ./cmd/localops doctor .
go run ./cmd/localops overview
```

## Building

```bash
go build ./cmd/localops
```

This produces a `localops` binary in the current directory.

## Installing

```bash
go install ./cmd/localops
```

This installs the `localops` binary to your Go bin directory (`go env GOBIN`,
or `$(go env GOPATH)/bin` if `GOBIN` is unset). Make sure that directory is
on your `PATH` to run `localops` directly.

## Quick Start

From inside a project you want to look at:

```bash
go run ./cmd/localops project add .
go run ./cmd/localops overview
go run ./cmd/localops project inspect .
go run ./cmd/localops doctor .
```

After installing:

```bash
localops project add .
localops overview
localops project inspect .
localops doctor .
```
