package modal

import (
	"testing"

	"github.com/xDarkicex/logic/fuzzy"
	"github.com/xDarkicex/memory"
)

type epistemicFixtures struct {
	pool  *memory.Pool
	arena *memory.Arena
}

func newEpistemicFixtures(t *testing.T) *epistemicFixtures {
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
	return &epistemicFixtures{pool: pool, arena: arena}
}

// setup2Agent creates a 3-world model with 2 agents.
// Agent 0 (Alice): knows world 1 from world 0.
// Agent 1 (Bob): knows world 2 from world 0.
func setup2Agent(t *testing.T, fx *epistemicFixtures) *EpistemicModel {
	t.Helper()
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	w2 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelAssociation) // Alice: w0→w1
	frame.AddRelation(w0, w2, RelCausal)      // Bob: w0→w2

	em := NewEpistemicModel(frame, []AgentID{0, 1}, 1, fx.pool, fx.arena)
	em.SetKnows(0, RelAssociation) // Alice
	em.SetKnows(1, RelCausal)      // Bob
	return em
}

func TestNewEpistemicModel(t *testing.T) {
	fx := newEpistemicFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	frame.AddWorld()

	em := NewEpistemicModel(frame, []AgentID{0, 1, 2}, 2, fx.pool, fx.arena)
	if len(em.Agents()) != 3 {
		t.Errorf("expected 3 agents, got %d", len(em.Agents()))
	}
}

