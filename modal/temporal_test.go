package modal

import (
	"testing"

	"github.com/xDarkicex/logic/fuzzy"
	"github.com/xDarkicex/memory"
)

type temporalFixtures struct {
	pool  *memory.Pool
	arena *memory.Arena
}

func newTemporalFixtures(t *testing.T) *temporalFixtures {
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
	return &temporalFixtures{pool: pool, arena: arena}
}

func setupTimeline(t *testing.T, fx *temporalFixtures, length int) *TemporalModel {
	t.Helper()
	frame := NewFrame(fx.pool, fx.arena)
	for i := 0; i < length; i++ {
		frame.AddWorld()
	}
	return NewTemporalModel(frame, length, 2, fx.pool, fx.arena)
}

func TestNewTemporalModel(t *testing.T) {
	fx := newTemporalFixtures(t)
	tm := setupTimeline(t, fx, 5)

	if tm.Len() != 5 {
		t.Errorf("Len: got %d, want 5", tm.Len())
	}
	tl := tm.Timeline()
	if len(tl) != 5 {
		t.Errorf("Timeline len: got %d, want 5", len(tl))
	}
	for i := 0; i < 5; i++ {
		if tl[i] != World(i) {
			t.Errorf("Timeline[%d] = %d, want %d", i, tl[i], i)
		}
	}
}

