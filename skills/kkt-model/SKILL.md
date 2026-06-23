---
name: kkt-model
description: Deep constrained optimization modeling for coding and product-engineering decisions. Use when a coding agent should capture user meaning, discover repo constraints and validation paths, inspect the system, identify objectives, system state, decision variables, domains, hard and soft constraint contracts, selected-plan binding constraints, execution-contract implications, multiple feasible models or architectures, sensitivity analysis, and interactive user tradeoffs before any implementation.
license: Apache-2.0
---

# KKT Model

Use this skill when the main deliverable is the model, not code. It is for architecture choices, feature-shaping, complex implementation options, scope negotiation, and high-impact tradeoffs.

Read `references/feature-optimization-model.md`, `references/layered-modeling-methods.md`, `references/state-contract.md`, and the internal `references/layers/` contracts before acting.

## Core Rule

Stay non-mutating by default. Intake before modeling. Inspect, select the appropriate modeling method, compare, and ask for user input where product or risk tradeoffs are genuinely undecidable from the repo.

Intent, discovery, and modeling are internal contract boundaries, not user-facing skills. Run them inside this skill and persist their state when durable output is useful.

## Workflow

1. Translate the user's rough input into an intent frame: user goal, desired behavior, user-visible success, scope boundary, priority signals, examples, and explicit user constraints.
2. Before asking, apply the owner-decision filter: inspect discoverable facts locally, choose conservative reversible defaults when risk is low and record them as assumptions, and ask only for owner decisions that materially change product behavior, risk, scope, approval, or execution mode. Do not ask for files, routes, schemas, tests, config, constraints, or validation commands that can be discovered locally.
3. Inspect relevant code, docs, configs, schemas, routes, tests, UI, infra, issues, or logs to discover repo facts, constraints, validation paths, and likely technical non-goals.
4. State explicit user requirements, discovered facts, inferred constraints, assumptions, and unknowns separately.
5. Build a discovery map when the request crosses multiple modules, workflows, or architecture boundaries.
6. Select the modeling method from the layered catalog and state why it fits the decision shape.
7. Build the shared optimization model from the intent frame and discovery results.
8. Produce 2-4 candidate models when meaningful alternatives exist.
9. Eliminate infeasible models.
10. Compare feasible models by:
   - hard-constraint satisfaction;
   - selected-plan binding constraints;
   - blast radius;
   - maintainability;
   - validation clarity;
   - reversibility;
   - fit with user intent.
11. Ask the user only for owner decisions: unresolved product choices, risk tolerance, scope boundaries, constraint relaxation, approval, or execution-mode ambiguity that cannot be resolved by repo inspection or conservative reversible defaults.
12. End with a selected model, implementation-ready brief, or a small set of user decisions needed before implementation.

## Candidate Model Shape

Use this format for each serious alternative:

```yaml
model_name:
method_used:
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
.kkt-model/<slug>/kkt.yaml
.kkt-model/<slug>/intent.md
.kkt-model/<slug>/discovery.md
.kkt-model/<slug>/model.md
```

Use `kkt.yaml` as the canonical state index and status record. Use the Markdown files for rich intent, discovery, and modeling context that would be lossy in YAML.

Do not create execution files unless switching to `$kkt-loop`.

## Do Not

- Do not modify code unless the user explicitly switches from modeling to implementation.
- Do not invent numeric scores for subjective criteria.
- Do not collapse materially different architectures into one vague plan.
- Do not ask for user input before inspecting discoverable context.
- Do not choose a method because it sounds rigorous; choose it because the decision shape calls for it.
- Do not recommend a model without explaining the selected-plan binding constraints and execution-contract implications.
