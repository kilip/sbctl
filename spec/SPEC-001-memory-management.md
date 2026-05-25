---
title: SPEC-001: Persistent Memory Management (SQLite + ent)
version: 1.2
date_created: 2026-05-26
last_updated: 2026-05-26
owner: Pak Bos
tags: [schema, architecture, sqlite, ent, go]
---

# Introduction

This specification defines the persistent storage layer for AI agent memory within `sbctl`. The primary focus is replacing the transient `InMemoryProvider` with a robust SQLite backend using the `ent` ORM, while maintaining minimal external dependencies and optimal performance.

## 1. Purpose & Scope

The objective is to provide a persistent data structure for episodic memory (conversation history) and semantic memory (long-term facts). The scope includes database schema design, database driver selection, and data retrieval strategies. This document serves as the implementation guide for developers building the memory layer.

## 2. Definitions

- **ent**: A graph-based entity framework for the Go programming language.
- **SQLite (Pure Go)**: A CGO-free SQL database implementation (e.g., modernc.org/sqlite).
- **UUID v7**: A time-ordered, lexicographically sortable UUID variant optimized for primary keys.
- **WAL Mode**: Write-Ahead Logging; an SQLite concurrency mode permitting simultaneous read and write operations.
- **Episodic Memory**: Session-based conversation history.
- **Semantic Memory**: Factual data extracted from interactions for long-term retrieval.
- **HITL (Human-in-the-Loop)**: A workflow requiring explicit human confirmation before executing sensitive actions.
- **Ent Hook**: An ent framework mechanism that intercepts entity mutations (create/update/delete) to trigger side effects automatically.

## 3. Requirements, Constraints & Guidelines

- **REQ-001 (Persistence Backend)**: The system MUST utilize SQLite as the primary persistence engine.
- **CON-001 (No-CGO Mandate)**: Implementation MUST use a pure Go SQLite driver (e.g., `modernc.org/sqlite`) for maximum portability and ease of cross-compilation.
- **IDN-001 (Identity Standard)**: All tables MUST utilize **UUID v7** for primary keys.
- **TIM-001 (Timestamp Naming)**: Temporal fields MUST be named `created`, `updated`, and `deleted` (omitting the `_at` suffix).
- **MEM-001 (Dual-Layer Memory)**: The architecture MUST support both Episodic Memory (Sessions/Messages) and Semantic Memory (Facts).
- **SEC-001 (Soft Deletion)**: Physical deletion is prohibited; entities MUST be marked via the `deleted` field for auditability and recovery.
- **SEC-002 (Encryption at Rest)**: Encryption at rest is **explicitly out of scope** for MVP. This may be revisited in a future release.
- **AUD-001 (Immutable Audit Log)**: The `AuditLog` table is **immutable by design**. Entries are append-only and MUST NOT be updated or soft-deleted. This is enforced via ent hooks at the schema level.

## 4. Interfaces & Data Contracts

### 4.1 Ent Schema Definitions

#### User Table
| Field | Type | Constraint |
|---|---|---|
| id | UUID (v7) | Primary Key |
| name | String | Required |
| role | Enum | "admin", "user" |
| created, updated, deleted | Time | Internal Audit |

#### Account Table (Platform Identity)
| Field | Type | Constraint |
|---|---|---|
| id | UUID (v7) | Primary Key |
| user_id | UUID | Foreign Key to User |
| platform | String | e.g., "telegram" |
| external_id | String | Indexed |
| status | Enum | "active", "banned", "deactivated" |
| metadata | JSON | Optional |

> **Note**: One User may have multiple Accounts across different platforms (e.g., Telegram, future interfaces). This is intentional and not over-engineering.

#### Session Table (Episodic Root)
| Field | Type | Constraint |
|---|---|---|
| id | UUID (v7) | Primary Key |
| account_id | UUID | Foreign Key to Account |
| profile_id | UUID | Foreign Key → `Profile`; nullable |
| title | String | Optional — short label for UI display |
| summary | Text | Optional — auto-generated session summary |
| created, updated, deleted | Time | Internal Audit |

> **Resolution rule**: if `profile_id IS NULL`, resolve to the user's `is_default = true` profile at runtime.

#### Profile Table
| Field | Type | Constraint |
|---|---|---|
| id | UUID (v7) | Primary Key |
| user_id | UUID | Foreign Key → `User` |
| name | String | Required; unique per user (e.g. `vault`, `sbctl`) |
| working_dir | String | Required; absolute or `~`-prefixed path |
| is_default | Bool | Default: `false`; only one record per user may be `true` |
| created, updated, deleted | Time | Internal Audit (soft-delete applies) |

> **Constraint**: Changing `is_default` to `true` on one profile MUST atomically set `is_default = false` on all other profiles for the same user. Enforce via ent hook or transaction.