func TestEvalAlways(t *testing.T) {
	fx := newTemporalFixtures(t)
	tm := setupTimeline(t, fx, 4)
	tm.SetTruth(0, fuzzy.VarID(0), 1.0)
	tm.SetTruth(1, fuzzy.VarID(0), 1.0)
	tm.SetTruth(2, fuzzy.VarID(0), 1.0)
	tm.SetTruth(3, fuzzy.VarID(0), 1.0)

	tv, err := tm.EvalAlways(Atom{ID: fuzzy.VarID(0)}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 1.0 {
		t.Errorf("□P when P always true: got %v, want 1.0", tv)
	}
}

func TestEvalAlwaysFailsAtEnd(t *testing.T) {
	fx := newTemporalFixtures(t)
	tm := setupTimeline(t, fx, 4)
	tm.SetTruth(0, fuzzy.VarID(0), 1.0)
	tm.SetTruth(1, fuzzy.VarID(0), 1.0)
	tm.SetTruth(2, fuzzy.VarID(0), 1.0)
	tm.SetTruth(3, fuzzy.VarID(0), 0.0) // fails at last world

	tv, err := tm.EvalAlways(Atom{ID: fuzzy.VarID(0)}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.0 {
		t.Errorf("□P when P fails at end: got %v, want 0.0", tv)
	}
}

func TestEvalAlwaysFromMiddle(t *testing.T) {
	fx := newTemporalFixtures(t)
	tm := setupTimeline(t, fx, 4)
	tm.SetTruth(0, fuzzy.VarID(0), 0.0) // false early, but we start at 2
	tm.SetTruth(1, fuzzy.VarID(0), 0.0)
	tm.SetTruth(2, fuzzy.VarID(0), 1.0)
	tm.SetTruth(3, fuzzy.VarID(0), 1.0)

	tv, err := tm.EvalAlways(Atom{ID: fuzzy.VarID(0)}, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 1.0 {
		t.Errorf("□P from middle: got %v, want 1.0", tv)
	}
}

func TestEvalAlwaysOutOfBounds(t *testing.T) {
	fx := newTemporalFixtures(t)
	tm := setupTimeline(t, fx, 2)

	tv, err := tm.EvalAlways(Atom{ID: fuzzy.VarID(0)}, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 1.0 {
		t.Errorf("out of bounds: got %v, want 1.0 (vacuously true)", tv)
	}
}

func TestEvalEventually(t *testing.T) {
	fx := newTemporalFixtures(t)
	tm := setupTimeline(t, fx, 4)
	tm.SetTruth(0, fuzzy.VarID(0), 0.0)
	tm.SetTruth(1, fuzzy.VarID(0), 0.0)
	tm.SetTruth(2, fuzzy.VarID(0), 1.0) // becomes true here
	tm.SetTruth(3, fuzzy.VarID(0), 0.0)

	tv, err := tm.EvalEventually(Atom{ID: fuzzy.VarID(0)}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 1.0 {
		t.Errorf("◇P when P holds at world 2: got %v, want 1.0", tv)
	}
}

func TestEvalEventuallyNeverTrue(t *testing.T) {
	fx := newTemporalFixtures(t)
	tm := setupTimeline(t, fx, 3)
	tm.SetTruth(0, fuzzy.VarID(0), 0.0)
	tm.SetTruth(1, fuzzy.VarID(0), 0.0)
	tm.SetTruth(2, fuzzy.VarID(0), 0.0)

	tv, err := tm.EvalEventually(Atom{ID: fuzzy.VarID(0)}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.0 {
		t.Errorf("◇P when never true: got %v, want 0.0", tv)
	}
}

func TestEvalEventuallyOutOfBounds(t *testing.T) {
	fx := newTemporalFixtures(t)
	tm := setupTimeline(t, fx, 2)

	tv, err := tm.EvalEventually(Atom{ID: fuzzy.VarID(0)}, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.0 {
		t.Errorf("out of bounds: got %v, want 0.0 (vacuously false)", tv)
	}
}

func TestEvalNext(t *testing.T) {
	fx := newTemporalFixtures(t)
	tm := setupTimeline(t, fx, 3)
	tm.SetTruth(0, fuzzy.VarID(0), 0.0)
	tm.SetTruth(1, fuzzy.VarID(0), 1.0) // next world
	tm.SetTruth(2, fuzzy.VarID(0), 0.0)

	tv, err := tm.EvalNext(Atom{ID: fuzzy.VarID(0)}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 1.0 {
		t.Errorf("○P at w0 (next=w1 where P=1.0): got %v, want 1.0", tv)
	}
}

func TestEvalNextLastWorld(t *testing.T) {
	fx := newTemporalFixtures(t)
	tm := setupTimeline(t, fx, 2)
	tm.SetTruth(1, fuzzy.VarID(0), 1.0)

	tv, err := tm.EvalNext(Atom{ID: fuzzy.VarID(0)}, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.0 {
		t.Errorf("○P at last world: got %v, want 0.0", tv)
	}
}

func TestEvalUntil(t *testing.T) {
	fx := newTemporalFixtures(t)
	tm := setupTimeline(t, fx, 5)
	// P holds at 0,1,2; Q holds at 3
	tm.SetTruth(0, fuzzy.VarID(0), 1.0) // P
	tm.SetTruth(1, fuzzy.VarID(0), 1.0) // P
	tm.SetTruth(2, fuzzy.VarID(0), 1.0) // P
	tm.SetTruth(3, fuzzy.VarID(0), 0.0) // P false here, but Q
	tm.SetTruth(3, fuzzy.VarID(1), 1.0) // Q at 3
	tm.SetTruth(4, fuzzy.VarID(0), 0.0)

	tv, err := tm.EvalUntil(Atom{ID: fuzzy.VarID(0)}, Atom{ID: fuzzy.VarID(1)}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 1.0 {
		t.Errorf("P U Q when Q at 3 and P at 0,1,2: got %v, want 1.0", tv)
	}
}

func TestEvalUntilNeverHolds(t *testing.T) {
	fx := newTemporalFixtures(t)
	tm := setupTimeline(t, fx, 3)
	// Q never holds
	tm.SetTruth(0, fuzzy.VarID(0), 1.0)
	tm.SetTruth(1, fuzzy.VarID(0), 1.0)
	tm.SetTruth(2, fuzzy.VarID(0), 1.0)

	tv, err := tm.EvalUntil(Atom{ID: fuzzy.VarID(0)}, Atom{ID: fuzzy.VarID(1)}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.0 {
		t.Errorf("P U Q when Q never holds: got %v, want 0.0", tv)
	}
}

func TestEvalUntilPBreaksBeforeQ(t *testing.T) {
	fx := newTemporalFixtures(t)
	tm := setupTimeline(t, fx, 4)
	tm.SetTruth(0, fuzzy.VarID(0), 1.0) // P at 0
	tm.SetTruth(1, fuzzy.VarID(0), 0.0) // P breaks at 1
	tm.SetTruth(2, fuzzy.VarID(1), 1.0) // Q at 2, but P already broke

	tv, err := tm.EvalUntil(Atom{ID: fuzzy.VarID(0)}, Atom{ID: fuzzy.VarID(1)}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.0 {
		t.Errorf("P U Q when P breaks before Q: got %v, want 0.0", tv)
	}
}

func TestEvalUntilOutOfBounds(t *testing.T) {
	fx := newTemporalFixtures(t)
	tm := setupTimeline(t, fx, 2)

	tv, err := tm.EvalUntil(Atom{ID: fuzzy.VarID(0)}, Atom{ID: fuzzy.VarID(1)}, 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.0 {
		t.Errorf("out of bounds: got %v, want 0.0", tv)
	}
}

func TestEvalWeakUntil(t *testing.T) {
	fx := newTemporalFixtures(t)
	tm := setupTimeline(t, fx, 4)
	tm.SetTruth(0, fuzzy.VarID(0), 1.0) // P
	tm.SetTruth(1, fuzzy.VarID(0), 1.0) // P
	tm.SetTruth(2, fuzzy.VarID(1), 1.0) // Q at 2
	tm.SetTruth(3, fuzzy.VarID(0), 0.0)

	tv, err := tm.EvalWeakUntil(Atom{ID: fuzzy.VarID(0)}, Atom{ID: fuzzy.VarID(1)}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 1.0 {
		t.Errorf("P W Q when Q at 2: got %v, want 1.0", tv)
	}
}

func TestEvalWeakUntilQNeverHolds(t *testing.T) {
	fx := newTemporalFixtures(t)
	tm := setupTimeline(t, fx, 3)
	tm.SetTruth(0, fuzzy.VarID(0), 1.0)
	tm.SetTruth(1, fuzzy.VarID(0), 1.0)
	tm.SetTruth(2, fuzzy.VarID(0), 1.0)

	tv, err := tm.EvalWeakUntil(Atom{ID: fuzzy.VarID(0)}, Atom{ID: fuzzy.VarID(1)}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 1.0 {
		t.Errorf("P W Q when Q never holds but P always: got %v, want 1.0", tv)
	}
}

func TestEvalWeakUntilPFailsBeforeQ(t *testing.T) {
	fx := newTemporalFixtures(t)
	tm := setupTimeline(t, fx, 4)
	tm.SetTruth(0, fuzzy.VarID(0), 1.0) // P
	tm.SetTruth(1, fuzzy.VarID(0), 0.0) // P breaks
	tm.SetTruth(2, fuzzy.VarID(1), 1.0) // Q at 2, but P already broke

	tv, err := tm.EvalWeakUntil(Atom{ID: fuzzy.VarID(0)}, Atom{ID: fuzzy.VarID(1)}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.0 {
		t.Errorf("P W Q when P breaks before Q: got %v, want 0.0", tv)
	}
}

func TestEvalWeakUntilPFailsAndQNeverHolds(t *testing.T) {
	fx := newTemporalFixtures(t)
	tm := setupTimeline(t, fx, 3)
	tm.SetTruth(0, fuzzy.VarID(0), 1.0)
	tm.SetTruth(1, fuzzy.VarID(0), 0.0) // P breaks here
	tm.SetTruth(2, fuzzy.VarID(0), 0.0)

	tv, err := tm.EvalWeakUntil(Atom{ID: fuzzy.VarID(0)}, Atom{ID: fuzzy.VarID(1)}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.0 {
		t.Errorf("P W Q when P fails at 1 and Q never: got %v, want 0.0", tv)
	}
}

func TestEvalAlwaysFuzzy(t *testing.T) {
	fx := newTemporalFixtures(t)
	tm := setupTimeline(t, fx, 3)
	tm.SetTruth(0, fuzzy.VarID(0), 1.0)
	tm.SetTruth(1, fuzzy.VarID(0), 0.7)
	tm.SetTruth(2, fuzzy.VarID(0), 0.5)

	tv, err := tm.EvalAlways(Atom{ID: fuzzy.VarID(0)}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.5 {
		t.Errorf("□P fuzzy: got %v, want 0.5 (min)", tv)
	}
}

func TestEvalEventuallyFuzzy(t *testing.T) {
	fx := newTemporalFixtures(t)
	tm := setupTimeline(t, fx, 3)
	tm.SetTruth(0, fuzzy.VarID(0), 0.2)
	tm.SetTruth(1, fuzzy.VarID(0), 0.5)
	tm.SetTruth(2, fuzzy.VarID(0), 0.9)

	tv, err := tm.EvalEventually(Atom{ID: fuzzy.VarID(0)}, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 0.9 {
		t.Errorf("◇P fuzzy: got %v, want 0.9 (max)", tv)
	}
}

func TestTemporalLinearity(t *testing.T) {
	// Verify that temporal operators respect the linear timeline:
	// worlds ONLY have access to future worlds, not past or arbitrary worlds.
	fx := newTemporalFixtures(t)
	tm := setupTimeline(t, fx, 4)
	tm.SetTruth(0, fuzzy.VarID(0), 1.0) // early, before start
	tm.SetTruth(1, fuzzy.VarID(0), 0.0)
	tm.SetTruth(2, fuzzy.VarID(0), 1.0) // after start
	tm.SetTruth(3, fuzzy.VarID(0), 0.0)

	// Starting at world 1, Always should NOT see world 0 (it's in the past)
	tv, err := tm.EvalAlways(Atom{ID: fuzzy.VarID(0)}, 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// World 1 = 0.0, so min(0.0, 1.0, 0.0) = 0.0 — world 0 excluded
	if tv != 0.0 {
		t.Errorf("□P from w1 should not see past world w0=1.0: got %v, want 0.0", tv)
	}

	// Starting at world 2, Eventually should see world 2 but not world 0
	tv, err = tm.EvalEventually(Atom{ID: fuzzy.VarID(0)}, 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tv != 1.0 {
		t.Errorf("◇P from w2 should see w2=1.0: got %v, want 1.0", tv)
	}
}
