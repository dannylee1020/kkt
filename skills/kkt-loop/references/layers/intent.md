# Intent Layer

Use this internal layer to turn a rough user request into a compact user-meaning contract for discovery. This is not a public skill entrypoint.

## Inputs

- User request.
- Existing `kkt.yaml` and `intent.md`, if continuing durable state.
- Relevant prior context supplied in the conversation.

## Workflow

1. Capture what the user wants, what should be different, what success looks like to the user, and what is explicitly in or out of scope.
2. Record examples, counterexamples, priority signals, and explicit user constraints when the user provides them.
3. Keep repo constraints, implementation constraints, and validation paths for discovery and modeling unless the user stated them explicitly.
4. Ask the smallest useful number of meaning-focused questions:
   - small or clear task: ask 0-1 questions;
   - medium task or some ambiguity: ask 1-3 targeted questions;
   - large, high-risk, or ambiguous task: run a short Socratic pass before discovery.
5. Apply the owner-decision filter before asking:
   - discoverable fact: defer to discovery instead of asking;
   - reversible default: assume the conservative low-risk option and record it;
   - owner decision: ask when the answer changes product behavior, risk, scope, approval, or execution mode;
   - blocking unknown: stop when no conservative default keeps the hard constraints feasible.
6. Do not ask questions that discovery can answer from the repo.
7. Write or update the intent layer state when durable state exists.
8. End with a layer contract and `next_layer_readiness`.

## Output Contract

```yaml
layer_contract:
  layer: intent
  status: complete | blocked
  method_used:
  inputs_consumed:
  outputs:
    user_goal:
    desired_behavior:
    user_visible_success:
    scope_boundary:
    examples:
    priority_signals:
    explicit_user_constraints:
    ambiguity_log:
    question_filter:
    can_continue_to_discovery:
  decisions:
  assumptions:
  unknowns:
  artifact_refs:
  next_layer_readiness:
```

## Guardrails

- Do not select implementation plans.
- Do not perform deep discovery.
- Do not ask the user to enumerate files, repo constraints, test commands, schemas, routes, or config that can be discovered locally.
- Do not ask about low-risk reversible defaults before discovery; record the default as an assumption.
- Do not treat inferred repo constraints or validation paths as intent-layer outputs.
- Do not hide blocking ambiguity in assumptions.
