# LocalOps

## Overview

LocalOps is a local developer tool for managing, inspecting, diagnosing, and eventually operating software projects from a single place.

The goal is to reduce the amount of manual work required to understand why a project works locally but fails in another environment such as CI, a build server, or a deployment platform.

LocalOps should remain useful as a standalone local tool first. Integrations with external systems are secondary.

## Core principles

- Local-first.
- Fast to use.
- Useful without extensive configuration.
- Prefer convention and automatic discovery over manual setup.
- Provide clear explanations instead of raw diagnostic output.
- Never hide failures or silently ignore problems.
- Avoid unnecessary complexity.
- Features should solve real developer workflow problems.

## Initial scope

The first area of LocalOps is project management.

LocalOps should be able to discover and register local projects and understand basic information about them.

Examples include:

- project name,
- project path,
- detected technologies,
- detected package/build systems,
- repository information,
- available project commands,
- basic project health information.

The initial interface may be CLI-based while the core functionality is developed.

## Projects

A project represents a local software project known to LocalOps.

Initial capabilities should eventually include:

- add a project,
- remove a project,
- list known projects,
- inspect a project,
- automatically detect basic project metadata.

Project detection should be based on the actual contents of the repository rather than requiring the user to manually describe the project.

## Technology Intelligence

LocalOps understands what a project is actually built with: its languages, runtime/platform, and frameworks, along with the evidence supporting each detection (which file, and why).

This answers "what is this project built with?" as a structured fact other LocalOps capabilities can consult, rather than each capability guessing independently. Detection is evidence-based and conservative: a framework is only reported when the project's own manifest authoritatively declares a dependency on it (e.g. `composer.json` requiring `laravel/framework`), never from a convention file's mere presence or a source-extension guess alone. LocalOps would rather under-report than confidently report a technology that isn't really there.

Detecting a technology does not imply full LocalOps support for it. Doctor, Validation, and other capabilities are extended to a technology independently and separately, on their own schedule — detection is the foundation those future extensions build on, not a promise that they already exist.

### Supported ecosystem families

LocalOps detects ten ecosystem families: JavaScript/TypeScript/Node.js, Python, PHP, Go, Rust, Java, C#/.NET, Ruby, Kotlin, and C/C++, along with the common frameworks within each that can be identified from strong project evidence (React, Next.js, Vue, Angular, Svelte, Express, and NestJS for JS/TS; Django, Flask, and FastAPI for Python; Laravel and Symfony for PHP; Gin, Fiber, and Echo for Go; Axum and Actix Web for Rust; Spring Boot for Java/Kotlin; ASP.NET Core for C#; Rails for Ruby; and Ktor for Kotlin).

`package.json`'s presence proves the Node.js/npm ecosystem is in use — it does not by itself prove JavaScript is one of the project's primary source languages. When `tsconfig.json` or a `typescript` dependency is present, LocalOps reports **TypeScript**, not TypeScript alongside a redundant "JavaScript" the user never asked about; JavaScript is only reported as the project's language when there is no TypeScript evidence at all. Node.js itself is always reported (as the runtime) whenever `package.json` exists, independent of which language is reported.

### Compact vs. full technology presentation

LocalOps presents technology information at two different densities, deliberately holding different amounts of information:

