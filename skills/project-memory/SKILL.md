---
name: project-memory
description: >
  Persistent memory system for AI agents working on long-running projects. Use this skill whenever an agent needs to remember decisions, actions, user preferences, errors, or context across sessions. Trigger this skill when: starting a new project session, making architectural decisions, completing significant actions, recording user preferences, encountering and resolving errors, or wrapping up a work session. Also trigger for weekly summaries and monthly archiving. If the user says "remember this", "log that", "what did we do last time", or similar — use this skill. Even if the user doesn't explicitly ask, proactively log important decisions and actions during any multi-step task.
---

# Project Memory Skill

Gives AI agents persistent memory across sessions using structured logs stored in `.agents/project/`.

## Directory Structure

```
cwd/
└── .agents/
    └── project/
        ├── logs/
        │   ├── YYYY-MM-DD.md          # Daily logs
        │   └── YYYY-Www.md            # Weekly summaries (e.g. 2025-W21.md)
        ├── archive/
        │   └── YYYY/
        │       └── MM.md              # Monthly archives (e.g. 2025/05.md)
        └── context.md                 # Persistent project context (always-loaded)
```

## Log Entry Format

All entries in daily logs follow this format:

```
- [YYYY-MM-DD HH:MM:SS] [TYPE] Description
```

Use the **current local datetime from system clock** for all timestamps. Format: `YYYY-MM-DD HH:MM:SS`.

**Entry Types:**
| Type | When to use |
|---|---|
| `DECISION` | Architectural, design, or strategic choices |
| `ACTION` | Significant file changes, commands run, installs |
| `PREFERENCE` | User preferences, style choices, workflow habits |
| `ERROR` | Bugs encountered, failed attempts |
| `FIX` | How an error was resolved |
| `NOTE` | Context, reminders, observations |
| `SESSION_START` | Beginning of a work session |
| `SESSION_END` | End of session with summary |
| `MILESTONE` | Task or feature completed |

---

## Workflow

### 1. Session Start

When starting a new session:

1. Check if `.agents/project/` exists
2. If it exists, read `context.md` and recent daily logs (last few sessions worth)
3. If it doesn't exist, initialize it (see **Auto-detect project init** below)

Then log session start:
```
- [YYYY-MM-DD HH:MM:SS] [SESSION_START] Resumed project. Loaded context: <brief summary of what you learned>
```

**ALWAYS** verify `.agents/project` is excluded from version control (e.g. listed in `.gitignore`). If not, add it.

### 2. During Session — What to Log

**Always log:**
- Any decision with rationale
- Files created/deleted/significantly modified
- Packages installed or removed
- User preferences stated explicitly
- Errors and their fixes
- Completed milestones

**Don't log:**
- Trivial reads/lookups
- Intermediate thoughts
- Things the user can see in the conversation

**How to write a good log entry:**
```
# Good — specific, with rationale
- [2025-05-24 10:15:42] [DECISION] Chose PostgreSQL over SQLite — project needs concurrent writes from multiple workers

# Bad — vague
- [2025-05-24 10:15:42] [DECISION] Chose a database
```

### 3. Appending to Daily Log

To safely append a log entry:
1. Read the existing content of `.agents/project/logs/YYYY-MM-DD.md`
2. If the file doesn't exist, create it with a header: `# Log YYYY-MM-DD`
3. Append the new entry at the bottom
4. Write the updated content back to the file

**Do not use shell append commands** (e.g. `echo >>`). Use your file read/write tools directly.

### 4. Updating context.md

`context.md` is loaded every session. Keep it lean and current — it should answer: *"What is this project and where are we right now?"*

Update it when:
- Project goals change
- Stack or architecture changes significantly
- A major milestone is reached
- User preferences accumulate

**Template:**
```markdown
# Project Context

## What This Is
<1-2 sentences>

## Current State
<What's working, what's in progress>

## Stack
<Key tech choices>

## User Preferences
<Bullet list of stated preferences>

## Active TODOs
<Short list — not a full backlog>

## Last Updated
<date>
```

### 5. Session End

Before ending, log a session summary:
```
- [YYYY-MM-DD HH:MM:SS] [SESSION_END] Completed: X, Y. Pending: Z. Next: <suggested next step>
```

---

## Weekly Summary

Generate at the start of each new week, or when a full week of logs has accumulated. Read that week's daily logs and distill what matters.

**File:** `.agents/project/logs/YYYY-Www.md` (e.g. `2025-W21.md`)

Read each day's log file for the past 7 days and summarize them.

**Weekly summary format:**
```markdown
# Week YYYY-Www (Mon DD MMM — Sun DD MMM)

## Highlights
- <Major milestones or decisions>

## Decisions Made
- <Key decisions with brief rationale>

## Problems & Fixes
- <Notable errors and how they were resolved>

## Preferences Captured
- <User preferences noted this week>

## State at End of Week
<What's working, what's in progress>
```

Keep it to ~30 lines max. Skip days with nothing significant.

---

## Monthly Archive

Run at end of each month, or at the first session of a new month.

**File:** `.agents/project/archive/YYYY/MM.md` (e.g. `archive/2025/05.md`)

Create the archive directory if it doesn't exist, then write the file.

**Monthly archive format:**
```markdown
# Archive — Month YYYY

## Summary
<2-3 sentences of what happened this month>

## Key Decisions
- <Most important decisions>

## Milestones
- <Completed features/tasks>

## Unresolved Issues
- <Known bugs or blockers left open>

## Preferences & Conventions Established
- <Stable user preferences and project conventions>
```

After archiving, prune daily logs older than the archived month to keep `.agents/project/logs/` clean.

---

## Additional Important Behaviors

### Auto-detect project init

If `.agents/project/` doesn't exist when a new project starts:

1. Create the directory structure:
   - `.agents/project/logs/`
   - `.agents/project/archive/`
2. Create `.agents/project/context.md` using the template above, with "Freshly initialized" as the current state
3. Create today's log file and add a `SESSION_START` entry: `Project memory initialized.`
4. Ensure `.agents/project` is excluded from version control

Then ask the user: *"Project memory initialized. What's this project about?"* and fill in `context.md`.

### Error pattern tracking

When logging an `ERROR` + `FIX` pair, also scan recent log files for the same error keyword. If it's a repeat, note it: `[ERROR] (repeat #2) ...`

Repeated errors should trigger a `DECISION` entry about a permanent fix.

### Preference consolidation

Periodically (every ~10 sessions or when `context.md` gets long), deduplicate preferences. Remove outdated ones, merge similar ones.

---

## Quick Reference

| Task | Action |
|---|---|
| Start session | Read `context.md` + recent daily logs |
| Log an entry | Append to `logs/YYYY-MM-DD.md` using system clock for timestamp |
| Big decision | Log `DECISION` + update `context.md` if it changes stack/goals |
| End session | Log `SESSION_END` with summary |
| New week | Generate weekly summary |
| New month | Generate monthly archive, prune old daily logs |
| Repeat error | Mark as repeat, consider permanent fix |