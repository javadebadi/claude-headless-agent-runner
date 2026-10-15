---
layout: default
title: Skills and self-checks
render_with_liquid: false
nav_order: 6
---

# 5. Skills and self-checks

A prompt says what to do. A **skill** says how, and can carry files with it. A
skill is a folder:

```
skills/course-check/
  SKILL.md       instructions, with a name and description in its front matter
  check.sh       anything SKILL.md refers to
```

The profile lists it:

```yaml
skills:
  - {name: course-check, from: /abs/path/to/skills/course-check}
```

It is copied into the job at `.claude/skills/course-check/`, where Claude Code
finds it. With `inline: true` its `SKILL.md` body is also added to the system
prompt, for guidance that must always apply rather than be discovered.

Skills come from the profile only. A request cannot add one.

## The most useful pattern: a checker the agent runs on its own work

Models write a 60-file output well and still get one filename or one required
field wrong. Don't hope; give the agent a script and make passing it the exit
condition:

1. The skill contains `check.sh`, which prints `OK` or one line per problem and
   exits non-zero on failure.
2. The profile allows exactly that command:
   `Bash(sh .claude/skills/course-check/check.sh:*)`.
3. The prompt ends with: *run the checker, fix every problem it names, run it
   again until it prints OK, do not finish before that.*

The agent then loops by itself. A deterministic script is better than asking the
model to "double-check": it can't be talked out of a failure.

A checker is just a script. For slides it might count the `---` separators; for
a carousel, that every image prompt names an existing post; for any format, that
a YAML file parses and a required field is present. Keep it small, dependency-free
and fast. The runner image has `sh` and Node, but no Python or Ruby, so write it
in POSIX shell or JavaScript unless you add an interpreter to your own image.

Two cautions:

- A checker proves the *shape*, not the *truth*. If the content must be correct
  (test cases, facts, numbers), the caller still reviews it.
- The checker is data the agent can read. Don't put secrets in it.

A complete one is [`examples/course/skills/course-check`](https://github.com/javadebadi/claude-headless-agent-runner/tree/main/examples/course/skills/course-check),
explained in [A course in a given format](examples/course.md).

Next: [Returning a folder of files](06-folder-output.md).
