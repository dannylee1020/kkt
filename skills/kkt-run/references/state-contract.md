# KKT State Contract

Use this when KKT state must survive across layers, turns, or coding agents. The goal is not to store everything in YAML; it is to make handoff state explicit, inspectable, and hard to confuse.

Load `schemas.md` only when a full `kkt.yaml`, layer output, or `guardrails.json` shape is needed.

## Persistence Tiers

| Tier | Skill | Durable files | Use when |
| --- | --- | --- | --- |
| Plan | `kkt` | none by default; optional `.kkt/kkt.yaml` | ordinary work where Markdown artifacts would be overhead. |
| Model | `kkt-model` | `.kkt/model/<slug>/kkt.yaml`, `intent.md`, `discovery.md`, `model.md`, `guardrails.json` | durable model or decision brief before execution. |
| Run | `kkt-run` | `.kkt/run/<slug>/kkt.yaml`, imported artifacts, `guardrails.json`, `plan.md`, `progress.md`, `evidence.md`, `notes.md` | completed model should be implemented now without loop state. |
| Loop | `kkt-loop` | `.kkt/loop/<slug>/kkt.yaml`, layer artifacts, `guardrails.json`, `plan.md`, `progress.md`, `evidence.md`, `notes.md`, `events.jsonl` | long-running or resumable execution. |

Durable `.kkt/` paths are rooted at the nearest Git/worktree root. Outside Git, the CLI falls back to the current directory.

## CLI Ownership

The `kkt` CLI is the canonical mutation interface for KKT state. Skills should use CLI commands for workspace creation, state reads, artifact recording, guardrails, approvals, progress, evidence, validation, and completion.

- `kkt.yaml`: current status, active layer, method choices, decisions, artifact references, approvals, stop conditions, and summaries.
- Markdown artifacts: detailed intent, discovery, modeling rationale, execution plan, progress, evidence, and notes.
- `guardrails.json`: modeled constraints, allowed paths, blocked paths, validation requirements, and drift policy.
- `events.jsonl`: loop-only append history for task transitions, evidence, approvals, blockers, validation, and completion.

Do not hand-edit `kkt.yaml` as the primary workflow operation when a CLI command exists.

## Useful Commands

```text
kkt start plan "<request>"
kkt start model "<request>"
kkt run from-model [model-workspace]
kkt start run "<request>"
kkt start loop "<request>"
kkt intent|discovery|model --method <method> "<layer output>"
kkt guardrails show|set|validate
kkt judge --checkpoint model-ready|pre-mutation|continuation|finalize --json
kkt validate
kkt done
```

## Artifact Boundaries

- `intent.md`: user meaning, success, scope, examples, priority signals, explicit constraints, unresolved meaning questions.
- `discovery.md`: files, symbols, components, workflows, discovered constraints, validation paths, coupling, evidence, confidence, unknowns.
- `model.md`: method selection, objective, constraints, decision variables, candidates, feasibility, selected plan, binding constraints, validation plan, sensitivity, execution implications, residual risk.
- `guardrails.json`: machine-readable drift contract; run and loop execution must not proceed when modeled constraints or allowed paths are empty.
- `plan.md`: execution tasks, acceptance criteria, validation plan, evidence required, stop conditions, continuation policy.
- `progress.md`: work log, progress narrative, blocker notes.
- `evidence.md`: validation map, command outputs, artifacts, final certificate.
- `notes.md`: observations, assumptions, open questions, deferred ideas.
- `events.jsonl`: append-only loop event history, not a competing source of truth for current state.

## Handoff Rules

1. Read `kkt.yaml` first when it exists.
2. Read the prior layer's artifact before acting.
3. Update only the current layer unless repairing a stale reference.
4. Append decisions instead of overwriting prior rationale.
5. Set `next_layer_readiness` before handing off.
6. Record blockers with the smallest user or system change that would unblock.
7. If new evidence invalidates an earlier layer, mark the active layer blocked and re-open the earlier layer instead of silently continuing.

## Judge Checkpoints

- `model-ready`: before implementation; blocks run/loop execution when model, guardrails, or allowed path bounds are incomplete.
- `pre-mutation`: before edits or side effects; blocks when approval is missing or changed paths violate bounds.
- `continuation`: before loop continuation; blocks on active stop conditions or replay drift.
- `finalize`: before `kkt done`; blocks when validation fails or current git changes violate path bounds.
- `pre-tool`, `post-tool`, `pre-compact`, `post-compact`: portable hook names for adapters.

Treat `block` as a hard stop. Treat `warn` as a contract-quality issue to repair before risky work. Treat `allow` as permission to continue to the next workflow step.
