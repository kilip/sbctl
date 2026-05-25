---
title: SPEC-002: Agent Provider & LLM Orchestration
version: 1.0
date_created: 2026-05-26
last_updated: 2026-05-26
owner: Pak Bos
tags: [architecture, logic, provider, llm, go]
---

# Introduction

This specification defines the architecture of the `internal/agent` module, which serves as the core orchestration engine for `sbctl`. This component manages the **Research -> Strategy -> Execution** workflow, abstracts interactions with multiple Large Language Model (LLM) providers (e.g., Anthropic, OpenAI), and governs the lifecycle of tool execution.

## 1. Purpose & Scope

The primary objective is to provide a unified interface for consistent, secure, and context-aware LLM interactions. The scope includes dynamic system prompt assembly, memory prefetching mechanisms (aligned with [SPEC-001](./SPEC-001-memory-management.md)), and Human-in-the-Loop (HITL) approval flows for high-risk operations.

## 2. Definitions

- **Provider**: An abstraction layer for various LLM APIs.
- **Prefetch**: The process of retrieving relevant contextual data from the persistence layer prior to LLM invocation.
- **ExecuteContext**: A stateful object carrying session history, metadata, and execution state throughout an interaction cycle.
- **Dangerous Tool**: A tool with destructive potential or broad system access (e.g., `terminal` execution, `write_file`).
- **HITL (Human-in-the-Loop)**: A workflow requiring explicit human confirmation before executing sensitive actions.

## 3. Requirements, Constraints & Guidelines

- **REQ-001 (Provider Agnosticism)**: The provider interface must remain decoupled from specific transport layers (e.g., Telegram, CLI).
- **REQ-002 (Contextual Injection)**: Every LLM request must be enriched with relevant long-term memory fragments.
- **CON-001 (XML-Based Isolation)**: Prefetched memory context must be encapsulated within `<memory-context>` tags to mitigate prompt injection and hallucination.
- **SEC-001 (Mandatory Approval)**: Execution of "Dangerous Tools" REQUIRES explicit user authorization via the communication interface ([SPEC-003](./SPEC-003-telegram-bot.md)).
- **GUD-001 (Session Identity)**: Utilize UUID v7 from `ent.Session` ([SPEC-001](./SPEC-001-memory-management.md)) as the primary identifier for execution cycles.

## 4. Interfaces & Data Contracts

### 4.1 Provider Interface
```go
type Provider interface {
    // Initialize configures the adapter with API keys and model parameters.
    Initialize(apiKey string, config map[string]interface{}) error

    // ExecuteContext processes multi-turn interactions with full state.
    // memContext is an XML-formatted string containing prefetched facts from SPEC-001.
    ExecuteContext(ctx context.Context, session *ent.Session, history []ent.Message, memContext string) (*IntentResult, error)
}
```

### 4.2 IntentResult
The data structure returned by the LLM, containing either a textual response or a set of tool-call requests.

## 5. Acceptance Criteria

- **AC-001**: Given a user query regarding "Project X", when the system performs prefetch, relevant facts from the `Memory` table must be retrieved and injected into the prompt.
- **AC-002**: Given a command to delete files (e.g., `rm -rf`), when the agent attempts to invoke the `terminal` tool, the system must intercept execution and set the message status to `Awaiting Approval`.
- **AC-003**: Given an LLM response requesting tool execution, the system must execute non-dangerous tools locally and return the output to the LLM automatically.

## 6. Test Automation Strategy

- **Mocking**: Implement mock providers to simulate LLM responses and tool-call triggers without external API costs.
- **Golden Files**: Utilize golden files to validate the structural integrity of dynamically assembled system prompts.
- **Concurrency**: Verify tool execution safety using the Go race detector during parallel execution tests.

## 7. Rationale & Context

- **XML Tagging**: Modern LLMs (Claude 3.5, GPT-4o) exhibit superior instruction following when data boundaries are explicitly defined via XML-like tags.
- **Prefetch Strategy**: To optimize context window usage and reduce latency/cost, only relevant memory fragments are injected rather than the entire database.

## 8. Dependencies & External Integrations

### External Systems
- **EXT-001 (LLM APIs)**: External processing engines (Anthropic, OpenAI) for language understanding and generation.

### Third-Party Services
- **SVC-001**: None - All logic is processed locally or via direct API calls to EXT-001.

### Infrastructure Dependencies
- **INF-001 (Persistence Layer)**: Aligned with [SPEC-001](./SPEC-001-memory-management.md) for session history and semantic memory retrieval.

### Data Dependencies
- **DAT-001**: None - Context is dynamically assembled from memory and user input.

### Technology Platform Dependencies
- **PLT-001**: Go Runtime - Core execution environment.

### Compliance Dependencies
- **COM-001**: None - No specific regulatory compliance requirements identified.

## 9. Examples & Edge Cases

### Memory Injection Format
```xml
<memory-context>
[SYSTEM NOTE: Long-term facts retrieved for this session]
- User prefers formal technical English for documentation.
- Project status: sbctl implementation phase.
</memory-context>
```

## 10. Validation Criteria

- **VAL-001**: Tool invocation metadata accuracy in the `Message` table.
- **VAL-002**: Verification of state preservation during multi-turn interactions.
- **VAL-003**: Confirmation of pause/resume logic during HITL approval flows.
- **VAL-004**: System prompt assembly integrity across different LLM providers.

## 11. Related Specifications / Further Reading

- [SPEC-001: Persistent Memory Management](./SPEC-001-memory-management.md)
- [SPEC-003: Telegram Bot Integration](./SPEC-003-telegram-bot.md)
