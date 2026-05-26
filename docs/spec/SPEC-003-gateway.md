---
title: SPEC-003: Gateway Interface & Telegram Implementation
version: 1.3
date_created: 2026-05-26
last_updated: 2026-05-26
owner: Pak Bos
tags: [interface, gateway, telegram, bot, hitl, go]
---

# Introduction

This specification defines the implementation of the `internal/gateway` module, serving as the primary communication interface for `sbctl`. The gateway abstracts all platform-specific transport concerns behind a unified interface, enabling the agent to communicate across multiple platforms (Telegram, WhatsApp, CLI, etc.) without coupling to any single provider. For MVP, Telegram is the only implementation.

## 1. Purpose & Scope

The objective is to define a clean, extensible gateway abstraction that:
- Integrates with the daemon worker lifecycle (SPEC-daemon)
- Routes inbound messages to the agent (SPEC-002)
- Resolves platform identities to persistent accounts and sessions (SPEC-001)
- Supports Human-in-the-Loop (HITL) approval flows (SPEC-002)
- Injects session context into the agent's system prompt (SPEC-002)

Scope includes the Gateway interface, session origin tracking, session key strategy, session reset policy, and the Telegram MVP implementation.

## 2. Definitions

- **Gateway**: A transport abstraction that implements the `daemon.Worker` interface. Each platform has its own Gateway worker.
- **GatewayContext**: Metadata describing the origin of an inbound message (platform, user, chat).
- **SessionSource**: Rich origin metadata used to build session keys and inject context into the system prompt. Inspired by Hermes gateway.
- **SessionContext**: Full context assembled for a session, injected into the agent's dynamic system prompt.
- **Session Key**: A deterministic string derived from `SessionSource`, used to map platform identities to SPEC-001 `Session` records.
- **Reset Policy**: Configuration governing when a session is automatically expired (idle timeout or daily reset).
- **HITL (Human-in-the-Loop)**: A workflow requiring explicit human confirmation before executing dangerous tools.
- **Allowlist**: A predefined list of authorized platform User IDs permitted to interact with the bot.
- **MarkdownV2**: Telegram's rich text format requiring strict character escaping.

## 3. Requirements, Constraints & Guidelines

- **REQ-001 (Platform Agnosticism)**: The `Gateway` interface MUST remain decoupled from any specific platform transport. All platform-specific logic lives in implementation packages (e.g., `internal/gateway/telegram`).
- **REQ-002 (Worker Integration)**: Every `Gateway` implementation MUST implement `daemon.Worker` so its lifecycle is managed by the daemon. Starting the daemon starts all configured gateways.
- **REQ-003 (Identity Verification)**: Each gateway implementation MUST verify user identity against its configured allowlist before processing any inbound message.
- **REQ-004 (HITL Support)**: The gateway MUST support approval flows using platform-native UI (e.g., Inline Keyboards for Telegram).
- **REQ-005 (Session Origin Tracking)**: Every inbound message MUST carry a `SessionSource` describing its origin for session key derivation and system prompt injection.
- **REQ-006 (Config-Driven Activation)**: A gateway is activated only when its credentials are present in the configuration. Missing credentials = gateway not started.
- **REQ-007 (Scalable to Multi-User)**: Session key strategy MUST support multi-user scenarios from day one, even if MVP only serves a single user.
- **CON-001 (Single Source of Truth)**: SPEC-001 is the single source of truth for `Session` and `Account` persistence. The gateway layer does NOT maintain its own session store.
- **CON-002 (Strict Escaping)**: All outgoing text for Telegram MUST pass through a `MarkdownV2Escaper` utility.
- **SEC-001 (Credential Protection)**: Bot tokens and API keys MUST never appear in logs, CLI output, or error messages.
- **UX-001 (Typing Indicator)**: The gateway SHOULD emit a typing/processing indicator while the agent is working, and stop once the response is sent or an error occurs.

## 4. Interfaces & Data Contracts

### 4.1 Gateway Interface

