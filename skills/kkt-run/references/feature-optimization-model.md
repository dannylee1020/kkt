# Constrained Optimization Contract

KKT is not a checklist. Every planning path must apply the same constrained-optimization kernel, even when the representation is compressed.

## Optimization Kernel

1. Define the objective.
2. Define decision variables and affected surfaces.
3. Formulate hard constraints and soft preferences.
4. Identify feasible and rejected candidates.
5. Select the best feasible implementation.
6. Identify binding constraints.
7. Define the validation certificate.

The agent owns modeling judgment. The CLI verifies contract structure and deterministic workflow state; it does not invent candidates or choose the optimum.

## Compressed Contract

The default `$kkt` representation is concise but must contain every kernel stage:

```markdown
## Objective Function

## Decision Variables and Affected Surfaces

## Constraint Functions
- Hard:
- Soft:

## Candidate Feasibility

## Selected Optimum

## Binding Constraints

## Validation Plan and Certificate
```

Rules:

- Keep each section to the smallest useful statement.
- Do not emit empty sections.
- Do not invent alternatives when discovery leaves one feasible candidate; state why it is the only feasible candidate.
- Reject candidates that violate hard constraints before selecting the optimum.
- Explain method names only when they materially affect the choice.
- Validation is the certificate that the selected optimum satisfies the model.

## Deep Contract

Use the full contract in `deep-optimization-model.md` for architecture choices, material alternatives, high-risk changes, unresolved owner tradeoffs, or substantial cross-module work.

## Profile Depth

- `$kkt`: compressed contract by default; escalate when decision complexity or risk requires it.
- `$kkt-model`: deep contract with alternatives, coupling, sensitivity, and owner decisions.
- `$kkt-run`: execute a completed compressed or deep model without re-modeling.
- `$kkt-loop`: use a compressed or deep model based on decision complexity; retain durable execution state.

Task duration does not by itself require deep modeling. A short security or migration change may require deep modeling; a long but straightforward change may use the compressed contract.

## Execution Boundary

The selected model chooses the feasible implementation. Execution plans may add sequencing, tasks, acceptance criteria, stop conditions, and validation commands, but must not replace or silently rewrite the optimization model.

## Re-Optimization

Re-optimize only when evidence changes the objective, constraints, feasible candidates, selected optimum, binding constraints, or validation feasibility. Preference alone is not a reason to restart the model.
