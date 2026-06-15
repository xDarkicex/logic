package modal

import (
	"testing"

	"github.com/xDarkicex/memory"
)

func newMinimizer(t *testing.T) *DFAMinimizer {
	t.Helper()
	pool, err := memory.NewPool(memory.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Reset)
	return NewDFAMinimizer(pool)
}

func TestDFATransition(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	dt := NewDFATransition(3, 2, pool)
	dt.Set(0, 0, 1)
	dt.Set(0, 1, 2)
	if dt.Get(0, 0) != 1 || dt.Get(0, 1) != 2 {
		t.Error("transition Set/Get mismatch")
	}
}

func TestHopcroftAllSame(t *testing.T) {
	dm := newMinimizer(t)
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	// 3 states, all non-accepting, 1 symbol, all transition to state 0.
	dt := NewDFATransition(3, 1, pool)
	dt.Set(0, 0, 0)
	dt.Set(1, 0, 0)
	dt.Set(2, 0, 0)
	acc := []bool{false, false, false}
	result := dm.Minimize(3, acc, dt)
	// All states should merge into 1.
	first := result[0]
	for i := 1; i < 3; i++ {
		if result[i] != first {
			t.Errorf("all-same: state %d should merge with 0", i)
		}
	}
}

func TestHopcroftAcceptSplit(t *testing.T) {
	dm := newMinimizer(t)
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	// 3 states: 0→1, 1→2, 2→2. State 2 is accepting.
	dt := NewDFATransition(3, 1, pool)
	dt.Set(0, 0, 1)
	dt.Set(1, 0, 2)
	dt.Set(2, 0, 2)
	acc := []bool{false, false, true}
	result := dm.Minimize(3, acc, dt)
	// State 2 (accepting) should be separate from 0,1.
	if result[2] == result[0] {
		t.Error("accepting state should not merge with non-accepting")
	}
	// States 0 and 1 have different destinations → should split.
	if result[0] == result[1] {
		t.Error("states with different transitions should split")
	}
}

func TestHopcroftMergeEquivalent(t *testing.T) {
	dm := newMinimizer(t)
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	// 4 states, 2 symbols. States 0,2 are equivalent (both non-accepting, same pattern).
	// 0--a→1, 0--b→3; 2--a→1, 2--b→3. State 1 accepting, 3 non-accepting.
	dt := NewDFATransition(4, 2, pool)
	dt.Set(0, 0, 1)
	dt.Set(0, 1, 3)
	dt.Set(1, 0, 1)
	dt.Set(1, 1, 1)
	dt.Set(2, 0, 1)
	dt.Set(2, 1, 3) // same pattern as 0
	dt.Set(3, 0, 3)
	dt.Set(3, 1, 3)
	acc := []bool{false, true, false, false}
	result := dm.Minimize(4, acc, dt)
	// States 0 and 2 should merge.
	if result[0] != result[2] {
		t.Errorf("equivalent states 0 and 2 should merge: %d vs %d", result[0], result[2])
	}
	// State 1 (accepting) should be separate.
	if result[0] == result[1] {
		t.Error("accepting state should not merge with non-accepting")
	}
}

func TestHopcroftSingleState(t *testing.T) {
	dm := newMinimizer(t)
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	dt := NewDFATransition(1, 1, pool)
	dt.Set(0, 0, 0)
	result := dm.Minimize(1, []bool{true}, dt)
	if len(result) != 1 || result[0] != 0 {
		t.Error("single state should stay single")
	}
}

func TestHopcroftTwoSymbols(t *testing.T) {
	dm := newMinimizer(t)
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	// 3 states, 2 symbols, all non-accepting.
	// 0--a→1, 0--b→2; 1--a→1, 1--b→1; 2--a→1, 2--b→2
	dt := NewDFATransition(3, 2, pool)
	dt.Set(0, 0, 1)
	dt.Set(0, 1, 2)
	dt.Set(1, 0, 1)
	dt.Set(1, 1, 1)
	dt.Set(2, 0, 1)
	dt.Set(2, 1, 2)
	acc := []bool{false, false, false}
	result := dm.Minimize(3, acc, dt)
	// Without accepting states, all states are language-equivalent (reject everything).
	// Hopcroft correctly merges them all into a single block.
	if result[0] != result[1] || result[1] != result[2] {
		t.Error("all-non-accepting states should merge (all reject every word)")
	}
}

func TestHopcroftCC(t *testing.T) {
	dm := newMinimizer(t)
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	dt := NewDFATransition(5, 2, pool)
	for i := int32(0); i < 5; i++ {
		dt.Set(i, 0, DFAState((i+1)%5))
		dt.Set(i, 1, DFAState((i+2)%5))
	}
	acc := []bool{false, true, false, true, false}
	for i := 0; i < 10; i++ {
		dm.Minimize(5, acc, dt)
	}
}
