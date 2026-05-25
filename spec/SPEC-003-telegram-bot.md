---
title: SPEC-003: Telegram Bot Integration & UX
version: 1.2
date_created: 2026-05-26
last_updated: 2026-05-26
owner: Pak Bos
tags: [interface, telegram, bot, hitl, go]
---

# Introduction

This specification defines the implementation of the `internal/telegram` module, serving as the primary mobile interface for `sbctl`. The bot functions as a secure communication bridge, a control panel for Human-in-the-Loop (HITL) action approvals, and a session management gateway.

## 1. Purpose & Scope

The objective is to provide a secure, responsive, and intuitive interface for the administrator. Scope includes long-polling management, allowlist-based security, MarkdownV2 message formatting, and callback handling for tool execution authorization.

## 2. Definitions

- **Long-polling**: A technique for retrieving updates from the Telegram server at regular intervals.
- **Allowlist**: A predefined list of authorized Telegram User IDs permitted to interact with the bot.
- **MarkdownV2**: Telegram's rich text format requiring strict character escaping.
- **Inline Keyboard**: Interactive buttons attached to messages for rapid user feedback (e.g., Approve/Deny).
- **CallbackQuery**: An incoming signal triggered when an Inline Keyboard button is pressed.

## 3. Requirements, Constraints & Guidelines

- **REQ-001 (Identity Verification)**: The system MUST verify user identity against the `allowed_user_ids` configuration before processing any incoming messages.
- **REQ-002 (HITL Workflow Support)**: The bot must facilitate the tool approval flow using Inline Keyboards as defined in [SPEC-002](./SPEC-002-agent-provider.md).
- **CON-001 (Strict Escaping)**: All outgoing text MUST pass through a `MarkdownV2Escaper` utility to ensure compatibility with the Telegram parsing engine.
- **SEC-001 (Credential Protection)**: The Telegram Bot Token must never be exposed in logs, CLI output, or error messages.
- **UX-001 (Feedback Indicators)**: The bot should emit a "typing" status while the Agent is processing a response, and stop once the response is sent or an error occurs.

## 4. Interfaces & Data Contracts

### 4.1 Account Handshake Logic
Upon receiving an update:
1. Extract the `external_id` (Telegram User ID).
2. Query the `Account` table defined in [SPEC-001](./SPEC-001-memory-management.md).
3. If no record exists and the ID is on the allowlist, provision new `User` and `Account` entities.
4. Associate the interaction with the user's active `Session`, or prompt the user to create one via `/new`.

### 4.2 Session Lifecycle

Sessions are explicitly managed by the user via commands:

| Command | Behavior |
|---|---|
| `/new` | Create a new `Session` record; set it as the active session for the current `Account`. Respond with a confirmation and the new session ID. |

> **Note**: There is no automatic session creation on message receipt. If a user sends a message without an active session, the bot must respond: *"No active session. Use /new to start one."*

### 4.3 Bot Commands

#### `/start` — Onboarding

Triggered when the user first interacts with the bot (or re-opens it).

1. Extract `external_id` (Telegram User ID).
2. Check allowlist — if not authorized, ignore silently.
3. If no `Account` exists for this `external_id`, provision `User` and `Account` entities (as per Section 4.1).
4. Respond with a welcome message and available commands.

Example response:
```
👋 Welcome to sbctl!
Your account has been set up.

Use /new to start a session and begin interacting with your Second Brain.
```

If the account already exists:
```
👋 Welcome back!
Use /new to start a new session.
```

#### `/new` — New Session

1. Create a new `Session` record linked to the current `Account`.
2. Set it as the active session.
3. Respond with confirmation.

Example response:
```
✅ New session started.
Session ID: 01926b3e-...
```

### 4.4 Profile Commands *(AMENDMENT-001)*

| Command | Description |
|---|---|
| `/profile` | List all profiles for the current user |
| `/profile <name>` | Switch the active profile for the current session |
| `/profile add <name> <path>` | Create a new profile |
| `/profile remove <name>` | Delete a profile (default profile cannot be removed) |

#### `/profile` — List

Responds with a formatted list of all user profiles. The active session profile is marked with `●`.

Example response:
```
Your profiles:
● vault   ~/brain       (default)
  sbctl   ~/code/sbctl
```

#### `/profile <name>` — Switch

1. Look up profile by `name` for the current user.
2. Update `session.profile_id` to the found profile's ID.
3. Respond with a confirmation message.

Example response:
```
Switched to profile: sbctl
Working directory: ~/code/sbctl
```

If the profile name is not found:
```
Profile "foo" not found. Use /profile to see available profiles.
```

#### `/profile add <name> <path>` — Create

1. Validate that `name` is unique for the user and `path` is a non-empty string.
2. Create a new `Profile` record with `is_default = false`.
3. Respond with confirmation.

Example response:
```
Profile "sbctl" created (~/code/sbctl).
Use /profile sbctl to switch.
```

#### `/profile remove <name>` — Delete

1. Reject if the profile is the user's default (`is_default = true`):
```
Cannot remove the default profile. Set another profile as default first.
```
2. Otherwise soft-delete the `Profile` record.
3. If the current session's `profile_id` matches the removed profile, reset `session.profile_id = NULL` (will fall back to default).

