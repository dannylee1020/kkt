import type { ExtensionAPI } from "@earendil-works/pi-coding-agent";

export default function (pi: ExtensionAPI) {
  const port = process.env.KKT_TEST_PORT;
  if (!port) throw new Error("KKT_TEST_PORT is required");

  pi.registerProvider("kkt-test", {
    baseUrl: `http://127.0.0.1:${port}/v1`,
    apiKey: "kkt-test",
    api: "openai-completions",
    models: [
      {
        id: "test",
        name: "KKT Hook Test Model",
        reasoning: false,
        input: ["text"],
        cost: { input: 0, output: 0, cacheRead: 0, cacheWrite: 0 },
        contextWindow: 16_000,
        maxTokens: 1_000,
      },
    ],
  });
}
