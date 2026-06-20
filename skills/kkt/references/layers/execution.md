# Execution Layer

Use this internal layer to implement an approved KKT model with focused edits. This is not a public skill entrypoint.

## Inputs

- Approved modeling layer contract.
- Execution contract or `plan.md`.
- Existing `kkt.yaml`, `model.md`, `plan.md`, `progress.md`, `evidence.md`, and `notes.md`, if continuing durable state.

## Workflow

1. Confirm the model is approved for execution.
2. Read the selected plan, binding constraints, stop conditions, validation plan, and expected files or surfaces.
3. Make the smallest feasible implementation step.
4. Update progress and evidence state when durable state exists.
5. Stop for destructive actions, credentials, paid services, scope expansion, missing approval, or model invalidation.
6. End with a layer contract and validation readiness.

## Output Contract

```yaml
layer_contract:
  layer: execution
  status: complete | blocked
  method_used:
  inputs_consumed:
  outputs:
    files_changed:
    execution_decisions:
    deviations:
    progress:
    evidence_generated:
    validation_readiness:
  decisions:
  assumptions:
  unknowns:
  artifact_refs:
  next_layer_readiness:
```

## Guardrails

- Do not execute unapproved models.
- Do not broaden scope because a refactor looks cleaner.
- Do not continue after a re-optimization trigger.
