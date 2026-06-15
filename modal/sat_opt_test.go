package modal

import (
	"testing"

	"github.com/xDarkicex/memory"
)

// ── B.2 DeadSubspaceTracker tests ──

func TestDeadSubspaceMarkAndCheck(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	dt := NewDeadSubspaceTracker(10, pool)
	dt.MarkDead([]int32{3, 5, 7})
	if !dt.IsDead([]int32{3}) {
		t.Error("var 3 should be dead")
	}
	if dt.IsDead([]int32{1, 2}) {
		t.Error("vars 1,2 should not be dead")
	}
	if dt.DeadCount() != 3 {
		t.Errorf("dead count: got %d, want 3", dt.DeadCount())
	}
}

func TestDeadSubspaceReset(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	dt := NewDeadSubspaceTracker(5, pool)
	dt.MarkDead([]int32{0, 1})
	dt.Reset()
	if dt.IsDead([]int32{0}) {
		t.Error("should be clear after reset")
	}
}

// ── B.3 VariableWeighter tests ──

func TestVariableWeighter(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	vw := NewVariableWeighter(5, pool)
	vw.WeightByClass([]int32{0, 1}, ClassSuspendable)
	vw.WeightByClass([]int32{2}, ClassRest)
	if vw.Weight(0) != 4.0 {
		t.Errorf("suspendable var weight: got %v, want 4.0", vw.Weight(0))
	}
	if vw.Weight(2) != 2.0 {
		t.Errorf("rest var weight: got %v, want 2.0", vw.Weight(2))
	}
	if vw.Weight(4) != 1.0 {
		t.Errorf("unset var weight: got %v, want 1.0", vw.Weight(4))
	}
}

func TestVariableWeighterTopN(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	vw := NewVariableWeighter(4, pool)
	vw.WeightByClass([]int32{1}, ClassSuspendable) // 4.0
	vw.WeightByClass([]int32{3}, ClassRest)         // 2.0
	vw.WeightByClass([]int32{0}, ClassObligation)   // 1.0
	top := vw.TopN(2)
	if len(top) != 2 {
		t.Fatalf("TopN(2) got %d", len(top))
	}
	if top[0] != 1 {
		t.Errorf("top[0]=%d, want 1 (weight 4.0)", top[0])
	}
	if top[1] != 3 {
		t.Errorf("top[1]=%d, want 3 (weight 2.0)", top[1])
	}
}

func TestVariableWeighterProps(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	vw := NewVariableWeighter(3, pool)
	vw.WeightByProps([]int32{0}, false, true, true)  // ev+un = 4.0
	vw.WeightByProps([]int32{1}, false, true, false)  // ev only = 3.0
	vw.WeightByProps([]int32{2}, false, false, true)  // un only = 2.0
	if vw.Weight(0) != 4.0 || vw.Weight(1) != 3.0 || vw.Weight(2) != 2.0 {
		t.Errorf("props weights: got %v %v %v", vw.Weight(0), vw.Weight(1), vw.Weight(2))
	}
}

// ── B.4 ComponentDecomposer tests ──

func TestDecomposeSingleClause(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	cd := NewComponentDecomposer(pool)
	result := cd.Decompose([][]int32{{1, 2, 3}})
	if len(result) != 1 {
		t.Errorf("single clause: got %d components", len(result))
	}
}

func TestDecomposeIndependent(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	cd := NewComponentDecomposer(pool)
	// Two independent components: {1,2} and {3,4}
	clauses := [][]int32{
		{1, 2},
		{3, 4},
	}
	result := cd.Decompose(clauses)
	if len(result) != 2 {
		t.Errorf("independent: got %d components, want 2", len(result))
	}
}

func TestDecomposeConnected(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	cd := NewComponentDecomposer(pool)
	// Connected via shared variable 2
	clauses := [][]int32{
		{1, 2},
		{2, 3},
		{3, 4},
	}
	result := cd.Decompose(clauses)
	if len(result) != 1 {
		t.Errorf("connected: got %d components, want 1", len(result))
	}
}

func TestDecomposeMixed(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	cd := NewComponentDecomposer(pool)
	// {1,2} connected, {3,4} connected, {5} isolated → 3 components
	clauses := [][]int32{
		{1, 2},
		{2, 3},  // connects 1,2,3
		{4, 5},  // separate component {4,5}
		{6},     // isolated
	}
	result := cd.Decompose(clauses)
	if len(result) != 3 {
		t.Errorf("mixed: got %d components, want 3", len(result))
	}
}

func TestDecomposeNegated(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	cd := NewComponentDecomposer(pool)
	// Negated variables use absolute value for connectivity.
	clauses := [][]int32{
		{1, -2},
		{-1, 3},
	}
	result := cd.Decompose(clauses)
	if len(result) != 1 {
		t.Errorf("negated: got %d components, want 1", len(result))
	}
}

func TestSatOptCC(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	dt := NewDeadSubspaceTracker(50, pool)
	vw := NewVariableWeighter(50, pool)
	cd := NewComponentDecomposer(pool)
	for i := 0; i < 20; i++ {
		dt.MarkDead([]int32{int32(i)})
		dt.Reset()
		vw.WeightByClass([]int32{int32(i)}, ClassSuspendable)
		cd.Decompose([][]int32{{int32(i), int32(i + 1)}})
	}
}
