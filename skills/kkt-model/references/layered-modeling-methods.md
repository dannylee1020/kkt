# Layered Modeling Methods

Use this catalog to choose the smallest method that makes the model reliable. Do not apply every method to every request.

Every layer must produce a contract-shaped output that the next layer can consume without relying on hidden session context.

## Method Profiles

### Plan Profile

Use for ordinary `$kkt` implementation work.

- Capture intent with a compact user goal, desired behavior, user-visible success, scope boundary, and explicit user constraints.
- Discover relevant facts, repo constraints, and validation paths from code, tests, docs, config, schemas, routes, UI, infra, logs, or issues.
- Apply the hard-constraint feasibility gate.
- Rank feasible plans lexicographically.
- Show the final modeling result and wait for approval before implementation.
- Validate with an execution certificate.

Do not expose formal method names in normal output unless they explain a material tradeoff.

### Deep Profile

Use for `$kkt-model` and complex planning.

- Use a goal / anti-goal frame, WHY / HOW ladder, and obstacle questions to clarify user meaning.
- Build a discovery map with traceability, coupling, blast radius, and evidence confidence.
- Select and record one intent method, one discovery method, and one modeling method based on the decision shape.
- If no specialized method fits, use the fallback set: `goal_anti_goal` for intent, `traceability_matrix` for discovery, and `lexicographic` for modeling. Record why the fallback is sufficient; do not invent a new method.
- Compare candidate models, reject infeasible options, and explain binding constraints.
- End with a selected model, a decision brief, or the smallest unresolved user decisions.

### Loop Profile

Use for `$kkt-loop`.

- Front-load deeper modeling before creating the durable workspace.
- Record discovery, coupling, method choice, selected model, and execution contract in the workspace through the `kkt` CLI.
- Show the final modeling result and wait for approval before workspace creation or execution.
- During continuation, re-optimize only when new evidence changes feasibility, constraints, objective fit, or validation feasibility.

## Layer Contract

Each layer output must include:

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

- `method_used` must name the actual method chosen, not a generic profile.
- For `$kkt-model` and `$kkt-loop`, record chosen intent/discovery/modeling methods with `kkt intent --method`, `kkt discovery --method`, and `kkt model --method`.
- The fallback methods are valid method choices, but only when the specialized selector does not materially improve the model.
- `inputs_consumed` must point to prior layer artifacts or summarize session-only input.
- `decisions` must include enough rationale for a different agent to continue.
- `unknowns` must distinguish harmless uncertainty from blocking ambiguity.
- `artifact_refs` should point to `kkt.yaml` and Markdown files when durable state exists.
- `next_layer_readiness` must say whether the next layer can proceed.

## Layer Methods

### Intent Capture

Purpose: turn rough user input into a compact meaning frame that discovery can inspect against the repo.

Methods:

- Goal / anti-goal frame: identify the desired user outcome and what the user wants out of scope.
- WHY / HOW ladder: ask WHY to find the real objective; ask HOW only when it clarifies desired behavior, not repo mechanics.
- Obstacle questions: ask what would make the interpretation unacceptable, too broad, or pointed at the wrong workflow.
- Example / counterexample prompts: ask for examples only when they would prevent a materially wrong interpretation.
- Tradeoff prompt: ask what matters more only when competing user priorities change the model.
- Owner-decision filter: classify each possible question as a discoverable fact, reversible default, owner decision, or blocking unknown before asking.

Adaptive question budget:

- Small or clear task: ask 0-1 clarifying questions.
- Medium task or some ambiguity: ask 1-3 targeted questions.
- Large, high-risk, or ambiguous task: run a short Socratic pass before discovery.
- Never ask the user for files, routes, schemas, test commands, or validation paths that discovery can find.
- Do not ask about low-risk reversible defaults; choose the conservative default, record it as an assumption, and let discovery or validation reopen it if contradicted.
- Ask only for owner decisions: product intent, risk tolerance, scope boundaries, approval, irreversible tradeoffs, destructive actions, external dependencies, paid services, credentials, or execution mode.

Meaning-focused prompt patterns:

