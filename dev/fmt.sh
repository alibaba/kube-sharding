#!/bin/bash

# go fmt modified files
git diff --name-only | xargs -L 1 go fmt