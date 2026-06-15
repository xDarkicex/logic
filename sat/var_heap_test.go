package sat

import (
	"testing"

	"github.com/xDarkicex/memory"
)

func testPool(t *testing.T) *memory.Pool {
	t.Helper()
	pool, err := memory.NewPool(memory.DefaultConfig())
	if err != nil {
		t.Fatalf("NewPool: %v", err)
	}
	t.Cleanup(func() { pool.Free() })
	return pool
}

func TestVarHeapNew(t *testing.T) {
	h := NewVarHeap(8, testPool(t))
	if !h.IsEmpty() {
		t.Error("new heap should be empty")
	}
	if h.Len() != 0 {
		t.Errorf("Len = %d, want 0", h.Len())
	}
}

func TestVarHeapPushMax(t *testing.T) {
	h := NewVarHeap(4, testPool(t))
	h.Update(0, 1.0)
	h.Update(1, 3.0)
	h.Update(2, 2.0)

	if h.Len() != 3 {
		t.Fatalf("Len = %d, want 3", h.Len())
	}
	m := h.Max()
	if m != 1 {
		t.Errorf("Max = %d, want 1 (score 3.0)", m)
	}
}

func TestVarHeapPopMax(t *testing.T) {
	h := NewVarHeap(4, testPool(t))
	h.Update(0, 1.0)
	h.Update(1, 5.0)
	h.Update(2, 3.0)

	first := h.PopMax()
	if first != 1 {
		t.Errorf("first PopMax = %d, want 1", first)
	}
	second := h.PopMax()
	if second != 2 {
		t.Errorf("second PopMax = %d, want 2", second)
	}
	third := h.PopMax()
	if third != 0 {
		t.Errorf("third PopMax = %d, want 0", third)
	}
	if !h.IsEmpty() {
		t.Error("heap should be empty after popping all")
	}
}

func TestVarHeapEmptyPopMax(t *testing.T) {
	h := NewVarHeap(4, testPool(t))
	if got := h.PopMax(); got != -1 {
		t.Errorf("PopMax on empty heap = %d, want -1", got)
	}
}

func TestVarHeapUpdateIncrease(t *testing.T) {
	h := NewVarHeap(4, testPool(t))
	h.Update(0, 1.0)
	h.Update(1, 2.0)
	h.Update(2, 3.0)

	// Increase 0 above all others
	h.Update(0, 5.0)
	m := h.Max()
	if m != 0 {
		t.Errorf("after increase, Max = %d, want 0", m)
	}
}

func TestVarHeapUpdateDecrease(t *testing.T) {
	h := NewVarHeap(4, testPool(t))
	h.Update(0, 5.0)
	h.Update(1, 2.0)
	h.Update(2, 3.0)

	// Decrease 0 below others
	h.Update(0, 1.0)
	m := h.Max()
	if m != 2 {
		t.Errorf("after decrease, Max = %d, want 2", m)
	}
}

func TestVarHeapContains(t *testing.T) {
	h := NewVarHeap(4, testPool(t))
	if h.Contains(0) {
		t.Error("Contains(0) before insert should be false")
	}
	h.Update(0, 1.0)
	if !h.Contains(0) {
		t.Error("Contains(0) after insert should be true")
	}
	h.Pop(0)
	if h.Contains(0) {
		t.Error("Contains(0) after pop should be false")
	}
}

func TestVarHeapScore(t *testing.T) {
	h := NewVarHeap(4, testPool(t))
	if s := h.Score(0); s != 0 {
		t.Errorf("Score of unregistered var = %v, want 0", s)
	}
	h.Update(0, 3.5)
	if s := h.Score(0); s != 3.5 {
		t.Errorf("Score = %v, want 3.5", s)
	}
}

func TestVarHeapRebuild(t *testing.T) {
	h := NewVarHeap(8, testPool(t))
	h.Update(0, 1.0)
	h.Update(1, 2.0)
	h.Update(2, 3.0)
	h.PopMax() // removes 2
	h.PopMax() // removes 1

	// Rebuild with different set
	h.Rebuild([]int{3, 4, 5})
	if h.Len() != 3 {
		t.Fatalf("after Rebuild Len = %d, want 3", h.Len())
	}
	// Scores are 0, so any ordering is fine; just check they're all in
	for _, idx := range []int{3, 4, 5} {
		if !h.Contains(idx) {
			t.Errorf("idx %d should be in heap after rebuild", idx)
		}
	}
}