```go
// internal/gateway/gateway.go

package gateway

import (
    "context"

    "github.com/kilip/sbctl/internal/daemon"
)

// Gateway is the core transport abstraction. Every platform implementation
// must implement this interface. It extends daemon.Worker so its lifecycle
// is managed by the daemon.
type Gateway interface {
    daemon.Worker // Name() string; Start(ctx context.Context) error

    // Outbound
    SendMessage(ctx context.Context, userID string, text string) error
    SendApproval(ctx context.Context, userID string, req ApprovalRequest) error
    SetTyping(ctx context.Context, userID string, active bool) error

    // Inbound handlers — registered before Start() is called
    OnMessage(handler MessageHandler)
    OnCallback(handler CallbackHandler)

    // Lifecycle
    Stop(ctx context.Context) error
}

// ApprovalRequest carries the data needed to present a HITL approval prompt.
type ApprovalRequest struct {
    ToolCallID  string
    Description string // Human-readable description of the action to approve
}

// MessageHandler is called for every inbound user message.
type MessageHandler func(ctx context.Context, gctx GatewayContext, text string)

// CallbackHandler is called when a user responds to an approval prompt.
type CallbackHandler func(ctx context.Context, gctx GatewayContext, data string)
```

### 4.2 GatewayContext

`GatewayContext` carries the raw origin metadata for a single inbound event. It is the input to `build_session_key()` and is used by the handler layer to resolve `Account` and `Session` from SPEC-001.

```go
// internal/gateway/session.go

// GatewayContext is the minimal origin metadata attached to every
// inbound event. It is transport-layer data only — no business logic.
type GatewayContext struct {
    Platform   string // e.g. "telegram", "whatsapp", "cli"
    ExternalID string // platform-specific user ID
    ChatType   string // "dm", "group", "channel", "thread"
    ChatID     string // platform-specific chat/conversation ID
    ChatName   string // human-readable chat name (optional)
    ThreadID   string // forum topic, Discord thread, etc. (optional)
    UserName   string // display name of the sender (optional)
    Raw        any    // platform-specific raw update payload, for edge cases
}
```

### 4.3 SessionSource

`SessionSource` is a richer version of `GatewayContext`, built after resolving optional platform-specific metadata. It is used for session key derivation and system prompt injection.

```go
// SessionSource describes where a message originated from.
// Used to:
// 1. Build a deterministic session key
// 2. Route responses back to the correct platform and chat
// 3. Inject session context into the agent system prompt
type SessionSource struct {
    Platform    string
    ChatID      string
    ChatName    string
    ChatType    string // "dm", "group", "channel", "thread"
    UserID      string
    UserName    string
    ThreadID    string
    ChatTopic   string // Channel topic/description (optional)
    GuildID     string // Discord guild / Slack workspace (optional)
    ParentChatID string // Parent channel when ChatID refers to a thread (optional)
    MessageID   string // ID of the triggering message (optional)
}

// Description returns a human-readable description of the source.
func (s SessionSource) Description() string { ... }
```

### 4.4 Session Key Strategy

`BuildSessionKey` is the single source of truth for session key construction. It produces a deterministic string that maps a platform interaction to a SPEC-001 `Session`.

```go
// BuildSessionKey derives a deterministic session key from a SessionSource.
//
// Key format:
//   DM:      {platform}:dm:{chat_id}[:{thread_id}]
//   Group:   {platform}:{chat_type}:{chat_id}[:{thread_id}][:{user_id}]
//
// Multi-user behavior:
//   - DMs are always per-user isolated.
//   - Threads are shared by default (thread_sessions_per_user=false).
//   - Groups are per-user isolated by default (group_sessions_per_user=true).
func BuildSessionKey(
    source SessionSource,
    groupSessionsPerUser bool,
    threadSessionsPerUser bool,
) string { ... }
```

**Examples:**

| Scenario | Key |
|---|---|
| Telegram DM | `telegram:dm:123456789` |
| Telegram group, per-user | `telegram:group:987654321:123456789` |
| Telegram forum topic (thread) | `telegram:group:987654321:thread_42` |
| Future WhatsApp DM | `whatsapp:dm:+628123456789` |

### 4.5 SessionContext

`SessionContext` is assembled from `SessionSource` and configuration. It is injected into the agent's dynamic system prompt so the agent understands its environment.

```go
// SessionContext carries full session metadata for system prompt injection.
type SessionContext struct {
    Source                 SessionSource
    ConnectedPlatforms     []string
    SharedMultiUserSession bool

    // Populated from SPEC-001 Session after resolution
    SessionKey  string
    SessionID   string
    CreatedAt   time.Time
    UpdatedAt   time.Time
}

// BuildSessionContextPrompt produces the dynamic system prompt section
// that tells the agent about its current session context.
func BuildSessionContextPrompt(ctx SessionContext) string { ... }
```

Example output injected into system prompt:

```
## Current Session Context

**Source:** Telegram (DM with Pak Bos)
**User:** Pak Bos
**Connected Platforms:** local (files on this machine), telegram: Connected ✓
**Session ID:** 01926b3e-...
```

### 4.6 Session Reset Policy

