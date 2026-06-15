package sat

import (
	"testing"

	"github.com/xDarkicex/memory"
)

func newTheorySolver(t *testing.T, n int) *TheorySolver {
	t.Helper()
	pool, err := memory.NewPool(memory.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Reset)
	return NewTheorySolver(n, pool)
}

func TestTheorySolverBasic(t *testing.T) {
	ts := newTheorySolver(t, 3)
	// (x0 ∨ x1) ∧ (¬x0 ∨ x2) ∧ (¬x1)
	// Encoding: var*2 = positive, var*2+1 = negative
	ts.AddClause([]int32{0 * 2, 1 * 2})       // x0 ∨ x1
	ts.AddClause([]int32{0*2 + 1, 2 * 2})     // ¬x0 ∨ x2
	ts.AddClause([]int32{1*2 + 1})             // ¬x1

	assign, ok := ts.Solve()
	if !ok {
		t.Fatal("expected satisfiable")
	}
	// x1 must be false (unit clause ¬x1).
	if assign[1] != 0 {
		t.Errorf("x1 must be false: got %d", assign[1])
	}
	// x0 must be true (from x0∨x1 with x1=false).
	if assign[0] != 1 {
		t.Errorf("x0 must be true: got %d", assign[0])
	}
	// x2 must be true (from ¬x0∨x2 with x0=true implies x2=true).
	if assign[2] != 1 {
		t.Errorf("x2 must be true: got %d", assign[2])
	}
}

func TestTheorySolverUnsat(t *testing.T) {
	ts := newTheorySolver(t, 2)
	ts.AddClause([]int32{0 * 2})
	ts.AddClause([]int32{0*2 + 1})

	_, ok := ts.Solve()
	if ok {
		t.Error("x0 ∧ ¬x0 should be UNSAT")
	}
}

func TestTheorySolverLarge(t *testing.T) {
	ts := newTheorySolver(t, 10)
	// Simple chain: x0, x0→x1, x1→x2, ..., → x9 all true.
	ts.AddClause([]int32{0 * 2})
	for i := int32(0); i < 9; i++ {
		ts.AddClause([]int32{i*2 + 1, (i + 1) * 2})
	}
	assign, ok := ts.Solve()
	if !ok {
		t.Fatal("expected satisfiable")
	}
	for i := int32(0); i < 10; i++ {
		if assign[i] != 1 {
			t.Errorf("x%d should be true", i)
		}
	}
}

// mockPlugin is a test theory plugin that rejects assignments where x0 = 0.
type mockPlugin struct{}

func (m *mockPlugin) Check(assign []int8) (bool, []int32) {
	if len(assign) > 0 && assign[0] == 0 {
		return false, []int32{0 * 2} // theory forces x0 = true
	}
	return true, nil
}

func (m *mockPlugin) Name() string { return "mock" }

func TestTheorySolverPlugin(t *testing.T) {
	ts := newTheorySolver(t, 3)
	ts.RegisterPlugin(&mockPlugin{})
	// (¬x0 ∨ x1) ∧ (x0 ∨ x2)
	ts.AddClause([]int32{0*2 + 1, 1 * 2})
	ts.AddClause([]int32{0 * 2, 2 * 2})

	assign, ok := ts.Solve()
	if !ok {
		t.Fatal("expected satisfiable")
	}
	// Theory forces x0 = true.
	if assign[0] != 1 {
		t.Errorf("theory should force x0=true: got %d", assign[0])
	}
}

func TestTheorySolverReset(t *testing.T) {
	ts := newTheorySolver(t, 2)
	ts.AddClause([]int32{0 * 2, 1 * 2})
	ts.Solve()
	ts.Reset()
	ts.AddClause([]int32{0 * 2})
	ts.AddClause([]int32{0*2 + 1})
	_, ok := ts.Solve()
	if ok {
		t.Error("should be UNSAT after reset and new clauses")
	}
}

func TestTheorySolverNumVars(t *testing.T) {
	ts := newTheorySolver(t, 5)
	if ts.NumVars() != 5 {
		t.Errorf("NumVars: got %d, want 5", ts.NumVars())
	}
}

func TestDplltCC(t *testing.T) {
	ts := newTheorySolver(t, 20)
	for i := int32(0); i < 20; i++ {
		if i%3 == 0 {
			ts.AddClause([]int32{i * 2})
		} else {
			ts.AddClause([]int32{i*2 + 1, ((i + 1) % 20) * 2})
		}
	}
	for i := 0; i < 5; i++ {
		ts.Solve()
		ts.Reset()
	}
}
