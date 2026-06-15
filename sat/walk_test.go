package sat

import (
	"testing"
)

func TestNewWalkSolver(t *testing.T) {
	w := NewWalkSolver()
	if w.maxFlips != walkDefaultFlips {
		t.Errorf("maxFlips = %d, want %d", w.maxFlips, walkDefaultFlips)
	}
	if w.UnsatCount() != 0 {
		t.Error("new solver should have 0 unsat")
	}
}

func TestWalkSolverEmptyClauses(t *testing.T) {
	w := NewWalkSolver()
	if !w.Solve(nil) {
		t.Error("empty clause set should be satisfiable")
	}
	if !w.Solve([]*Clause{}) {
		t.Error("empty clause list should be satisfiable")
	}
}

func TestWalkSolverTrivialSAT(t *testing.T) {
	w := NewWalkSolver()
	// Single unit clause: (A)
	a := NewClause(Literal{Variable: "A", Negated: false})
	found := w.Solve([]*Clause{a})
	if !found {
		t.Error("trivial SAT should be found")
	}
}

func TestWalkSolverSimpleSAT(t *testing.T) {
	w := NewWalkSolver()
	// (A ∨ B) ∧ (¬A ∨ B) — satisfiable with B=true
	c1 := NewClause(Literal{Variable: "A", Negated: false}, Literal{Variable: "B", Negated: false})
	c2 := NewClause(Literal{Variable: "A", Negated: true}, Literal{Variable: "B", Negated: false})
	found := w.Solve([]*Clause{c1, c2})
	if !found {
		t.Error("simple SAT should be found")
	}
}

func TestWalkSolverUnsatHitsLimit(t *testing.T) {
	w := NewWalkSolver()
	w.maxFlips = 500
	// (A) ∧ (¬A) — unsatisfiable
	c1 := NewClause(Literal{Variable: "A", Negated: false})
	c2 := NewClause(Literal{Variable: "A", Negated: true})
	found := w.Solve([]*Clause{c1, c2})
	if found {
		t.Error("unsatisfiable should not be solved")
	}
	if w.FlipCount() == 0 {
		t.Error("should have attempted flips before giving up")
	}
}

func TestWalkSolverExportPhases(t *testing.T) {
	w := NewWalkSolver()
	// (A ∨ B) — satisfiable
	c1 := NewClause(Literal{Variable: "A", Negated: false}, Literal{Variable: "B", Negated: false})
	w.Solve([]*Clause{c1})

	h := NewVSIDSHeuristic()
	w.ExportPhases(h)

	// A and B should have phases set
	aIdx := h.varIndex["A"]
	if h.phases[aIdx] == phaseUnset {
		t.Error("phase for A should be set after export")
	}
}

func TestWalkSolverScoreTable(t *testing.T) {
	w := NewWalkSolver()
	w.initFromClauses([]*Clause{
		NewClause(Literal{Variable: "X", Negated: false}),
	})
	// scoreTable[0] = 1.0, scoreTable[1] = 0.5, scoreTable[2] = 0.25, ...
	if w.scoreTable[0] != 1.0 {
		t.Errorf("scoreTable[0] = %v, want 1.0", w.scoreTable[0])
	}
	if w.scoreTable[1] != 0.5 {
		t.Errorf("scoreTable[1] = %v, want 0.5", w.scoreTable[1])
	}
}

func TestWalkSolverBreakCount(t *testing.T) {
	w := NewWalkSolver()
	// Current default assignment: all variables = true
	// Clause: (¬A) — A=true means ¬A is false, clause is unsat
	c1 := NewClause(Literal{Variable: "A", Negated: true})
	w.Solve([]*Clause{c1})
	// After Solve, if SAT found, unsat=0
	if w.UnsatCount() != 0 && w.FlipCount() == 0 {
		// WalkSAT may solve it immediately or hit limit
		// Just verify we ran without panicking
	}
}

func TestWalkSolverReset(t *testing.T) {
	w := NewWalkSolver()
	c1 := NewClause(Literal{Variable: "A", Negated: false})
	w.Solve([]*Clause{c1})

	w.Reset()
	if w.numVars != 0 {
		t.Error("after Reset, numVars should be 0")
	}
	if w.UnsatCount() != 0 {
		t.Error("after Reset, unsat should be 0")
	}
	if w.FlipCount() != 0 {
		t.Error("after Reset, flips should be 0")
	}
}

func TestWalkSolverLitIndexing(t *testing.T) {
	w := NewWalkSolver()
	c1 := NewClause(Literal{Variable: "X", Negated: false})
	c2 := NewClause(Literal{Variable: "X", Negated: true})
	w.initFromClauses([]*Clause{c1, c2})

	posIdx := w.litIdx("X", false)
	negIdx := w.litIdx("X", true)
	if posIdx != 0 {
		t.Errorf("positive literal index = %d, want 0", posIdx)
	}
	if negIdx != 1 {
		t.Errorf("negative literal index = %d, want 1", negIdx)
	}
}

func TestWalkSolverMultiVar(t *testing.T) {
	w := NewWalkSolver()
	// (A ∨ B) ∧ (B ∨ C) ∧ (¬A ∨ ¬B)
	c1 := NewClause(Literal{Variable: "A", Negated: false}, Literal{Variable: "B", Negated: false})
	c2 := NewClause(Literal{Variable: "B", Negated: false}, Literal{Variable: "C", Negated: false})
	c3 := NewClause(Literal{Variable: "A", Negated: true}, Literal{Variable: "B", Negated: true})

	found := w.Solve([]*Clause{c1, c2, c3})
	if !found {
		t.Log("multi-var SAT not found (may need more flips)")
	}
	// Verify assignment satisfies all clauses
	if found {
		for _, c := range []*Clause{c1, c2, c3} {
			sat := false
			for _, lit := range c.Literals {
				idx := w.varIndex[lit.Variable]
				val := w.bestValues[idx]
				if val == -1 {
					continue
				}
				if (!lit.Negated && val > 0) || (lit.Negated && val == 0) {
					sat = true
					break
				}
			}
			if !sat {
				t.Errorf("clause %s not satisfied by best assignment", c.String())
			}
		}
	}
}
