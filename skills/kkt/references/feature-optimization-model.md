# kkt Contract

Use this reference for KKT, KKT Loop, and KKT Model workflows. The point is not literal numeric optimization. The point is to make coding-agent behavior more predictable by translating implementation work into constrained modeling.

## Request Intake

Start by translating rough user input into a compact intent frame, then let discovery fill in repo facts, constraints, and validation paths. Do not require the user to pre-write a complete KKT request.

Capture user meaning first:

```yaml
request_intake:
  intent:
    user_goal:
      value:
      confidence: explicit | inferred | assumption | unknown
    desired_behavior:
      value:
      confidence: explicit | inferred | assumption | unknown
    user_visible_success:
      value:
      confidence: explicit | inferred | assumption | unknown
    scope_boundary:
      value:
      confidence: explicit | inferred | assumption | unknown
    examples:
      value:
      confidence: explicit | inferred | assumption | unknown
    priority_signals:
      value:
      confidence: explicit | inferred | assumption | unknown
    explicit_user_constraints:
      value:
      confidence: explicit | inferred | assumption | unknown
    ambiguity_log:
      value:
      confidence: explicit | inferred | assumption | unknown
    can_continue_to_discovery:
      value: true | false
      confidence: explicit | inferred | assumption | unknown
  execution_mode:
    value: implement | loop | model_only
    confidence: explicit | inferred | assumption | unknown
```

Discovery adds repo-grounded intake:

```yaml
request_intake:
  discovery:
    repo_facts:
      value:
      confidence: observed | inferred | assumption | unknown
    discovered_constraints:
      value:
      confidence: observed | inferred | assumption | unknown
    validation_paths:
      value:
      confidence: observed | inferred | assumption | unknown
    remaining_unknowns:
      value:
      confidence: observed | inferred | assumption | unknown
```

Use this intake process before modeling:

1. Parse explicit user statements first.
2. Ask only the smallest useful set of meaning-focused questions before discovery.
3. Inspect relevant repo context to infer discoverable constraints, validation paths, and likely technical non-goals.
4. Label intent fields as explicit, inferred, assumption, or unknown; label discovery fields as observed, inferred, assumption, or unknown.
5. Ask the user only when an unknown materially changes product meaning, feasibility, risk, scope, or execution mode.
6. Otherwise proceed with conservative assumptions and carry them into the model.

Default execution modes:

- KKT: `implement`.
- KKT Loop: `loop`.
- KKT Model: `model_only`.

Do not ask the user to identify files, routes, schemas, tests, validation commands, or config that can be discovered locally.

## Mathematical Translation

Treat a coding task as:

```text
choose implementation decisions
that best satisfy the user's objective
subject to project, architecture, security, data, UI, infrastructure, validation, and scope constraints
```

Use mathematical modeling semantics:

- Feasibility comes before optimization.
- Hard constraints are predicates; violating one makes a plan infeasible.
- Soft constraints compare feasible plans.
- Decision variables have domains.
- Binding constraints are solution-audit findings: active or limiting constraints that explain why the selected solution has its shape.
- Validation is an execution certificate, not a vibes-based summary.

## Method Profiles

Choose the smallest modeling profile that fits the skill and request:

- Daily profile (`kkt`): use compact intake, discovery facts, hard-constraint feasibility, lexicographic ranking, approval before implementation, and validation certificate. Keep formal method names mostly hidden unless they explain a material tradeoff.
- Deep profile (`kkt-model`): use the layered method catalog for intent capture, discovery, coupling, method selection, candidate comparison, and user tradeoff decisions.
- Loop profile (`kkt-loop`): front-load deeper planning, show the final model for approval, then create durable workspace files and execute with evidence-backed continuation.

Use `references/layered-modeling-methods.md` when the request needs method selection beyond the daily profile.

## State Persistence Tiers

Use `references/state-contract.md` as the authoritative state contract.

- Daily tier (`kkt`): no durable files by default. If state is needed, use one compact `kkt.yaml`; do not create Markdown layer artifacts.
- Model tier (`kkt-model`): use `.kkt-model/<slug>/kkt.yaml`, `intent.md`, `discovery.md`, and `model.md` when the model needs durable context.
- Loop tier (`kkt-loop`): use `.kkt/<slug>/kkt.yaml`, `intent.md`, `discovery.md`, `model.md`, `plan.md`, `progress.md`, `evidence.md`, and `notes.md`.

YAML carries canonical state, status, decisions, method invocations, and artifact references. Markdown carries rich context that would become lossy if compressed into YAML.

## Optimization Model

