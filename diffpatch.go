// Package diffpatch computes line-level diffs between two texts, produces
// structured patch objects, and applies or reverts those patches — all with
// zero external dependencies.
//
// The core algorithm is Myers' O(ND) difference algorithm, the same one used
// by Git. All exported types are plain Go structs designed to be marshalled to
// JSON or processed programmatically by automated agents.
//
// # Quick Start
//
//	patch, err := diffpatch.Diff("hello\nworld\n", "hello\nGo\n")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	result, err := diffpatch.Apply("hello\nworld\n", patch)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println(result) // "hello\nGo\n"
package diffpatch

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Operation describes the kind of change represented by a single hunk line.
type Operation int

const (
	// OperationEqual means the line is unchanged between the source and target.
	OperationEqual Operation = iota
	// OperationInsert means the line was added in the target.
	OperationInsert
	// OperationDelete means the line was removed from the source.
	OperationDelete
)

// String returns a human-readable label for the operation.
func (o Operation) String() string {
	switch o {
	case OperationEqual:
		return "equal"
	case OperationInsert:
		return "insert"
	case OperationDelete:
		return "delete"
	default:
		return "unknown"
	}
}

// MarshalText implements encoding.TextMarshaler so that Operation serialises
// as a readable string in JSON and other text-based formats.
func (o Operation) MarshalText() ([]byte, error) {
	return []byte(o.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (o *Operation) UnmarshalText(data []byte) error {
	switch string(data) {
	case "equal":
		*o = OperationEqual
	case "insert":
		*o = OperationInsert
	case "delete":
		*o = OperationDelete
	default:
		return fmt.Errorf("diffpatch: unknown operation %q", string(data))
	}
	return nil
}

// Change represents a single line-level change within a diff.
type Change struct {
	// Operation is the kind of change.
	Operation Operation `json:"operation"`
	// Text is the line content, including its trailing newline if present.
	Text string `json:"text"`
}

// Hunk is a contiguous block of changes (equal, insert, or delete lines).
// A Patch is composed of one or more Hunks.
type Hunk struct {
	// SourceStart is the zero-based line index in the source where this hunk begins.
	SourceStart int `json:"source_start"`
	// TargetStart is the zero-based line index in the target where this hunk begins.
	TargetStart int `json:"target_start"`
	// Changes is the ordered list of line-level operations in this hunk.
	Changes []Change `json:"changes"`
}

// Patch is a structured, serialisable description of the difference between
// two texts. It can be applied to the source to produce the target, or
// reverted on the target to recover the source.
type Patch struct {
	// Hunks contains every contiguous group of changes.
	Hunks []Hunk `json:"hunks"`
	// SourceLineCount is the total number of lines in the source text.
	SourceLineCount int `json:"source_line_count"`
	// TargetLineCount is the total number of lines in the target text.
	TargetLineCount int `json:"target_line_count"`
}

// IsEmpty reports whether the patch contains no changes (source equals target).
func (p Patch) IsEmpty() bool {
	for _, h := range p.Hunks {
		for _, c := range h.Changes {
			if c.Operation != OperationEqual {
				return false
			}
		}
	}
	return true
}

// Stats returns the number of inserted and deleted lines in the patch.
func (p Patch) Stats() (inserted, deleted int) {
	for _, h := range p.Hunks {
		for _, c := range h.Changes {
			switch c.Operation {
			case OperationInsert:
				inserted++
			case OperationDelete:
				deleted++
			}
		}
	}
	return inserted, deleted
}

// ConflictError is returned by Apply or Revert when the patch cannot be
// applied cleanly because the target text does not match what the patch expects.
type ConflictError struct {
	// HunkIndex is the zero-based index of the hunk that caused the conflict.
	HunkIndex int
	// LineNumber is the line in the target text where the mismatch was detected.
	LineNumber int
	// Expected is the text the patch expected to find.
	Expected string
	// Got is the text that was actually found.
	Got string
}

func (e *ConflictError) Error() string {
	return fmt.Sprintf(
		"diffpatch: conflict in hunk %d at line %d: expected %q but got %q",
		e.HunkIndex, e.LineNumber, e.Expected, e.Got,
	)
}

// ErrEmptyPatch is returned when an operation requires a non-empty patch but
// an empty one was provided.
var ErrEmptyPatch = errors.New("diffpatch: patch is empty")

// Options configures the behaviour of Diff.
type Options struct {
	// Context is the number of equal lines to include around each changed block.
	// The default (zero value) includes all lines (a full diff).
	// Set to 3 for a standard unified-diff style context window.
	Context int

	// IgnoreTrailingWhitespace strips trailing spaces and tabs from each line
	// before comparison. The original content is preserved in the patch.
	IgnoreTrailingWhitespace bool
}

// Diff computes the line-level difference between source and target and returns
// a Patch describing every change. The context.Context argument is used only
// for cancellation of extremely large inputs; pass context.Background() for
// normal use.
//
//	patch, err := diffpatch.Diff("a\nb\nc\n", "a\nB\nc\n")
func Diff(source, target string) (Patch, error) {
	return DiffContext(context.Background(), source, target, Options{})
}

// DiffWithOptions computes the diff using the supplied Options.
//
//	patch, err := diffpatch.DiffWithOptions("a\nb\n", "a\nB\n", diffpatch.Options{Context: 3})
func DiffWithOptions(source, target string, opts Options) (Patch, error) {
	return DiffContext(context.Background(), source, target, opts)
}

// DiffContext is the full-featured variant that accepts a context for
// cancellation and a complete Options struct.
func DiffContext(ctx context.Context, source, target string, opts Options) (Patch, error) {
	sourceLines := splitLines(source)
	targetLines := splitLines(target)

	cmpSource := sourceLines
	cmpTarget := targetLines
	if opts.IgnoreTrailingWhitespace {
		cmpSource = trimTrailing(sourceLines)
		cmpTarget = trimTrailing(targetLines)
	}

	editScript, err := myersDiff(ctx, cmpSource, cmpTarget)
	if err != nil {
		return Patch{}, err
	}

	changes := buildChanges(editScript, sourceLines, targetLines)
	hunks := buildHunks(changes, opts.Context)

	return Patch{
		Hunks:           hunks,
		SourceLineCount: len(sourceLines),
		TargetLineCount: len(targetLines),
	}, nil
}

// Apply applies the patch to source and returns the patched text. It returns
// a ConflictError if the source does not match what the patch expects.
//
//	result, err := diffpatch.Apply("hello\nworld\n", patch)
func Apply(source string, patch Patch) (string, error) {
	return applyPatch(source, patch, false)
}

// Revert applies the patch in reverse: given the target text it recovers the
// source. It returns a ConflictError if the target does not match expectations.
//
//	original, err := diffpatch.Revert("hello\nGo\n", patch)
func Revert(target string, patch Patch) (string, error) {
	return applyPatch(target, patch, true)
}

// Unified returns a classic unified-diff string representation of the patch,
// suitable for display in terminals or code review tools.
//
//	fmt.Println(diffpatch.Unified(patch, "before.txt", "after.txt"))
func Unified(patch Patch, sourceName, targetName string) string {
	if patch.IsEmpty() {
		return ""
	}
	var builder strings.Builder
	fmt.Fprintf(&builder, "--- %s\n+++ %s\n", sourceName, targetName)
	for _, hunk := range patch.Hunks {
		sourceCount, targetCount := hunkLineCounts(hunk)
		fmt.Fprintf(&builder, "@@ -%d,%d +%d,%d @@\n",
			hunk.SourceStart+1, sourceCount,
			hunk.TargetStart+1, targetCount,
		)
		for _, c := range hunk.Changes {
			switch c.Operation {
			case OperationEqual:
				builder.WriteString(" ")
			case OperationInsert:
				builder.WriteString("+")
			case OperationDelete:
				builder.WriteString("-")
			}
			builder.WriteString(c.Text)
			if !strings.HasSuffix(c.Text, "\n") {
				builder.WriteString("\n")
			}
		}
	}
	return builder.String()
}

// ---- internal helpers -------------------------------------------------------

func splitLines(text string) []string {
	if text == "" {
		return nil
	}
	lines := strings.SplitAfter(text, "\n")
	// SplitAfter on "a\nb\n" yields ["a\n","b\n",""] — drop trailing empty.
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	return lines
}

func trimTrailing(lines []string) []string {
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = strings.TrimRight(l, " \t")
	}
	return out
}

// editOp is an internal enum used during the Myers algorithm.
type editOp int

const (
	editEqual  editOp = iota
	editInsert editOp = iota
	editDelete editOp = iota
)

type editEntry struct {
	op          editOp
	sourceIndex int
	targetIndex int
}

// myersDiff runs the Myers O(ND) diff algorithm and returns a flat list of
// (op, sourceIndex, targetIndex) triples in order.
func myersDiff(ctx context.Context, source, target []string) ([]editEntry, error) {
	sourceLen := len(source)
	targetLen := len(target)

	if sourceLen == 0 && targetLen == 0 {
		return nil, nil
	}

	max := sourceLen + targetLen
	// v holds the furthest-reaching D-path endpoints keyed by diagonal k.
	v := make([]int, 2*max+1)
	// trace stores a snapshot of v after each D step for backtracking.
	trace := make([][]int, 0, max)

	for d := 0; d <= max; d++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}

		snapshot := make([]int, len(v))
		copy(snapshot, v)
		trace = append(trace, snapshot)

		for k := -d; k <= d; k += 2 {
			var x int
			idx := k + max
			if k == -d || (k != d && v[idx-1] < v[idx+1]) {
				x = v[idx+1]
			} else {
				x = v[idx-1] + 1
			}
			y := x - k
			for x < sourceLen && y < targetLen && source[x] == target[y] {
				x++
				y++
			}
			v[idx] = x
			if x >= sourceLen && y >= targetLen {
				return backtrack(trace, source, target, max), nil
			}
		}
	}
	return backtrack(trace, source, target, max), nil
}

