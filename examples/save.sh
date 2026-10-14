#!/bin/sh
# Writes the "files" of a /run response to a folder, and prints the rest of the response.
# (a runner does not write files for you: that is the caller's job). Needs jq.
set -e
jq -e '.files' "$1" >/dev/null || { jq . "$1" | head -40; exit 1; }
mkdir -p "$2"
jq -r '.files | keys[]' "$1" | while IFS= read -r path; do
  mkdir -p "$2/$(dirname "$path")"
  jq -j --arg p "$path" '.files[$p]' "$1" > "$2/$path"
done
jq -c 'del(.files)' "$1"
