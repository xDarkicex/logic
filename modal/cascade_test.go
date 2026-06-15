package modal

import (
	"testing"

	"github.com/xDarkicex/logic/fuzzy"
	"github.com/xDarkicex/memory"
)

type cascadeFixtures struct {
	pool  *memory.Pool
	arena *memory.Arena
	reg   *Registry
	casc  *Cascade
}

func newCascadeFixtures(t *testing.T) *cascadeFixtures {
	t.Helper()
	pool, err := memory.NewPool(memory.DefaultConfig())
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	arena, err := memory.NewArena(1024 * 1024)
	if err != nil {
		t.Fatalf("arena: %v", err)
	}
	t.Cleanup(func() {
		pool.Reset()
		arena.Free()
	})
	reg := NewRegistry(128, pool)
	rw := NewRewriter(reg, pool)
	return &cascadeFixtures{
		pool:  pool,
		arena: arena,
		reg:   reg,
		casc:  NewCascade(reg, rw, pool, arena),
	}
}

func (fx *cascadeFixtures) a(id uint32) Formula { return fx.reg.Intern(Atom{ID: fuzzy.VarID(id)}) }

// --- Tier 1: Syntactic shortcut tests ---

func TestCascadeTier1Tautology(t *testing.T) {
	fx := newCascadeFixtures(t)
	// P ∨ ¬P — trivially true
	p := fx.a(1)
	f := fx.reg.Intern(Or{Left: p, Right: Not{Formula: p}})
	sat, _ := fx.casc.Prove(f)
	if !sat {
		t.Error("P ∨ ¬P should be sat via Tier 1")
	}
}

func TestCascadeTier1Contradiction(t *testing.T) {
	fx := newCascadeFixtures(t)
	// P ∧ ¬P — trivially false
	p := fx.a(1)
	f := fx.reg.Intern(And{Left: p, Right: Not{Formula: p}})
	sat, _ := fx.casc.Prove(f)
	if sat {
		t.Error("P ∧ ¬P should be unsat via Tier 1")
	}
}

func TestCascadeTier1ImplicationIdentity(t *testing.T) {
	fx := newCascadeFixtures(t)
	p := fx.a(1)
	f := fx.reg.Intern(Implies{Antecedent: p, Consequent: p})
	sat, _ := fx.casc.Prove(f)
	if !sat {
		t.Error("P → P should be sat via Tier 1")
	}
}

func TestCascadeTier1NotTautology(t *testing.T) {
	fx := newCascadeFixtures(t)
	// □(P ∨ ¬P) — tautology under box
	p := fx.a(1)
	inner := fx.reg.Intern(Or{Left: p, Right: Not{Formula: p}})
	f := fx.reg.Intern(Box{Formula: inner, Rel: RelCausal})
	// Tier 1 won't directly recognize this as a tautology (Box wrapper)
	// Falls through to Tier 2/3
	sat, _ := fx.casc.Prove(f)
	if !sat {
		t.Error("□(P ∨ ¬P) should be sat (Tier 2 or 3)")
	}
}

func TestCascadeTier1SimpleAtom(t *testing.T) {
	fx := newCascadeFixtures(t)
	sat, _ := fx.casc.Prove(fx.a(1))
	if !sat {
		t.Error("single atom should be sat")
	}
}

// --- Tier 2: Shallow check tests ---

func TestCascadeTier2ShallowSat(t *testing.T) {
	fx := newCascadeFixtures(t)
	// P ∧ Q — satisfiable in a 1-world model
	p := fx.a(1)
	q := fx.a(2)
	f := fx.reg.Intern(And{Left: p, Right: q})
	sat, model := fx.casc.Prove(f)
	if !sat {
		t.Error("P ∧ Q should be sat in shallow check")
	}
	if model == nil {
		t.Fatal("expected model")
	}
}

func TestCascadeTier2ShallowUnsat(t *testing.T) {
	fx := newCascadeFixtures(t)
	// P ∧ ¬P ∧ Q — contradiction in any model
	p := fx.a(1)
	q := fx.a(2)
	f := fx.reg.Intern(And{
		Left:  And{Left: p, Right: Not{Formula: p}},
		Right: q,
	})
	sat, _ := fx.casc.Prove(f)
	if sat {
		t.Error("P ∧ ¬P ∧ Q should be unsat")
	}
}

