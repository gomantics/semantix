# Create Pull Request

Create a GitHub pull request for the current branch.

## Steps

1. **Check current state** (run in parallel)
   - `git status` - see uncommitted changes
   - `git log --oneline -5` - recent commits
   - `git branch --show-current` - current branch name
   - `git diff main...HEAD --stat` - changes vs main

2. **Handle uncommitted changes**
   - If there are uncommitted changes, ask if I want to commit first

3. **Push branch**
   - Push to origin with tracking: `git push -u origin HEAD`

4. **Create PR with gh CLI**
   - Write PR body to a temporary file first
   - Use `gh pr create --body-file` to read from it
   - Title: concise, imperative mood
   - Body: Summary bullets + Test Plan checklist

## PR Format

```bash
# Write PR body to temp file
cat > /tmp/pr-body.md <<'EOF'
## Summary

- What changed and why

## Test Plan

- [ ] Verification steps
EOF

# Create PR using the file
gh pr create --title "Brief title" --body-file /tmp/pr-body.md

# Clean up
rm /tmp/pr-body.md
```

## Guidelines

- **Title**: Imperative mood (e.g., "Add episode ingestion endpoint")
- **Summary**: 2-5 bullet points describing changes
- **Test Plan**: Checklist of manual verification steps

## Troubleshooting

If you get authentication errors with `gh`, run:

```bash
unset GITHUB_TOKEN
```

Then retry the command. This clears any conflicting token from the environment.
