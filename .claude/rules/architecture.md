# Architecture

Treat `docs/product.md` and `docs/architecture.md` as the source of truth for product direction and architectural decisions.

## Before changing architecture

- Inspect the existing implementation first.
- Follow established project patterns unless there is a concrete reason to change them.
- Introduce only the structure required by the current task.
- Prefer the smallest design that cleanly solves the current requirement.

Do not introduce architecture for functionality that does not exist yet.

In particular, do not create speculative:

- plugin systems,
- provider systems,
- factories,
- registries,
- generic repositories,
- service layers that only forward calls,
- dependency injection frameworks,
- event buses,
- adapter layers,
- extension points,
- abstractions with only one hypothetical future consumer.

## Boundaries

Keep concerns separated when their responsibilities are actually different.

For the current LocalOps scope:

- CLI code handles input and presentation.
- Project logic handles project concepts and inspection.
- Persistence handles storing and retrieving LocalOps state.
- Project inspection must remain independent from CLI presentation.

Do not move logic into a new layer solely to satisfy an architectural pattern.

## Changes

Architectural changes must be justified by a current requirement or existing problem.

Valid reasons include:

- responsibilities have clearly diverged,
- meaningful duplication exists,
- multiple real implementations exist,
- testing requires a useful boundary,
- current coupling prevents a required feature.

"LocalOps may need this later" is not sufficient justification.

If a requested implementation would materially conflict with `docs/architecture.md`, identify the conflict before proceeding.

If the architecture changes materially, update `docs/architecture.md` together with the implementation.
