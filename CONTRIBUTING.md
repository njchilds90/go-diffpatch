# Contributing to go-diffpatch

Thank you for your interest in contributing. This document explains how to get
involved.

---

## Reporting issues

Open a GitHub Issue. Please include:

- Go version (`go version`)
- A minimal, self-contained reproducing example
- The actual and expected output

---

## Proposing changes

1. Open an Issue first to discuss significant changes before writing code.
2. Fork the repository and create a descriptive branch (for example: `feature/binary-diff-support`).
3. Write or update tests. All changes require table-driven tests in `*_test.go` files.
4. Run the full test suite locally before submitting:
```
   go test -race -count=1 ./...
   go vet ./...
```
5. Open a Pull Request against `main`. Describe what changed and why.

---

## Code style

- Follow standard Go idioms and naming conventions.
- All exported symbols must have GoDoc comments.
- No external runtime dependencies — this is a zero-dependency library.
- Keep the public API surface small and composable.

---

## Versioning

This project follows [Semantic Versioning](https://semver.org/).

- Patch releases (1.0.x): bug fixes, no API changes.
- Minor releases (1.x.0): new functionality, fully backwards-compatible.
- Major releases (x.0.0): breaking API changes, discussed in Issues first.

---

## License

By contributing you agree that your contributions will be licensed under the
MIT License.
```

---

## Release and Verification Instructions

### Create the tag and release
```
Step 1 — Tag v1.0.0 via the GitHub UI

1. Open https://github.com/njchilds90/go-diffpatch
2. On the right sidebar, click "Releases"
3. Click "Create a new release"
4. In the "Choose a tag" dropdown, type: v1.0.0
   Then click "Create new tag: v1.0.0 on publish"
5. Set Target to: main
6. Set Release title to: v1.0.0 — Initial Release
7. Paste the following into the description box:

---
## go-diffpatch v1.0.0

Production-ready initial release.

### Highlights

- Compute structured, JSON-serialisable diffs with Myers O(ND) algorithm
- Apply and revert patches with typed ConflictError
- Context-window diffs and trailing-whitespace ignoring
- Zero external dependencies
- Full test suite, race-detector clean, CI across Go 1.21–1.23

See [CHANGELOG.md](CHANGELOG.md) for the complete list of additions.
---

8. Leave "Set as the latest release" checked.
9. Click "Publish release".
```

### Trigger pkg.go.dev indexing
```
Step 2 — Trigger indexing (two options, either works)

Option A (fastest):
  Visit this URL in your browser:
  https://sum.golang.org/lookup/github.com/njchilds90/go-diffpatch@v1.0.0
  Wait ~10 seconds, then visit:
  https://pkg.go.dev/github.com/njchilds90/go-diffpatch

Option B:
  Run from any machine with Go installed:
  GOPROXY=proxy.golang.org go install github.com/njchilds90/go-diffpatch@v1.0.0

After approximately 5–10 minutes, the full GoDoc page will be live at:
https://pkg.go.dev/github.com/njchilds90/go-diffpatch
```

### Semantic versioning guidance
```
v1.0.x  — Bug fixes. No API additions or removals.
v1.x.0  — New exports (new Option fields, new helper functions).
           All existing callers continue to compile unchanged.
v2.0.0  — Breaking changes only. Requires discussion in a GitHub Issue first.
           Module path would change to: github.com/njchilds90/go-diffpatch/v2
