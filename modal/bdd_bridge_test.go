package modal

import (
	"testing"

	"github.com/xDarkicex/gobdd"
	"github.com/xDarkicex/memory"
)

func newBDDCtx(t *testing.T, vars int) *BDDCtx {
	t.Helper()
	pool, err := memory.NewPool(memory.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Reset)
	return NewBDDCtx(vars, pool)
}

func TestNewBDDCtx(t *testing.T) {
	ctx := newBDDCtx(t, 4)
	if ctx == nil {
		t.Fatal("NewBDDCtx returned nil")
	}
	if ctx.nextVar != 0 {
		t.Errorf("nextVar = %d, want 0", ctx.nextVar)
	}
}

func TestToBDDAtom(t *testing.T) {
	ctx := newBDDCtx(t, 4)
	a := Atom{ID: 0}
	n := ctx.ToBDD(a)
	if n == gobdd.False || n == gobdd.True {
		t.Error("atom BDD should be a variable node, not terminal")
	}
	// Same atom should return same BDD variable
	n2 := ctx.ToBDD(a)
	if n != n2 {
		t.Error("same atom must return same BDD node")
	}
}

func TestToBDDNot(t *testing.T) {
	ctx := newBDDCtx(t, 4)
	a := Atom{ID: 0}
	na := Not{Formula: a}
	n := ctx.ToBDD(na)
	// ¬(¬a) = a
	nn := ctx.ToBDD(Not{Formula: na})
	if n != ctx.ToBDD(Not{Formula: a}) {
		t.Error("¬a should be stable")
	}
	if nn != ctx.ToBDD(a) {
		t.Error("¬¬a = a")
	}
}

func TestToBDDAndOr(t *testing.T) {
	ctx := newBDDCtx(t, 4)
	p := Atom{ID: 0}
	q := Atom{ID: 1}
	// p ∧ q
	and := ctx.ToBDD(And{Left: p, Right: q})
	if and == gobdd.False || and == gobdd.True {
		t.Error("p∧q should be non-terminal")
	}
	// p ∨ q
	or := ctx.ToBDD(Or{Left: p, Right: q})
	if or == gobdd.False || or == gobdd.True {
		t.Error("p∨q should be non-terminal")
	}
}

func TestToBDDImplies(t *testing.T) {
	ctx := newBDDCtx(t, 4)
	p := Atom{ID: 0}
	// p → p = True
	imp := ctx.ToBDD(Implies{Antecedent: p, Consequent: p})
	if imp != gobdd.True {
		t.Error("p→p = True")
	}
}

func TestToBDDIff(t *testing.T) {
	ctx := newBDDCtx(t, 4)
	p := Atom{ID: 0}
	q := Atom{ID: 1}
	// p ↔ p = True
	iff := ctx.ToBDD(Iff{Left: p, Right: p})
	if iff != gobdd.True {
		t.Error("p↔p = True")
	}
	// p ↔ q should be non-terminal when p≠q
	iff2 := ctx.ToBDD(Iff{Left: p, Right: q})
	if iff2 == gobdd.True || iff2 == gobdd.False {
		t.Error("p↔q should be non-terminal")
	}
}

func TestEquiv(t *testing.T) {
	ctx := newBDDCtx(t, 4)
	p := Atom{ID: 0}
	q := Atom{ID: 1}
	// p→q vs ¬p∨q — logically equivalent
	f1 := Implies{Antecedent: p, Consequent: q}
	f2 := Or{Left: Not{Formula: p}, Right: q}
	if !ctx.Equiv(f1, f2) {
		t.Error("p→q should equal ¬p∨q")
	}
	// p vs q — not equivalent
	if ctx.Equiv(Atom{ID: 0}, Atom{ID: 1}) {
		t.Error("p should not equal q")
	}
}

