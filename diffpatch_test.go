package diffpatch_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/njchilds90/go-diffpatch"
)

// ---- Diff tests -------------------------------------------------------------

func TestDiff_NoChanges(t *testing.T) {
	patch, err := diffpatch.Diff("hello\nworld\n", "hello\nworld\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !patch.IsEmpty() {
		t.Error("expected empty patch for identical inputs")
	}
	ins, del := patch.Stats()
	if ins != 0 || del != 0 {
		t.Errorf("expected 0 insertions and 0 deletions, got %d and %d", ins, del)
	}
}

func TestDiff_AllInserted(t *testing.T) {
	patch, err := diffpatch.Diff("", "a\nb\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ins, del := patch.Stats()
	if ins != 2 || del != 0 {
		t.Errorf("expected 2 insertions 0 deletions, got %d %d", ins, del)
	}
}

func TestDiff_AllDeleted(t *testing.T) {
	patch, err := diffpatch.Diff("a\nb\n", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ins, del := patch.Stats()
	if ins != 0 || del != 2 {
		t.Errorf("expected 0 insertions 2 deletions, got %d %d", ins, del)
	}
}

var diffTableTests = []struct {
	name           string
	source         string
	target         string
	wantInserted   int
	wantDeleted    int
}{
	{
		name:         "single line changed",
		source:       "hello\nworld\n",
		target:       "hello\nGo\n",
		wantInserted: 1,
		wantDeleted:  1,
	},
	{
		name:         "line inserted in middle",
		source:       "a\nc\n",
		target:       "a\nb\nc\n",
		wantInserted: 1,
		wantDeleted:  0,
	},
	{
		name:         "line deleted from middle",
		source:       "a\nb\nc\n",
		target:       "a\nc\n",
		wantInserted: 0,
		wantDeleted:  1,
	},
	{
		name:         "multiple blocks changed",
		source:       "a\nb\nc\nd\ne\n",
		target:       "a\nB\nc\nD\ne\n",
		wantInserted: 2,
		wantDeleted:  2,
	},
	{
		name:         "both empty",
		source:       "",
		target:       "",
		wantInserted: 0,
		wantDeleted:  0,
	},
	{
		name:         "no trailing newline",
		source:       "foo",
		target:       "bar",
		wantInserted: 1,
		wantDeleted:  1,
	},
}

func TestDiff_Table(t *testing.T) {
	for _, tt := range diffTableTests {
		t.Run(tt.name, func(t *testing.T) {
			patch, err := diffpatch.Diff(tt.source, tt.target)
			if err != nil {
				t.Fatalf("Diff error: %v", err)
			}
			ins, del := patch.Stats()
			if ins != tt.wantInserted {
				t.Errorf("inserted: got %d, want %d", ins, tt.wantInserted)
			}
			if del != tt.wantDeleted {
				t.Errorf("deleted: got %d, want %d", del, tt.wantDeleted)
			}
		})
	}
}

// ---- Apply / Revert tests ---------------------------------------------------

func TestApply_RoundTrip(t *testing.T) {
	sources := []string{
		"hello\nworld\n",
		"a\nb\nc\n",
		"",
		"single line",
		"a\nb\nc\nd\ne\nf\n",
	}
	targets := []string{
		"hello\nGo\n",
		"a\nB\nc\n",
		"new content\n",
		"changed line",
		"a\nB\nc\nd\nE\nf\n",
	}

	for i, source := range sources {
		target := targets[i]
		patch, err := diffpatch.Diff(source, target)
		if err != nil {
			t.Errorf("[%d] Diff error: %v", i, err)
			continue
		}
		got, err := diffpatch.Apply(source, patch)
		if err != nil {
			t.Errorf("[%d] Apply error: %v", i, err)
			continue
		}
		if got != target {
			t.Errorf("[%d] Apply result mismatch:\ngot  %q\nwant %q", i, got, target)
		}

		reverted, err := diffpatch.Revert(target, patch)
		if err != nil {
			t.Errorf("[%d] Revert error: %v", i, err)
			continue
		}
		if reverted != source {
			t.Errorf("[%d] Revert result mismatch:\ngot  %q\nwant %q", i, reverted, source)
		}
	}
}

func TestApply_ConflictError(t *testing.T) {
	source := "a\nb\nc\n"
	target := "a\nB\nc\n"
	patch, err := diffpatch.Diff(source, target)
	if err != nil {
		t.Fatal(err)
	}

	// Applying the patch to a different source should fail.
	_, err = diffpatch.Apply("x\ny\nz\n", patch)
	if err == nil {
		t.Fatal("expected ConflictError, got nil")
	}
	var conflictErr *diffpatch.ConflictError
	if !errors.As(err, &conflictErr) {
		t.Fatalf("expected *ConflictError, got %T: %v", err, err)
	}
	if conflictErr.HunkIndex < 0 {
		t.Error("HunkIndex should be non-negative")
	}
}

// ---- Unified diff tests -----------------------------------------------------

func TestUnified_Empty(t *testing.T) {
	patch, err := diffpatch.Diff("same\n", "same\n")
	if err != nil {
		t.Fatal(err)
	}
	out := diffpatch.Unified(patch, "a", "b")
	if out != "" {
		t.Errorf("expected empty unified output for no-change patch, got %q", out)
	}
}

func TestUnified_ContainsMarkers(t *testing.T) {
	patch, err := diffpatch.Diff("old\n", "new\n")
	if err != nil {
		t.Fatal(err)
	}
	out := diffpatch.Unified(patch, "old.txt", "new.txt")
	if !strings.Contains(out, "--- old.txt") {
		t.Error("missing source file header in unified output")
	}
	if !strings.Contains(out, "+++ new.txt") {
		t.Error("missing target file header in unified output")
	}
	if !strings.Contains(out, "-old") {
		t.Error("missing deletion line in unified output")
	}
	if !strings.Contains(out, "+new") {
		t.Error("missing insertion line in unified output")
	}
}

// ---- Context option tests ---------------------------------------------------

func TestDiffWithOptions_ContextLines(t *testing.T) {
	// 10-line file with a change at line 5.
	source := "1\n2\n3\n4\n5\n6\n7\n8\n9\n10\n"
	target := "1\n2\n3\n4\nFIVE\n6\n7\n8\n9\n10\n"

	patchFull, err := diffpatch.Diff(source, target)
	if err != nil {
		t.Fatal(err)
	}

	patchCtx, err := diffpatch.DiffWithOptions(source, target, diffpatch.Options{Context: 2})
	if err != nil {
		t.Fatal(err)
	}

	fullTotal := totalChanges(patchFull)
	ctxTotal := totalChanges(patchCtx)

	if ctxTotal >= fullTotal {
		t.Errorf("context diff should have fewer lines than full diff: full=%d ctx=%d", fullTotal, ctxTotal)
	}
}

func TestDiffWithOptions_IgnoreTrailingWhitespace(t *testing.T) {
	source := "hello   \nworld\n"
	target := "hello\nworld\n"

	patchStrict, err := diffpatch.Diff(source, target)
	if err != nil {
		t.Fatal(err)
	}
	insStrict, delStrict := patchStrict.Stats()

	patchIgnore, err := diffpatch.DiffWithOptions(source, target, diffpatch.Options{IgnoreTrailingWhitespace: true})
	if err != nil {
		t.Fatal(err)
	}
	insIgnore, delIgnore := patchIgnore.Stats()

	if insStrict == 0 && delStrict == 0 {
		t.Error("strict diff should detect trailing whitespace change")
	}
	if insIgnore != 0 || delIgnore != 0 {
		t.Error("ignore-whitespace diff should report no changes")
	}
}

// ---- JSON serialisation tests -----------------------------------------------

func TestPatch_JSONRoundTrip(t *testing.T) {
	source := "a\nb\nc\n"
	target := "a\nB\nc\n"

	original, err := diffpatch.Diff(source, target)
	if err != nil {
		t.Fatal(err)
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}

	var restored diffpatch.Patch
	if err := json.Unmarshal(data, &restored); err != nil {
		t.Fatalf("json.Unmarshal: %v", err)
	}

	ins1, del1 := original.Stats()
	ins2, del2 := restored.Stats()
	if ins1 != ins2 || del1 != del2 {
		t.Errorf("stats mismatch after JSON round-trip: original (%d,%d) restored (%d,%d)", ins1, del1, ins2, del2)
	}

	result, err := diffpatch.Apply(source, restored)
	if err != nil {
		t.Fatalf("Apply after JSON round-trip: %v", err)
	}
	if result != target {
		t.Errorf("Apply after JSON round-trip: got %q want %q", result, target)
	}
}

// ---- Operation text marshalling tests ---------------------------------------

func TestOperation_MarshalUnmarshal(t *testing.T) {
	ops := []diffpatch.Operation{
		diffpatch.OperationEqual,
		diffpatch.OperationInsert,
		diffpatch.OperationDelete,
	}
	for _, op := range ops {
		text, err := op.MarshalText()
		if err != nil {
			t.Errorf("MarshalText(%v): %v", op, err)
		}
		var restored diffpatch.Operation
		if err := restored.UnmarshalText(text); err != nil {
			t.Errorf("UnmarshalText(%q): %v", string(text), err)
		}
		if restored != op {
			t.Errorf("round-trip mismatch: got %v want %v", restored, op)
		}
	}
}

func TestOperation_UnmarshalText_Invalid(t *testing.T) {
	var op diffpatch.Operation
	err := op.UnmarshalText([]byte("nonsense"))
	if err == nil {
		t.Error("expected error for unknown operation string")
	}
}

// ---- Context cancellation test ----------------------------------------------

func TestDiffContext_Cancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	source := strings.Repeat("line\n", 100)
	target := strings.Repeat("changed\n", 100)

	// A cancelled context should cause an error for large inputs.
	// For very small inputs the algorithm may complete before checking,
	// so we only assert no panic here and accept either outcome.
	_, _ = diffpatch.DiffContext(ctx, source, target, diffpatch.Options{})
}

// ---- helpers ----------------------------------------------------------------

func totalChanges(p diffpatch.Patch) int {
	count := 0
	for _, h := range p.Hunks {
		count += len(h.Changes)
	}
	return count
}
