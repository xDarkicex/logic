package modal

import (
	"testing"

	"github.com/xDarkicex/logic/fuzzy"
	"github.com/xDarkicex/memory"
)

type porFixtures struct {
	pool   *memory.Pool
	reg    *Registry
	matrix *DepMatrix
	por    *POR
}

func newPORFixtures(t *testing.T) *porFixtures {
	t.Helper()
	pool, err := memory.NewPool(memory.DefaultConfig())
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Reset)
	reg := NewRegistry(128, pool)
	matrix := NewDepMatrix(128, 64, pool)
	por := NewPOR(matrix, reg, pool)
	return &porFixtures{pool: pool, reg: reg, matrix: matrix, por: por}
}

func (fx *porFixtures) atom(id uint32) Formula { return fx.reg.Intern(Atom{ID: fuzzy.VarID(id)}) }
func (fx *porFixtures) regAndPin(f Formula) Formula {
	c := fx.reg.Intern(f)
	id := fx.reg.GetID(c)
	if id >= 0 {
		ComputeDeps(c, fx.matrix, id)
	}
	return c
}

func TestPORCommuteIndependent(t *testing.T) {
	fx := newPORFixtures(t)
	// □p and ◇q — p and q are different atoms → commute
	p := fx.regAndPin(fx.atom(1))
	q := fx.regAndPin(fx.atom(2))
	bp := fx.regAndPin(Box{Formula: p, Rel: RelCausal})
	dq := fx.regAndPin(Diamond{Formula: q, Rel: RelCausal})
	if !fx.por.Commute(bp, dq) {
		t.Error("□p and ◇q should commute (disjoint atoms)")
	}
}

func TestPORCommuteDependent(t *testing.T) {
	fx := newPORFixtures(t)
	// □p and ◇p — both depend on atom p → do NOT commute
	p := fx.regAndPin(fx.atom(1))
	bp := fx.regAndPin(Box{Formula: p, Rel: RelCausal})
	dp := fx.regAndPin(Diamond{Formula: p, Rel: RelCausal})
	if fx.por.Commute(bp, dp) {
		t.Error("□p and ◇p should NOT commute (shared atom p)")
	}
}

func TestPORCommuteSameAtom(t *testing.T) {
	fx := newPORFixtures(t)
	a := fx.regAndPin(fx.atom(1))
	b := fx.regAndPin(fx.atom(1))
	if fx.por.Commute(a, b) {
		t.Error("same atom should NOT commute (identical footprint)")
	}
}

func TestPORStubbornSetSingle(t *testing.T) {
	fx := newPORFixtures(t)
	p := fx.regAndPin(fx.atom(1))
	pfs := []PrefixedFormula{{World: 0, Formula: p}}
	indices := fx.por.StubbornSet(pfs)
	if len(indices) != 1 || indices[0] != 0 {
		t.Errorf("single formula: got %v, want [0]", indices)
	}
}

func TestPORStubbornSetAllIndependent(t *testing.T) {
	fx := newPORFixtures(t)
	a1 := fx.regAndPin(fx.atom(1))
	a2 := fx.regAndPin(fx.atom(2))
	a3 := fx.regAndPin(fx.atom(3))
	pfs := []PrefixedFormula{
		{World: 0, Formula: a1},
		{World: 0, Formula: a2},
		{World: 0, Formula: a3},
	}
	// All independent → stubborn set should contain at least one (seed + closure)
	indices := fx.por.StubbornSet(pfs)
	if len(indices) < 1 {
		t.Error("should have at least one element in stubborn set")
	}
	// With all independent, the seed is the only one selected (no dna edges)
	if len(indices) != 1 {
		t.Logf("all independent: got %d indices (seed=0, no dependencies)", len(indices))
	}
}