func TestCascadeTier2Box(t *testing.T) {
	fx := newCascadeFixtures(t)
	// □P in 1-world frame with self-loop — satisfiable
	p := fx.a(1)
	f := fx.reg.Intern(Box{Formula: p, Rel: RelCausal})
	sat, _ := fx.casc.Prove(f)
	if !sat {
		t.Error("□P in reflexive 1-world frame should be sat")
	}
}

// --- Tier 3: Full Couvreur tests ---

func TestCascadeTier3Complex(t *testing.T) {
	fx := newCascadeFixtures(t)
	// □(P → ◇Q) ∧ ◇P — requires multiple worlds
	p := fx.a(1)
	q := fx.a(2)
	dq := fx.reg.Intern(Diamond{Formula: q, Rel: RelCausal})
	imp := fx.reg.Intern(Implies{Antecedent: p, Consequent: dq})
	bx := fx.reg.Intern(Box{Formula: imp, Rel: RelCausal})
	dp := fx.reg.Intern(Diamond{Formula: p, Rel: RelCausal})
	f := fx.reg.Intern(And{Left: bx, Right: dp})
	sat, _ := fx.casc.Prove(f)
	if !sat {
		t.Error("□(P→◇Q) ∧ ◇P should be sat (may need Couvreur)")
	}
}

// --- isTrivialTrue/False unit tests ---

func TestIsTrivialTrue(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	reg := NewRegistry(16, pool)
	p := reg.Intern(Atom{ID: fuzzy.VarID(1)})

	if !isTrivialTrue(reg.Intern(Or{Left: p, Right: Not{Formula: p}})) {
		t.Error("P ∨ ¬P is trivial true")
	}
	if isTrivialTrue(reg.Intern(And{Left: p, Right: Not{Formula: p}})) {
		t.Error("P ∧ ¬P is NOT trivial true")
	}
	if isTrivialTrue(p) {
		t.Error("atom is NOT trivial true")
	}
}

func TestIsTrivialFalse(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	reg := NewRegistry(16, pool)
	p := reg.Intern(Atom{ID: fuzzy.VarID(1)})

	if !isTrivialFalse(reg.Intern(And{Left: p, Right: Not{Formula: p}})) {
		t.Error("P ∧ ¬P is trivial false")
	}
	if isTrivialFalse(reg.Intern(Or{Left: p, Right: Not{Formula: p}})) {
		t.Error("P ∨ ¬P is NOT trivial false")
	}
	if isTrivialFalse(p) {
		t.Error("atom is NOT trivial false")
	}
}

// --- isNegationOf tests ---

func TestIsNegationOf(t *testing.T) {
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	reg := NewRegistry(16, pool)
	p := reg.Intern(Atom{ID: fuzzy.VarID(1)})
	q := reg.Intern(Atom{ID: fuzzy.VarID(2)})
	np := reg.Intern(Not{Formula: p})

	if !isNegationOf(np, p) {
		t.Error("¬P is negation of P")
	}
	if isNegationOf(p, q) {
		t.Error("P is NOT negation of Q")
	}
	if isNegationOf(p, p) {
		t.Error("P is NOT negation of P (no Not wrapper)")
	}
}

func TestCascadeAllTiersAgree(t *testing.T) {
	fx := newCascadeFixtures(t)
	// Formula that should get same result at all tiers
	p := fx.a(1)
	f := fx.reg.Intern(And{Left: p, Right: p}) // P ∧ P ≡ P — satisfiable

	sat1, _ := fx.casc.Prove(f)
	// Run same formula again to verify consistency
	sat2, _ := fx.casc.Prove(f)
	if sat1 != sat2 {
		t.Error("same formula should give consistent results")
	}
	if !sat1 {
		t.Error("P ∧ P should be sat")
	}
}

func TestCascadeDiamondSat(t *testing.T) {
	fx := newCascadeFixtures(t)
	// ◇P should be satisfiable
	p := fx.a(1)
	f := fx.reg.Intern(Diamond{Formula: p, Rel: RelCausal})
	sat, _ := fx.casc.Prove(f)
	if !sat {
		t.Error("◇P should be sat")
	}
}

func TestCascadeNegatedTautology(t *testing.T) {
	fx := newCascadeFixtures(t)
	// ¬(P ∨ ¬P) should be unsatisfiable
	p := fx.a(1)
	inner := fx.reg.Intern(Or{Left: p, Right: Not{Formula: p}})
	f := fx.reg.Intern(Not{Formula: inner})
	sat, _ := fx.casc.Prove(f)
	if sat {
		t.Error("¬(P ∨ ¬P) should be unsat")
	}
}
