---
layout: default
title: A summary
render_with_liquid: false
nav_order: 1
parent: Worked examples
---

# Example: a summary

The smallest useful job: one text in, one paragraph out. It is the profile from
[page 1](../01-first-job.md); this page is the checklist for adapting it.

```yaml
summarize:
  model: claude-haiku-4-5-20251001   # a small model is enough
  budget: 0.25
  timeout: 5m
  output: summary.md
  inputs: [text]
  params: [language]
  allowed_tools: ["Read"]
```

Why it is shaped this way:

- **A cheap model and a low budget.** The work is small. The budget is the
  ceiling a bug or a hostile caller can reach, so set it close to what the job
  needs.
- **One input, no refs.** The text is free-form, so it is an `input`. There is
  nothing to read from disk, so `reads` is empty and the agent has no folder
  to wander in.
- **`Read` only.** It needs nothing else.

- **A small model needs an explicit prompt.** Name the file and the tool ("open
  it with the Read tool"). Haiku given just "Read inputs/text and summarize it"
  sometimes replied that it saw no text and never opened the file.

Adapting it: change the prompt and the output name. "Translate", "extract the
action items", "rewrite for a child" and "classify" are all this profile with a
different sentence.

Next example: [Slides from a book](book-to-slides.md).