Reset policy is evaluated by the handler layer using SPEC-001 `Session.updated` as the last-activity timestamp. The gateway itself does not store session state.

```go
// SessionResetPolicy governs automatic session expiry.
type SessionResetPolicy struct {
    Mode        string // "none", "idle", "daily", "both"
    IdleMinutes int    // used when Mode is "idle" or "both"
    AtHour      int    // 0-23; used when Mode is "daily" or "both"
}

// ShouldReset returns a reset reason ("idle", "daily") if the session
// should be reset, or "" if it is still valid.
func ShouldReset(policy SessionResetPolicy, lastActivity time.Time) string { ... }
```

### 4.7 Callback Data Schema

Consistent with v1.0–v1.2:

- `approve:<tool_call_id>` — proceed with tool execution
- `deny:<tool_call_id>` — abort execution, return user-friendly error

### 4.8 Resolution Flow

```
Inbound platform event (e.g. Telegram update)
  │
  ▼
GatewayContext          ← built by gateway/telegram
  │
  ▼
SessionSource           ← enriched from GatewayContext
BuildSessionKey()       ← deterministic key
  │
  ▼
Handler Layer
  ├── resolve/create Account   (SPEC-001: platform + external_id)
  ├── resolve/create Session   (SPEC-001: account_id + session_key)
  ├── evaluate ShouldReset()   (SPEC-001: Session.updated)
  └── build SessionContext     (for system prompt injection)
  │
  ▼
ExecuteContext (SPEC-002)
  ├── Session     *ent.Session
  ├── History     []ent.Message
  ├── MemContext  string        ← prefetched memory (SPEC-001)
  └── WorkingDir  string        ← from Profile (SPEC-002)
  │
  ▼
Agent Processing (SPEC-002)
  │
  ▼
Gateway.SendMessage() / SendApproval()
```

## 5. Package Structure

```
internal/
  gateway/
    gateway.go          ← Gateway interface, ApprovalRequest, MessageHandler,
                           CallbackHandler
    session.go          ← GatewayContext, SessionSource, SessionContext,
                           SessionResetPolicy, BuildSessionKey(),
                           BuildSessionContextPrompt(), ShouldReset(),
                           IsSharedMultiUserSession()
    telegram/
      bot.go            ← TelegramGateway: implements Gateway interface
      formatter.go      ← MarkdownV2 escaper utility
      poller.go         ← Long-polling loop and update dispatch
```

## 6. Configuration

All gateway configuration is stored in the existing Viper config (`~/.sbctl/config.json`). A gateway is activated only when its credentials are present.

```json
{
  "gateway": {
    "session": {
      "group_sessions_per_user": true,
      "thread_sessions_per_user": false,
      "reset_policy": {
        "mode": "both",
        "idle_minutes": 30,
        "at_hour": 3
      }
    },
    "telegram": {
      "token": "bot<token>",
      "allowed_user_ids": [123456789]
    }
  }
}
```

JSON Schema additions (`docs/config-schema.json`):

```json
"gateway": {
  "type": "object",
  "properties": {
    "session": {
      "type": "object",
      "properties": {
        "group_sessions_per_user": { "type": "boolean" },
        "thread_sessions_per_user": { "type": "boolean" },
        "reset_policy": {
          "type": "object",
          "properties": {
            "mode": { "type": "string", "enum": ["none","idle","daily","both"] },
            "idle_minutes": { "type": "integer" },
            "at_hour": { "type": "integer", "minimum": 0, "maximum": 23 }
          }
        }
      }
    },
    "telegram": {
      "type": "object",
      "properties": {
        "token": { "type": "string" },
        "allowed_user_ids": {
          "type": "array",
          "items": { "type": "integer" }
        }
      },
      "required": ["token", "allowed_user_ids"]
    }
  }
}
```

## 7. Daemon Integration

Gateways are registered as workers in the daemon provider, consistent with the existing `GetGitSync()` pattern:

```go
// internal/config/gateway.go

func (c *Config) GetGateways() []daemon.Worker {
    var workers []daemon.Worker
    if c.Gateway.Telegram.Token != "" {
        workers = append(workers, telegram.New(&c.Gateway.Telegram, &c.Gateway.Session))
    }
    // future: WhatsApp, CLI
    return workers
}
```

```go
// internal/config/daemon.go

provider := func() []daemon.Worker {
    cfg := GetConfig()
    workers := []daemon.Worker{
        cfg.GetGitSync(),
    }
    workers = append(workers, cfg.GetGateways()...)
    return workers
}
```

