import Fastify from "fastify";
import { decideIteration } from "./decide.js";
import type { DecideRequest, DecideResponse } from "./types.js";

const PORT = Number(process.env.PORT) || 4001;
const HOST = process.env.HOST ?? "0.0.0.0";

const app = Fastify({
  logger: {
    transport: {
      target: "pino-pretty",
      options: {
        colorize: true,
        translateTime: "HH:MM:ss",
        ignore: "pid,hostname",
        messageFormat:
          "{msg} {if req.method}{req.method} {req.url}{end} {if res.statusCode}→ {res.statusCode} ({responseTime}ms){end}",
      },
    },
  },
});

// ─── Health ───────────────────────────────────────────────────────────────────

app.get("/health", async () => {
  return { status: "ok", service: "master-agent" };
});

// ─── Decide Iteration ─────────────────────────────────────────────────────────

app.post<{ Body: DecideRequest; Reply: DecideResponse | { error: string } }>(
  "/decide-iteration",
  async (req, reply) => {
    const { extra_context } = req.body ?? {};
    const extraLog = extra_context ? ` [extra: "${extra_context}"]` : "";
    req.log.info(`decide-iteration requested${extraLog}`);

    try {
      req.log.info("gathering project context...");
      req.log.info("calling claude CLI (claude-opus-4-6)...");

      const start = Date.now();
      const result = await decideIteration(extra_context);
      const elapsed = Date.now() - start;

      const { plan } = result;
      req.log.info(
        `✓ plan ready  round=${plan.round_number}  "${plan.title}"  priority=${plan.priority}  scope=${plan.estimated_scope}  tasks=${plan.tasks.length}  elapsed=${elapsed}ms`
      );
      plan.tasks.forEach((t) => {
        req.log.info(`  task ${t.id}: ${t.description}`);
      });

      return result;
    } catch (err: unknown) {
      const message = err instanceof Error ? err.message : String(err);
      req.log.error(`decide-iteration failed: ${message}`);
      reply.status(500);
      return { error: message };
    }
  }
);

// ─── Boot ─────────────────────────────────────────────────────────────────────

try {
  await app.listen({ port: PORT, host: HOST });
} catch (err) {
  app.log.error(err);
  process.exit(1);
}
