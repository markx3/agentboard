---
name: release
description: Tag and push a new release version. Runs tests, creates a git tag, and pushes to trigger the GitHub Actions release pipeline.
disable-model-invocation: true
---

# Release

Create a new release for agentboard.

## Steps

1. Run tests with race detection:
   ```bash
   go test -race ./...
   ```
   Stop if any test fails.

2. Check current version tags:
   ```bash
   git tag --sort=-v:refname | head -5
   ```

3. Ask the user what version to release (e.g., v0.2.0). Follow semver.

4. Ensure the working tree is clean:
   ```bash
   git status --porcelain
   ```
   If there are uncommitted changes, ask the user to commit or stash first.

5. Create the annotated tag:
   ```bash
   git tag -a <version> -m "Release <version>"
   ```

6. Push the tag to trigger the release pipeline:
   ```bash
   git push origin <version>
   ```

7. Report the release URL: `https://github.com/markx3/agentboard/releases`
