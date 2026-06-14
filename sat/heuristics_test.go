package sat

import (
	"testing"
)

func TestVSIDSHeuristic_Basic(t *testing.T) {
	h := NewVSIDSHeuristic()
	
	if h.Name() != "VSIDS-LRB-Enhanced" {
		t.Errorf("Unexpected name: %s", h.Name())
	}

	assignment := make(Assignment)
	
	unassigned := []string{"A", "B", "C"}
	
	// Initial choice should be one of them (likely A due to alphabetical/order)
	chosen := h.ChooseVariable(unassigned, assignment)
	if chosen == "" {
		t.Error("Should choose a variable")
	}

	// Update to increase activity of "B"
	c1 := NewClause(Literal{Variable: "B", Negated: false})
	h.Update(c1)

	// Now B should have higher activity and be chosen
	chosen = h.ChooseVariable(unassigned, assignment)
	if chosen != "B" {
		t.Errorf("Expected B to be chosen, got %s", chosen)
	}

	// Polarity test
	// We recorded B as positive (not negated), so preferred polarity should be positive (false negated)
	// Wait, the update logic says:
	// if lit.Negated { polarityScores -= 0.1 } else { polarityScores += 0.1 }
	// phaseCache = lit.Negated
	// GetPreferredPolarity returns phaseCache first. So it will return lit.Negated = false.
	polarity := h.GetPreferredPolarity("B")
	if polarity != false {
		t.Error("Expected positive polarity for B")
	}
	
	h.Reset()
	if len(h.activity) != 0 {
		t.Error("Reset failed")
	}
}

func TestLubyRestartStrategy(t *testing.T) {
	l := NewLubyRestartStrategy()
	
	stats := SolverStatistics{Conflicts: 0}
	
	// Initially should not restart
	if l.ShouldRestart(stats) {
		t.Error("Should not restart at 0 conflicts")
	}

	// Trigger Luby restart at 100 conflicts (sequence[0] * 100 = 100)
	stats.Conflicts = 100
	if !l.ShouldRestart(stats) {
		t.Error("Should restart at 100 conflicts")
	}
	
	l.OnRestart()
	
	// Now sequence index is 1, value is 1, so next threshold is 100 + 100 = 200 conflicts
	stats.Conflicts = 200
	if !l.ShouldRestart(stats) {
		t.Error("Should restart at 200 conflicts")
	}
	
	l.Reset()
	if l.index != 0 {
		t.Error("Reset failed")
	}
}

func TestActivityBasedDeletion(t *testing.T) {
	a := NewActivityBasedDeletion()
	stats := SolverStatistics{}

	// Core clauses should never be deleted
	core := NewClause(Literal{Variable: "A", Negated: false})
	core.Learned = true
	core.Tier = 0
	core.LBD = 2
	core.Glue = true

	if a.ShouldDeleteFromTier(core, 0, stats) {
		t.Error("Core clauses should not be deleted")
	}

	// Mid-tier clause with low activity
	mid := NewClause(Literal{Variable: "B", Negated: false}, Literal{Variable: "C", Negated: false})
	mid.Learned = true
	mid.Tier = 1
	mid.LBD = 5
	mid.Activity = 0.01 // Below midThreshold 0.15

	if !a.ShouldDeleteFromTier(mid, 1, stats) {
		t.Error("Mid tier clause with low activity should be deleted")
	}

	// Local-tier clause
	local := NewClause(Literal{Variable: "D", Negated: false}, Literal{Variable: "E", Negated: false})
	local.Learned = true
	local.Tier = 2
	local.LBD = 8
	local.Activity = 0.05 // Below localThreshold 0.10

	if !a.ShouldDeleteFromTier(local, 2, stats) {
		t.Error("Local tier clause with low activity should be deleted")
	}
	
	// Update threshold logic
	a.Update([]*Clause{core, mid, local})
	
	a.Reset()
	if a.activityThreshold != 0.1 {
		t.Error("Reset failed")
	}
}