- **Compact summary** (the Projects list and the Project Detail header): a quick-scan identity — primary language(s), then the most characteristic framework(s), then runtime/platform, capped at a small fixed number of chips with the rest collapsed into a single "+N" indicator (its hidden names available via tooltip). Git repository status is not part of this summary; it is near-universal repository metadata, not a Technology Intelligence fact. The package manager is shown alongside it as subordinate text, not as a technology chip — it is metadata about the stack, not a technology identity. This summary never wraps to a second line and never grows a row's or header's height, regardless of how many technologies were detected.
- **Full picture** (Project Detail's Overview "Stack" section): every detected technology, grouped by kind (Languages, Runtime/Platform, Frameworks), alongside the package manager.

Both are derived from the same single Technology Intelligence detection result; the desktop frontend never independently re-derives technology priority.

**Framework precedence in the compact summary**: when a project has both a framework and a more-identifying framework built on it (e.g. React underneath Next.js, or Express underneath NestJS), the compact summary shows only the more identifying one — Next.js, not React; NestJS, not Express. This is a presentation choice only: Technology Intelligence detection and Overview's full grouped picture continue to show both.

### Current capability by technology

| Technology | Detection | Environment usage | Doctor | Validation |
|---|---|---|---|---|
| JavaScript/TypeScript/Node.js | Supported | Supported | Supported | Supported |
| Go | Supported | Supported | Supported | Supported |
| Python | Supported | Supported | Not yet | Not yet |
| PHP | Supported | Supported | Not yet | Not yet |
| Laravel | Supported | Supported | Not yet | Not yet |
| Symfony | Supported | Supported | Not yet | Not yet |
| Rust | Supported | Supported | Not yet | Not yet |
| Java | Supported | Supported | Not yet | Not yet |
| C#/.NET | Supported | Supported | Not yet | Not yet |
| Ruby | Supported | Supported | Not yet | Not yet |
| Kotlin | Supported | Supported | Not yet | Not yet |
| C/C++ | Supported | Supported | Not yet | Not yet |

"Environment usage" means statically recognizable source-code environment-variable usage, per the forms listed in "Environment Source Usage" below — not full semantic analysis of that language. "Not yet" means exactly that: Doctor and Validation have not been extended to that technology yet, not that LocalOps attempted and failed. This table must only ever reflect what is actually implemented and tested, never aspirational coverage.

## Doctor

Doctor will diagnose problems with a project and its environment.

The long-term goal is to answer questions such as:

- Is the local environment correctly configured?
- Are required tools installed?
- Are expected versions being used?
- Are dependencies or configuration files missing?
- Are build commands valid?
- Are there differences that could cause CI or deployment failures?

Doctor should prefer actionable diagnostics.

A useful diagnostic should explain:

1. what was checked,
2. what was found,
3. why it may be a problem,
4. what can be done about it.

## Project Overview

Overview gives a single, at-a-glance view of every registered project's
health, combining project registration, project inspection, and Doctor.

For each registered project, Overview reports whether it is:

- healthy — inspected successfully and every Doctor check passed,
- has issues — inspected successfully but at least one Doctor check did
  not pass,
- unavailable — its registered path could not be inspected (for example,
  it no longer exists).

A project that is unavailable does not prevent the rest of the registered
projects from being evaluated.

Overview does not introduce new detection or diagnostics of its own; it
composes the existing project inspection and Doctor capabilities.

## Project Validation

Validation answers a different question than Doctor: given the project's
detected technology and the current local environment, can its basic
project checks actually run successfully on this machine?

Unlike project inspection and Doctor, which are read-only, Validation is an
explicitly active operation: the user invoked `localops validate`, and it
actively runs the project's own validation commands.

For this milestone, Validation runs:

- for a Go module: `go test`, `go vet`, and `go build`,
- for a Node.js project: its recognized `package.json` scripts (`lint`,
  `typecheck`, `test`, `build`), through the project's effective package
  manager.

Before running anything, Validation inspects the project and runs Doctor.
If a required tool is unavailable or incompatible, only the checks that
depend on it are blocked; independent checks still run.

Validation never installs dependencies, modifies package manifests, runs
package-manager install/update commands, deploys anything, or runs any
command beyond the specific ones listed above.

## Environment Contract

Environment Contract answers: what environment variables does this
project declare that it expects, and which of those are available in the
current local environment?

It is read-only, like project inspection and Doctor.

Supported contract files, which declare the expected variable names, are
`.env.example`, `.env.sample`, and `.env.template`. If more than one
exists, their declared variables are combined into a union.

Supported local sources, checked for whether a declared variable is
present, are `.env.local`, `.env`, and the current process environment,
in that precedence order.

Environment Contract never interprets, stores, prints, or otherwise
exposes an environment variable's value. An env file's bytes must be read
from disk to identify its declared keys, but a value is never retained
past that; only variable names, the source file(s) that declare or
satisfy them, and presence/absence are ever represented.

## Environment Source Usage

Environment also answers a second question: which environment variables
does the project's own source code statically and recognizably use? This
extends the mental model from declared/satisfied to three independent
facts — **used**, **declared**, **satisfied** — so LocalOps can
distinguish:

1. used + declared + satisfied — healthy,
2. used + declared + missing — a declared variable the source needs that
   is not locally satisfied,
3. used + undeclared — the source uses a variable the contract never
   declared,
4. declared + no supported usage found — an informational hygiene signal,
   not proof the variable is actually unused.

Supported usage forms are deliberately narrow, matching the ten ecosystem
families Technology Intelligence detects (see "Technology Intelligence"
above). Only a statically knowable string-literal key is ever recognized;
a dynamic expression (a variable, concatenation, a method call, and so
on) as the key is never statically knowable and is skipped, for every
language below.

- **Go**: `os.Getenv("NAME")`, `os.LookupEnv("NAME")`, recognized via the
  standard library's own `go/parser`/`go/ast`, resolved against the
  file's own `os` import (a plain import or an explicit single-level
  alias).
