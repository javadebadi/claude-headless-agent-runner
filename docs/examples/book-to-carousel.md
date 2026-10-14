---
layout: default
title: Instagram posts from a book
render_with_liquid: false
nav_order: 3
parent: Worked examples
---

# Example: Instagram posts from a book

Turn a book into a carousel: for each slide a caption, and for each caption the
prompt for the image that goes with it. They must agree (the third image shows
what the third caption says), so one session writes all of them into a folder.

The runnable files are in [`examples/carousel`](https://github.com/javadebadi/claude-headless-agent-runner/tree/main/examples/carousel):
`examples/run.sh carousel` runs it, with a short sample book.

## The profile

```yaml
profiles:
  book-to-carousel:
    description: Turn a book into Instagram carousel posts
    model: claude-sonnet-5-5
    budget: 2.00
    timeout: 15m
    output_dir: posts
    inputs: [brief]
    params: [posts, slides_per_post, language]
    reads: [ROOT/book]
    allowed_tools:
      - "Read"
      - "Bash(sh .claude/skills/carousel-check/check.sh:*)"
    skills:
      - {name: carousel-check, from: ROOT/skills/carousel-check}
    system: |
      You write social media posts from books. Write only text files under
      {{.Output}}/.
    prompt: |
      Read the book in book/ and the brief in {{index .Inputs "brief"}}.
      Write {{index .Params "posts"}} posts in {{index .Params "language"}}, each
      a carousel of {{index .Params "slides_per_post"}} slides:
        {{.Output}}/NN/caption.md          the caption: a hook, a line per idea, a closing question
        {{.Output}}/NN/slide-MM.md         the text on slide MM, at most 25 words
        {{.Output}}/NN/image-MM.md         a prompt for the image of slide MM
      The first line of each image prompt is "Illustrates slide-MM.md." and the rest
      describes a scene that shows what that slide says. Use only ideas that are in
      the book. Then run the checker:
        sh .claude/skills/carousel-check/check.sh {{.Output}}
      Fix what it lists and run it again until it prints OK.
```

## The request

```sh
curl -X POST localhost:8080/run -d '{
  "profile": "book-to-carousel",
  "inputs": {"brief": "Friendly tone. Each post is one idea a reader can use today."},
  "refs":   [{"from": "/data/books/my-book", "to": "book"}],
  "params": {"posts": "3", "slides_per_post": "5", "language": "English"}
}'
```

## Then what

The answer carries `files`, a map of `01/caption.md`, `01/slide-01.md`,
`01/image-01.md`, and so on. What happens next is the caller's: send each
`image-*.md` to an image model, lay the slide text over the returned image,
schedule the post. The runner stops at text, which keeps it small and keeps
credentials for image models and social accounts out of the container.

## Choices

- **A folder per post** makes the layout the checker verifies: the same files
  exist for every post, and every image prompt names a slide that exists.
- **"Only ideas that are in the book"** and the brief's tone are where a human
  would check. The checker proves the shape, not that the book says it.
- **No social credentials in the service.** Publishing is another system with
  other permissions ([page 6](../06-deploying.md)).
