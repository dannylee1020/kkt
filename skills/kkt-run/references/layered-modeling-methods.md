# Layered Modeling Methods

Use this catalog only when method selection materially improves planning. The Optimized Plan Contract is defined in `feature-optimization-model.md`; this reference selects how to fill it.

## Intent Methods

- **Goal / anti-goal**: desired result and explicit non-goals.
- **WHY / HOW ladder**: clarify the real objective.
- **Obstacle questions**: identify unacceptable interpretations or failure modes.
- **Example / counterexample**: prevent a materially wrong interpretation.
- **Pairwise questions**: resolve owner priorities when they change the decision.

Question budget: clear work 0–1; medium ambiguity 1–3; large or risky work gets a short Socratic pass after quick inspection.

## Discovery Methods

- **Naive/local**: direct `git`, `rg`, file reads, and tests for local changes.
- **Traceability matrix**: map intent to code, contracts, tests, docs, config, and operations.
- **Coupling map**: map callers, public interfaces, state owners, side effects, and downstream contracts.
- **DSM-lite**: use a small dependency matrix when cross-module blast radius is unclear.
- **Structural discovery**: use `ast-grep` when text search is ambiguous.

Record observed facts, inferred constraints, assumptions, unknowns, validation paths, and material negative searches.

## Modeling Methods

Check hard-constraint feasibility before ranking.

| Decision shape | Method | Use when |
| --- | --- | --- |
| Normal implementation choice | Lexicographic | Priorities are clear. |
| Branching facts determine feasibility | Decision tree | Repo facts eliminate paths. |
| Staged transition or rollout | Shortest path | Real states and forbidden transitions exist. |
| Comparable feasible options | Ordinal MCDA | Tradeoffs span blast radius, maintainability, validation, speed, or reversibility. |
| User priorities are unclear | Pairwise AHP-style questions | Owner priorities determine the choice. |
| No option dominates | Outranking shortlist | A defensible shortlist is better than a forced winner. |

Fallback: `goal_anti_goal`, `traceability_matrix`, and `lexicographic`. Do not select a method merely because it sounds rigorous.
