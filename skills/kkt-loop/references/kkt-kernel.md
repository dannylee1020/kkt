# KKT Kernel

This is the mandatory operating contract for every KKT skill. It should stay short enough to read on every invocation.

## Invariants

- Capture user meaning before modeling.
- Discover repo facts before asking the user for file names, routes, schemas, tests, commands, or config.
- Treat prior host-agent or plan-mode output as scaffold, not truth.
- Check hard constraints before comparing plans.
- Pick the smallest feasible plan that satisfies the user's objective and protects existing contracts.
- Get approval before mutation when the skill is in a planning, model, run, or loop gate.
- Validate with evidence before completion.
- Re-optimize only when new evidence changes facts, constraints, feasible plans, selected decisions, or validation feasibility.

## Question Filter

Before asking, classify the unknown:

- `discoverable_fact`: inspect the repo, docs, tests, config, schemas, routes, logs, or issues.
- `reversible_default`: choose the conservative low-risk default and label it as an assumption.
- `owner_decision`: ask when the answer changes product behavior, scope, risk, approval, destructive actions, external dependencies, credentials, paid services, or execution mode.
- `blocking_unknown`: stop when no conservative default keeps the hard constraints feasible.

## Core Model

Every KKT model or approval-ready plan needs:

- objective function;
- known constraints, with explicit, discovered, inferred, and assumed sources separated;
- decision variables with allowed domains and chosen values;
- files, modules, APIs, workflows, docs, migrations, or operational surfaces expected to change;
- candidate plans, with infeasible options rejected before comparison;
- selected feasible plan and why it dominates alternatives;
- binding constraints that shaped the selected plan;
- validation plan and evidence required;
- residual risk.

## Discovery Rules

- Use `git` and `rg` as the default proof tools.
- Use `ast-grep` or `sg` when syntax matters and text search would be ambiguous.
- Use language-native tools when they reveal package boundaries, types, route maps, generated-code boundaries, or test contracts.
- Optional tools improve confidence but must not block basic KKT operation.
- Record important negative searches when they shape the selected plan.

## Completion Certificate

Finish with:

```text
Objective: satisfied / not satisfied
Hard constraints: satisfied / violations listed
Binding constraints: respected / changed
Validation evidence: commands, checks, artifacts, or reason validation was not possible
Residual risk: concise notes
```
