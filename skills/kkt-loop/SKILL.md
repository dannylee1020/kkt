---
name: kkt-loop
description: Durable constrained optimization execution loop for long-running coding work. Use when a coding agent should capture user meaning, discover repo constraints and validation paths, build an optimization model and execution contract, create a file-backed workspace, execute with continuation, validate with evidence, re-optimize when new facts invalidate the plan, and continue through goal-mode tools until acceptance criteria or stop conditions are met.
license: Apache-2.0
---

# KKT Loop

Use this skill for long-running, autonomous, or multi-step coding work where progress, evidence, and continuation state must survive across turns. It extends the KKT model into a durable execution loop.

Read `references/kkt-kernel.md`, `references/feature-optimization-model.md`, `references/layered-modeling-methods.md`, and `references/state-contract.md` before acting. Read `references/plan-assimilation.md` only when prior plan text exists. Read `references/discovery-tooling.md` during discovery. Read `references/schemas.md` only when writing or auditing full state, guardrail, or layer-output shapes. If `ultragoal` is available, use its goal-state and continuation semantics as the runtime pattern.

## Core Rule

Complete intake and discovery before modeling. Show the final model and get explicit approval before launching goal mode or implementing. On every continuation, use the CLI to read current state, choose the next task, record progress/evidence, and stop only when acceptance criteria are proven or a stop condition is hit.

## CLI-First Workflow

Use the `kkt` CLI as the workflow control path. The skill owns reasoning policy; the CLI owns workspace creation, current state, task state, criteria, approval, progress, evidence, validation, completion, and `events.jsonl`.

```text
kkt start loop "<user request>"
kkt status [--json]
kkt next
kkt intent --method <goal_anti_goal|why_how|obstacle_questions|pairwise_questions> "<intent frame>"
kkt discovery --method <naive|traceability_matrix|coupling_map|dsm_lite> "<repo facts and constraints>"
kkt model --method <lexicographic|decision_tree|shortest_path|ordinal_mcda|pairwise_ahp|outranking> "<objective_function, files_to_modify, constraint_functions, decision_variables, validation_proof, and selected model>"
kkt guardrails validate
kkt judge --checkpoint model-ready --json
kkt plan "<execution contract>"
kkt criteria add "<acceptance criterion>"
kkt task add "<task>"
kkt approve "<approved scope>"
kkt hooks arm --mode enforce
kkt judge --checkpoint pre-mutation --json
kkt task start <task-id>
kkt progress "<progress update>"
kkt evidence --for <criterion-id> --command "<validation command>" "<validation evidence>"
kkt task done <task-id>
kkt criteria satisfy <criterion-id>
kkt validate --run
kkt judge --checkpoint continuation --json
kkt judge --checkpoint finalize --json
kkt done
kkt resume
kkt replay --check
```

If `kkt` is missing, stop and ask the user to install or upgrade KKT. Do not hand-write replacement state. Use `kkt validate --run` when guardrails list required commands; `kkt evidence` records narrative evidence and criterion mapping, not deterministic command proof.

## Durable State

Create project-root `.kkt/loop/<slug>/` with `kkt start loop "<user request>"` so intent, discovery, model, guardrails, and approval state are durable. The workspace starts with approval pending; do not execute or launch continuation until approval is recorded. Use `references/state-contract.md` for file layout, artifact boundaries, and loop-state semantics. Load `references/schemas.md` only when a full YAML or JSON shape is needed.

Loop workspaces use:

- `kkt.yaml` as the current contract.
- Markdown artifacts for rich intent, discovery, model, plan, progress, evidence, and notes.
- `events.jsonl` as append-only history for task transitions, evidence, approval, blockers, validation, and completion. Use it for resume context and replay consistency checks, not as a replacement source of truth for `kkt.yaml`.

## Workflow

