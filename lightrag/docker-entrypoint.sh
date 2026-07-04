#!/bin/sh
set -eu

if [ "$#" -gt 0 ] && [ "$1" != "lightrag-server" ]; then
    exec "$@"
fi

rm -rf /app/inputs
mkdir -p /app/inputs

index=1
find /app/documents -type f -iname '*.pdf' -print | sort | while IFS= read -r source; do
    destination="$(printf '/app/inputs/doc_%03d.pdf' "$index")"
    cp "$source" "$destination"
    index=$((index + 1))
done

exec "$@"
