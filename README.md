![kkt](assets/kkt-readme-modern.png)

<p align="center">
  <strong>Start modeling your implementation</strong>
</p>

<p align="center">
  <a href="https://github.com/dannylee1020/kkt/actions/workflows/release-binaries.yml"><img alt="ci status" src="https://img.shields.io/github/actions/workflow/status/dannylee1020/kkt/release-binaries.yml?label=ci"></a>
  <a href="https://github.com/dannylee1020/kkt/actions/workflows/codeql.yml"><img alt="security status" src="https://img.shields.io/github/actions/workflow/status/dannylee1020/kkt/codeql.yml?label=security"></a>
  <a href="https://github.com/dannylee1020/kkt/releases/latest"><img alt="release version" src="https://img.shields.io/github/v/release/dannylee1020/kkt?sort=semver&display_name=tag"></a>
  <a href="LICENSE"><img alt="license: Apache 2.0" src="https://img.shields.io/badge/license-Apache--2.0-2563eb.svg"></a>
</p>

<hr style="width: 100%; border: 0; border-top: 2px solid #e5e7eb;">

kkt applies [constrained optimization](https://en.wikipedia.org/wiki/Constrained_optimization) to coding-agent workflows. Named after the [Karush-Kuhn-Tucker conditions](https://en.wikipedia.org/wiki/Karush%E2%80%93Kuhn%E2%80%93Tucker_conditions), it translates mathematical modeling discipline into a practical framework for identifying application constraints, choosing feasible implementation paths, and validating the result.

## How It Works

```text
Without KKT:

request --> agent --> plan --> edits --> validation


With KKT:

request --> agent --> kkt(optimization modeling) --> edits --> validation
                                 |
                                 v
         objective + constraints + decision variables + proof
```


## The Model

The core idea:

```text
choose
  x in X

maximize
  alignment(user_goal, x)

subject to
  C_app(x)
  C_arch(x)
  C_data(x)
  C_ui(x)
  C_infra(x)
  C_validation(x)
```

where:

- `x` is the implementation decision vector
- `X` is the feasible implementation region
- `C_*` are application constraints
- the selected plan is the best feasible plan, not the first plausible plan
- validation is the certificate that the selected plan satisfies the model

kkt does not implement a literal numerical solver. It borrows the discipline of constrained optimization and applies it to coding-agent decisions: feasibility first, optimization second, validation as the certificate.

## Install

Recommended install:

```bash
npx @dannylee1020/kkt install --target all
```

This installs kkt skills for every supported agent it can find:

- Claude Code: `~/.claude/skills`
- Codex, Pi, and OpenCode: `~/.agents/skills`

Install for one agent instead:

```bash
npx @dannylee1020/kkt install --target claude
npx @dannylee1020/kkt install --target codex
npx @dannylee1020/kkt install --target pi
npx @dannylee1020/kkt install --target opencode
```

Upgrade kkt:

```bash
npx @dannylee1020/kkt upgrade --target all
```

Choose a CLI install location:

```bash
npx @dannylee1020/kkt install --bin-dir ~/.local/bin
```

Alternative shell installer:

```bash
curl -fsSL https://raw.githubusercontent.com/dannylee1020/kkt/main/scripts/install.sh | bash
```

The CLI uses a release binary when available, or builds from source with Go. Use `KKT_VERSION` to pin a release tag, or `KKT_BINARY_URL` to install from an explicit binary URL.

## Why KKT

Most coding-agent workflows turn a request into a plan. That helps, but it carries risk: the plan focuses on what to change, not what must stay unchanged.

kkt shifts the frame from planning to modeling, so the solution is built around the constraints already present in the codebase. It treats implementation as a constrained optimization problem: define the objective, mark the boundaries that cannot move, compare the viable paths, and name the proof that will make the result credible.

Instead of:

```text
build xyz
```

kkt pushes the agent toward:

```text
what is the best feasible implementation,
given what must stay true?
```

For coding agents, "what must stay true" is usually concrete:

- public contracts and API behavior
- architecture boundaries
- files, modules, endpoints, schemas, and migrations
- security, privacy, and data-integrity rules
- UI and product boundaries
- infrastructure and runtime limits
- validation evidence required before completion

the value is forcing feasibility before optimization: reject plans that violate hard constraints, compare the remaining plans, choose the best feasible path, then validate against the model.


## kkt vs plan mode

kkt can replace plan mode, or it can run after plan mode to harden a rough plan. Plan mode sketches the path; kkt checks that path against repo facts, constraints, and validation proof before edits.

| question | plan mode | kkt |
| --- | --- | --- |
| What is it optimizing for? | Coordination and sequencing | Best feasible implementation |
| What comes before edits? | A step list | Goal, constraints, chosen path, validation proof |
| How are assumptions handled? | Often left in the plan | Verified or marked as assumptions |
| When is it enough? | Small to medium work | Work where boundaries, contracts, or validation matter |
| Where does state live? | Usually chat context | Chat-first; optional `.kkt/` state when useful |

Plan mode asks, "What should we do?" kkt asks, "What is the best feasible implementation, given what must stay true?"

## Quick Start

Most users start with `$kkt`:

```text
$kkt <feature, bug fix, or refactor>
```

Use the deeper workflows when the task needs them:

```text
$kkt-model <architecture or tradeoff question>
$kkt-run <implement completed model>
$kkt-loop <long-running implementation>
```

Skill invocation syntax varies by agent:

```text
Codex:       $kkt, $kkt-model, $kkt-run, $kkt-loop
Claude Code: /kkt, /kkt-model, /kkt-run, /kkt-loop
Pi:          /skill:kkt, /skill:kkt-model, /skill:kkt-run, /skill:kkt-loop
OpenCode:    ask OpenCode to use the relevant kkt skill
```

## Choose a Workflow

| workflow | use it for | what it produces | durable state |
| --- | --- | --- | --- |
| `$kkt` | normal feature work, bug fixes, and refactors | compact model, approval, implementation, validation | optional `.kkt/kkt.yaml` |
| `$kkt-model` | architecture choices and tradeoff analysis | selected model or decision brief | `.kkt/model/<slug>/` |
| `$kkt-run` | implementation from a completed model | approved execution with deterministic drift checks | `.kkt/run/<slug>/` |
| `$kkt-loop` | long-running or continuation-heavy work | durable workspace, progress, evidence, completion audit | `.kkt/loop/<slug>/` |

KKT is skill-first. The skills are what you invoke from your coding agent. The CLI is the deterministic tool those skills use for `.kkt/` scaffolding, status, guardrails, state persistence, and validation.

kkt turns rough input into an intent frame:

```text
user goal
desired behavior
user-visible success
scope boundary
explicit user constraints
```

The user does not need to provide all of this upfront. Repo constraints, affected files, and validation paths are discovered from the codebase when possible and marked as assumptions when needed.

When KKT is invoked after a prior plan, it treats the plan as untrusted scaffold:

```text
plan output --> extract signals --> classify claims --> verify facts --> optimize KKT model
```

Plan claims do not become KKT facts until KKT verifies them or explicitly carries them as assumptions.

Before edits, the selected model should name:

- objective function
- files or surfaces to modify
- hard and soft constraint functions
- decision variables with chosen values
- validation proof required for completion

Expected final audit:

```text
Objective: satisfied
Hard constraints: satisfied
Binding constraints: respected
Validation evidence: tests, checks, artifacts, or reason validation was not possible
Residual risk: remaining uncertainty
```

## CLI and State

Agent uses the cli to persist state across layers, continuation and agent turns.

Commands:

```bash
kkt start plan|model|run|loop "<request>"
kkt status [--json]
kkt next [--json]
kkt show [artifact]
kkt guardrails show|set|validate
kkt judge --checkpoint model-ready|pre-mutation|continuation|finalize --json
kkt validate [--run]
kkt done
```

Discovery uses agent tools directly. Use `rg` for broad text and file discovery, then use `ast-grep` for syntax-aware questions such as call sites, imports, handlers, declarations, component patterns, and error-handling shapes. Optional helpers such as `fd`, `ctags`, `tokei`, and repo-native language tools are used when available, but discovery should not be routed through a KKT CLI command.

Advanced workflow commands:

```bash
kkt intent|discovery|model --method <method> "<layer output>"
kkt run from-model [model-workspace]
kkt evidence --for <criterion-id> --command "<command>" "<validation evidence>"
kkt criteria add|satisfy|block
kkt task add|start|done|skip|block
kkt approve
kkt block
kkt resume
kkt replay --check
```

All durable state lives under the project root's `.kkt/`. For plan-tier work, `.kkt/kkt.yaml` can carry the compact planning contract. For model, run, and loop workspaces, Markdown files hold richer context, and `guardrails.json` carries the deterministic drift contract.

For run and loop workspaces, `kkt judge` checks explicit workflow state: approval, validation, replay state, stop conditions, and changed-path bounds. It is deterministic; semantic code-behavior judgment is not claimed as part of the current guardrail layer.

When guardrails list `validation.required_commands`, use `kkt validate --run` to execute and record deterministic command proof. `kkt evidence` records narrative evidence and criterion mapping; it does not satisfy required command proof by itself.

For loop workspaces, `kkt.yaml` is the canonical current contract, `events.jsonl` is the append-only audit and resume history, and `kkt replay --check` reports drift between the event history and current state without mutating either file.

## License

Apache-2.0
