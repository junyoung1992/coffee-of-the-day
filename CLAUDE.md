# CLAUDE.md

## CODE STYLE

### API
- All APIs must be implemented in compliance with the OpenAPI Specification.
- The API specification must be maintained in openapi.yml using the standard OpenAPI format.

### COMMENTS
- Write explanatory comments in Korean.
- Keep technical terms, domain terminology, method names, and identifiers in their original language when translation would reduce clarity.
- If the business logic flow cannot be clearly understood from method names alone, comments explaining the logic flow must be added.
- Avoid unnecessary comments that only describe obvious or self-explanatory code.
- Use comments to explain why, not what, whenever possible.

## TEST

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

## COMMIT

- Follow the Conventional Commits specification.
- Do not create commits automatically.
- All commits must be performed only after review is completed and upon explicit user request.

```
<type>[optional scope]: <description>

[optional body]

[optional footer(s)]
```

- fix: a commit of the type fix patches a bug in your codebase
- feat: a commit of the type feat introduces a new feature to the codebase
- BREAKING CHANGE: a commit that has a footer BREAKING CHANGE:, or appends a ! after the type/scope, introduces a breaking API change
- others: build:, chore:, ci:, docs:, style:, refactor:, perf:, test:, revert:

## STUDY

- After completing each phase task, write a study document explaining what was built and why.
- The target audience is a developer with Java/Spring experience but no Go or TypeScript/React background.
- Explain design choices by mapping them to familiar Java/Spring concepts where possible.
- Focus on *why*, not just *what* — code listings alone are not enough.
- File location: `guide/phase/phase_{N}_{M}_{backend|frontend}.md`
  - Example: `study/phase_1_3_backend.md`, `study/phase_1_4_backend.md`
- Include the study document in the same commit as the implementation.

## REFERENCE

Read spec.md first when requirements are unclear.

- spec.md: functional and business specification
- plan.md: development plan
- tasks.md: implementation task checklist
- guide/architecture/backend.md: backend architecture rationale and dependency decisions
- guide/architecture/frontend.md: frontend architecture rationale and dependency decisions
