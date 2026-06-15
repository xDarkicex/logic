package modal

import (
	"testing"

	"github.com/xDarkicex/memory"
)

func newWorldSetPool(t *testing.T) *memory.Pool {
	t.Helper()
	pool, err := memory.NewPool(memory.DefaultConfig())
	if err != nil {
		t.Fatalf("pool: %v", err)
	}
	t.Cleanup(pool.Reset)
	return pool
}

func TestNewWorldSet(t *testing.T) {
	pool := newWorldSetPool(t)
	ws := NewWorldSet(100, pool)
	if ws.Count() != 0 {
		t.Errorf("new set: got %d, want 0", ws.Count())
	}
	if ws.words != 2 {
		t.Errorf("100 worlds: got %d words, want 2", ws.words)
	}
}

func TestWorldSetAddContains(t *testing.T) {
	pool := newWorldSetPool(t)
	ws := NewWorldSet(128, pool)
	ws.Add(5)
	ws.Add(70)
	ws.Add(100)

	if !ws.Contains(5) {
		t.Error("should contain world 5")
	}
	if !ws.Contains(70) {
		t.Error("should contain world 70")
	}
	if !ws.Contains(100) {
		t.Error("should contain world 100")
	}
	if ws.Contains(6) {
		t.Error("should NOT contain world 6")
	}
}

func TestWorldSetRemove(t *testing.T) {
	pool := newWorldSetPool(t)
	ws := NewWorldSet(64, pool)
	ws.Add(10)
	ws.Remove(10)
	if ws.Contains(10) {
		t.Error("should not contain world 10 after remove")
	}
}

func TestWorldSetClear(t *testing.T) {
	pool := newWorldSetPool(t)
	ws := NewWorldSet(64, pool)
	ws.Add(1)
	ws.Add(2)
	ws.Clear()
	if ws.Count() != 0 {
		t.Errorf("after clear: got %d, want 0", ws.Count())
	}
}

func TestWorldSetCount(t *testing.T) {
	pool := newWorldSetPool(t)
	ws := NewWorldSet(200, pool)
	ws.Add(0)
	ws.Add(63)
	ws.Add(64)
	ws.Add(127)
	if ws.Count() != 4 {
		t.Errorf("count: got %d, want 4", ws.Count())
	}
}

func TestWorldSetUnion(t *testing.T) {
	pool := newWorldSetPool(t)
	a := NewWorldSet(128, pool)
	b := NewWorldSet(128, pool)
	a.Add(1)
	a.Add(2)
	b.Add(2)
	b.Add(3)
	a.Union(b)
	if !a.Contains(1) || !a.Contains(2) || !a.Contains(3) {
		t.Error("union should contain worlds 1,2,3")
	}
	if a.Count() != 3 {
		t.Errorf("union count: got %d, want 3", a.Count())
	}
}

func TestWorldSetIntersect(t *testing.T) {
	pool := newWorldSetPool(t)
	a := NewWorldSet(128, pool)
	b := NewWorldSet(128, pool)
	a.Add(1)
	a.Add(2)
	a.Add(3)
	b.Add(2)
	b.Add(3)
	b.Add(4)
	a.Intersect(b)
	if a.Contains(1) || a.Contains(4) {
		t.Error("intersection should NOT contain 1 or 4")
	}
	if !a.Contains(2) || !a.Contains(3) {
		t.Error("intersection should contain 2 and 3")
	}
}

func TestWorldSetSubtract(t *testing.T) {
	pool := newWorldSetPool(t)
	a := NewWorldSet(128, pool)
	b := NewWorldSet(128, pool)
	a.Add(1)
	a.Add(2)
	a.Add(3)
	b.Add(2)
	a.Subtract(b)
	if a.Contains(2) {
		t.Error("subtract should remove world 2")
	}
	if !a.Contains(1) || !a.Contains(3) {
		t.Error("subtract should keep worlds 1,3")
	}
}

