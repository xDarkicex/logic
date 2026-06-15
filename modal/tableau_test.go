package modal

import (
	"testing"

	"github.com/xDarkicex/logic/fuzzy"
	"github.com/xDarkicex/memory"
)

type tableauFixtures struct {
	prover *Prover
	pool   *memory.Pool
	arena  *memory.Arena
}

func newTableauFixtures(t *testing.T) *tableauFixtures {
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
	return &tableauFixtures{
		prover: NewProver(pool, arena),
		pool:   pool,
		arena:  arena,
	}
}

func (fx *tableauFixtures) makeFrame() *Frame {
	return NewFrame(fx.pool, fx.arena)
}

func TestProveSatisfiableAtom(t *testing.T) {
	fx := newTableauFixtures(t)
	defer fx.prover.Close()

	sat, model := fx.prover.ProveSatisfiable(Atom{ID: fuzzy.VarID(0)}, fx.makeFrame())
	if !sat {
		t.Error("atom should be satisfiable")
	}
	if model == nil {
		t.Fatal("expected model")
	}
	tv := model.Truth(0, fuzzy.VarID(0))
	if tv != 1.0 {
		t.Errorf("expected atom true in model, got %v", tv)
	}
}

func TestProveSatisfiableContradiction(t *testing.T) {
	fx := newTableauFixtures(t)
	defer fx.prover.Close()

	// φ ∧ ¬φ should be unsatisfiable
	f := And{Left: Atom{ID: fuzzy.VarID(0)}, Right: Not{Formula: Atom{ID: fuzzy.VarID(0)}}}
	sat, _ := fx.prover.ProveSatisfiable(f, fx.makeFrame())
	if sat {
		t.Error("P ∧ ¬P should be unsatisfiable")
	}
}

func TestProveValidTautology(t *testing.T) {
	fx := newTableauFixtures(t)
	defer fx.prover.Close()

	// P ∨ ¬P should be valid
	f := Or{Left: Atom{ID: fuzzy.VarID(0)}, Right: Not{Formula: Atom{ID: fuzzy.VarID(0)}}}
	if !fx.prover.ProveValid(f, fx.makeFrame()) {
		t.Error("P ∨ ¬P should be valid")
	}
}

func TestProveValidNonTautology(t *testing.T) {
	fx := newTableauFixtures(t)
	defer fx.prover.Close()

	// P ∧ Q is not valid
	f := And{Left: Atom{ID: fuzzy.VarID(0)}, Right: Atom{ID: fuzzy.VarID(1)}}
	if fx.prover.ProveValid(f, fx.makeFrame()) {
		t.Error("P ∧ Q should not be valid")
	}
}

func TestProveEntailsSimple(t *testing.T) {
	fx := newTableauFixtures(t)
	defer fx.prover.Close()

	// P ∧ Q should entail P
	premises := []Formula{And{Left: Atom{ID: fuzzy.VarID(0)}, Right: Atom{ID: fuzzy.VarID(1)}}}
	conclusion := Atom{ID: fuzzy.VarID(0)}
	if !fx.prover.ProveEntails(premises, conclusion, fx.makeFrame()) {
		t.Error("P∧Q should entail P")
	}
}

func TestProveEntailsNonEntailment(t *testing.T) {
	fx := newTableauFixtures(t)
	defer fx.prover.Close()

	// P should NOT entail P∧Q
	premises := []Formula{Atom{ID: fuzzy.VarID(0)}}
	conclusion := And{Left: Atom{ID: fuzzy.VarID(0)}, Right: Atom{ID: fuzzy.VarID(1)}}
	if fx.prover.ProveEntails(premises, conclusion, fx.makeFrame()) {
		t.Error("P should not entail P∧Q")
	}
}

