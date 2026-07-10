---
name: kkt
description: Constrained optimization workflow for ordinary coding requests. Use when a coding agent should capture user meaning, discover repo constraints and validation paths, build a mathematical-style optimization model, derive an execution contract, select a feasible implementation plan, execute the change, validate with evidence, and finish with a constraint audit.
license: Apache-2.0
---

# KKT

Use this skill for ordinary coding work that needs stricter planning than default plan mode. Apply constrained optimization as a discipline: choose the best feasible implementation given what must stay true.

Read `references/kkt-kernel.md` before acting. Read `references/feature-optimization-model.md` when building the compact model. Read `references/state-contract.md` only when optional plan-tier durable state is explicitly needed. Read `references/plan-assimilation.md` only when prior plan text exists. Read `references/discovery-tooling.md` during non-trivial discovery or structural search. Read `references/layered-modeling-methods.md` only when method selection beyond the plan profile is needed. Read `references/schemas.md` only when a full copyable state, guardrail, or layer-output shape is needed.

## Core Rule

Intake before modeling. Discovery before asking for repo facts. Model before editing. Show the final modeling result and get explicit user approval before implementation. Finish only with validation evidence or an explicit blocker.

## Persistence Policy

Default to no durable `.kkt/` state for `$kkt`. For ordinary coding work, keep the compact model, approval, execution notes, validation evidence, and final audit in the conversation.

Use optional plan-tier persistence only when the user explicitly asks for durable state, handoff/resume support is needed, or validation/evidence must survive across turns while the task still fits the lightweight `$kkt` profile. If the work needs rich durable artifacts, guardrails, task queues, continuation, deep option modeling, or deterministic drift checks, do not stretch `$kkt`; switch to `$kkt-model`, `$kkt-run`, or `$kkt-loop` as appropriate.

## Optional Durable Plan-Tier Workflow

When optional plan-tier persistence is justified, use the `kkt` CLI. The skill owns reasoning policy; the CLI owns state creation, reads, mutation, approval, evidence, validation, and completion.

```text
kkt start plan "<user request>"
kkt status [--json]
kkt next
kkt model "<objective_function, files_to_modify, constraint_functions, decision_variables, validation_proof, and selected compact model>"
kkt approve "<approved scope>"
kkt evidence "<validation evidence>"
kkt validate [--run]
kkt done
```

If `kkt` is missing and optional durable plan-tier state is required, stop and ask the user to install or upgrade KKT. Do not hand-write replacement state. Use `kkt validate --run` when guardrails define required commands; `kkt evidence` alone is narrative evidence, not command proof.

## Workflow

1. Capture intent: user goal, desired behavior, user-visible success, scope boundary, examples, priority signals, and explicit user constraints.
2. If prior plan text exists, assimilate it as untrusted scaffold: extract signals, classify claims, verify discoverable facts, and keep unverified claims as assumptions or candidates.
3. Apply the owner-decision filter before asking: inspect discoverable facts locally, assume low-risk reversible defaults, ask only for owner decisions, and stop for blocking unknowns.
4. Inspect relevant code, docs, tests, config, schemas, routes, UI, infra, logs, or issues before forming the model. Use `rg` directly for broad text and file discovery, `ast-grep` directly for structural search when syntax matters, `git` for repository state, and language-native commands when they provide stronger evidence.
5. Separate explicit user statements, prior-plan assumptions, discovered facts, inferred constraints, assumptions, unknowns, and owner decisions.
6. Build a compact model: objective function, system state, files to modify, constraint functions, decision variables, hard/soft constraints, feasible plans, selected plan, binding constraints, and sensitivity.
7. Derive the execution contract: acceptance criteria, validation plan, evidence required, and stop conditions.
8. Reject infeasible plans, then choose the best feasible plan by this order: user request, correctness/security/data/public contracts, blast radius, existing architecture, maintainability, validation clarity.
9. Show the final modeling result and wait for approval before editing.
10. Execute the approved plan with focused edits, then validate with evidence and finish with a constraint audit.

## Output Discipline

For ordinary `$kkt` tasks, keep the model brief and do not create durable state by default. For optional durable plan-tier state, use project-root `.kkt/kkt.yaml` through `kkt` commands; do not hand-edit `kkt.yaml` as the primary workflow operation. Load full schemas only when writing or auditing durable state. Switch to `$kkt-model` for deeper non-mutating modeling, `$kkt-run` to implement a completed model with guardrails, or `$kkt-loop` for long-running continuation.

Before implementation, expose a compact optimized plan: objective function, known constraints, files to modify, constraint functions, decision variables, selected plan, rejected alternatives, binding constraints, validation proof plan, and residual risk. Keep formal method names hidden unless they explain a material tradeoff.

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
