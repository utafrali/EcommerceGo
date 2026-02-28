import { execSync } from "child_process";
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
  session_id?: string;
}

function extractJson(text: string): string {
  // Strip markdown fences if present
  const fenced = text.match(/```(?:json)?\s*([\s\S]*?)```/);
  if (fenced) return fenced[1].trim();
  // Find bare JSON object
  const bare = text.match(/\{[\s\S]*\}/);
  if (bare) return bare[0];
  return text.trim();
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
    `Analyze the repository state below and decide the next pipeline iteration.`,
    "",
    contextText,
  ].join("\n");

  // Call claude CLI in non-interactive print mode.
  // Stdin carries the full prompt; tools are disabled so it just responds.
  // CLAUDECODE env var must be unset to allow nested invocation.
  const env = { ...process.env };
  delete env["CLAUDECODE"];

  const raw = execSync(
    `claude -p --output-format json --model claude-opus-4-6 --allowedTools ""`,
    {
      input: fullPrompt,
      encoding: "utf-8",
      maxBuffer: 10 * 1024 * 1024,
      timeout: 180_000,
      env,
    }
  );

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

  return {
    plan,
    model: "claude-opus-4-6",
    thinking_enabled: false,
  };
}
