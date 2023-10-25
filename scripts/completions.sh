#!/bin/sh
set -e
rm -rf completions
mkdir completions
for sh in bash zsh fish; do
    go run ./cmd/taskpoet completion "$sh" >"completions/taskpoet.$sh"
done