func backtrack(trace [][]int, source, target []string, offset int) []editEntry {
	x := len(source)
	y := len(target)
	var entries []editEntry

	for d := len(trace) - 1; d >= 0 && (x > 0 || y > 0); d-- {
		v := trace[d]
		k := x - y
		idx := k + offset

		var prevK int
		if k == -d || (k != d && v[idx-1] < v[idx+1]) {
			prevK = k + 1
		} else {
			prevK = k - 1
		}
		prevX := v[prevK+offset]
		prevY := prevX - prevK

		for x > prevX && y > prevY {
			entries = append(entries, editEntry{editEqual, x - 1, y - 1})
			x--
			y--
		}
		if d > 0 {
			if x == prevX {
				entries = append(entries, editEntry{editInsert, x, y - 1})
				y--
			} else {
				entries = append(entries, editEntry{editDelete, x - 1, y})
				x--
			}
		}
	}

	// Reverse to get source-order.
	for left, right := 0, len(entries)-1; left < right; left, right = left+1, right-1 {
		entries[left], entries[right] = entries[right], entries[left]
	}
	return entries
}

func buildChanges(entries []editEntry, source, target []string) []Change {
	changes := make([]Change, 0, len(entries))
	for _, e := range entries {
		switch e.op {
		case editEqual:
			changes = append(changes, Change{OperationEqual, source[e.sourceIndex]})
		case editInsert:
			changes = append(changes, Change{OperationInsert, target[e.targetIndex]})
		case editDelete:
			changes = append(changes, Change{OperationDelete, source[e.sourceIndex]})
		}
	}
	return changes
}

