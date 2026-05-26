---
title: SPEC-004: Tool System
version: 1.0
date_created: 2026-05-26
last_updated: 2026-05-26
owner: Pak Bos
tags: [architecture, tools, registry, agent, go]
---

# Introduction

This specification defines the Tool System for `sbctl`, which provides the
mechanism for the AI agent to interact with the filesystem, Obsidian vault,
memory, and system. It covers the `Tool` interface, `ToolRegistry`, provider
conversion (Anthropic/OpenAI), and all MVP tool implementations.

## 1. Purpose & Scope

The objective is to provide a unified, provider-agnostic tool system that:
- Defines a single `Tool` interface all tools implement
- Manages tool registration, discovery, and availability checks
- Converts tool schemas to Anthropic and OpenAI formats at runtime
- Enforces HITL approval for dangerous tools (aligned with SPEC-002)
- Groups tools into toolsets for lifecycle management

Scope includes the registry, core types, provider converters, helpers,
and all MVP tool implementations.

## 2. Definitions

- **Tool**: A single capability the agent can invoke (e.g. `read_file`).
- **ToolEntry**: Complete metadata + handler for a registered tool.
- **Toolset**: A named group of related tools (e.g. `filesystem`, `obsidian`).
- **Classification**: Whether a tool is `safe` (auto-execute) or `dangerous` (requires HITL).
- **CheckFn**: An optional availability probe for a tool. Result is TTL-cached.
- **Generation**: A monotonically-increasing counter bumped on every registry mutation. Used for external cache invalidation.
- **HandlerFunc**: The Go function signature for tool execution.
- **ExecuteContext**: Session state passed to every tool execution (aligned with SPEC-002).
- **ProviderToolDef**: A provider-specific tool schema (Anthropic or OpenAI format).

## 3. Requirements, Constraints & Guidelines

- **REQ-001 (Provider Agnosticism)**: Tool schemas MUST be defined once in a provider-agnostic format and converted at runtime.
- **REQ-002 (Fail-Safe Classification)**: Any tool without an explicit `Classification` MUST default to `Dangerous` — aligned with SPEC-002 AC-008.
- **REQ-003 (Self-Registration)**: Each tool file MUST self-register via Go's `init()` function. No central list of tools.
- **REQ-004 (Thread Safety)**: The registry MUST be safe for concurrent reads and writes.
- **REQ-005 (CheckFn TTL)**: `CheckFn` results MUST be cached for ~30 seconds to avoid probing external state on every LLM turn.
- **REQ-006 (Shadow Protection)**: Registering a tool whose name is already taken by a different toolset MUST be rejected with an error.
- **CON-001 (No CGO)**: All tool implementations MUST be CGO-free for cross-platform portability.
- **CON-002 (WorkingDir)**: All filesystem tools MUST operate relative to `ExecuteContext.WorkingDir` — never the process working directory.
- **GUD-001 (Single Registry)**: One package-level singleton `Registry` is used across the entire application.

## 4. Interfaces & Data Contracts

### 4.1 Core Types

```go
// Classification determines whether HITL approval is required.
type Classification string

const (
    Safe      Classification = "safe"      // auto-execute
    Dangerous Classification = "dangerous" // requires HITL approval
)

// JSONSchema is a provider-agnostic parameter schema.
type JSONSchema struct {
    Type        string                `json:"type"`
    Description string                `json:"description,omitempty"`
    Properties  map[string]JSONSchema `json:"properties,omitempty"`
    Required    []string              `json:"required,omitempty"`
    Items       *JSONSchema           `json:"items,omitempty"`
    Enum        []string              `json:"enum,omitempty"`
}

// ToolSchema is the provider-agnostic tool definition.
type ToolSchema struct {
    Name        string     `json:"name"`
    Description string     `json:"description"`
    Parameters  JSONSchema `json:"parameters"`
}

// ToolResult is the unified result from every tool handler.
type ToolResult struct {
    Content  string         `json:"content,omitempty"`
    Error    string         `json:"error,omitempty"`
    Metadata map[string]any `json:"metadata,omitempty"`
}

// ExecuteContext carries session state for tool execution.
// Aligned with SPEC-002 §4.3.
type ExecuteContext struct {
    WorkingDir string // resolved from Profile.working_dir
    SessionID  string
    UserID     string
}

// HandlerFunc is the function signature for tool execution.
type HandlerFunc func(ctx context.Context, exec ExecuteContext, params map[string]any) ToolResult
```

### 4.2 ToolEntry

```go
type ToolEntry struct {
    Name           string         // unique identifier e.g. "read_file"
    Toolset        string         // group e.g. "filesystem"
    Classification Classification // defaults to Dangerous if empty
    Schema         ToolSchema
    Handler        HandlerFunc
    CheckFn        func() bool    // optional; TTL-cached ~30s
    Emoji          string         // for UI/logs e.g. "📖"
    MaxResultSizeChars int        // 0 = use global default
    DynamicSchemaOverrides func() map[string]any // optional runtime schema overrides
}
```

