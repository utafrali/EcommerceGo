import Anthropic from "@anthropic-ai/sdk";
import { gatherContext, formatContextForPrompt } from "./context.js";
import type { IterationPlan, DecideResponse } from "./types.js";

const client = new Anthropic({
  apiKey: process.env.ANTHROPIC_API_KEY,
});

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
Respond with a single JSON object (no markdown fences). Schema:
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

export async function decideIteration(
  extraContext?: string
): Promise<DecideResponse> {
  const ctx = gatherContext(extraContext);
  const contextText = formatContextForPrompt(ctx);

  const userMessage = `Analyze the repository state below and decide the next pipeline iteration.\n\n${contextText}`;

  const response = await client.messages.create({
    model: "claude-opus-4-6",
    max_tokens: 8000,
    thinking: { type: "adaptive" },
    system: SYSTEM_PROMPT,
    messages: [{ role: "user", content: userMessage }],
  });

  // Extract the text block (thinking blocks are separate)
  const textBlock = response.content.find((b) => b.type === "text");
  if (!textBlock || textBlock.type !== "text") {
    throw new Error("Claude returned no text content");
  }

  let plan: IterationPlan;
  try {
    plan = JSON.parse(textBlock.text) as IterationPlan;
  } catch {
    // Try to extract JSON from the text if it contains extra content
    const jsonMatch = textBlock.text.match(/\{[\s\S]*\}/);
    if (!jsonMatch) {
      throw new Error(
        `Failed to parse iteration plan from Claude response: ${textBlock.text.slice(0, 200)}`
      );
    }
    plan = JSON.parse(jsonMatch[0]) as IterationPlan;
  }

  const hasThinking = response.content.some((b) => b.type === "thinking");

  return {
    plan,
    model: response.model,
    thinking_enabled: hasThinking,
  };
}
