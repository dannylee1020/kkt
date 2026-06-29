---
name: kkt-model
description: Deep constrained optimization modeling for coding and product-engineering decisions. Use when a coding agent should capture user meaning, discover repo constraints and validation paths, inspect the system, identify objectives, system state, decision variables, domains, hard and soft constraint contracts, selected-plan binding constraints, execution-contract implications, multiple feasible models or architectures, sensitivity analysis, and interactive user tradeoffs before any implementation.
license: Apache-2.0
---

# KKT Model

Use this skill when the deliverable is a model, decision brief, or implementation-ready recommendation rather than code. It is for architecture choices, feature shaping, complex implementation options, scope negotiation, and high-impact tradeoffs.

Read `references/feature-optimization-model.md`, `references/layered-modeling-methods.md`, `references/state-contract.md`, and `references/layers/` before acting.

## Core Rule

Stay non-mutating by default. Intake before modeling. Discovery before asking for repo facts. Select the modeling method that fits the decision shape. Ask only for product, risk, scope, approval, or execution-mode choices that cannot be resolved from inspection or conservative reversible defaults.

## CLI-First Workflow

Use the `kkt` CLI whenever durable model state is useful. The skill owns modeling judgment; the CLI owns workspace creation, state reads, artifact recording, validation, and completion.

```text
kkt start model "<user request>"
kkt status
kkt next
kkt intent "<intent frame>"
kkt discovery "<repo facts and constraints>"
kkt model "<selected model and tradeoffs>"
kkt validate
kkt done
```

If `kkt` is missing and durable model state is needed, stop and ask the user to install or upgrade KKT. Do not hand-write replacement state.

## Workflow

1. Capture intent: user goal, desired behavior, user-visible success, scope boundary, examples, priority signals, and explicit user constraints.
2. Inspect relevant code, docs, configs, schemas, routes, tests, UI, infra, issues, or logs before choosing a model.
3. Separate explicit requirements, discovered facts, inferred constraints, assumptions, unknowns, and owner decisions.
4. Build a discovery map when the decision crosses modules, workflows, contracts, or architecture boundaries.
5. Select the modeling method from the layered catalog and state why it fits.
6. Build the shared optimization model from intent and discovery: objective, system state, decision variables, hard/soft constraints, candidates, feasibility, binding constraints, sensitivity, and execution implications.
7. Produce 2-4 candidate models when meaningful alternatives exist; eliminate infeasible models before comparing feasible ones.
8. Compare feasible models by hard-constraint satisfaction, binding constraints, blast radius, maintainability, validation clarity, reversibility, and fit with user intent.
9. Ask the user only for unresolved owner decisions; otherwise select the best feasible model.
10. Record durable output with `kkt intent`, `kkt discovery`, `kkt model`, and `kkt validate` when a workspace exists.

## End States

End with one of:

- `Selected model`: one feasible model is recommended and ready for implementation.
- `Decision needed`: the smallest user decisions required to select a model.
- `No feasible model`: the hard constraints that block feasibility and the relaxation that would restore it.

For each serious alternative, include the method used, objective fit, decision-variable assignments, hard-constraint status, binding constraints, tradeoffs, execution-contract implications, residual risks, and when to choose it.

## Durable Output

For substantial modeling work, use project-root `.kkt/model/<slug>/` through `kkt` commands. `kkt.yaml` is the state index; Markdown files carry rich intent, discovery, and modeling context. Do not create execution files unless switching to `$kkt-loop`.

## Do Not

- Do not modify code unless the user explicitly switches from modeling to implementation.
- Do not invent numeric scores for subjective criteria.
- Do not collapse materially different architectures into one vague plan.
- Do not ask for user input before inspecting discoverable context.
- Do not choose a method because it sounds rigorous; choose it because the decision shape calls for it.
- Do not recommend a model without selected-plan binding constraints and execution implications.
