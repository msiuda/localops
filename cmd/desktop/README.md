# LocalOps desktop

The LocalOps desktop application, built with [Wails v3](https://v3.wails.io/)
(Go backend, React + TypeScript + Vite frontend). See the repository root
[README](../../README.md) for the full picture — this file only covers
running this app directly.

## Development

```bash
wails3 task dev
```

## Production build

```bash
wails3 build
```

Produces `bin/localops`.

## Structure

- `main.go` — Wails v3 entry point (window configuration, service binding).
- `frontend/` — the React/TypeScript/Vite UI.
- Backend logic is not implemented here: `main.go` only wires up
  `internal/desktop.Service` (in the repository root's `internal/`), which
  adapts existing LocalOps core packages for the frontend.
