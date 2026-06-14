package fuzzy

import (
	"math"

	"github.com/xDarkicex/memory"
)

// Union returns the maximum membership between two fuzzy sets.
func Union(a, b *FuzzySet, pool *memory.Pool) *FuzzySet {
	length := len(a.Universe)
	res := NewFuzzySet(0, a.Universe, pool)
	for i := 0; i < length; i++ {
		res.Members[i] = MaxTConorm(a.Members[i], b.Members[i])
	}
	return res
}

// Intersection returns the minimum membership between two fuzzy sets.
func Intersection(a, b *FuzzySet, pool *memory.Pool) *FuzzySet {
	length := len(a.Universe)
	res := NewFuzzySet(0, a.Universe, pool)
	for i := 0; i < length; i++ {
		res.Members[i] = MinTNorm(a.Members[i], b.Members[i])
	}
	return res
}

// Complement returns the negation (1 - μ) of a fuzzy set.
func Complement(a *FuzzySet, pool *memory.Pool) *FuzzySet {
	length := len(a.Universe)
	res := NewFuzzySet(0, a.Universe, pool)
	for i := 0; i < length; i++ {
		res.Members[i] = StandardNegation(a.Members[i])
	}
	return res
}

// IsSubset checks if ∀x: μ_a(x) ≤ μ_b(x).
func IsSubset(a, b *FuzzySet) bool {
	for i, mA := range a.Members {
		if mA > b.Members[i] {
			return false
		}
	}
	return true
}

// Equals checks if two sets have identical memberships.
func Equals(a, b *FuzzySet) bool {
	if len(a.Universe) != len(b.Universe) {
		return false
	}
	for i, mA := range a.Members {
		if !mA.Equals(b.Members[i]) {
			return false
		}
	}
	return true
}

// Concentration squares the membership function (dilation^times equivalent if integer).
// Represents μ^times.
func Concentration(a *FuzzySet, times int, pool *memory.Pool) *FuzzySet {
	length := len(a.Universe)
	res := NewFuzzySet(0, a.Universe, pool)
	for i := 0; i < length; i++ {
		res.Members[i] = TruthValue(math.Pow(float64(a.Members[i]), float64(times)))
	}
	return res
}

// Dilation raises the membership function to 1/times.
func Dilation(a *FuzzySet, times int, pool *memory.Pool) *FuzzySet {
	length := len(a.Universe)
	res := NewFuzzySet(0, a.Universe, pool)
	power := 1.0 / float64(times)
	for i := 0; i < length; i++ {
		res.Members[i] = TruthValue(math.Pow(float64(a.Members[i]), power))
	}
	return res
}

// CartesianProduct returns a 2D matrix (represented flatly) using MinTNorm.
// Resulting universe size is len(a.Universe) * len(b.Universe).
func CartesianProduct(a, b *FuzzySet, pool *memory.Pool) *FuzzySet {
	n := len(a.Universe)
	m := len(b.Universe)
	total := n * m

	// Flat universe for pairs: since FuzzySet is 1D, we store flat indices
	// (this might not be the most pure representation, but satisfies the signature).
	// A more robust Cartesian product might return a different matrix type, but
	// for Phase 3 constraints, we map it into a 1D FuzzySet where Universe is an index.
	uCopy := memory.MustPoolSlice[float64](pool, total)
	uCopy = uCopy[:total]
	members := memory.MustPoolSlice[TruthValue](pool, total)
	members = members[:total]

	idx := 0
	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			uCopy[idx] = float64(idx) // Proxy universe
			members[idx] = MinTNorm(a.Members[i], b.Members[j])
			idx++
		}
	}

	return &FuzzySet{
		ID:       0,
		Universe: uCopy,
		Members:  members,
	}
}