func TestExpandAndRule(t *testing.T) {
	fx := newTableauFixtures(t)
	defer fx.prover.Close()

	n := fx.prover.allocNode()
	n.Formulas = append(n.Formulas, PrefixedFormula{
		World: 0, Formula: And{Left: Atom{ID: fuzzy.VarID(0)}, Right: Atom{ID: fuzzy.VarID(1)}},
	})
	n.Formulas = append(n.Formulas, PrefixedFormula{World: 0, Formula: Atom{ID: fuzzy.VarID(2)}})

	fx.prover.expandAndRule(n, 0, 0)

	if len(n.Formulas) != 3 {
		t.Fatalf("expected 3 formulas, got %d", len(n.Formulas))
	}
	_, ok := n.Formulas[0].Formula.(Atom)
	if !ok {
		t.Error("first formula should be atom(0)")
	}
	_, ok = n.Formulas[1].Formula.(Atom)
	if !ok {
		t.Error("second formula should be atom(2)")
	}
	_, ok = n.Formulas[2].Formula.(Atom)
	if !ok {
		t.Error("third formula should be atom(1)")
	}
}

func TestExpandOrRule(t *testing.T) {
	fx := newTableauFixtures(t)
	defer fx.prover.Close()

	n := fx.prover.allocNode()
	n.Prefix = append(n.Prefix, 0)
	n.Formulas = append(n.Formulas, PrefixedFormula{
		World: 0, Formula: Or{Left: Atom{ID: fuzzy.VarID(0)}, Right: Atom{ID: fuzzy.VarID(1)}},
	})

	stack := []*TableauNode{n}
	fx.prover.expandOrRule(n, 0, 0, &stack)

	if len(stack) != 2 {
		t.Fatalf("expected 2 nodes on stack, got %d", len(stack))
	}
	// Left child (current node) should have atom(0)
	if len(n.Formulas) != 1 {
		t.Fatalf("left child: expected 1 formula, got %d", len(n.Formulas))
	}
	// Right child should have atom(1)
	right := stack[1]
	if len(right.Formulas) != 1 {
		t.Fatalf("right child: expected 1 formula, got %d", len(right.Formulas))
	}
}

func TestExpandNotRuleDoubleNegation(t *testing.T) {
	fx := newTableauFixtures(t)
	defer fx.prover.Close()

	n := fx.prover.allocNode()
	n.Formulas = append(n.Formulas, PrefixedFormula{
		World: 0, Formula: Not{Formula: Not{Formula: Atom{ID: fuzzy.VarID(0)}}},
	})

	fx.prover.expandNotRule(n, 0, 0)

	if _, ok := n.Formulas[0].Formula.(Atom); !ok {
		t.Error("¬¬P should reduce to P")
	}
}

func TestExpandNotRuleDeMorganAnd(t *testing.T) {
	fx := newTableauFixtures(t)
	defer fx.prover.Close()

	n := fx.prover.allocNode()
	n.Formulas = append(n.Formulas, PrefixedFormula{
		World: 0, Formula: Not{Formula: And{
			Left: Atom{ID: fuzzy.VarID(0)}, Right: Atom{ID: fuzzy.VarID(1)},
		}},
	})

	fx.prover.expandNotRule(n, 0, 0)

	// ¬(P∧Q) → ¬P ∨ ¬Q
	if _, ok := n.Formulas[0].Formula.(Or); !ok {
		t.Error("¬(P∧Q) should become Or")
	}
}

func TestExpandNotRuleDeMorganOr(t *testing.T) {
	fx := newTableauFixtures(t)
	defer fx.prover.Close()

	n := fx.prover.allocNode()
	n.Formulas = append(n.Formulas, PrefixedFormula{
		World: 0, Formula: Not{Formula: Or{
			Left: Atom{ID: fuzzy.VarID(0)}, Right: Atom{ID: fuzzy.VarID(1)},
		}},
	})

	fx.prover.expandNotRule(n, 0, 0)

	if len(n.Formulas) != 2 {
		t.Fatalf("¬(P∨Q) should produce 2 formulas, got %d", len(n.Formulas))
	}
}

func TestExpandNotRuleNotBox(t *testing.T) {
	fx := newTableauFixtures(t)
	defer fx.prover.Close()

	n := fx.prover.allocNode()
	n.Formulas = append(n.Formulas, PrefixedFormula{
		World: 0, Formula: Not{Formula: Box{Formula: Atom{ID: fuzzy.VarID(0)}, Rel: RelCausal}},
	})

	fx.prover.expandNotRule(n, 0, 0)

	if _, ok := n.Formulas[0].Formula.(Diamond); !ok {
		t.Error("¬□P should become ◇¬P")
	}
}