#### Message Table (Episodic History)
| Field | Type | Constraint |
|---|---|---|
| id | UUID (v7) | Primary Key |
| session_id | UUID | Foreign Key to Session |
| role | Enum | "system", "user", "assistant", "tool" |
| content | JSON | Required — see Content Format below |
| status | Enum | "pending", "awaiting_approval", "done" |
| token_count | Integer | Optional |
| metadata | JSON | Optional (Tool Call IDs, etc.) |

**Message Content Format**

The `content` field is stored as JSON to accommodate different message roles:

```json
// role: user or assistant
{ "text": "Hello, what is the status of Project X?" }

// role: tool
{
  "tool_call_id": "abc123",
  "result": "...",
  "error": null
}
```

**Message Ordering**

Messages are sorted by `id` (UUID v7) as the primary sort key, leveraging its time-ordered property. Implementors should be aware that in the rare case of clock skew or bulk inserts within the same millisecond, ordering within that window is not guaranteed. For MVP, this is acceptable.

#### Memory Table (Semantic Facts)
| Field | Type | Constraint |
|---|---|---|
| id | UUID (v7) | Primary Key |
| user_id | UUID | Foreign Key to User |
| session_id | UUID | Foreign Key to Session — source session |
| message_id | UUID | Foreign Key to Message — source message |
| content | Text | Required |
| metadata | JSON | Optional |
| embedding | Blob | **Post-MVP** — Encoded []float32 |
| created, updated | Time | Internal Audit |

> **Note on Embedding**: The `embedding` column is reserved in the schema but will not be populated until post-MVP. For MVP, semantic retrieval is performed via full-text or metadata filtering only.

#### AuditLog Table (Security & Traceability)
| Field | Type | Constraint |
|---|---|---|
| id | UUID (v7) | Primary Key |
| user_id | UUID | Foreign Key to User |
| action | String | e.g., "tool_execution", "config_change" |
| entity_type | String | e.g., "message", "memory" |
| entity_id | UUID | Reference to target entity |
| metadata | JSON | Context (e.g., tool arguments, raw response) |
| created | Time | Required |

> **Note**: AuditLog is **append-only and immutable**. There are no `updated` or `deleted` fields by design. Entries are written automatically via ent hooks on every data-modifying operation — no manual logging required by business logic layers.

### 4.2 Memory Extraction Flow

Semantic memory extraction happens **before the agent sends a response**, not at session end. The flow is:

```
User Message → Agent Processing → Extract Facts (if any) → Save to Memory → Send Response
```

The agent is responsible for evaluating whether a conversation turn contains facts worth persisting. If facts are identified, they are written to the `Memory` table immediately, referencing the originating `session_id` and `message_id`.

## 5. Acceptance Criteria

- **AC-001**: Given an empty database, when `ent` executes auto-migration, then all defined tables must be correctly provisioned without errors.
- **AC-002**: Given a new message, when persisted to the `Message` table, then the generated ID must be a valid, chronologically ordered UUID v7.
- **AC-003**: Given soft-deleted data, when executing standard queries, the data must remain hidden. To explicitly query soft-deleted records, the caller must apply a `WhereDeleted` (or equivalent ent predicate) filter to override the default `deleted IS NULL` scope.
- **AC-004**: The system must operate in CGO-disabled environments (e.g., static builds on Alpine Linux).
- **AC-005**: Given a "Dangerous Tool" execution, when the operation completes, then the system must record the execution details in the `AuditLog` via ent hook — no manual logging call required.
- **AC-006**: Given a message with role "tool", the `content` field must be valid JSON containing at minimum `tool_call_id` and `result` keys.
- **AC-007**: Given a fact extracted during agent processing, it must be persisted to the `Memory` table with valid `session_id` and `message_id` references before the agent response is sent.
- **AC-008** *(AMENDMENT-001)*: Given a new User, a `vault` profile with `working_dir = ~/brain` and `is_default = true` must be auto-provisioned via ent hook on User creation.
- **AC-009** *(AMENDMENT-001)*: Given two profiles for the same user, only one may have `is_default = true` at any time — enforced at the persistence layer.
- **AC-010** *(AMENDMENT-001)*: Given a `Session` with `profile_id = NULL`, the Agent must resolve and use the user's default profile.

## 6. Test Automation Strategy

- **Test Levels**: Focus on unit testing for schemas and functional database integration.
- **Frameworks**: `testing` (Go standard library), `enttest` (ent testing utilities).
- **Data Management**: Utilize SQLite In-Memory (`file::memory:?cache=shared`) for unit tests to ensure isolation and performance.
- **Coverage**: Minimum 80% code coverage for CRUD operations within the storage layer.
- **Concurrency**: Execute tests with the `-race` flag to detect potential data races under concurrent access.
- **Vector Search Validation**: Deferred to post-MVP alongside embedding implementation.

## 7. Rationale & Context

