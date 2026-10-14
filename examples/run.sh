#!/bin/sh
# Runs an example end to end: starts the runner with the example's profile, posts
# its request, writes the returned files to a folder.
# Needs `claude` logged in (or a credential in the environment), curl and jq.
# A real run spends real money: see the profile's budget.
#
# Usage: examples/run.sh <course|carousel> [output folder]
set -e
name="${1:?usage: run.sh <course|carousel> [output folder]}"
here=$(cd "$(dirname "$0")" && pwd)
dir="$here/$name"
out="${2:-$dir/out}"
work=$(mktemp -d)
server=""
trap '[ -z "$server" ] || kill "$server" 2>/dev/null; rm -rf "$work"' EXIT

make -C "$here/.." build >/dev/null
# profiles need absolute paths: ROOT stands for this example's folder
sed "s|ROOT|$dir|g" "$dir/profiles.yaml" > "$work/profiles.yaml"
sed "s|ROOT|$dir|g" "$dir/request.json"  > "$work/request.json"

AGENT_PROFILES="$work/profiles.yaml" PORT=8089 "$here/../bin/agent-runner" serve &
server=$!
sleep 2
curl -sS --max-time 2400 -X POST localhost:8089/run -d @"$work/request.json" -o "$work/response.json"

"$here/save.sh" "$work/response.json" "$out"
echo "written to $out"
