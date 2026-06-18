#!/usr/bin/env node
import {
  cpSync,
  existsSync,
  mkdirSync,
  readdirSync,
  readFileSync,
  rmSync,
  statSync,
} from "node:fs";
import { createHash } from "node:crypto";
import { homedir, platform } from "node:os";
import path from "node:path";
import { fileURLToPath } from "node:url";

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const packageRoot = path.resolve(__dirname, "..");
const sourceSkillsRoot = path.join(packageRoot, "skills");
const skillNames = ["kkt", "kkt-loop", "kkt-model"];

const usageText = `KKT skills installer

Usage:
  kkt-skills install [options]
  kkt-skills uninstall [options]
  kkt-skills doctor

Options:
  --target <name>   default | codex | claude | pi | opencode | all
  --local [path]    Install to project-local skill directories. Defaults to cwd.
  --dir <path>      Install to an explicit skill root directory.
  --force           Overwrite existing KKT skill directories.
  --dry-run         Print operations without writing files.
  --help, -h        Show this help.

Default install:
  kkt-skills install

Installs to:
  ~/.agents/skills   (Codex, Pi, OpenCode shared location)
  ~/.claude/skills   (Claude Code location)

Examples:
  kkt-skills install --target codex
  kkt-skills install --target claude --local .
  kkt-skills install --target pi
  kkt-skills install --target opencode --local .
  kkt-skills install --target codex --dir /tmp/kkt-skills --dry-run
`;

function parseArgs(argv) {
  const args = {
    command: argv[0] || "help",
    target: "default",
    local: false,
    localPath: process.cwd(),
    dir: undefined,
    force: false,
    dryRun: false,
    help: false,
  };

  for (let i = 1; i < argv.length; i += 1) {
    const arg = argv[i];
    if (arg === "--help" || arg === "-h") {
      args.help = true;
    } else if (arg === "--target") {
      args.target = requiredValue(argv, ++i, "--target");
    } else if (arg === "--local") {
      args.local = true;
      const next = argv[i + 1];
      if (next && !next.startsWith("-")) {
        args.localPath = next;
        i += 1;
      }
    } else if (arg === "--dir") {
      args.dir = requiredValue(argv, ++i, "--dir");
    } else if (arg === "--force") {
      args.force = true;
    } else if (arg === "--dry-run") {
      args.dryRun = true;
    } else {
      fail(`Unknown argument: ${arg}`);
    }
  }

  return args;
}

function requiredValue(argv, index, flag) {
  const value = argv[index];
  if (!value || value.startsWith("-")) {
    fail(`${flag} requires a value.`);
  }
  return value;
}

function fail(message) {
  console.error(`Error: ${message}`);
  console.error("");
  console.error(usageText.trimEnd());
  process.exit(1);
}

function expandHome(value) {
  if (!value) return value;
  if (value === "~") return homedir();
  if (value.startsWith("~/")) return path.join(homedir(), value.slice(2));
  return value;
}

function xdgConfigHome() {
  return process.env.XDG_CONFIG_HOME
    ? expandHome(process.env.XDG_CONFIG_HOME)
    : path.join(homedir(), ".config");
}

function resolveRoot(candidate) {
  return path.resolve(expandHome(candidate));
}

function nativeRoot(target, local, localPath) {
  const projectRoot = resolveRoot(localPath);
  if (target === "codex") {
    return local ? path.join(projectRoot, ".agents", "skills") : path.join(homedir(), ".agents", "skills");
  }
  if (target === "claude") {
    return local ? path.join(projectRoot, ".claude", "skills") : path.join(homedir(), ".claude", "skills");
  }
  if (target === "pi") {
    return local ? path.join(projectRoot, ".pi", "skills") : path.join(homedir(), ".pi", "agent", "skills");
  }
  if (target === "opencode") {
    return local ? path.join(projectRoot, ".opencode", "skills") : path.join(xdgConfigHome(), "opencode", "skills");
  }
  throw new Error(`Unsupported native target: ${target}`);
}

