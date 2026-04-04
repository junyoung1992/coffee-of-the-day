# AGENTS.md

## Purpose

This file is the cross-agent source of truth for repository-wide instructions.
When a tool also supports its own memory file, keep tool-specific wrappers minimal and defer shared rules to this file.

## Project Context

Coffee of the Day — a personal coffee journal app (POC, single-user).

- **Backend**: Go + chi + sqlc + SQLite (`backend/`)
- **Frontend**: React + TypeScript + Vite + TanStack Query + Tailwind CSS v4 (`frontend/`)

### Directory Structure

```
backend/
  cmd/server/main.go                    # entrypoint
  internal/{handler,service,repository,domain}/
  db/{migrations,queries}/
  config/
frontend/
  src/{pages,components,api,types,hooks}/
  src/types/schema.ts                   # auto-generated from docs/openapi.yml — never edit manually
docs/
  spec.md                               # service spec — consult when requirements are unclear
  openapi.yml                           # API contract (single source of truth)
  arch/                                 # architecture docs (updated before PR creation)
    backend.md                          # consult for backend architecture rationale and dependency decisions
    frontend.md                         # consult for frontend architecture rationale and dependency decisions
  backlog.md                            # open work — consult when the issue is related to backlog items
  study/                                # learning guides (not needed during development)
  issues/                               # per-issue development artifacts
    initial/                            # initial build (phases 1-4) archived
    {issue-number}-{title}/             # created per GitHub issue
      plan.md                           # consult when issue requirements are unclear
      tasks.md                          # implementation checklist for the issue
      review/                           # (optional) consult when doing code review or refactoring
      guide/                            # (optional)
  postmortems/                          # incident and debugging reports (cross-issue)
```

## Workflow

### Common

- All APIs must comply with the OpenAPI spec; maintain in `docs/openapi.yml`.
- Always write tests when adding or modifying functionality. Do not modify failing tests without first analyzing the root cause.
- Comments in Korean. Keep technical terms in original language.
- `docs/spec.md` and `docs/openapi.yml` must always reflect actual application state.
  - If `docs/spec.md` needs to change, propose the change to the user and get approval before starting development.
- Before creating a PR, finalize documentation:
  - Update `docs/arch/` if architecture or key design decisions changed.
  - Review `docs/spec.md` to ensure it reflects the current application state.
- Follow the Conventional Commits specification. Always include a description body, not just the subject line. Commit messages must be written in English.
- Do not create commits automatically. Commit only after review is completed and the user explicitly requests it.
- PRs are **squash-merged** into main. After merge, delete the feature branch (both remote and local).

### Backend

- Domain logic and feature-level modules → unit tests. Service-level workflows and integrations → integration tests.
- Use unit tests by default unless the scope requires interaction between multiple components.

### Frontend

- Preferred workflow: backend → `docs/openapi.yml` update → `npm run generate` → frontend implementation.
- `src/types/*.ts` may re-export or derive types from `schema.ts`, but must not duplicate definitions that belong in `docs/openapi.yml`.
- UI logic, state, reusable modules → unit tests. User flows, component interactions → integration/E2E tests.
- Critical user journeys must have E2E tests.

| Script | Command | When to run |
|--------|---------|-------------|
| Unit tests | `npm test` | After any component, hook, or utility change |
| E2E tests | `npm run test:e2e` | Run locally before creating a PR (not in CI) |
| Type generation | `npm run generate` | After `docs/openapi.yml` changes |
