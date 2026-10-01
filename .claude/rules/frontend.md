# Frontend (cmd/desktop/frontend)

Read `docs/frontend.md` before any non-trivial frontend change. It is the
source of truth for structure, routing, and conventions; this file is the
enforceable checklist.

## Required

- Use TanStack Router for all navigation. Never reintroduce manual
  `useState<View>`-style page switching.
- Use TanStack Query for any state that comes from a Wails service call.
  Never hand-write a `useState`/`useEffect` loading-error-refetch hook for
  server state — add a query/mutation to the relevant `entities/*/model` or
  `features/*/model` instead.
- Only `entities/*/api/*Api.ts` may import a generated Wails binding
  directly. Everything else goes through that module.
- Place new code by what it is, not by convenience: app shell/providers →
  `app/`; a route file → `routes/` (thin — validate params, render a
  feature); a user action/use case → `features/`; domain data/presentation
  → `entities/`; framework-agnostic reusable code → `shared/`.
- Before adding a one-off style or component, check `shared/ui/atoms` and
  `shared/ui/molecules` for an existing primitive. Atomic Design applies
  only to that shared layer — never force a feature-specific component into
  "atom/molecule/organism" terminology.
- Use the `@/*` import alias (`@bindings/*` only inside a `*Api.ts` file).
  Never write a `../../../` relative import when `@/` reaches the same file.
- Add or update tests for user-visible behavior alongside any behavior
  change (Vitest + Testing Library; mock only the entity's `api` module,
  never a real Wails process).
- Before considering frontend work done, it must pass: `pnpm typecheck`,
  `pnpm lint`, `pnpm test:run`, `pnpm format:check`, `pnpm build` (or
  `make check` from the repo root, which runs all of these plus the Go
  suite).

## Also avoid

- Adding a dependency (a state manager, a UI framework, a form library,
  Zod, etc.) without a concrete requirement in the current task — see
  `docs/frontend.md` and ask before installing anything system/global.
- Leaving a milestone with a known UX/UI inconsistency, dead code, or a
  duplicated implementation "to polish later."
