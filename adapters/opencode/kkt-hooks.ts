type KktHookResult = {
  verdict?: "allow" | "warn" | "block";
  reason?: string;
  evidence?: string[];
  repair?: string[];
};

function runKktHook(event: "pre-tool" | "post-tool", payload: unknown, cwd: string): KktHookResult {
  const proc = Bun.spawnSync({
    cmd: ["kkt", "hook", event, "--agent", "opencode", "--json", JSON.stringify(payload)],
    cwd,
    stdout: "pipe",
    stderr: "pipe",
  });
  if (proc.exitCode !== 0) {
    return { verdict: "allow", reason: "kkt hook unavailable" };
  }
  try {
    return JSON.parse(new TextDecoder().decode(proc.stdout || new Uint8Array())) as KktHookResult;
  } catch {
    return { verdict: "allow", reason: "kkt hook returned invalid JSON" };
  }
}

function blockReason(result: KktHookResult): string {
  const lines = [result.reason, ...(result.evidence ?? []), ...(result.repair ?? [])].filter(Boolean);
  return lines.join("\n") || "Blocked by KKT hook guardrails";
}

export const KktHooks = async ({ directory, worktree }: { directory: string; worktree?: string }) => {
  const cwd = worktree || directory;
  return {
    "tool.execute.before": async (input: any, output: any) => {
      const result = runKktHook("pre-tool", { tool: input?.tool, args: output?.args, cwd }, cwd);
      if (result.verdict === "block") {
        throw new Error(blockReason(result));
      }
    },
    "tool.execute.after": async (input: any, output: any) => {
      const result = runKktHook("post-tool", { tool: input?.tool, args: output?.args, cwd }, cwd);
      if (result.verdict === "block") {
        throw new Error(blockReason(result));
      }
    },
  };
};

export default KktHooks;
