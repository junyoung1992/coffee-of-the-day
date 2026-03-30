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
  spec.md                               # global service spec
  openapi.yml                           # global API spec
  arch/                                 # global architecture docs (updated after each merge)
  lang/                                 # language study guides
  issues/                               # per-issue development artifacts
    initial/                            # initial build (phases 1-4) archived
    {issue-number}-{title}/             # created per GitHub issue
      plan.md
      tasks.md
      review/
      guide/
  postmortems/                          # incident and debugging reports (cross-issue)
```

## Reference

Read these files only when relevant to the task at hand — do not read them preemptively.

### Global (maintained across all issues)

- `docs/spec.md`: consult when overall service requirements are unclear
- `docs/openapi.yml`: the single source of truth for the API contract
- `docs/arch/backend.md`: consult for backend architecture rationale and dependency decisions
- `docs/arch/frontend.md`: consult for frontend architecture rationale and dependency decisions
- `docs/backlog.md`: open work — security, testing, technical debt, and features; consult when the issue is related to backlog items

### Issue-scoped (read from the active issue's directory)

- `docs/issues/{issue}/plan.md`: consult when requirements for the current issue are unclear
- `docs/issues/{issue}/tasks.md`: use as an implementation checklist for the current issue
- `docs/issues/{issue}/review/`: consult when doing code review or refactoring

### For learning (not needed during development)

- `docs/lang/go.md`, `docs/lang/typescript.md`: language-specific study guides for the user

## Code Style

### API

- All APIs must be implemented in compliance with the OpenAPI Specification.
- The API specification must be maintained in `docs/openapi.yml` using the standard OpenAPI format.

### Frontend Type Generation

- `docs/openapi.yml` is the single source of truth for frontend API types.
- Frontend types must never be written by hand based on backend source code.
- Run `npm run generate` inside `frontend/` whenever `docs/openapi.yml` changes to regenerate `src/types/schema.ts`.
- `src/types/schema.ts` is auto-generated. Never edit it manually.
- `src/types/*.ts` files may re-export or derive types from `schema.ts`, but must not duplicate type definitions that belong in `docs/openapi.yml`.
- Preferred workflow: backend implementation → `docs/openapi.yml` update → `npm run generate` → frontend implementation.

### Comments

- Write explanatory comments in Korean.
- Keep technical terms, domain terminology, method names, and identifiers in their original language when translation would reduce clarity.
- If the business logic flow cannot be clearly understood from method names alone, add comments that explain the logic flow.
- Avoid unnecessary comments that only restate obvious code.
- Prefer comments that explain why over comments that explain what.

## Test

### Backend

- Always write tests when adding or modifying functionality.
- Domain logic and feature-level modules must be covered by unit tests.
- Service-level workflows, external integrations, and end-to-end business flows should be covered by integration tests.
- Use unit tests by default unless the test scope requires interaction between multiple components.
- Do not modify failing existing tests without first analyzing the root cause.

### Frontend

- Always write tests when adding or modifying functionality.
- UI logic, state management, and reusable modules must be covered by unit tests.
- User flows, component interactions, and API integration scenarios should be covered by integration or end-to-end tests.
- Prefer behavior-focused tests based on user interactions rather than implementation details.
- Do not modify failing existing tests without first analyzing the root cause.
- Critical user journeys must be validated with end-to-end tests.

Available test scripts (run inside `frontend/`):

| Script | Command | When to run |
|--------|---------|-------------|
| Unit tests | `npm test` | After any component, hook, or utility change |
| E2E tests | `npm run test:e2e` | Run locally before creating a PR (not included in CI) |
| Type generation | `npm run generate` | After `docs/openapi.yml` changes |

## Documentation

- After merging an issue, update `docs/arch/` if the architecture or key design decisions have changed.
- `docs/spec.md` and `docs/openapi.yml` must always reflect the actual application state.
  - If `docs/spec.md` needs to change, propose the change to the user and get approval before starting development.

## Commit

- Follow the Conventional Commits specification.
- Do not create commits automatically. Commit only after review is completed and the user explicitly requests it.
