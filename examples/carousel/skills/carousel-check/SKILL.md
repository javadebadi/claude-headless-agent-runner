---
name: carousel-check
description: Check a folder of carousel posts before finishing. Run it after writing and fix every problem it reports.
---

# Checking the posts

When the posts are written, run:

    sh .claude/skills/carousel-check/check.sh posts

It prints `OK` when every post is complete, otherwise one line per problem. Fix
each one in the file it names and run it again. Do not finish until it prints `OK`.
