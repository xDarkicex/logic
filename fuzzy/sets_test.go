package fuzzy

import (
	"testing"

	"github.com/xDarkicex/memory"
)

func TestSetOperations(t *testing.T) {
	cfg := memory.DefaultConfig()
	pool, err := memory.NewPool(cfg)
	if err != nil {
		t.Fatalf("Failed to create pool: %v", err)
	}
	defer pool.Free()

	universe := []float64{10, 20, 30}
	a := NewFuzzySet(1, universe, pool)
	a.Members[0] = 0.2
	a.Members[1] = 0.8
	a.Members[2] = 0.5

	b := NewFuzzySet(2, universe, pool)
	b.Members[0] = 0.5
	b.Members[1] = 0.3
	b.Members[2] = 0.9

	// Union
	u := Union(a, b, pool)
	assertClose(t, 0.5, u.Members[0], "Union 0")
	assertClose(t, 0.8, u.Members[1], "Union 1")
	assertClose(t, 0.9, u.Members[2], "Union 2")

	// Intersection
	i := Intersection(a, b, pool)
	assertClose(t, 0.2, i.Members[0], "Intersect 0")
	assertClose(t, 0.3, i.Members[1], "Intersect 1")
	assertClose(t, 0.5, i.Members[2], "Intersect 2")

	// Complement
	c := Complement(a, pool)
	assertClose(t, 0.8, c.Members[0], "Comp 0")
	assertClose(t, 0.2, c.Members[1], "Comp 1")
	assertClose(t, 0.5, c.Members[2], "Comp 2")

	// IsSubset
	if IsSubset(a, b) {
		t.Error("a is not subset of b")
	}
	subA := NewFuzzySet(3, universe, pool)
	subA.Members[0] = 0.1
	subA.Members[1] = 0.5
	subA.Members[2] = 0.4
	if !IsSubset(subA, a) {
		t.Error("subA should be subset of a")
	}

	// Equals
	if Equals(a, b) {
		t.Error("a is not equal to b")
	}
	aCopy := NewFuzzySet(4, universe, pool)
	copy(aCopy.Members, a.Members)
	if !Equals(a, aCopy) {
		t.Error("a should equal aCopy")
	}
	diffLen := NewFuzzySet(5, []float64{10, 20}, pool)
	if Equals(a, diffLen) {
		t.Error("Sets with different universe length are not equal")
	}

	// Concentration
	conc := Concentration(a, 2, pool)
	assertClose(t, 0.04, conc.Members[0], "Conc 0")

	// Dilation
	dil := Dilation(a, 2, pool)
	assertClose(t, TruthValue(0.4472135954999579), dil.Members[0], "Dil 0")

	// Cartesian Product
	cp := CartesianProduct(a, b, pool)
	if len(cp.Universe) != 9 {
		t.Errorf("Expected cp universe len 9, got %v", len(cp.Universe))
	}
	// a.Members[0]=0.2, b.Members[0]=0.5 -> MinTNorm -> 0.2
	assertClose(t, 0.2, cp.Members[0], "CP 0,0")
	// a.Members[2]=0.5, b.Members[2]=0.9 -> MinTNorm -> 0.5
	assertClose(t, 0.5, cp.Members[8], "CP 2,2")
}
