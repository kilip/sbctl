---
title: SPEC-001: Persistent Memory Management (SQLite + ent)
version: 1.0
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

## 3. Requirements, Constraints & Guidelines

- **REQ-001 (Persistence Backend)**: The system MUST utilize SQLite as the primary persistence engine.
- **CON-001 (No-CGO Mandate)**: Implementation MUST use a pure Go SQLite driver (e.g., `modernc.org/sqlite`) for maximum portability and ease of cross-compilation.
- **IDN-001 (Identity Standard)**: All tables MUST utilize **UUID v7** for primary keys.
- **TIM-001 (Timestamp Naming)**: Temporal fields MUST be named `created`, `updated`, and `deleted` (omitting the `_at` suffix).
- **MEM-001 (Dual-Layer Memory)**: The architecture MUST support both Episodic Memory (Sessions/Messages) and Semantic Memory (Facts).
- **SEC-001 (Soft Deletion)**: Physical deletion is prohibited; entities MUST be marked via the `deleted` field for auditability and recovery.

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
| metadata | JSON | Optional |

#### Session Table (Episodic Root)
| Field | Type | Constraint |
|---|---|---|
| id | UUID (v7) | Primary Key |
| account_id | UUID | Foreign Key to Account |
| created, updated, deleted | Time | Internal Audit |

#### Message Table (Episodic History)
| Field | Type | Constraint |
|---|---|---|
| id | UUID (v7) | Primary Key |
| session_id | UUID | Foreign Key to Session |
| role | Enum | "system", "user", "assistant", "tool" |
| content | Text | Required |
| token_count | Integer | Optional |
| metadata | JSON | Optional (Tool Call IDs, etc.) |

#### Memory Table (Semantic Facts)
| Field | Type | Constraint |
|---|---|---|
| id | UUID (v7) | Primary Key |
| user_id | UUID | Foreign Key to User |
| content | Text | Required |
| metadata | JSON | Optional |
| embedding | Blob | Required (Encoded []float32) |
| created, updated | Time | Internal Audit |

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

## 5. Acceptance Criteria

- **AC-001**: Given an empty database, when `ent` executes auto-migration, then all defined tables must be correctly provisioned without errors.
- **AC-002**: Given a new message, when persisted to the `Message` table, then the generated ID must be a valid, chronologically ordered UUID v7.
- **AC-003**: Given soft-deleted data, when executing standard queries, then the data must remain hidden unless explicitly requested.
- **AC-004**: The system must operate in CGO-disabled environments (e.g., static builds on Alpine Linux).
- **AC-005**: Given a "Dangerous Tool" execution, when the operation completes, then the system must record the execution details in the `AuditLog`.

## 6. Test Automation Strategy

- **Test Levels**: Focus on unit testing for schemas and functional database integration.
- **Frameworks**: `testing` (Go standard library), `enttest` (ent testing utilities).
- **Data Management**: Utilize SQLite In-Memory (`file::memory:?cache=shared`) for unit tests to ensure isolation and performance.
- **Coverage**: Minimum 80% code coverage for CRUD operations within the storage layer.
- **Concurrency**: Execute tests with the `-race` flag to detect potential data races under concurrent access.
- **Vector Search Validation**: Verify cosine similarity accuracy using a controlled dataset with known similarity scores.

## 7. Rationale & Context

- **Choice of SQLite**: As `sbctl` is a local/personal tool, SQLite provides zero-configuration persistence without the overhead of external database servers.
- **UUID v7 Adoption**: Facilitates future multi-device synchronization and maintains chronological integrity without the collisions associated with standard integer auto-increment during merge operations.
- **WAL Mode Necessity**: Bots frequently receive concurrent updates; WAL mode prevents "database is locked" errors by allowing simultaneous reads and writes.
- **CGO-Free Vector Search**: Since `sqlite-vss` requires CGO, `sbctl` employs a "Fetch-then-Rank" strategy. Embeddings are stored as `BLOB` (`[]float32`). During retrieval, the system fetches candidate records (filtered by metadata or recent limits) and calculates Cosine Similarity in the application layer. Given a personal memory scope (typically <10,000 records), latency remains sub-50ms.

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
- **PLT-004 (github.com/gonum/floats)**: Optional library for high-performance vector operations in Go.

### Compliance Dependencies
- **COM-001**: None - No specific regulatory compliance requirements identified.

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

### Embedding Conversion (Helper)
```go
func EncodeEmbedding(vec []float32) []byte {
    buf := new(bytes.Buffer)
    binary.Write(buf, binary.LittleEndian, vec)
    return buf.Bytes()
}
```

## 10. Validation Criteria

- **VAL-001**: Successful build with `CGO_ENABLED=0` to ensure no-CGO compliance.
- **VAL-002**: Automatic application of `deleted IS NULL` filters confirmed via schema tests.
- **VAL-003**: Foreign Key integrity validated for cascading deletes and protection rules.
- **VAL-004**: Audit log entries verified for all data-modifying tool executions.
- **VAL-005**: Vector search accuracy verified within 0.01 margin for known similarity scores.

## 11. Related Specifications / Further Reading

- [entgo.io Documentation](https://entgo.io/docs/getting-started)
- [RFC 9562: UUID Version 7](https://www.rfc-editor.org/rfc/rfc9562.html)
- [SPEC-002: Agent Provider & LLM Orchestration](./SPEC-002-agent-provider.md)
