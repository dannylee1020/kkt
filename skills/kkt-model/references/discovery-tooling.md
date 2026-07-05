# Discovery Tooling

Use this during discovery to prove repo facts accurately without making optional tools blockers.

## Rule

Use the lightest reliable tool that can prove the needed fact. Prefer exact, inspectable evidence over broad inference. Record important negative searches when they shape the model.

Discovery tools are read-only during planning and modeling. Rewrite, update, or codegen modes belong to approved execution.

## Tool Ladder

- Core: `git` for repository boundary, tracked files, status, diffs, history, and changed-path scope.
- Core: `rg` for strings, filenames, config keys, docs, tests, shell scripts, and negative searches.
- Preferred: `ast-grep` or `sg` for syntax-aware search when call sites, imports, handlers, declarations, components, error handling, or API shapes matter.
- Optional: `fd`, `ctags`, `tokei`, `scc`, and repo-native commands when they add confidence.
- Language-native: `go list`, `go test`, `cargo metadata`, `npm` scripts, `tsc`, `pytest --collect-only`, Gradle, Maven, Rails, Django, or equivalent repo tooling.

## Selection Rules

- Use `git ls-files` or `rg --files` before broad recursive scans.
- Use `rg` for text and file discovery.
- Use AST search when syntax matters and available text evidence would be noisy or ambiguous.
- Use language-native tooling for package boundaries, route maps, type contracts, test commands, or generated-code boundaries.
- If a preferred tool is unavailable, record that fact and fall back to the next reliable evidence source.

## Ast-Grep Policy

Allowed during discovery:

```text
ast-grep --pattern '<pattern>' --lang <language>
ast-grep --pattern '<pattern>' --lang <language> --json
ast-grep scan --config sgconfig.yml --json
sg --pattern '<pattern>' --lang <language>
```

Forbidden during planning or modeling:

```text
ast-grep --rewrite ...
ast-grep scan --update-all
ast-grep scan --interactive
```

Rewrites require the execution layer, approval, and normal KKT guardrails.

## Guardrails

- Do not make optional tools a prerequisite for basic KKT operation.
- Do not claim structural certainty from plain text search when AST search or language-native tools are needed and available.
- Do not use generated, vendored, or build-output files as primary evidence unless the task explicitly targets them.
