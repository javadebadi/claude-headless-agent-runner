---
layout: default
title: Slides from a book
render_with_liquid: false
nav_order: 2
parent: Worked examples
---

# Example: slides from a book

A folder to read (the book) and a text brief, one file out. The profile, request
and answer are in [page 2](../02-inputs-params-refs.md#an-example-slides-from-a-book);
here are the choices behind them.

- **The book is a ref, not an input.** A book is many files and too large to paste
  into JSON. It lives in a folder under `reads`, and the request names it.
- **The brief is an input.** "Main argument, one key idea per chapter" is
  free text, and different for each call.
- **Counts and audience are params.** Short, plain, and the prompt says exactly
  where they go.
- **The output is one file** because slides are one deck. Marp turns a
  `slides.md` separated by `---` into PDF or PowerPoint.

Things worth knowing:

- Say in the prompt how the agent should find the book (`book/`), because that is
  the only place it learns the folder's name.
- "Write exactly N slides" is followed most of the time and missed sometimes.
  Count `---` in the caller, or in a checker the agent runs
  ([page 4](../04-skills-and-checks.md)).
- If you also want speaker notes or images that match the slides, you have
  crossed into files that must agree: use a folder.
  [Instagram posts from a book](book-to-carousel.md) does that.
