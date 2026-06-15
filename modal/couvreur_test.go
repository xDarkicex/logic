package modal

import (
	"testing"

	"github.com/xDarkicex/logic/fuzzy"
	"github.com/xDarkicex/memory"
)

type couvreurFixtures struct {
	pool  *memory.Pool
	arena *memory.Arena
}

func newCouvreurFixtures(t *testing.T) *couvreurFixtures {
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
	return &couvreurFixtures{pool: pool, arena: arena}
}

func makeFrame(t *testing.T, fx *couvreurFixtures) *Frame {
	t.Helper()
	return NewFrame(fx.pool, fx.arena)
}

func TestCouvreurAtom(t *testing.T) {
	fx := newCouvreurFixtures(t)
	cp := NewCouvreurProver(fx.pool, fx.arena)

	sat, model := cp.ProveSatisfiable(Atom{ID: fuzzy.VarID(0)}, makeFrame(t, fx))
	if !sat {
		t.Error("atom should be satisfiable")
	}
	if model == nil {
		t.Fatal("expected model")
	}
}

func TestCouvreurContradiction(t *testing.T) {
	fx := newCouvreurFixtures(t)
	cp := NewCouvreurProver(fx.pool, fx.arena)

	f := And{Left: Atom{ID: fuzzy.VarID(0)}, Right: Not{Formula: Atom{ID: fuzzy.VarID(0)}}}
	sat, _ := cp.ProveSatisfiable(f, makeFrame(t, fx))
	if sat {
		t.Error("P ∧ ¬P should be unsatisfiable")
	}
}

func TestCouvreurNestedAndOr(t *testing.T) {
	fx := newCouvreurFixtures(t)
	cp := NewCouvreurProver(fx.pool, fx.arena)

	// (P ∨ Q) ∧ (¬P ∨ R) — satisfiable (P=true gives true)
	f := And{
		Left:  Or{Left: Atom{ID: fuzzy.VarID(0)}, Right: Atom{ID: fuzzy.VarID(1)}},
		Right: Or{Left: Not{Formula: Atom{ID: fuzzy.VarID(0)}}, Right: Atom{ID: fuzzy.VarID(2)}},
	}
	sat, _ := cp.ProveSatisfiable(f, makeFrame(t, fx))
	if !sat {
		t.Error("(P∨Q)∧(¬P∨R) should be satisfiable")
	}
}

func TestCouvreurBox(t *testing.T) {
	fx := newCouvreurFixtures(t)
	cp := NewCouvreurProver(fx.pool, fx.arena)
	frame := makeFrame(t, fx)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)

	// □P at w0 — requires P at w1. P_atom at w0 doesn't exist.
	f := Box{Formula: Atom{ID: fuzzy.VarID(0)}, Rel: RelCausal}
	sat, _ := cp.ProveSatisfiable(f, frame)
	if !sat {
		t.Error("□P should be satisfiable (P at w1 is unconstrained → can be true)")
	}
}

func TestCouvreurDiamond(t *testing.T) {
	fx := newCouvreurFixtures(t)
	cp := NewCouvreurProver(fx.pool, fx.arena)
	frame := makeFrame(t, fx)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)

	f := Diamond{Formula: Atom{ID: fuzzy.VarID(0)}, Rel: RelCausal}
	sat, _ := cp.ProveSatisfiable(f, frame)
	if !sat {
		t.Error("◇P should be satisfiable")
	}
}

func TestCouvreurDeadStatePruning(t *testing.T) {
	fx := newCouvreurFixtures(t)
	cp := NewCouvreurProver(fx.pool, fx.arena)

	// (P ∧ ¬P) ∨ (Q ∧ ¬Q) — both branches contradictory
	f := Or{
		Left:  And{Left: Atom{ID: 0}, Right: Not{Formula: Atom{ID: 0}}},
		Right: And{Left: Atom{ID: 1}, Right: Not{Formula: Atom{ID: 1}}},
	}
	sat, _ := cp.ProveSatisfiable(f, makeFrame(t, fx))
	if sat {
		t.Error("all branches contradictory should be unsatisfiable")
	}
}

func TestCouvreurComplexNesting(t *testing.T) {
	fx := newCouvreurFixtures(t)
	cp := NewCouvreurProver(fx.pool, fx.arena)

	// □(P → ◇Q) — satisfiable
	f := Box{
		Formula: Implies{
			Antecedent: Atom{ID: fuzzy.VarID(0)},
			Consequent: Diamond{Formula: Atom{ID: fuzzy.VarID(1)}, Rel: RelCausal},
		},
		Rel: RelCausal,
	}
	sat, _ := cp.ProveSatisfiable(f, makeFrame(t, fx))
	if !sat {
		t.Error("□(P → ◇Q) should be satisfiable")
	}
}

