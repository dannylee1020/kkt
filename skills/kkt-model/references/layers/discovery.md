# Discovery Layer

Use this internal layer to convert intent into inspected facts and traceable system context. This is not a public skill entrypoint.

## Inputs

- Intent layer contract.
- Existing `kkt.yaml`, `intent.md`, and `discovery.md`, if continuing durable state.
- Relevant repository files, docs, tests, config, schemas, routes, UI, infra, logs, or issues.

## Workflow

1. Read the intent frame before searching.
2. Choose the smallest discovery method that fits the blast radius: naive/local discovery, traceability matrix, coupling map, or DSM-lite.
3. Inspect relevant artifacts with evidence.
4. Record discovered files, symbols, components, functions, workflows, constraints, variables, validation paths, coupling, and confidence.
5. Mark unknowns that remain after inspection.
6. Write or update the discovery layer state when durable state exists.
7. End with a layer contract and `next_layer_readiness`.

## Output Contract

```yaml
layer_contract:
  layer: discovery
  status: complete | blocked
  method_used:
  inputs_consumed:
  outputs:
    relevant_artifacts:
    discovered_symbols:
    coupling:
    constraints:
    variables:
    validation_paths:
    evidence_confidence:
  decisions:
  assumptions:
  unknowns:
  artifact_refs:
  next_layer_readiness:
```

## Guardrails

- Do not edit files.
- Do not pick the final implementation plan.
- Do not treat uninspected guesses as discovered facts.
