---
title: Your first job
render_with_liquid: false
nav_order: 2
---

# 1. Your first job

You will start the runner with one profile and ask it for a summary.

## What you need

- Go, to build the runner (`make build`).
- Claude Code, logged in (`claude` works in your terminal), or a credential in
  the environment: `CLAUDE_CODE_OAUTH_TOKEN` (from `claude setup-token`) or
  `ANTHROPIC_API_KEY`. The runner never reads it; Claude Code does.

## The profile

A profile is a job description, written by **you, the deployer**. Save this as
`profiles.yaml`:

```yaml
profiles:
  summarize:
    description: Summarize a text in one paragraph
    model: claude-haiku-4-5-20251001
    budget: 0.25          # USD; the session stops if it would spend more
    timeout: 5m
    output: summary.md    # the one file the agent writes, and what comes back
    inputs: [text]        # the request must supply this
    params: [language]    # the request may supply this
    allowed_tools: ["Read"]
    system: |
      You summarize text. Write only {{.Output}}.
    prompt: |
      Read {{index .Inputs "text"}} and write a one paragraph summary of it to
      {{.Output}}, in {{index .Params "language"}}. Reply with one line saying
      what you wrote.
```

Notice what is fixed here and not up to the caller: the model, the spending
limit, the time limit, what the agent may use, what file it must write. The
caller only fills in `text` and `language`.

`system` and `prompt` are templates. `{{.Output}}` is the file the agent must
write; `{{index .Inputs "text"}}` is the path where the request's `text` was
placed for the agent to read; `{{index .Params "language"}}` is a value from
the request.

## Run it

```sh
make build
AGENT_PROFILES=profiles.yaml PORT=8080 bin/agent-runner serve
```

See what it offers (this lists inputs and parameters, never permissions):

```sh
curl localhost:8080/profiles
```

Ask for a job:

```sh
curl -X POST localhost:8080/run -d '{
  "profile": "summarize",
  "inputs": {"text": "Go is a statically typed, compiled language designed at Google. It has garbage collection, built-in concurrency, and a fast compiler."},
  "params": {"language": "English"}
}'
```

## The answer

```json
{
  "output": "Go is a compiled, statically typed language from Google with garbage collection, built-in concurrency and a fast compiler.",
  "notes": "Wrote summary.md",
  "model": "claude-haiku-4-5-20251001",
  "cost_usd": 0.004,
  "turns": 3,
  "duration_ms": 6100,
  "usage": {"input_tokens": 12, "output_tokens": 60, "cache_read_tokens": 0, "cache_creation_tokens": 0},
  "denied": []
}
```

- `output` is the content of the file the agent wrote.
- `notes` is what the agent replied at the end (the prompt asked for one line).
- `cost_usd` is Claude Code's own estimate, not a bill.
- `denied` lists tool calls the profile refused. Empty is good; if it isn't,
  [page 3](03-permissions.md) shows how to read it.

(The numbers above are illustrative.)

## When it goes wrong

| Status | Meaning |
|---|---|
| 400 | the request is not allowed: unknown profile, a missing input, a field the profile does not declare. Nothing ran. |
| 502 | the job failed. The message says why. If a session ran, the body still has `cost_usd` and `usage`: a failed job spends tokens too. |
| 503 | the caller gave up (disconnected) |

Jobs run **one at a time**, because each has its own files and a session is
heavy. A second request waits for the first.

Next: [Inputs, parameters and folders](02-inputs-params-refs.md).
