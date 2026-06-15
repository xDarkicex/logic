package modal

import (
	"testing"

	"github.com/xDarkicex/memory"
)

func newDegeneralizer(t *testing.T) *Degeneralizer {
	t.Helper()
	pool, err := memory.NewPool(memory.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Reset)
	return NewDegeneralizer(pool)
}

func TestDegenZeroSets(t *testing.T) {
	dg := newDegeneralizer(t)
	edges := []struct{ Src, Dst int; Visited []int }{
		{0, 1, nil},
		{1, 0, nil},
	}
	newEdges, lc := dg.Degeneralize(0, 2, edges)
	if lc != 1 {
		t.Errorf("zero sets: levelCount=%d, want 1", lc)
	}
	if len(newEdges) != 2 {
		t.Errorf("zero sets: got %d edges, want 2", len(newEdges))
	}
	for _, e := range newEdges {
		if !e.Accept {
			t.Error("zero sets: all edges should be accepting")
		}
	}
}

func TestDegenOneSet(t *testing.T) {
	dg := newDegeneralizer(t)
	// 2 states, 1 acceptance set, 2 edges.
	// Edge 0→1 visits set {0}, edge 1→0 visits nothing.
	edges := []struct{ Src, Dst int; Visited []int }{
		{0, 1, []int{0}},
		{1, 0, nil},
	}
	newEdges, lc := dg.Degeneralize(1, 2, edges)
	if lc != 1 {
		t.Errorf("one set: levelCount=%d, want 1", lc)
	}
	// Each original edge produces 1 degeneralized edge (1 level copy).
	if len(newEdges) < 2 {
		t.Errorf("one set: got %d edges, want at least 2", len(newEdges))
	}
	// Edge 0→1 visits set 0 → acceptance.
	hasAcc := false
	for _, e := range newEdges {
		if e.Accept {
			hasAcc = true
		}
	}
	if !hasAcc {
		t.Error("one set: should have an accepting edge")
	}
}

func TestDegenTwoSets(t *testing.T) {
	dg := newDegeneralizer(t)
	// 2 states, 2 acceptance sets.
	edges := []struct{ Src, Dst int; Visited []int }{
		{0, 1, []int{0}},     // visits set 0
		{1, 0, []int{1}},     // visits set 1
	}
	newEdges, lc := dg.Degeneralize(2, 2, edges)
	if lc != 2 {
		t.Errorf("two sets: levelCount=%d, want 2", lc)
	}
	// 2 original states × 2 levels = 4 degeneralized states.
	// 2 original edges × 2 levels = up to 4 degeneralized edges.
	if len(newEdges) < 2 {
		t.Errorf("two sets: got %d edges, want at least 2", len(newEdges))
	}
	// With 2 sets, should have accepting edges when both sets visited in order.
	hasAcc := false
	for _, e := range newEdges {
		if e.Accept {
			hasAcc = true
		}
	}
	if !hasAcc {
		t.Error("two sets: should have an accepting edge (both sets visited)")
	}
}

func TestAdvance(t *testing.T) {
	dg := newDegeneralizer(t)
	// level 0, visited {0}, total 2 → advance to 1, not accepting.
	next, acc := dg.advance(0, []int{0}, 2)
	if next != 1 || acc {
		t.Errorf("advance(0,{0},2): got (%d,%v), want (1,false)", next, acc)
	}
	// level 1, visited {1}, total 2 → advance to 0, accepting.
	next, acc = dg.advance(1, []int{1}, 2)
	if next != 0 || !acc {
		t.Errorf("advance(1,{1},2): got (%d,%v), want (0,true)", next, acc)
	}
	// level 0, visited {} → stay at 0.
	next, acc = dg.advance(0, nil, 3)
	if next != 0 || acc {
		t.Errorf("advance(0,nil,3): got (%d,%v), want (0,false)", next, acc)
	}
	// Skip: level 0, visited {0,2}, total 3 → jump to 3? Actually maxV=2, maxV+1=3≥total→accepting, reset to 0.
	next, acc = dg.advance(0, []int{0, 2}, 3)
	if next != 0 || !acc {
		t.Errorf("advance(0,{0,2},3): got (%d,%v), want (0,true)", next, acc)
	}
}

func TestDegenCC(t *testing.T) {
	dg := newDegeneralizer(t)
	edges := []struct{ Src, Dst int; Visited []int }{
		{0, 1, []int{0}},
		{1, 2, []int{1}},
		{2, 0, []int{0, 1}},
	}
	for i := 0; i < 10; i++ {
		dg.Degeneralize(3, 3, edges)
	}
}