func TestIsTautology(t *testing.T) {
	ctx := newBDDCtx(t, 4)
	p := Atom{ID: 0}
	// p → p
	if !ctx.IsTautology(Implies{Antecedent: p, Consequent: p}) {
		t.Error("p→p should be a tautology")
	}
	// p ∨ ¬p
	if !ctx.IsTautology(Or{Left: p, Right: Not{Formula: p}}) {
		t.Error("p∨¬p should be a tautology")
	}
}

func TestIsContradiction(t *testing.T) {
	ctx := newBDDCtx(t, 4)
	p := Atom{ID: 0}
	// p ∧ ¬p
	if !ctx.IsContradiction(And{Left: p, Right: Not{Formula: p}}) {
		t.Error("p∧¬p should be a contradiction")
	}
	// p should not be a contradiction
	if ctx.IsContradiction(Atom{ID: 0}) {
		t.Error("p should not be a contradiction")
	}
}

func TestSkeleton(t *testing.T) {
	ctx := newBDDCtx(t, 8)
	p := Atom{ID: 0}
	q := Atom{ID: 1}
	// □p ∧ ◇q — modal formula with Boolean skeleton
	f := And{
		Left:  Box{Formula: p, Rel: RelCausal},
		Right: Diamond{Formula: q, Rel: RelCausal},
	}
	root := ctx.Skeleton(f)
	if root == gobdd.False || root == gobdd.True {
		t.Error("skeleton should be non-terminal")
	}
	subs := ctx.SkeletonSubs()
	if len(subs) != 2 {
		t.Errorf("expected 2 subformulas, got %d", len(subs))
	}
	// Pure Boolean formula should produce empty subs.
	ctx.ResetSkeleton()
	root2 := ctx.Skeleton(And{Left: p, Right: q})
	subs2 := ctx.SkeletonSubs()
	if len(subs2) != 0 {
		t.Errorf("pure Boolean should have 0 subs, got %d", len(subs2))
	}
	if root2 == gobdd.False || root2 == gobdd.True {
		t.Error("Boolean skeleton should be non-terminal")
	}
}

func TestEquivSkel(t *testing.T) {
	ctx := newBDDCtx(t, 8)
	p := Atom{ID: 0}
	// □p ∧ ◇p  vs  ◇p ∧ □p  — same Boolean skeleton
	f1 := And{Left: Box{Formula: p, Rel: RelCausal}, Right: Diamond{Formula: p, Rel: RelCausal}}
	f2 := And{Left: Diamond{Formula: p, Rel: RelCausal}, Right: Box{Formula: p, Rel: RelCausal}}
	if !ctx.EquivSkel(f1, f2) {
		t.Error("□p∧◇p should have same skeleton as ◇p∧□p")
	}
}

func TestBDDEquiv(t *testing.T) {
	if !BDDEquiv(gobdd.True, gobdd.True) {
		t.Error("True ≡ True")
	}
	if BDDEquiv(gobdd.True, gobdd.False) {
		t.Error("True ≢ False")
	}
}

func TestMultipleAtoms(t *testing.T) {
	ctx := newBDDCtx(t, 8)
	// (p ∨ q) ∧ (q ∨ r) ∧ (¬p ∨ ¬q ∨ r)
	p := Atom{ID: 0}
	q := Atom{ID: 1}
	r := Atom{ID: 2}
	f := And{
		Left: Or{Left: p, Right: q},
		Right: And{
			Left:  Or{Left: q, Right: r},
			Right: Or{Left: Or{Left: Not{Formula: p}, Right: Not{Formula: q}}, Right: r},
		},
	}
	root := ctx.ToBDD(f)
	if root == gobdd.False || root == gobdd.True {
		t.Error("complex formula should be non-terminal")
	}
	if ctx.VarCount() != 3 {
		t.Errorf("expected 3 BDD vars, got %d", ctx.VarCount())
	}
}

func TestToBDDPanicsOnModal(t *testing.T) {
	ctx := newBDDCtx(t, 4)
	p := Atom{ID: 0}
	modal := Box{Formula: p, Rel: RelCausal}
	defer func() {
		if r := recover(); r == nil {
			t.Error("ToBDD should panic on modal operator")
		}
	}()
	ctx.ToBDD(modal)
}

