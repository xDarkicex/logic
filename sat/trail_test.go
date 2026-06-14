package sat

import (
	"testing"
)

func TestDecisionTrailImpl_Basic(t *testing.T) {
	trail := NewDecisionTrail()
	defer trail.Close()

	if trail.GetCurrentLevel() != 0 {
		t.Errorf("Expected initial level 0, got %d", trail.GetCurrentLevel())
	}
	if trail.GetTrailSize() != 0 {
		t.Errorf("Expected initial size 0, got %d", trail.GetTrailSize())
	}

	// Level 1 assignment
	trail.Assign("A", true, 1, nil) // Decision
	if trail.GetTrailSize() != 1 {
		t.Errorf("Expected size 1, got %d", trail.GetTrailSize())
	}
	if trail.GetCurrentLevel() != 1 {
		t.Errorf("Expected level 1, got %d", trail.GetCurrentLevel())
	}
	if !trail.IsDecisionVariable("A") {
		t.Error("Expected A to be a decision variable")
	}
	
	// Propagated assignment
	reason := NewClause(Literal{Variable: "A", Negated: true}, Literal{Variable: "B", Negated: false})
	trail.Assign("B", true, 1, reason)
	
	if trail.IsDecisionVariable("B") {
		t.Error("Expected B to NOT be a decision variable")
	}

	// Backtrack
	unassigned := trail.Backtrack(0)
	if len(unassigned) != 2 {
		t.Errorf("Expected 2 unassigned variables, got %d", len(unassigned))
	}
	if trail.GetTrailSize() != 0 {
		t.Errorf("Expected size 0 after backtrack, got %d", trail.GetTrailSize())
	}
	if trail.GetCurrentLevel() != 0 {
		t.Errorf("Expected level 0 after backtrack, got %d", trail.GetCurrentLevel())
	}
}

func TestDecisionTrailImpl_ImplicationChain(t *testing.T) {
	trail := NewDecisionTrail()
	defer trail.Close()

	// Level 1: A=1 (Decision)
	trail.Assign("A", true, 1, nil)
	
	// A -> B
	reasonAB := NewClause(Literal{Variable: "A", Negated: true}, Literal{Variable: "B", Negated: false})
	trail.Assign("B", true, 1, reasonAB)
	
	// B -> C
	reasonBC := NewClause(Literal{Variable: "B", Negated: true}, Literal{Variable: "C", Negated: false})
	trail.Assign("C", true, 1, reasonBC)

	chain := trail.GetImplicationChain("C")
	
	// The chain should be C, B, A
	if len(chain) != 3 {
		t.Fatalf("Expected chain of length 3, got %d", len(chain))
	}
	if chain[0].Variable != "C" || chain[1].Variable != "B" || chain[2].Variable != "A" {
		t.Errorf("Unexpected chain: %v", chain)
	}
}

func TestDecisionTrailImpl_AdvancedGetters(t *testing.T) {
	trail := NewDecisionTrail()
	defer trail.Close()

	trail.Assign("A", true, 1, nil)
	trail.Assign("B", false, 1, nil)
	trail.Assign("C", true, 2, nil)

	level1 := trail.GetTrailAtLevel(1)
	if len(level1) != 2 {
		t.Errorf("Expected 2 assignments at level 1, got %d", len(level1))
	}

	level2 := trail.GetTrailAtLevel(2)
	if len(level2) != 1 {
		t.Errorf("Expected 1 assignment at level 2, got %d", len(level2))
	}

	decisions := trail.GetDecisionVariablesAtLevel(1)
	if len(decisions) != 2 {
		t.Errorf("Expected 2 decisions at level 1, got %d", len(decisions))
	}

	levels := trail.GetAllLevels()
	if len(levels) != 2 {
		t.Errorf("Expected 2 active levels, got %d", len(levels))
	}
	
	trail.Clear()
	if trail.GetTrailSize() != 0 {
		t.Errorf("Clear failed")
	}
}
