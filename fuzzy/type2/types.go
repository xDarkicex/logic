package type2

import (
	"fmt"

	"github.com/xDarkicex/logic/fuzzy"
)

// FOUInterval represents a [lower, upper] truth value interval.
type FOUInterval struct {
	Lower fuzzy.TruthValue
	Upper fuzzy.TruthValue
}

// IntervalType2Set represents an Interval Type-2 fuzzy set,
// characterized by an Upper Membership Function (UMF) and a Lower Membership Function (LMF).
type IntervalType2Set struct {
	ID    fuzzy.VarID
	Upper *fuzzy.FuzzySet
	Lower *fuzzy.FuzzySet
}

// NewIntervalType2Set creates a new IntervalType2Set.
func NewIntervalType2Set(id fuzzy.VarID, upper, lower *fuzzy.FuzzySet) (*IntervalType2Set, error) {
	if len(upper.Universe) != len(lower.Universe) {
		return nil, fmt.Errorf("upper and lower universe lengths must match")
	}
	
	// Enforce Upper >= Lower
	for i := 0; i < len(upper.Members); i++ {
		if lower.Members[i] > upper.Members[i] {
			return nil, fmt.Errorf("lower membership cannot exceed upper membership at index %d", i)
		}
	}

	return &IntervalType2Set{
		ID:    id,
		Upper: upper,
		Lower: lower,
	}, nil
}

// Membership evaluates the Footprint of Uncertainty (FOU) at a crisp input x.
func (s *IntervalType2Set) Membership(x float64) FOUInterval {
	return FOUInterval{
		Lower: s.Lower.Membership(x),
		Upper: s.Upper.Membership(x),
	}
}

// Type2LinguisticVar maps linguistic terms to IntervalType2Sets.
type Type2LinguisticVar struct {
	ID    fuzzy.VarID
	Terms map[fuzzy.VarID]*IntervalType2Set
}

// NewType2LinguisticVar creates a new Type2LinguisticVar.
func NewType2LinguisticVar(id fuzzy.VarID) *Type2LinguisticVar {
	return &Type2LinguisticVar{
		ID:    id,
		Terms: make(map[fuzzy.VarID]*IntervalType2Set),
	}
}

// AddTerm registers a term.
func (lv *Type2LinguisticVar) AddTerm(id fuzzy.VarID, set *IntervalType2Set) {
	lv.Terms[id] = set
}

// GetTerm retrieves a term by ID.
func (lv *Type2LinguisticVar) GetTerm(id fuzzy.VarID) *IntervalType2Set {
	return lv.Terms[id]
}
