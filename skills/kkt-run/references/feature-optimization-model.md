# Feature Optimization Model

Use this reference when building or reviewing a KKT model. Keep the active model compact; load `schemas.md` only when a full copyable shape is needed.

## Intake

Start with user meaning, then let discovery fill in repo facts.

Capture:

- user goal;
- desired behavior;
- user-visible success;
- scope boundary;
- examples or counterexamples, if supplied;
- priority signals;
- explicit user constraints;
- execution mode: `implement`, `model_only`, `run`, or `loop`.

When prior plan text exists, read `plan-assimilation.md` first. Preserve the user's original request, classify prior-plan claims, verify discoverable facts, and keep unverified claims as assumptions or candidates.

Do not ask the user for repo facts that local discovery can verify. Use the question filter in `kkt-kernel.md`.

## Model Shape

Treat the task as:

```text
choose implementation decisions
that best satisfy the user's objective
subject to project, architecture, security, data, UI, infrastructure, validation, and scope constraints
```

Build the model around:

- `objective_function`: ordered terms used to compare feasible plans.
- `system_state`: inspected facts plus labeled assumptions.
- `decision_variables`: choices the agent may make, each with an allowed domain.
- `files_to_modify`: expected files, modules, APIs, workflows, migrations, docs, or operational surfaces.
- `constraint_contract`: hard predicates for feasibility and soft preferences for ranking.
- `feasible_region`: candidate plans that satisfy all hard constraints.
- `selected_plan`: chosen feasible plan and why it wins.
- `binding_constraints`: active constraints that shape the selected plan.
- `validation_proof`: commands, checks, artifacts, or explicit limitations needed before completion.

## Method Profiles

- Plan profile (`kkt`): compact intake, local discovery, hard-constraint feasibility, lexicographic ranking, approval before edits, validation certificate.
- Deep profile (`kkt-model`): layered method selection, candidate comparison, binding constraints, sensitivity, guardrail variables, and user tradeoff questions when needed.
- Run profile (`kkt-run`): import a completed model, validate guardrails, run deterministic checkpoints, execute only the selected plan.
- Loop profile (`kkt-loop`): durable model plus continuation state, task/criteria tracking, evidence-backed continuation, and re-optimization only on material new evidence.

Read `layered-modeling-methods.md` for deep or loop modeling choices. Do not expose formal method names in ordinary `$kkt` output unless they explain a material tradeoff.

## Decision Variables

Decision variables are implementation choices with explicit domains. Common variables include:

- files or modules to modify;
- endpoint, CLI, schema, or event shape;
- data storage or migration strategy;
- state ownership;
- UI placement;
- validation strategy;
- compatibility strategy;
- dependency strategy;
- rollout or fallback strategy.

Poor: `database: decide storage`

Better: `persistence_strategy: add nullable fields to existing table`, with allowed and disallowed options recorded.

## Constraints

Hard constraints make a plan infeasible when violated:

- explicit user constraints and non-goals;
- correctness, security, privacy, and data integrity;
- public API or persisted-data compatibility;
- framework, runtime, infrastructure, or deployment limits;
- no destructive action, credentials, paid services, or external side effects without approval.

Soft constraints rank feasible plans:

- minimal diff and blast radius;
- existing style and architecture fit;
- readability, maintainability, reversibility, and validation clarity.

Use lexicographic priority instead of fake numeric scores:

1. Satisfy the user request.
2. Preserve correctness, security, data integrity, and public contracts.
3. Minimize blast radius.
4. Match existing architecture and conventions.
5. Improve maintainability where cheap.
6. Prefer validation clarity over elegance.

## Selected Plan Contract

An approval-ready plan or selected model must include:

- objective function;
- known constraints by source: explicit, discovered, inferred, assumptions;
- decision variables and chosen values;
- affected files or surfaces;
- feasible and rejected candidates;
- selected plan;
- binding constraints;
- validation plan and proof required;
- execution implications;
- guardrail variables when durable run or loop state will enforce scope;
- residual risk.

If a field is irrelevant, say `none` or give a short reason. Do not hide constraints inside prose.

## Re-Optimization

Re-optimize when new evidence changes system state, feasible region, hard constraints, selected-plan binding constraints, selected decisions, or validation feasibility.

Do not re-optimize merely because another plan feels more elegant.
