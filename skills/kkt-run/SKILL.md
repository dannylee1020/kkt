---
name: kkt-run
description: Execute an existing KKT model with deterministic drift guardrails. Use when a coding agent should pick up a completed kkt-model workspace, preserve the optimized model and guardrail contract, check model-readiness before mutation, implement the selected plan, validate with evidence, and finish without the long-running kkt-loop continuation machinery.
license: Apache-2.0
---

# KKT Run

Use this skill when modeling is already complete and the user wants implementation now. It is the execution tier between `$kkt-model` and `$kkt-loop`: richer than a compact plan, but lighter than a durable long-running loop.

Read `references/kkt-kernel.md` and `references/state-contract.md` before acting. Read `references/feature-optimization-model.md` when the completed compressed model must be interpreted, and `references/deep-optimization-model.md` when the completed deep model must be interpreted, repaired, or checked for drift. Read `references/plan-assimilation.md` only when the selected model used prior-plan assimilation. Read `references/discovery-tooling.md` only when verifying drift or missing facts. Read `references/layered-modeling-methods.md` only when reopening an incomplete model. Read `references/schemas.md` only when auditing full state or guardrail shapes.

## Core Rule

Run only the selected model. Before edits, load the model workspace, materialize the bounded execution plan, show the complete execution contract, and get explicit approval. `approve`, `next`, and `done` enforce model readiness, mutation readiness, and finalization internally. During implementation, repair a blocked transition rather than manually sequencing checkpoints.

## CLI-First Workflow

Use the `kkt` CLI as the workflow control path. The skill owns implementation judgment; the CLI owns deterministic workspace creation, guardrail state, checkpoint results, evidence, validation, and completion.

```text
kkt run from-model [model-workspace]
kkt show model
kkt plan "<execution contract>"
kkt approve "<approved scope>"
kkt hooks arm --mode enforce
kkt next
kkt progress "<progress update>"
kkt evidence --command "<validation command>" "<validation evidence>"
kkt validate --run
kkt done
```

If no completed compressed or deep model workspace exists, switch back to `$kkt` or `$kkt-model` instead of inventing one. If the work needs continuation, autonomous execution, multiple resumptions, task queues, or event replay, start from the selected model with `$kkt-loop` / `kkt loop from-model [model-workspace]` rather than adding loop machinery to a run workspace.

## Workflow

1. Resolve or create the run workspace with `kkt run from-model [model-workspace]`.
2. Read `kkt show model` and `kkt show guardrails`; confirm whether the selected model used plan assimilation without promoting unverified assumptions.
3. Record `kkt plan` before approval. It must name execution steps, validation, evidence, and stop conditions derived from the selected model.
4. Show the user the complete execution contract: selected model, plan, hard constraints, paths, validation commands, and residual risk.
5. Get explicit approval with `kkt approve`. It enforces model, guardrail, path-bound, and execution-contract readiness internally.
6. When hook adapters are installed, run `kkt hooks arm --mode enforce` after approval. Hooks auto-disarm on `kkt done`, `kkt block`, or contract invalidation.
7. Run `kkt next` before implementation. It is the authoritative readiness check; repair its reported block rather than manually sequencing judges.
8. Implement the smallest change that satisfies the selected model. Do not expand scope, add unrelated cleanup, or change the model unless new evidence invalidates feasibility. A material model, guardrail, or plan change invalidates approval and requires a new approval.
9. Record progress and validation evidence with CLI commands. Use `kkt evidence` for narrative evidence, not deterministic command proof.
10. Run `kkt validate --run` when guardrails list required commands, then `kkt done`; `done` performs the finalize check internally.

## Stop Conditions

Stop before editing when approval or `next` blocks, guardrails are missing or invalid for the intended change, hook baseline checks show post-approval mutations outside allowed paths, changed files hit blocked paths, the selected model no longer matches repo facts, destructive action is required, credentials/secrets/external access/paid services are required, or implementation would expand beyond the model.

## Do Not

- Do not use `$kkt-run` as a substitute for modeling.
- Do not execute an incomplete or blocked model workspace.
- Do not continue past a blocked transition; use diagnostics to repair it.
- Do not create loop tasks or rely on `events.jsonl`; use `$kkt-loop` for continuation.
- Do not silently change the selected model during implementation.