func TestWorldSetEquals(t *testing.T) {
	pool := newWorldSetPool(t)
	a := NewWorldSet(64, pool)
	b := NewWorldSet(64, pool)
	a.Add(1)
	a.Add(2)
	b.Add(1)
	b.Add(2)
	if !a.Equals(b) {
		t.Error("identical sets should be equal")
	}
	b.Add(3)
	if a.Equals(b) {
		t.Error("different sets should not be equal")
	}
}

func TestWorldSetIsSubset(t *testing.T) {
	pool := newWorldSetPool(t)
	a := NewWorldSet(64, pool)
	b := NewWorldSet(64, pool)
	a.Add(1)
	a.Add(2)
	b.Add(1)
	b.Add(2)
	b.Add(3)
	if !a.IsSubset(b) {
		t.Error("{1,2} ⊆ {1,2,3}")
	}
	if b.IsSubset(a) {
		t.Error("{1,2,3} ⊄ {1,2}")
	}
}

func TestWorldSetNext(t *testing.T) {
	pool := newWorldSetPool(t)
	ws := NewWorldSet(200, pool)
	ws.Add(0)
	ws.Add(63)
	ws.Add(64)
	ws.Add(127)
	ws.Add(150)

	var got []World
	w, ok := ws.Next(0)
	for ok {
		got = append(got, w)
		w, ok = ws.Next(w + 1)
	}
	if len(got) != 5 {
		t.Errorf("Next iteration: got %d, want 5", len(got))
	}
}

func TestWorldSetFill(t *testing.T) {
	pool := newWorldSetPool(t)
	ws := NewWorldSet(128, pool)
	ws.Fill(100)
	if ws.Count() != 100 {
		t.Errorf("Fill(100): got %d, want 100", ws.Count())
	}
	if !ws.Contains(0) || !ws.Contains(99) {
		t.Error("Fill should include worlds 0-99")
	}
	if ws.Contains(100) {
		t.Error("Fill should NOT include world 100")
	}
}

func TestAccessibleSet(t *testing.T) {
	pool := newWorldSetPool(t)
	arena, _ := memory.NewArena(1024 * 1024)
	defer arena.Free()

	frame := NewFrame(pool, arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	w2 := frame.AddWorld()
	w3 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)
	frame.AddRelation(w0, w2, RelCausal)
	frame.AddRelation(w0, w3, RelCausal)

	ws := frame.AccessibleSet(w0, RelCausal, pool)
	if ws.Count() != 3 {
		t.Errorf("AccessibleSet: got %d, want 3", ws.Count())
	}
	if !ws.Contains(w1) || !ws.Contains(w2) || !ws.Contains(w3) {
		t.Error("should contain all three targets")
	}
}

func TestAccessibleSetWeighted(t *testing.T) {
	pool := newWorldSetPool(t)
	arena, _ := memory.NewArena(1024 * 1024)
	defer arena.Free()

	frame := NewFrame(pool, arena)
	w0 := frame.AddWorld()
	w1 := frame.AddWorld()
	w2 := frame.AddWorld()
	frame.AddRelation(w0, w1, RelCausal)
	frame.AddWeightedRelation(w0, w2, RelCausal, 0.5)

	ws := frame.AccessibleSet(w0, RelCausal, pool)
	if ws.Count() != 2 {
		t.Errorf("mixed edges: got %d, want 2", ws.Count())
	}
}

func TestWorldSetLargeScale(t *testing.T) {
	pool := newWorldSetPool(t)
	ws := NewWorldSet(500, pool)
	for i := 0; i < 500; i += 2 {
		ws.Add(World(i))
	}
	if ws.Count() != 250 {
		t.Errorf("250 even worlds: got %d", ws.Count())
	}
	// Verify random access
	if !ws.Contains(0) || ws.Contains(1) || !ws.Contains(498) || ws.Contains(499) {
		t.Error("even/odd verification failed")
	}
}
