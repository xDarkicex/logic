package fuzzy

import (
	"testing"
)

func TestGeneralActivation(t *testing.T) {
	a := NewGeneralActivation()
	if a.Name() != "General" {
		t.Errorf("name = %s, want General", a.Name())
	}

	strengths := []TruthValue{0.3, 0.7, 0.0}
	a.Activate(strengths)
	// Should be unchanged
	assertClose(t, 0.3, strengths[0], "strength 0 unchanged")
	assertClose(t, 0.7, strengths[1], "strength 1 unchanged")
	assertClose(t, 0.0, strengths[2], "strength 2 unchanged")
}

func TestProportionalActivation(t *testing.T) {
	a := NewProportionalActivation()
	if a.Name() != "Proportional" {
		t.Errorf("name = %s, want Proportional", a.Name())
	}

	t.Run("normalizes to sum 1", func(t *testing.T) {
		strengths := []TruthValue{0.3, 0.7}
		a.Activate(strengths)
		assertClose(t, 0.3, strengths[0], "prop 0.3")
		assertClose(t, 0.7, strengths[1], "prop 0.7")
		var sum float64
		for _, s := range strengths {
			sum += float64(s)
		}
		if !approxEqual(sum, 1.0) {
			t.Errorf("sum = %v, want 1.0", sum)
		}
	})

	t.Run("all zeros unchanged", func(t *testing.T) {
		strengths := []TruthValue{0, 0, 0}
		a.Activate(strengths)
		for i, s := range strengths {
			if s != 0 {
				t.Errorf("strength[%d] = %v, want 0", i, s)
			}
		}
	})
}

func TestThresholdActivation(t *testing.T) {
	t.Run("GreaterThan", func(t *testing.T) {
		a := NewThresholdActivation(0.5, GreaterThan)
		strengths := []TruthValue{0.3, 0.7, 0.5, 0.9}
		a.Activate(strengths)
		assertClose(t, 0.0, strengths[0], "0.3 not > 0.5")
		assertClose(t, 0.7, strengths[1], "0.7 > 0.5")
		assertClose(t, 0.0, strengths[2], "0.5 not > 0.5")
		assertClose(t, 0.9, strengths[3], "0.9 > 0.5")
	})

	t.Run("GreaterOrEqual", func(t *testing.T) {
		a := NewThresholdActivation(0.5, GreaterOrEqual)
		strengths := []TruthValue{0.3, 0.5}
		a.Activate(strengths)
		assertClose(t, 0.0, strengths[0], "0.3 not >= 0.5")
		assertClose(t, 0.5, strengths[1], "0.5 >= 0.5")
	})

	t.Run("LessThan", func(t *testing.T) {
		a := NewThresholdActivation(0.5, LessThan)
		strengths := []TruthValue{0.3, 0.7}
		a.Activate(strengths)
		assertClose(t, 0.3, strengths[0], "0.3 < 0.5")
		assertClose(t, 0.0, strengths[1], "0.7 not < 0.5")
	})

	t.Run("Equal", func(t *testing.T) {
		a := NewThresholdActivation(0.5, Equal)
		// 0.5 passes (within tolerance), 0.8 does not
		strengths := []TruthValue{0.5, 0.8}
		a.Activate(strengths)
		assertClose(t, 0.5, strengths[0], "0.5 == 0.5 (kept)")
		assertClose(t, 0.0, strengths[1], "0.8 != 0.5 (zeroed)")
	})

	t.Run("NotEqual", func(t *testing.T) {
		a := NewThresholdActivation(0.5, NotEqual)
		strengths := []TruthValue{0.5, 0.7}
		a.Activate(strengths)
		assertClose(t, 0.0, strengths[0], "0.5 == 0.5 (should be zeroed)")
		assertClose(t, 0.7, strengths[1], "0.7 != 0.5")
	})
}

func TestFirstActivation(t *testing.T) {
	a := NewFirstActivation(2)
	if a.Name() != "First" {
		t.Errorf("name = %s, want First", a.Name())
	}

	t.Run("keeps first N", func(t *testing.T) {
		strengths := []TruthValue{0.3, 0.7, 0.5, 0.9}
		a.Activate(strengths)
		assertClose(t, 0.3, strengths[0], "first kept")
		assertClose(t, 0.7, strengths[1], "second kept")
		assertClose(t, 0.0, strengths[2], "third zeroed")
		assertClose(t, 0.0, strengths[3], "fourth zeroed")
	})

	t.Run("N larger than length", func(t *testing.T) {
		a2 := NewFirstActivation(10)
		strengths := []TruthValue{0.3, 0.7}
		a2.Activate(strengths)
		assertClose(t, 0.3, strengths[0], "all kept 0")
		assertClose(t, 0.7, strengths[1], "all kept 1")
	})
}

func TestLastActivation(t *testing.T) {
	a := NewLastActivation(2)
	if a.Name() != "Last" {
		t.Errorf("name = %s, want Last", a.Name())
	}

	strengths := []TruthValue{0.3, 0.7, 0.5, 0.9}
	a.Activate(strengths)
	assertClose(t, 0.0, strengths[0], "first zeroed")
	assertClose(t, 0.0, strengths[1], "second zeroed")
	assertClose(t, 0.5, strengths[2], "third kept")
	assertClose(t, 0.9, strengths[3], "fourth kept")
}

func TestHighestActivation(t *testing.T) {
	a := NewHighestActivation(2)
	if a.Name() != "Highest" {
		t.Errorf("name = %s, want Highest", a.Name())
	}

	strengths := []TruthValue{0.3, 0.9, 0.5, 0.1}
	a.Activate(strengths)
	assertClose(t, 0.0, strengths[0], "0.3 below top 2")
	assertClose(t, 0.9, strengths[1], "0.9 top")
	assertClose(t, 0.5, strengths[2], "0.5 second highest")
	assertClose(t, 0.0, strengths[3], "0.1 below top 2")
}

func TestLowestActivation(t *testing.T) {
	a := NewLowestActivation(2)
	if a.Name() != "Lowest" {
		t.Errorf("name = %s, want Lowest", a.Name())
	}

	strengths := []TruthValue{0.3, 0.9, 0.2, 0.1}
	a.Activate(strengths)
	assertClose(t, 0.0, strengths[0], "0.3 above 2 lowest")
	assertClose(t, 0.0, strengths[1], "0.9 above 2 lowest")
	assertClose(t, 0.2, strengths[2], "0.2 is among 2 lowest")
	assertClose(t, 0.1, strengths[3], "0.1 is the lowest")
}
