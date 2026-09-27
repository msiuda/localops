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

## Future direction

LocalOps may later support:

- build validation,
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
