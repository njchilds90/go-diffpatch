# go-diffpatch

[![CI](https://github.com/njchilds90/go-diffpatch/actions/workflows/ci.yml/badge.svg)](https://github.com/njchilds90/go-diffpatch/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/njchilds90/go-diffpatch.svg)](https://pkg.go.dev/github.com/njchilds90/go-diffpatch)
[![Go Report Card](https://goreportcard.com/badge/github.com/njchilds90/go-diffpatch)](https://goreportcard.com/report/github.com/njchilds90/go-diffpatch)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

Compute, serialize, and apply text diffs and patches in pure Go with structured,
machine-readable output.

---

## Why go-diffpatch?

Go's standard library has no diff or patch facility. Third-party options are
either display-only (unified diff strings) or buried inside larger tools. 
**go-diffpatch** fills the gap with a small, focused API that covers the 
complete diff–serialize–apply–revert cycle:

| Need | How |
|---|---|
| Compute what changed | `Diff` / `DiffWithOptions` / `DiffContext` |
| Describe changes as data | `Patch`, `Hunk`, `Change` — plain structs, JSON-ready |
| Apply a patch forward | `Apply` |
| Apply a patch in reverse | `Revert` |
| Display for humans | `Unified` |

**Zero external dependencies.** Pure Go. Race-detector clean.

---

## Installation
```bash
go get github.com/njchilds90/go-diffpatch@latest
```

---

## Quick start
```go
package main

import (
    "fmt"
    "log"

    "github.com/njchilds90/go-diffpatch"
)

func main() {
    source := "hello\nworld\n"
    target := "hello\nGo\n"

    patch, err := diffpatch.Diff(source, target)
    if err != nil {
        log.Fatal(err)
    }

    // Inspect the change.
    inserted, deleted := patch.Stats()
    fmt.Printf("inserted=%d deleted=%d\n", inserted, deleted)

    // Apply the patch to produce the target.
    result, err := diffpatch.Apply(source, patch)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(result) // "hello\nGo\n"

    // Revert the patch to recover the source.
    original, err := diffpatch.Revert(target, patch)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println(original) // "hello\nworld\n"

    // Pretty-print as a unified diff.
    fmt.Println(diffpatch.Unified(patch, "before.txt", "after.txt"))
}
```

---

## Structured output (machine-readable / AI-agent friendly)

Every `Patch` is a plain Go struct that marshals cleanly to JSON:
```go
import "encoding/json"

data, _ := json.Marshal(patch)
// {
//   "hunks": [
//     {
//       "source_start": 1,
//       "target_start": 1,
//       "changes": [
//         {"operation": "delete", "text": "world\n"},
//         {"operation": "insert", "text": "Go\n"}
//       ]
//     }
//   ],
//   "source_line_count": 2,
//   "target_line_count": 2
// }
```

Restore from JSON and apply:
```go
var restored diffpatch.Patch
json.Unmarshal(data, &restored)
result, _ := diffpatch.Apply(source, restored)
```

---

## Context-aware diff (limited context window)
```go
patch, err := diffpatch.DiffWithOptions(source, target, diffpatch.Options{
    Context: 3, // include 3 equal lines around each change
})
```

---

## Ignoring trailing whitespace
```go
patch, err := diffpatch.DiffWithOptions(source, target, diffpatch.Options{
    IgnoreTrailingWhitespace: true,
})
```

---

## Cancellation for large inputs
```go
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

patch, err := diffpatch.DiffContext(ctx, source, target, diffpatch.Options{})
```

---

## Error handling

Conflict errors carry structured information for programmatic handling:
```go
result, err := diffpatch.Apply(source, patch)
if err != nil {
    var conflictErr *diffpatch.ConflictError
    if errors.As(err, &conflictErr) {
        fmt.Printf("Conflict in hunk %d at line %d\n",
            conflictErr.HunkIndex, conflictErr.LineNumber)
        fmt.Printf("Expected: %q\n", conflictErr.Expected)
        fmt.Printf("Got:      %q\n", conflictErr.Got)
    }
}
```

---

## API reference

Full GoDoc: https://pkg.go.dev/github.com/njchilds90/go-diffpatch

| Symbol | Description |
|---|---|
| `Diff(source, target string) (Patch, error)` | Compute a full line-level diff |
| `DiffWithOptions(source, target string, opts Options) (Patch, error)` | Diff with configuration |
| `DiffContext(ctx, source, target string, opts Options) (Patch, error)` | Diff with cancellation |
| `Apply(source string, patch Patch) (string, error)` | Apply patch to source → target |
| `Revert(target string, patch Patch) (string, error)` | Apply patch in reverse → source |
| `Unified(patch Patch, sourceName, targetName string) string` | Unified diff string for display |
| `Patch` | Serialisable patch object |
| `Hunk` | Contiguous block of changes |
| `Change` | Single line change (operation + text) |
| `Operation` | `OperationEqual`, `OperationInsert`, `OperationDelete` |
| `Options` | Controls context lines and whitespace handling |
| `ConflictError` | Structured apply/revert error |

---

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

---

## License

MIT — see [LICENSE](LICENSE).
```

---