1. Check current goal state with `get_goal` if available; do not create a second active goal without explicit user direction.
2. Capture intent: user goal, desired behavior, user-visible success, scope boundary, examples, priority signals, and explicit user constraints.
3. If prior plan text exists, assimilate it as untrusted scaffold before modeling: extract signals, classify each claim, verify discoverable facts, and treat unverified claims as assumptions or candidates.
4. Run the interactive intent checkpoint before deep discovery: after any quick inspection needed to avoid asking repo-fact questions, ask 1-3 owner-decision questions when goal, success, scope, risk, or tradeoff preference is still ambiguous. For large, high-risk, or especially ambiguous work, run a short Socratic pass.
5. Apply the owner-decision filter before asking: inspect discoverable facts locally, assume low-risk reversible defaults, ask only for owner decisions, and stop for blocking unknowns.
6. Inspect relevant repo context and validation paths before writing the model. Use `rg` directly for broad text and file discovery, `ast-grep` directly for structural search when syntax matters, `git` for repository state, optional helpers and language-native commands when they provide stronger evidence.
7. Select one intent method, one discovery method, and one modeling method from the layered catalog; record each with the matching `kkt ... --method` command. When no specialized method fits, use the fallback set (`goal_anti_goal`, `traceability_matrix`, `lexicographic`) and record why the fallback is sufficient instead of forcing an advanced method.
8. Build the optimization model and execution contract from intent and discovery using the loop profile. The pre-approval output must include objective function, known constraints, files to modify or affected surfaces, constraint functions, decision variables, candidate feasibility, selected plan, binding constraints, validation proof plan, execution implications, residual risk, acceptance criteria, evidence required, guardrail constraints, allowed paths, blocked paths, and stop conditions.
9. Run `kkt guardrails validate` and `kkt judge --checkpoint model-ready --json`; repair or stop on any blocking result.
10. Show the final model and wait for explicit approval.
11. After approval, record the plan with CLI commands, add criteria/tasks, and record approval.
12. When hook adapters are installed, run `kkt hooks arm --mode enforce` after approval to enable project-local deterministic hook enforcement for this workspace. Hooks are off by default and auto-disarm on `kkt done` or `kkt block`.
13. Launch `create_goal` only when goal tools are available, no active goal exists, and the user asked to run now; otherwise output the exact `/goal` command.
14. Before each work segment, run `kkt status --json`, `kkt next`, and `kkt judge --checkpoint continuation --json`; use `kkt next --json` when a machine-readable next action helps; inspect `kkt show state`, `kkt show progress`, and `kkt show evidence` as needed.
15. Before modifying files or running side-effecting tools, run `kkt judge --checkpoint pre-mutation --json`; it blocks missing approval and explicitly blocked-path drift while ignoring unrelated unchanged branch work outside `allowed_paths`. When hooks are armed, hook baseline checks additionally block new post-approval mutations outside `allowed_paths`.
16. Execute only the current or CLI-reported next task, update progress/evidence with criterion-linked evidence, update task and criteria state, and run `kkt validate --run` when required commands exist.
17. Run `kkt judge --checkpoint finalize --json` before `kkt done`.
18. Re-optimize with `kkt model --method <method>` only when new evidence changes feasibility, constraints, or objective fit.

## Goal Objective Template

```text
Execute the KKT workspace at the project root's .kkt/loop/<slug>/plan.md. Follow kkt.yaml, intent.md, discovery.md, model.md, guardrails.json, plan.md, progress.md, evidence.md, notes.md, and events.jsonl. Use kkt status --json, kkt next, kkt judge, kkt hooks arm when hook adapters are installed, kkt task, kkt progress, kkt evidence, kkt criteria, kkt validate --run when required commands exist, and kkt done as the workflow control surface. Re-read state and run the continuation judge before each continuation, re-optimize only when evidence changes feasibility, and stop only for blocking judge results, listed stop conditions, proven acceptance criteria, or explicit user input.
```

Do not set a token budget unless the user explicitly provides one.

## Stop Conditions

Stop before continuing when approval is missing, no feasible plan satisfies hard constraints, a destructive action is required, credentials/secrets/external access/paid services are required, a listed stop condition is active, a task or criterion is blocked, or continuing expands scope.

## Do Not

- Do not use workspace files as a substitute for understanding the request.
- Do not create workspace files, launch goal mode, or begin implementation before approval.
- Do not create multiple workspaces for the same objective.
- Do not include secrets or credentials in workspace files.
- Do not continue past a stop condition or blocked criterion.
- Do not re-optimize on preference alone; re-optimize only when evidence changes feasibility, constraints, or objective fit.