- What should be different after this change?
- What workflow or user is affected?
- What should stay out of scope from your perspective?
- Can you give an example or counterexample?
- What would be an unacceptable interpretation?
- If there is a tradeoff, what matters more?
- Is this a product/risk/scope decision only the owner can make, or can discovery answer it?

Output contract:

- user goal;
- desired behavior;
- user-visible success;
- scope boundary;
- examples and counterexamples, if useful;
- priority signals;
- explicit user constraints;
- ambiguity log;
- question filter result for any unresolved high-impact question;
- whether discovery can proceed.

### Discovery

Purpose: convert intent into inspected facts and traceable system context.

Methods:

- Traceability matrix: map user intent to files, modules, routes, schemas, tests, docs, config, and runtime boundaries.
- Dependency / coupling map: identify upstream and downstream modules, callers, public interfaces, state owners, and side effects.
- DSM-lite: use a small dependency matrix when a change crosses several modules or workflows and blast radius is unclear.
- Impact radius: classify likely change radius as low, medium, or high using discovered evidence.
- Naive/local discovery: use direct `rg`, file reads, and tests for small changes with obvious locality.

Output contract:

- relevant artifacts and why they matter;
- symbols, components, functions, routes, schemas, tests, or workflows discovered;
- discovered repo, architecture, validation, security, data, UI, infrastructure, or scope constraints;
- candidate validation paths and their confidence;
- coupling and blast-radius notes;
- evidence confidence for each important fact;
- unknowns that remain after inspection.

### Modeling

Purpose: select the feasible plan or decision brief from intent and discovery.

Always apply the hard-constraint feasibility gate before ranking.

Use this selector:

| Decision shape | Method | Use when |
| --- | --- | --- |
| Normal implementation choice | Lexicographic ranking | Priorities are clear and correctness / safety / contracts outrank lower preferences. |
| Branching facts determine feasibility | Decision tree | Repo facts such as existing adapter, migration allowance, credentials, or public API compatibility eliminate paths. |
| Staged transition or rollout | Shortest path | There are real states, transitions, costs, and forbidden states. |
| Several feasible options with comparable tradeoffs | Ordinal MCDA | Alternatives differ across blast radius, maintainability, validation clarity, speed, or reversibility. |
| User priorities are unclear | AHP-style pairwise questions | The choice depends on risk tolerance, speed, migration tolerance, or architecture preference. |
| No feasible option clearly dominates | Outranking shortlist | Complex architecture choices need a defensible shortlist rather than a single forced winner. |

Do not invent numeric scores for subjective criteria. Prefer ordinal labels unless the user supplies actual weights.

Output contract:

- method selected and why;
- objective;
- known constraints grouped as explicit, discovered, inferred, and assumptions;
- decision variables with allowed domains, chosen values, and rationale;
- candidate plans or models, including feasible and rejected options;
- infeasible options and violated constraints;
- selected plan and why it dominates alternatives;
- binding constraints and non-binding constraints checked;
- validation plan;
- sensitivity notes;
- execution-contract implications.

### Execution

Purpose: implement the approved model with focused edits.

Methods:

- Smallest feasible step: edit only the surfaces selected by the model.
- Contract-preserving change: preserve public APIs, data contracts, security boundaries, and user non-goals unless approval changed them.
- Re-optimization trigger: stop when new evidence changes feasibility, selected decisions, or validation feasibility.

Output contract:

- files changed;
- decisions made during execution;
- deviations from the approved model;
- progress state;
- evidence generated or still needed;
- re-optimization triggers encountered.

### Validation

Purpose: prove the selected model and execution contract were satisfied.

Methods:

- Acceptance-criteria map: map each criterion to evidence.
- Hard-constraint audit: explicitly check feasibility constraints.
- Binding-constraint audit: verify the selected plan still respects the constraints that shaped it.
- Residual-risk register: list claims not proven by validation.

Output contract:

- validation commands, checks, or artifacts;
- acceptance criteria status;
- hard-constraint status;
- binding-constraint status;
- unvalidated claims;
- residual risk;
- final certificate.
