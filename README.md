# LocalOps

LocalOps is a local developer tool for managing and inspecting local
software projects, with basic diagnostics for whether your machine has the
tools a project needs. It ships as a CLI (`cmd/localops`) and, as of this
milestone, a native desktop application (`cmd/desktop`) — both operate on
the same registered-project data.

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
  - **Technology Intelligence**: structured detection of the languages,
    runtime/platform, and frameworks a project is built with, across ten
    ecosystem families (JavaScript/TypeScript/Node.js, Python, PHP, Go,
    Rust, Java, C#/.NET, Ruby, Kotlin, C/C++) plus their common
    frameworks (React, Next.js, Vue, Angular, Svelte, Express, NestJS,
    Django, Flask, FastAPI, Laravel, Symfony, Gin, Fiber, Echo, Axum,
    Actix Web, Spring Boot, ASP.NET Core, Rails, Ktor), based only on
    authoritative project manifests (`package.json`, `composer.json`,
    `go.mod`, `Cargo.toml`, `pom.xml`/`build.gradle(.kts)`, `*.csproj`,
    `Gemfile`, `CMakeLists.txt`, and similar) — never a convention file's
    mere presence or a source-extension guess alone, to keep false
    positives rare. Detecting a technology does not by itself mean Doctor
    or Validation support it yet; see `docs/product.md`'s support matrix
    for exactly what each technology currently has.
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

Registration, inspection, Doctor, and Overview are all read-only with
respect to the project being examined.

- **Validate** (`localops validate <path>`) is different: it **actively
  runs** the project's validation commands, rather than only inspecting or
  diagnosing. Before running anything, it inspects the project and runs
  Doctor; if a required tool is unavailable or incompatible, only the
  checks that depend on it are blocked — independent checks still run.
  - For a Go module: `go test ./...`, `go vet ./...`, `go build ./...`.
  - For a Node.js project: its recognized `package.json` scripts — `lint`,
    `typecheck`, `test`, `build` — run through the project's effective
    package manager (e.g. `pnpm run lint`). A script is only run if
    declared; unrelated scripts are ignored. Node checks are blocked if
    `node_modules` is missing (dependencies are not installed).
  - Validate never installs dependencies, modifies package manifests, runs
    package-manager install/update commands, deploys anything, runs dev
    servers, runs migrations or seed commands, or runs any script beyond
    the recognized ones above.
  - A project can be both a Go module and a Node.js project; Validate runs
    checks for both.
- **Environment** (`localops environment <path>`) analyzes a project's
  Environment Contract: which environment variables it declares it
  expects, and which are currently satisfied locally. It is read-only.
  - **Contract sources** (declare expected variable names): `.env.example`,
    `.env.sample`, `.env.template`. If more than one exists, their declared
    variables are combined into a union.
  - **Local sources** (checked for whether a declared variable is
    present): `.env.local`, then `.env`, then the current process
    environment — in that precedence order.
  - Environment **never interprets, stores, prints, or otherwise exposes
    an environment variable's value** — only variable names, which source
    file(s) declare or satisfy them, and presence/absence are ever shown.
    (An env file's bytes are read from disk to identify its declared
    keys, but a value is never retained past that.)
  - It also scans the project's own source code for statically recognizable
    environment-variable usage, and correlates it with the contract, so it
    can answer not just "declared vs. satisfied" but **used, declared, and
    satisfied** together:
    - Supported across all ten Technology Intelligence ecosystem
      families — Go, JavaScript/TypeScript, Python, PHP (plus Laravel's
      `env()` and Symfony's `%env()%`, each gated by that framework
      actually being detected), Rust, Java, Kotlin, C#, Ruby, C, and
      C++. See `docs/product.md`'s "Environment Source Usage" section
      for the exact recognized form per language.
    - This scan runs even when no Environment Contract exists, so a
      project with no contract can still report variables the source
      uses — but it never reads `.env`/`.env.local` or the process
      environment unless at least one variable is declared.
    - Recognition is static and conservative: comments and ordinary string
      contents are never matched, and a dynamic access (a variable key, or
      any other non-literal expression) is not statically knowable and is
      skipped — false negatives on exotic syntax are preferred over false
      positives. A used-but-undeclared variable is reported as
      **undeclared**, never as missing; a declared variable with no
      detected usage is reported as "no supported static usage found",
      never as definitively unused (it may be used dynamically, from an
      unsupported language, or from a file LocalOps could not scan).
    - The scan recurses from the project root, skipping `.git`,
      `node_modules`, `vendor`, `dist`, `build`, `coverage`, `.next`,
      `target`, `obj`, `.gradle`, `.idea`, `.vscode`, `__pycache__`,
      `.venv`, and `venv` (deliberately not `bin`, which holds genuine
      source/scripts in some ecosystems), never follows symlinks, and
      only looks at each supported language's own source extensions.

## Commands

```text
localops --help
localops project add <path>
localops project list
localops project inspect <path>
localops doctor <path>
localops overview
localops validate <path>
localops environment <path>
```

Run `localops help`, `localops project help`, `localops doctor --help`,
`localops overview --help`, `localops validate --help`, or
`localops environment --help` for details on any command.

## Running from the repository

```bash
go run ./cmd/localops project inspect .
go run ./cmd/localops doctor .
go run ./cmd/localops overview
go run ./cmd/localops validate .
go run ./cmd/localops environment .
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
go run ./cmd/localops validate .
go run ./cmd/localops environment .
```

After installing:

```bash
localops project add .
localops overview
localops project inspect .
localops doctor .
localops validate .
localops environment .
```

`localops validate` actively runs the project's own validation commands
(e.g. `go test`, or `pnpm run lint`) — unlike the other commands above, it
is not read-only. `localops environment` is read-only, and never prints or
stores any environment variable's value.

## Desktop application

LocalOps also ships as a native desktop app, built with
[Wails v3](https://v3.wails.io/) (Go backend, React + TypeScript + Vite
frontend). The CLI remains fully supported and unaffected. There is one
shared registered-project registry (`internal/storage.DefaultPath()`),
and registration behaves identically from either entry point: both the
CLI's `project add` and the desktop app's Add Project call the same
`storage.Store.AddProject` operation, so a project added from either one
shows up in the other.

### Current capability

The desktop app implements two screens:

- **Projects** — a dashboard of every registered project's health, backed
  by the real storage → project inspection → Doctor → Overview chain (the
  same one the CLI's `localops overview` uses). Shows each project's name,
  path, health (healthy / issues / unavailable), detected technologies,
  package manager, and a concise preview of failed Doctor checks, with
  loading, empty, and error states. **Add Project** opens the native OS
  directory picker, registers the chosen folder (identical semantics to
  CLI `project add`, including rejecting a duplicate), and reloads the
  list — available from the header and from the empty state. Each row is
  a real interactive control (native keyboard focus, Enter/Space to open)
  — selecting one opens Project Detail. The sidebar's **Projects** entry
  is enabled and returns to this list from anywhere in the app, including
  from Project Detail.
- **Project Detail** — opened from a Projects row, with a back action
  returning to Projects. A selected project is identified by its
  registered path, carried as a typed TanStack Router search parameter
  (`/project?path=...&tab=...`) rather than a raw path URL segment.
  Its header shows the project's name, path, health, detected
  technologies, and package manager. Below that, four sections:
  - **Overview** — a compact summary of stack, package manager, Doctor
    issue count, and Environment contract/missing-variable/undeclared-usage
    status (the undeclared-usage line only appears when there is one).
  - **Doctor** — the full Doctor report (every check, not just failures):
    tool, pass/fail, detected version, and failure detail.
  - **Environment** — the Environment Contract and source-usage analysis
    as a single comparison matrix (a real `<table>`, sticky header while
    scrolling a long variable list): **Variable / Local / Contract /
    Usage**, with the concrete source filenames (e.g. `.env.local · .env`,
    `.env.example`) shown once in the column headers rather than repeated
    on every row. A used-but-undeclared variable reads **Undeclared** in
    Contract and **Not evaluated** in Local — never **Missing**, since
    LocalOps never checks local/process env for a name the contract
    doesn't declare. A declared variable with no detected usage shows a
    dash with a "no supported static usage found" tooltip, never "unused."
    Multiple usage locations collapse to the first plus a `+N more` count.
    Never renders a value.
  - **Validation** — an informational placeholder. Validation is **not
    runnable from the desktop app yet**: LocalOps can run a project's
    tests/builds/lint/typecheck from the CLI (`localops validate <path>`),
    and desktop execution — since Validation can run real project
    commands — will be added later as its own deliberate, explicit
    workflow. Opening Project Detail never runs anything beyond the
    existing read-only Doctor/Environment checks.

  Project Detail loads automatically when opened and has independent
  loading/error/unavailable states, so an Environment analysis failure
  never hides the project's Overview/Doctor information. There is no
  manual refresh action in either screen; Projects reloads automatically
  after Add Project succeeds.

Doctor, Validation, and Environment (the global sidebar entries, distinct
from Project Detail's own sections of the same name) and Settings remain
disabled navigation placeholders — they establish the app's information
architecture but are not implemented yet. Project remove and an in-app
theme switch are not implemented in this milestone either.

### Architecture

```text
cmd/desktop/            Wails v3 entry point (frameless window, service binding)
  frontend/             React + TypeScript + Vite UI (TanStack Router +
                         Query, Tailwind v4 — see docs/frontend.md)
internal/desktop/       Thin Wails-facing service: adapts internal/storage,
                         internal/overview, and internal/environment for
                         the frontend as small UI-facing DTOs, and wraps
                         the native directory-picker dialog. It does not
                         duplicate storage, inspection, Doctor, Overview,
                         Validation, or Environment logic.
```

The frontend never recomputes project health or re-parses the Environment
Contract; it only renders what `internal/desktop.Service`
(`GetOverview`, `GetProjectDetail`, `AddProject`, `PickProjectDirectory`)
— exposed to the frontend through Wails's generated TypeScript bindings,
no HTTP/REST/JSON-RPC layer — already computed in Go. `GetProjectDetail`
only ever operates on a path already present in the shared registry; it
never inspects an arbitrary unregistered path. `AddProject` and the CLI's
`project add` both call `storage.Store.AddProject`, the single shared
registration operation (path resolution, duplicate detection, and
persistence); neither entry point duplicates that logic itself.

### Developing the desktop app

Prerequisites: Go, Node.js, [pnpm](https://pnpm.io/installation) (LocalOps's
frontend package manager), and Make (for the convenience workflow below).
Wails itself does not need to be installed manually — `make bootstrap`
installs the pinned version.

The recommended entry point is the root `Makefile` (run `make help` for the
full list of targets; it's a convenience wrapper around the commands below,
not a requirement for building LocalOps). `make bootstrap` is sufficient for
first-time setup:

```bash
make bootstrap   # one-time: install pinned Wails CLI, frontend deps, generate bindings
make dev         # live-reloading development mode
make check       # gofmt/test/vet + frontend typecheck/build
make build       # production build
```

The underlying raw commands (what `make` delegates to) still work directly:

```bash
# One-time: install the Wails v3 CLI
go install github.com/wailsapp/wails/v3/cmd/wails3@v3.0.0-beta.26

cd cmd/desktop
wails3 task dev     # live-reloading development mode
wails3 build        # production build (binary at cmd/desktop/bin/localops)
```
