---
name: kkt-run
description: Execute an existing KKT model with deterministic drift guardrails. Use when a coding agent should pick up a completed kkt-model workspace, preserve the optimized model and guardrail contract, check model-readiness before mutation, implement the selected plan, validate with evidence, and finish without the long-running kkt-loop continuation machinery.
license: Apache-2.0
---

# KKT Run

Use this skill when modeling is already complete and the user wants implementation now. It is the execution tier between `$kkt-model` and `$kkt-loop`: richer than a compact plan, but lighter than a durable long-running loop.

Read `references/kkt-kernel.md` and `references/state-contract.md` before acting. Read `references/feature-optimization-model.md` only when the completed model must be interpreted, repaired, or checked for drift. Read `references/plan-assimilation.md` only when the selected model used prior-plan assimilation. Read `references/discovery-tooling.md` only when verifying drift or missing facts. Read `references/layered-modeling-methods.md` only when reopening an incomplete model. Read `references/schemas.md` only when auditing full state or guardrail shapes.

## Core Rule

Run only the selected model. Before edits, load the model workspace, validate the guardrail contract, run the model-ready judge checkpoint, show the execution contract, and get explicit approval. During implementation, use the pre-mutation and finalize checkpoints to catch drift before changing files or finishing.

## CLI-First Workflow

Use the `kkt` CLI as the workflow control path. The skill owns implementation judgment; the CLI owns deterministic workspace creation, guardrail state, checkpoint results, evidence, validation, and completion.

```text
kkt run from-model [model-workspace]
kkt status [--json]
kkt show model
kkt guardrails validate
kkt judge --checkpoint model-ready --json
kkt approve "<approved scope>"
kkt hooks arm --mode enforce
kkt judge --checkpoint pre-mutation --json
kkt plan "<execution contract>"
kkt progress "<progress update>"
kkt evidence --command "<validation command>" "<validation evidence>"
kkt validate --run
kkt judge --checkpoint finalize --json
kkt done
```

If no completed model workspace exists, switch back to `$kkt-model` instead of inventing one. If the work needs continuation, autonomous execution, multiple resumptions, task queues, or event replay, switch to `$kkt-loop`.

## Workflow

1. Resolve or create the run workspace with `kkt run from-model [model-workspace]`.
2. Read `kkt status --json`, `kkt show model`, `kkt show guardrails`, and `kkt guardrails validate`.
3. Run `kkt judge --checkpoint model-ready --json`. Treat `block` as a hard stop, `warn` as a contract-quality issue to repair before risky edits, and `allow` as permission to seek approval. This checkpoint blocks if modeled constraints or allowed path bounds are missing.
4. Confirm whether the selected model used plan assimilation. Preserve the model's classification of prior-plan claims and do not promote unverified prior-plan assumptions during execution.
5. Show the user the execution contract: selected model, hard constraints, allowed paths, blocked paths, validation commands, and residual risk.
6. Get explicit approval and record it with `kkt approve`.
7. When hook adapters are installed, run `kkt hooks arm --mode enforce` after approval to enable project-local deterministic hook enforcement for this workspace. Hooks are off by default and auto-disarm on `kkt done` or `kkt block`.
8. Before modifying files, run `kkt judge --checkpoint pre-mutation --json`; it blocks missing approval and explicitly blocked-path drift while ignoring unrelated unchanged branch work outside `allowed_paths`. When hooks are armed, hook baseline checks additionally block new post-approval mutations outside `allowed_paths`.
9. Implement the smallest change that satisfies the selected model. Do not expand scope, add unrelated cleanup, or change the model unless new evidence invalidates feasibility.
10. Record progress and validation evidence with CLI commands. Use `kkt evidence` for narrative evidence, not as deterministic command proof.
11. Run `kkt validate --run` when guardrails list required commands, then run `kkt judge --checkpoint finalize --json` before `kkt done`.

## Stop Conditions

Stop before editing when the model-ready checkpoint blocks, guardrails are missing or invalid for the intended change, approval is missing, hook baseline checks show post-approval mutations outside allowed paths, changed files hit blocked paths, the selected model no longer matches repo facts, destructive action is required, credentials/secrets/external access/paid services are required, or implementation would expand beyond the model.

## Do Not

- Do not use `$kkt-run` as a substitute for modeling.
- Do not execute an incomplete or blocked model workspace.
- Do not continue past a blocking judge result.
- Do not create loop tasks or rely on `events.jsonl`; use `$kkt-loop` for continuation.
- Do not silently change the selected model during implementation.
