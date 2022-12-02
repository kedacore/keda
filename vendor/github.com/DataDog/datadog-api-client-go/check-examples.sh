#!/usr/bin/env bash

# trap "rm -rf build" EXIT

for f in examples/*/*/*.go ; do
    df="build/$(dirname "$f")/$(basename "$f" .go)"
    mkdir -p "$df"
    cp "$f" "$df/main.go"
done

if (find ./build/examples/*/*/* -type d -print0 | xargs -0 go build -o /dev/null -ldflags "-s -w"); then
    echo -e "Examples are buildable"
else
    echo -e "Failed to build examples"
    exit 1
fi