### 4.5 Callback Data Schema
- `approve:<tool_call_id>`: Signals the Agent ([SPEC-002](./SPEC-002-agent-provider.md)) to proceed with execution.
- `deny:<tool_call_id>`: Signals the Agent ([SPEC-002](./SPEC-002-agent-provider.md)) to abort execution and return a user-friendly error.

## 5. Acceptance Criteria

- **AC-001**: Given a message from an unauthorized user, the bot must ignore the input and refrain from creating database entries.
- **AC-002**: Given a response exceeding 4096 characters, the bot must segment the message into multiple parts, indexed as `(1/n)`.
- **AC-003**: Given a "Dangerous Tool" execution request (e.g., `terminal rm -rf`), the bot must present a confirmation message with explicit `Approve` and `Deny` buttons.
- **AC-004** *(AMENDMENT-001 addendum)*: Given `/profile sbctl`, the bot must confirm the switch and display the resolved `working_dir`.
- **AC-005** *(AMENDMENT-001 addendum)*: Given `/profile remove vault` on the default profile, the bot must reject with a descriptive error.
- **AC-006** *(AMENDMENT-001 addendum)*: Given `/profile add sbctl ~/code/sbctl` with a duplicate name, the bot must reject with a descriptive error.
- **AC-007** *(AMENDMENT-002)*: Given a `/start` command from an authorized new user, the bot must provision `User` and `Account` entities and respond with a welcome message.
- **AC-008** *(AMENDMENT-002)*: Given a `/new` command, the bot must create a new `Session` and set it as active for the current `Account`.
- **AC-009** *(AMENDMENT-002)*: Given any non-command message sent without an active session, the bot must respond with a prompt to use `/new`.

## 6. Test Automation Strategy

- **Mocking**: Utilize a mock Telegram API server to validate polling logic and message dispatching.
- **E2E Testing**: Verify the full lifecycle: Telegram Message -> Persistence Layer -> Agent Processing -> Telegram Response.
- **Unit Testing**: Implement exhaustive tests for the `MarkdownV2Escaper` using edge-case characters and nested formatting.

## 7. Rationale & Context

- **MarkdownV2 Adoption**: Provides professional-grade formatting (code blocks, monospacing) essential for technical and engineering outputs.
- **Inline Keyboards for HITL**: Offers the most efficient and secure method for mobile users to authorize high-risk actions without manual command entry.
- **Explicit Session Management via `/new`**: Avoids ambiguity around automatic session creation. Users have full control over session boundaries, which maps directly to context window management for the Agent.
- **`/start` for Onboarding**: Follows Telegram bot conventions; serves as the canonical entry point for account provisioning without requiring a separate setup step.

## 8. Dependencies & External Integrations

### External Systems
- **EXT-001 (Telegram Bot API)**: The external provider for message transport.

### Third-Party Services
- **SVC-001**: None - Logic is handled by internal modules.

### Infrastructure Dependencies
- **INF-001 (Agent Manager)**: Defined in [SPEC-002](./SPEC-002-agent-provider.md) for core logic processing.
- **INF-002 (Persistence Layer)**: Defined in [SPEC-001](./SPEC-001-memory-management.md) for account verification and session persistence.

### Data Dependencies
- **DAT-001**: None - Data is sourced from the Telegram API updates.

### Technology Platform Dependencies
- **PLT-001**: Go Runtime - Core execution environment.
- **PLT-002**: `github.com/go-telegram/bot` - Telegram Bot API client library (pure Go, actively maintained).

### Compliance Dependencies
- **COM-001**: None - No specific regulatory compliance requirements identified.

## 9. Examples & Edge Cases

### Dangerous Tool UI
Message: *"Agent requests execution: `rm -rf ./tmp`. Authorize?"*
Buttons: `[ ✅ Approve ]` `[ ❌ Deny ]`

### MarkdownV2 Escaping Example
Input: `Hello *World*! [v1.0]`
Output: `Hello \*World\*\! \[v1.0\]`

### No Active Session
User sends: `"What is the status of Project X?"`
Bot responds: `"No active session. Use /new to start one."`

## 10. Validation Criteria

- **VAL-001**: Verification of per-session concurrency to prevent polling loop blockages.
- **VAL-002**: Accuracy of MarkdownV2 escaping for all special character sets.
- **VAL-003**: Confirmation of correct session association for multi-user environments.
- **VAL-004**: Verification of callback signal transmission to the Agent Manager.
- **VAL-005** *(AMENDMENT-002)*: `/start` correctly provisions new users and handles existing users gracefully.
- **VAL-006** *(AMENDMENT-002)*: `/new` always creates a fresh `Session` and updates the active session reference on `Account`.

## 11. Change Log

| Version | Date | Summary |
|---|---|---|
| 1.0 | 2026-05-26 | Initial release |
| 1.1 | 2026-05-26 | AMENDMENT-001: Profile commands |
| 1.2 | 2026-05-26 | AMENDMENT-002: `/start` onboarding, `/new` session lifecycle, `go-telegram/bot` library, UX-001 typing indicator stop condition |

## 12. Related Specifications / Further Reading

- [SPEC-001: Persistent Memory Management](./SPEC-001-memory-management.md)
- [SPEC-002: Agent Provider & LLM Orchestration](./SPEC-002-agent-provider.md)
- [Telegram Bot API Documentation](https://core.telegram.org/bots/api)
- [go-telegram/bot Documentation](https://github.com/go-telegram/bot)