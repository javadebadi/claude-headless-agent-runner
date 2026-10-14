---
layout: default
title: Tutorial
render_with_liquid: false
nav_order: 1
permalink: /
---

# agent-runner tutorial

agent-runner turns "do this piece of work with Claude Code" into one HTTP call.
You describe the kind of work once, in a **profile**. After that, anyone who can
reach the service can ask for a **job** of that kind and get the result back,
without a terminal, a login or any Claude Code knowledge.

It is not about any one subject. The same runner writes a summary, turns a book
into slides or into a set of Instagram posts, drafts a course, fills in a
report. What changes is only the profile.

Read the pages in order the first time; each one adds one idea and builds on the
previous job.

| | Page | You learn |
|---|---|---|
| 1 | [Your first job](01-first-job.md) | build it, write a profile, call it, read the answer |
| 2 | [Inputs, parameters and folders](02-inputs-params-refs.md) | how a request feeds a job: texts, plain words, folders to read |
| 3 | [Permissions](03-permissions.md) | what the agent may do, and what that does and doesn't protect |
| 4 | [Skills and self-checks](04-skills-and-checks.md) | give the agent instructions and a script to check its own work |
| 5 | [Returning a folder of files](05-folder-output.md) | `output_dir`, for work whose files must agree |
| 6 | [Deploying it](06-deploying.md) | the image, credentials, Cloud Run settings |

## Worked examples

Complete, each with a profile and a request you can adapt:

- [A summary](examples/summary.md): the smallest useful job, one text in, one file out.
- [Slides from a book](examples/book-to-slides.md): a folder to read, one file out.
- [Instagram posts from a book](examples/book-to-carousel.md): captions and image prompts that must agree, so a folder out.
- [A course in a given format](examples/course.md): a larger folder, with a script the agent runs to check it.

The runnable files for the last two are in [`examples/`](https://github.com/javadebadi/claude-headless-agent-runner/tree/main/examples).

The [README](https://github.com/javadebadi/claude-headless-agent-runner#readme) is the reference; these pages are the walkthrough.