func TestPORStubbornSetAllDependent(t *testing.T) {
	fx := newPORFixtures(t)
	p := fx.regAndPin(fx.atom(1))
	// Three formulas all depending on atom 1 → all in stubborn set
	b1 := fx.regAndPin(Box{Formula: p, Rel: RelCausal})
	b2 := fx.regAndPin(Diamond{Formula: p, Rel: RelCausal})
	b3 := fx.regAndPin(Not{Formula: p})
	pfs := []PrefixedFormula{
		{World: 0, Formula: b1},
		{World: 0, Formula: b2},
		{World: 0, Formula: b3},
	}
	indices := fx.por.StubbornSet(pfs)
	if len(indices) != 3 {
		t.Errorf("all dependent on same atom: got %d, want 3", len(indices))
	}
}

func TestPORStubbornSetMixed(t *testing.T) {
	fx := newPORFixtures(t)
	p := fx.regAndPin(fx.atom(1))
	q := fx.regAndPin(fx.atom(2))
	r := fx.regAndPin(fx.atom(3))
	// bp and r share atom 1, pa and q share atom 2, atom 2 is independent
	bp := fx.regAndPin(Box{Formula: p, Rel: RelCausal})
	pa := fx.regAndPin(And{Left: p, Right: q})
	rAtom := r
	pfs := []PrefixedFormula{
		{World: 0, Formula: bp},    // depends on {1}
		{World: 0, Formula: pa},    // depends on {1,2}
		{World: 0, Formula: rAtom}, // depends on {3}
	}
	// dna: [0,1] connected via atom 1. 2 is independent of both.
	indices := fx.por.StubbornSet(pfs)
	// Seed=0, closure via dna[0,1]=true adds 1. 2 doesn't accord with 0 or 1.
	if len(indices) < 2 {
		t.Error("should include connected pair [0,1]")
	}
	// Verify 2 is NOT in set (independent)
	for _, idx := range indices {
		if idx == 2 {
			t.Error("index 2 (independent r) should NOT be in stubborn set")
		}
	}
}

func TestPORFilter(t *testing.T) {
	fx := newPORFixtures(t)
	a1 := fx.regAndPin(fx.atom(1))
	a2 := fx.regAndPin(fx.atom(2))
	pfs := []PrefixedFormula{
		{World: 0, Formula: a1},
		{World: 0, Formula: a2},
	}
	filtered := fx.por.Filter(pfs)
	if len(filtered) < 1 {
		t.Error("filter should return at least one formula")
	}
}

func TestPORFootprintFallback(t *testing.T) {
	fx := newPORFixtures(t)
	// Formula NOT in PINS matrix — should use fallback computation
	a := fx.atom(1)
	fp := fx.por.footprint(a)
	if fp == nil {
		t.Error("footprint fallback should return non-nil slice")
	}
}

func TestPORCommuteComplex(t *testing.T) {
	fx := newPORFixtures(t)
	p := fx.regAndPin(fx.atom(1))
	q := fx.regAndPin(fx.atom(2))
	// □(p∧q) and ◇p: □(p∧q) depends on {1,2}, ◇p depends on {1} → overlap → no commute
	complex := fx.regAndPin(Box{Formula: fx.regAndPin(And{Left: p, Right: q}), Rel: RelCausal})
	simple := fx.regAndPin(Diamond{Formula: p, Rel: RelCausal})
	if fx.por.Commute(complex, simple) {
		t.Error("□(p∧q) and ◇p should NOT commute (shared atom 1)")
	}
}

func TestPORStubbornSetEmpty(t *testing.T) {
	fx := newPORFixtures(t)
	indices := fx.por.StubbornSet(nil)
	if len(indices) != 0 {
		t.Error("empty formulas should return empty stubborn set")
	}
}

func TestPORStubbornSetTwo(t *testing.T) {
	fx := newPORFixtures(t)
	p := fx.regAndPin(fx.atom(1))
	pfs := []PrefixedFormula{
		{World: 0, Formula: p},
		{World: 0, Formula: p}, // same atom — dependent
	}
	indices := fx.por.StubbornSet(pfs)
	if len(indices) != 2 {
		t.Errorf("two dependent formulas: got %d, want 2", len(indices))
	}
}

