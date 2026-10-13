# The generic agent runner: Claude Code, a browser and the runner binary. It
# knows no particular job; a deployment adds its tools and its profiles file on
# top (see content-to-carousel/Dockerfile.agent for the Knaph one).
#
# The credential is not in the image: set CLAUDE_CODE_OAUTH_TOKEN (from
# `claude setup-token`) or ANTHROPIC_API_KEY in the environment.
#
#   docker build -t agent-runner .
FROM golang:1.27 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /agent-runner ./cmd/agent-runner

# headless-shell is a small Chrome for rendering and checking things.
FROM chromedp/headless-shell:latest
# Claude Code is pinned: it changes often, and an unattended update could change
# what a profile's permissions mean. Bump it on purpose, and run the tests.
ARG CLAUDE_CODE_VERSION=2.1.296
RUN apt-get update \
 && apt-get install -y --no-install-recommends ca-certificates nodejs npm git ripgrep fonts-liberation fonts-noto-color-emoji \
 && npm install -g @anthropic-ai/claude-code@${CLAUDE_CODE_VERSION} \
 && rm -rf /var/lib/apt/lists/* /root/.npm
COPY --from=build /agent-runner /usr/local/bin/agent-runner
# A writable home for Claude Code's own state, and scratch space for jobs.
ENV HOME=/tmp/agent-home \
    AGENT_WORK_ROOT=/tmp/agent-work \
    DISABLE_AUTOUPDATER=1 \
    CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1
RUN mkdir -p /tmp/agent-home /tmp/agent-work
WORKDIR /tmp
# Cloud Run sets PORT; AGENT_PROFILES is set by the image built on this one.
ENTRYPOINT ["agent-runner"]
CMD ["serve"]
