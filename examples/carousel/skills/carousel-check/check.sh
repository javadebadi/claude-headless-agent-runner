#!/bin/sh
# Every post folder (01, 02, ...) has a caption and, for each slide-MM.md, an
# image-MM.md whose first line is "Illustrates slide-MM.md.". Slides are at most
# 25 words. POSIX sh only.
root="${1:?usage: check.sh <posts folder>}"
errors=""
err() { errors="$errors$1: $2
"; }

n=0
for dir in "$root"/*/; do
  [ -d "$dir" ] || continue
  n=$((n+1))
  post=$(basename "$dir")
  case "$post" in [0-9][0-9]) ;; *) err "$post" "post folders are numbered 01, 02, ..." ;; esac
  [ -s "$dir/caption.md" ] || err "$post/caption.md" "missing or empty"
  slides=0
  for s in "$dir"/slide-*.md; do
    [ -f "$s" ] || continue
    slides=$((slides+1))
    m=$(basename "$s" .md); m=${m#slide-}
    [ "$(wc -w < "$s")" -le 25 ] || err "$post/slide-$m.md" "more than 25 words"
    img="$dir/image-$m.md"
    if [ ! -f "$img" ]; then err "$post/image-$m.md" "missing (every slide needs an image prompt)"
    elif [ "$(head -n 1 "$img")" != "Illustrates slide-$m.md." ]; then
      err "$post/image-$m.md" "first line must be: Illustrates slide-$m.md."
    fi
  done
  [ "$slides" -ge 1 ] || err "$post" "no slides"
  for i in "$dir"/image-*.md; do
    [ -f "$i" ] || continue
    m=$(basename "$i" .md); m=${m#image-}
    [ -f "$dir/slide-$m.md" ] || err "$post/image-$m.md" "there is no slide-$m.md for it"
  done
done
[ "$n" -ge 1 ] || err "$root" "no posts"

if [ -z "$errors" ]; then echo OK; else printf '%s' "$errors"; exit 1; fi