Each gateway worker has an isolated lifecycle:
- Telegram crash does not affect other workers
- Config reload via `OnReload` restarts affected gateways only
- `Stop()` called on daemon context cancellation for clean shutdown

## 8. Telegram MVP Implementation

### 8.1 TelegramGateway

`TelegramGateway` in `internal/gateway/telegram/bot.go` implements the full `Gateway` interface using long-polling via `github.com/go-telegram/bot`.

```go
type TelegramGateway struct {
    config          *TelegramConfig
    sessionConfig   *SessionConfig
    bot             *bot.Bot
    messageHandler  gateway.MessageHandler
    callbackHandler gateway.CallbackHandler
}

func (t *TelegramGateway) Name() string { return "gateway.telegram" }
func (t *TelegramGateway) Start(ctx context.Context) error { ... }
func (t *TelegramGateway) Stop(ctx context.Context) error { ... }
func (t *TelegramGateway) SendMessage(...) error { ... }
func (t *TelegramGateway) SendApproval(...) error { ... }
func (t *TelegramGateway) SetTyping(...) error { ... }
func (t *TelegramGateway) OnMessage(handler gateway.MessageHandler) { ... }
func (t *TelegramGateway) OnCallback(handler gateway.CallbackHandler) { ... }
```

### 8.2 Bot Commands

All commands from v1.0–v1.2 are retained unchanged:

#### `/start` — Onboarding

1. Extract Telegram User ID.
2. Check allowlist — if not authorized, ignore silently (no response, no DB entry).
3. If no `Account` exists, provision `User` and `Account` via SPEC-001.
4. Respond with welcome message.

```
👋 Welcome to sbctl!
Your account has been set up.

Use /new to start a session and begin interacting with your Second Brain.
```

If account already exists:
```
👋 Welcome back!
Use /new to start a new session.
```

#### `/new` — New Session

1. Create a new `Session` record linked to the current `Account` (SPEC-001).
2. Set as active session.
3. Respond with confirmation.

```
✅ New session started.
Session ID: 01926b3e-...
```

#### `/profile` Commands

| Command | Description |
|---|---|
| `/profile` | List all profiles for current user |
| `/profile <name>` | Switch active profile for current session |
| `/profile add <name> <path>` | Create a new profile |
| `/profile remove <name>` | Delete a profile (default profile cannot be removed) |

See v1.1 for full command behavior specification.

### 8.3 HITL Approval UI

When the agent triggers a dangerous tool (SPEC-002):

Message:
```
⚠️ Agent requests execution:
`rm -rf ./tmp`

Authorize?
```

Buttons: `[ ✅ Approve ]` `[ ❌ Deny ]`

Callback data: `approve:<tool_call_id>` / `deny:<tool_call_id>`

### 8.4 Message Segmentation

Responses exceeding 4096 characters MUST be split into multiple messages, indexed as `(1/n)`.

### 8.5 MarkdownV2 Escaping

All outgoing text passes through `formatter.go`:

```
Input:  Hello *World*! [v1.0]
Output: Hello \*World\*\! \[v1\.0\]
```

### 8.6 No Active Session

If a user sends a message without an active session:
```
No active session. Use /new to start one.
```

## 9. Acceptance Criteria

**Gateway Interface (new in v1.3):**
- **AC-001-v13**: Given a configured Telegram token, when the daemon starts, a `TelegramGateway` worker MUST be started automatically.
- **AC-002-v13**: Given no Telegram token in config, when the daemon starts, NO Telegram worker is started and NO error is returned.
- **AC-003-v13**: Given two configured gateways (future), when one crashes, the other MUST continue running unaffected.
- **AC-004-v13**: Given an inbound Telegram message, a `GatewayContext` MUST be constructed and passed to the `MessageHandler`.
- **AC-005-v13**: Given a `SessionSource`, `BuildSessionKey()` MUST return a deterministic, consistent key across restarts.
- **AC-006-v13**: Given a session idle beyond `idle_minutes`, `ShouldReset()` MUST return `"idle"`.
- **AC-007-v13**: Given a session last active before the configured `at_hour` today, `ShouldReset()` MUST return `"daily"`.
- **AC-008-v13**: Given `reset_policy.mode = "none"`, `ShouldReset()` MUST always return `""`.

