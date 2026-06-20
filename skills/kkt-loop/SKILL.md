---
name: kkt-loop
description: Durable constrained optimization execution loop for long-running coding work. Use when a coding agent should capture user meaning, discover repo constraints and validation paths, build an optimization model and execution contract, create a file-backed workspace, execute with continuation, validate with evidence, re-optimize when new facts invalidate the plan, and continue through goal-mode tools until acceptance criteria or stop conditions are met.
license: Apache-2.0
---

# KKT Loop

Use this skill for long-running, autonomous, or multi-step coding work where progress and evidence must survive continuation. Apply the kkt optimization model and execution contract, then use an `ultragoal`-style durable workspace for execution.

Read `references/feature-optimization-model.md`, `references/layered-modeling-methods.md`, `references/state-contract.md`, and the internal `references/layers/` contracts before acting. If the `ultragoal` skill is available, use its goal-state and continuation semantics as the runtime pattern.

## Core Rule

Complete request intake before modeling. Show the final modeling result and get explicit user approval before creating durable state or launching a loop. On every continuation, re-read the workspace, update progress and evidence, and stop only when the acceptance criteria are proven or a stop condition is hit.

Intent, discovery, modeling, execution, and validation are internal contract boundaries, not user-facing skills. Run them inside this skill and persist their state through the workspace.

## Workspace Location

Create files under the current repo:

```text
.kkt/<slug>/
```

Use a short filesystem-safe slug derived from the objective.

## Workspace Files

Always create:

```text
.kkt/<slug>/kkt.yaml
.kkt/<slug>/intent.md
.kkt/<slug>/discovery.md
.kkt/<slug>/model.md
.kkt/<slug>/plan.md
.kkt/<slug>/evidence.md
.kkt/<slug>/progress.md
.kkt/<slug>/notes.md
```

- `kkt.yaml`: canonical state index, active layer, method choices, artifact references, approval state, stop conditions.
- `intent.md`: user goal, desired behavior, user-visible success, scope boundary, examples, priority signals, explicit user constraints, unresolved meaning questions.
- `discovery.md`: discovered files, symbols, workflows, constraints, validation paths, coupling, evidence, confidence, unknowns.
- `model.md`: optimization model, candidate plans, selected plan, sensitivity notes.
- `plan.md`: execution contract, tasks, acceptance criteria, validation plan, stop conditions, continuation policy.
- `evidence.md`: validation map, command outputs, artifacts, and proof log.
- `progress.md`: current status, active task, task list, work log, blockers.
- `notes.md`: observations, assumptions, open questions, deferred ideas.

## Workflow

1. Check current goal state with `get_goal` if available. Do not create a second active goal without explicit user direction.
2. Translate the user's rough input into an intent frame: user goal, desired behavior, user-visible success, scope boundary, priority signals, examples, and explicit user constraints.
3. Ask only the smallest useful set of meaning-focused questions; do not ask for files, routes, schemas, tests, config, constraints, or validation commands that can be discovered locally.
4. Inspect relevant repo context to discover repo facts, constraints, validation paths, and likely technical non-goals before writing the model.
5. Ask only when an unresolved field materially changes feasibility, product behavior, risk, scope, or execution mode.
6. Build the optimization model and derive the execution contract from the intent frame and discovery results using the loop profile.
7. Show the final modeling result and wait for explicit user approval.
8. Write the durable workspace only after approval.
9. Launch `create_goal` if goal tools are available, no active goal exists, and the user asked to run now. Otherwise output the exact `/goal` command.
10. During execution, before each work segment:
   - read `kkt.yaml`, `intent.md`, `discovery.md`, `model.md`, `plan.md`, `evidence.md`, and `progress.md`;
   - choose the next incomplete task;
   - verify no stop condition is active.
11. After each meaningful work segment:
   - update `kkt.yaml` and `progress.md`;
   - record validation or attempted validation in `evidence.md`;
   - record discoveries and assumptions in `notes.md`;
   - update `model.md` only if new evidence changes feasibility or the selected plan.

## Goal Objective Template

Use this objective when launching or preparing goal mode:

```text
Execute the KKT workspace at .kkt/<slug>/plan.md. Follow kkt.yaml, intent.md, discovery.md, model.md, plan.md, progress.md, evidence.md, and notes.md. Re-read them before each continuation, keep state, progress, and evidence current, re-optimize only when new evidence invalidates the selected plan, and stop only for listed stop conditions, proven acceptance criteria, or explicit user input.
```

Do not set a token budget unless the user explicitly provides one.

## Required File Templates

`kkt.yaml`:

```yaml
schema_version: 1
workspace_type: loop
profile: loop
status: approved | executing | validating | complete | blocked
active_layer: execution
layers:
  intent:
    status: complete
    summary: ""
    artifact: intent.md
  discovery:
    status: complete
    summary: ""
    artifact: discovery.md
  modeling:
    status: complete
    summary: ""
    artifact: model.md
  execution:
    status: pending
    summary: ""
    artifact: plan.md
  validation:
    status: pending
    summary: ""
    artifact: evidence.md
method_invocations: []
decision_log: []
artifact_refs:
  intent: intent.md
  discovery: discovery.md
  model: model.md
  plan: plan.md
  progress: progress.md
  evidence: evidence.md
  notes: notes.md
approval:
  required: true
  status: approved
  approved_scope: ""
stop_conditions: []
```

`intent.md`:

```markdown
# Intent

## Request Frame
| Field | Value | Confidence |
| --- | --- | --- |
| User goal | <What the user wants accomplished.> | explicit / inferred / assumption / unknown |
| Desired behavior | <What should be different after the work.> | explicit / inferred / assumption / unknown |
| User-visible success | <What success looks like from the user's perspective.> | explicit / inferred / assumption / unknown |
| Scope boundary | <What is in scope and what should stay out of scope.> | explicit / inferred / assumption / unknown |
| Examples | <Examples or counterexamples that clarify meaning.> | explicit / inferred / assumption / unknown |
| Priority signals | <User-stated priorities or tradeoffs.> | explicit / inferred / assumption / unknown |
| Explicit user constraints | <Only constraints the user directly stated.> | explicit / inferred / assumption / unknown |

## Clarifications
- <Only unresolved meaning questions that materially affect product behavior, risk, or scope.>
```

`discovery.md`:

```markdown
# Discovery

## Discovery Map
| Artifact / Surface | Why Relevant | Constraint or Variable | Evidence | Confidence |
| --- | --- | --- | --- | --- |
| <file/module/route/schema/test/workflow> | <reason> | <constraint or variable> | <evidence> | observed / inferred / assumption |

## Coupling and Blast Radius
- Coupling map: <modules, workflows, or state boundaries affected.>
- Central nodes: <high-impact artifacts, if any.>
- Risky boundaries: <public contracts, data, security, infra, or UX boundaries.>
- Impact radius: low / medium / high.

## Unknowns
- <Unknowns discovery could not resolve.>

## Validation Candidates
- <Command, check, artifact, or manual verification path discovered from the repo.>
```

`model.md`:

```markdown
# Optimization Model

## Request Intake
| Field | Value | Confidence |
| --- | --- | --- |
| User goal | <What the user wants accomplished.> | explicit / inferred / assumption / unknown |
| Desired behavior | <What should be different after the work.> | explicit / inferred / assumption / unknown |
| User-visible success | <What success looks like from the user's perspective.> | explicit / inferred / assumption / unknown |
| Scope boundary | <What is in scope and what should stay out of scope.> | explicit / inferred / assumption / unknown |
| Examples | <Examples or counterexamples that clarify meaning.> | explicit / inferred / assumption / unknown |
| Priority signals | <User-stated priorities or tradeoffs.> | explicit / inferred / assumption / unknown |
| Explicit user constraints | <Only constraints the user directly stated.> | explicit / inferred / assumption / unknown |
| Execution mode | loop | default |

## Objective
<One durable outcome.>

## System State
### Facts
- <Repo/runtime facts discovered.>

### Assumptions
- <Assumptions to revisit if contradicted.>

## Discovery Map
| Artifact / Surface | Why Relevant | Constraint or Variable | Evidence | Confidence |
| --- | --- | --- | --- | --- |
| <file/module/route/schema/test/workflow> | <reason> | <constraint or variable> | <evidence> | observed / inferred / assumption |

## Coupling and Blast Radius
- Coupling map: <modules, workflows, or state boundaries affected.>
- Central nodes: <high-impact artifacts, if any.>
- Risky boundaries: <public contracts, data, security, infra, or UX boundaries.>
- Impact radius: low / medium / high.

## Validation Candidates
- <Command, check, artifact, or manual verification path discovered from the repo.>

## Method Selector
- Profile: loop
- Modeling method: <lexicographic ranking / decision tree / shortest path / ordinal MCDA / AHP-style questions / outranking shortlist>
- Reason: <why this method fits the decision shape.>

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
1. Re-read kkt.yaml, intent.md, discovery.md, model.md, plan.md, evidence.md, and progress.md.
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
- Do not create workspace files, launch goal mode, or begin implementation before the final modeling result is explicitly approved.
- Do not create multiple workspaces for the same objective.
- Do not include secrets or credentials in workspace files.
- Do not continue past a listed stop condition.
- Do not re-optimize on preference alone; re-optimize when evidence changes feasibility, constraints, or objective fit.
