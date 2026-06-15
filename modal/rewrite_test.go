package modal

import (
	"testing"

	"github.com/xDarkicex/logic/fuzzy"
	"github.com/xDarkicex/memory"
)

type rewriteFixtures struct {
	pool *memory.Pool
	reg  *Registry
	rw   *Rewriter
}

func newRewriteFixtures(t *testing.T) *rewriteFixtures {
	t.Helper()
	pool, err := memory.NewPool(memory.DefaultConfig())
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Reset)
	reg := NewRegistry(128, pool)
	rw := NewRewriter(reg, pool)
	return &rewriteFixtures{
		pool: pool, reg: reg, rw: rw,
	}
}

func (fx *rewriteFixtures) a(id uint32) Formula    { return fx.reg.Intern(Atom{ID: fuzzy.VarID(id)}) }
func (fx *rewriteFixtures) bx(f Formula) Formula   { return fx.reg.Intern(Box{Formula: f, Rel: RelCausal}) }
func (fx *rewriteFixtures) dia(f Formula) Formula  { return fx.reg.Intern(Diamond{Formula: f, Rel: RelCausal}) }
func (fx *rewriteFixtures) and(a, b Formula) Formula { return fx.reg.Intern(And{Left: a, Right: b}) }
func (fx *rewriteFixtures) or(a, b Formula) Formula  { return fx.reg.Intern(Or{Left: a, Right: b}) }
// --- §14 simplify tests ---

func TestSimplifyIdempotentAtom(t *testing.T) {
	fx := newRewriteFixtures(t)
	if fx.rw.Simplify(fx.a(1)) != fx.a(1) {
		t.Error("atom should simplify to itself")
	}
}

func TestSimplifyDiamondEventual(t *testing.T) {
	fx := newRewriteFixtures(t)
	d1 := fx.dia(fx.a(1))
	d2 := fx.dia(d1)
	// ◇(◇a) → ◇a: result should be d1 (still a Diamond, but the inner one)
	if fx.rw.Simplify(d2) != d1 {
		t.Error("◇(◇a) should collapse to inner ◇a")
	}
}

func TestSimplifyBoxUniversal(t *testing.T) {
	fx := newRewriteFixtures(t)
	b1 := fx.bx(fx.a(1))
	b2 := fx.bx(b1)
	if fx.rw.Simplify(b2) != b1 {
		t.Error("□(□a) should collapse to inner □a")
	}
}

func TestSimplifyGF(t *testing.T) {
	fx := newRewriteFixtures(t)
	d := fx.dia(fx.a(1))
	gf := fx.bx(d)
	if fx.rw.Simplify(gf) != d {
		t.Error("□◇a should simplify to ◇a (same pointer)")
	}
}

func TestSimplifyFG(t *testing.T) {
	fx := newRewriteFixtures(t)
	b := fx.bx(fx.a(1))
	fg := fx.dia(b)
	if fx.rw.Simplify(fg) != b {
		t.Error("◇□a should simplify to □a (same pointer)")
	}
}

func TestSimplifyDiamondAtomStays(t *testing.T) {
	fx := newRewriteFixtures(t)
	// ◇a: atom not eventual → stays ◇
	if _, ok := fx.rw.Simplify(fx.dia(fx.a(1))).(Diamond); !ok {
		t.Error("◇(atom) should stay ◇")
	}
}

func TestSimplifyBoxAtomStays(t *testing.T) {
	fx := newRewriteFixtures(t)
	// □a: atom not universal → stays □
	if _, ok := fx.rw.Simplify(fx.bx(fx.a(1))).(Box); !ok {
		t.Error("□(atom) should stay □")
	}
}

func TestSimplifyTripleBox(t *testing.T) {
	fx := newRewriteFixtures(t)
	// □□□a: innermost □a universal → middle collapses → outer collapses
	b3 := fx.bx(fx.bx(fx.bx(fx.a(1))))
	result := fx.rw.Simplify(b3)
	b, ok := result.(Box)
	if !ok {
		t.Fatal("should be Box")
	}
	if _, ok := b.Formula.(Atom); !ok {
		t.Error("remaining Box should contain atom")
	}
}

func TestSimplifyAnd(t *testing.T) {
	fx := newRewriteFixtures(t)
	// ◇a ∧ b: should preserve And structure
	r := fx.rw.Simplify(fx.and(fx.dia(fx.a(1)), fx.a(2)))
	if _, ok := r.(And); !ok {
		t.Error("◇a ∧ b should still be And")
	}
}

