package modal

import (
	"testing"

	"github.com/xDarkicex/memory"
)

type axiomFixtures struct {
	pool  *memory.Pool
	arena *memory.Arena
}

func newAxiomFixtures(t *testing.T) *axiomFixtures {
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
	return &axiomFixtures{pool: pool, arena: arena}
}

func TestSystemString(t *testing.T) {
	tests := []struct {
		sys  System
		want string
	}{
		{SystemK, "K"},
		{SystemD, "D"},
		{SystemT, "T"},
		{SystemB, "B"},
		{SystemS4, "S4"},
		{SystemS5, "S5"},
		{System(99), "unknown"},
	}
	for _, tt := range tests {
		if got := tt.sys.String(); got != tt.want {
			t.Errorf("System(%d).String() = %q, want %q", tt.sys, got, tt.want)
		}
	}
}

func TestEnforceSystemK(t *testing.T) {
	fx := newAxiomFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	frame.AddWorld()

	EnforceSystemK(frame) // should be no-op, no panic
}

func TestEnforceSystemD(t *testing.T) {
	fx := newAxiomFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal) // w0 has successor, w1 does not

	EnforceSystemD(frame, RelCausal)

	if !frame.IsAccessible(w1, w1, RelCausal) {
		t.Error("w1 should have a self-loop after D enforcement")
	}
}

func TestEnforceSystemT(t *testing.T) {
	fx := newAxiomFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)

	EnforceSystemT(frame, RelCausal)

	if !frame.IsAccessible(w0, w0, RelCausal) {
		t.Error("w0 should access itself after T enforcement")
	}
	if !frame.IsAccessible(w1, w1, RelCausal) {
		t.Error("w1 should access itself after T enforcement")
	}
}

func TestEnforceSystemB(t *testing.T) {
	fx := newAxiomFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)

	EnforceSystemB(frame, RelCausal)

	if !frame.IsAccessible(w0, w0, RelCausal) {
		t.Error("reflexivity: w0→w0 should exist")
	}
	if !frame.IsAccessible(w1, w0, RelCausal) {
		t.Error("symmetry: w1→w0 should exist")
	}
}

func TestEnforceSystemS4(t *testing.T) {
	fx := newAxiomFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	w2 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)
	frame.AddRelation(w1, w2, RelCausal)

	EnforceSystemS4(frame, RelCausal)

	if !frame.IsAccessible(w0, w2, RelCausal) {
		t.Error("transitivity: w0→w2 should exist")
	}
	if !frame.IsAccessible(w0, w0, RelCausal) {
		t.Error("reflexivity: w0→w0 should exist")
	}
}

func TestEnforceSystemS5(t *testing.T) {
	fx := newAxiomFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	w2 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)
	frame.AddRelation(w1, w2, RelCausal)

	EnforceSystemS5(frame, RelCausal)

	// Equivalence: all pairs should be connected
	for _, w := range []World{w0, w1, w2} {
		for _, v := range []World{w0, w1, w2} {
			if !frame.IsAccessible(w, v, RelCausal) {
				t.Errorf("S5: %d→%d should exist", w, v)
			}
		}
	}
}

func TestValidateFrameK(t *testing.T) {
	fx := newAxiomFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	frame.AddWorld()

	if err := ValidateFrameAgainst(frame, SystemK, RelCausal); err != nil {
		t.Errorf("K should always be valid: %v", err)
	}
}

func TestValidateFrameD(t *testing.T) {
	fx := newAxiomFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal) // w0 has successor, w1 does not

	err := ValidateFrameAgainst(frame, SystemD, RelCausal)
	if err == nil {
		t.Error("frame should fail D validation (w1 has no successor)")
	}

	EnforceSystemD(frame, RelCausal)
	if err := ValidateFrameAgainst(frame, SystemD, RelCausal); err != nil {
		t.Errorf("frame should pass D after enforcement: %v", err)
	}
}