func buildHunks(changes []Change, contextLines int) []Hunk {
	if len(changes) == 0 {
		return nil
	}

	// If contextLines is zero we use "infinite" context (all lines in one hunk).
	unlimited := contextLines <= 0

	var hunks []Hunk
	sourcePos := 0
	targetPos := 0

	// Find ranges of non-equal changes and expand by contextLines each side.
	i := 0
	for i < len(changes) {
		// Skip equal lines until we hit a change.
		if changes[i].Operation == OperationEqual {
			sourcePos++
			targetPos++
			i++
			continue
		}

		// Found a change. Determine hunk start (rewind by contextLines).
		hunkStart := i
		if !unlimited {
			hunkStart = max(0, i-contextLines)
		} else {
			hunkStart = 0
		}

		// Adjust sourcePos/targetPos back.
		srcStart := sourcePos
		tgtStart := targetPos
		for j := i - 1; j >= hunkStart; j-- {
			if changes[j].Operation != OperationInsert {
				srcStart--
			}
			if changes[j].Operation != OperationDelete {
				tgtStart--
			}
		}

		// Advance forward, absorbing changes and context.
		j := i
		for j < len(changes) {
			if changes[j].Operation != OperationEqual {
				// Keep consuming until we run out of changes or see more than
				// contextLines equal lines in a row (for limited context).
				j++
				continue
			}
			if unlimited {
				j++
				continue
			}
			// Count consecutive equal lines.
			equalRun := 0
			k := j
			for k < len(changes) && changes[k].Operation == OperationEqual {
				k++
				equalRun++
			}
			if k >= len(changes) || equalRun > 2*contextLines {
				// End of changes or enough separation for a new hunk.
				j += min(contextLines, equalRun)
				break
			}
			j = k
		}

		hunkChanges := make([]Change, j-hunkStart)
		copy(hunkChanges, changes[hunkStart:j])

		hunks = append(hunks, Hunk{
			SourceStart: srcStart,
			TargetStart: tgtStart,
			Changes:     hunkChanges,
		})

		// Advance main cursor past this hunk.
		for ; i < j; i++ {
			if changes[i].Operation != OperationInsert {
				sourcePos++
			}
			if changes[i].Operation != OperationDelete {
				targetPos++
			}
		}
	}

	if unlimited && len(changes) > 0 {
		// Rebuild as a single hunk containing all changes.
		hunks = []Hunk{{SourceStart: 0, TargetStart: 0, Changes: changes}}
	}

	return hunks
}