func TestCouvreurDoubleNegation(t *testing.T) {
	fx := newCouvreurFixtures(t)
	cp := NewCouvreurProver(fx.pool, fx.arena)

	// ¬¬P — satisfiable iff P is
	f := Not{Formula: Not{Formula: Atom{ID: fuzzy.VarID(0)}}}
	sat, _ := cp.ProveSatisfiable(f, makeFrame(t, fx))
	if !sat {
		t.Error("¬¬P should be satisfiable")
	}
}

func TestCouvreurDeMorganAnd(t *testing.T) {
	fx := newCouvreurFixtures(t)
	cp := NewCouvreurProver(fx.pool, fx.arena)

	// ¬(P∧Q) → ¬P ∨ ¬Q
	f := Not{Formula: And{
		Left: Atom{ID: fuzzy.VarID(0)}, Right: Atom{ID: fuzzy.VarID(1)},
	}}
	sat, _ := cp.ProveSatisfiable(f, makeFrame(t, fx))
	if !sat {
		t.Error("¬(P∧Q) should be satisfiable")
	}
}

func TestCouvreurSameBranchDetection(t *testing.T) {
	fx := newCouvreurFixtures(t)
	cp := NewCouvreurProver(fx.pool, fx.arena)

	// P ∧ P — satisfiable. Couvreur should detect the duplicate formulas
	sat, _ := cp.ProveSatisfiable(And{Left: Atom{ID: 0}, Right: Atom{ID: 0}}, makeFrame(t, fx))
	if !sat {
		t.Error("P ∧ P should be satisfiable")
	}
}

func TestCouvreurOrBranching(t *testing.T) {
	fx := newCouvreurFixtures(t)
	cp := NewCouvreurProver(fx.pool, fx.arena)

	// (P ∧ Q) ∨ (P ∧ R) — satisfiable
	f := Or{
		Left:  And{Left: Atom{ID: 0}, Right: Atom{ID: 1}},
		Right: And{Left: Atom{ID: 0}, Right: Atom{ID: 2}},
	}
	sat, _ := cp.ProveSatisfiable(f, makeFrame(t, fx))
	if !sat {
		t.Error("(P∧Q)∨(P∧R) should be satisfiable")
	}
}

func TestCouvreurNotBox(t *testing.T) {
	fx := newCouvreurFixtures(t)
	cp := NewCouvreurProver(fx.pool, fx.arena)
	frame := makeFrame(t, fx)
	frame.AddWorld()
	frame.AddWorld()

	f := Not{Formula: Box{Formula: Atom{ID: 0}, Rel: RelCausal}}
	// ¬□P → ◇¬P — satisfiable (requires a world where ¬P)
	sat, _ := cp.ProveSatisfiable(f, frame)
	if !sat {
		t.Error("¬□P should be satisfiable")
	}
}

func TestCouvreurNotDiamond(t *testing.T) {
	fx := newCouvreurFixtures(t)
	cp := NewCouvreurProver(fx.pool, fx.arena)
	frame := makeFrame(t, fx)
	frame.AddWorld()

	f := Not{Formula: Diamond{Formula: Atom{ID: 0}, Rel: RelCausal}}
	// ¬◇P → □¬P — satisfiable (vacuously true if no accessible worlds)
	sat, _ := cp.ProveSatisfiable(f, frame)
	if !sat {
		t.Error("¬◇P should be satisfiable")
	}
}

func TestCouvreurMultiComponent(t *testing.T) {
	fx := newCouvreurFixtures(t)
	cp := NewCouvreurProver(fx.pool, fx.arena)

	// □P ∧ ◇Q — satisfiable (needs both a world where P and an accessible world where Q)
	f := And{
		Left:  Box{Formula: Atom{ID: 0}, Rel: RelCausal},
		Right: Diamond{Formula: Atom{ID: 1}, Rel: RelCausal},
	}
	sat, _ := cp.ProveSatisfiable(f, makeFrame(t, fx))
	if !sat {
		t.Error("□P∧◇Q should be satisfiable")
	}
}

func TestCouvreurSCCMerge(t *testing.T) {
	fx := newCouvreurFixtures(t)
	cp := NewCouvreurProver(fx.pool, fx.arena)

	// A formula with structure that creates equivalent branches:
	// (P ∧ Q) ∨ (P ∧ Q) — both branches identical, SCC merge should detect
	f := Or{
		Left:  And{Left: Atom{ID: 0}, Right: Atom{ID: 1}},
		Right: And{Left: Atom{ID: 0}, Right: Atom{ID: 1}},
	}
	sat, _ := cp.ProveSatisfiable(f, makeFrame(t, fx))
	if !sat {
		t.Error("(P∧Q)∨(P∧Q) should be satisfiable")
	}
}
