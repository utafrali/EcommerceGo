# EcommerceGo Agent Communication Protocol

## Overview

This document defines the formal communication protocol for all AI agents operating within the EcommerceGo project. Every inter-agent communication MUST conform to this specification. Deviation from this protocol will cause coordination failures and must be treated as a blocker.

The system is hierarchical: **Master Agent is the sole coordinator**. Sub-agents never communicate directly with each other. All messages flow through Master.

---

## 1. Message Format

All messages are structured JSON objects. Every message sent between agents must use the following envelope:

```json
{
  "message_id": "msg_<uuid_v4>",
  "timestamp": "2026-02-25T10:00:00Z",
  "from": "<agent_id>",
  "to": "<agent_id>",
  "type": "<message_type>",
  "priority": "<priority_level>",
  "thread_id": "thread_<uuid_v4>",
  "context": {
    "sprint": "sprint-01",
    "milestone": "phase-1-foundation",
    "feature": "product-catalog",
    "task_id": "task_<uuid_v4>",
    "parent_task_id": "task_<uuid_v4> | null",
    "correlation_id": "<uuid_v4>"
  },
  "payload": {}
}
```

### Field Definitions

| Field | Type | Required | Description |
|---|---|---|---|
| `message_id` | string | yes | Globally unique message identifier. Format: `msg_` + UUIDv4 |
| `timestamp` | string | yes | ISO 8601 UTC timestamp of message creation |
| `from` | string | yes | Sending agent ID (see Agent Registry below) |
| `to` | string | yes | Receiving agent ID. Always `master` for sub-agent outbound |
| `type` | string | yes | Message type from the defined type catalog |
| `priority` | string | yes | One of: `critical`, `high`, `medium`, `low` |
| `thread_id` | string | yes | Groups related messages in a conversation chain |
| `context` | object | yes | Task and project context fields |
| `payload` | object | yes | Type-specific message body |

### Agent Registry

| Agent ID | Role |
|---|---|
| `master` | Master Orchestrator Agent |
| `tpm` | Technical Program Manager Agent |
| `product-manager` | Product Manager Agent |
| `backend-dev` | Backend Developer Agent |
| `frontend-dev` | Frontend Developer Agent |
| `devops` | DevOps / Infrastructure Agent |
| `qa` | Quality Assurance Agent |
| `security` | Security Agent |

---

## 2. Message Types

### `task_assignment`

Sent exclusively by Master to a sub-agent. Assigns a concrete unit of work.

```json
{
  "type": "task_assignment",
  "payload": {
    "task_id": "task_<uuid>",
    "title": "Implement CartService.AddItem in cart service",
    "description": "Implement the AddItem business logic following the product service pattern.",
    "acceptance_criteria": [
      "AddItem validates product exists via gRPC call to product-service",
      "AddItem persists cart to Redis with 24h TTL",
      "AddItem publishes cart.item_added event to Kafka topic ecommerce.cart.item_added",
      "Unit tests cover happy path, product-not-found, and invalid quantity cases",
      "Function signatures use context.Context as first parameter"
    ],
    "inputs": {
      "reference_files": [
        "services/product/internal/service/product.go",
        "services/product/internal/domain/product.go"
      ],
      "dependencies_ready": ["product-service gRPC proto compiled"]
    },
    "outputs_expected": [
      "services/cart/internal/service/cart.go",
      "services/cart/internal/service/cart_test.go"
    ],
    "estimated_complexity": "medium",
    "deadline_hint": "before checkout-service task_<uuid> begins"
  }
}
```

### `status_update`

Sent by sub-agents to Master to report progress on an in-flight task.

```json
{
  "type": "status_update",
  "payload": {
    "task_id": "task_<uuid>",
    "status": "in_progress",
    "percent_complete": 60,
    "summary": "CartService struct and AddItem skeleton implemented. Writing unit tests now.",
    "files_modified": [
      "services/cart/internal/service/cart.go"
    ],
    "blockers": [],
    "next_step": "Complete table-driven tests then send review_request"
  }
}
```

### `review_request`

Sent by sub-agents to Master when a task output is ready for review.

```json
{
  "type": "review_request",
  "payload": {
    "task_id": "task_<uuid>",
    "summary": "CartService.AddItem implemented with full test coverage.",
    "deliverables": [
      {
        "file": "services/cart/internal/service/cart.go",
        "description": "Service implementation with AddItem, RemoveItem, GetCart"
      },
      {
        "file": "services/cart/internal/service/cart_test.go",
        "description": "12 table-driven tests, all passing"
      }
    ],
    "self_assessment": {
      "acceptance_criteria_met": ["all"],
      "known_gaps": [],
      "suggested_next_tasks": ["Implement cart HTTP handler", "Wire Redis repository"]
    },
    "test_results": "go test ./... PASS (12/12)"
  }
}
```

### `question`

Sent by sub-agents to Master when clarification is needed before proceeding. Do not block on ambiguity without asking.

```json
{
  "type": "question",
  "payload": {
    "task_id": "task_<uuid>",
    "question": "Should cart items cap quantity at 99 or be unconstrained?",
    "context": "The acceptance criteria do not specify a maximum quantity. Inventory service is not yet available to query.",
    "options": [
      {"option": "A", "description": "Cap at 99, return INVALID_INPUT error above that"},
      {"option": "B", "description": "No cap at service level, enforce at handler validation"},
      {"option": "C", "description": "Configurable via environment variable, default 99"}
    ],
    "impact_of_delay": "Blocking test completion for cart service"
  }
}
```

