package modal

import (
	"testing"

	"github.com/xDarkicex/logic/fuzzy"
	"github.com/xDarkicex/memory"
)

// testFixtures holds pre-allocated memory for modal tests.
type testFixtures struct {
	pool  *memory.Pool
	arena *memory.Arena
}

func newFixtures(t *testing.T) *testFixtures {
	t.Helper()
	pool, err := memory.NewPool(memory.DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	arena, err := memory.NewArena(1024 * 1024)
	if err != nil {
		t.Fatalf("failed to create arena: %v", err)
	}
	t.Cleanup(func() {
		pool.Reset()
		arena.Free()
	})
	return &testFixtures{pool: pool, arena: arena}
}

// setup2WorldModel creates a 2-world frame with a single relation type and 2 atoms.
func setup2WorldModel(t *testing.T, fx *testFixtures) (*Frame, *Model) {
	t.Helper()
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)
	model := NewModel(frame, 2, fx.pool, fx.arena)
	model.SetTruth(w0, fuzzy.VarID(0), 1.0)
	model.SetTruth(w0, fuzzy.VarID(1), 0.0)
	model.SetTruth(w1, fuzzy.VarID(0), 0.0)
	model.SetTruth(w1, fuzzy.VarID(1), 1.0)
	return frame, model
}

// --- Atom tests ---

