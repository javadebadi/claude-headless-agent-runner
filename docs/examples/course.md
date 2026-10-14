---
title: A course in a given format
render_with_liquid: false
nav_order: 4
parent: Worked examples
---

# Example: a course in a given format

A larger folder, in a format the agent must copy exactly, with a script it runs
to confirm it did. Files are in [`examples/course`](https://github.com/javadebadi/claude-headless-agent-runner/tree/main/examples/course).

The idea is general: *"here is one finished example in our format; make a new
one on a different subject."* A course, a spec, a product page, a lesson plan, a
runbook.

## The pieces

```
examples/course/
  profiles.yaml            the profile
  request.json             a sample request
  sample/                  a small finished course, the format to copy
  skills/course-check/     SKILL.md and check.sh, the checker
```

- **The sample is a ref.** `refs: [{"from": ".../sample", "to": "sample"}]`.
  The prompt tells the agent to study it and copy the structure but never the
  text. Showing a finished example is more reliable than describing a format.
- **The checker is a skill.** `check.sh` knows the format's rules: numbered
  modules, a title in every file, a quiz last, `status: draft`. The prompt makes
  passing it the exit condition ([page 4](../04-skills-and-checks.md)).
- **Only one command is allowed** besides reading: the checker, by exact path.
- **The output is a folder** (`output_dir: course`), because lessons, exercises
  and quizzes must be consistent.

## Run it

```sh
examples/run.sh course              # writes to examples/course/out
sh examples/course/skills/course-check/check.sh examples/course/out
```

`run.sh` builds the runner, fills the absolute paths into the profile and request
(profiles need absolute paths), starts the service, posts the request and saves
the files. The second line runs the checker once more as the caller, which is a
good habit: don't rely on the agent's word that it passed. It needs `claude`
logged in (or a credential in the environment) and `jq`. A run costs real money: the
profile's budget is `4.00`.

## Setting `status: draft`

The prompt says to, and the checker fails on anything else. That is the pattern
for "a person must review this before it goes live": the agent can't publish
because it can't produce anything but a draft.

## What the checker doesn't cover

It does not know whether the content is right. If the course has exercises with
expected output, someone has to run them. A deterministic check of shape plus a
human check of substance is the usual split.
