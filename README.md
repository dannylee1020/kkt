![kkt](assets/kkt-readme-modern.png)

<p align="center">
  <strong>Start modeling your implementation</strong>
</p>

<p align="center">
  <a href="package.json"><img alt="license: Apache 2.0" src="https://img.shields.io/badge/license-Apache--2.0-16a34a.svg"></a>
  <a href="package.json"><img alt="node >=22" src="https://img.shields.io/badge/node-%3E%3D22-f97316.svg"></a>
</p>

<hr style="width: 100%; border: 0; border-top: 2px solid #e5e7eb;">

kkt applies [constrained optimization](https://en.wikipedia.org/wiki/Constrained_optimization) to coding-agent workflows. Named after the [Karush-Kuhn-Tucker conditions](https://en.wikipedia.org/wiki/Karush%E2%80%93Kuhn%E2%80%93Tucker_conditions), it translates mathematical modeling discipline into a practical framework for identifying application constraints, choosing feasible implementation paths, and validating the result.

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

## Install

Install with curl:

```bash
curl -fsSL https://raw.githubusercontent.com/dannylee1020/kkt/main/scripts/install.sh | bash
```

Upgrade an existing install:

```bash
curl -fsSL https://raw.githubusercontent.com/dannylee1020/kkt/main/scripts/install.sh | bash -s -- upgrade
```

By default, this installs the skills for Codex, Claude Code, Pi, and OpenCode using their shared skill locations:

```text
~/.agents/skills
~/.claude/skills
```

Target-specific installs use the same shared path for Codex, Pi, and OpenCode:

```text
Codex:       ~/.agents/skills
Claude Code: ~/.claude/skills
Pi:          ~/.agents/skills
OpenCode:    ~/.agents/skills
```

Useful options:

```bash
--target codex
--target claude
--target pi
--target opencode
--local
--force
--dry-run
```

From a checkout:

```bash
scripts/install.sh
scripts/install.sh upgrade
```

## Quick Start

Use the skill syntax supported by your agent:

```text
Codex:       $kkt
Claude Code: /kkt
Pi:          /skill:kkt
OpenCode:    ask OpenCode to use the kkt skill
```

Start with a rough request:

```text
Use $kkt to add export-to-csv for reports.
```

Add constraints only when you already know them:

```text
Use $kkt to add export-to-csv for reports.

Constraints:
- Reuse the existing reports api.
- Do not change billing code.
- Do not add dependencies.

Validation:
- Add or update the smallest useful test.
- Run the relevant test command if available.
```

kkt should inspect the repo, infer discoverable constraints, and ask only for decisions that materially affect feasibility, product behavior, risk, or execution mode.

## How It Works

Default agent flow:

```text
user request
  -> plausible plan
  -> edits
  -> summary
```

with kkt:

```text
user request
  -> request intake
  -> constraint discovery
  -> feasible region
  -> optimization modeling with top N plans.
  -> approval
  -> focused edits
  -> validation certificate
```

## Skills

| skill | use it for | output |
| --- | --- | --- |
| `$kkt` | normal feature work, bug fixes, and refactors | lightweight model, approval, implementation, validation |
| `$kkt-loop` | long-running or autonomous work | deeper planning, approval, durable workspace, progress, evidence |
| `$kkt-model` | architecture choices and tradeoff analysis | selected model or decision brief |

`$kkt` is lightweight and does not create durable files by default. It shows the selected model for approval before editing.

Persistence tiers:

| tier | skill | durable state |
| --- | --- | --- |
| daily | `$kkt` | none by default; optional compact `kkt.yaml` for small state handoff |
| model | `$kkt-model` | `.kkt-model/<slug>/kkt.yaml`, `intent.md`, `discovery.md`, `model.md` |
| loop | `$kkt-loop` | `.kkt/<slug>/kkt.yaml`, `intent.md`, `discovery.md`, `model.md`, `plan.md`, `progress.md`, `evidence.md`, `notes.md` |

`kkt.yaml` is the canonical state index. Markdown files hold richer context when YAML would lose detail.

`$kkt-loop` creates `.kkt/<slug>/` with:

```text
kkt.yaml
intent.md
discovery.md
model.md
plan.md
evidence.md
progress.md
notes.md
```

It plans first, asks for approval, then creates the workspace and executes through the durable loop.

`$kkt-model` is non-mutating by default. It inspects, models, compares feasible alternatives, and asks for user input only when the tradeoff cannot be resolved from the repo.

Advanced methods such as coupling maps, decision trees, staged-path planning, and tradeoff ranking are available when deeper modeling is needed, while daily `$kkt` stays compact.

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
