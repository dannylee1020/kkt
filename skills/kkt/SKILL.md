---
name: kkt
description: Constrained optimization workflow for ordinary coding requests. Use when a coding agent should capture user meaning, discover repo constraints and validation paths, build a mathematical-style optimization model, derive an execution contract, select a feasible implementation plan, execute the change, validate with evidence, and finish with a constraint audit.
license: Apache-2.0
---

# KKT

Use this skill for normal coding work that benefits from stricter planning than default plan mode. Apply Karush-Kuhn-Tucker-inspired constrained modeling as a discipline, not as numeric optimization.

Read `references/feature-optimization-model.md` and `references/state-contract.md` before acting. Use `references/layered-modeling-methods.md` and the internal `references/layers/` contracts only when the request needs method selection beyond the daily profile.

## Core Rule

Intake before modeling. Model before editing. Show the final modeling result and get explicit user approval before implementation. Derive an execution contract before changing code. Finish only with validation evidence or an explicit blocker.

## Workflow

1. Translate the user's rough input into an intent frame: user goal, desired behavior, user-visible success, scope boundary, priority signals, examples, and explicit user constraints.
2. Before asking, apply the owner-decision filter: inspect discoverable facts locally, choose conservative reversible defaults when risk is low and record them as assumptions, and ask only for owner decisions that materially change product behavior, risk, scope, approval, or execution mode. Do not ask for files, routes, schemas, tests, config, constraints, or validation commands that can be discovered locally.
3. Inspect relevant code, docs, tests, config, schemas, routes, UI, infra, logs, or issues to discover repo facts, constraints, validation paths, and likely technical non-goals before forming the model.
4. Separate explicit user statements, discovered facts, inferred constraints, assumptions, unknowns, and owner decisions. Ask only when a high-impact product choice, irreversible tradeoff, external dependency, destructive action, scope expansion, or infeasible ambiguity remains.
5. Build a compact optimization model using the daily profile:
   - request intake;
   - objective;
   - system state;
   - decision variables and their domains;
   - hard and soft constraint contract;
   - feasible plans;
   - selected plan;
   - selected-plan binding audit;
   - sensitivity analysis.
6. Derive a compact execution contract:
   - acceptance criteria;
   - validation plan;
   - evidence required;
   - stop conditions.
7. Eliminate infeasible plans before comparing feasible plans.
8. Select the best feasible plan by lexicographic objective order:
   1. satisfy the user request;
   2. preserve correctness, security, data integrity, and public contracts;
   3. minimize blast radius;
   4. match existing architecture and conventions;
   5. improve maintainability where cheap;
   6. prefer validation clarity over elegance.
9. Show the final modeling result and wait for explicit user approval before implementation.
10. Execute the approved plan with focused edits.
11. Validate against the optimization model and execution contract.
12. End with a short constraint audit.

## Output Discipline

For small tasks, keep the model brief. Do not create durable files unless the user asks, the task becomes long-running, or `$kkt-loop` is more appropriate.

When durable state is useful for normal `$kkt` work, use the daily tier from `references/state-contract.md`: a single `kkt.yaml` with compact layer summaries, decisions, artifact references, approval state, and validation evidence. Do not create Markdown layer artifacts in the daily tier; switch to `$kkt-model` or `$kkt-loop` when discovery or modeling context needs rich capture.

The intent, discovery, modeling, execution, and validation layers are internal contract boundaries, not user-facing skills. Run them inside this skill when needed; do not ask the user to invoke layer names directly.

Before implementation, always expose a concise modeling result and ask for approval. Include the objective, selected plan, relevant rejected alternatives, binding constraints, expected files or surfaces, validation plan, and residual risk. Keep formal method names hidden unless they explain a material tradeoff.

Use this final shape:

```text
Objective: satisfied / not satisfied
Hard constraints: satisfied / violations listed
Binding constraints: respected / changed
Validation evidence: commands, checks, artifacts, or reason validation was not possible
Residual risk: concise notes
```

## Re-Optimization

If execution discovers a fact that invalidates the selected plan, pause implementation, update the model, choose the new best feasible plan, and continue only if no stop condition is hit.

## Execution Stop Conditions

Stop and ask the user before continuing when:

- no feasible plan satisfies the hard constraints;
- the user does not approve the final modeling result;
- a destructive action is required;
- credentials, secrets, external access, or paid services are required;
- multiple feasible plans differ mainly by product intent;
- continuing would expand scope beyond the user request.

## Do Not

- Do not use fake numeric scores for subjective qualities.
- Do not treat the first plausible plan as selected before feasibility checks.
- Do not edit files before the final modeling result is explicitly approved.
- Do not follow existing project patterns silently when they appear wrong; flag the issue.
- Do not make broad refactors unless the model shows they are necessary.
- Do not finish without validation evidence or an explicit statement that validation could not be run.
