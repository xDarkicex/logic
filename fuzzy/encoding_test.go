package fuzzy

import (
	"testing"

	"github.com/xDarkicex/memory"
)

func TestPopulationEncoder(t *testing.T) {
	pool, err := memory.NewPool(memory.DefaultConfig())
	if err != nil {
		t.Fatalf("failed to create pool: %v", err)
	}
	defer pool.Free()

	// n <= 0
	funcs := PopulationEncoder(0, 0, 100, pool)
	if len(funcs) != 0 {
		t.Errorf("Expected 0 funcs, got %d", len(funcs))
	}

	funcs = PopulationEncoder(-1, 0, 100, pool)
	if len(funcs) != 0 {
		t.Errorf("Expected 0 funcs, got %d", len(funcs))
	}

	// n == 1
	funcs = PopulationEncoder(1, 0, 100, pool)
	if len(funcs) != 1 {
		t.Fatalf("Expected 1 func, got %d", len(funcs))
	}
	// It should peak at 50, and hit 0 at 0 and 100
	assertClose(t, 1.0, funcs[0](50), "Single func peak")
	assertClose(t, 0.0, funcs[0](0), "Single func min")
	assertClose(t, 0.0, funcs[0](100), "Single func max")

	// n == 5
	// min=0, max=100. Step = 100 / 4 = 25
	// Peaks at: 0, 25, 50, 75, 100
	funcs = PopulationEncoder(5, 0, 100, pool)
	if len(funcs) != 5 {
		t.Fatalf("Expected 5 funcs, got %d", len(funcs))
	}

	// Func 0 peaks at 0, goes to 0 at 25
	assertClose(t, 1.0, funcs[0](0), "Func 0 peak")
	assertClose(t, 0.0, funcs[0](25), "Func 0 end")

	// Func 1 peaks at 25, goes to 0 at 0 and 50
	assertClose(t, 0.0, funcs[1](0), "Func 1 start")
	assertClose(t, 1.0, funcs[1](25), "Func 1 peak")
	assertClose(t, 0.0, funcs[1](50), "Func 1 end")

	// Verify overlapping summation at midpoint (12.5) -> should be 0.5 + 0.5 = 1.0
	assertClose(t, 0.5, funcs[0](12.5), "Func 0 at 12.5")
	assertClose(t, 0.5, funcs[1](12.5), "Func 1 at 12.5")
}
