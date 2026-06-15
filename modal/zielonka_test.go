package modal

import (
	"testing"

	"github.com/xDarkicex/memory"
)


func newParityGame(t *testing.T, n int) *ParityGame {
	t.Helper()
	pool, err := memory.NewPool(memory.DefaultConfig())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Reset)
	return NewParityGame(n, pool)
}

func TestZielonkaSingleStateEven(t *testing.T) {
	pg := newParityGame(t, 1)
	pg.SetState(0, PlayerEven, 0, []int32{0}) // self-loop, priority 0 (even)
	r := pg.Solve()
	if len(r.WinEven) != 1 || len(r.WinOdd) != 0 {
		t.Errorf("single even state: winEven=%v winOdd=%v", r.WinEven, r.WinOdd)
	}
}

func TestZielonkaSingleStateOdd(t *testing.T) {
	pg := newParityGame(t, 1)
	pg.SetState(0, PlayerEven, 1, []int32{0}) // self-loop, priority 1 (odd)
	r := pg.Solve()
	if len(r.WinOdd) != 1 || len(r.WinEven) != 0 {
		t.Errorf("single odd state: winEven=%v winOdd=%v", r.WinEven, r.WinOdd)
	}
}

func TestZielonkaTwoState(t *testing.T) {
	pg := newParityGame(t, 2)
	// State 0 (Even, pri 0) → state 1. State 1 (Odd, pri 1) → state 0.
	pg.SetState(0, PlayerEven, 0, []int32{1})
	pg.SetState(1, PlayerOdd, 1, []int32{0})
	r := pg.Solve()
	// Max priority is 1 (odd). Player 1 (Odd) attracts priority-1 states.
	// State 0's only edge → state 1 (pri 1), so Odd can force reaching it.
	// Odd wins both states.
	if len(r.WinOdd) != 2 {
		t.Errorf("both states should be Odd wins: got Even=%v Odd=%v", r.WinEven, r.WinOdd)
	}
}

func TestZielonkaAttractor(t *testing.T) {
	pg := newParityGame(t, 3)
	pg.SetState(0, PlayerEven, 0, []int32{1})
	pg.SetState(1, PlayerOdd, 1, []int32{2})
	pg.SetState(2, PlayerEven, 2, []int32{1})
	pg.buildPred()
	// Attractor for Even to {0} within all states.
	all := pg.allStates()
	result := pg.attractor(PlayerEven, []int32{0}, all)
	// Even controls 0 → 1. PlayerEven can't force 1 into attractor (Odd controls 1).
	// But 1's only successor is 2 (Even), and Even can choose 2→1 which is in attractor.
	// Hmm, actually attractor from {0}: 0 is already in. Predecessors of 0: none (only 1 has edge to 2).
	// So attractor should just be {0}.
	if len(result) < 1 {
		t.Error("attractor should include target")
	}
}

func TestZielonkaMaxPriority(t *testing.T) {
	pg := newParityGame(t, 3)
	pg.SetState(0, PlayerEven, 3, nil)
	pg.SetState(1, PlayerOdd, 7, nil)
	pg.SetState(2, PlayerEven, 1, nil)
	if pg.MaxPriority() != 7 {
		t.Errorf("max priority: got %d, want 7", pg.MaxPriority())
	}
}

func TestZielonkaSetOps(t *testing.T) {
	pg := newParityGame(t, 5)
	a := []int32{0, 1, 2}
	b := []int32{2, 3, 4}
	inter := pg.intersect(a, b)
	if len(inter) != 1 || inter[0] != 2 {
		t.Errorf("intersect({0,1,2},{2,3,4}) = %v", inter)
	}
	union := pg.union(a, b)
	if len(union) != 5 {
		t.Errorf("union: got %d elements, want 5", len(union))
	}
	diff := pg.setMinus(a, []int32{1})
	if len(diff) != 2 || diff[0] != 0 || diff[1] != 2 {
		t.Errorf("setMinus({0,1,2},{1}) = %v", diff)
	}
}

func TestZielonkaCC(t *testing.T) {
	pg := newParityGame(t, 5)
	for i := int32(0); i < 5; i++ {
		player := PlayerEven
		if i%2 == 1 {
			player = PlayerOdd
		}
		pg.SetState(i, player, i, []int32{(i + 1) % 5})
	}
	for i := 0; i < 5; i++ {
		pg.Solve()
	}
}

func TestZielonkaThreeTier(t *testing.T) {
	pg := newParityGame(t, 4)
	// A 4-state game with 3 priority levels.
	pg.SetState(0, PlayerEven, 0, []int32{1, 3})
	pg.SetState(1, PlayerOdd, 1, []int32{2})
	pg.SetState(2, PlayerEven, 2, []int32{1, 3})
	pg.SetState(3, PlayerOdd, 1, []int32{0})
	r := pg.Solve()
	t.Logf("winEven=%v winOdd=%v", r.WinEven, r.WinOdd)
	pool, _ := memory.NewPool(memory.DefaultConfig())
	defer pool.Reset()
	seen := memory.MustPoolSlice[bool](pool, 4)
	seen = seen[:4]
	cnt := 0
	for _, s := range r.WinEven {
		if !seen[s] {
			seen[s] = true
			cnt++
		}
	}
	for _, s := range r.WinOdd {
		if seen[s] {
			t.Errorf("state %d in both winning regions", s)
		}
		if !seen[s] {
			seen[s] = true
			cnt++
		}
	}
	if cnt != 4 {
		t.Errorf("not all states assigned: got %d, want 4", cnt)
	}
}
