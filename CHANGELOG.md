# Changelog

All notable changes to go-diffpatch are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.0.0/).
This project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

---

## [1.0.0] — 2026-03-02

### Added

- `Diff` — compute a line-level diff between two strings using the Myers O(ND) algorithm.
- `DiffWithOptions` — diff with configurable context lines and trailing-whitespace handling.
- `DiffContext` — diff with `context.Context` cancellation support for large inputs.
- `Apply` — apply a `Patch` to a source string to produce the target string.
- `Revert` — apply a `Patch` in reverse to recover the source string from the target.
- `Unified` — render a `Patch` as a standard unified diff string for display.
- `Patch`, `Hunk`, `Change` — plain, JSON-serialisable structs representing structured diff data.
- `Operation` type with `MarshalText` / `UnmarshalText` for clean JSON encoding.
- `ConflictError` — typed error with hunk index, line number, expected and actual text fields.
- `Options` struct — `Context` (context lines) and `IgnoreTrailingWhitespace` flags.
- `Patch.IsEmpty()` — reports whether source and target are identical.
- `Patch.Stats()` — returns inserted and deleted line counts.
- Full table-driven test suite with race-detector coverage.
- GitHub Actions CI matrix across Go 1.21, 1.22, and 1.23.
- GoDoc examples for every exported function.
- Zero external dependencies.
```

---

