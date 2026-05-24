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
- [YYYY-MM-DD HH:MM] [TYPE] Description
```

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

## Workflow

### 1. Session Start

When starting a new session, always:

```bash
# Check if project memory exists
ls cwd/.agents/project/

# Load context
cat .agents/project/context.md

# Load recent logs (last 3 days)
cat .agents/project/logs/YYYY-MM-DD.md  # today
cat .agents/project/logs/YYYY-MM-DD.md  # yesterday
cat .agents/project/logs/YYYY-MM-DD.md  # day before
```

Then log session start:
```
- [2025-05-24 09:00] [SESSION_START] Resumed project. Loaded context: <brief summary of what you learned>
```

- **ALWAYS** verify `.agents/project` is listed in `.gitignore`, if not listed then add `.agents/project` to `.gitignore`

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
- [2025-05-24 10:15] [DECISION] Chose PostgreSQL over SQLite — project needs concurrent writes from multiple workers

# Bad — vague
- [2025-05-24 10:15] [DECISION] Chose a database
```

### 3. Appending to Daily Log

```bash
mkdir -p .agents/project/logs
LOG_FILE=".agents/project/logs/$(date +%Y-%m-%d).md"

# Create file with header if new day
if [ ! -f "$LOG_FILE" ]; then
  echo "# Log $(date +%Y-%m-%d)" >> "$LOG_FILE"
  echo "" >> "$LOG_FILE"
fi

# Append entry
echo "- [$(date '+%Y-%m-%d %H:%M')] [TYPE] Description" >> "$LOG_FILE"
```

### 4. Updating context.md

`context.md` is loaded every session. Keep it lean and current — it should answer: *"What is this project and where are we right now?"*

Update it when:
- Project goals change
- Stack or architecture changes significantly
- Major milestone reached
- User preferences pile up

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
- [2025-05-24 17:30] [SESSION_END] Completed: X, Y. Pending: Z. Next: <suggested next step>
```

---

## Weekly Summary

Generate every Monday (or when 7 days of logs exist). Read the week's daily logs and distill what matters.

**File:** `.agents/project/logs/YYYY-Www.md` (e.g. `2025-W21.md`)

```bash
# Read the week's logs
for i in 0 1 2 3 4 5 6; do
  date -d "-$i days" +%Y-%m-%d 2>/dev/null || date -v-${i}d +%Y-%m-%d
done
```

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

Run at end of each month (or first session of new month).

**File:** `.agents/project/archive/YYYY/MM.md` (e.g. `archive/2025/05.md`)

```bash
mkdir -p .agents/project/archive/$(date +%Y)
```

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

After archiving, **prune daily logs** older than the archived month to keep `.agents/project/logs/` clean.

---

## Additional Important Behaviors

### Auto-detect project init
If `.agents/project/` doesn't exist when a new project starts:

```bash
mkdir -p .agents/project/logs
mkdir -p .agents/project/archive

cat > .agents/project/context.md << 'EOF'
# Project Context

## What This Is
<fill in>

## Current State
Freshly initialized.

## Stack
<fill in>

## User Preferences
<none yet>

## Active TODOs
<fill in>

## Last Updated
YYYY-MM-DD
EOF

echo "- [$(date '+%Y-%m-%d %H:%M')] [SESSION_START] Project memory initialized." >> .agents/project/logs/$(date +%Y-%m-%d).md
```

**ALWAYS**: Ensure `.agents/project` is added to `.gitignore`.

Then ask the user: *"Project memory initialized. What's this project about?"* and fill in `context.md`.

### Error pattern tracking
When logging an `ERROR` + `FIX` pair, also check if the same error appeared in recent logs:
```bash
grep "ERROR.*<keyword>" .agents/project/logs/*.md
```
If it's a repeat, note it: `[ERROR] (repeat #2) ...` — repeated errors should trigger a `DECISION` entry about a permanent fix.

### Preference consolidation
Periodically (every ~10 sessions or when context.md gets long), deduplicate preferences in `context.md`. Remove outdated ones, merge similar ones.

---

## Quick Reference

| Task | Action |
|---|---|
| Start session | Read `context.md` + last 3 daily logs |
| Log an entry | Append to `logs/YYYY-MM-DD.md` |
| Big decision | Log `DECISION` + update `context.md` if it changes the stack/goals |
| End session | Log `SESSION_END` with summary |
| Every Monday | Generate weekly summary |
| New month | Generate monthly archive, prune old daily logs |
| Repeat error | Mark as repeat, consider permanent fix |