func TestAtomEvaluate(t *testing.T) {
	fx := newFixtures(t)
	_, model := setup2WorldModel(t, fx)

	a := Atom{ID: fuzzy.VarID(0)}
	tv, err := a.Evaluate(World(0), model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 1.0 {
		t.Errorf("Atom(0) at w0: got %v, want 1.0", tv)
	}
	tv, err = a.Evaluate(World(1), model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.0 {
		t.Errorf("Atom(0) at w1: got %v, want 0.0", tv)
	}
}

func TestAtomEvaluateOutOfBounds(t *testing.T) {
	fx := newFixtures(t)
	_, model := setup2WorldModel(t, fx)

	a := Atom{ID: fuzzy.VarID(99)} // beyond numVars
	tv, _ := a.Evaluate(World(0), model)
	if tv != 0.0 {
		t.Errorf("out-of-bounds atom: got %v, want 0.0", tv)
	}
}

// --- Box tests ---

func TestBoxEvaluate(t *testing.T) {
	fx := newFixtures(t)
	_, model := setup2WorldModel(t, fx)

	// At w0, only w1 is accessible. w1 has atom(0)=0.0, atom(1)=1.0
	// □atom(0) at w0 = min over {w1} = 0.0
	b := Box{Formula: Atom{ID: fuzzy.VarID(0)}, Rel: RelCausal}
	tv, err := b.Evaluate(World(0), model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.0 {
		t.Errorf("Box(atom0) at w0: got %v, want 0.0", tv)
	}

	// □atom(1) at w0 = min over {w1} = 1.0
	b = Box{Formula: Atom{ID: fuzzy.VarID(1)}, Rel: RelCausal}
	tv, err = b.Evaluate(World(0), model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 1.0 {
		t.Errorf("Box(atom1) at w0: got %v, want 1.0", tv)
	}
}

func TestBoxEvaluateNoAccessible(t *testing.T) {
	fx := newFixtures(t)
	_, model := setup2WorldModel(t, fx)

	// w1 has no outgoing edges — □ is vacuously true (min over empty set = 1.0)
	b := Box{Formula: Atom{ID: fuzzy.VarID(0)}, Rel: RelCausal}
	tv, err := b.Evaluate(World(1), model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 1.0 {
		t.Errorf("Box with no accessible worlds: got %v, want 1.0", tv)
	}
}

// --- Diamond tests ---

func TestDiamondEvaluate(t *testing.T) {
	fx := newFixtures(t)
	_, model := setup2WorldModel(t, fx)

	// At w0, w1 is accessible. w1 has atom(0)=0.0
	// ◇atom(0) at w0 = max over {w1} = 0.0
	d := Diamond{Formula: Atom{ID: fuzzy.VarID(0)}, Rel: RelCausal}
	tv, err := d.Evaluate(World(0), model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.0 {
		t.Errorf("Diamond(atom0) at w0: got %v, want 0.0", tv)
	}

	// ◇atom(1) at w0 = max over {w1} = 1.0
	d = Diamond{Formula: Atom{ID: fuzzy.VarID(1)}, Rel: RelCausal}
	tv, err = d.Evaluate(World(0), model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 1.0 {
		t.Errorf("Diamond(atom1) at w0: got %v, want 1.0", tv)
	}
}

func TestDiamondEvaluateNoAccessible(t *testing.T) {
	fx := newFixtures(t)
	_, model := setup2WorldModel(t, fx)

	// w1 has no outgoing edges — ◇ is vacuously false (max over empty set = 0.0)
	d := Diamond{Formula: Atom{ID: fuzzy.VarID(0)}, Rel: RelCausal}
	tv, err := d.Evaluate(World(1), model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.0 {
		t.Errorf("Diamond with no accessible worlds: got %v, want 0.0", tv)
	}
}

// --- Not tests ---

func TestNotEvaluate(t *testing.T) {
	fx := newFixtures(t)
	_, model := setup2WorldModel(t, fx)

	n := Not{Formula: Atom{ID: fuzzy.VarID(0)}}
	tv, err := n.Evaluate(World(0), model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.0 {
		t.Errorf("Not(atom0) at w0 (atom0=1.0): got %v, want 0.0", tv)
	}
	tv, err = n.Evaluate(World(1), model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 1.0 {
		t.Errorf("Not(atom0) at w1 (atom0=0.0): got %v, want 1.0", tv)
	}
}

// --- And tests ---

func TestAndEvaluate(t *testing.T) {
	fx := newFixtures(t)
	_, model := setup2WorldModel(t, fx)

	a := And{Left: Atom{ID: fuzzy.VarID(0)}, Right: Atom{ID: fuzzy.VarID(1)}}
	tv, err := a.Evaluate(World(0), model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Min(1.0, 0.0) = 0.0
	if tv != 0.0 {
		t.Errorf("And(1.0,0.0) at w0: got %v, want 0.0", tv)
	}
}

func TestAndEvaluateBothTrue(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w := frame.AddWorld()
	model := NewModel(frame, 2, fx.pool, fx.arena)
	model.SetTruth(w, fuzzy.VarID(0), 1.0)
	model.SetTruth(w, fuzzy.VarID(1), 1.0)

	a := And{Left: Atom{ID: fuzzy.VarID(0)}, Right: Atom{ID: fuzzy.VarID(1)}}
	tv, err := a.Evaluate(w, model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 1.0 {
		t.Errorf("And(1.0,1.0): got %v, want 1.0", tv)
	}
}

// --- Or tests ---

func TestOrEvaluate(t *testing.T) {
	fx := newFixtures(t)
	_, model := setup2WorldModel(t, fx)

	o := Or{Left: Atom{ID: fuzzy.VarID(0)}, Right: Atom{ID: fuzzy.VarID(1)}}
	tv, err := o.Evaluate(World(0), model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Max(1.0, 0.0) = 1.0
	if tv != 1.0 {
		t.Errorf("Or(1.0,0.0) at w0: got %v, want 1.0", tv)
	}
}

func TestOrEvaluateBothFalse(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w := frame.AddWorld()
	model := NewModel(frame, 2, fx.pool, fx.arena)
	model.SetTruth(w, fuzzy.VarID(0), 0.0)
	model.SetTruth(w, fuzzy.VarID(1), 0.0)

	o := Or{Left: Atom{ID: fuzzy.VarID(0)}, Right: Atom{ID: fuzzy.VarID(1)}}
	tv, err := o.Evaluate(w, model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.0 {
		t.Errorf("Or(0.0,0.0): got %v, want 0.0", tv)
	}
}

// --- Implies tests ---

func TestImpliesEvaluate(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w := frame.AddWorld()
	model := NewModel(frame, 2, fx.pool, fx.arena)
	model.SetTruth(w, fuzzy.VarID(0), 0.0)
	model.SetTruth(w, fuzzy.VarID(1), 1.0)

	// 0.0 → 1.0 = 1.0
	imp := Implies{Antecedent: Atom{ID: fuzzy.VarID(0)}, Consequent: Atom{ID: fuzzy.VarID(1)}}
	tv, err := imp.Evaluate(w, model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 1.0 {
		t.Errorf("Implies(0.0,1.0): got %v, want 1.0", tv)
	}
}

func TestImpliesEvaluateAntecedentGreater(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w := frame.AddWorld()
	model := NewModel(frame, 2, fx.pool, fx.arena)
	model.SetTruth(w, fuzzy.VarID(0), 0.7)
	model.SetTruth(w, fuzzy.VarID(1), 0.3)

	// 0.7 → 0.3 = 0.3 (Gödel: if a > c, return c)
	imp := Implies{Antecedent: Atom{ID: fuzzy.VarID(0)}, Consequent: Atom{ID: fuzzy.VarID(1)}}
	tv, err := imp.Evaluate(w, model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.3 {
		t.Errorf("Implies(0.7,0.3): got %v, want 0.3", tv)
	}
}

// --- Iff tests ---

func TestIffEvaluate(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w := frame.AddWorld()
	model := NewModel(frame, 2, fx.pool, fx.arena)
	model.SetTruth(w, fuzzy.VarID(0), 1.0)
	model.SetTruth(w, fuzzy.VarID(1), 1.0)

	iff := Iff{Left: Atom{ID: fuzzy.VarID(0)}, Right: Atom{ID: fuzzy.VarID(1)}}
	tv, err := iff.Evaluate(w, model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 1.0 {
		t.Errorf("Iff(1.0,1.0): got %v, want 1.0", tv)
	}
}

func TestIffEvaluateDifferent(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w := frame.AddWorld()
	model := NewModel(frame, 2, fx.pool, fx.arena)
	model.SetTruth(w, fuzzy.VarID(0), 0.0)
	model.SetTruth(w, fuzzy.VarID(1), 1.0)

	iff := Iff{Left: Atom{ID: fuzzy.VarID(0)}, Right: Atom{ID: fuzzy.VarID(1)}}
	tv, err := iff.Evaluate(w, model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.0 {
		t.Errorf("Iff(0.0,1.0): got %v, want 0.0", tv)
	}
}

// --- Frame tests ---

func TestNewFrame(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	if frame.WorldCount() != 0 {
		t.Errorf("new frame world count: got %d, want 0", frame.WorldCount())
	}
}

func TestAddWorld(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	if w0 != 0 {
		t.Errorf("first world: got %d, want 0", w0)
	}
	if w1 != 1 {
		t.Errorf("second world: got %d, want 1", w1)
	}
	if frame.WorldCount() != 2 {
		t.Errorf("world count: got %d, want 2", frame.WorldCount())
	}
}

func TestAddRelation(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)

	if !frame.IsAccessible(w0, w1, RelCausal) {
		t.Error("w0→w1 should be accessible via RelCausal")
	}
}

func TestAccessible(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	w2 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)
	frame.AddRelation(w0, w2, RelCausal)
	frame.AddRelation(w0, w2, RelProcedural)

	targets := frame.Accessible(w0, RelCausal)
	if len(targets) != 2 {
		t.Fatalf("accessible count: got %d, want 2", len(targets))
	}
	if targets[0] != w1 || targets[1] != w2 {
		t.Errorf("accessible worlds: got %v, want [1,2]", targets)
	}

	targets = frame.Accessible(w0, RelProcedural)
	if len(targets) != 1 || targets[0] != w2 {
		t.Errorf("accessible via procedural: got %v, want [2]", targets)
	}

	targets = frame.Accessible(w1, RelCausal)
	if len(targets) != 0 {
		t.Errorf("w1 has no outgoing edges: got %v, want []", targets)
	}
}

func TestIsAccessible(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)

	if !frame.IsAccessible(w0, w1, RelCausal) {
		t.Error("expected accessible")
	}
	if frame.IsAccessible(w1, w0, RelCausal) {
		t.Error("expected not accessible (reverse)")
	}
	if frame.IsAccessible(w0, w1, RelProcedural) {
		t.Error("expected not accessible (wrong relation)")
	}
}

func TestReflexiveClosure(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)

	frame.ReflexiveClosure(RelCausal)

	if !frame.IsAccessible(w0, w0, RelCausal) {
		t.Error("reflexive closure: w0→w0 should exist")
	}
	if !frame.IsAccessible(w1, w1, RelCausal) {
		t.Error("reflexive closure: w1→w1 should exist")
	}
	if !frame.IsAccessible(w0, w1, RelCausal) {
		t.Error("reflexive closure: original edge should remain")
	}
}

func TestSymmetricClosure(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)

	frame.SymmetricClosure(RelCausal)

	if !frame.IsAccessible(w1, w0, RelCausal) {
		t.Error("symmetric closure: w1→w0 should exist")
	}
	if !frame.IsAccessible(w0, w1, RelCausal) {
		t.Error("symmetric closure: original edge should remain")
	}
}

func TestTransitiveClosure(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	w2 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)
	frame.AddRelation(w1, w2, RelCausal)

	frame.TransitiveClosure(RelCausal)

	if !frame.IsAccessible(w0, w2, RelCausal) {
		t.Error("transitive closure: w0→w2 should exist (via w1)")
	}
	if !frame.IsAccessible(w0, w1, RelCausal) {
		t.Error("transitive closure: original edge should remain")
	}
}

func TestTransitiveClosureEmpty(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	frame.TransitiveClosure(RelCausal) // should not panic
}

// --- Model tests ---

func TestNewModel(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	frame.AddWorld()
	frame.AddWorld()

	model := NewModel(frame, 3, fx.pool, fx.arena)
	if model.NumVars() != 3 {
		t.Errorf("NumVars: got %d, want 3", model.NumVars())
	}
	if model.Frame() != frame {
		t.Error("Frame() should return the frame")
	}
}

func TestSetTruthAndTruth(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w := frame.AddWorld()
	model := NewModel(frame, 2, fx.pool, fx.arena)

	model.SetTruth(w, fuzzy.VarID(0), 0.75)
	model.SetTruth(w, fuzzy.VarID(1), 0.25)

	if tv := model.Truth(w, fuzzy.VarID(0)); tv != 0.75 {
		t.Errorf("Truth(var0): got %v, want 0.75", tv)
	}
	if tv := model.Truth(w, fuzzy.VarID(1)); tv != 0.25 {
		t.Errorf("Truth(var1): got %v, want 0.25", tv)
	}
}

func TestTruthOutOfBounds(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w := frame.AddWorld()
	model := NewModel(frame, 1, fx.pool, fx.arena)

	if tv := model.Truth(World(99), fuzzy.VarID(0)); tv != 0.0 {
		t.Errorf("out-of-bounds world: got %v, want 0.0", tv)
	}
	if tv := model.Truth(w, fuzzy.VarID(99)); tv != 0.0 {
		t.Errorf("out-of-bounds var: got %v, want 0.0", tv)
	}
}

// --- Nested formula tests ---

func TestNestedModal(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	w2 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)
	frame.AddRelation(w1, w2, RelCausal)

	model := NewModel(frame, 1, fx.pool, fx.arena)
	model.SetTruth(w0, fuzzy.VarID(0), 0.5)
	model.SetTruth(w1, fuzzy.VarID(0), 0.0)
	model.SetTruth(w2, fuzzy.VarID(0), 1.0)

	// □□atom(0) at w0:
	//   Box(Box(atom0)) at w0 = min over {w1} of Box(atom0) at w1
	//   Box(atom0) at w1 = min over {w2} of atom0 at w2 = 1.0
	//   Result = 1.0
	inner := Box{Formula: Atom{ID: fuzzy.VarID(0)}, Rel: RelCausal}
	outer := Box{Formula: inner, Rel: RelCausal}
	tv, err := outer.Evaluate(w0, model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 1.0 {
		t.Errorf("□□atom(0) at w0: got %v, want 1.0", tv)
	}
}

func TestDiamondBox(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	w2 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)
	frame.AddRelation(w0, w2, RelCausal)

	model := NewModel(frame, 1, fx.pool, fx.arena)
	model.SetTruth(w0, fuzzy.VarID(0), 1.0)
	model.SetTruth(w1, fuzzy.VarID(0), 0.0)
	model.SetTruth(w2, fuzzy.VarID(0), 0.5)

	// ◇¬atom(0) at w0:
	//   Diamond(Not(atom0)) at w0 = max over {w1,w2} of Not(atom0)
	//   Not(atom0) at w1 = 1.0, Not(atom0) at w2 = 0.5
	//   Result = 1.0
	inner := Not{Formula: Atom{ID: fuzzy.VarID(0)}}
	outer := Diamond{Formula: inner, Rel: RelCausal}
	tv, err := outer.Evaluate(w0, model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 1.0 {
		t.Errorf("◇¬atom(0) at w0: got %v, want 1.0", tv)
	}
}

func TestComplexFormula(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)

	model := NewModel(frame, 2, fx.pool, fx.arena)
	model.SetTruth(w0, fuzzy.VarID(0), 0.8)
	model.SetTruth(w0, fuzzy.VarID(1), 0.3)
	model.SetTruth(w1, fuzzy.VarID(0), 0.2)
	model.SetTruth(w1, fuzzy.VarID(1), 0.9)

	// □(atom(0) → ◇atom(1)) at w0:
	//   At w1: ◇atom(1) = max over {} = 0.0 (no outgoing edges from w1)
	//   At w1: atom(0) → ◇atom(1) = 0.2 → 0.0 = 0.0 (Gödel: a > c, return c)
	//   □(impl) = min over {w1} = 0.0
	dia := Diamond{Formula: Atom{ID: fuzzy.VarID(1)}, Rel: RelCausal}
	imp := Implies{Antecedent: Atom{ID: fuzzy.VarID(0)}, Consequent: dia}
	box := Box{Formula: imp, Rel: RelCausal}
	tv, err := box.Evaluate(w0, model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.0 {
		t.Errorf("□(atom0 → ◇atom1) at w0: got %v, want 0.0", tv)
	}
}

// --- Edge cases ---

func TestEmptyFrame(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	model := NewModel(frame, 1, fx.pool, fx.arena)

	// No worlds — Atom evaluation should be out of bounds
	a := Atom{ID: fuzzy.VarID(0)}
	tv, _ := a.Evaluate(World(0), model)
	if tv != 0.0 {
		t.Errorf("atom in empty model: got %v, want 0.0", tv)
	}
}

func TestMultipleRelationTypes(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)
	frame.AddRelation(w0, w1, RelProcedural)
	frame.AddRelation(w0, w1, RelAssociation)

	model := NewModel(frame, 1, fx.pool, fx.arena)
	model.SetTruth(w1, fuzzy.VarID(0), 1.0)

	// □ with different relations should all see w1
	for _, rel := range []RelationType{RelCausal, RelProcedural, RelAssociation} {
		b := Box{Formula: Atom{ID: fuzzy.VarID(0)}, Rel: rel}
		tv, err := b.Evaluate(w0, model)
		if err != nil {
			t.Fatalf("unexpected error for rel %d: %v", rel, err)
		}
		if tv != 1.0 {
			t.Errorf("Box with rel %d: got %v, want 1.0", rel, tv)
		}
	}
}

func TestCyclicAccessibility(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)
	frame.AddRelation(w1, w0, RelCausal)

	model := NewModel(frame, 1, fx.pool, fx.arena)
	model.SetTruth(w0, fuzzy.VarID(0), 0.3)
	model.SetTruth(w1, fuzzy.VarID(0), 0.7)

	// □□atom(0) at w0: min over w1 of Box at w1 = min(0.7, 0.3) = 0.3
	inner := Box{Formula: Atom{ID: fuzzy.VarID(0)}, Rel: RelCausal}
	outer := Box{Formula: inner, Rel: RelCausal}
	tv, err := outer.Evaluate(w0, model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.3 {
		t.Errorf("□□atom(0) with cycle: got %v, want 0.3", tv)
	}
}

func TestFuzzyTruthValues(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	w2 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)
	frame.AddRelation(w0, w2, RelCausal)

	model := NewModel(frame, 1, fx.pool, fx.arena)
	model.SetTruth(w0, fuzzy.VarID(0), 0.5)
	model.SetTruth(w1, fuzzy.VarID(0), 0.2)
	model.SetTruth(w2, fuzzy.VarID(0), 0.8)

	// □atom(0) = min(0.2, 0.8) = 0.2
	b := Box{Formula: Atom{ID: fuzzy.VarID(0)}, Rel: RelCausal}
	tv, _ := b.Evaluate(w0, model)
	if tv != 0.2 {
		t.Errorf("□ with fuzzy: got %v, want 0.2", tv)
	}

	// ◇atom(0) = max(0.2, 0.8) = 0.8
	d := Diamond{Formula: Atom{ID: fuzzy.VarID(0)}, Rel: RelCausal}
	tv, _ = d.Evaluate(w0, model)
	if tv != 0.8 {
		t.Errorf("◇ with fuzzy: got %v, want 0.8", tv)
	}
}

// --- CC verification — all helpers must be exercised through tests ---

func TestEvalQuantifiedDirect(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	w2 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)
	frame.AddRelation(w0, w2, RelCausal)

	model := NewModel(frame, 1, fx.pool, fx.arena)
	model.SetTruth(w1, fuzzy.VarID(0), 0.3)
	model.SetTruth(w2, fuzzy.VarID(0), 0.9)

	// evalQuantified with isBox=true → min
	tv, err := evalQuantified(Atom{ID: fuzzy.VarID(0)}, w0, model, RelCausal, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.3 {
		t.Errorf("evalQuantified(box): got %v, want 0.3", tv)
	}

	// evalQuantified with isBox=false → max
	tv, err = evalQuantified(Atom{ID: fuzzy.VarID(0)}, w0, model, RelCausal, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.9 {
		t.Errorf("evalQuantified(diamond): got %v, want 0.9", tv)
	}
}

// errorFormula is a test helper that always returns an error.
// Used to exercise error propagation paths for 100% coverage.
type errorFormula struct{}

func (e errorFormula) Evaluate(w World, m *Model) (TruthValue, error) {
	return 0, errTest
}

// errTest is a sentinel error for testing.
var errTest = &testError{}

type testError struct{}

func (e *testError) Error() string { return "test error" }

func TestBoxEvaluateError(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)
	model := NewModel(frame, 1, fx.pool, fx.arena)

	b := Box{Formula: errorFormula{}, Rel: RelCausal}
	_, err := b.Evaluate(w0, model)
	if err == nil {
		t.Error("expected error from Box containing errorFormula")
	}
}

func TestDiamondEvaluateError(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)
	model := NewModel(frame, 1, fx.pool, fx.arena)

	d := Diamond{Formula: errorFormula{}, Rel: RelCausal}
	_, err := d.Evaluate(w0, model)
	if err == nil {
		t.Error("expected error from Diamond containing errorFormula")
	}
}

func TestNotEvaluateError(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w := frame.AddWorld()
	model := NewModel(frame, 1, fx.pool, fx.arena)

	n := Not{Formula: errorFormula{}}
	_, err := n.Evaluate(w, model)
	if err == nil {
		t.Error("expected error from Not containing errorFormula")
	}
}

func TestAndEvaluateError(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w := frame.AddWorld()
	model := NewModel(frame, 1, fx.pool, fx.arena)

	a := And{Left: errorFormula{}, Right: Atom{ID: fuzzy.VarID(0)}}
	_, err := a.Evaluate(w, model)
	if err == nil {
		t.Error("expected error from And with error left")
	}

	a = And{Left: Atom{ID: fuzzy.VarID(0)}, Right: errorFormula{}}
	_, err = a.Evaluate(w, model)
	if err == nil {
		t.Error("expected error from And with error right")
	}
}

func TestOrEvaluateError(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w := frame.AddWorld()
	model := NewModel(frame, 1, fx.pool, fx.arena)

	o := Or{Left: errorFormula{}, Right: Atom{ID: fuzzy.VarID(0)}}
	_, err := o.Evaluate(w, model)
	if err == nil {
		t.Error("expected error from Or with error left")
	}
}

func TestImpliesEvaluateError(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w := frame.AddWorld()
	model := NewModel(frame, 1, fx.pool, fx.arena)

	imp := Implies{Antecedent: errorFormula{}, Consequent: Atom{ID: fuzzy.VarID(0)}}
	_, err := imp.Evaluate(w, model)
	if err == nil {
		t.Error("expected error from Implies with error antecedent")
	}
}

func TestOrEvaluateErrorRight(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w := frame.AddWorld()
	model := NewModel(frame, 1, fx.pool, fx.arena)

	o := Or{Left: Atom{ID: fuzzy.VarID(0)}, Right: errorFormula{}}
	_, err := o.Evaluate(w, model)
	if err == nil {
		t.Error("expected error from Or with error right")
	}
}

func TestImpliesEvaluateErrorRight(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w := frame.AddWorld()
	model := NewModel(frame, 1, fx.pool, fx.arena)

	imp := Implies{Antecedent: Atom{ID: fuzzy.VarID(0)}, Consequent: errorFormula{}}
	_, err := imp.Evaluate(w, model)
	if err == nil {
		t.Error("expected error from Implies with error consequent")
	}
}

func TestAndEvaluateEqual(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w := frame.AddWorld()
	model := NewModel(frame, 2, fx.pool, fx.arena)
	model.SetTruth(w, fuzzy.VarID(0), 0.5)
	model.SetTruth(w, fuzzy.VarID(1), 0.5)

	a := And{Left: Atom{ID: fuzzy.VarID(0)}, Right: Atom{ID: fuzzy.VarID(1)}}
	tv, err := a.Evaluate(w, model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.5 {
		t.Errorf("And(0.5,0.5): got %v, want 0.5", tv)
	}
}

func TestAndEvaluateLeftSmaller(t *testing.T) {
	fx := newFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w := frame.AddWorld()
	model := NewModel(frame, 2, fx.pool, fx.arena)
	model.SetTruth(w, fuzzy.VarID(0), 0.3)
	model.SetTruth(w, fuzzy.VarID(1), 0.7)

	a := And{Left: Atom{ID: fuzzy.VarID(0)}, Right: Atom{ID: fuzzy.VarID(1)}}
	tv, err := a.Evaluate(w, model)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.3 {
		t.Errorf("And(0.3,0.7): got %v, want 0.3", tv)
	}
}
