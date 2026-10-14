---
name: course-check
description: Check a course folder before finishing. Run it after writing and fix every problem it reports.
---

# Checking a course

When the course is written, run:

    sh .claude/skills/course-check/check.sh course

It prints `OK` when the course is valid, otherwise one line per problem. Fix each
problem in the file it names and run it again. Do not finish until it prints `OK`.
