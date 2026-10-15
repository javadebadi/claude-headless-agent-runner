---
layout: default
title: Deploying it
render_with_liquid: false
nav_order: 8
---

# 7. Deploying it

## The image

The `Dockerfile` builds a generic image: Claude Code (a pinned version, because it
changes often and a silent update could change what a permission means), a
headless Chrome, and the runner. It knows no job. You build **your** image on it,
adding what your profiles need: a profiles file, skills, tools.

```dockerfile
FROM agent-runner:latest
COPY profiles.yaml /etc/agent/profiles.yaml
COPY skills/ /opt/skills/
ENV AGENT_PROFILES=/etc/agent/profiles.yaml
```

Remember that paths in a profile are absolute, and inside the container they are
the container's paths (`/opt/skills/course-check`, not your laptop's).

## The credential

Never in the image. Give the container one of:

- `CLAUDE_CODE_OAUTH_TOKEN`, from `claude setup-token` (valid for a year)
- `ANTHROPIC_API_KEY`

From a secret store, as an environment variable. The runner doesn't read or log it.

## Settings

| flag | environment | |
|---|---|---|
| `--profiles` | `AGENT_PROFILES` | the profiles file (required) |
| `--addr` | `PORT` | where to listen (Cloud Run sets `PORT`) |
| `--work-root` | `AGENT_WORK_ROOT` | where job scratch folders go |
| `--claude` | `AGENT_CLAUDE` | the `claude` binary |

## Cloud Run

- **Concurrency 1.** Jobs run one at a time; this keeps a second request from
  queueing inside a busy instance.
- **Request timeout above the longest profile `timeout`.** Cloud Run's default is
  5 minutes. A profile with `timeout: 15m` needs a service timeout above that,
  or the platform cuts the connection first.
- **Memory** for the session plus whatever `refs` copies: the scratch folder is
  in memory there.
- **Ingress and auth.** Anyone who can call it can spend your budget. Require
  authentication, and keep each profile's `budget` as low as the job allows.
- **Sources as mounted buckets.** Books, themes and skills can live in storage
  mounted into the container; the runner only sees folders.

## One deployment per kind of work

Permissions are a guard rail, not a sandbox ([page 4](04-permissions.md)). The
sandbox is the deployment. A service that turns books into posts holds one
credential and nothing else; a service that can touch a repository is a different
service, with different callers.

That is the end of the walkthrough. The [worked examples](examples/) put the pieces
together, and the [README](https://github.com/javadebadi/claude-headless-agent-runner#readme)
is the reference.
