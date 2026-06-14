package fuzzy

import "math"

// Activation controls which rules fire in a RuleBlock.
// All methods mutate the strengths slice in-place to avoid allocations.
// Ported from fuzzylite's Activation class hierarchy.
// Different from implication: implication modulates each rule's consequent,
// while activation selects WHICH rules are evaluated at all.
type Activation interface {
	// Activate filters and modifies rule firing strengths in-place.
	// A strength of 0 means the rule does not fire.
	Activate(strengths []TruthValue)

	// Name returns the activation method name.
	Name() string
}

// GeneralActivation fires every rule with its original strength.
// This is the default — no filtering, all rules participate.
type GeneralActivation struct{}

// NewGeneralActivation creates a General activation method.
func NewGeneralActivation() *GeneralActivation {
	return &GeneralActivation{}
}

// Name returns "General".
func (a *GeneralActivation) Name() string { return "General" }

// Activate is a no-op: all strengths pass through unchanged.
func (a *GeneralActivation) Activate(strengths []TruthValue) {}

// ProportionalActivation normalizes rule strengths so they sum to 1.
// Equivalent to softmax-normalizing rule activations.
type ProportionalActivation struct{}

// NewProportionalActivation creates a Proportional activation method.
func NewProportionalActivation() *ProportionalActivation {
	return &ProportionalActivation{}
}

// Name returns "Proportional".
func (a *ProportionalActivation) Name() string { return "Proportional" }

// Activate normalizes strengths to sum to 1, in-place.
func (a *ProportionalActivation) Activate(strengths []TruthValue) {
	var sum float64
	for _, s := range strengths {
		sum += float64(s)
	}
	if sum == 0 {
		return
	}
	for i, s := range strengths {
		strengths[i] = TruthValue(float64(s) / sum)
	}
}

// ThresholdActivation fires only rules whose strength passes the comparison.
type ThresholdActivation struct {
	Value      TruthValue
	Comparison Comparison
}

// Comparison is a comparison operator for threshold activation.
type Comparison int

const (
	GreaterThan   Comparison = iota // strength > threshold
	GreaterOrEqual                   // strength >= threshold
	LessThan                         // strength < threshold
	LessOrEqual                      // strength <= threshold
	Equal                            // strength == threshold
	NotEqual                         // strength != threshold
)

// NewThresholdActivation creates a Threshold activation method.
func NewThresholdActivation(value TruthValue, cmp Comparison) *ThresholdActivation {
	return &ThresholdActivation{Value: value, Comparison: cmp}
}

// Name returns "Threshold".
func (a *ThresholdActivation) Name() string { return "Threshold" }

// Activate zeroes out strengths that don't satisfy the comparison.
func (a *ThresholdActivation) Activate(strengths []TruthValue) {
	for i, s := range strengths {
		if !a.satisfies(s) {
			strengths[i] = 0
		}
	}
}

func (a *ThresholdActivation) satisfies(s TruthValue) bool {
	switch a.Comparison {
	case GreaterThan:
		return s > a.Value
	case GreaterOrEqual:
		return s >= a.Value
	case LessThan:
		return s < a.Value
	case LessOrEqual:
		return s <= a.Value
	case Equal:
		return math.Abs(float64(s-a.Value)) < 1e-9
	case NotEqual:
		return math.Abs(float64(s-a.Value)) >= 1e-9
	}
	return false
}

// FirstActivation fires only the first N rules (in registration order).
type FirstActivation struct {
	N int
}

// NewFirstActivation creates a First activation method.
func NewFirstActivation(n int) *FirstActivation {
	return &FirstActivation{N: n}
}

// Name returns "First".
func (a *FirstActivation) Name() string { return "First" }

// Activate zeroes strengths beyond the first N.
func (a *FirstActivation) Activate(strengths []TruthValue) {
	for i := a.N; i < len(strengths); i++ {
		strengths[i] = 0
	}
}