func TestValidateFrameT(t *testing.T) {
	fx := newAxiomFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)

	err := ValidateFrameAgainst(frame, SystemT, RelCausal)
	if err == nil {
		t.Error("frame should fail T validation (no self-loops)")
	}

	EnforceSystemT(frame, RelCausal)
	if err := ValidateFrameAgainst(frame, SystemT, RelCausal); err != nil {
		t.Errorf("frame should pass T after enforcement: %v", err)
	}
}

func TestValidateFrameB(t *testing.T) {
	fx := newAxiomFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)

	err := ValidateFrameAgainst(frame, SystemB, RelCausal)
	if err == nil {
		t.Error("frame should fail B (no reflexivity or symmetry)")
	}

	EnforceSystemB(frame, RelCausal)
	if err := ValidateFrameAgainst(frame, SystemB, RelCausal); err != nil {
		t.Errorf("frame should pass B after enforcement: %v", err)
	}
}

func TestValidateFrameS4(t *testing.T) {
	fx := newAxiomFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	w2 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)
	frame.AddRelation(w1, w2, RelCausal)

	err := ValidateFrameAgainst(frame, SystemS4, RelCausal)
	if err == nil {
		t.Error("frame should fail S4 (no reflexivity, no transitivity)")
	}

	EnforceSystemS4(frame, RelCausal)
	if err := ValidateFrameAgainst(frame, SystemS4, RelCausal); err != nil {
		t.Errorf("frame should pass S4 after enforcement: %v", err)
	}
}

func TestValidateFrameS5(t *testing.T) {
	fx := newAxiomFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)

	err := ValidateFrameAgainst(frame, SystemS5, RelCausal)
	if err == nil {
		t.Error("frame should fail S5 (no equivalence)")
	}

	EnforceSystemS5(frame, RelCausal)
	if err := ValidateFrameAgainst(frame, SystemS5, RelCausal); err != nil {
		t.Errorf("frame should pass S5 after enforcement: %v", err)
	}
}

func TestValidateSymmetric(t *testing.T) {
	fx := newAxiomFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)

	err := validateSymmetric(frame, RelCausal)
	if err == nil {
		t.Error("asymmetric frame should fail")
	}

	frame.AddRelation(w1, w0, RelCausal)
	err = validateSymmetric(frame, RelCausal)
	if err != nil {
		t.Errorf("symmetric frame should pass: %v", err)
	}
}

func TestValidateTransitive(t *testing.T) {
	fx := newAxiomFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	w2 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)
	frame.AddRelation(w1, w2, RelCausal)

	err := validateTransitive(frame, RelCausal)
	if err == nil {
		t.Error("non-transitive frame should fail")
	}

	frame.AddRelation(w0, w2, RelCausal)
	err = validateTransitive(frame, RelCausal)
	if err != nil {
		t.Errorf("transitive frame should pass: %v", err)
	}
}

func TestValidateReflexive(t *testing.T) {
	fx := newAxiomFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()

	err := validateReflexive(frame, RelCausal)
	if err == nil {
		t.Error("non-reflexive frame should fail")
	}

	frame.AddRelation(w0, w0, RelCausal)
	err = validateReflexive(frame, RelCausal)
	if err != nil {
		t.Errorf("reflexive frame should pass: %v", err)
	}
}

func TestEnforceSystemDMultipleRelations(t *testing.T) {
	fx := newAxiomFixtures(t)
	frame := NewFrame(fx.pool, fx.arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)

	EnforceSystemD(frame, RelProcedural) // should add self-loops for procedural only

	// Causal: w1 still has no successor
	if frame.IsAccessible(w1, w1, RelCausal) {
		t.Error("w1 should NOT have causal self-loop — only procedural was enforced")
	}
	// Procedural: both worlds should have self-loops
	if !frame.IsAccessible(w0, w0, RelProcedural) {
		t.Error("w0 should have procedural self-loop")
	}
	if !frame.IsAccessible(w1, w1, RelProcedural) {
		t.Error("w1 should have procedural self-loop")
	}
}
