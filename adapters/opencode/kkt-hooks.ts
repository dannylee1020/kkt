type KktHookResult = {
  verdict?: "allow" | "warn" | "block";
  reason?: string;
  evidence?: string[];
  repair?: string[];
};

async function runKktHook(event: "pre-tool" | "post-tool", payload: unknown, cwd: string): Promise<KktHookResult> {
  try {
    const proc = Bun.spawn({
      cmd: ["kkt", "hook", event, "--agent", "opencode", "--json", JSON.stringify(payload)],
      cwd,
      stdout: "pipe",
      stderr: "pipe",
      timeout: 5000,
    });
    const stdoutPromise = proc.stdout?.text() ?? Promise.resolve("");
    const exitCode = await proc.exited;
    const stdout = await stdoutPromise;
    if (exitCode !== 0) {
      return { verdict: "allow", reason: "kkt hook unavailable" };
    }
    try {
      return JSON.parse(stdout || "{}") as KktHookResult;
    } catch {
      return { verdict: "allow", reason: "kkt hook returned invalid JSON" };
    }
  } catch {
    return { verdict: "allow", reason: "kkt hook unavailable" };
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
      const result = await runKktHook("pre-tool", { tool: input?.tool, args: output?.args, cwd }, cwd);
      if (result.verdict === "block") {
        throw new Error(blockReason(result));
      }
    },
    "tool.execute.after": async (input: any, _output: any) => {
      const result = await runKktHook("post-tool", { tool: input?.tool, args: input?.args, cwd }, cwd);
      if (result.verdict === "block") {
        throw new Error(blockReason(result));
      }
    },
  };
};

export default KktHooks;
