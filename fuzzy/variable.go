package fuzzy

// Defuzzifier computes a crisp value from a fuzzy set.
type Defuzzifier interface {
	Defuzzify(set *FuzzySet) float64
	Name() string
}

// centroidDefuzz wraps the Centroid defuzzification function.
type centroidDefuzz struct{}

func (d *centroidDefuzz) Defuzzify(set *FuzzySet) float64 { return Centroid(set) }
func (d *centroidDefuzz) Name() string                     { return "Centroid" }

// meanOfMaxDefuzz wraps MeanOfMax.
type meanOfMaxDefuzz struct{}

func (d *meanOfMaxDefuzz) Defuzzify(set *FuzzySet) float64 { return MeanOfMax(set) }
func (d *meanOfMaxDefuzz) Name() string                     { return "MeanOfMaximum" }

// smallestOfMaxDefuzz wraps SmallestOfMax.
type smallestOfMaxDefuzz struct{}

func (d *smallestOfMaxDefuzz) Defuzzify(set *FuzzySet) float64 { return SmallestOfMax(set) }
func (d *smallestOfMaxDefuzz) Name() string                     { return "SmallestOfMaximum" }

// largestOfMaxDefuzz wraps LargestOfMax.
type largestOfMaxDefuzz struct{}

func (d *largestOfMaxDefuzz) Defuzzify(set *FuzzySet) float64 { return LargestOfMax(set) }
func (d *largestOfMaxDefuzz) Name() string                     { return "LargestOfMaximum" }

// bisectorDefuzz wraps Bisector.
type bisectorDefuzz struct{}

func (d *bisectorDefuzz) Defuzzify(set *FuzzySet) float64 { return Bisector(set) }
func (d *bisectorDefuzz) Name() string                     { return "Bisector" }

// NewCentroidDefuzzifier returns a Centroid defuzzifier.
func NewCentroidDefuzzifier() Defuzzifier { return &centroidDefuzz{} }

// NewMeanOfMaxDefuzzifier returns a MeanOfMaximum defuzzifier.
func NewMeanOfMaxDefuzzifier() Defuzzifier { return &meanOfMaxDefuzz{} }

// NewSmallestOfMaxDefuzzifier returns a SmallestOfMaximum defuzzifier.
func NewSmallestOfMaxDefuzzifier() Defuzzifier { return &smallestOfMaxDefuzz{} }

// NewLargestOfMaxDefuzzifier returns a LargestOfMaximum defuzzifier.
func NewLargestOfMaxDefuzzifier() Defuzzifier { return &largestOfMaxDefuzz{} }

// NewBisectorDefuzzifier returns a Bisector defuzzifier.
func NewBisectorDefuzzifier() Defuzzifier { return &bisectorDefuzz{} }

// OutputVariable wraps a LinguisticVar with output-specific configuration.
// Ported from fuzzylite's OutputVariable class.
// Multiple OutputVariables allow per-output defuzzification strategies.
type OutputVariable struct {
	Variable       *LinguisticVar
	Defuzzifier    Defuzzifier
	Aggregation    func(a, b TruthValue) TruthValue
	DefaultValue   float64
	LockValidRange bool
	ValidMin       float64
	ValidMax       float64
	fuzzyOutput    *FuzzySet // accumulated result during evaluation
}

// NewOutputVariable creates an OutputVariable wrapping the given LinguisticVar.
func NewOutputVariable(variable *LinguisticVar) *OutputVariable {
	return &OutputVariable{
		Variable:     variable,
		Defuzzifier:  NewCentroidDefuzzifier(),
		Aggregation:  MaxTConorm,
		DefaultValue: 0,
		ValidMin:     0,
		ValidMax:     1,
	}
}

// SetDefuzzifier sets the defuzzification method.
func (ov *OutputVariable) SetDefuzzifier(d Defuzzifier) { ov.Defuzzifier = d }

// SetAggregation sets the aggregation operator.
func (ov *OutputVariable) SetAggregation(fn func(a, b TruthValue) TruthValue) { ov.Aggregation = fn }

// SetDefaultValue sets the fallback value when defuzzification fails.
func (ov *OutputVariable) SetDefaultValue(v float64) { ov.DefaultValue = v }

// SetLockValidRange locks the output value to [min, max].
func (ov *OutputVariable) SetLockValidRange(min, max float64) {
	ov.LockValidRange = true
	ov.ValidMin = min
	ov.ValidMax = max
}

// Defuzzify defuzzifies the accumulated fuzzy output, applying default and lock-range.
func (ov *OutputVariable) Defuzzify() float64 {
	if ov.fuzzyOutput == nil || ov.fuzzyOutput.Height() == 0 {
		return ov.DefaultValue
	}
	result := ov.Defuzzifier.Defuzzify(ov.fuzzyOutput)
	if ov.LockValidRange {
		if result < ov.ValidMin {
			result = ov.ValidMin
		}
		if result > ov.ValidMax {
			result = ov.ValidMax
		}
	}
	return result
}

// clear resets the accumulated fuzzy output.
func (ov *OutputVariable) clear() {
	ov.fuzzyOutput = nil
}
