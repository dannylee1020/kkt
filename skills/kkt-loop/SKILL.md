---
name: kkt-loop
description: Durable constrained optimization execution loop for long-running coding work. Use when a coding agent should translate rough user input into a durable request frame, infer missing non-goals, constraints, validation expectations, and execution mode, build an optimization model and execution contract, create a file-backed workspace, execute with continuation, validate with evidence, re-optimize when new facts invalidate the plan, and continue through goal-mode tools until acceptance criteria or stop conditions are met.
license: Apache-2.0
---

# KKT Loop

Use this skill for long-running, autonomous, or multi-step coding work where progress and evidence must survive continuation. Apply the kkt optimization model and execution contract, then use an `ultragoal`-style durable workspace for execution.

Read `references/feature-optimization-model.md` before acting. If the `ultragoal` skill is available, use its goal-state and continuation semantics as the runtime pattern.

## Core Rule

Complete request intake before modeling. Create durable state before launching a loop. On every continuation, re-read the workspace, update progress and evidence, and stop only when the acceptance criteria are proven or a stop condition is hit.

## Workspace Location

Create files under the current repo:

```text
.kkt/<slug>/
```

Use a short filesystem-safe slug derived from the objective.

## Workspace Files

Always create:

```text
.kkt/<slug>/model.md
.kkt/<slug>/plan.md
.kkt/<slug>/evidence.md
.kkt/<slug>/progress.md
.kkt/<slug>/notes.md
```

- `model.md`: optimization model, candidate plans, selected plan, sensitivity notes.
- `plan.md`: execution contract, tasks, acceptance criteria, validation plan, stop conditions, continuation policy.
- `evidence.md`: validation map, command outputs, artifacts, and proof log.
- `progress.md`: current status, active task, task list, work log, blockers.
- `notes.md`: observations, assumptions, open questions, deferred ideas.

## Workflow

1. Check current goal state with `get_goal` if available. Do not create a second active goal without explicit user direction.
2. Translate the user's rough input into a request frame: feature or problem, known non-goals, hard constraints, validation expectations, and execution mode.
3. Inspect relevant repo context to infer discoverable intake fields before writing the model.
4. Ask only when an unresolved field materially changes feasibility, product behavior, risk, or execution mode.
5. Build the optimization model and derive the execution contract from the request frame.
6. Write the durable workspace.
7. Launch `create_goal` if goal tools are available, no active goal exists, and the user asked to run now. Otherwise output the exact `/goal` command.
8. During execution, before each work segment:
   - read `model.md`, `plan.md`, `evidence.md`, and `progress.md`;
   - choose the next incomplete task;
   - verify no stop condition is active.
9. After each meaningful work segment:
   - update `progress.md`;
   - record validation or attempted validation in `evidence.md`;
   - record discoveries and assumptions in `notes.md`;
   - update `model.md` only if new evidence changes feasibility or the selected plan.

## Goal Objective Template

Use this objective when launching or preparing goal mode:

```text
Execute the KKT workspace at .kkt/<slug>/plan.md. Follow model.md, plan.md, progress.md, evidence.md, and notes.md. Re-read them before each continuation, keep progress and evidence current, re-optimize only when new evidence invalidates the selected plan, and stop only for listed stop conditions, proven acceptance criteria, or explicit user input.
```

Do not set a token budget unless the user explicitly provides one.

## Required File Templates

`model.md`:

```markdown
# Optimization Model

## Request Intake
| Field | Value | Confidence |
| --- | --- | --- |
| Feature or problem | <What the user wants solved.> | explicit / inferred / assumption / unknown |
| Known non-goals | <What should stay out of scope.> | explicit / inferred / assumption / unknown |
| Hard constraints | <Requirements that make plans infeasible if violated.> | explicit / inferred / assumption / unknown |
| Validation expectations | <Evidence needed to prove completion.> | explicit / inferred / assumption / unknown |
| Execution mode | loop | explicit / inferred / assumption |

## Objective
<One durable outcome.>

## System State
### Facts
- <Repo/runtime facts discovered.>

### Assumptions
- <Assumptions to revisit if contradicted.>

## Decision Variables
| Variable | Allowed Domain | Disallowed Options | Chosen Value | Rationale |
| --- | --- | --- | --- | --- |
| <variable> | <options> | <options> | <value> | <why> |

## Constraint Contract
### Hard
- <Must hold for feasibility.>

### Soft
- <Preference used to compare feasible plans.>

## Feasible Region
- <Candidate plan that satisfies hard constraints.>

## Selected Plan
<Chosen feasible plan and why it dominates alternatives.>

## Solution Audit
### Binding Constraints
- <Active or limiting constraint materially shaping the selected plan.>

### Non-Binding Constraints
- <Checked constraint that does not materially shape the selected plan.>

## Sensitivity Analysis
- <What would change if a binding constraint were relaxed.>
```

`plan.md`:

```markdown
# Execution Contract: <Goal Title>

## Execution Plan
- [ ] <Task.>
  End condition: <What proves this task is complete.>

## Acceptance Criteria
- [ ] <Checkable final condition.>

## Validation Plan
- <Command, check, artifact, or manual verification needed.>

## Evidence Required
- <Proof that must be recorded before completion.>

## Stop Conditions
Stop before continuing if:
- <Hard ambiguity, destructive action, credential need, external dependency, or scope expansion.>

## Continuation Policy
1. Re-read model.md, plan.md, evidence.md, and progress.md.
2. Select the next incomplete task.
3. Execute the smallest feasible step.
4. Record evidence and progress.
5. Re-optimize only when new evidence invalidates the selected plan.
```

`evidence.md`:

```markdown
# Evidence

## Acceptance Criteria Map
| Criterion | Evidence Needed | Status | Evidence |
| --- | --- | --- | --- |
| <criterion> | <command/check/artifact> | Pending |  |

## Validation Log
| Time | Command / Check | Result | Notes |
| --- | --- | --- | --- |
```

`progress.md`:

```markdown
# Progress

## Current Status
- State: Not started
- Current task: <first task>
- Last updated: <timestamp>

## Task List
- [ ] <Task>

## Work Log
| Time | Update |
| --- | --- |
| <time> | Workspace created. |

## Blockers
- None
```

`notes.md`:

```markdown
# Notes

## Observations
- <Relevant discoveries.>

## Assumptions
- <Assumptions.>

## Open Questions
- <Questions.>

## Deferred Ideas
- <Ideas intentionally left out.>
```

## Do Not

- Do not use workspace files as a substitute for understanding the request.
- Do not create multiple workspaces for the same objective.
- Do not include secrets or credentials in workspace files.
- Do not continue past a listed stop condition.
- Do not re-optimize on preference alone; re-optimize when evidence changes feasibility, constraints, or objective fit.