- **Choice of SQLite**: As `sbctl` is a local/personal tool, SQLite provides zero-configuration persistence without the overhead of external database servers.
- **UUID v7 Adoption**: Facilitates future multi-device synchronization and maintains chronological integrity without the collisions associated with standard integer auto-increment during merge operations.
- **WAL Mode Necessity**: Bots frequently receive concurrent updates; WAL mode prevents "database is locked" errors by allowing simultaneous reads and writes.
- **CGO-Free Vector Search**: Since `sqlite-vss` and `sqlite-vec` require CGO and are not reliably cross-platform (Linux/macOS/Windows), `sbctl` employs a "Fetch-then-Rank" strategy for post-MVP. Embeddings are stored as `BLOB` (`[]float32`). During retrieval, the system fetches candidate records and calculates Cosine Similarity in the application layer. Given a personal memory scope (typically <10,000 records), latency remains sub-50ms.
- **JSON Message Content**: The `content` field uses JSON to cleanly accommodate the structural differences between user/assistant text and tool call results, avoiding fragile string parsing.
- **Inline Memory Extraction**: Extracting facts before responding (rather than at session end) eliminates the ambiguity of defining "session end" in an async messaging context and prevents data loss on unexpected disconnects.
- **AuditLog via Ent Hooks**: Centralizing audit writes in ent hooks ensures consistent, complete logging without requiring every business logic layer to remember to log.

## 8. Dependencies & External Integrations

### External Systems
- **EXT-001 (SQLite Engine)**: The underlying data storage engine.

### Third-Party Services
- **SVC-001**: None - No external third-party services are required for the persistence layer.

### Infrastructure Dependencies
- **INF-001 (Local Filesystem)**: The SQLite database file resides on the local filesystem.

### Data Dependencies
- **DAT-001**: None - This layer is the primary source of truth for agent memory.

### Technology Platform Dependencies
- **PLT-001 (entgo.io)**: ORM framework for entity management and migration.
- **PLT-002 (modernc.org/sqlite)**: Pure Go SQLite driver with no C dependencies.
- **PLT-003 (google/uuid)**: Library for UUID v7 generation.
- **PLT-004 (github.com/gonum/floats)**: Optional library for high-performance vector operations — **Post-MVP**.

### Compliance Dependencies
- **COM-001**: None - No specific regulatory compliance requirements identified.

## 8b. Default Data Seeding *(AMENDMENT-001)*

When a new `User` is provisioned, automatically create one `Profile` record via ent hook on `User` creation:

```go
Profile{
    UserID:     user.ID,
    Name:       "vault",
    WorkingDir: "~/brain",
    IsDefault:  true,
}
```

---

## 9. Examples & Edge Cases

### UUID v7 Generation in ent
```go
field.UUID("id", uuid.UUID{}).
    Default(func() uuid.UUID {
        id, _ := uuid.NewV7()
        return id
    })
```

### WAL Mode Initialization
```go
client, err := ent.Open("sqlite", "file:sbctl.db?cache=shared&_pragma=foreign_keys(1)&_journal_mode=WAL")
```

### Embedding Conversion (Helper — Post-MVP)
```go
func EncodeEmbedding(vec []float32) []byte {
    buf := new(bytes.Buffer)
    binary.Write(buf, binary.LittleEndian, vec)
    return buf.Bytes()
}
```

### Querying Soft-Deleted Records
```go
// Standard query — soft-deleted records excluded automatically
client.Message.Query().Where(message.DeletedIsNil()).All(ctx)

// Explicit query including soft-deleted records
client.Message.Query().All(ctx) // override default scope via ent predicate
```

### AuditLog Hook Example
```go
// Registered at schema level — fires automatically on any Message mutation
func AuditHook(next ent.Mutator) ent.Mutator {
    return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
        val, err := next.Mutate(ctx, m)
        if err == nil {
            // write to AuditLog
        }
        return val, err
    })
}
```

## 10. Validation Criteria

- **VAL-001**: Successful build with `CGO_ENABLED=0` to ensure no-CGO compliance.
- **VAL-002**: Automatic application of `deleted IS NULL` filters confirmed via schema tests.
- **VAL-003**: Foreign Key integrity validated for cascading deletes and protection rules.
- **VAL-004**: Audit log entries verified for all data-modifying tool executions via ent hook — no direct call required from business logic.
- **VAL-005**: Vector search accuracy — **deferred to post-MVP**.
- **VAL-006**: Memory extraction verified to complete before agent response is dispatched.
- **VAL-007**: Message `content` JSON structure validated per role type (user/assistant/tool).

## 11. Related Specifications / Further Reading

- [entgo.io Documentation](https://entgo.io/docs/getting-started)
- [RFC 9562: UUID Version 7](https://www.rfc-editor.org/rfc/rfc9562.html)
- [SPEC-002: Agent Provider & LLM Orchestration](./SPEC-002-agent-provider.md)