---
layout: default
title: Permissions
render_with_liquid: false
nav_order: 5
---

# 4. Permissions

An agent that can read and write files and run commands is useful, and it
needs limits. This page says exactly what the profile controls, checked against
the real Claude Code CLI rather than assumed.

## The pieces

```yaml
permission_mode: acceptEdits      # the default; you usually leave it out
allowed_tools:
  - "Read"
  - "Bash(sh .claude/skills/course-check/check.sh:*)"
denied_tools:
  - "Read(**/.env)"
```

- `allowed_tools` and `denied_tools` are Claude Code permission rules. In a
  headless session nobody can click "allow", so a tool that is not allowed is
  simply refused.
- `permission_mode: acceptEdits` lets the agent **write inside the job's scratch
  folder and nowhere else**.

## What this does not do

Two surprises, both measured:

1. **Path-scoped write rules are ignored.** `Write(summary.md)` does nothing in
   headless Claude Code. Writing is confined by the mode and the scratch
   folder, not by a rule, so don't write one and feel safe.
2. **Reading is always allowed.** Claude Code always permits reading files and
   read-only commands like `ls`. Add `Read` to `allowed_tools` anyway so the
   intent is written down, and keep sensitive things out of reach rather than
   relying on a deny rule.

So the rules are a guard rail, **not a sandbox**. The real boundary is the
deployment: a container that holds nothing but the one credential, and a job
whose only output is the file or folder it returns. One deployment per kind of
work, and never mix a service that holds repository or database credentials
with one that runs jobs from untrusted input.

## Reading the `denied` list

Every response carries `denied`: the tool calls the profile refused. It is how
you tune a profile. In practice you will see this:

```json
"denied": ["Bash {\"command\":\"cat sample/course.yaml modules/...\"}"]
```

The agent tried `cat` through Bash. It was refused (only the checker script is
allowed), and it fell back to `Read`, which is allowed. The job still
succeeded, so nothing to fix. When a job *fails* and `denied` is full of one
command, that command probably needs an `allowed_tools` rule.

A `denied` entry can also mean the agent was **misled**, for example text in an
input that told it to run something. A refusal is the guard rail doing its work;
a pattern of them is worth a look.

## Allowing a command narrowly

Allow the exact command, not a family:

```yaml
allowed_tools:
  - "Bash(sh .claude/skills/course-check/check.sh:*)"   # this script, any arguments
```

Not `Bash(sh:*)` or `Bash`.

Next: [Skills and self-checks](05-skills-and-checks.md).
