# Layered Modeling Methods

Use this catalog for `$kkt-model`, `$kkt-loop`, or unusually complex `$kkt` work. Do not apply every method to every request.

Every layer must produce enough output for the next layer to continue without hidden session context. Load `schemas.md` only when a copyable layer-contract shape is needed.

## Profiles

### Plan Profile

Use for ordinary `$kkt` implementation work:

- compact intent frame;
- local discovery with `git`, `rg`, and structural search when useful;
- hard-constraint feasibility gate;
- lexicographic ranking of feasible plans;
- approval before implementation;
- validation certificate.

Keep formal method names out of normal output unless they explain a material tradeoff.

### Deep Profile

Use for `$kkt-model` and complex planning:

- ask targeted owner-decision questions after quick inspection;
- build a discovery map with evidence confidence;
- select and record one intent, discovery, and modeling method;
- compare candidate models and reject infeasible options first;
- explain binding constraints, sensitivity, and execution implications.

Fallback set: `goal_anti_goal`, `traceability_matrix`, `lexicographic`. Use it when specialized methods do not materially improve reliability.

### Loop Profile

Use for `$kkt-loop`:

- front-load deeper modeling before execution;
- write durable state through the CLI;
- require approval before workspace execution or goal launch;
- continue from CLI state and evidence;
- re-optimize only when new evidence changes feasibility, constraints, objective fit, or validation feasibility.

## Intent Methods

Purpose: turn rough user input into a meaning frame discovery can test against the repo.

- Goal / anti-goal: desired outcome and explicit non-goals.
- WHY / HOW ladder: find the real objective; ask HOW only when it clarifies desired behavior.
- Obstacle questions: identify interpretations that would be unacceptable, too broad, or aimed at the wrong workflow.
- Example / counterexample: use when examples would prevent a materially wrong interpretation.
- Tradeoff prompt: ask only when competing priorities change the model.
- Owner-decision filter: classify possible questions as discoverable facts, reversible defaults, owner decisions, or blocking unknowns.

Question budget:

- clear task: 0-1 questions;
- medium ambiguity: 1-3 targeted questions;
- large, high-risk, or ambiguous work: short Socratic pass after quick inspection.

## Discovery Methods

Purpose: convert intent into inspected facts and traceable system context.

- Naive/local discovery: direct `git`, `rg`, file reads, and tests for obvious locality.
- Traceability matrix: map intent to files, modules, routes, schemas, tests, docs, config, and runtime boundaries.
- Coupling map: identify callers, public interfaces, state owners, side effects, and downstream contracts.
- DSM-lite: use a small dependency matrix when several modules/workflows make blast radius unclear.
- Structural discovery: use `ast-grep` when syntax matters.

Discovery output must distinguish observed facts, inferred constraints, assumptions, unknowns, validation paths, unavailable preferred tools, fallback notes, and important negative searches.

## Modeling Methods

Always apply the hard-constraint feasibility gate before ranking.

| Decision shape | Method | Use when |
| --- | --- | --- |
| Normal implementation choice | Lexicographic ranking | Priorities are clear and correctness / safety / contracts outrank lower preferences. |
| Branching facts determine feasibility | Decision tree | Repo facts eliminate paths. |
| Staged transition or rollout | Shortest path | There are real states, transitions, costs, and forbidden states. |
| Comparable feasible options | Ordinal MCDA | Alternatives differ across blast radius, maintainability, validation clarity, speed, or reversibility. |
| User priorities are unclear | AHP-style pairwise questions | Risk, speed, migration, or architecture preference determines the choice. |
| No feasible option clearly dominates | Outranking shortlist | Architecture choices need a defensible shortlist rather than a forced winner. |

Do not invent numeric scores for subjective criteria. Prefer ordinal labels unless the user supplies actual weights.

## Execution Methods

- Smallest feasible step: edit only the surfaces selected by the model.
- Contract-preserving change: preserve public APIs, data contracts, security boundaries, and user non-goals unless approval changed them.
- Re-optimization trigger: stop when new evidence changes feasibility, selected decisions, or validation feasibility.

## Validation Methods

- Acceptance-criteria map: map each criterion to evidence.
- Hard-constraint audit: explicitly check feasibility constraints.
- Binding-constraint audit: verify the selected plan still respects the constraints that shaped it.
- Residual-risk register: list claims not proven by validation.

Final validation must include commands, checks, artifacts, unavailable checks, residual risk, and a completion certificate.
