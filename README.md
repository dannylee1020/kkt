![kkt](assets/kkt-readme-modern.png)

<p align="center">
  <strong>Start modeling your implementation</strong>
</p>

<p align="center">
  <a href="LICENSE"><img alt="license: Apache 2.0" src="https://img.shields.io/badge/license-Apache--2.0-16a34a.svg"></a>
</p>

<hr style="width: 100%; border: 0; border-top: 2px solid #e5e7eb;">

kkt applies [constrained optimization](https://en.wikipedia.org/wiki/Constrained_optimization) to coding-agent workflows. Named after the [Karush-Kuhn-Tucker conditions](https://en.wikipedia.org/wiki/Karush%E2%80%93Kuhn%E2%80%93Tucker_conditions), it translates mathematical modeling discipline into a practical framework for identifying application constraints, choosing feasible implementation paths, and validating the result.

## How It Works

```text
Without KKT:

request --> agent --> plan --> edits --> validation --> finish


With KKT:

request --> agent --> KKT modeling --> edits --> validation --> finish
                           |
                           v
        constraints --> optimization --> verification
```

## Why kkt
Good implementation plans are shaped as much by what not to do as by what to do.

kkt makes those limits explicit. Before choosing an implementation path, it pushes the agent to identify the constraints that define a safe change: public contracts that must not break, architecture boundaries that must not be crossed, data rules that must not be weakened, and validation that must not be skipped.

Instead of:

```text
build xyz
```

model the work as:

```text
what is the optimized implementation,
given what must stay true?
```

The result is a more disciplined implementation plan: fewer accidental side effects, clearer tradeoffs, smaller edits, and validation tied to the actual constraints of the work.

For coding agents, those constraints are usually concrete:

- existing architecture and public contracts
- files, modules, endpoints, schemas, and migrations
- security, privacy, and data-integrity rules
- ui and product boundaries
- infrastructure and runtime limits
- validation evidence required to prove completion

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

## Interface

KKT is skill-first. Invoke `$kkt`, `$kkt-model`, or `$kkt-loop` from your coding agent. The CLI is the tool those skills rely on for `.kkt/` scaffolding, status, state persistence and validation,

| piece | what it offers | use when |
| --- | --- | --- |
| Skills | Primary KKT workflows inside your coding agent. Includes `$kkt`, `$kkt-model`, and `$kkt-loop`. | You want KKT to guide planning, approval, implementation, and validation. |
| CLI | Deterministic `.kkt/` state scaffolding and validation used by the skills. | Durable state needs to stay consistent across KKT layers and continuations. |


## Install

Install KKT:

```bash
curl -fsSL https://raw.githubusercontent.com/dannylee1020/kkt/main/scripts/install.sh | bash
```

The installer auto-detects supported coding agents, installs KKT skills into their global skill directories, and installs the companion CLI. Plain `install` is safe to rerun: it installs missing skills, keeps existing skills unchanged, and still installs or updates the CLI. Use `upgrade` when you want to replace installed KKT skills with the latest downloaded copy.

```bash
scripts/install.sh --target codex
scripts/install.sh --target claude
scripts/install.sh --target all
scripts/install.sh --bin-dir ~/.local/bin
scripts/install.sh upgrade
```

From a checkout:

```bash
scripts/install.sh
```

## Quick Start

Invoke the skill directly:

```text
$kkt <feature to implement>
$kkt-model <architecture or tradeoff question>
$kkt-loop <long-running implementation>
```

Skill invocation syntax varies by agent:

```text
Codex:       $kkt
Claude Code: /kkt
Pi:          /skill:kkt
OpenCode:    ask OpenCode to use the kkt skill
```

The skills are installed from the downloaded source archive. The CLI is downloaded as a release binary when available, or built from source with Go. Use `KKT_VERSION` to pin a release tag, or `KKT_BINARY_URL` to install from an explicit binary URL.

CLI workflow commands:

```bash
kkt start plan|model|loop "<request>"
kkt status
kkt next
kkt show [artifact]
kkt intent|discovery|model|plan|progress|evidence|notes
kkt criteria add|satisfy|block
kkt task add|start|done|skip|block
kkt approve
kkt block
kkt validate
kkt done
```

For loop workspaces, `kkt.yaml` is the current contract, `events.jsonl` is the append-only event log, and Markdown files hold rich context and evidence.


## Skills

| skill | use it for | output | durable state |
| --- | --- | --- | --- |
| `$kkt` | normal feature work, bug fixes, and refactors | lightweight model, approval, implementation, validation | optional `.kkt/kkt.yaml` |
| `$kkt-model` | architecture choices and tradeoff analysis | selected model or decision brief | `.kkt/model/<slug>/`|
| `$kkt-loop` | long-running or autonomous work | deeper planning, approval, durable workspace, progress, evidence | `.kkt/loop/<slug>/`|

All durable state lives under `.kkt/`. `kkt.yaml` is the canonical state index. Markdown files hold richer context when YAML would lose detail. Advanced methods such as coupling maps, decision trees, staged-path planning, and tradeoff ranking are available when deeper modeling is needed, while `$kkt` stays compact.

## Request Shape

kkt turns rough input into an intent frame before modeling:

```text
user goal
desired behavior
user-visible success
scope boundary
explicit user constraints
```

The user does not need to provide all of this upfront. Repo constraints, affected files, and validation paths are discovered from the codebase when possible and marked as assumptions when needed.

Expected final audit:

```text
Objective: satisfied
Hard constraints: satisfied
Binding constraints: respected
Validation evidence: tests, checks, artifacts, or reason validation was not possible
Residual risk: remaining uncertainty
```

## License

Apache-2.0
