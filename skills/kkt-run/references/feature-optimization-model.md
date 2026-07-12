# Optimized Plan Contract

Use this reference whenever a KKT skill creates, reviews, or repairs a plan. It is the one planning-output contract for compact `$kkt`, fresh `$kkt-loop`, and deep `$kkt-model` work. `schemas.md` is only its optional serialization reference.

## Intake

Capture user goal, desired behavior, user-visible success, scope boundary, examples, priorities, explicit constraints, and execution mode. Let discovery establish repo facts. Treat prior-plan claims as assumptions or candidates until verified.

## Required Shape

Return these sections in this exact order. Do not omit a section: write `None — <reason>` when it is irrelevant.

```markdown
## Objective Function

## Known Constraints
- Explicit:
- Discovered:
- Inferred:
- Assumptions:

## Decision Variables

## Affected Surfaces

## Constraint Functions
- Hard:
- Soft:

## Candidate Feasibility
- Feasible:
- Rejected:

## Selected Plan

## Binding Constraints

## Validation Plan and Proof

## Execution Implications

## Guardrail Variables

## Analysis Extensions

## Residual Risk
```

Meaning:

- **Decision Variables** name the allowed choices, domains, selected values, and rationale.
- **Affected Surfaces** name expected changed and protected files, modules, APIs, data, docs, or operations.
- **Candidate Feasibility** rejects hard-constraint violations before comparing viable choices.
- **Validation Plan and Proof** names commands, checks, artifacts, or explicit limits needed to prove completion.
- **Execution Implications** states what run or loop must materialize; it is not the execution contract itself.
- **Guardrail Variables** states modeled constraints, allowed paths, blocked paths, required validation, and drift policy when durable execution is used.
- **Analysis Extensions** holds method rationale, discovery confidence, coupling, sensitivity, or owner-decision analysis.

## Feasibility and Selection

Hard constraints include explicit non-goals, correctness, security, privacy, data integrity, public contracts, runtime limits, and unapproved destructive or external actions. Soft constraints rank feasible candidates in this order:

1. Satisfy the user request.
2. Preserve correctness, security, data integrity, and public contracts.
3. Minimize blast radius.
4. Match existing architecture.
5. Improve maintainability where cheap.
6. Prefer clear validation.

Do not invent numeric scores for subjective qualities.

## Profile Depth

- **Compact (`kkt`)**: concise entries; only material alternatives; `Analysis Extensions` is usually `None`.
- **Loop (`kkt-loop`, fresh)**: the same shape, with enough staging, validation, and continuation implications to derive the execution contract.
- **Deep (`kkt-model`)**: the same shape, with serious alternatives, method rationale, coupling, sensitivity, and unresolved owner decisions in `Analysis Extensions`.

## Execution Boundary

The Optimized Plan Contract selects the feasible implementation. Run and loop then materialize a separate execution contract: ordered steps, acceptance criteria, validation plan, evidence required, stop conditions, and continuation policy. Do not put mutable task progress in the optimized plan.

## Re-Optimization

Re-optimize only when evidence changes system facts, feasible candidates, hard constraints, selected decisions, binding constraints, or validation feasibility. A material contract change invalidates execution approval.
