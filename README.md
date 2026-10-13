# agent-runner

A small service that runs Claude Code headless as a worker for hire. You give
it a **job**: which kind (a *profile*), some inputs, and folders to read. It
prepares a scratch working folder, runs one Claude Code session there with only
the permissions the profile allows, and hands back **one file** plus what the
session cost.

It knows nothing about what the jobs are for. Everything specific (what to
read, which commands the agent may run, what to prompt) is in the profiles file
of the deployment. Nothing here imports anything from the rest of the Knaph
repository, so this folder can become a repository of its own.

## The idea in one picture

```
caller ──POST /run {profile, inputs, refs, params}──▶ agent-runner
                                                        │ 1. check the request against the profile
                                                        │ 2. make a scratch folder:
                                                        │      .claude/skills/<skills of the profile>
                                                        │      inputs/<the request's texts>
                                                        │      <folders the request asked to copy>
                                                        │ 3. run `claude -p` there, permissions from the profile
                                                        │ 4. read back the profile's one output file
caller ◀── {output, notes, cost_usd, turns, ...} ───────┘   (the scratch folder is deleted)
```

## Profiles: who decides what

A **profile** is a job description, written by whoever deploys the runner (see
`profiles.example.yaml`). A request can only *choose* a profile and fill in the
inputs and parameters it declares. Anything else is refused with a 400, and
unknown JSON fields are refused too, so nobody believes they set permissions
that were never theirs to set.

| set by the deployment (profile) | set by the request |
|---|---|
| model, budget (USD), timeout | which profile |
| the prompt and system text (templates) | the named input texts the profile declares |
| the commands the agent may run | the parameters it declares (plain words only) |
| the folders a request may copy from (`reads`) | which folders to copy, from inside `reads` |
| the skills copied into every job | |
| the one file the agent writes and the job returns | |

A reference to a folder is resolved to its real path and must be inside one of
the profile's `reads`. `..`, symlinks that leave the folder, absolute targets
and writes under `.claude` are refused, and a folder with a symlink inside it
is refused rather than followed.

## Skills and materials

- **Skills** are folders of instructions (a `SKILL.md` and what it refers to).
  The profile lists them; each is copied to `.claude/skills/<name>/` in the job
  folder where Claude Code finds it, and with `inline: true` its `SKILL.md`
  body is also added to the system prompt, for guidance that must always apply.
- **Refs** are folders the request copies in, such as a theme with its own
  `guide.md`. In a container the sources are typically mounted buckets, which
  is how skills and themes live in storage without the runner knowing about it.

## API

- `POST /run` with `{"profile", "inputs": {name: text}, "refs": [{"from",
  "to"}], "params": {name: value}}` answers `{output, notes, model, cost_usd,
  turns, duration_ms, denied}`. `denied` lists tool calls the profile refused,
  which usually means the profile needs a rule, or the agent was misled.
  Status: 400 the request is not allowed, 502 the job failed (the message says
  why), 503 the caller gave up.
- `GET /profiles` lists the profiles and what they take (not their permissions).
- `GET /health`.

Jobs run **one at a time**: a session is heavy and one job's files and browser
must not be another's. Run it with a Cloud Run concurrency of 1.

## Running it

```
make build
AGENT_PROFILES=profiles.example.yaml CLAUDE_CODE_OAUTH_TOKEN=... bin/agent-runner serve
```

Settings (flag, or environment variable): `--profiles` / `AGENT_PROFILES`
(required), `--addr` / `PORT`, `--work-root` / `AGENT_WORK_ROOT`, `--claude` /
`AGENT_CLAUDE`.

**The credential** is read by Claude Code itself from its environment: a
subscription token from `claude setup-token` as `CLAUDE_CODE_OAUTH_TOKEN` (a
year, then renew), or `ANTHROPIC_API_KEY`. The runner never reads or logs it.
Locally, if you are logged in to Claude Code, it uses that login.

The image (`Dockerfile`) holds Claude Code (a pinned version: it changes often,
and a silent update could change what a permission means), a headless Chrome and
the runner. A deployment builds on it, adding its tools and its profiles file;
see `content-to-carousel/Dockerfile.agent`.

## What the permissions do and do not do (read this)

Checked against the real Claude Code CLI (2.1.286), not assumed:

- `permission_mode: acceptEdits` (the default here) lets the agent write inside
  the job's folder and **refuses writes anywhere else**. Path-scoped rules like
  `Write(spec.yaml)` are **not honored** by headless Claude Code in either mode,
  so writing is confined by the mode and the scratch folder, not by a rule.
- Commands the profile does not allow are refused, **except** that Claude Code
  always allows reading files and running read-only commands such as `ls`.
  With `acceptEdits`, simple file commands (`touch`, `mkdir`, …) in the job
  folder are allowed too.

So the rules are a guard rail, **not a sandbox**. The boundary is the
deployment: the container should hold nothing but the one credential, and the
job's only output is the one returned file. Use one deployment (service,
service account, secrets) per kind of work: a spec-drafting service holds no
repository or issue-tracker credentials, and a service that does must not also
run jobs from untrusted input. Inputs are data the agent reads, and could try to
instruct it; keep what it can reach small enough that this does not matter.

## Layout

```
cmd/agent-runner/       the server's entry point
internal/profile/       the profiles file: parsing and validation
internal/job/           preparing the folder, running Claude Code, confinement
internal/api/           POST /run, GET /profiles, GET /health
profiles.example.yaml   an example profiles file
Dockerfile              the generic image (Claude Code + Chrome + runner)
```

`make check` runs vet, gofmt and the tests (a stand-in for `claude` makes them
independent of any login).
