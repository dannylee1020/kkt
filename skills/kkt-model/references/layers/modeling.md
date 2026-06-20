# Modeling Layer

Use this internal layer to select a feasible plan or decision brief from intent and discovery. This is not a public skill entrypoint.

## Inputs

- Intent layer contract.
- Discovery layer contract.
- Existing `kkt.yaml`, `intent.md`, `discovery.md`, and `model.md`, if continuing durable state.

## Workflow

1. Read intent and discovery before selecting a method.
2. Apply the hard-constraint feasibility gate.
3. Select the modeling method that fits the decision shape: lexicographic ranking, decision tree, shortest path, ordinal MCDA, pairwise questions, or outranking shortlist.
4. Compare candidate plans or models.
5. Reject infeasible options with violated constraints.
6. Identify the selected plan, binding constraints, sensitivity notes, and execution-contract implications.
7. Write or update the modeling layer state when durable state exists.
8. End with a layer contract and `next_layer_readiness`.

## Output Contract

```yaml
layer_contract:
  layer: modeling
  status: complete | blocked
  method_used:
  inputs_consumed:
  outputs:
    selected_plan:
    candidates:
    infeasible_options:
    binding_constraints:
    sensitivity:
    execution_contract_implications:
  decisions:
  assumptions:
  unknowns:
  artifact_refs:
  next_layer_readiness:
```

## Guardrails

- Do not use fake numeric scores for subjective criteria.
- Do not skip feasibility checks.
- Do not edit code or launch execution.