**Retained from v1.0–v1.2:**
- **AC-001**: Given a message from an unauthorized user, the bot MUST ignore the input and refrain from creating database entries.
- **AC-002**: Given a response exceeding 4096 characters, the bot MUST segment the message into multiple parts, indexed as `(1/n)`.
- **AC-003**: Given a dangerous tool execution request, the bot MUST present a confirmation message with explicit `Approve` and `Deny` buttons.
- **AC-004**: Given `/profile sbctl`, the bot MUST confirm the switch and display the resolved `working_dir`.
- **AC-005**: Given `/profile remove vault` on the default profile, the bot MUST reject with a descriptive error.
- **AC-006**: Given `/profile add sbctl ~/code/sbctl` with a duplicate name, the bot MUST reject with a descriptive error.
- **AC-007**: Given a `/start` command from an authorized new user, the bot MUST provision `User` and `Account` entities and respond with a welcome message.
- **AC-008**: Given a `/new` command, the bot MUST create a new `Session` and set it as active for the current `Account`.
- **AC-009**: Given any non-command message sent without an active session, the bot MUST respond with a prompt to use `/new`.

## 10. Test Automation Strategy

- **Unit Tests**: `BuildSessionKey()` with all chat types, multi-user permutations, and edge cases.
- **Unit Tests**: `ShouldReset()` for all policy modes, boundary conditions (exact idle boundary, daily cutover).
- **Unit Tests**: `MarkdownV2Escaper` with all special character sets.
- **Integration Tests**: Mock Telegram API server to validate polling, allowlist enforcement, and message dispatch.
- **E2E Tests**: Full lifecycle — Telegram Message → Handler → SPEC-001 → Agent → Telegram Response.
- **Concurrency**: Run with `-race` flag; `BuildSessionKey()` is a pure function (no locks needed).

## 11. Rationale & Context

- **Gateway extends Worker**: Natural fit — gateways are long-running background processes. Daemon manages all workers uniformly; no special-casing needed.
- **Isolated per-platform workers**: Independent failure domains. A Telegram outage does not affect git sync or future WhatsApp gateway.
- **Config-driven activation**: Zero config = zero workers started. No error, no panic. Consistent with `GetGitSync()` pattern.
- **SessionSource vs GatewayContext**: `GatewayContext` is raw transport metadata (what the platform gives us). `SessionSource` is enriched metadata used for business logic (session keys, system prompt). Separation prevents leaking transport concerns into agent logic.
- **No gateway session store**: SPEC-001 is the single source of truth. Avoiding a second session store eliminates an entire class of consistency bugs.
- **`ShouldReset()` in handler layer**: Reset evaluation needs SPEC-001 `Session.updated` — a DB value. Gateway has no DB access. Handler layer is the correct place.
- **Borrowed from Hermes**: `SessionSource`, `BuildSessionKey()`, `ShouldReset()`, `BuildSessionContextPrompt()` are proven patterns from production. Adapting them to Go avoids re-inventing solved problems.

## 12. Dependencies & External Integrations

### External Systems
- **EXT-001 (Telegram Bot API)**: Message transport for MVP.

### Third-Party Libraries
- **PLT-001**: `github.com/go-telegram/bot` — pure Go Telegram Bot API client (actively maintained).

### Internal Dependencies
- **INF-001 (Daemon Worker)**: `internal/daemon.Worker` — gateway lifecycle management.
- **INF-002 (Persistence Layer)**: SPEC-001 — `Account`, `Session`, `Profile` entities.
- **INF-003 (Agent Manager)**: SPEC-002 — `ExecuteContext`, tool execution, HITL flow.

### Configuration
- **CFG-001**: `internal/config` — Viper-based config, existing system.

## 13. Change Log

| Version | Date | Summary |
|---|---|---|
| 1.0 | 2026-05-26 | Initial release — Telegram Bot Integration |
| 1.1 | 2026-05-26 | AMENDMENT-001: Profile commands |
| 1.2 | 2026-05-26 | AMENDMENT-002: `/start` onboarding, `/new` session lifecycle, `go-telegram/bot`, UX-001 typing indicator |
| 1.3 | 2026-05-26 | AMENDMENT-003: Gateway abstraction — `Gateway` interface extends `daemon.Worker`; `GatewayContext`; `SessionSource`; `BuildSessionKey()`; `SessionResetPolicy`; `ShouldReset()`; `BuildSessionContextPrompt()`; config-driven multi-platform worker activation; Telegram becomes MVP implementation |

## 14. Related Specifications / Further Reading

- [SPEC-001: Persistent Memory Management](./SPEC-001-memory-management.md)
- [SPEC-002: Agent Provider & LLM Orchestration](./SPEC-002-agent-provider.md)
- [Telegram Bot API Documentation](https://core.telegram.org/bots/api)
- [go-telegram/bot Documentation](https://github.com/go-telegram/bot)
