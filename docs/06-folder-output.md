---
layout: default
title: Returning a folder of files
render_with_liquid: false
nav_order: 7
---

# 6. Returning a folder of files

So far a job returned one file (`output`). Some work is many files that have to
agree with each other: slides and the pages they refer to, captions and the image
prompts that illustrate them, a course and its lessons. One session should write
them all, so they can be consistent. Use `output_dir` in place of `output`:

```yaml
profiles:
  book-to-pack:
    output_dir: pack        # the folder the agent fills; every file comes back
    ...
    prompt: |
      Write {{.Output}}/slides/NN.md, {{.Output}}/pages/NN.md and
      {{.Output}}/image-prompts/NN.md ...
```

`{{.Output}}` is now the folder's name, `pack`. A profile sets exactly one of
`output` and `output_dir`.

## The answer

The response has `files`, a map from each file's path inside the folder to its
text, and no `output`:

```json
{
  "files": {
    "slides/01.md": "# Title\n- point",
    "pages/01.md": "...",
    "image-prompts/01.md": "Illustrates pages/03.md. A watercolor of ..."
  },
  "notes": "Wrote 10 slides, 10 pages, 5 image prompts",
  "cost_usd": 1.1
}
```

## The runner returns text, not a link

It stores nothing and makes no zip. **The caller writes the files**: to disk, to a
zip, to its own bucket, or straight into the next step (each image prompt into an
image model). A few lines of shell do it:

```sh
jq -r '.files | keys[]' response.json | while read -r p; do
  mkdir -p "out/$(dirname "$p")"
  jq -j --arg p "$p" '.files[$p]' response.json > "out/$p"
done
```

(`examples/save.sh` is this, with a check that the job succeeded.)

## Limits

A folder holds **text files only**: a symlink or binary file fails the job. At
most 500 files and 8 MiB. A request can't copy a ref *into* the output folder.
An empty or missing folder fails like a missing file, and the error still carries
the session's cost.

## The count is a request, not a guarantee

The model usually produces the number you ask for and is sometimes off by one.
Either have the caller check the counts, or, better, put the count in a checker
([page 5](05-skills-and-checks.md)) so the agent corrects itself.

## One session or several?

Many files that depend on each other: one job with `output_dir`. Many
independent files: one job each. If you need a series of jobs to stay consistent,
pass the earlier result into the next as an `input`.

Next: [Deploying it](07-deploying.md).
