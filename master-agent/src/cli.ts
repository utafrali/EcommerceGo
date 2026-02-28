#!/usr/bin/env node
/**
 * master-agent CLI
 * Usage:
 *   npx tsx src/cli.ts                    # decide next iteration
 *   npx tsx src/cli.ts --extra "focus on X"
 *   npx tsx src/cli.ts --execute          # decide + immediately run via claude
 *   npx tsx src/cli.ts --watch 15         # loop every 15 minutes
 */

import { execSync, spawn } from "child_process";
import * as readline from "readline";
import { gatherContext, formatContextForPrompt } from "./context.js";
import { decideIteration } from "./decide.js";
import { log, section, step, indent, printPlan, color, c } from "./logger.js";
import type { IterationPlan } from "./types.js";

// ─── Args ─────────────────────────────────────────────────────────────────────

const args = process.argv.slice(2);
const extraIdx = args.indexOf("--extra");
const extraContext = extraIdx !== -1 ? args[extraIdx + 1] : undefined;
const shouldExecute = args.includes("--execute") || args.includes("-e");
const watchIdx = args.indexOf("--watch");
const watchMinutes = watchIdx !== -1 ? Number(args[watchIdx + 1]) || 15 : null;

// ─── Helpers ──────────────────────────────────────────────────────────────────

function ask(question: string): Promise<string> {
  const rl = readline.createInterface({
    input: process.stdin,
    output: process.stdout,
  });
  return new Promise((resolve) => {
    rl.question(question, (answer) => {
      rl.close();
      resolve(answer.trim().toLowerCase());
    });
  });
}

async function runExecution(plan: IterationPlan, repoRoot: string) {
  section("EXECUTION");
  step("exec", color(c.cyan, "Spawning claude to implement the plan..."));
  log("");

  const prompt = [
    `Implement pipeline round ${plan.round_number}: "${plan.title}"`,
    "",
    `Rationale: ${plan.rationale}`,
    "",
    "Tasks to implement:",
    ...plan.tasks.map(
      (t) =>
        `- ${t.id}: ${t.description}\n  Files: ${t.files_affected.join(", ")}\n  Why: ${t.rationale}`
    ),
    "",
    "Work through each task systematically. Read files before editing. Run tests after changes where possible.",
  ].join("\n");

  // Unset CLAUDECODE so nested invocation works
  const env = { ...process.env };
  delete env["CLAUDECODE"];

  const child = spawn(
    "claude",
    [
      "-p",
      prompt,
      "--model",
      "claude-opus-4-6",
      "--allowedTools",
      "Bash,Read,Edit,Write,Glob,Grep",
    ],
    {
      cwd: repoRoot,
      env,
      stdio: ["ignore", "pipe", "pipe"],
    }
  );

  let buffer = "";

  child.stdout.on("data", (chunk: Buffer) => {
    const text = chunk.toString();
    buffer += text;
    process.stdout.write(text);
  });

  child.stderr.on("data", (chunk: Buffer) => {
    process.stderr.write(color(c.gray, chunk.toString()));
  });

  await new Promise<void>((resolve, reject) => {
    child.on("close", (code) => {
      if (code === 0 || code === null) {
        resolve();
      } else {
        reject(new Error(`claude exited with code ${code}`));
      }
    });
    child.on("error", reject);
  });

  section("DONE");
  step("done", `Round ${plan.round_number} executed.`);

  // Auto-commit suggestion
  log("");
  log(
    `  ${color(c.gray, "To commit:")}  git add -A && git commit -m "feat: pipeline round ${plan.round_number} — ${plan.title}"`
  );
  log("");

  return buffer;
}

// ─── Main loop ────────────────────────────────────────────────────────────────

