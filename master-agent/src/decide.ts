import { spawn } from "child_process";
import { gatherContext, formatContextForPrompt } from "./context.js";
import type { IterationPlan, DecideResponse } from "./types.js";

const SYSTEM_PROMPT = `You are the Master Agent for the EcommerceGo project — an AI-driven, open-source, microservices e-commerce platform.

Your role is to analyze the current state of the codebase and determine the best next pipeline iteration to improve code quality, robustness, security, performance, or developer experience.

## Project Stack
- 12 Go 1.23+ microservices (chi router, PostgreSQL, Redis, Kafka KRaft)
- TypeScript BFF (Fastify)
- Next.js 15 frontend
- Shared pkg/ libraries (logger, database, kafka, middleware, errors, health, config, validator, pagination)

## Pipeline Iteration Guidelines
- Each iteration focuses on ONE clear theme (e.g., "observability", "error handling", "test coverage", "API consistency")
- Tasks should be concrete, with specific files or packages mentioned
- Avoid repeating work from previous rounds
- Prefer improvements that benefit multiple services at once
- Consider: security, performance, observability, test coverage, code quality, DX, documentation

## Output Format
Respond with a single JSON object (no markdown fences, no explanation outside JSON). Schema:
{
  "round_number": <integer, next after the last completed round>,
  "title": <string, short descriptive title for this iteration>,
  "rationale": <string, 2-3 sentences explaining WHY this is the most valuable next step>,
  "priority": <"low" | "medium" | "high" | "critical">,
  "tasks": [
    {
      "id": <string, e.g. "T1">,
      "description": <string, concise action>,
      "files_affected": [<string, relative file paths or package names>],
      "rationale": <string, one sentence>
    }
  ],
  "estimated_scope": <"small" | "medium" | "large">,
  "skip_reason": <string or null, if no meaningful work remains>
}`;

interface ClaudeCliResult {
  type: string;
  subtype: string;
  result: string;
  is_error: boolean;
}

function extractJson(text: string): string {
  const fenced = text.match(/```(?:json)?\s*([\s\S]*?)```/);
  if (fenced) return fenced[1].trim();
  const bare = text.match(/\{[\s\S]*\}/);
  if (bare) return bare[0];
  return text.trim();
}

/**
 * Calls the claude CLI via async spawn.
 *
 * Key fixes vs the previous version:
 *  1. `--allowedTools ""` (empty string) caused the CLI to hang indefinitely —
 *     omit the flag entirely when no tools are needed.
 *  2. The full prompt (system + user) is written to stdin; `-p` with no
 *     argument puts the CLI in non-interactive stdin-read mode.
 */
function callClaude(prompt: string): Promise<string> {
  return new Promise((resolve, reject) => {
    // Strip CLAUDECODE so the nested-session guard doesn't fire.
    const env: Record<string, string> = {};
    for (const [k, v] of Object.entries(process.env)) {
      if (k !== "CLAUDECODE" && v !== undefined) env[k] = v;
    }

    const child = spawn(
      "claude",
      ["-p", "--output-format", "json", "--model", "claude-opus-4-6"],
      { env, stdio: ["pipe", "pipe", "pipe"] }
    );

    let stdout = "";
    let stderr = "";

    child.stdout.on("data", (chunk: Buffer) => { stdout += chunk.toString(); });
    child.stderr.on("data", (chunk: Buffer) => { stderr += chunk.toString(); });

    const timeout = setTimeout(() => {
      child.kill("SIGTERM");
      reject(new Error("claude CLI timed out after 300s"));
    }, 300_000);

    child.on("close", (code) => {
      clearTimeout(timeout);
      if (code === 0 || (code === null && stdout)) {
        resolve(stdout);
      } else {
        reject(
          new Error(
            `claude CLI exited with code ${code}. stderr: ${stderr.slice(0, 500)}`
          )
        );
      }
    });

    child.on("error", (err) => {
      clearTimeout(timeout);
      reject(err);
    });

    child.stdin.write(prompt, "utf-8", (err) => {
      if (err) {
        clearTimeout(timeout);
        child.kill();
        reject(err);
        return;
      }
      child.stdin.end();
    });
  });
}

export async function decideIteration(
  extraContext?: string
): Promise<DecideResponse> {
  const ctx = gatherContext(extraContext);
  const contextText = formatContextForPrompt(ctx);

  const fullPrompt = [
    SYSTEM_PROMPT,
    "",
    "---",
    "",
    "Analyze the repository state below and decide the next pipeline iteration.",
    "",
    contextText,
  ].join("\n");

  const raw = await callClaude(fullPrompt);

  let cliResult: ClaudeCliResult;
  try {
    cliResult = JSON.parse(raw) as ClaudeCliResult;
  } catch {
    throw new Error(`Failed to parse claude CLI output: ${raw.slice(0, 300)}`);
  }

  if (cliResult.is_error) {
    throw new Error(`Claude CLI returned an error: ${cliResult.result}`);
  }

  const jsonText = extractJson(cliResult.result);

  let plan: IterationPlan;
  try {
    plan = JSON.parse(jsonText) as IterationPlan;
  } catch {
    throw new Error(
      `Failed to parse IterationPlan from: ${jsonText.slice(0, 300)}`
    );
  }

  return { plan, model: "claude-opus-4-6", thinking_enabled: false };
}
