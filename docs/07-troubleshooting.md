---
layout: default
title: Troubleshooting
render_with_liquid: false
nav_order: 8
---

# 7. Troubleshooting

Each entry shows what you did, what came back, how to read it, and the fix. They
are real: the first one happened while writing this tutorial.

## 1. "I need to see what text you want me to summarize"

### What you did

A profile whose prompt is short and assumes the agent will work out the rest:

```yaml
profiles:
  summarize:
    model: claude-haiku-4-5-20251001
    output: summary.md
    inputs: [text]
    params: [language]
    allowed_tools: ["Read"]
    system: |
      You summarize text. Write only {{.Output}}.
    prompt: |
      Read {{index .Inputs "text"}} and write a one paragraph summary of it to
      {{.Output}}, in {{index .Params "language"}}. Reply with one line saying
      what you wrote.
```

and a request that did include the text:

```sh
curl -X POST localhost:8080/run -d '{
  "profile": "summarize",
  "inputs": {"text": "Go is a statically typed, compiled language designed at Google. It has garbage collection, built-in concurrency, and a fast compiler."},
  "params": {"language": "English"}
}'
```

### What came back

A 502 (the job failed):

```json
{
  "error": "the agent finished without writing summary.md: I need to see what text you want me to summarize. Could you either:\n1. Provide the text directly in your message, or\n2. Point me to the file path where the input text is located\n\nOnce you provide that, I'll read it and write a one-paragraph summary to summary.md.",
  "cost_usd": 0.0181695,
  "turns": 1,
  "duration_ms": 3715,
  "usage": {"input_tokens": 10, "output_tokens": 285, "cache_read_tokens": 12425, "cache_creation_tokens": 7746}
}
```

### How to read it

The message sounds like your **request** is missing the text, but it isn't: the
text was sent and the runner wrote it to `inputs/text`. Look at the other fields:

- **`turns: 1`.** Opening a file and writing another takes at least three turns.
  One turn means the agent answered in plain text and **never called a tool**.
  It didn't open `inputs/text`, so from its point of view there was no text.
- **No `denied`.** Nothing was refused, so this is not a permissions problem
  ([page 3](03-permissions.md)).
- **It still cost money** (`cost_usd`): a failed job spends tokens too.

So the problem is the **prompt for this model**, not the request. A small model
(here Haiku) given a terse instruction sometimes doesn't reach for its tools.

Recent versions of the runner say this in the error itself, by appending:

```text
(the session ended after one turn without using any tool, so it never read the
inputs or wrote the output: the prompt may be too vague for this model; name the
input files and the tools to use, or try a larger model)
```

If your error has no such note, you are on an older runner: use the `turns`
reading above.

### The fix

Say plainly where the content is and which tool to use:

```yaml
    system: |
      You summarize text. Use your tools to read the input file and to write the
      output file.
    prompt: |
      The text to summarize is in the file {{index .Inputs "text"}}. Open it with
      the Read tool, then write a one paragraph summary of it to {{.Output}}, in
      {{index .Params "language"}}. Reply with one line saying what you wrote.
```

In testing, the terse prompt failed 4 of 4 runs with Haiku and the explicit one
worked 4 of 4. Sonnet worked with either, so switching `model` to
`claude-sonnet-5-5` is the other fix, at a higher price per job.

### The habit

Run a new profile several times with the model you will use before you rely on
it, and look at `turns` in the answers. A successful job that reads a file and
writes one takes three or more.
