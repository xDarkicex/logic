package modal

import (
	"testing"

	"github.com/xDarkicex/memory"
)

func newClassifier(t *testing.T) *LTLClassifier {
	t.Helper()
	pool, err := memory.NewPool(memory.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Reset)
	return NewLTLClassifier(pool)
}

func TestClassifyAtom(t *testing.T) {
	c := newClassifier(t)
	p := Atom{ID: 0}
	if !c.IsObligation(p) {
		t.Error("atoms should be obligation (safety)")
	}
	if c.IsSuspendable(p) {
		t.Error("atoms should not be suspendable")
	}
	if c.Classify(p) != ClassObligation {
		t.Error("atoms classify as obligation")
	}
}

func TestClassifyBox(t *testing.T) {
	c := newClassifier(t)
	p := Atom{ID: 0}
	// □p — always p (safety)
	b := Box{Formula: p, Rel: RelCausal}
	if !c.IsObligation(b) {
		t.Error("□p should be obligation (safety)")
	}
	if c.IsSuspendable(b) {
		t.Error("□p should not be suspendable")
	}
}

func TestClassifyDiamond(t *testing.T) {
	c := newClassifier(t)
	p := Atom{ID: 0}
	// ◇p — eventually p (eventual, not obligation)
	d := Diamond{Formula: p, Rel: RelCausal}
	if c.IsObligation(d) {
		t.Error("◇p should not be obligation (liveness, not safety)")
	}
	// ◇□p — suspendable (eventual at top, universal inside)
	inner := Box{Formula: p, Rel: RelCausal}
	ds := Diamond{Formula: inner, Rel: RelCausal}
	if !c.IsSuspendable(ds) {
		t.Error("◇□p should be suspendable")
	}
	if c.Classify(ds) != ClassSuspendable {
		t.Error("◇□p should classify as suspendable")
	}
}

func TestClassifyUntil(t *testing.T) {
	c := newClassifier(t)
	p := Atom{ID: 0}
	q := Atom{ID: 1}
	// p U q — general (not obligation, not suspendable)
	u := Until{Left: p, Right: q}
	if c.IsObligation(u) {
		t.Error("p U q should not be obligation")
	}
	if c.IsSuspendable(u) {
		t.Error("p U q should not be suspendable")
	}
	if c.Classify(u) != ClassRest {
		t.Error("p U q classifies as rest")
	}
}

func TestClassifyAndOr(t *testing.T) {
	c := newClassifier(t)
	p := Atom{ID: 0}
	q := Atom{ID: 1}
	// (□p) ∧ (◇q) — obligation ∧ eventual → not obligation
	and := And{Left: Box{Formula: p, Rel: RelCausal}, Right: Diamond{Formula: q, Rel: RelCausal}}
	if c.IsObligation(and) {
		t.Error("□p ∧ ◇q should not be obligation (contains liveness)")
	}
	// (□p) ∨ (□q) — obligation ∨ obligation → obligation
	or := Or{Left: Box{Formula: p, Rel: RelCausal}, Right: Box{Formula: q, Rel: RelCausal}}
	if !c.IsObligation(or) {
		t.Error("□p ∨ □q should be obligation")
	}
}

func TestSplitConjunction(t *testing.T) {
	c := newClassifier(t)
	p := Atom{ID: 0}
	q := Atom{ID: 1}
	// □p ∧ □◇q ∧ (r U s) — obligation ∧ suspendable ∧ rest
	obl := Box{Formula: p, Rel: RelCausal}
	sus := Box{Formula: Diamond{Formula: q, Rel: RelCausal}, Rel: RelCausal}
	rest := Until{Left: Atom{ID: 2}, Right: Atom{ID: 3}}
	f := And{Left: obl, Right: And{Left: sus, Right: rest}}

	g := c.SplitConjunction(f)
	if len(g.Obligation) != 1 {
		t.Errorf("expected 1 obligation, got %d", len(g.Obligation))
	}
	if len(g.Suspendable) != 1 {
		t.Errorf("expected 1 suspendable, got %d", len(g.Suspendable))
	}
	if len(g.Rest) != 1 {
		t.Errorf("expected 1 rest, got %d", len(g.Rest))
	}
}

func TestSplitAllObligation(t *testing.T) {
	c := newClassifier(t)
	p := Atom{ID: 0}
	q := Atom{ID: 1}
	obl := And{Left: p, Right: Box{Formula: q, Rel: RelCausal}}
	if c.Classify(obl) != ClassObligation {
		t.Error("atom ∧ □q should be obligation")
	}
	g := c.SplitConjunction(obl)
	if !g.HasOnlyObligation() {
		t.Error("all-obligation conjunction should have HasOnlyObligation=true")
	}
}

func TestSplitDisjunction(t *testing.T) {
	c := newClassifier(t)
	p := Atom{ID: 0}
	obl := Box{Formula: p, Rel: RelCausal}
	rest := Until{Left: Atom{ID: 1}, Right: Atom{ID: 2}}
	f := Or{Left: obl, Right: rest}

	g := c.SplitDisjunction(f)
	if len(g.Obligation) != 1 {
		t.Errorf("expected 1 obligation, got %d", len(g.Obligation))
	}
	if len(g.Rest) != 1 {
		t.Errorf("expected 1 rest, got %d", len(g.Rest))
	}
	if g.IsEmpty() {
		t.Error("group should not be empty")
	}
}

func TestFlattenAnd(t *testing.T) {
	c := newClassifier(t)
	p := Atom{ID: 0}
	q := Atom{ID: 1}
	r := Atom{ID: 2}
	f := And{Left: p, Right: And{Left: q, Right: r}}
	result := c.flattenAnd(f)
	if len(result) != 3 {
		t.Errorf("flattenAnd: expected 3, got %d", len(result))
	}
}

func TestClassifyNext(t *testing.T) {
	c := newClassifier(t)
	p := Atom{ID: 0}
	// ○p — preserves classification
	n := Next{Formula: p}
	if !c.IsObligation(n) {
		t.Error("○p should be obligation")
	}
	// ○◇p — preserves eventual
	n2 := Next{Formula: Diamond{Formula: p, Rel: RelCausal}}
	if c.IsObligation(n2) {
		t.Error("○◇p should not be obligation")
	}
}

func TestCCCompliance(t *testing.T) {
	// Verify all exported functions complete within reasonable time.
	c := newClassifier(t)
	p := Atom{ID: 0}
	for i := 0; i < 100; i++ {
		c.Classify(p)
		c.IsObligation(p)
		c.IsSuspendable(p)
	}
	g := c.SplitConjunction(And{Left: p, Right: p})
	if g.IsEmpty() {
		t.Error("should not be empty")
	}
}
