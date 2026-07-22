#!/usr/bin/env python3
"""Deterministic OpenAI-compatible streaming server for runtime hook tests."""

from __future__ import annotations

import json
import shlex
import sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path


class Handler(BaseHTTPRequestHandler):
    server_version = "kkt-hook-test/1"

    def log_message(self, _format: str, *_args: object) -> None:
        return

    def do_POST(self) -> None:  # noqa: N802
        length = int(self.headers.get("Content-Length", "0"))
        raw = self.rfile.read(length)
        request = json.loads(raw or b"{}")
        log_path = self.server.log_path  # type: ignore[attr-defined]
        with log_path.open("a", encoding="utf-8") as handle:
            handle.write(json.dumps(request, sort_keys=True) + "\n")

        is_title_request = "title generator" in json.dumps(request).lower()
        if is_title_request:
            response = final_response(request.get("model", "kkt-test"))
        else:
            actual_requests = 0
            for line in log_path.open(encoding="utf-8"):
                previous = json.loads(line)
                if "title generator" not in json.dumps(previous).lower():
                    actual_requests += 1
            if actual_requests == 1:
                response = tool_call_response(
                    self.server.tool_name,  # type: ignore[attr-defined]
                    self.server.target,  # type: ignore[attr-defined]
                    request.get("model", "kkt-test"),
                )
            else:
                response = final_response(request.get("model", "kkt-test"))

        payload = "".join(f"data: {json.dumps(chunk)}\n\n" for chunk in response)
        payload += "data: [DONE]\n\n"
        encoded = payload.encode("utf-8")
        self.send_response(200)
        self.send_header("Content-Type", "text/event-stream")
        self.send_header("Cache-Control", "no-cache")
        self.send_header("Content-Length", str(len(encoded)))
        self.end_headers()
        self.wfile.write(encoded)
        self.wfile.flush()


def tool_call_response(tool_name: str, target: str, model: str) -> list[dict]:
    if tool_name == "write":
        arguments = {"path": target, "content": "KKT runtime hook test\n"}
    elif tool_name == "bash":
        arguments = {
            "command": f"touch {shlex.quote(target)}",
            "description": "KKT runtime hook test",
        }
    else:
        raise SystemExit(f"unsupported test tool: {tool_name}")

    return [
        {
            "id": "kkt-hook-test-1",
            "object": "chat.completion.chunk",
            "model": model,
            "choices": [
                {
                    "index": 0,
                    "delta": {
                        "role": "assistant",
                        "tool_calls": [
                            {
                                "index": 0,
                                "id": "call_kkt_hook_test",
                                "type": "function",
                                "function": {
                                    "name": tool_name,
                                    "arguments": json.dumps(arguments),
                                },
                            }
                        ],
                    },
                    "finish_reason": "tool_calls",
                }
            ],
        }
    ]


def final_response(model: str) -> list[dict]:
    return [
        {
            "id": "kkt-hook-test-2",
            "object": "chat.completion.chunk",
            "model": model,
            "choices": [
                {
                    "index": 0,
                    "delta": {"role": "assistant", "content": "runtime hook test complete"},
                    "finish_reason": "stop",
                }
            ],
        }
    ]


def main() -> None:
    if len(sys.argv) != 6:
        raise SystemExit("usage: fake-openai-server.py PORT READY TARGET TOOL LOG")
    port = int(sys.argv[1])
    ready_path = Path(sys.argv[2])
    target = sys.argv[3]
    tool_name = sys.argv[4]
    log_path = Path(sys.argv[5])
    log_path.write_text("", encoding="utf-8")

    server = ThreadingHTTPServer(("127.0.0.1", port), Handler)
    server.target = target  # type: ignore[attr-defined]
    server.tool_name = tool_name  # type: ignore[attr-defined]
    server.log_path = log_path  # type: ignore[attr-defined]
    ready_path.write_text(str(server.server_port), encoding="utf-8")
    server.serve_forever()


if __name__ == "__main__":
    main()