- **JavaScript/TypeScript**: `process.env.NAME`, `process.env["NAME"]`/
  `['NAME']`, and the `import.meta.env` equivalents, via a small
  purpose-built lexical scanner (comments and ordinary string contents
  are skipped). Vite's own built-in `import.meta.env` keys — `MODE`,
  `BASE_URL`, `PROD`, `DEV`, and `SSR` — are never treated as
  user-defined Environment Contract variables: they are excluded from
  the used/declared/satisfied correlation entirely (never "undeclared,"
  never expected in `.env.example`), since Vite injects them into every
  project regardless of what the user declares. This exclusion is scoped
  to `import.meta.env` only; `process.env.DEV` is an ordinary,
  user-controlled usage and is still recognized normally. A project's
  own `import.meta.env.VITE_*` variables are unaffected and still
  recognized.
- **Python**: `os.getenv("NAME")`, `os.environ["NAME"]`,
  `os.environ.get("NAME")`, resolved against the file's own `import os`
  (or `import os as alias`).
- **PHP**: `getenv("NAME")`, `$_ENV["NAME"]`/`['NAME']`,
  `$_SERVER["NAME"]`/`['NAME']` unconditionally; **Laravel's**
  `env("NAME")` and **Symfony's** `%env(NAME)%` (in YAML/XML
  configuration) only when that framework was actually detected — `env()`
  reads as an ordinary function name outside a Laravel project, so it is
  never recognized without that evidence.
- **Rust**: `std::env::var("NAME")`/`var_os("NAME")` always; the bare
  `env::var(...)`/`var_os(...)` form only when the file's own
  `use std::env` import is found.
- **Java** and **Kotlin**: `System.getenv("NAME")` (the same JVM
  standard-library call).
- **C#**: `Environment.GetEnvironmentVariable("NAME")`, including the
  fully-qualified `System.Environment...` form.
- **Ruby**: `ENV["NAME"]`/`['NAME']`, `ENV.fetch("NAME")`/`('NAME')`.
- **C**: `getenv("NAME")`.
- **C++**: `getenv("NAME")` and `std::getenv("NAME")`.

This recognition is intentionally conservative: false negatives on
exotic or dynamic syntax are preferred over false positives from a blind
text match. "No supported static usage found" is never presented as
proof a variable is unused — dynamic access, an unsupported language, or
an unscanned file could still use it.

Source-usage scanning runs even when no Environment Contract exists, so
a project with no contract can still report used-but-undeclared
variables — but it never reads `.env`/`.env.local` or queries the
process environment unless at least one variable is declared, preserving
Environment Contract's existing guarantee against unnecessary access to
files that may carry secrets.

LocalOps does not scan shell scripts or CI configuration for usage, and
does not scan generic configuration files — the one exception is
Symfony's own `%env(NAME)%` syntax in its YAML/XML configuration, which is
only recognized when Symfony itself was detected, and is not a general
YAML/XML scanning capability. It is intended to eventually support
comparing the local environment against CI or deployment environments,
but that comparison is not implemented yet.

## Future direction

LocalOps may later support:

- environment comparison,
- CI configuration inspection,
- deployment simulation,
- deployment platform integrations,
- Jenkins diagnostics,
- AWS Amplify diagnostics,
- project-specific automation,
- reusable project checks.

These are future capabilities and should not influence the initial architecture unless required by current functionality.

## Out of scope for the initial version

The initial version does not need:

- graphical user interface,
- cloud synchronization,
- user accounts,
- remote project management,
- deployment execution,
- plugin architecture,
- distributed execution,
- complex configuration systems.

These capabilities should not be introduced speculatively.

## First milestone

The first milestone is a small but functional CLI capable of managing local projects.

Conceptually:

```text
localops project add .
localops project list
localops project inspect <project>
```

The exact CLI design is not final and may evolve during implementation.

The milestone is complete when LocalOps can persist a local project, retrieve it later, and inspect enough information from the repository to provide useful project metadata.
