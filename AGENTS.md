# AGENTS.md

## Purpose

- This file is the cross-agent source of truth for repository-wide instructions.
- When a tool also supports its own memory file, keep tool-specific wrappers minimal and defer shared rules to this file.

## Reference

- Read `spec.md` and `plan.md` first when requirements are unclear.
- Use `tasks.md` as the implementation checklist.
- Use `guide/architecture/backend.md` for backend architecture rationale and dependency decisions.
- Use `guide/architecture/frontend.md` for frontend architecture rationale and dependency decisions.

## Code Style

### API

- All APIs must be implemented in compliance with the OpenAPI Specification.
- The API specification must be maintained in `openapi.yml` using the standard OpenAPI format.

### Frontend Type Generation

- `openapi.yml` is the single source of truth for frontend API types.
- Frontend types must never be written by hand based on backend source code.
- Run `npm run generate` inside `frontend/` whenever `openapi.yml` changes to regenerate `src/types/schema.ts`.
- `src/types/schema.ts` is auto-generated. Never edit it manually.
- `src/types/*.ts` files may re-export or derive types from `schema.ts`, but must not duplicate type definitions that belong in `openapi.yml`.
- Preferred workflow: backend implementation, then `openapi.yml` update, then `npm run generate`, then frontend implementation.

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

## Documentation

- After completing each phase task, write a study document that explains what was built and why.
- The target audience is a developer with Java/Spring experience but no Go or TypeScript/React background.
- Explain design choices by mapping them to familiar Java/Spring concepts where possible.
- Focus on why, not just what. Code listings alone are not enough.
- File location: `guide/phase/phase_{N}_{M}_{backend|frontend}.md`
- Example paths: `guide/phase/phase_1_3_backend.md`, `guide/phase/phase_1_4_backend.md`
- Include the study document in the same commit as the implementation.

## Commit

- Follow the Conventional Commits specification.
- Do not create commits automatically.
- Perform commits only after review is completed and the user explicitly requests a commit.

```text
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

- `fix`: patches a bug in the codebase.
- `feat`: introduces a new feature to the codebase.
- `BREAKING CHANGE`: use a `BREAKING CHANGE:` footer, or append `!` after the type or scope, when introducing a breaking API change.
- Other common types: `build`, `chore`, `ci`, `docs`, `style`, `refactor`, `perf`, `test`, `revert`.