function targetRoots(args) {
  if (args.dir) {
    return [{ label: args.target === "default" ? "explicit" : args.target, root: resolveRoot(args.dir) }];
  }

  if (args.target === "default") {
    if (args.local) {
      const projectRoot = resolveRoot(args.localPath);
      return [
        { label: "shared-local", root: path.join(projectRoot, ".agents", "skills") },
        { label: "claude-local", root: path.join(projectRoot, ".claude", "skills") },
      ];
    }
    return [
      { label: "shared-global", root: path.join(homedir(), ".agents", "skills") },
      { label: "claude-global", root: path.join(homedir(), ".claude", "skills") },
    ];
  }

  if (args.target === "all") {
    const targets = ["codex", "claude", "pi", "opencode"];
    return dedupeRoots(
      targets.map((target) => ({ label: target, root: nativeRoot(target, args.local, args.localPath) })),
    );
  }

  if (["codex", "claude", "pi", "opencode"].includes(args.target)) {
    return [{ label: args.target, root: nativeRoot(args.target, args.local, args.localPath) }];
  }

  fail(`Unsupported target: ${args.target}`);
}

function dedupeRoots(entries) {
  const seen = new Set();
  const result = [];
  for (const entry of entries) {
    const resolved = path.resolve(entry.root);
    if (seen.has(resolved)) continue;
    seen.add(resolved);
    result.push({ ...entry, root: resolved });
  }
  return result;
}

function hashFile(filePath) {
  return createHash("sha256").update(readFileSync(filePath)).digest("hex");
}

function listFiles(root) {
  const files = [];
  function walk(current, prefix) {
    for (const entry of readdirSync(current, { withFileTypes: true })) {
      const absolute = path.join(current, entry.name);
      const relative = path.join(prefix, entry.name);
      if (entry.isDirectory()) {
        walk(absolute, relative);
      } else if (entry.isFile()) {
        files.push(relative.split(path.sep).join("/"));
      }
    }
  }
  walk(root, "");
  return files.sort();
}

function directoriesEqual(left, right) {
  if (!existsSync(left) || !existsSync(right)) return false;
  if (!statSync(left).isDirectory() || !statSync(right).isDirectory()) return false;
  const leftFiles = listFiles(left);
  const rightFiles = listFiles(right);
  if (leftFiles.length !== rightFiles.length) return false;
  for (let i = 0; i < leftFiles.length; i += 1) {
    if (leftFiles[i] !== rightFiles[i]) return false;
    if (hashFile(path.join(left, leftFiles[i])) !== hashFile(path.join(right, rightFiles[i]))) {
      return false;
    }
  }
  return true;
}

function ensureSourceSkills() {
  if (!existsSync(sourceSkillsRoot)) {
    fail(`Source skills directory not found: ${sourceSkillsRoot}`);
  }
  for (const name of skillNames) {
    const skillDir = path.join(sourceSkillsRoot, name);
    const skillFile = path.join(skillDir, "SKILL.md");
    if (!existsSync(skillFile)) {
      fail(`Missing source skill: ${skillFile}`);
    }
  }
}

function installToRoot(root, options) {
  const operations = [];
  for (const name of skillNames) {
    const source = path.join(sourceSkillsRoot, name);
    const target = path.join(root, name);
    if (existsSync(target)) {
      if (directoriesEqual(source, target)) {
        operations.push({ action: "skip", source, target, reason: "already current" });
        continue;
      }
      if (!options.force) {
        operations.push({ action: "conflict", source, target, reason: "target differs; rerun with --force" });
        continue;
      }
      operations.push({ action: "overwrite", source, target });
    } else {
      operations.push({ action: "copy", source, target });
    }
  }

  const conflicts = operations.filter((operation) => operation.action === "conflict");
  if (conflicts.length > 0) {
    for (const conflict of conflicts) {
      console.error(`Conflict: ${conflict.target} (${conflict.reason})`);
    }
    console.error("No files were changed for this target.");
    process.exitCode = 1;
    return operations;
  }

  if (!options.dryRun) {
    mkdirSync(root, { recursive: true });
    for (const operation of operations) {
      if (operation.action === "skip") continue;
      if (operation.action === "overwrite") {
        rmSync(operation.target, { recursive: true, force: true });
      }
      cpSync(operation.source, operation.target, { recursive: true });
    }
  }

  return operations;
}