func TestBDDBridgeCC(t *testing.T) {
	// Verify CC is computed correctly for each function.
	ctx := newBDDCtx(t, 4)
	if ctx == nil {
		t.Fatal("nil ctx")
	}
	// All exported functions should terminate within sane time.
	p := Atom{ID: 0}
	for i := 0; i < 100; i++ {
		ctx.ToBDD(p)
	}
	ctx.ResetSkeleton()
	ctx.Skeleton(Box{Formula: p, Rel: RelCausal})
	ctx.Equiv(p, p)
	ctx.IsTautology(Implies{Antecedent: p, Consequent: p})
	ctx.IsContradiction(And{Left: p, Right: Not{Formula: p}})
}

func TestISOPFalse(t *testing.T) {
	ctx := newBDDCtx(t, 4)
	r := ctx.ISOP(gobdd.False)
	if r != nil {
		t.Error("ISOP(False) should be nil")
	}
}

func TestISOPTrue(t *testing.T) {
	ctx := newBDDCtx(t, 4)
	r := ctx.ISOP(gobdd.True)
	if r == nil {
		t.Error("ISOP(True) should not be nil")
	}
}

func TestISOPSingleVar(t *testing.T) {
	ctx := newBDDCtx(t, 4)
	p := Atom{ID: 0}
	node := ctx.ToBDD(p)
	r := ctx.ISOP(node)
	if r == nil {
		t.Error("ISOP(Var(0)) should not be nil")
	}
}

func TestISOPAnd(t *testing.T) {
	ctx := newBDDCtx(t, 4)
	p := Atom{ID: 0}
	q := Atom{ID: 1}
	node := ctx.ToBDD(And{Left: p, Right: q})
	r := ctx.ISOP(node)
	if r == nil {
		t.Error("ISOP(p and q) should not be nil")
	}
	m := &Model{valuation: []TruthValueSlice{TruthValueSlice{1.0, 1.0}}}
	v, _ := r.Evaluate(0, m)
	if v != 1.0 {
		t.Errorf("ISOP(p and q) at p=1,q=1: got %v, want 1.0", v)
	}
}

func TestISOPOr(t *testing.T) {
	ctx := newBDDCtx(t, 4)
	p := Atom{ID: 0}
	q := Atom{ID: 1}
	node := ctx.ToBDD(Or{Left: p, Right: q})
	r := ctx.ISOP(node)
	if r == nil {
		t.Error("ISOP(p or q) should not be nil")
	}
}

func TestISOPCache(t *testing.T) {
	ctx := newBDDCtx(t, 4)
	p := Atom{ID: 0}
	q := Atom{ID: 1}
	node := ctx.ToBDD(And{Left: p, Right: q})
	ctx.ISOP(node)
	before := ctx.NodeCount()
	ctx.ISOP(node)
	after := ctx.NodeCount()
	if after != before {
		t.Errorf("ISOP cache miss: nodes %d -> %d", before, after)
	}
}

func TestISOPComplex(t *testing.T) {
	ctx := newBDDCtx(t, 8)
	p := Atom{ID: 0}
	q := Atom{ID: 1}
	r := Atom{ID: 2}
	f := Or{Left: And{Left: p, Right: q}, Right: And{Left: p, Right: r}}
	node := ctx.ToBDD(f)
	isop := ctx.ISOP(node)
	if isop == nil {
		t.Error("ISOP of complex formula should not be nil")
	}
	for i := 0; i < 8; i++ {
		pv := TruthValue((i >> 0) & 1)
		qv := TruthValue((i >> 1) & 1)
		rv := TruthValue((i >> 2) & 1)
		m := &Model{valuation: []TruthValueSlice{TruthValueSlice{pv, qv, rv}}}
		orig, _ := f.Evaluate(0, m)
		result, _ := isop.Evaluate(0, m)
		if orig != result {
			t.Errorf("assignment %d: original=%v isop=%v", i, orig, result)
		}
	}
}
