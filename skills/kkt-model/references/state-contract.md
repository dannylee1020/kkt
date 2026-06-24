# KKT State Contract

Use this contract when KKT state must survive across layers, agent turns, or coding agents. The goal is not to store everything in YAML. The goal is to make state handoff explicit, inspectable, and hard to confuse.

Layers are internal contract boundaries. They are not public skills and should not be exposed as user commands.

## Persistence Tiers

| Tier | Skill | Durable files | Use when |
| --- | --- | --- | --- |
| Plan | `kkt` | none by default; optional `.kkt/kkt.yaml` | the task is small enough that rich Markdown context would be overhead. |
| Model | `kkt-model` | `.kkt/model/<slug>/kkt.yaml`, `intent.md`, `discovery.md`, `model.md` | the output is a durable model or decision brief before execution. |
| Loop | `kkt-loop` | `.kkt/loop/<slug>/kkt.yaml`, `intent.md`, `discovery.md`, `model.md`, `plan.md`, `progress.md`, `evidence.md`, `notes.md` | the task is long-running, multi-step, or needs continuation. |

Use the `kkt` CLI for deterministic state scaffolding and validation:

```text
kkt start plan "<request>"
kkt start model "<request>"
kkt start loop "<request>"
kkt validate
```

## Canonical Rule

`kkt.yaml` is the canonical state index. Markdown files carry rich context.

- Put statuses, active layer, method choices, decisions, artifact references, approvals, stop conditions, and summaries in YAML.
- Put detailed discovery maps, modeling rationale, plans, evidence logs, and notes in Markdown.
- Do not compress large discovery or modeling context into YAML if doing so would lose useful detail.
- Do not rely on hidden session context for decisions that affect later layers.

## kkt.yaml Shape

```yaml
schema_version: 1
workspace_type: plan | model | loop
profile: plan | model | loop
status: initialized | modeling | approved | executing | validating | complete | blocked
active_layer: intent | discovery | modeling | execution | validation
layers:
  intent:
    status: pending | complete | blocked
    method: goal_anti_goal | why_how | obstacle_questions | pairwise_questions
    summary: ""
    artifact: intent.md
  discovery:
    status: pending | complete | blocked
    method: naive | traceability_matrix | coupling_map | dsm_lite
    summary: ""
    artifact: discovery.md
  modeling:
    status: pending | complete | blocked
    method: lexicographic | decision_tree | shortest_path | ordinal_mcda | pairwise_ahp | outranking
    summary: ""
    artifact: model.md
  execution:
    status: pending | complete | blocked
    method: smallest_feasible_step | contract_preserving_change
    summary: ""
    artifact: plan.md
  validation:
    status: pending | complete | blocked
    method: acceptance_map | hard_constraint_audit | binding_constraint_audit
    summary: ""
    artifact: evidence.md
method_invocations:
  - layer:
    method:
    reason:
    inputs:
    outputs:
decision_log:
  - decision:
    reason:
    constraints:
    alternatives:
    timestamp:
artifact_refs:
  intent:
  discovery:
  model:
  plan:
  progress:
  evidence:
  notes:
approval:
  required: true
  status: not_required | pending | approved | rejected
  approved_scope:
stop_conditions: []
```

Omit artifact keys that do not apply to the tier. For `kkt`, a compact `kkt.yaml` can keep layer summaries inline and leave Markdown artifacts empty or absent.

## Layer Handoff Rules

1. Read `kkt.yaml` first when it exists.
2. Read the prior layer's artifact before acting.
3. Update only the current layer's state unless correcting a clearly stale reference.
4. Append decisions instead of overwriting prior rationale.
5. Set `next_layer_readiness` in the layer output before handing off.
6. If a layer is blocked, record the blocker and the smallest user or system change that would unblock it.
7. If new evidence invalidates an earlier layer, mark the active layer as blocked and re-open the earlier layer instead of silently continuing.

## Artifact Boundaries

- `intent.md`: what the user wants, desired behavior, user-visible success, scope boundaries, examples, priority signals, explicit user constraints, and unresolved meaning questions.
- `discovery.md`: files, symbols, components, functions, workflows, discovered constraints, validation paths, coupling, evidence, confidence, and unknowns.
- `model.md`: method selection, candidates, feasibility, selected plan, binding constraints, sensitivity, and execution implications.
- `plan.md`: execution tasks, acceptance criteria, validation plan, evidence required, stop conditions, and continuation policy.
- `progress.md`: current task, task list, work log, and blockers.
- `evidence.md`: validation map, command outputs, artifacts, and final certificate.
- `notes.md`: observations, assumptions, open questions, and deferred ideas.
