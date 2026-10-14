---
title: Inputs, parameters and folders
render_with_liquid: false
nav_order: 3
---

# 2. Inputs, parameters and folders

A job has three ways to receive what it works on. Each exists for a reason.

| | In the request | In the agent's folder | Use it for |
|---|---|---|---|
| **inputs** | `"inputs": {"brief": "..."}` | a text file, `inputs/brief` | free text: a brief, an article, a list |
| **params** | `"params": {"slides": "10"}` | a value in the prompt only | short plain words that steer the job: counts, a language, an audience |
| **refs** | `"refs": [{"from": "/data/books/my-book", "to": "book"}]` | a copy of the folder, at `book/` | material too big or too structured for a text: a book, a theme, a sample |

The profile declares all three (`inputs: [brief]`, `params: [slides]`,
`reads: [/data/books]`). A request that sends anything not declared is refused
with a 400. That is deliberate: a caller can't believe they changed something
that was never theirs to change.

## Params are plain words

A param is for a number or a word, not for instructions. Because it lands inside
your prompt, the runner accepts only letters, digits, spaces and `. _ : / -`, up
to 100 characters. Anything longer or more free-form belongs in `inputs`, which
the agent reads as a file and treats as data. A declared param the request leaves
out reads as empty in the prompt rather than as an error, so word the prompt to
cope, or have the caller always send it.

## Refs: folders to read

`reads` in the profile is the list of folders a request may copy from. A ref's
`from` must be an **absolute path to a folder** inside one of them. It is copied
into the job's scratch folder under `to`.

Refused: a path outside `reads`, `..`, a relative `from`, a symlink that leaves
the folder, a folder containing a symlink, and a `to` under `.claude`. A single
file cannot be referenced, so put a book in a folder of its own, or the whole
surrounding folder is copied.

## An example: slides from a book

```yaml
profiles:
  book-to-slides:
    description: Turn a markdown book into a slide deck
    model: claude-sonnet-5-5
    budget: 1.00
    timeout: 10m
    output: slides.md
    inputs: [brief]
    params: [slide_count, audience]
    reads: [/Users/me/Documents/books]
    allowed_tools: ["Read"]
    system: |
      You turn books into slide decks. Write {{.Output}}.
    prompt: |
      Read the book in book/ and the brief in {{index .Inputs "brief"}}.
      Write exactly {{index .Params "slide_count"}} slides for
      {{index .Params "audience"}} to {{.Output}}. Separate slides with ---.
      Each slide has a title and 3-5 bullets. Reply with one line.
```

```sh
curl -X POST localhost:8080/run -d '{
  "profile": "book-to-slides",
  "inputs": {"brief": "The main argument, one key idea per chapter, end with a summary."},
  "refs":   [{"from": "/Users/me/Documents/books/my-book", "to": "book"}],
  "params": {"slide_count": "10", "audience": "software engineers"}
}'
```

The prompt refers to `book/`, the `to` of the ref. Pick names and use them
consistently: the prompt is the only place the agent learns where things are.

The full walkthrough is in [Slides from a book](examples/book-to-slides.md).

## What a request can never do

It can't add skills, tools or permissions. If a caller has guidance for the job,
it arrives as an input or a ref, and the agent reads it as data.

Next: [Permissions](03-permissions.md).
