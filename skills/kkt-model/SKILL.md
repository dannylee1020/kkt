---
name: kkt-model
description: Deep constrained optimization modeling for coding and product-engineering decisions. Use when a coding agent should capture user meaning, discover repo constraints and validation paths, inspect the system, identify objectives, system state, decision variables, domains, hard and soft constraint contracts, selected-plan binding constraints, execution-contract implications, multiple feasible models or architectures, sensitivity analysis, and interactive user tradeoffs before any implementation.
license: Apache-2.0
---

# KKT Model

Use this skill when the deliverable is a model, decision brief, or implementation-ready recommendation rather than code. It is for architecture choices, feature shaping, complex implementation options, scope negotiation, and high-impact tradeoffs.

Read `references/kkt-kernel.md` and `references/feature-optimization-model.md` before acting. Read `references/layered-modeling-methods.md` when choosing methods, which is normal for deep modeling. Read `references/state-contract.md` when durable model state is useful. Read `references/plan-assimilation.md` only when prior plan text exists. Read `references/discovery-tooling.md` during discovery. Read `references/schemas.md` only when writing or auditing full state, guardrail, or layer-output shapes.

## Core Rule

Stay non-mutating by default. Intake before modeling. Discovery before asking for repo facts. Select the modeling method that fits the decision shape. Ask only for product, risk, scope, approval, or execution-mode choices that cannot be resolved from inspection or conservative reversible defaults.

## CLI-First Workflow

Use the `kkt` CLI whenever durable model state is useful. The skill owns modeling judgment; the CLI owns workspace creation, state reads, artifact recording, validation, and completion.

```text
kkt start model "<user request>"
kkt status [--json]
kkt next
kkt intent --method <goal_anti_goal|why_how|obstacle_questions|pairwise_questions> "<intent frame>"
kkt discovery --method <naive|traceability_matrix|coupling_map|dsm_lite> "<repo facts and constraints>"
kkt model --method <lexicographic|decision_tree|shortest_path|ordinal_mcda|pairwise_ahp|outranking> "<canonical Optimized Plan Contract>"
kkt guardrails set '<constraints and path bounds JSON>'
kkt guardrails validate
kkt validate
kkt done
```

If `kkt` is missing and durable model state is needed, stop and ask the user to install or upgrade KKT. Do not hand-write replacement state.

## Workflow

1. Capture intent: user goal, desired behavior, user-visible success, scope boundary, examples, priority signals, and explicit user constraints.
2. If prior plan text exists, assimilate it as untrusted scaffold before modeling: extract signals, classify each claim, verify discoverable facts, and treat unverified claims as assumptions or candidates.
3. Run the interactive intent checkpoint before deep discovery: after any quick inspection needed to avoid asking repo-fact questions, ask 1-3 owner-decision questions when goal, success, scope, risk, or tradeoff preference is still ambiguous. For large, high-risk, or especially ambiguous work, run a short Socratic pass using WHY/HOW, obstacle, example/counterexample, or pairwise tradeoff prompts.
4. Inspect relevant code, docs, configs, schemas, routes, tests, UI, infra, issues, or logs before choosing a model. Use `rg` directly for broad text and file discovery, `ast-grep` directly for structural search when syntax matters, `git` for repository state, optional helpers and language-native commands when they provide stronger evidence.
5. Separate explicit requirements, prior-plan assumptions, discovered facts, inferred constraints, assumptions, unknowns, and owner decisions.
6. Build a discovery map when the decision crosses modules, workflows, contracts, or architecture boundaries.
7. Select one intent method, one discovery method, and one modeling method from the layered catalog; record each with the matching `kkt ... --method` command. When no specialized method fits, use the fallback set (`goal_anti_goal`, `traceability_matrix`, `lexicographic`) and record why the fallback is sufficient instead of forcing an advanced method.
8. Produce the canonical Optimized Plan Contract from `feature-optimization-model.md` at deep depth, using `Analysis Extensions` for method rationale, coupling, sensitivity, and unresolved owner decisions.
9. Produce 2-4 candidate models when meaningful alternatives exist; eliminate infeasible models before comparing feasible ones.
10. Compare feasible models by hard-constraint satisfaction, binding constraints, blast radius, maintainability, validation clarity, reversibility, and fit with user intent.
11. Ask the user only for unresolved owner decisions; otherwise select the best feasible model.
12. Translate the selected model into guardrail variables: constraints, allowed paths, blocked paths, validation evidence, and required commands.
13. Record durable output with `kkt intent --method`, `kkt discovery --method`, `kkt model --method`, `kkt guardrails set`, `kkt guardrails validate`, and `kkt validate` when a workspace exists.

## End States

End with one of:

- `Selected model`: one feasible model is recommended and ready for implementation.
- `Decision needed`: the smallest user decisions required to select a model.
- `No feasible model`: the hard constraints that block feasibility and the relaxation that would restore it.

For each serious alternative, include the method used, objective fit, decision-variable assignments, hard-constraint status, binding constraints, tradeoffs, execution-contract implications, residual risks, and when to choose it.

Decision briefs must use the canonical Optimized Plan Contract from `references/feature-optimization-model.md` at deep depth.

## Durable Output

For substantial modeling work, use project-root `.kkt/model/<slug>/` through `kkt` commands. `kkt.yaml` is the state index; Markdown files carry rich intent, discovery, and modeling context. `guardrails.json` carries the modeled constraints and path bounds that `$kkt-run` or `$kkt-loop` will enforce before mutation. A completed model is the reusable handoff: use `kkt run from-model [model-workspace]` for bounded execution or `kkt loop from-model [model-workspace]` for durable iterative execution. Load `references/schemas.md` only when a full copyable shape is needed. Do not create execution files unless switching to `$kkt-run` or `$kkt-loop`.

## Do Not

- Do not modify code unless the user explicitly switches from modeling to implementation.
- Do not invent numeric scores for subjective criteria.
- Do not collapse materially different architectures into one vague plan.
- Do not ask for user input before inspecting discoverable context.
- Do not choose a method because it sounds rigorous; choose it because the decision shape calls for it.
- Do not recommend a model without selected-plan binding constraints and execution implications.
- Do not finish a durable model without guardrail constraints and non-empty allowed paths.
