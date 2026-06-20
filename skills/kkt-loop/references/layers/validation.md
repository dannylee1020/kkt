# Validation Layer

Use this internal layer to prove the selected model and execution contract were satisfied. This is not a public skill entrypoint.

## Inputs

- Execution layer contract.
- Execution contract or `plan.md`.
- Existing `kkt.yaml`, `plan.md`, `progress.md`, `evidence.md`, and `notes.md`, if continuing durable state.

## Workflow

1. Map acceptance criteria to required evidence.
2. Run the smallest useful validation checks available.
3. Audit hard constraints and binding constraints explicitly.
4. Record commands, outputs, artifacts, unvalidated claims, and residual risk.
5. Update validation state when durable state exists.
6. End with the final certificate or a blocker.

## Output Contract

```yaml
layer_contract:
  layer: validation
  status: complete | blocked
  method_used:
  inputs_consumed:
  outputs:
    validation_evidence:
    acceptance_criteria_status:
    hard_constraints_status:
    binding_constraints_status:
    unvalidated_claims:
    residual_risk:
    final_certificate:
  decisions:
  assumptions:
  unknowns:
  artifact_refs:
  next_layer_readiness:
```

## Guardrails

- Do not claim completion without evidence or an explicit validation limitation.
- Do not hide failed or unavailable checks.
- Do not make product edits unless the user explicitly switches back to execution.
