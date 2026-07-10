import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

type KktHookResult = {
  verdict?: "allow" | "warn" | "block";
  reason?: string;
  evidence?: string[];
  repair?: string[];
};

async function runKktHook(pi: ExtensionAPI, event: "pre-tool" | "post-tool", payload: unknown, cwd: string, signal?: AbortSignal): Promise<KktHookResult> {
  const result = await pi.exec("kkt", ["hook", event, "--agent", "pi", "--json", JSON.stringify(payload)], {
    cwd,
    signal,
    timeout: 5000,
  });
  if (result.code !== 0) {
    return { verdict: "allow", reason: "kkt hook unavailable" };
  }
  try {
    return JSON.parse(result.stdout || "{}") as KktHookResult;
  } catch {
    return { verdict: "allow", reason: "kkt hook returned invalid JSON" };
  }
}

function blockReason(result: KktHookResult): string {
  const lines = [result.reason, ...(result.evidence ?? []), ...(result.repair ?? [])].filter(Boolean);
  return lines.join("\n") || "Blocked by KKT hook guardrails";
}

export default function (pi: ExtensionAPI) {
  pi.on("tool_call", async (event, ctx) => {
    const result = await runKktHook(
      pi,
      "pre-tool",
      { toolName: event.toolName, input: event.input, cwd: ctx.cwd },
      ctx.cwd,
      ctx.signal,
    );
    if (result.verdict === "block") {
      return { block: true, reason: blockReason(result) };
    }
    return undefined;
  });

  pi.on("tool_result", async (event, ctx) => {
    const result = await runKktHook(
      pi,
      "post-tool",
      { toolName: event.toolName, input: event.input, cwd: ctx.cwd },
      ctx.cwd,
      ctx.signal,
    );
    if (result.verdict === "block") {
      return {
        isError: true,
        content: [{ type: "text", text: blockReason(result) }],
      };
    }
    return undefined;
  });
}
