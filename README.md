![kkt](assets/kkt-readme-modern.png)

<p align="center">
  <strong>Start modeling your implementation</strong>
</p>

<p align="center">
  <a href="package.json"><img alt="version: 0.1.0" src="https://img.shields.io/badge/version-0.1.0-2563eb.svg"></a>
  <a href="package.json"><img alt="license: Apache 2.0" src="https://img.shields.io/badge/license-Apache--2.0-16a34a.svg"></a>
  <a href="package.json"><img alt="node >=22" src="https://img.shields.io/badge/node-%3E%3D22-f97316.svg"></a>
</p>

<hr style="width: 100%; border: 0; border-top: 2px solid #e5e7eb;">

## Why kkt

kkt applies constrained optimization to coding-agent workflows. Named after the Karush-Kuhn-Tucker conditions, it translates mathematical modeling discipline into a practical framework for identifying application constraints, choosing feasible implementation paths, and validating the result.

kkt is distributed as portable Agent Skills: plain `SKILL.md` instructions plus local references. It is designed to work across Codex, Claude Code, Pi, and OpenCode without vendor-specific skill metadata.

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

By default, this installs the skills for Codex, Claude Code, Pi, and OpenCode using their shared skill locations:

```text
~/.agents/skills
~/.claude/skills
```

Target-specific paths are also available for agents that prefer their own skill directory:

```text
Codex:       ~/.agents/skills
Claude Code: ~/.claude/skills
Pi:          ~/.pi/agent/skills
OpenCode:    ~/.config/opencode/skills
```

Useful options:

```bash
--target codex
--target claude
--target pi
--target opencode
--local
--dry-run
```

From a checkout:

```bash
scripts/install.sh
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
  -> selected plan
  -> focused edits
  -> validation certificate
```

The contract:

```text
optimization model   objective, variables, constraints, feasible region, selected plan
solution audit       binding constraints and sensitivity analysis
execution contract   acceptance criteria, validation plan, evidence, stop conditions
```

## Skills

| skill | use it for | output |
| --- | --- | --- |
| `$kkt` | normal feature work, bug fixes, and refactors | model, implement, validate |
| `$kkt-loop` | long-running or autonomous work | durable workspace, progress, evidence |
| `$kkt-model` | architecture choices and tradeoff analysis | selected model or decision brief |

`$kkt` is lightweight and does not create durable files by default.

`$kkt-loop` creates `.kkt/<slug>/` with:

```text
model.md
plan.md
evidence.md
progress.md
notes.md
```

`$kkt-model` is non-mutating by default. It inspects, models, compares feasible alternatives, and asks for user input only when the tradeoff cannot be resolved from the repo.

## Request Shape

kkt turns rough input into a request frame before modeling:

```text
objective
known non-goals
constraints
validation expectations
```

The user does not need to provide all of this upfront. Missing fields are inferred from the codebase when possible and marked as assumptions when needed.

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