### `completion`

Sent by sub-agents to Master when a task is fully done and accepted.

```json
{
  "type": "completion",
  "payload": {
    "task_id": "task_<uuid>",
    "title": "CartService.AddItem implementation",
    "outcome": "All 12 tests passing. Files committed to feature branch.",
    "artifacts": [
      "services/cart/internal/service/cart.go",
      "services/cart/internal/service/cart_test.go"
    ],
    "notes": "Quantity capped at 99 per Master decision in thread_<uuid>.",
    "unlocks_tasks": ["task_<uuid_handler>", "task_<uuid_integration_test>"]
  }
}
```

### `blocker`

Sent by sub-agents to Master when progress is completely halted. Use sparingly — attempt reasonable resolution first.

```json
{
  "type": "blocker",
  "payload": {
    "task_id": "task_<uuid>",
    "blocker_type": "dependency_missing | ambiguity | technical | external",
    "description": "Cannot implement gRPC call to product-service: proto file not yet generated.",
    "attempted_resolutions": [
      "Checked proto/ directory — product.proto exists but generated Go code is absent",
      "Makefile proto-gen target requires protoc installed; CI environment lacks it"
    ],
    "required_from": "devops",
    "urgency": "high",
    "workaround_possible": false
  }
}
```

---

## 3. Priority Levels

| Priority | Definition | Response SLA |
|---|---|---|
| `critical` | Production-blocking or security issue. No work can proceed. | Immediate |
| `high` | Blocks 2+ downstream tasks or a milestone deliverable. | Same session |
| `medium` | Slows progress but a workaround exists. | Next session |
| `low` | Nice-to-have information or minor polish. | Backlog |

Default priority for `task_assignment` is `medium` unless Master specifies otherwise.
Default priority for `blocker` is `high` unless otherwise justified.

---

## 4. Task Lifecycle

```
ASSIGNED
   │
   ▼
IN_PROGRESS  ──► BLOCKED (sub-agent sends blocker message)
   │                 │
   │                 ▼
   │            Master resolves or reassigns
   │                 │
   ▼                 ▼
REVIEW ◄─────────────┘
   │
   ├─► REVISION_REQUESTED (Master sends follow-up task_assignment with changes)
   │         │
   │         ▼
   │       IN_PROGRESS (again)
   │
   ▼
COMPLETED
```

State transitions are reported via `status_update` and `completion` messages. Master is responsible for tracking the authoritative state of every task.

---

## 5. Routing Rules

**Rule 1: Sub-agents never message sub-agents.**
If Backend Developer needs a DevOps artifact, it sends a `question` or flags a `blocker` to Master. Master resolves it with DevOps and provides the result back to Backend Developer.

**Rule 2: All task assignments originate from Master.**
No sub-agent may self-assign work or re-delegate to another sub-agent. Sub-agents may *suggest* follow-up tasks in `review_request.self_assessment.suggested_next_tasks`, but Master decides.

**Rule 3: Questions must include options.**
When asking a question, always provide at least two concrete options. Do not ask open-ended questions without proposed answers.

**Rule 4: Status updates on long tasks.**
For any task estimated to take more than one work session, send a `status_update` at the 50% mark and again when entering the review phase.

**Rule 5: Include context in every message.**
Always populate `context.task_id`, `context.sprint`, and `context.milestone`. Missing context causes tracking failures.

---

## 6. Handoff Protocol

When passing work from one agent to another (e.g., Backend Developer completes an API, then Frontend Developer needs to consume it), the handoff follows this sequence:

1. **Producing agent** sends `completion` to Master with full artifact list.
2. **Master** validates output against acceptance criteria.
3. **Master** creates a new `task_assignment` for the consuming agent, including in `inputs.dependencies_ready` the exact artifact paths from the producing agent's `completion.artifacts`.
4. **Consuming agent** acknowledges with a `status_update` (status: `in_progress`) within the same session.

Handoff artifacts must always be real file paths relative to the repository root. Never use abstract descriptions like "the service I built" — cite the exact file.

---

## 7. Escalation Protocol

Escalate to Master (raise priority to `critical` or `high`) when any of the following occur:

| Trigger | Action |
|---|---|
| Blocker unresolved for more than one session | Send `blocker` with `urgency: critical` |
| Acceptance criteria are contradictory | Send `question` immediately before writing any code |
| Security vulnerability discovered during implementation | Send `blocker` with `blocker_type: technical`, tag security agent |
| A design decision will affect 3+ services | Send `question` before proceeding |
| Test coverage falls below threshold (80% unit / 60% integration) | Send `blocker` to Master and QA agent via Master |
| An output from another agent is incorrect or missing | Send `blocker` citing the originating task_id |

Master is always available to receive escalations. When in doubt, escalate rather than guess.

---

## 8. Message Thread Management

- All reply messages must carry the same `thread_id` as the initiating message.
- A new `thread_id` is generated only for new, independent work streams.
- `task_assignment` messages always start a new thread.
- All subsequent messages for that task (status updates, reviews, completions) use the same `thread_id`.
- Questions and their answers must share the same `thread_id` as the parent task.

---

## 9. Versioning

This protocol is at version `1.0.0`. Any agent that receives a message it cannot parse must respond with a `blocker` citing `blocker_type: ambiguity` and include the raw message for Master to resolve.
