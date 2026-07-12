---
name: kkt-loop
description: Durable constrained optimization execution loop for long-running coding work. Use when a coding agent should capture user meaning, discover repo constraints and validation paths, build an optimization model and execution contract, create a file-backed workspace, execute with continuation, validate with evidence, re-optimize when new facts invalidate the plan, and continue through goal-mode tools until acceptance criteria or stop conditions are met.
license: Apache-2.0
---

# KKT Loop

Use this skill for long-running, autonomous, or multi-step coding work where progress, evidence, and continuation state must survive across turns. It orchestrates a reusable KKT model contract through a durable execution loop: build that model from a fresh request or import a completed `$kkt-model` workspace.

Read `references/kkt-kernel.md` and `references/state-contract.md` before acting. For fresh planning or re-optimization, also read `references/feature-optimization-model.md` and `references/layered-modeling-methods.md`; for a completed-model import, read the imported `model.md` and `guardrails.json` instead unless repair is needed. Read `references/plan-assimilation.md` only when prior plan text exists, `references/discovery-tooling.md` during fresh discovery, and `references/schemas.md` only when writing or auditing full state, guardrail, or layer-output shapes. If `ultragoal` is available, use its goal-state and continuation semantics as the runtime pattern.

## Core Rule

Complete intake and discovery before modeling. Show the final model and get explicit approval before launching goal mode or implementing. On every continuation, use the CLI to read current state, choose the next task, record progress/evidence, and stop only when acceptance criteria are proven or a stop condition is hit.

## CLI-First Workflow

Use the `kkt` CLI as the workflow control path. The skill owns reasoning policy; the CLI owns workspace creation, current state, task state, criteria, approval, progress, evidence, validation, completion, and `events.jsonl`.

```text
# Fresh request: build the shared model contract in this loop workspace.
kkt start loop "<user request>"
# Completed deep model: import it without repeating discovery or model selection.
kkt loop from-model [model-workspace]
kkt intent --method <goal_anti_goal|why_how|obstacle_questions|pairwise_questions> "<intent frame>"
kkt discovery --method <naive|traceability_matrix|coupling_map|dsm_lite> "<repo facts and constraints>"
kkt model --method <lexicographic|decision_tree|shortest_path|ordinal_mcda|pairwise_ahp|outranking> "<canonical Optimized Plan Contract>"
kkt plan "<execution contract>"
kkt criteria add "<acceptance criterion>"
kkt task add "<task>"
kkt approve "<approved scope>"
kkt hooks arm --mode enforce
kkt next
kkt task start <task-id>
kkt progress "<progress update>"
kkt evidence --for <criterion-id> --command "<validation command>" "<validation evidence>"
kkt task done <task-id>
kkt criteria satisfy <criterion-id>
kkt validate --run
kkt done

# Diagnostics only when blocked or resuming:
kkt resume
kkt status [--json]
kkt judge --checkpoint <checkpoint> --json
kkt replay --check
```

If `kkt` is missing, stop and ask the user to install or upgrade KKT. Do not hand-write replacement state. Use `kkt validate --run` when guardrails list required commands; `kkt evidence` records narrative evidence and criterion mapping, not deterministic command proof.

## Durable State

Create project-root `.kkt/loop/<slug>/` with `kkt start loop "<user request>"` for a fresh model, or use `kkt loop from-model [model-workspace]` to create a loop workspace that imports a completed model's intent, discovery, model, and guardrails. Workspace artifacts are created during planning; implementation, hook arming, goal launch, and continuation remain approval-gated. The workspace starts with approval pending. Use `references/state-contract.md` for file layout, artifact boundaries, and loop-state semantics. Load `references/schemas.md` only when a full YAML or JSON shape is needed.

Loop workspaces use:

- `kkt.yaml` as the current contract.
- Markdown artifacts for rich intent, discovery, model, plan, progress, evidence, and notes.
- `events.jsonl` as append-only history for task transitions, evidence, approval, blockers, validation, and completion. Use it for resume context and replay consistency checks, not as a replacement source of truth for `kkt.yaml`.

## Workflow