func TestSetKnowsAndKnows(t *testing.T) {
	fx := newEpistemicFixtures(t)
	em := setup2Agent(t, fx)
	em.SetTruth(0, fuzzy.VarID(0), 1.0)
	em.SetTruth(1, fuzzy.VarID(0), 0.0) // Alice's only accessible world — P is false here
	em.SetTruth(2, fuzzy.VarID(0), 1.0) // Bob's only accessible world — P is true here

	// Alice knows P? At w0, Alice accesses w1 where P=0.0 → Alice doesn't know P
	tv, err := em.Knows(0, Atom{ID: fuzzy.VarID(0)}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.0 {
		t.Errorf("Alice Knows P when P=0 at w1: got %v, want 0.0", tv)
	}

	// Bob knows P? At w0, Bob accesses w2 where P=1.0 → Bob knows P
	tv, err = em.Knows(1, Atom{ID: fuzzy.VarID(0)}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 1.0 {
		t.Errorf("Bob Knows P when P=1 at w2: got %v, want 1.0", tv)
	}
}

func TestKnowsDefaultRelation(t *testing.T) {
	fx := newEpistemicFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal) // default relation

	em := NewEpistemicModel(frame, []AgentID{0}, 1, fx.pool, fx.arena)
	// Don't set any knowledge relation — should default to RelCausal
	em.SetTruth(1, fuzzy.VarID(0), 1.0)

	tv, err := em.Knows(0, Atom{ID: fuzzy.VarID(0)}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 1.0 {
		t.Errorf("knows with default relation: got %v, want 1.0", tv)
	}
}

func TestBelieves(t *testing.T) {
	fx := newEpistemicFixtures(t)
	em := setup2Agent(t, fx)
	em.SetBelief(0, RelAssociation) // Same as knowledge for Alice
	em.SetBelief(1, RelCausal)      // Different: Bob believes via Procedural
	em.SetBelief(1, RelProcedural)   // Actually Bob believes via Procedural

	em.SetTruth(0, fuzzy.VarID(0), 1.0)
	em.SetTruth(1, fuzzy.VarID(0), 0.0)
	em.SetTruth(2, fuzzy.VarID(0), 1.0)

	// Alice believes P via RelAssociation (accesses w1, P=0.0)
	tv, err := em.Believes(0, Atom{ID: fuzzy.VarID(0)}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.0 {
		t.Errorf("Alice Believes P: got %v, want 0.0", tv)
	}
}

func TestSetBeliefOverwrite(t *testing.T) {
	fx := newEpistemicFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)
	frame.AddRelation(w0, w1, RelProcedural)

	em := NewEpistemicModel(frame, []AgentID{0}, 1, fx.pool, fx.arena)
	em.SetBelief(0, RelCausal)
	em.SetBelief(0, RelProcedural) // overwrite

	rel := em.beliefRel(0)
	if rel != RelProcedural {
		t.Errorf("belief rel after overwrite: got %d, want %d", rel, RelProcedural)
	}
}

func TestCommonKnowledge(t *testing.T) {
	fx := newEpistemicFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	// w0→w1 for both agents
	frame.AddRelation(w0, w1, RelCausal)
	frame.AddRelation(w0, w1, RelAssociation)

	em := NewEpistemicModel(frame, []AgentID{0, 1}, 1, fx.pool, fx.arena)
	em.SetKnows(0, RelCausal)
	em.SetKnows(1, RelAssociation)
	em.SetTruth(w0, fuzzy.VarID(0), 1.0)
	em.SetTruth(w1, fuzzy.VarID(0), 1.0) // P everywhere

	tv, err := em.CommonKnowledge([]AgentID{0, 1}, Atom{ID: fuzzy.VarID(0)}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 1.0 {
		t.Errorf("CommonKnowledge when P holds everywhere: got %v, want 1.0", tv)
	}
}

func TestCommonKnowledgeFails(t *testing.T) {
	fx := newEpistemicFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)
	frame.AddRelation(w0, w1, RelAssociation)

	em := NewEpistemicModel(frame, []AgentID{0, 1}, 1, fx.pool, fx.arena)
	em.SetKnows(0, RelCausal)
	em.SetKnows(1, RelAssociation)
	em.SetTruth(w0, fuzzy.VarID(0), 1.0)
	em.SetTruth(w1, fuzzy.VarID(0), 0.0) // P fails at w1

	tv, err := em.CommonKnowledge([]AgentID{0, 1}, Atom{ID: fuzzy.VarID(0)}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.0 {
		t.Errorf("CommonKnowledge when P fails at w1: got %v, want 0.0", tv)
	}
}

func TestCommonKnowledgeEmptyFrame(t *testing.T) {
	fx := newEpistemicFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	em := NewEpistemicModel(frame, []AgentID{0}, 1, fx.pool, fx.arena)
	tv, err := em.CommonKnowledge([]AgentID{0}, Atom{ID: fuzzy.VarID(0)}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.0 {
		t.Errorf("empty frame: got %v, want 0.0", tv)
	}
}

func TestDistributedKnowledge(t *testing.T) {
	fx := newEpistemicFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	w2 := frame.AddWorld()
	// Alice accesses w1, Bob accesses w2
	frame.AddRelation(w0, w1, RelCausal)
	frame.AddRelation(w0, w2, RelAssociation)

	em := NewEpistemicModel(frame, []AgentID{0, 1}, 1, fx.pool, fx.arena)
	em.SetKnows(0, RelCausal)
	em.SetKnows(1, RelAssociation)
	// Only w0 is in the intersection (w0→w0 is NOT set, so intersection is empty)
	// Actually, without reflexivity, the intersection of worlds accessible by BOTH
	// Alice AND Bob from w0 is empty → vacuously true → returns 1.0
	// Wait, no. Alice accesses w1 only. Bob accesses w2 only.
	// Intersection = ∅ → no worlds need to satisfy P → result = 1.0 (vacuously true)

	em.SetTruth(w1, fuzzy.VarID(0), 0.0)
	em.SetTruth(w2, fuzzy.VarID(0), 0.0)

	tv, err := em.DistributedKnowledge([]AgentID{0, 1}, Atom{ID: fuzzy.VarID(0)}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 1.0 {
		t.Errorf("DistributedKnowledge with empty intersection: got %v, want 1.0", tv)
	}
}

func TestDistributedKnowledgeWithOverlap(t *testing.T) {
	fx := newEpistemicFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	// Both Alice and Bob access w1 from w0
	frame.AddRelation(w0, w1, RelCausal)
	frame.AddRelation(w0, w1, RelAssociation)

	em := NewEpistemicModel(frame, []AgentID{0, 1}, 1, fx.pool, fx.arena)
	em.SetKnows(0, RelCausal)
	em.SetKnows(1, RelAssociation)
	em.SetTruth(w1, fuzzy.VarID(0), 0.5)

	tv, err := em.DistributedKnowledge([]AgentID{0, 1}, Atom{ID: fuzzy.VarID(0)}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.5 {
		t.Errorf("DistributedKnowledge with overlapping access: got %v, want 0.5", tv)
	}
}

func TestDistributedKnowledgeEmptyGroup(t *testing.T) {
	fx := newEpistemicFixtures(t)
	em := setup2Agent(t, fx)
	tv, err := em.DistributedKnowledge(nil, Atom{ID: fuzzy.VarID(0)}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.0 {
		t.Errorf("empty group: got %v, want 0.0", tv)
	}
}

func TestIsKnowledgeConsistent(t *testing.T) {
	fx := newEpistemicFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)

	em := NewEpistemicModel(frame, []AgentID{0}, 1, fx.pool, fx.arena)
	em.SetKnows(0, RelCausal)
	em.SetTruth(w1, fuzzy.VarID(0), 0.5) // P is 0.5 at the only accessible world
	// At w1, □P = 0.5, □¬P = 0.5. Both > 0 → inconsistent
	// Wait, □P = min over{access from w0} of P. From w0, Accessible returns [w1].
	// □P at w0 = P(w1) = 0.5. □¬P at w0 = 1 - P(w1) = 0.5.
	// Since 0.5 > 0 and 0.5 > 0, this IS inconsistent (fuzzy contradiction)
	if em.IsKnowledgeConsistent(0) {
		t.Error("inconsistent knowledge should return false")
	}
}

func TestIsKnowledgeConsistentTrue(t *testing.T) {
	fx := newEpistemicFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)

	em := NewEpistemicModel(frame, []AgentID{0}, 1, fx.pool, fx.arena)
	em.SetKnows(0, RelCausal)
	em.SetTruth(w1, fuzzy.VarID(0), 1.0) // P=1 at w1
	// □P = 1.0, □¬P = 0.0 → not both > 0 → consistent

	if !em.IsKnowledgeConsistent(0) {
		t.Error("consistent knowledge should return true")
	}
}
