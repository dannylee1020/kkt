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

Optionally, install hooks for more reliable, deterministic guardrail flow. **This feature is still in beta**.

```bash
npx @dannylee1020/kkt install --hooks
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
| Where does state live? | Usually chat context | `$kkt` is chat-first; durable `.kkt/` state is opt-in or used by deeper workflows |

Plan mode asks, "What should we do?" kkt asks, "What is the best feasible implementation, given what must stay true?"

## Quick Start

Most users start with `$kkt`:

```text
$kkt <feature, bug fix, or refactor>
```

Use the deeper workflows when the task needs them:

```text
$kkt-model <architecture or tradeoff question>
$kkt-run <implement completed model with bounded execution>
$kkt-loop <long-running implementation, fresh or from a completed model>
```

For a large change, model once and choose the execution strategy afterward:

```text
$kkt-model <large change>
$kkt-run   # bounded implementation
# or
$kkt-loop  # durable iterative implementation
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
| `$kkt` | normal feature work, bug fixes, and refactors | compressed constrained-optimization contract, approval, implementation, validation | none by default; optional `.kkt/kkt.yaml` only by explicit need |
| `$kkt-model` | architecture choices and tradeoff analysis | deep constrained-optimization model with alternatives and sensitivity | `.kkt/model/<slug>/` |
| `$kkt-run` | bounded implementation from a completed model | approved execution with deterministic drift checks | `.kkt/run/<slug>/` |
| `$kkt-loop` | fresh long-running work or iterative execution of a completed model | durable workspace, progress, evidence, completion audit | `.kkt/loop/<slug>/` |


`$kkt-loop` owns continuation rather than a separate planning method. A fresh loop creates the shared model internally; a preplanned loop imports it with `kkt loop from-model [model-workspace]`. Both run and loop materialize their execution plan before approval; loops also materialize tasks and acceptance criteria before approval.

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

Before edits, every selected model must preserve KKT's optimization kernel:

- objective function
- decision variables and affected surfaces
- hard and soft constraint functions
- feasible and rejected candidates
- selected optimum
- binding constraints
- validation certificate required for completion

Routine work uses a compressed contract. Architecture, high-risk, ambiguous, or materially multi-option work uses the deep contract.

Expected final audit:

```text
Objective: satisfied
Hard constraints: satisfied
Binding constraints: respected
Validation evidence: tests, checks, artifacts, or reason validation was not possible
Residual risk: remaining uncertainty
```

## CLI and State

Most users do not need to run the CLI directly. The installed skills use it as a deterministic control plane for durable state, guardrails, approval, validation evidence, and workflow progress.

Reference command groups:

```bash
# workspace creation
kkt start plan|model|loop|run "<request>"
kkt run from-model [model-workspace]
kkt loop from-model [model-workspace]

# record planning and execution state
kkt intent
kkt discovery
kkt model
kkt guardrails set '<json>'
kkt guardrails configure --allowed '<paths>' --command '<validation command>'
kkt plan
kkt task
kkt criteria
kkt progress
kkt evidence

# workflow transitions
kkt approve
kkt next
kkt resume
kkt validate --run
kkt done
kkt block

# diagnostics and beta hook integration
kkt status
kkt show
kkt judge
kkt replay
kkt hooks
kkt hook
```

Run `kkt help` for exact syntax. Transition commands enforce their own readiness checks: `approve` validates model readiness, `next` validates run mutation readiness and loop continuation, `task start` validates loop mutation readiness, and `done` validates finalization. Use `status`, `judge`, `guardrails validate`, and `replay` to diagnose a block rather than as required happy-path ceremony. `kkt start run` is the compact direct-execution workflow; use `kkt run from-model` when importing a completed model.

`kkt status --json` exposes the active layer map, contract version, validation and guardrail checks, loop replay state, task/criterion counts, evidence count, active stop conditions, and next action. Use `kkt guardrails configure` for validated flag-based updates such as `--allowed`, `--blocked`, `--command`, `--evidence`, `--acceptance`, `--stop`, `--block-on`, `--warn-on`, and `--mode`; raw `guardrails set` remains available for complete JSON contracts.

Ordinary `$kkt` tasks stay chat-first by default. They still apply constrained optimization; only the representation is compressed. Durable workflow state, when used, lives under the project root's `.kkt/`; deeper `model`, `run`, and `loop` workflows use that directory for artifacts, guardrails, evidence, and replayable progress. Discovery still uses normal agent tools such as search, shell commands, and repo-native language tooling rather than a KKT-specific discovery command.

## Hooks

Coding agents such as Codex, Claude Code, Pi, and OpenCode can run hooks around tool use. KKT's hook adapters plug into those agent hook systems so an approved `run` or `loop` workspace can enforce its guardrail boundaries while the agent is editing.

Before and after tool execution, the adapter asks the current project whether the proposed or actual file mutation stays inside modeled `allowed_paths` and away from `blocked_paths`. If hooks are not installed or not armed, normal agent behavior is unchanged. If hooks are armed, out-of-scope edits can be blocked deterministically instead of relying only on the agent to remember checkpoints.

> [!WARNING]
> Hooks are beta and installed separately because they modify coding-agent runtime behavior. 

## License

Apache-2.0