```yaml
optimization_model:
  request_intake:
    User meaning plus discovered repo facts, discovered constraints, validation paths, and execution mode.

  objective:
    The outcome to optimize for.

  system_state:
    facts:
      Repo, runtime, product, architecture, or test facts discovered by inspection.
    assumptions:
      Claims not yet proven but safe enough to proceed with, unless contradicted.

  decision_variables:
    The choices the agent is allowed to make. Each variable must include an allowed domain.

  constraint_contract:
    hard:
      Must be satisfied for feasibility.
    soft:
      Preferences used to compare feasible plans.

  feasible_region:
    Candidate plans that satisfy all hard constraints.

  objective_order:
    Lexicographic selection rule for choosing among feasible plans.

  selected_plan:
    The feasible plan chosen and why it dominates alternatives.

  solution_audit:
    binding_constraints:
      Constraints that are active, limiting, or materially shaping the selected plan.
    non_binding_constraints:
      Constraints checked but not limiting for the selected plan.

  sensitivity_analysis:
    What would change if a binding constraint were relaxed.
```

`request_intake.intent.user_visible_success` is user meaning, not a validation command. Discovery and modeling translate it into acceptance criteria, validation plan, and evidence required before implementation.

## Implementation Approval Gate

For implementation modes (`implement` and `loop`), show the final modeling result and wait for explicit user approval before mutating files, creating a durable workspace, launching a goal, or starting execution.

The modeling result must include:

- objective;
- selected plan;
- rejected alternatives or infeasible paths when relevant;
- binding constraints;
- expected files, modules, surfaces, or workflow areas to touch;
- validation plan;
- residual risk.

If the user changes the model, re-optimize before asking for approval again.

## Execution Contract

```yaml
execution_contract:
  acceptance_criteria:
    Checkable end states that prove the selected plan is complete.
  validation_plan:
    Commands, checks, artifacts, or manual verification needed.
  evidence_required:
    Proof that must be recorded before completion.
  stop_conditions:
    Conditions that require user input, termination, or re-optimization.
  continuation_policy:
    Runtime rules for selecting next steps and updating evidence.
```

## Decision Variables

Define variables as implementation choices with explicit domains.

Poor:

```yaml
database: decide storage
```

Better:

```yaml
decision_variables:
  persistence_strategy:
    allowed_domain:
      - reuse existing table
      - add migration and new table
      - add nullable fields to existing table
    disallowed_options:
      - external managed auth service
      - plaintext secret storage
    chosen_value: add migration and new table
    rationale: existing table cannot represent expiry and one-time token use safely
```

Common variables:

- files or modules to modify;
- endpoint shape;
- data storage or migration strategy;
- state ownership;
- UI placement;
- validation strategy;
- compatibility strategy;
- dependency strategy;
- rollout or fallback strategy.

## Constraint Contract

Hard constraints:

- explicit user constraints and scope boundaries;
- security and privacy requirements;
- public API compatibility;
- data integrity;
- existing framework/runtime constraints;
- no new dependencies when specified;
- no destructive action without approval.

Soft constraints:

- minimal diff;
- existing style and architecture fit;
- readability;
- testability;
- maintainability;
- low operational risk.

## Solution Audit

Binding constraints:

- no database migration allowed;
- existing API shape must remain stable;
- missing credentials prevent live verification;
- fragile legacy code makes broad refactor too risky;
- UI surface area is constrained.

Non-binding constraints:

- constraints that were checked but did not affect the selected solution.

## Objective Order

Use lexicographic priority rather than fake numeric scores:

1. Satisfy the user request.
2. Preserve correctness, security, data integrity, and public contracts.
3. Minimize blast radius.
4. Match existing architecture and conventions.
5. Improve maintainability where cheap.
6. Prefer validation clarity over elegance.

If a lower-priority objective conflicts with a higher-priority objective, the higher-priority objective wins.

## Candidate Plan Comparison

For each serious plan, capture:

```yaml
plan:
  summary:
  feasibility:
  hard_constraints_satisfied:
  hard_constraints_violated:
  binding_constraints:
  decision_variable_assignments:
  tradeoffs:
  validation_path:
  residual_risk:
```

Reject any plan with hard-constraint violations unless the user explicitly relaxes that constraint.

## Re-Optimization

Re-optimize when new evidence changes:

- system state;
- feasible region;
- hard constraints;
- selected-plan binding constraints or sensitivity analysis;
- selected decision-variable values;
- validation feasibility.

Do not re-optimize merely because another plan feels more elegant.

## Execution Certificate

A completion certificate must answer:

```text
Objective satisfied?
Hard constraints satisfied?
Binding constraints respected?
Which evidence proves the acceptance criteria?
What validation could not be run?
What residual risk remains?
```

Evidence can include tests, typechecks, builds, lint checks, manual runtime checks, screenshots, logs, database rows, API responses, or explicit reasoning when executable validation is not available.
