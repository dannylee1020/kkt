# KKT Schemas

Optional serialization reference for copyable shapes. Do not load this on every invocation; use it when writing or auditing durable KKT state, layer contracts, plan-assimilation output, discovery tooling output, or guardrails. The Optimized Plan shape serializes the canonical contract in `feature-optimization-model.md`; it adds no planning semantics.

## Layer Contract

```yaml
layer_contract:
  layer:
  status: pending | complete | blocked
  method_used:
  inputs_consumed:
  outputs:
  decisions:
  assumptions:
  unknowns:
  artifact_refs:
  next_layer_readiness:
```

## Optimized Plan

```yaml
optimized_plan:
  objective_function:
  known_constraints:
    explicit:
    discovered:
    inferred:
    assumptions:
  decision_variables:
    - name:
      allowed_domain:
      chosen_value:
      rationale:
  files_to_modify:
    - path_or_surface:
      change_type:
      rationale:
  constraint_functions:
    hard:
      - name:
        predicate:
        source:
        status:
    soft:
      - name:
        preference:
        source:
        status:
  candidates:
    feasible:
    rejected:
  selected_plan:
  binding_constraints:
  validation_plan:
  validation_proof:
  execution_implications:
  guardrail_variables:
  analysis_extensions:
  residual_risk:
```

## Plan Assimilation

```yaml
plan_assimilation:
  source:
    original_request:
    prior_plan_text:
  extracted_signals:
    proposed_goal:
    proposed_steps:
    suspected_surfaces:
    implied_constraints:
    validation_suggestions:
    open_questions:
    risk_signals:
  classification:
    explicit_user_input:
    plan_assumption:
    discoverable_fact:
    candidate_decision:
    candidate_constraint:
    candidate_validation:
    unknown_or_ambiguous:
  verification:
    verified_facts:
    rejected_claims:
    assumptions_carried:
    searches_or_evidence:
  next_layer_readiness:
```

## Discovery Tooling

```yaml
discovery_tooling:
  available:
    core:
    preferred:
    optional:
    language_native:
  unavailable:
  searches:
    - tool:
      purpose:
      query_or_pattern:
      paths:
      result_summary:
      confidence:
  negative_searches:
    - tool:
      query_or_pattern:
      paths:
      implication:
  fallback_notes:
```

## Execution Contract

```yaml
execution_contract:
  acceptance_criteria:
  validation_plan:
  evidence_required:
  stop_conditions:
  continuation_policy:
```

`planning_contract` in `kkt.yaml` is lightweight plan-tier state metadata, not an alternate Optimized Plan Contract.

## kkt.yaml

```yaml
schema_version: 1
workspace_type: plan | model | run | loop
profile: plan | model | run | loop
status: initialized | modeling | approved | executing | validating | complete | blocked
active_layer: intent | discovery | modeling | execution | validation
layers:
  intent:
    status: pending | complete | blocked
    method: pending | goal_anti_goal | why_how | obstacle_questions | pairwise_questions
    summary: ""
    artifact: intent.md
  discovery:
    status: pending | complete | blocked
    method: pending | naive | traceability_matrix | coupling_map | dsm_lite
    summary: ""
    artifact: discovery.md
  modeling:
    status: pending | complete | blocked
    method: pending | lexicographic | decision_tree | shortest_path | ordinal_mcda | pairwise_ahp | outranking
    summary: ""
    artifact: model.md
  execution:
    status: pending | complete | blocked
    method: pending | smallest_feasible_step | contract_preserving_change
    summary: ""
    artifact: plan.md
  validation:
    status: pending | complete | blocked
    method: pending | acceptance_map | hard_constraint_audit | binding_constraint_audit
    summary: ""
    artifact: evidence.md
method_invocations: []
decision_log: []
planning_contract:
  objective_function:
  files_to_modify:
  constraint_functions:
  decision_variables:
  validation_proof:
artifact_refs:
approval:
  required: true
  status: not_required | pending | approved | rejected
  approved_scope:
stop_conditions: []
loop_state:
  current_task: ""
  tasks: []
  acceptance_criteria: []
  evidence: []
  stop_conditions: []
```

## guardrails.json

```json
{
  "schema_version": 1,
  "source": {
    "workspace_type": "model",
    "workspace": ".kkt/model/<slug>",
    "request": ""
  },
  "constraints": [
    {
      "id": "stable-contract",
      "kind": "architecture",
      "severity": "block",
      "statement": "Preserve the selected model's public contract.",
      "allowed_paths": ["internal/workflow/**"],
      "blocked_paths": ["dist/**"]
    }
  ],
  "change_bounds": {
    "allowed_paths": ["internal/workflow/**"],
    "blocked_paths": [".git/**", ".env*", "dist/**"],
    "require_explicit_approval_outside_allowed": true
  },
  "workflow": {
    "execution_mode": "run",
    "requires_approval_before_mutation": true,
    "requires_validation_before_done": true
  },
  "validation": {
    "acceptance_criteria": [],
    "required_commands": [],
    "evidence_required": ["scope audit confirms only allowed paths changed"]
  },
  "drift_policy": {
    "block_on": [
      "missing_approval",
      "empty_allowed_paths",
      "changed_blocked_path",
      "validation_failed"
    ],
    "warn_on": []
  }
}
```
