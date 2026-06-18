---
name: kkt-model
description: Deep constrained optimization modeling for coding and product-engineering decisions. Use when a coding agent should translate rough user input into a request frame, infer missing non-goals, constraints, validation expectations, and execution mode, inspect the system, identify objectives, system state, decision variables, domains, hard and soft constraint contracts, selected-plan binding constraints, execution-contract implications, multiple feasible models or architectures, sensitivity analysis, and interactive user tradeoffs before any implementation.
license: Apache-2.0
---

# KKT Model

Use this skill when the main deliverable is the model, not code. It is for architecture choices, feature-shaping, complex implementation options, scope negotiation, and high-impact tradeoffs.

Read `references/feature-optimization-model.md` before acting.

## Core Rule

Stay non-mutating by default. Intake before modeling. Inspect, model, compare, and ask for user input where product or risk tradeoffs are genuinely undecidable from the repo.

## Workflow

1. Translate the user's rough input into a request frame: feature or problem, known non-goals, hard constraints, validation expectations, and execution mode.
2. Inspect relevant code, docs, configs, schemas, routes, tests, UI, infra, issues, or logs to infer discoverable intake fields.
3. State explicit user requirements, discovered facts, inferred constraints, assumptions, and unknowns separately.
4. Build the shared optimization model from the request frame.
5. Produce 2-4 candidate models when meaningful alternatives exist.
6. Eliminate infeasible models.
7. Compare feasible models by:
   - hard-constraint satisfaction;
   - selected-plan binding constraints;
   - blast radius;
   - maintainability;
   - validation clarity;
   - reversibility;
   - fit with user intent.
8. Ask the user only for unresolved product choices, risk tolerance, scope boundaries, constraint relaxation, or execution-mode ambiguity.
9. End with a selected model, implementation-ready brief, or a small set of user decisions needed before implementation.

## Candidate Model Shape

Use this format for each serious alternative:

```yaml
model_name:
objective_fit:
decision_variable_assignments:
hard_constraints_satisfied:
hard_constraints_violated:
binding_constraints:
tradeoffs:
execution_contract_implications:
residual_risks:
when_to_choose:
```

## Interactive Input

Ask for user input when:

- two or more feasible models differ mostly by product intent;
- a binding constraint could be relaxed and materially improves the solution;
- implementation risk depends on tolerance for migration, refactor, dependency, downtime, or UX change;
- the user request is under-specified and repo inspection cannot resolve the ambiguity.

Do not ask the user to identify files, symbols, routes, or config that can be discovered locally.

## End States

End with one of:

- `Selected model`: one feasible model is recommended and ready for implementation.
- `Decision needed`: list the smallest user decisions required to select a model.
- `No feasible model`: explain which hard constraints make the request infeasible and what relaxation would restore feasibility.

## Optional Durable Output

For substantial modeling work, write a modeling artifact only when useful or requested:

```text
.kkt-model/<slug>/model.md
```

Do not create execution files unless switching to `$kkt-loop`.

## Do Not

- Do not modify code unless the user explicitly switches from modeling to implementation.
- Do not invent numeric scores for subjective criteria.
- Do not collapse materially different architectures into one vague plan.
- Do not ask for user input before inspecting discoverable context.
- Do not recommend a model without explaining the selected-plan binding constraints and execution-contract implications.
