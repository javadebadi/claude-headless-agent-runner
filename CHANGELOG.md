# Changelog

All notable changes to this project are listed here. The format follows
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and versions follow
[Semantic Versioning](https://semver.org/).

## [Unreleased]

### Added

- When a job ends without its output after one turn and no refusal, the error
  now says the agent used no tool and the prompt may be too vague for the model
  (the agent's own reply, "I need to see the text", reads like a bad request).
- A multi-page tutorial in `docs/`, published as a GitHub Pages site: a first job, inputs and folders, permissions,
  skills and self-checks, folder output, deploying, and worked examples.
- Runnable examples in `examples/` (a course, and Instagram carousel posts from a
  book), each with a checker the agent runs on its own work, and `examples/run.sh`.

## [0.3.0] - 2026-10-13

### Added

- A profile can return a folder of files instead of one file: set `output_dir`
  in place of `output`. The response then carries `files`, a map from each
  file's path inside the folder to its text, and no `output`. Use it for work
  whose files must agree with each other, such as a course or a slide deck with
  its pages. A profile sets exactly one of `output` and `output_dir`.
- Limits on an output folder: text files only (a symlink or a binary file fails
  the job), at most 500 files and 8 MiB, and a request cannot copy a ref into
  it. An empty or missing folder fails like a missing file, with the cost of
  the session in the error.
- An example `book-to-pack` profile in `profiles.example.yaml`, README sections
  for a book-to-slides job and for folder output, and a test that keeps the
  example profiles file valid.

### Changed

- `output` in a response is left out when the profile uses `output_dir`.
  Responses of single-file profiles are unchanged.

## [0.2.0]

### Added

- The response reports the tokens a session used (`usage`), and a failed job
  reports what it spent (`cost_usd`, `turns`, `duration_ms`, `usage`).

## [0.1.0]

- First release: `POST /run`, `GET /profiles` and `GET /health`; profiles with
  skills, inputs, params and read-only folder refs; one file back per job.

[0.3.0]: https://github.com/javadebadi/claude-headless-agent-runner/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/javadebadi/claude-headless-agent-runner/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/javadebadi/claude-headless-agent-runner/releases/tag/v0.1.0
