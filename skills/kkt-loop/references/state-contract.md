# KKT State Contract

Use this reference only when workflow state must survive across layers, turns, or agents. The CLI is the canonical interface for creating and mutating durable state.

## Persistence Tiers

| Tier | Skill | Durable state | Use when |
| --- | --- | --- | --- |
| Plan | `kkt` | none by default; optional `.kkt/kkt.yaml` | lightweight handoff, resume, or evidence. |
| Model | `kkt-model` | `.kkt/model/<slug>/` | reusable deep model before execution. |
| Run | `kkt-run` | `.kkt/run/<slug>/` | bounded execution of a completed model. |
| Loop | `kkt-loop` | `.kkt/loop/<slug>/` plus `events.jsonl` | long-running or resumable execution. |

Durable paths are rooted at the nearest Git/worktree root; outside Git, use the current directory.

## Artifact Boundaries

- `intent.md`: user meaning and unresolved meaning questions.
- `discovery.md`: repo facts, constraints, validation paths, and unknowns.
- `model.md`: the canonical Optimized Plan Contract from `feature-optimization-model.md`.
- `guardrails.json`: machine-readable scope, constraints, validation requirements, and drift policy.
- `plan.md`: execution contract: ordered steps, criteria, validation, evidence, stops, and continuation policy.
- `progress.md`, `evidence.md`, `notes.md`: execution record, proof, and observations.
- `events.jsonl`: loop-only append history; never the competing current-state source.

Run and loop approval requires a complete model, valid guardrails, and `plan.md`; loops also require tasks and acceptance criteria.

## Handoff Rules

1. Read `kkt.yaml`, then the prior layer artifact.
2. Update only the current layer unless repairing a stale reference.
3. Append decisions; do not erase prior rationale.
4. Set next-layer readiness and record the smallest blocker.
5. If evidence invalidates an earlier layer, re-open it instead of silently continuing.

## Judge Checkpoints

- `model-ready`: model, guardrails, path bounds, and execution contract are complete.
- `pre-mutation`: approval and path scope permit the change.
- `continuation`: loop replay and stop conditions permit another segment.
- `finalize`: validation and path scope permit completion.

`block` is a hard stop; repair `warn` before risky work.
