import Fastify from "fastify";
import { decideIteration } from "./decide.js";
import type { DecideRequest } from "./types.js";

const PORT = Number(process.env.PORT) || 4001;
const HOST = process.env.HOST ?? "0.0.0.0";

const app = Fastify({ logger: true });

// ─── Health ───────────────────────────────────────────────────────────────────

app.get("/health", async () => {
  return { status: "ok", service: "master-agent" };
});

// ─── Decide Iteration ─────────────────────────────────────────────────────────

app.post<{ Body: DecideRequest }>("/decide-iteration", async (req, reply) => {
  const { extra_context } = req.body ?? {};

  try {
    const result = await decideIteration(extra_context);
    return result;
  } catch (err: unknown) {
    const message = err instanceof Error ? err.message : String(err);
    reply.status(500);
    return { error: message };
  }
});

// ─── Boot ─────────────────────────────────────────────────────────────────────

try {
  await app.listen({ port: PORT, host: HOST });
  console.log(`master-agent listening on ${HOST}:${PORT}`);
} catch (err) {
  app.log.error(err);
  process.exit(1);
}
