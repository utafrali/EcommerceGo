#!/usr/bin/env node
/**
 * master-agent CLI — pretty printer & runner
 *
 * Usage:
 *   npm run decide                      # decide next iteration (interactive)
 *   npm run decide -- --extra "X"       # pass extra context
 *   npm run decide:execute              # decide + auto-run via claude
 *   npm run decide:watch                # loop every 15 min
 */

import { spawn } from "child_process";
import * as readline from "readline";
import * as fs from "fs";
import * as path from "path";
import { fileURLToPath } from "url";
import { log, section, step, printPlan, color, c } from "./logger.js";
import { decideIteration } from "./decide.js";
import type { IterationPlan } from "./types.js";

// ─── Config ───────────────────────────────────────────────────────────────────

const REPO_ROOT =
  process.env.REPO_ROOT ??
  path.resolve(fileURLToPath(import.meta.url), "../../../");

// ─── Args ─────────────────────────────────────────────────────────────────────

const args = process.argv.slice(2);
const extraIdx = args.indexOf("--extra");
const extraContext = extraIdx !== -1 ? args[extraIdx + 1] : undefined;
const shouldExecute = args.includes("--execute") || args.includes("-e");
const watchIdx = args.indexOf("--watch");
const watchMinutes = watchIdx !== -1 ? Number(args[watchIdx + 1]) || 15 : null;

// ─── Execution ────────────────────────────────────────────────────────────────

async function runExecution(plan: IterationPlan): Promise<void> {
  section("EXECUTION");
  step("exec", color(c.cyan, `Spawning claude to implement round ${plan.round_number}...`));
  log(`  ${color(c.gray, "(you can watch progress in real-time below)")}`);
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

  const env: Record<string, string> = {};
  for (const [k, v] of Object.entries(process.env)) {
    if (k !== "CLAUDECODE" && v !== undefined) env[k] = v;
  }

  const child = spawn(
    "claude",
    ["-p", prompt, "--model", "claude-opus-4-6", "--allowedTools", "Bash,Read,Edit,Write,Glob,Grep"],
    { cwd: REPO_ROOT, env, stdio: ["ignore", "pipe", "pipe"] }
  );

  child.stdout!.on("data", (chunk: Buffer) => process.stdout.write(chunk));
  child.stderr!.on("data", (chunk: Buffer) =>
    process.stderr.write(color(c.gray, chunk.toString()))
  );

  await new Promise<void>((resolve, reject) => {
    child.on("close", (code) =>
      code === 0 || code === null ? resolve() : reject(new Error(`claude exited ${code}`))
    );
    child.on("error", reject);
  });

  section("DONE");
  step("done", color(c.green, `Round ${plan.round_number} executed.`));
  log("");

  // Auto-commit all changes
  section("COMMITTING");
  step("arrow", color(c.yellow, "Running git add -A && git commit..."));
  const commitMsg = `feat: pipeline round ${plan.round_number} — ${plan.title}`;
  const commitChild = spawn(
    "bash",
    ["-c", `git add -A && git commit -m "${commitMsg.replace(/"/g, '\\"')}"`],
    { cwd: REPO_ROOT, env, stdio: ["ignore", "pipe", "pipe"] }
  );
  commitChild.stdout!.on("data", (chunk: Buffer) => process.stdout.write(chunk));
  commitChild.stderr!.on("data", (chunk: Buffer) => process.stderr.write(chunk));
  const committed = await new Promise<boolean>((resolve) => {
    commitChild.on("close", (code) => {
      if (code === 0) {
        step("ok", color(c.green, `Committed: "${commitMsg}"`));
        resolve(true);
      } else {
        step("arrow", color(c.yellow, "Nothing to commit or git error — skipping push."));
        resolve(false);
      }
    });
    commitChild.on("error", () => resolve(false));
  });

  if (committed) {
    step("arrow", color(c.yellow, "Pushing to origin..."));
    const pushChild = spawn("git", ["push"], { cwd: REPO_ROOT, env, stdio: ["ignore", "pipe", "pipe"] });
    pushChild.stdout!.on("data", (chunk: Buffer) => process.stdout.write(chunk));
    pushChild.stderr!.on("data", (chunk: Buffer) => process.stderr.write(chunk));
    await new Promise<void>((resolve) => {
      pushChild.on("close", (code) => {
        if (code === 0) {
          step("ok", color(c.green, "Pushed to origin."));
        } else {
          step("arrow", color(c.red, "git push failed — check remote/auth."));
        }
        resolve();
      });
      pushChild.on("error", () => resolve());
    });
  }
  log("");
}

