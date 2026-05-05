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