func applyPatch(text string, patch Patch, revert bool) (string, error) {
	lines := splitLines(text)
	var output []string
	pos := 0 // current position in lines

	for hunkIndex, hunk := range patch.Hunks {
		start := hunk.SourceStart
		if revert {
			start = hunk.TargetStart
		}

		// Copy unchanged lines up to the hunk start.
		for pos < start {
			if pos >= len(lines) {
				return "", &ConflictError{
					HunkIndex:  hunkIndex,
					LineNumber: pos,
					Expected:   "(line in source)",
					Got:        "(end of input)",
				}
			}
			output = append(output, lines[pos])
			pos++
		}

		for changeIndex, change := range hunk.Changes {
			switch {
			case change.Operation == OperationEqual:
				if pos >= len(lines) {
					return "", &ConflictError{
						HunkIndex:  hunkIndex,
						LineNumber: pos,
						Expected:   change.Text,
						Got:        "(end of input)",
					}
				}
				if lines[pos] != change.Text {
					return "", &ConflictError{
						HunkIndex:  hunkIndex,
						LineNumber: changeIndex,
						Expected:   change.Text,
						Got:        lines[pos],
					}
				}
				output = append(output, lines[pos])
				pos++

			case !revert && change.Operation == OperationDelete:
				if pos >= len(lines) || lines[pos] != change.Text {
					got := "(end of input)"
					if pos < len(lines) {
						got = lines[pos]
					}
					return "", &ConflictError{
						HunkIndex:  hunkIndex,
						LineNumber: changeIndex,
						Expected:   change.Text,
						Got:        got,
					}
				}
				pos++ // consume the deleted line, do not emit

			case !revert && change.Operation == OperationInsert:
				output = append(output, change.Text)

			case revert && change.Operation == OperationInsert:
				if pos >= len(lines) || lines[pos] != change.Text {
					got := "(end of input)"
					if pos < len(lines) {
						got = lines[pos]
					}
					return "", &ConflictError{
						HunkIndex:  hunkIndex,
						LineNumber: changeIndex,
						Expected:   change.Text,
						Got:        got,
					}
				}
				pos++

			case revert && change.Operation == OperationDelete:
				output = append(output, change.Text)
			}
		}
	}

	// Append any remaining lines after the last hunk.
	output = append(output, lines[pos:]...)
	return strings.Join(output, ""), nil
}

func hunkLineCounts(hunk Hunk) (sourceCount, targetCount int) {
	for _, c := range hunk.Changes {
		if c.Operation != OperationInsert {
			sourceCount++
		}
		if c.Operation != OperationDelete {
			targetCount++
		}
	}
	return
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
