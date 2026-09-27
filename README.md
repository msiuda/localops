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
  and its package manager) are available on your `PATH`. Doctor only checks
  for executables — it does not check versions, run builds, or execute any
  of those tools.

Everything above is read-only with respect to the project being inspected.

## Commands

```text
localops --help
localops project add <path>
localops project list
localops project inspect <path>
localops doctor <path>
```

Run `localops help`, `localops project help`, or `localops doctor --help`
for details on any command.

## Running from the repository

```bash
go run ./cmd/localops project inspect .
go run ./cmd/localops doctor .
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
go run ./cmd/localops project inspect .
go run ./cmd/localops doctor .
```

After installing:

```bash
localops project inspect .
localops doctor .
localops project add .
localops project list
```