### 4.3 ToolRegistry Interface

```go
type ToolRegistry interface {
    // Registration
    Register(entry ToolEntry) error
    MustRegister(entry ToolEntry)   // panics on error; for use in init()
    Deregister(name string)

    // Lookup
    Get(name string) (ToolEntry, bool)
    List() []ToolEntry
    ListByToolset(toolset string) []ToolEntry
    ListByClassification(c Classification) []ToolEntry

    // Provider definitions (respects CheckFn TTL cache)
    GetDefinitions(format ProviderFormat, names []string) []ProviderToolDef

    // Dispatch
    Dispatch(ctx context.Context, name string, params map[string]any, exec ExecuteContext) ToolResult

    // Availability
    IsToolsetAvailable(toolset string) bool
    InvalidateCheckFnCache()

    // Cache invalidation
    Generation() int
}
```

### 4.4 Provider Conversion

```go
type ProviderFormat string

const (
    ProviderAnthropic ProviderFormat = "anthropic"
    ProviderOpenAI    ProviderFormat = "openai"
)

// ProviderToolDef is a provider-specific tool schema (opaque to the registry).
type ProviderToolDef map[string]any

// Anthropic format:
// {
//   "name": "read_file",
//   "description": "...",
//   "input_schema": { "type": "object", "properties": {...}, "required": [...] }
// }

// OpenAI format:
// {
//   "type": "function",
//   "function": {
//     "name": "read_file",
//     "description": "...",
//     "parameters": { "type": "object", "properties": {...}, "required": [...] }
//   }
// }
```

### 4.5 Helpers

```go
// ToolError returns a ToolResult carrying an error.
func ToolError(msg string, extra ...map[string]any) ToolResult

// ToolSuccess returns a ToolResult carrying content.
func ToolSuccess(content string, meta ...map[string]any) ToolResult

// ToolJSON returns a ToolResult with JSON-serialized data as content.
func ToolJSON(data any) ToolResult
```

### 4.6 Self-Registration Pattern

Every tool file registers itself via `init()`:

```go
// internal/agent/tools/filesystem/read_file.go
func init() {
    Registry.MustRegister(tools.ToolEntry{
        Name:           "read_file",
        Toolset:        "filesystem",
        Classification: tools.Safe,
        Emoji:          "📖",
        Schema: tools.ToolSchema{
            Name:        "read_file",
            Description: "Read the contents of a file at the given path.",
            Parameters: tools.JSONSchema{
                Type: "object",
                Properties: map[string]tools.JSONSchema{
                    "path": {
                        Type:        "string",
                        Description: "Relative path to the file from WorkingDir.",
                    },
                },
                Required: []string{"path"},
            },
        },
        Handler: handleReadFile,
    })
}
```

## 5. Package Structure

```
internal/
  agent/
    tools/
      tool.go           ← ToolEntry, Classification, JSONSchema, ToolResult,
                           ExecuteContext, HandlerFunc
      registry.go       ← ToolRegistry implementation, TTL cache,
                           generation counter, singleton Registry var
      convert.go        ← ToAnthropicTool(), ToOpenAITool(), GetDefinitions()
      helpers.go        ← ToolError(), ToolSuccess(), ToolJSON()
      filesystem/
        read_file.go    ← safe
        write_file.go   ← dangerous
        delete_file.go  ← dangerous
        list_dir.go     ← safe
      obsidian/
        read_note.go    ← safe
        create_note.go  ← dangerous
        update_note.go  ← dangerous
        delete_note.go  ← dangerous
        list_notes.go   ← safe
        search_notes.go ← safe
      memory/
        search_memory.go ← safe
      system/
        terminal.go     ← dangerous
```

## 6. MVP Tool Definitions

### 6.1 Toolset: `filesystem`

| Tool | Classification | Description |
|---|---|---|
| `read_file` | safe | Read file contents |
| `write_file` | dangerous | Write/overwrite a file |
| `delete_file` | dangerous | Delete a file |
| `list_dir` | safe | List directory contents |

### 6.2 Toolset: `obsidian`

| Tool | Classification | Description |
|---|---|---|
| `read_note` | safe | Read an Obsidian note by path |
| `create_note` | dangerous | Create a new note |
| `update_note` | dangerous | Update note content or frontmatter |
| `delete_note` | dangerous | Delete a note |
| `list_notes` | safe | List notes, optionally filtered by folder/tag |
| `search_notes` | safe | Full-text search across the vault |

### 6.3 Toolset: `memory`

