---
name: kkt
description: Constrained optimization workflow for ordinary coding requests. Use when a coding agent should translate rough user input into a request frame, infer missing non-goals, constraints, validation expectations, and execution mode, build a mathematical-style optimization model, derive an execution contract, select a feasible implementation plan, execute the change, validate with evidence, and finish with a constraint audit.
license: Apache-2.0
---

# KKT

Use this skill for normal coding work that benefits from stricter planning than default plan mode. Apply Karush-Kuhn-Tucker-inspired constrained modeling as a discipline, not as numeric optimization.

Read `references/feature-optimization-model.md` before acting.

## Core Rule

Intake before modeling. Model before editing. Derive an execution contract before changing code. Finish only with validation evidence or an explicit blocker.

## Workflow

1. Translate the user's rough input into a request frame: feature or problem, known non-goals, hard constraints, validation expectations, and execution mode.
2. Inspect relevant code, docs, tests, config, schemas, routes, UI, infra, logs, or issues to infer discoverable intake fields before forming the model.
3. Separate explicit user statements, discovered facts, inferred constraints, assumptions, and unknowns. Ask only when a high-impact product choice or infeasible ambiguity remains.
4. Build a compact optimization model:
   - request intake;
   - objective;
   - system state;
   - decision variables and their domains;
   - hard and soft constraint contract;
   - feasible plans;
   - selected plan;
   - selected-plan binding audit;
   - sensitivity analysis.
5. Derive a compact execution contract:
   - acceptance criteria;
   - validation plan;
   - evidence required;
   - stop conditions.
6. Eliminate infeasible plans before comparing feasible plans.
7. Select the best feasible plan by lexicographic objective order:
   1. satisfy the user request;
   2. preserve correctness, security, data integrity, and public contracts;
   3. minimize blast radius;
   4. match existing architecture and conventions;
   5. improve maintainability where cheap;
   6. prefer validation clarity over elegance.
8. Execute the selected plan with focused edits.
9. Validate against the optimization model and execution contract.
10. End with a short constraint audit.

## Output Discipline

For small tasks, keep the model brief. Do not create durable files unless the user asks, the task becomes long-running, or `$kkt-loop` is more appropriate.

Before implementation, expose the request frame or model only when useful for user steering or when the task has material tradeoffs. Otherwise keep it as working structure and proceed.

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
- a destructive action is required;
- credentials, secrets, external access, or paid services are required;
- multiple feasible plans differ mainly by product intent;
- continuing would expand scope beyond the user request.

## Do Not

- Do not use fake numeric scores for subjective qualities.
- Do not treat the first plausible plan as selected before feasibility checks.
- Do not follow existing project patterns silently when they appear wrong; flag the issue.
- Do not make broad refactors unless the model shows they are necessary.
- Do not finish without validation evidence or an explicit statement that validation could not be run.