1. Check current goal state with `get_goal` if available; do not create a second active goal without explicit user direction.
2. Choose the entry path. For a fresh request, create `kkt start loop` and complete steps 3-8. When the current or supplied workspace is a completed `$kkt-model` workspace, default to `kkt loop from-model [model-workspace]` rather than creating a competing loop model; read its imported artifacts and guardrails, and skip directly to step 9 unless new evidence invalidates the model.
3. Capture intent: user goal, desired behavior, user-visible success, scope boundary, examples, priority signals, and explicit user constraints.
4. If prior plan text exists, assimilate it as untrusted scaffold before modeling: extract signals, classify each claim, verify discoverable facts, and treat unverified claims as assumptions or candidates.
5. Run the interactive intent checkpoint before deep discovery: after any quick inspection needed to avoid asking repo-fact questions, ask 1-3 owner-decision questions when goal, success, scope, risk, or tradeoff preference is still ambiguous. For large, high-risk, or especially ambiguous work, run a short Socratic pass.
6. Apply the owner-decision filter before asking: inspect discoverable facts locally, assume low-risk reversible defaults, ask only for owner decisions, and stop for blocking unknowns.
7. Inspect relevant repo context and validation paths before writing the model. Use `rg` directly for broad text and file discovery, `ast-grep` directly for structural search when syntax matters, `git` for repository state, optional helpers and language-native commands when they provide stronger evidence.
8. Select and record methods, then produce the canonical Optimized Plan Contract from `feature-optimization-model.md` at loop depth. Include enough execution implications to derive tasks, criteria, validation, and continuation policy.
9. Materialize the execution contract before approval: record `plan.md`, acceptance criteria, tasks, required validation evidence, and stop conditions. Fresh and imported loops use this same step; imported loops do not repeat deep modeling.
10. Show the final model and complete execution contract, then get approval with `kkt approve`. It validates the model, guardrails, path bounds, plan, tasks, and criteria internally.
11. When hook adapters are installed, run `kkt hooks arm --mode enforce` after approval. Hooks auto-disarm on `kkt done`, `kkt block`, or execution-contract invalidation.
12. Launch `create_goal` only when goal tools are available, no active goal exists, and the user asked to run now; otherwise output the exact `/goal` command.
13. Before each work segment, run `kkt next` (or `kkt next --json`). It checks replay, stop conditions, approval, and task state, then returns one permitted action. Use `kkt resume` for a fuller diagnostic packet only when blocked or resuming.
14. Start the reported task with `kkt task start`. It enforces continuation and pre-mutation readiness internally.
15. Execute only the current task, update progress/evidence with criterion-linked evidence, update task and criteria state, and run `kkt validate --run` when required commands exist. `criteria satisfy` requires mapped evidence immediately.
16. Run `kkt done`; it performs the finalize check internally.
17. Re-optimize with `kkt model --method <method>` only when new evidence changes feasibility, constraints, or objective fit. A material model, guardrail, plan, task, or criterion change invalidates approval; get approval again.

## Goal Objective Template

```text
Execute the KKT workspace at the project root's `.kkt/loop/<slug>/plan.md`. Follow `kkt.yaml`, `model.md`, `guardrails.json`, `plan.md`, progress, evidence, and events. Use `kkt next` as the continuation control surface, `kkt task start` as the mutation gate, and `kkt done` as the finalize gate. Use `kkt resume`, `kkt status`, `kkt judge`, or `kkt replay` only to diagnose a reported block. Re-optimize only when evidence changes feasibility, and stop for a blocked transition, listed stop condition, proven acceptance criteria, or explicit user input.
```

Do not set a token budget unless the user explicitly provides one.

## Stop Conditions

Stop before continuing when approval is missing, no feasible plan satisfies hard constraints, a destructive action is required, credentials/secrets/external access/paid services are required, a listed stop condition is active, a task or criterion is blocked, or continuing expands scope.

## Do Not

- Do not use workspace files as a substitute for understanding the request.
- Do not begin implementation, arm hooks, launch goal mode, or continue autonomous execution before approval. Workspace files are required for planning and may be created before approval.
- Do not create multiple workspaces for the same objective.
- Do not include secrets or credentials in workspace files.
- Do not continue past a stop condition or blocked criterion.
- Do not re-optimize on preference alone; re-optimize only when evidence changes feasibility, constraints, or objective fit.