| Tool | Classification | Description |
|---|---|---|
| `search_memory` | safe | Search semantic memory from SPEC-001 Memory table |

### 6.4 Toolset: `system`

| Tool | Classification | Description |
|---|---|---|
| `terminal` | dangerous | Execute a shell command in WorkingDir |

## 7. CheckFn TTL Cache

- Cache duration: **30 seconds**
- Keyed per `CheckFn` function pointer
- Swallows panics/errors → returns `false` (tool marked unavailable)
- Invalidated via `InvalidateCheckFnCache()` after config changes
- Per-call dedup: same `CheckFn` called only once per `GetDefinitions()` pass

## 8. HITL Integration

Aligned with SPEC-002 §4.5 and SPEC-003 §4.1:

- `Classification: Dangerous` → agent intercepts before execution
- Message status set to `awaiting_approval` in SPEC-001 `Message` table
- Gateway sends approval prompt via `SendApproval()` (SPEC-003)
- On `approve:<tool_call_id>` → `Registry.Dispatch()` is called
- On `deny:<tool_call_id>` → error returned to LLM as tool result
- Timeout: **30 minutes** — auto-cancel with user notification

## 9. Acceptance Criteria

- **AC-001**: Given a tool registered without `Classification`, its effective classification MUST be `Dangerous`.
- **AC-002**: Given two tools with the same name from different toolsets, the second `Register()` call MUST return an error.
- **AC-003**: Given a `CheckFn` that returns `false`, the tool MUST be excluded from `GetDefinitions()` output.
- **AC-004**: Given a `CheckFn` that panics, the tool MUST be treated as unavailable (not crash the registry).
- **AC-005**: Given `GetDefinitions()` called twice within 30 seconds, `CheckFn` MUST be called only once (TTL cache hit).
- **AC-006**: Given `InvalidateCheckFnCache()` called, the next `GetDefinitions()` MUST re-probe all `CheckFn`s.
- **AC-007**: Given a `write_file` tool call from the LLM, the agent MUST intercept and set message status to `awaiting_approval` before execution.
- **AC-008**: Given a `read_file` tool call, the path MUST be resolved relative to `ExecuteContext.WorkingDir`.
- **AC-009**: Given `Generation()` called before and after `Register()`, the value MUST have incremented.
- **AC-010**: Given all tool files imported, the registry MUST contain all tools from all toolsets via `init()` self-registration.

## 10. Test Automation Strategy

- **Unit Tests**: Registry operations (register, deregister, shadow protection, classification default).
- **Unit Tests**: CheckFn TTL cache (hit, miss, invalidation, panic recovery).
- **Unit Tests**: Provider conversion (Anthropic and OpenAI format correctness).
- **Unit Tests**: Each tool handler with mocked filesystem/vault/memory.
- **Integration Tests**: Full dispatch flow — LLM params → Dispatch() → ToolResult.
- **Concurrency**: Run with `-race` flag; registry must be safe under parallel Register/Dispatch.
- **Coverage**: Minimum 80% for registry and tool handlers.

## 11. Rationale & Context

- **`init()` self-registration**: Eliminates a central tool list that needs manual maintenance. Adding a new tool = add a file. Removing = delete the file. The import in the main package activates it.
- **Fail-safe Dangerous default**: Prevents accidental auto-execution of new unclassified tools. Explicitly opt-in to `Safe`.
- **TTL cache for CheckFn**: External probes (docker, binary existence, env vars) are slow and change on human timescales. 30s cache balances freshness vs. performance.
- **Generation counter**: Allows callers (e.g. agent's tool definition cache) to cheaply detect registry changes without diffing the full tool list.
- **Provider-agnostic schema**: Define once, convert at call time. Avoids schema drift between providers.
- **WorkingDir enforcement**: All filesystem ops relative to `ExecuteContext.WorkingDir` prevents tools from accessing arbitrary paths outside the configured vault/project.

## 12. Dependencies & External Integrations

### Internal Dependencies
- **INF-001 (SPEC-001)**: `search_memory` tool queries the `Memory` table.
- **INF-002 (SPEC-002)**: `ExecuteContext` is populated by the agent before dispatch.
- **INF-003 (SPEC-003)**: Gateway calls `Dispatch()` after HITL approval.

### External Dependencies
- None — all tools operate on local filesystem or internal DB.

### Technology Platform Dependencies
- **PLT-001**: Go standard library (`os`, `path/filepath`, `context`).
- **PLT-002**: Pure Go only — no CGO.

## 13. Related Specifications / Further Reading

- [SPEC-001: Persistent Memory Management](./SPEC-001-memory-management.md)
- [SPEC-002: Agent Provider & LLM Orchestration](./SPEC-002-agent-provider.md)
- [SPEC-003: Gateway Interface & Telegram](./SPEC-003-gateway.md)