// ─── Main ─────────────────────────────────────────────────────────────────────

async function runOnce() {
  log("");
  log(color(c.bold + c.cyan, "  ╔══════════════════════════════╗"));
  log(color(c.bold + c.cyan, "  ║   EcommerceGo Master Agent   ║"));
  log(color(c.bold + c.cyan, "  ╚══════════════════════════════╝"));
  log(`  ${color(c.gray, `repo: ${REPO_ROOT}`)}`);

  section("ASKING CLAUDE");
  if (extraContext) {
    step("arrow", `Extra context: ${color(c.yellow, extraContext)}`);
  }
  step("thinking", color(c.yellow, "Calling claude-opus-4-6 (30–120s)..."));
  log("");

  const start = Date.now();
  const response = await decideIteration(extraContext);
  const elapsed = ((Date.now() - start) / 1000).toFixed(1);

  step("ok", `Response received in ${color(c.green, elapsed + "s")}`);
  step("ok", `Round ${color(c.cyan, String(response.plan.round_number))} — ${color(c.bold, response.plan.title)}`);
  step("ok", `Tasks: ${color(c.cyan, String(response.plan.tasks.length))}  Priority: ${response.plan.priority}  Scope: ${response.plan.estimated_scope}`);

  section("ITERATION PLAN");
  printPlan(response.plan);

  if (response.plan.skip_reason) {
    log(`  ${color(c.yellow, "No work needed. Exiting.")}`);
    return;
  }

  if (shouldExecute) {
    log("");
    log(color(c.bold + c.green, "  --execute flag set. Starting execution..."));
    await runExecution(response.plan);
    return;
  }

  log("");
  const answer = await ask(
    color(c.bold, `  Execute this plan via claude? ${color(c.gray, "[y/N/s(save)]")} `)
  );

  if (answer === "y" || answer === "yes") {
    await runExecution(response.plan);
  } else if (answer === "s" || answer === "save") {
    const outFile = `iteration-plan-round-${response.plan.round_number}.json`;
    fs.writeFileSync(outFile, JSON.stringify(response.plan, null, 2));
    step("ok", `Plan saved to ${color(c.cyan, outFile)}`);
  } else {
    step("arrow", color(c.gray, "Skipped. Run with --execute to auto-run."));
  }
}

function ask(question: string): Promise<string> {
  const rl = readline.createInterface({ input: process.stdin, output: process.stdout });
  return new Promise((resolve) => {
    rl.question(question, (a) => { rl.close(); resolve(a.trim().toLowerCase()); });
  });
}

async function watchLoop(intervalMinutes: number) {
  log(color(c.bold + c.magenta, `  ⏱  Watch mode — running every ${intervalMinutes} minutes`));
  log(color(c.gray, "  Press Ctrl+C to stop.\n"));
  while (true) {
    try { await runOnce(); } catch (err) {
      log(color(c.red, `  Error: ${err instanceof Error ? err.message : String(err)}`));
    }
    log("");
    log(color(c.gray, `  Next run in ${intervalMinutes}m. Press Ctrl+C to stop.`));
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
  log(color(c.red + c.bold, "  ERROR: ") + (err instanceof Error ? err.message : String(err)));
  process.exit(1);
}