// LastActivation fires only the last N rules (in registration order).
// Inverse of FirstActivation. Ported from fuzzylite's Last activation.
type LastActivation struct {
	N int
}

// NewLastActivation creates a Last activation method.
func NewLastActivation(n int) *LastActivation {
	return &LastActivation{N: n}
}

// Name returns "Last".
func (a *LastActivation) Name() string { return "Last" }

// Activate zeroes strengths before the last N.
func (a *LastActivation) Activate(strengths []TruthValue) {
	start := len(strengths) - a.N
	if start < 0 {
		start = 0
	}
	for i := 0; i < start; i++ {
		strengths[i] = 0
	}
}

// HighestActivation fires only the N rules with the highest activation degrees.
// Ported from fuzzylite's Highest activation.
type HighestActivation struct {
	N     int
	scratch []float64 // temp buffer for sorting, caller-provided
}

// NewHighestActivation creates a Highest activation method.
func NewHighestActivation(n int) *HighestActivation {
	return &HighestActivation{N: n}
}

// Name returns "Highest".
func (a *HighestActivation) Name() string { return "Highest" }

// Activate keeps the N highest strengths, zeroes the rest.
func (a *HighestActivation) Activate(strengths []TruthValue) {
	if a.N >= len(strengths) {
		return
	}
	threshold := a.nthHighest(strengths)
	for i, s := range strengths {
		if float64(s) < threshold {
			strengths[i] = 0
		}
	}
}

// nthHighest returns the value of the Nth highest element using selection.
func (a *HighestActivation) nthHighest(s []TruthValue) float64 {
	// Reuse or allocate scratch buffer
	a.scratch = a.scratch[:0]
	for _, v := range s {
		a.scratch = append(a.scratch, float64(v))
	}
	// Partial selection sort for the first n elements (descending)
	for i := 0; i < a.N && i < len(a.scratch); i++ {
		maxIdx := i
		for j := i + 1; j < len(a.scratch); j++ {
			if a.scratch[j] > a.scratch[maxIdx] {
				maxIdx = j
			}
		}
		a.scratch[i], a.scratch[maxIdx] = a.scratch[maxIdx], a.scratch[i]
	}
	return a.scratch[a.N-1]
}

// LowestActivation fires only the N rules with the lowest non-zero activation degrees.
// Ported from fuzzylite's Lowest activation.
type LowestActivation struct {
	N       int
	scratch []float64 // temp buffer, caller-provided
}

// NewLowestActivation creates a Lowest activation method.
func NewLowestActivation(n int) *LowestActivation {
	return &LowestActivation{N: n}
}

// Name returns "Lowest".
func (a *LowestActivation) Name() string { return "Lowest" }

// Activate keeps the N lowest non-zero strengths, zeroes the rest.
func (a *LowestActivation) Activate(strengths []TruthValue) {
	if a.N >= len(strengths) {
		return
	}
	threshold := a.nthLowestNonZero(strengths)
	for i, s := range strengths {
		if float64(s) > threshold || s == 0 {
			strengths[i] = 0
		}
	}
}

// nthLowestNonZero returns the value of the Nth lowest non-zero element.
func (a *LowestActivation) nthLowestNonZero(s []TruthValue) float64 {
	// Filter non-zero into scratch
	a.scratch = a.scratch[:0]
	for _, v := range s {
		if v != 0 {
			a.scratch = append(a.scratch, float64(v))
		}
	}
	n := a.N
	if n > len(a.scratch) {
		n = len(a.scratch)
	}
	if n == 0 {
		return 0
	}
	// Partial selection sort ascending
	for i := 0; i < n && i < len(a.scratch); i++ {
		minIdx := i
		for j := i + 1; j < len(a.scratch); j++ {
			if a.scratch[j] < a.scratch[minIdx] {
				minIdx = j
			}
		}
		a.scratch[i], a.scratch[minIdx] = a.scratch[minIdx], a.scratch[i]
	}
	return a.scratch[n-1]
}