async function runOnce() {
  const repoRoot =
    process.env.REPO_ROOT ??
    new URL("../../", import.meta.url).pathname.replace(/\/$/, "");

  log("");
  log(
    color(c.bold + c.cyan, "  ╔══════════════════════════════╗")
  );
  log(color(c.bold + c.cyan, "  ║   EcommerceGo Master Agent   ║"));
  log(color(c.bold + c.cyan, "  ╚══════════════════════════════╝"));
  log(`  ${color(c.gray, `repo: ${repoRoot}`)}`);

  // 1. Gather context
  section("GATHERING CONTEXT");
  const ctx = gatherContext(extraContext);

  step("ok", `Services:  ${color(c.cyan, ctx.services.join(", "))}`);
  step("ok", `Packages:  ${color(c.cyan, ctx.packages.join(", "))}`);

  const commitLines = ctx.recentCommits.split("\n").filter(Boolean);
  step("ok", `Commits:   ${color(c.cyan, String(commitLines.length))} loaded`);
  indent(commitLines.slice(0, 5));

  const roundLines = ctx.pipelineRounds.split("###").filter((l) => l.trim());
  step(
    "ok",
    `Pipeline rounds: ${color(c.cyan, String(roundLines.length))} found`
  );

  if (extraContext) {
    step("arrow", `Extra context: ${color(c.yellow, extraContext)}`);
  }

  const contextText = formatContextForPrompt(ctx);
  const tokenEst = Math.round(contextText.length / 4);
  step("ok", `Context size: ~${color(c.cyan, String(tokenEst))} tokens`);

  // 2. Ask Claude
  section("ASKING CLAUDE");
  step(
    "thinking",
    color(c.yellow, "Calling claude-opus-4-6 via CLI session (may take ~30s)...")
  );
  log("");

  const start = Date.now();
  const response = await decideIteration(extraContext);
  const elapsed = ((Date.now() - start) / 1000).toFixed(1);

  step("ok", `Response received in ${color(c.green, elapsed + "s")}`);

  // 3. Print plan
  section("ITERATION PLAN");
  printPlan(response.plan);

  if (response.plan.skip_reason) {
    log(`  ${color(c.yellow, "No work needed. Exiting.")}`);
    return;
  }

  // 4. Execute?
  if (shouldExecute) {
    log("");
    log(color(c.bold + c.green, "  --execute flag set. Starting execution..."));
    await runExecution(response.plan, repoRoot);
    return;
  }

  // Interactive prompt
  log("");
  const answer = await ask(
    color(
      c.bold,
      `  Execute this plan via claude? ${color(c.gray, "[y/N/s(save)]")} `
    )
  );

  if (answer === "y" || answer === "yes") {
    await runExecution(response.plan, repoRoot);
  } else if (answer === "s" || answer === "save") {
    const outFile = `iteration-plan-round-${response.plan.round_number}.json`;
    const { writeFileSync } = await import("fs");
    writeFileSync(outFile, JSON.stringify(response.plan, null, 2));
    step("ok", `Plan saved to ${color(c.cyan, outFile)}`);
  } else {
    step("arrow", color(c.gray, "Skipped. Run with --execute to auto-run."));
  }
}

async function watchLoop(intervalMinutes: number) {
  log(
    color(
      c.bold + c.magenta,
      `  ⏱  Watch mode — running every ${intervalMinutes} minutes`
    )
  );
  log(color(c.gray, "  Press Ctrl+C to stop.\n"));

  while (true) {
    try {
      await runOnce();
    } catch (err) {
      log(color(c.red, `  Error: ${err instanceof Error ? err.message : String(err)}`));
    }

    log("");
    log(
      color(
        c.gray,
        `  Next run in ${intervalMinutes}m. Press Ctrl+C to stop.`
      )
    );
    await new Promise((r) => setTimeout(r, intervalMinutes * 60 * 1000));
  }
}

// ─── Entry ────────────────────────────────────────────────────────────────────

try {
  if (watchMinutes !== null) {
    await watchLoop(watchMinutes);
  } else {
    await runOnce();
  }
} catch (err) {
  log("");
  log(
    color(c.red + c.bold, "  ERROR: ") +
      (err instanceof Error ? err.message : String(err))
  );
  process.exit(1);
}
