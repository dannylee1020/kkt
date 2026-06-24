---
name: kkt
description: Constrained optimization workflow for ordinary coding requests. Use when a coding agent should capture user meaning, discover repo constraints and validation paths, build a mathematical-style optimization model, derive an execution contract, select a feasible implementation plan, execute the change, validate with evidence, and finish with a constraint audit.
license: Apache-2.0
---

# KKT

Use this skill for ordinary coding work that needs stricter planning than default plan mode. Apply constrained optimization as a discipline: choose the best feasible implementation given what must stay true.

Read `references/feature-optimization-model.md` and `references/state-contract.md` before acting. Use `references/layered-modeling-methods.md` and `references/layers/` only when the request needs method selection beyond the plan profile.

## Core Rule

Intake before modeling. Discovery before asking for repo facts. Model before editing. Show the final modeling result and get explicit user approval before implementation. Finish only with validation evidence or an explicit blocker.

## CLI-First Workflow

Use the `kkt` CLI whenever durable state is useful. The skill owns reasoning policy; the CLI owns state creation, reads, mutation, approval, evidence, validation, and completion.

```text
kkt start plan "<user request>"
kkt status
kkt next
kkt model "<selected compact model>"
kkt approve "<approved scope>"
kkt evidence "<validation evidence>"
kkt validate
kkt done
```

If `kkt` is missing and durable state is needed, stop and ask the user to install or upgrade KKT. Do not hand-write replacement state.

## Workflow

1. Capture intent: user goal, desired behavior, user-visible success, scope boundary, examples, priority signals, and explicit user constraints.
2. Apply the owner-decision filter before asking: inspect discoverable facts locally, assume low-risk reversible defaults, ask only for owner decisions, and stop for blocking unknowns.
3. Inspect relevant code, docs, tests, config, schemas, routes, UI, infra, logs, or issues before forming the model.
4. Separate explicit user statements, discovered facts, inferred constraints, assumptions, unknowns, and owner decisions.
5. Build a compact model: objective, system state, decision variables, hard/soft constraints, feasible plans, selected plan, binding constraints, and sensitivity.
6. Derive the execution contract: acceptance criteria, validation plan, evidence required, and stop conditions.
7. Reject infeasible plans, then choose the best feasible plan by this order: user request, correctness/security/data/public contracts, blast radius, existing architecture, maintainability, validation clarity.
8. Show the final modeling result and wait for approval before editing.
9. Execute the approved plan with focused edits, then validate with evidence and finish with a constraint audit.

## Output Discipline

For small tasks, keep the model brief and avoid durable state unless it helps. For durable plan-tier state, use `.kkt/kkt.yaml` through `kkt` commands; do not hand-edit `kkt.yaml` as the primary workflow operation. Switch to `$kkt-model` for deeper non-mutating modeling or `$kkt-loop` for long-running continuation.

Before implementation, expose: objective, selected plan, rejected alternatives, binding constraints, expected files or surfaces, validation plan, and residual risk. Keep formal method names hidden unless they explain a material tradeoff.

Final audit shape:

```text
Objective: satisfied / not satisfied
Hard constraints: satisfied / violations listed
Binding constraints: respected / changed
Validation evidence: commands, checks, artifacts, or reason validation was not possible
Residual risk: concise notes
```

## Stop Conditions

Stop and ask before continuing when no feasible plan satisfies hard constraints, approval is missing, destructive action is required, credentials/secrets/external access/paid services are required, feasible plans differ mainly by product intent, or continuing expands scope.

## Do Not

- Do not use fake numeric scores for subjective qualities.
- Do not treat the first plausible plan as selected before feasibility checks.
- Do not edit before the final modeling result is approved.
- Do not silently follow existing patterns that appear wrong; flag them.
- Do not make broad refactors unless required by the model.
- Do not finish without validation evidence or an explicit validation limitation.