func TestSimplifyOr(t *testing.T) {
	fx := newRewriteFixtures(t)
	// □a ∨ b: should preserve Or structure
	r := fx.rw.Simplify(fx.or(fx.bx(fx.a(1)), fx.a(2)))
	if _, ok := r.(Or); !ok {
		t.Error("□a ∨ b should still be Or")
	}
}

// --- §15 split tests ---

func TestSplitTwoIndependent(t *testing.T) {
	fx := newRewriteFixtures(t)
	a1 := fx.a(1); a2 := fx.a(2); a3 := fx.a(3); a4 := fx.a(4)
	g1 := fx.and(a1, a2)
	g2 := fx.and(a3, a4)
	conj := fx.and(g1, g2)
	if len(fx.rw.SplitConjunction(conj)) != 2 {
		t.Errorf("should split into 2, got %d", len(fx.rw.SplitConjunction(conj)))
	}
}

func TestSplitDependent(t *testing.T) {
	fx := newRewriteFixtures(t)
	a1 := fx.a(1); a2 := fx.a(2)
	ab := fx.or(a1, a2)
	c := fx.a(1) // shares atom 1 → overlap
	if len(fx.rw.SplitConjunction(fx.and(ab, c))) != 1 {
		t.Error("overlapping atoms should prevent split")
	}
}

func TestSplitSingle(t *testing.T) {
	fx := newRewriteFixtures(t)
	if len(fx.rw.SplitConjunction(fx.a(1))) != 1 {
		t.Error("single formula should not split")
	}
}

func TestSplitThree(t *testing.T) {
	fx := newRewriteFixtures(t)
	g1 := fx.and(fx.a(1), fx.a(2))
	g2 := fx.and(fx.a(3), fx.a(4))
	g3 := fx.a(5)
	// And(g1, And(g2, g3)): g1={1,2}, And(g2,g3)={3,4,5} → independent → 2 components
	conj := fx.and(g1, fx.and(g2, g3))
	if len(fx.rw.SplitConjunction(conj)) != 2 {
		t.Errorf("2 independent groups, got %d", len(fx.rw.SplitConjunction(conj)))
	}
}

func TestSimplifyPropsAtom(t *testing.T) {
	fx := newRewriteFixtures(t)
	_, ev, un := fx.rw.simplifyProps(fx.a(1))
	if ev || un {
		t.Errorf("atom: ev=%v un=%v, want false,false", ev, un)
	}
}

func TestSimplifyPropsDiamond(t *testing.T) {
	fx := newRewriteFixtures(t)
	_, ev, un := fx.rw.simplifyProps(fx.dia(fx.a(1)))
	if !ev || un {
		t.Errorf("◇atom: ev=%v un=%v, want true,false", ev, un)
	}
}

func TestSimplifyPropsBox(t *testing.T) {
	fx := newRewriteFixtures(t)
	_, ev, un := fx.rw.simplifyProps(fx.bx(fx.a(1)))
	if ev || !un {
		t.Errorf("□atom: ev=%v un=%v, want false,true", ev, un)
	}
}

func TestSimplifyPropsAnd(t *testing.T) {
	fx := newRewriteFixtures(t)
	// ◇a ∧ □b: left ev=true un=false, right ev=false un=true
	// AND: ev = lev||rev = true, un = lun&&run = false&&true = false
	_, ev, un := fx.rw.simplifyProps(fx.and(fx.dia(fx.a(1)), fx.bx(fx.a(2))))
	if !ev {
		t.Error("◇a∧□b: should be eventual")
	}
	if un {
		t.Error("◇a∧□b: should NOT be universal (◇a inherits atom's un=false)")
	}
}

func TestSimplifyPropsOr(t *testing.T) {
	fx := newRewriteFixtures(t)
	// ◇a ∨ □b: left ev=true un=false, right ev=false un=true
	// OR: ev = lev&&rev = true&&false = false, un = lun||run = false||true = true
	_, ev, un := fx.rw.simplifyProps(fx.or(fx.dia(fx.a(1)), fx.bx(fx.a(2))))
	if ev {
		t.Error("◇a∨□b: should NOT be eventual (□b is not)")
	}
	if !un {
		t.Error("◇a∨□b: should be universal (right is)")
	}
}
