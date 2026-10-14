#!/bin/sh
# Checks a course folder against the format in sample/. POSIX sh only, so it
# runs in the runner image (which has no Python or Ruby).
root="${1:?usage: check.sh <course folder>}"
errors=""
err() { errors="$errors$1: $2
"; }

front_title() { awk 'NR==1&&$0!="---"{exit 1} NR>1&&$0=="---"{exit 0} /^title: *[^ ]/{f=1} END{exit !f}' "$1"; }

c="$root/course.yaml"
if [ ! -f "$c" ]; then err course.yaml missing; else
  for k in slug title summary; do grep -q "^$k: *[^ ]" "$c" || err course.yaml "missing $k"; done
  grep -q '^status: draft$' "$c" || err course.yaml "status must be draft until a person reviews it"
  grep -q '^id:' "$c" && err course.yaml "has an id field: ids are assigned later"
fi

n=0
for dir in "$root"/modules/*/; do
  [ -d "$dir" ] || continue
  name=$(basename "$dir")
  want=$(printf '%02d-' "$n")
  case "$name" in "$want"*) ;; *) err "modules/$name" "should be numbered $want<slug>" ;; esac
  grep -q '^title: *[^ ]' "$dir/module.yaml" 2>/dev/null || err "modules/$name/module.yaml" "missing or no title"

  i=0; last=""; lessons=0
  for f in $(ls "$dir" | grep -v '^module.yaml$' | sort); do
    last="$f"
    want=$(printf '%02d-' "$i")
    case "$f" in "$want"*) ;; *) err "modules/$name/$f" "numbering should start $want" ;; esac
    case "$f" in
      [0-9][0-9]-lesson-*.md)   lessons=$((lessons+1)); front_title "$dir/$f" || err "modules/$name/$f" "needs front matter with a title"
                                [ "$(wc -w < "$dir/$f")" -gt 30 ] || err "modules/$name/$f" "body is too short" ;;
      [0-9][0-9]-exercise-*.md) front_title "$dir/$f" || err "modules/$name/$f" "needs front matter with a title"
                                grep -q '^## Task' "$dir/$f" && grep -q '^## Expected output' "$dir/$f" || err "modules/$name/$f" "needs '## Task' and '## Expected output'" ;;
      [0-9][0-9]-quiz-*.md)     front_title "$dir/$f" || err "modules/$name/$f" "needs front matter with a title"
                                [ "$(grep -c '^ *- \[x\]' "$dir/$f")" -ge 1 ] || err "modules/$name/$f" "needs questions with one - [x] answer each" ;;
      *) err "modules/$name/$f" "unexpected name (NN-lesson-*.md, NN-exercise-*.md, NN-quiz-*.md)" ;;
    esac
    i=$((i+1))
  done
  [ "$lessons" -ge 1 ] || err "modules/$name" "needs at least one lesson"
  case "$last" in *-quiz-*) ;; *) err "modules/$name" "must end with a quiz" ;; esac
  n=$((n+1))
done
[ "$n" -ge 1 ] || err "$root" "no modules"

if [ -z "$errors" ]; then echo OK; else printf '%s' "$errors"; exit 1; fi