function printOperations(command, rootEntry, operations, dryRun) {
  const prefix = dryRun ? "[dry-run]" : command === "uninstall" ? "[removed]" : "[installed]";
  console.log(`${prefix} ${rootEntry.label}: ${rootEntry.root}`);
  for (const operation of operations) {
    const name = path.basename(operation.target);
    if (operation.action === "skip") {
      console.log(`  - ${name}: already current`);
    } else if (operation.action === "conflict") {
      console.log(`  - ${name}: conflict`);
    } else {
      console.log(`  - ${name}: ${operation.action}`);
    }
  }
}

function uninstallFromRoot(root, options) {
  const operations = skillNames.map((name) => ({
    action: existsSync(path.join(root, name)) ? "remove" : "skip",
    target: path.join(root, name),
  }));

  if (!options.dryRun) {
    for (const operation of operations) {
      if (operation.action === "remove") {
        rmSync(operation.target, { recursive: true, force: true });
      }
    }
  }

  return operations;
}

function printUninstallOperations(rootEntry, operations, dryRun) {
  const prefix = dryRun ? "[dry-run]" : "[removed]";
  console.log(`${prefix} ${rootEntry.label}: ${rootEntry.root}`);
  for (const operation of operations) {
    const name = path.basename(operation.target);
    console.log(`  - ${name}: ${operation.action === "remove" ? "remove" : "not installed"}`);
  }
}

function invocationHints(entries) {
  const labels = new Set(entries.map((entry) => entry.label));
  const roots = new Set(entries.map((entry) => path.resolve(entry.root)));
  const hints = [];

  if (labels.has("shared-global") || labels.has("shared-local") || labels.has("codex") || [...roots].some((root) => root.endsWith(`${path.sep}.agents${path.sep}skills`))) {
    hints.push("Codex: use $kkt, $kkt-loop, or $kkt-model");
    hints.push("Pi: use /skill:kkt, /skill:kkt-loop, or /skill:kkt-model");
    hints.push("OpenCode: use the skill tool with kkt, kkt-loop, or kkt-model");
  }
  if (labels.has("claude-global") || labels.has("claude-local") || labels.has("claude") || [...roots].some((root) => root.endsWith(`${path.sep}.claude${path.sep}skills`))) {
    hints.push("Claude Code: use /kkt, /kkt-loop, or /kkt-model");
  }
  if (labels.has("pi") || [...roots].some((root) => root.includes(`${path.sep}.pi${path.sep}`))) {
    hints.push("Pi: use /skill:kkt, /skill:kkt-loop, or /skill:kkt-model");
  }
  if (labels.has("opencode") || [...roots].some((root) => root.includes(`${path.sep}opencode${path.sep}skills`) || root.endsWith(`${path.sep}.opencode${path.sep}skills`))) {
    hints.push("OpenCode: use the skill tool with kkt, kkt-loop, or kkt-model");
  }

  return [...new Set(hints)];
}

function doctor() {
  ensureSourceSkills();
  console.log("KKT skills package looks valid.");
  console.log(`Platform: ${platform()}`);
  console.log(`Package root: ${packageRoot}`);
  console.log(`Source skills: ${sourceSkillsRoot}`);
  for (const name of skillNames) {
    console.log(`- ${name}: ${path.join(sourceSkillsRoot, name, "SKILL.md")}`);
  }
}

function main() {
  const args = parseArgs(process.argv.slice(2));
  if (args.help || args.command === "help") {
    console.log(usageText.trimEnd());
    return;
  }

  if (args.command === "doctor") {
    doctor();
    return;
  }

  if (!["install", "uninstall"].includes(args.command)) {
    fail(`Unsupported command: ${args.command}`);
  }

  ensureSourceSkills();
  const roots = targetRoots(args);

  if (args.command === "install") {
    for (const rootEntry of roots) {
      const operations = installToRoot(rootEntry.root, { dryRun: args.dryRun, force: args.force });
      printOperations("install", rootEntry, operations, args.dryRun);
    }
    if (process.exitCode) return;
    const hints = invocationHints(roots);
    if (hints.length > 0) {
      console.log("");
      console.log("Usage hints:");
      for (const hint of hints) console.log(`- ${hint}`);
    }
    return;
  }

  for (const rootEntry of roots) {
    const operations = uninstallFromRoot(rootEntry.root, { dryRun: args.dryRun });
    printUninstallOperations(rootEntry, operations, args.dryRun);
  }
}

main();