func TestExpandNotRuleNotDiamond(t *testing.T) {
	fx := newTableauFixtures(t)
	defer fx.prover.Close()

	n := fx.prover.allocNode()
	n.Formulas = append(n.Formulas, PrefixedFormula{
		World: 0, Formula: Not{Formula: Diamond{Formula: Atom{ID: fuzzy.VarID(0)}, Rel: RelCausal}},
	})

	fx.prover.expandNotRule(n, 0, 0)

	if _, ok := n.Formulas[0].Formula.(Box); !ok {
		t.Error("¬◇P should become □¬P")
	}
}

func TestExpandBoxRule(t *testing.T) {
	fx := newTableauFixtures(t)
	defer fx.prover.Close()

	arena2, _ := memory.NewArena(1024 * 1024)
	defer arena2.Free()
	frame := NewFrame(fx.pool, arena2)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	w2 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)
	frame.AddRelation(w0, w2, RelCausal)

	n := fx.prover.allocNode()
	n.Formulas = append(n.Formulas, PrefixedFormula{
		World: w0, Formula: Box{Formula: Atom{ID: fuzzy.VarID(0)}, Rel: RelCausal},
	})

	fx.prover.expandBoxRule(n, 0, w0, frame)

	if len(n.Formulas) != 2 {
		t.Fatalf("□ should expand to 2 accessible worlds, got %d", len(n.Formulas))
	}
	if n.Formulas[0].World != w1 || n.Formulas[1].World != w2 {
		t.Error("wrong target worlds")
	}
}

func TestExpandBoxRuleNoAccessible(t *testing.T) {
	fx := newTableauFixtures(t)
	defer fx.prover.Close()

	arena2, _ := memory.NewArena(1024 * 1024)
	defer arena2.Free()
	frame := NewFrame(fx.pool, arena2)
	w0 := frame.AddWorld()

	n := fx.prover.allocNode()
	n.Formulas = append(n.Formulas, PrefixedFormula{
		World: w0, Formula: Box{Formula: Atom{ID: fuzzy.VarID(0)}, Rel: RelCausal},
	})

	fx.prover.expandBoxRule(n, 0, w0, frame)

	// No accessible worlds — the formula should be removed
	if len(n.Formulas) != 0 {
		t.Errorf("vacuously true box should be removed, got %d formulas", len(n.Formulas))
	}
}

func TestExpandDiamondRule(t *testing.T) {
	fx := newTableauFixtures(t)
	defer fx.prover.Close()

	arena2, _ := memory.NewArena(1024 * 1024)
	defer arena2.Free()
	frame := NewFrame(fx.pool, arena2)
	w0 := frame.AddWorld()

	n := fx.prover.allocNode()
	n.Formulas = append(n.Formulas, PrefixedFormula{
		World: w0, Formula: Diamond{Formula: Atom{ID: fuzzy.VarID(0)}, Rel: RelCausal},
	})

	fx.prover.expandDiamondRule(n, 0, w0, frame)

	if frame.WorldCount() != 2 {
		t.Fatalf("◇ should create a new world, got %d worlds", frame.WorldCount())
	}
	if len(n.Prefix) != 1 {
		t.Errorf("prefix should contain the new world, got %d worlds", len(n.Prefix))
	}
}

