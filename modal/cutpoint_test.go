package modal

import (
	"testing"

	"github.com/xDarkicex/memory"
)

func newRelabeler(t *testing.T) *CutPointRelabeler {
	t.Helper()
	pool, err := memory.NewPool(memory.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Reset)
	return NewCutPointRelabeler(pool)
}

func TestRelabelAtom(t *testing.T) {
	cr := newRelabeler(t)
	p := Atom{ID: 0}
	candidates := cr.RelabelCandidates(p)
	// Atom: leaf node, no children → not articulation point.
	if len(candidates) != 0 {
		t.Errorf("atom should have 0 candidates, got %d", len(candidates))
	}
}

func TestRelabelSimpleBoolean(t *testing.T) {
	cr := newRelabeler(t)
	p := Atom{ID: 0}
	q := Atom{ID: 1}
	// p ∧ q — root has 2 children, root is articulation point (no parent).
	// But children (atoms) are not articulation points.
	f := And{Left: p, Right: q}
	candidates := cr.RelabelCandidates(f)
	// Root is AP, but children are not AP → not a candidate.
	if len(candidates) != 0 {
		t.Errorf("p∧q should have 0 candidates, got %d", len(candidates))
	}
}

func TestRelabelNestedBoolean(t *testing.T) {
	cr := newRelabeler(t)
	p := Atom{ID: 0}
	q := Atom{ID: 1}
	r := Atom{ID: 2}
	// (p ∧ q) ∧ r — root has 2 children: (p∧q) and r.
	// (p∧q) has children p and q.
	// (p∧q) might be an articulation point if removing it disconnects the graph.
	f := And{Left: And{Left: p, Right: q}, Right: r}
	candidates := cr.RelabelCandidates(f)
	// The inner (p∧q) is Boolean. Root has (p∧q) and r as children.
	// (p∧q) separates p,q from r — it IS an articulation point.
	// But (p∧q)'s children (p, q) are atoms → not APs.
	// So no candidates.
	if len(candidates) != 0 {
		t.Errorf("(p∧q)∧r should have 0 candidates, got %d", len(candidates))
	}
}

func TestRelabelModalFormula(t *testing.T) {
	cr := newRelabeler(t)
	p := Atom{ID: 0}
	q := Atom{ID: 1}
	// □(p ∧ q) — Box node with (p∧q) inside.
	// Box is not Boolean → not a candidate.
	f := Box{Formula: And{Left: p, Right: q}, Rel: RelCausal}
	candidates := cr.RelabelCandidates(f)
	// (p∧q) is Boolean. Its children are p and q (atoms → not APs).
	if len(candidates) != 0 {
		t.Errorf("□(p∧q) should have 0 candidates, got %d", len(candidates))
	}
}

func TestRelabelSharedAtoms(t *testing.T) {
	cr := newRelabeler(t)
	a := Atom{ID: 0}
	b := Atom{ID: 1}
	// (a ∧ b) U (a ∧ ¬b) — a and b are shared across Until sides.
	// Should NOT be relabelable as p0 U p1 because a and b are shared.
	left := And{Left: a, Right: b}
	right := And{Left: a, Right: Not{Formula: b}}
	f := Until{Left: left, Right: right}
	candidates := cr.RelabelCandidates(f)
	// Left (a∧b) and right (a∧¬b) share atoms a and b.
	// a is shared across both sides → the (a∧b) nodes are NOT articulation points
	// because removing them doesn't disconnect the graph (a is connected elsewhere).
	if len(candidates) != 0 {
		t.Errorf("shared-atom formula should have 0 candidates, got %d", len(candidates))
	}
}

func TestArticulationPoints(t *testing.T) {
	cr := newRelabeler(t)
	p := Atom{ID: 0}
	q := Atom{ID: 1}
	r := Atom{ID: 2}
	// Build a chain: p ∧ (q ∧ r)
	f := And{Left: p, Right: And{Left: q, Right: r}}
	nodes := cr.buildGraph(f)
	art := cr.articulationPoints(nodes)
	// Root (And) should be AP (has >1 children in undirected DFS).
	if len(nodes) != 5 {
		t.Errorf("expected 5 nodes, got %d", len(nodes))
	}
	// At least the inner And (q∧r) should be an articulation point
	// because removing it disconnects q and r from the rest.
	hasAP := false
	for i, a := range art {
		if a {
			t.Logf("node %d is articulation point: %T", i, nodes[i].f)
			hasAP = true
		}
	}
	if !hasAP {
		t.Error("expected at least one articulation point")
	}
}

func TestCountNodes(t *testing.T) {
	cr := newRelabeler(t)
	p := Atom{ID: 0}
	q := Atom{ID: 1}
	// p ∧ q = 3 nodes (2 atoms + 1 And)
	n := cr.countNodes(And{Left: p, Right: q})
	if n != 3 {
		t.Errorf("countNodes(p∧q)=%d, want 3", n)
	}
	// ¬(p ∧ q) = 4 nodes (2 atoms + 1 And + 1 Not)
	n = cr.countNodes(Not{Formula: And{Left: p, Right: q}})
	if n != 4 {
		t.Errorf("countNodes(¬(p∧q))=%d, want 4", n)
	}
}

func TestRelabelCC(t *testing.T) {
	cr := newRelabeler(t)
	p := Atom{ID: 0}
	f := And{Left: p, Right: And{Left: Atom{ID: 1}, Right: Atom{ID: 2}}}
	for i := 0; i < 50; i++ {
		cr.RelabelCandidates(f)
	}
}
