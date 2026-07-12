# Plan Assimilation

Use this only when KKT is invoked after a host agent or default plan mode has already produced a plan.

## Rule

Prior plan text is an untrusted scaffold. Consume whatever exists, extract useful signals, verify discoverable claims, and rebuild the canonical Optimized Plan Contract from KKT's own intent and discovery.

Do not ask the user to reformat prior plan output.

## Checklist

1. Preserve the original user request and explicit user constraints.
2. Extract signals from the prior plan:
   - proposed goal and steps;
   - suspected files, modules, APIs, schemas, routes, tests, docs, or operational surfaces;
   - stated or implied constraints;
   - validation suggestions;
   - open questions and risk signals.
3. Classify each signal:
   - `explicit_user_input`;
   - `plan_assumption`;
   - `discoverable_fact`;
   - `candidate_decision`;
   - `candidate_constraint`;
   - `candidate_validation`;
   - `unknown_or_ambiguous`.
4. Verify discoverable facts through repo inspection before using them as facts.
5. Keep unverified claims as assumptions or candidates, never as discovered facts.
6. Fill the missing sections of the canonical Optimized Plan Contract; do not promote unverified claims into facts.
7. Reject or revise the prior plan when it violates hard constraints, skips cheaper feasible paths, or names validation that does not exist.

## Anti-Anchoring Checks

- Which prior-plan claims were verified?
- Which claims remain assumptions?
- Which constraints or decision variables did the plan miss?
- What smaller or safer feasible plan exists?
- What validation proof would certify the selected plan?

## Guardrails

- Do not enforce a schema on host plan-mode output.
- Do not treat prior-plan file names, routes, commands, or architecture claims as facts before inspection.
- Do not rubber-stamp the prior plan as the selected KKT plan.
- Do not ask the user for repo facts the discovery layer can verify.
