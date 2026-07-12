# KKT Kernel

Mandatory operating rules for every KKT skill. Keep this reference short enough to read on every invocation.

## Invariants

- Capture user meaning before modeling.
- Discover repo facts before asking for repo facts.
- Treat prior plans as scaffold, not truth.
- Check hard constraints before comparing candidates.
- Choose the smallest feasible plan that satisfies the objective and preserves contracts.
- Get approval before mutation when the workflow requires it.
- Validate with evidence before completion.
- Re-optimize only when evidence changes facts, constraints, feasibility, selected decisions, or validation feasibility.

## Question Filter

Classify each unknown before asking:

- `discoverable_fact`: inspect the repo, docs, tests, config, logs, or issues.
- `reversible_default`: choose and label the conservative low-risk default.
- `owner_decision`: ask when product behavior, scope, risk, approval, destructive action, external dependency, credentials, paid service, or execution mode changes.
- `blocking_unknown`: stop when no conservative default remains feasible.

## Completion Certificate

```text
Objective: satisfied / not satisfied
Hard constraints: satisfied / violations listed
Binding constraints: respected / changed
Validation evidence: commands, checks, artifacts, or reason validation was not possible
Residual risk: concise notes
```
