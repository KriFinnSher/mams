# AGENTS

## Task Execution Flow

1. Take one task from the agreed backlog by task ID.
2. Clarify ambiguities only if they block safe implementation.
3. Implement the task as an isolated, minimal change.
4. Validate locally with relevant checks (tests, curl, SQL checks, etc.).
5. Commit with message format:
   `[TASK-ID] <task title>`
6. Do not push until explicit user instruction.
7. Keep infra tasks runnable locally first, so the repo can be cloned to a VM and reused with minimal changes.

## Conventions

- Keep migrations incremental and reversible (`up.sql` + `down.sql`).
- Prefer explicit constraints and indexes early.
- Preserve task isolation: one commit per task whenever feasible.
- Operate only inside this repository. If a task requires actions outside the repo boundary, request explicit access or provide the exact terminal command for the user to run.
- Respect task scope strictly: implement only what is explicitly required by the current task ID, no extra bootstrap or adjacent features.
- Go code style priorities:
- `1)` Laconic code: short, clear names; avoid duplicated checks and over-extraction of tiny one-off helpers.
- `2)` Generality: each function/package should do one logical job; no cross-package leakage or unjustified intermediate structs.
- `3)` Comments only where they add meaning; skip obvious comments.
- `4)` Tests: table-driven style; scenarios should come from product requirements, not implementation trivia.
- Shared helpers should be placed in shared packages (for example, HTTP response helpers in `utils/http.go`), not duplicated inside handlers.
- Interfaces used by handlers/services should be placed in `contract.go` in the same package.
- `contract.go` must include: `//go:generate mockgen -source=contract.go -destination=mocks/contract.go -package=mocks`
- Tests must use generated mocks from `mockgen`, not handwritten stubs.
- For local bootstrap, backend startup may run idempotent DB migrations and minimal technical seed data required for smoke/login checks.
