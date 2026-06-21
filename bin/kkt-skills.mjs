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
const skillNames = [
  "kkt",
  "kkt-loop",
  "kkt-model",
];

const legacySkillNames = [
  "kkt-intent",
  "kkt-discovery",
  "kkt-modeling",
  "kkt-execution",
  "kkt-validation",
];

const usageText = `KKT skills installer

Usage:
  kkt-skills install [options]
  kkt-skills upgrade [options]
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

Clean upgrade:
  kkt-skills upgrade

Installs to:
  ~/.agents/skills   (Codex, Pi, OpenCode shared location)
  ~/.claude/skills   (Claude Code location)

Examples:
  kkt-skills install --target codex
  kkt-skills install --target claude --local .
  kkt-skills install --target pi
  kkt-skills install --target opencode --local .
  kkt-skills install --target codex --dir /tmp/kkt-skills --dry-run
  kkt-skills upgrade --target codex
`;

function parseArgs(argv) {
  const firstArg = argv[0];
  const args = {
    command: !firstArg || firstArg === "--help" || firstArg === "-h" ? "help" : firstArg,
    target: "default",
    local: false,
    localPath: process.cwd(),
    dir: undefined,
    force: false,
    dryRun: false,
    help: firstArg === "--help" || firstArg === "-h",
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

function resolveRoot(candidate) {
  return path.resolve(expandHome(candidate));
}

function nativeRoot(target, local, localPath) {
  const projectRoot = resolveRoot(localPath);
  if (target === "codex" || target === "pi" || target === "opencode") {
    return local ? path.join(projectRoot, ".agents", "skills") : path.join(homedir(), ".agents", "skills");
  }
  if (target === "claude") {
    return local ? path.join(projectRoot, ".claude", "skills") : path.join(homedir(), ".claude", "skills");
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
        operations.push({ action: "conflict", source, target, reason: "target differs; run upgrade or rerun install with --force" });
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

function knownInstalledSkillNames() {
  return [...new Set([...skillNames, ...legacySkillNames])];
}

function upgradeToRoot(root, options) {
  const operations = [];
  for (const name of knownInstalledSkillNames()) {
    const target = path.join(root, name);
    operations.push({
      action: existsSync(target) ? "remove" : "skip",
      target,
    });
  }

  for (const name of skillNames) {
    operations.push({
      action: "copy",
      source: path.join(sourceSkillsRoot, name),
      target: path.join(root, name),
    });
  }

  if (!options.dryRun) {
    mkdirSync(root, { recursive: true });
    for (const operation of operations) {
      if (operation.action === "remove") {
        rmSync(operation.target, { recursive: true, force: true });
      }
    }
    for (const operation of operations) {
      if (operation.action === "copy") {
        cpSync(operation.source, operation.target, { recursive: true });
      }
    }
  }

  return operations;
}

function printOperations(command, rootEntry, operations, dryRun) {
  const prefix = dryRun
    ? "[dry-run]"
    : command === "uninstall"
      ? "[removed]"
      : command === "upgrade"
        ? "[upgraded]"
        : "[installed]";
  console.log(`${prefix} ${rootEntry.label}: ${rootEntry.root}`);

  const namesByAction = (action) => operations
    .filter((operation) => operation.action === action)
    .map((operation) => path.basename(operation.target));
  const printSummary = (label, names) => {
    if (names.length === 0) return;
    console.log(`  - ${label}: ${names.join(", ")}`);
  };

  if (command === "upgrade") {
    const removedLegacy = namesByAction("remove")
      .filter((name) => legacySkillNames.includes(name));
    printSummary("installed", skillNames);
    printSummary("removed legacy", removedLegacy);
    return;
  }

  const installed = namesByAction("copy");
  const updated = namesByAction("overwrite");
  const current = namesByAction("skip");
  const conflicts = namesByAction("conflict");

  printSummary("conflicts", conflicts);
  if (conflicts.length > 0) return;

  printSummary("installed", installed);
  printSummary("updated", updated);
  if (installed.length === 0 && updated.length === 0 && conflicts.length === 0) {
    printSummary("already current", current);
  }
}

function uninstallFromRoot(root, options) {
  const operations = knownInstalledSkillNames().map((name) => ({
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

  if (
    labels.has("shared-global")
    || labels.has("shared-local")
    || labels.has("codex")
    || labels.has("pi")
    || labels.has("opencode")
    || [...roots].some((root) => root.endsWith(`${path.sep}.agents${path.sep}skills`))
  ) {
    hints.push("Codex: use $kkt, $kkt-loop, or $kkt-model");
    hints.push("Pi: use /skill:kkt, /skill:kkt-loop, or /skill:kkt-model");
    hints.push("OpenCode: use the skill tool with kkt, kkt-loop, or kkt-model");
  }
  if (labels.has("claude-global") || labels.has("claude-local") || labels.has("claude") || [...roots].some((root) => root.endsWith(`${path.sep}.claude${path.sep}skills`))) {
    hints.push("Claude Code: use /kkt, /kkt-loop, or /kkt-model");
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

  if (!["install", "upgrade", "uninstall"].includes(args.command)) {
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

  if (args.command === "upgrade") {
    for (const rootEntry of roots) {
      const operations = upgradeToRoot(rootEntry.root, { dryRun: args.dryRun });
      printOperations("upgrade", rootEntry, operations, args.dryRun);
    }
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
