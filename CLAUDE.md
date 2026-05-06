# Claude Code Instructions

## Environment

- **Go binary**: `/Users/ankit.sarda/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.25.5.darwin-arm64/bin/go`

## Git Workflow Rules

1. **Branch creation**: Before creating any branch, always ask the user for the GUS work item ID (e.g. `W-12345678`). Use it as the branch name prefix: `W-12345678-short-description`.

2. **Commit messages**: Every commit message must start with the GUS work item ID in the format `@W-12345678: <message>`.
