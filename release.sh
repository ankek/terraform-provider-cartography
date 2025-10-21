#!/bin/bash

go mod init github.com/you/your-repo

goreleaser init

# run a "local-only" release to see if it works using the release command:
goreleaser release --snapshot --clean

# verify your .goreleaser.yaml is valid by running the check command:

goreleaser check

# The minimum permissions the GITHUB_TOKEN should have to run this are write:packages
export GITHUB_TOKEN="YOUR_GH_TOKEN"
export GPG_FINGERPRINT="YOUR_GPG_FINGERPRINT"
export GPG_FINGERPRINT="160A 79E7 0A3B 6CA2 9E52  BD09 66EC 348A 14A3 676B"


# create a tag and push it to GitHub:
git tag -a v0.1.0 -m "First release"
git push origin v0.1.0

# If you don't want to create a tag yet, you can also run GoReleaser without publishing based on the latest commit by using the --snapshot flag:
goreleaser release --snapshot

# run GoReleaser at the root of your repository:

goreleaser release

# That's all it takes!
# Dry run
# Verify dependencies
# check if you have every tool needed for the current configuration:

goreleaser healthcheck

# Build-only Mode
# Build command will build the project:
goreleaser build

# This can be useful as part of CI pipelines to verify the project builds without errors for all build targets.
# Release Flags
# Use the --skip=publish flag to skip publishing:
goreleaser release --skip=publish

# You can check the command line usage help here or with:
# goreleaser --help
