![kkt](assets/kkt-readme-modern.png)

<p align="center">
  <strong>Start modeling your implementation</strong>
</p>

<p align="center">
  <a href="LICENSE"><img alt="license: Apache 2.0" src="https://img.shields.io/badge/license-Apache--2.0-16a34a.svg"></a>
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

## Choose a Surface

| surface | what it offers | install | use when |
| --- | --- | --- | --- |
| Skills | Lightweight manual KKT workflows inside your coding agent. Includes `$kkt`, `$kkt-model`, and `$kkt-loop`. | `scripts/install.sh` | You want to try KKT with explicit skill invocation and a small setup surface. |
| CLI | Agent-invoked KKT workflow state through global agent instructions and `.kkt/<run>/` files. | `scripts/install-cli.sh`, then `kkt init <agent>` | You want the coding agent to call KKT during normal coding work. |

The skills path is the manual surface. The CLI path sets up the coding agent to call KKT from `AGENTS.md`, `CLAUDE.md`, or the equivalent instruction file.

## Install

Install skills:

```bash
curl -fsSL https://raw.githubusercontent.com/dannylee1020/kkt/main/scripts/install.sh | bash
```

The skills installer auto-detects supported coding agents and installs KKT skills into their global skill directories. Use an explicit target when needed:

```bash
scripts/install.sh --target codex
scripts/install.sh --target claude
scripts/install.sh --target all
```

Install the CLI:

```bash
curl -fsSL https://raw.githubusercontent.com/dannylee1020/kkt/main/scripts/install-cli.sh | bash
kkt init codex
```

From a checkout:

```bash
scripts/install.sh
scripts/install-cli.sh --bin-dir ~/.local/bin
kkt init codex
```

Supported CLI setup:

| agent | setup command | integration |
| --- | --- | --- |
| Codex | `kkt init codex` | `~/.agents/AGENTS.md` instructions |
| Claude Code | `kkt init claude` | `~/.claude/CLAUDE.md` instructions |
| Pi | `kkt init pi` | `~/.agents/AGENTS.md` instructions |
| OpenCode | `kkt init opencode` | `~/.agents/AGENTS.md` instructions |
| All | `kkt init all` | shared `~/.agents/AGENTS.md` plus `~/.claude/CLAUDE.md` instructions |

## Quick Start

| surface | normal use | notes |
| --- | --- | --- |
| Skills | `$kkt <feature to implement>` | Use `$kkt-model` for deeper architecture tradeoffs and `$kkt-loop` for longer work. |
| CLI | Ask your coding agent for coding work normally. | The agent calls `kkt` when its project instructions say to use KKT. |

Skill invocation varies by agent:

```text
Codex:       $kkt
Claude Code: /kkt
Pi:          /skill:kkt
OpenCode:    ask OpenCode to use the kkt skill
```

CLI setup and debugging commands:

```bash
kkt init codex --dry-run
scripts/install-cli.sh --dry-run
```

The skills installer is shell-only and does not require Node or Go. It copies the KKT skill directories into the detected agent skill roots.

The CLI installer uses `KKT_BINARY_URL` when a prebuilt binary is available. Until release binaries exist, source installs require Go for the CLI build.

## How It Works

```text
Without KKT:

request --> agent --> first plausible plan --> edits --> validation --> finish


With KKT:
request --> agent --> KKT --> optimization modeled plan --> edits --> validation --> finish
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