func TestPORCommuteDisconnectedGroups(t *testing.T) {
	fx := newPORFixtures(t)
	a1 := fx.regAndPin(fx.atom(1))
	a2 := fx.regAndPin(fx.atom(2))
	a3 := fx.regAndPin(fx.atom(3))
	a4 := fx.regAndPin(fx.atom(4))
	// Group A: depends on {1,2}, Group B: depends on {3,4} → commute
	gA := fx.regAndPin(And{Left: a1, Right: a2})
	gB := fx.regAndPin(And{Left: a3, Right: a4})
	if !fx.por.Commute(gA, gB) {
		t.Error("disjoint groups {1,2} and {3,4} should commute")
	}
}

func TestPORStubbornSetPreservesAllWhenConnected(t *testing.T) {
	fx := newPORFixtures(t)
	p := fx.regAndPin(fx.atom(1))
	q := fx.regAndPin(fx.atom(2))
	bridge := fx.regAndPin(And{Left: p, Right: q})
	pfs := []PrefixedFormula{
		{World: 0, Formula: p},      // {1}
		{World: 0, Formula: bridge}, // {1,2}
		{World: 0, Formula: q},      // {2}
	}
	// All connected via the bridge → all should be in stubborn set
	indices := fx.por.StubbornSet(pfs)
	if len(indices) != 3 {
		t.Errorf("all connected via bridge: got %d, want 3", len(indices))
	}
}

func TestPORStubbornSetNonTrivial(t *testing.T) {
	fx := newPORFixtures(t)
	// Five formulas: two independent groups of 2 and 3, connected within groups
	p := fx.regAndPin(fx.atom(1))
	q := fx.regAndPin(fx.atom(2))
	r := fx.regAndPin(fx.atom(3))
	s := fx.regAndPin(fx.atom(4))
	tv := fx.regAndPin(fx.atom(5))
	pfs := []PrefixedFormula{
		{World: 0, Formula: p},                     // {1}
		{World: 0, Formula: fx.regAndPin(And{Left: p, Right: q})}, // {1,2}
		{World: 0, Formula: r},                     // {3}
		{World: 0, Formula: fx.regAndPin(And{Left: r, Right: s})}, // {3,4}
		{World: 0, Formula: tv},                    // {5}
	}
	indices := fx.por.StubbornSet(pfs)
	// Seed=0, closure adds 1 (dna[0,1]). 2-4 independent of {0,1}
	if len(indices) != 2 {
		t.Errorf("two connected groups + independent: got %d, want 2", len(indices))
	}
}

func TestPORClosureChain(t *testing.T) {
	fx := newPORFixtures(t)
	// Chain: f0={1,2}, f1={2,3}, f2={3,4}, f3={5}
	// dna: 0-1 (share 2), 1-2 (share 3). 0-2 not directly connected, but transitive.
	// f3 is independent of all.
	a1 := fx.regAndPin(fx.atom(1))
	a2 := fx.regAndPin(fx.atom(2))
	a3 := fx.regAndPin(fx.atom(3))
	a4 := fx.regAndPin(fx.atom(4))
	a5 := fx.regAndPin(fx.atom(5))
	pfs := []PrefixedFormula{
		{World: 0, Formula: fx.regAndPin(And{Left: a1, Right: a2})}, // {1,2}
		{World: 0, Formula: fx.regAndPin(And{Left: a2, Right: a3})}, // {2,3}
		{World: 0, Formula: fx.regAndPin(And{Left: a3, Right: a4})}, // {3,4}
		{World: 0, Formula: a5},                                      // {5}
	}
	indices := fx.por.StubbornSet(pfs)
	// Transitive closure via dna: 0→1→2 → all three connected
	if len(indices) != 3 {
		t.Errorf("transitive chain: got %d, want 3 (0,1,2 connected, 3 independent)", len(indices))
	}
}
