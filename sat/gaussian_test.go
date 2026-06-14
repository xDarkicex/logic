package sat

import (
	"testing"
)

func TestGaussianEliminator_Basic(t *testing.T) {
	ge := NewGaussianEliminator()
	if ge == nil {
		t.Fatal("Expected GaussianEliminator, got nil")
	}

	if ge.maxMatrixRows != 300 {
		t.Errorf("Expected maxMatrixRows 300, got %d", ge.maxMatrixRows)
	}

	geWithConfig := NewGaussianEliminatorWithConfig(100, 50, 2, 10, 1000)
	if geWithConfig.maxMatrixRows != 100 {
		t.Errorf("Expected maxMatrixRows 100, got %d", geWithConfig.maxMatrixRows)
	}

	// Should not run if disabled or insufficient XORs
	ge.disabled = true
	if ge.ShouldRunGaussian(10000, 10) {
		t.Error("Should not run if disabled")
	}

	ge.disabled = false
	if ge.ShouldRunGaussian(100, 10) {
		t.Error("Should not run if conflicts < lastGaussian + frequency")
	}

	if ge.ShouldRunGaussian(10000, 2) {
		t.Error("Should not run if XOR count < 5")
	}

	if !ge.ShouldRunGaussian(10000, 10) {
		t.Error("Should run with sufficient conflicts and XOR clauses")
	}
}

func TestGaussianEliminator_PerformGaussianElimination(t *testing.T) {
	ge := NewGaussianEliminator()
	ge.minXORSize = 2
	
	// A XOR B = 1
	// B XOR C = 0
	// A XOR C = 0 (Contradiction because A=1^B, C=B, so A^C = 1^B^B = 1)
	
	xor1 := NewXORClause([]string{"A", "B"}, true)
	xor2 := NewXORClause([]string{"B", "C"}, false)
	xor3 := NewXORClause([]string{"A", "C"}, false)
	
	ecnf := &ExtendedCNF{
		CNF:        NewCNF(),
		XORClauses: []*XORClause{xor1, xor2, xor3},
	}
	
	assignment := make(Assignment)
	
	result, err := ge.PerformGaussianElimination(ecnf, assignment, 10000)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if !result.ConflictFound {
		t.Error("Expected conflict to be found in A^B=1, B^C=0, A^C=0")
	}
	
	// Test unit propagation
	ge.Reset()
	// A XOR B = 1
	// B = 1 (assigned)
	// Expect A = 0
	xor4 := NewXORClause([]string{"A", "B"}, true)
	ecnf.XORClauses = []*XORClause{xor4}
	
	assignment["B"] = true
	
	// We need at least 2 unassigned variables in a clause to build the matrix according to buildMatrix()
	// So A XOR B where B is assigned will be SKIPPED by buildMatrix.
	// Let's do A XOR B XOR C = 1
	// B = 1
	// So A XOR C = 0
	xor5 := NewXORClause([]string{"A", "B", "C"}, true)
	ecnf.XORClauses = []*XORClause{xor5}
	
	// Let's use a standard matrix reduction to learn a unit
	// A XOR B XOR C = 1
	// A XOR B = 0
	// Result: C = 1
	xorA := NewXORClause([]string{"A", "B", "C"}, true)
	xorB := NewXORClause([]string{"A", "B"}, false)
	ecnf.XORClauses = []*XORClause{xorA, xorB}
	
	result2, err := ge.PerformGaussianElimination(ecnf, make(Assignment), 20000)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	
	if len(result2.UnitsLearned) != 1 {
		t.Errorf("Expected 1 unit learned, got %d", len(result2.UnitsLearned))
	} else {
		unit := result2.UnitsLearned[0]
		if unit.Variable != "C" || unit.Negated {
			t.Errorf("Expected C=1, got %v", unit)
		}
	}
}

func TestGaussianEliminator_AutoDisable(t *testing.T) {
	ge := NewGaussianEliminator()
	
	// Fake some runs
	ge.stats.TotalRuns = 10
	ge.stats.VariablesEliminated = 0
	ge.stats.UnitPropagations = 0
	
	if !ge.shouldDisable() {
		t.Error("Should disable if extremely ineffective")
	}
}
