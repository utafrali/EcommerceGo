import { execSync } from "child_process";
import { readdirSync, readFileSync, existsSync } from "fs";
import { join } from "path";

const REPO_ROOT = process.env.REPO_ROOT ?? join(import.meta.dirname, "../..");

function run(cmd: string): string {
  try {
    return execSync(cmd, { cwd: REPO_ROOT, encoding: "utf-8" }).trim();
  } catch {
    return "";
  }
}

function getRecentCommits(): string {
  return run("git log --oneline -30");
}

function getLatestCommitBody(): string {
  return run("git log -1 --format=%B");
}

function getGitStatus(): string {
  return run("git status --short");
}

function getServices(): string[] {
  const servicesDir = join(REPO_ROOT, "services");
  if (!existsSync(servicesDir)) return [];
  return readdirSync(servicesDir, { withFileTypes: true })
    .filter((d) => d.isDirectory())
    .map((d) => d.name);
}

function getPackages(): string[] {
  const pkgDir = join(REPO_ROOT, "pkg");
  if (!existsSync(pkgDir)) return [];
  return readdirSync(pkgDir, { withFileTypes: true })
    .filter((d) => d.isDirectory())
    .map((d) => d.name);
}

function getMasterAgentPrompt(): string {
  const promptPath = join(REPO_ROOT, "agents/master/prompt.md");
  if (!existsSync(promptPath)) return "";
  try {
    return readFileSync(promptPath, "utf-8");
  } catch {
    return "";
  }
}

function getPipelineRounds(): string {
  const memoryDir = join(
    REPO_ROOT,
    ".claude/projects/-Users-ugurtafrali-Dev-EcommerceGo/memory"
  );
  if (!existsSync(memoryDir)) return "";

  const files = readdirSync(memoryDir)
    .filter((f) => f.startsWith("pipeline-round") && f.endsWith(".md"))
    .sort();

  return files
    .map((f) => {
      try {
        const content = readFileSync(join(memoryDir, f), "utf-8");
        // Return only the first 500 chars of each to keep context manageable
        return `### ${f}\n${content.slice(0, 500)}`;
      } catch {
        return `### ${f}\n(unreadable)`;
      }
    })
    .join("\n\n");
}

export interface ProjectContext {
  recentCommits: string;
  latestCommitBody: string;
  gitStatus: string;
  services: string[];
  packages: string[];
  masterAgentPrompt: string;
  pipelineRounds: string;
  extraContext: string;
}

export function gatherContext(extraContext?: string): ProjectContext {
  return {
    recentCommits: getRecentCommits(),
    latestCommitBody: getLatestCommitBody(),
    gitStatus: getGitStatus(),
    services: getServices(),
    packages: getPackages(),
    masterAgentPrompt: getMasterAgentPrompt(),
    pipelineRounds: getPipelineRounds(),
    extraContext: extraContext ?? "",
  };
}

export function formatContextForPrompt(ctx: ProjectContext): string {
  const lines: string[] = [];

  lines.push("## Repository State");
  lines.push("");
  lines.push("### Services");
  lines.push(ctx.services.join(", "));
  lines.push("");
  lines.push("### Shared Packages");
  lines.push(ctx.packages.join(", "));
  lines.push("");
  lines.push("### Recent Commits (last 30)");
  lines.push("```");
  lines.push(ctx.recentCommits);
  lines.push("```");
  lines.push("");
  lines.push("### Latest Commit Body");
  lines.push("```");
  lines.push(ctx.latestCommitBody);
  lines.push("```");
  lines.push("");
  if (ctx.gitStatus) {
    lines.push("### Uncommitted Changes");
    lines.push("```");
    lines.push(ctx.gitStatus);
    lines.push("```");
    lines.push("");
  }
  if (ctx.pipelineRounds) {
    lines.push("### Previous Pipeline Rounds (summaries)");
    lines.push(ctx.pipelineRounds);
    lines.push("");
  }
  if (ctx.extraContext) {
    lines.push("### Additional Context from Caller");
    lines.push(ctx.extraContext);
    lines.push("");
  }

  return lines.join("\n");
}
