// ANSI color helpers — no extra deps
export const c = {
  reset: "\x1b[0m",
  bold: "\x1b[1m",
  dim: "\x1b[2m",
  red: "\x1b[31m",
  green: "\x1b[32m",
  yellow: "\x1b[33m",
  blue: "\x1b[34m",
  magenta: "\x1b[35m",
  cyan: "\x1b[36m",
  white: "\x1b[37m",
  gray: "\x1b[90m",
  bgRed: "\x1b[41m",
  bgGreen: "\x1b[42m",
  bgYellow: "\x1b[43m",
  bgBlue: "\x1b[44m",
};

export function color(code: string, text: string): string {
  return `${code}${text}${c.reset}`;
}

const ICONS = {
  ok: color(c.green, "✓"),
  fail: color(c.red, "✗"),
  arrow: color(c.cyan, "→"),
  bullet: color(c.gray, "•"),
  thinking: color(c.yellow, "⟳"),
  plan: color(c.blue, "📋"),
  task: color(c.magenta, "▸"),
  exec: color(c.cyan, "▶"),
  warn: color(c.yellow, "⚠"),
  done: color(c.green, "✔"),
};

export function log(msg: string) {
  process.stdout.write(msg + "\n");
}

export function section(title: string) {
  log("");
  log(color(c.bold + c.cyan, `── ${title} `).padEnd(70, "─"));
}

export function step(icon: keyof typeof ICONS, msg: string) {
  log(`  ${ICONS[icon]}  ${msg}`);
}

export function indent(lines: string[], prefix = "     ") {
  lines.forEach((l) => log(`${prefix}${color(c.gray, l)}`));
}

export function priorityBadge(priority: string): string {
  const map: Record<string, string> = {
    critical: color(c.bgRed + c.white + c.bold, ` ${priority.toUpperCase()} `),
    high: color(c.red + c.bold, `[${priority.toUpperCase()}]`),
    medium: color(c.yellow + c.bold, `[${priority.toUpperCase()}]`),
    low: color(c.gray, `[${priority.toUpperCase()}]`),
  };
  return map[priority] ?? `[${priority}]`;
}

export function scopeBadge(scope: string): string {
  const map: Record<string, string> = {
    large: color(c.red, scope),
    medium: color(c.yellow, scope),
    small: color(c.green, scope),
  };
  return map[scope] ?? scope;
}

export function hr(char = "─", len = 70) {
  log(color(c.gray, char.repeat(len)));
}

export function printPlan(plan: {
  round_number: number;
  title: string;
  rationale: string;
  priority: string;
  estimated_scope: string;
  tasks: Array<{
    id: string;
    description: string;
    files_affected: string[];
    rationale: string;
  }>;
  skip_reason?: string | null;
}) {
  log("");
  hr("═");
  log(
    color(c.bold + c.blue, `  📋  ROUND ${plan.round_number}`) +
      "  " +
      color(c.bold + c.white, plan.title)
  );
  log(
    `  ${priorityBadge(plan.priority)}  scope: ${scopeBadge(plan.estimated_scope)}  tasks: ${color(c.cyan, String(plan.tasks.length))}`
  );
  hr("═");

  log("");
  log(color(c.bold, "  Rationale"));
  plan.rationale.split(". ").forEach((sentence) => {
    if (sentence.trim())
      log(`  ${color(c.gray, "│")}  ${sentence.trim() + (sentence.endsWith(".") ? "" : ".")}`);
  });

  if (plan.skip_reason) {
    log("");
    log(`  ${ICONS.warn}  ${color(c.yellow, "SKIP: " + plan.skip_reason)}`);
    return;
  }

  log("");
  log(color(c.bold, "  Tasks"));
  hr();
  plan.tasks.forEach((task) => {
    log(`  ${ICONS.task}  ${color(c.bold + c.magenta, task.id)}  ${task.description}`);
    log(`       ${color(c.gray, task.rationale)}`);
    if (task.files_affected.length) {
      log(
        `       ${color(c.dim, "files: " + task.files_affected.slice(0, 4).join(", ") + (task.files_affected.length > 4 ? ` +${task.files_affected.length - 4} more` : ""))}`
      );
    }
    log("");
  });
  hr();
}