func TestIsContradictory(t *testing.T) {
	// P at world 0, ¬P at world 0 → contradictory
	fs := []PrefixedFormula{
		{World: 0, Formula: Atom{ID: fuzzy.VarID(0)}},
		{World: 0, Formula: Not{Formula: Atom{ID: fuzzy.VarID(0)}}},
	}
	if !isContradictory(fs) {
		t.Error("P and ¬P at same world should be contradictory")
	}

	// P at world 0, ¬P at world 1 → not contradictory
	fs = []PrefixedFormula{
		{World: 0, Formula: Atom{ID: fuzzy.VarID(0)}},
		{World: 1, Formula: Not{Formula: Atom{ID: fuzzy.VarID(0)}}},
	}
	if isContradictory(fs) {
		t.Error("P and ¬P at different worlds should NOT be contradictory")
	}

	// P at world 0, Q at world 0 → not contradictory
	fs = []PrefixedFormula{
		{World: 0, Formula: Atom{ID: fuzzy.VarID(0)}},
		{World: 0, Formula: Atom{ID: fuzzy.VarID(1)}},
	}
	if isContradictory(fs) {
		t.Error("P and Q should not be contradictory")
	}
}

func TestIsComplete(t *testing.T) {
	fs := []PrefixedFormula{
		{World: 0, Formula: Atom{ID: fuzzy.VarID(0)}},
		{World: 0, Formula: Not{Formula: Atom{ID: fuzzy.VarID(1)}}},
	}
	if !isComplete(fs) {
		t.Error("all literals should be complete")
	}

	fs = append(fs, PrefixedFormula{World: 0, Formula: And{
		Left: Atom{ID: fuzzy.VarID(0)}, Right: Atom{ID: fuzzy.VarID(1)},
	}})
	if isComplete(fs) {
		t.Error("non-literal formula should be incomplete")
	}
}

func TestProveSatisfiableBoxDiamond(t *testing.T) {
	fx := newTableauFixtures(t)
	defer fx.prover.Close()

	// □◇P — should be satisfiable
	f := Box{Formula: Diamond{Formula: Atom{ID: fuzzy.VarID(0)}, Rel: RelCausal}, Rel: RelCausal}
	sat, _ := fx.prover.ProveSatisfiable(f, fx.makeFrame())
	if !sat {
		t.Error("□◇P should be satisfiable")
	}
}

func TestProveSatisfiableModalComplex(t *testing.T) {
	fx := newTableauFixtures(t)
	defer fx.prover.Close()

	// □(P → ◇Q) — should be satisfiable
	f := Box{
		Formula: Implies{
			Antecedent: Atom{ID: fuzzy.VarID(0)},
			Consequent: Diamond{Formula: Atom{ID: fuzzy.VarID(1)}, Rel: RelCausal},
		},
		Rel: RelCausal,
	}
	sat, _ := fx.prover.ProveSatisfiable(f, fx.makeFrame())
	if !sat {
		t.Error("□(P → ◇Q) should be satisfiable")
	}
}

func TestProveSatisfiableEmpty(t *testing.T) {
	fx := newTableauFixtures(t)
	defer fx.prover.Close()

	// A single atom should trivially be satisfiable
	sat, model := fx.prover.ProveSatisfiable(Atom{ID: fuzzy.VarID(7)}, fx.makeFrame())
	if !sat {
		t.Error("atom should be satisfiable")
	}
	if model == nil {
		t.Fatal("expected model")
	}
}

func TestExtractModel(t *testing.T) {
	fx := newTableauFixtures(t)
	defer fx.prover.Close()

	arena2, _ := memory.NewArena(1024 * 1024)
	defer arena2.Free()
	frame := NewFrame(fx.pool, arena2)
	frame.AddWorld()

	n := fx.prover.allocNode()
	n.Prefix = append(n.Prefix, 0)
	n.Formulas = append(n.Formulas,
		PrefixedFormula{World: 0, Formula: Atom{ID: fuzzy.VarID(0)}},
		PrefixedFormula{World: 0, Formula: Not{Formula: Atom{ID: fuzzy.VarID(1)}}},
		PrefixedFormula{World: 0, Formula: Atom{ID: fuzzy.VarID(2)}},
	)

	model := extractModel(n, frame, fx.pool)
	if model.Truth(0, fuzzy.VarID(0)) != 1.0 {
		t.Error("atom 0 should be true")
	}
	if model.Truth(0, fuzzy.VarID(2)) != 1.0 {
		t.Error("atom 2 should be true")
	}
	if model.NumVars() != 3 {
		t.Errorf("expected 3 vars, got %d", model.NumVars())
	}
}
