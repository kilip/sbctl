---
title: SPEC-003: Telegram Bot Integration & UX
version: 1.0
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
- **UX-001 (Feedback Indicators)**: The bot should emit a "typing" status while the Agent is processing a response.

## 4. Interfaces & Data Contracts

### 4.1 Account Handshake Logic
Upon receiving an update:
1. Extract the `external_id` (Telegram User ID).
2. Query the `Account` table defined in [SPEC-001](./SPEC-001-memory-management.md).
3. If no record exists and the ID is on the allowlist, provision new `User` and `Account` entities.
4. Associate the interaction with an active `Session` or initialize a new one.

### 4.2 Callback Data Schema
- `approve:<tool_call_id>`: Signals the Agent ([SPEC-002](./SPEC-002-agent-provider.md)) to proceed with execution.
- `deny:<tool_call_id>`: Signals the Agent ([SPEC-002](./SPEC-002-agent-provider.md)) to abort execution and return a user-friendly error.

## 5. Acceptance Criteria

- **AC-001**: Given a message from an unauthorized user, the bot must ignore the input and refrain from creating database entries.
- **AC-002**: Given a response exceeding 4096 characters, the bot must segment the message into multiple parts, indexed as `(1/n)`.
- **AC-003**: Given a "Dangerous Tool" execution request (e.g., `terminal rm -rf`), the bot must present a confirmation message with explicit `Approve` and `Deny` buttons.

## 6. Test Automation Strategy

- **Mocking**: Utilize a mock Telegram API server to validate polling logic and message dispatching.
- **E2E Testing**: Verify the full lifecycle: Telegram Message -> Persistence Layer -> Agent Processing -> Telegram Response.
- **Unit Testing**: Implement exhaustive tests for the `MarkdownV2Escaper` using edge-case characters and nested formatting.

## 7. Rationale & Context

- **MarkdownV2 Adoption**: Provides professional-grade formatting (code blocks, monospacing) essential for technical and engineering outputs.
- **Inline Keyboards for HITL**: Offers the most efficient and secure method for mobile users to authorize high-risk actions without manual command entry.

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

### Compliance Dependencies
- **COM-001**: None - No specific regulatory compliance requirements identified.

## 9. Examples & Edge Cases

### Dangerous Tool UI
Message: *"Agent requests execution: `rm -rf ./tmp`. Authorize?"*
Buttons: `[ ✅ Approve ]` `[ ❌ Deny ]`

### MarkdownV2 Escaping Example
Input: `Hello *World*! [v1.0]`
Output: `Hello \*World\*\! \[v1.0\]`

## 10. Validation Criteria

- **VAL-001**: Verification of per-session concurrency to prevent polling loop blockages.
- **VAL-002**: Accuracy of MarkdownV2 escaping for all special character sets.
- **VAL-003**: Confirmation of correct session association for multi-user environments.
- **VAL-004**: Verification of callback signal transmission to the Agent Manager.

## 11. Related Specifications / Further Reading

- [SPEC-001: Persistent Memory Management](./SPEC-001-memory-management.md)
- [SPEC-002: Agent Provider & LLM Orchestration](./SPEC-002-agent-provider.md)
- [Telegram Bot API Documentation](https://core.telegram.org/bots/api)