func TestVarHeapRescale(t *testing.T) {
	h := NewVarHeap(4, testPool(t))
	h.Update(0, 1e100)
	h.Update(1, 2e100)

	h.Rescale(1e-100)
	if s := h.Score(1); s < 1.9 || s > 2.1 {
		t.Errorf("after rescale, Score(1) = %v, want ~2.0", s)
	}
	if s := h.Score(0); s < 0.9 || s > 1.1 {
		t.Errorf("after rescale, Score(0) = %v, want ~1.0", s)
	}
}

func TestVarHeapReset(t *testing.T) {
	h := NewVarHeap(4, testPool(t))
	h.Update(0, 1.0)
	h.Update(1, 2.0)
	h.Reset()

	if !h.IsEmpty() {
		t.Error("heap should be empty after Reset")
	}
	if h.Contains(0) || h.Contains(1) {
		t.Error("no variables should be in heap after Reset")
	}
	if h.Score(0) != 0 || h.Score(1) != 0 {
		t.Error("scores should be zero after Reset")
	}
}

func TestVarHeapGrow(t *testing.T) {
	h := NewVarHeap(2, testPool(t))
	// Insert many variables to force grow
	for i := 0; i < 100; i++ {
		h.Update(i, float64(i))
	}
	if h.Len() != 100 {
		t.Errorf("Len = %d, want 100", h.Len())
	}
	if h.Max() != 99 {
		t.Errorf("Max = %d, want 99", h.Max())
	}
}

func TestVarHeapPop(t *testing.T) {
	h := NewVarHeap(8, testPool(t))
	h.Update(0, 1.0)
	h.Update(1, 3.0)
	h.Update(2, 2.0)
	h.Update(3, 4.0)

	h.Pop(1) // middle element
	if h.Contains(1) {
		t.Error("popped element should not be contained")
	}
	if h.Len() != 3 {
		t.Errorf("Len = %d, want 3", h.Len())
	}
	// Heap should still be valid: max should be 3
	if h.Max() != 3 {
		t.Errorf("after Pop, Max = %d, want 3", h.Max())
	}
}

func TestVarHeapPushDuplicate(t *testing.T) {
	h := NewVarHeap(4, testPool(t))
	h.Update(0, 1.0)
	h.Push(0) // should be no-op
	if h.Len() != 1 {
		t.Errorf("Len = %d, want 1 (push duplicate)", h.Len())
	}
}

func TestVarHeapSequence(t *testing.T) {
	// Simulate a VSIDS-like usage pattern
	h := NewVarHeap(8, testPool(t))

	// Initialize variables with varying scores
	scores := []float64{3.0, 1.0, 4.0, 2.0, 5.0, 0.5, 3.5, 1.5}
	for i, s := range scores {
		h.Update(i, s)
	}

	// Pop top 3
	top := []int{h.PopMax(), h.PopMax(), h.PopMax()}
	expected := []int{4, 2, 6} // scores: 5.0, 4.0, 3.5
	for i, want := range expected {
		if top[i] != want {
			t.Errorf("pop[%d] = %d, want %d", i, top[i], want)
		}
	}

	// Bump score of var 0 (was 3.0, now 6.0)
	h.Update(0, 6.0)
	if h.Max() != 0 {
		t.Errorf("after bump, Max = %d, want 0", h.Max())
	}
}

func TestVarHeapSingleElement(t *testing.T) {
	h := NewVarHeap(4, testPool(t))
	h.Update(0, 42.0)

	if h.Len() != 1 {
		t.Errorf("Len = %d, want 1", h.Len())
	}
	if h.Max() != 0 {
		t.Errorf("Max = %d, want 0", h.Max())
	}

	popped := h.PopMax()
	if popped != 0 {
		t.Errorf("PopMax = %d, want 0", popped)
	}
	if !h.IsEmpty() {
		t.Error("heap should be empty after pop")
	}
}

func TestVarHeapEqualScores(t *testing.T) {
	h := NewVarHeap(4, testPool(t))
	h.Update(0, 1.0)
	h.Update(1, 1.0)
	h.Update(2, 1.0)

	// Max should be any of them; heap order among equals is arbitrary
	m := h.Max()
	if m < 0 || m > 2 {
		t.Errorf("Max = %d, want 0, 1, or 2", m)
	}
	// Popping all should work
	count := 0
	for !h.IsEmpty() {
		h.PopMax()
		count++
	}
	if count != 3 {
		t.Errorf("popped %d elements, want 3", count)
	}
}

func TestVarHeapZeroCapacity(t *testing.T) {
	h := NewVarHeap(0, testPool(t))
	if h.Len() != 0 {
		t.Errorf("Len = %d, want 0", h.Len())
	}
	// Should be able to push even with 0 initial capacity (triggers grow)
	h.Update(0, 1.0)
	if h.Len() != 1 {
		t.Errorf("Len = %d after push, want 1", h.Len())
	}
}
