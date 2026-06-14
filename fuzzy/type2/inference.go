package type2

import (
	"fmt"

	"github.com/xDarkicex/logic/core"
	"github.com/xDarkicex/logic/fuzzy"
	"github.com/xDarkicex/memory"
)

// evaluateType2Antecedents evaluates the antecedents for a Type-2 rule,
// returning the FOUInterval (firing strength footprint).
func evaluateType2Antecedents(rule fuzzy.FuzzyRule, inputs map[fuzzy.VarID]float64, variables map[fuzzy.VarID]*Type2LinguisticVar, tnorm func(a, b fuzzy.TruthValue) fuzzy.TruthValue) (FOUInterval, error) {
	if len(rule.Antecedents) == 0 {
		return FOUInterval{Lower: rule.Weight, Upper: rule.Weight}, nil
	}

	firingStrength := FOUInterval{Lower: 1.0, Upper: 1.0}

	for i, cond := range rule.Antecedents {
		lv, ok := variables[cond.Variable]
		if !ok {
			return FOUInterval{}, fmt.Errorf("variable %d not found", cond.Variable)
		}
		term := lv.GetTerm(cond.Term)
		if term == nil {
			return FOUInterval{}, fmt.Errorf("term %d not found for variable %d", cond.Term, cond.Variable)
		}

		val, hasInput := inputs[cond.Variable]
		if !hasInput {
			return FOUInterval{}, fmt.Errorf("missing input for variable %d", cond.Variable)
		}

		membership := term.Membership(val)
		if cond.Negated {
			// standard negation flips upper/lower bounds: 1 - upper becomes new lower.
			membership = FOUInterval{
				Lower: fuzzy.StandardNegation(membership.Upper),
				Upper: fuzzy.StandardNegation(membership.Lower),
			}
		}

		if i == 0 {
			firingStrength = membership
		} else {
			firingStrength.Lower = tnorm(firingStrength.Lower, membership.Lower)
			firingStrength.Upper = tnorm(firingStrength.Upper, membership.Upper)
		}
	}

	// Apply rule weight
	return FOUInterval{
		Lower: fuzzy.ProductTNorm(firingStrength.Lower, rule.Weight),
		Upper: fuzzy.ProductTNorm(firingStrength.Upper, rule.Weight),
	}, nil
}

// Type2Engine evaluates Interval Type-2 fuzzy rules.
type Type2Engine struct {
	rules     []fuzzy.FuzzyRule
	variables map[fuzzy.VarID]*Type2LinguisticVar
	tnorm     func(a, b fuzzy.TruthValue) fuzzy.TruthValue
	tconorm   func(a, b fuzzy.TruthValue) fuzzy.TruthValue
	implic    func(a, b fuzzy.TruthValue) fuzzy.TruthValue
}

// NewType2Engine creates a new Type2Engine.
func NewType2Engine() *Type2Engine {
	return &Type2Engine{
		variables: make(map[fuzzy.VarID]*Type2LinguisticVar),
		tnorm:     fuzzy.MinTNorm,
		tconorm:   fuzzy.MaxTConorm,
		implic:    fuzzy.MinTNorm,
	}
}

// AddRule adds a Type-2 fuzzy rule.
func (e *Type2Engine) AddRule(rule fuzzy.FuzzyRule) {
	e.rules = append(e.rules, rule)
}

// AddVariable registers a linguistic variable.
func (e *Type2Engine) AddVariable(v *Type2LinguisticVar) {
	e.variables[v.ID] = v
}

// Evaluate runs the inference engine and returns an aggregated IntervalType2Set.
func (e *Type2Engine) Evaluate(inputs map[fuzzy.VarID]float64, pool *memory.Pool) (*IntervalType2Set, error) {
	var aggUpper, aggLower *fuzzy.FuzzySet

	for _, rule := range e.rules {
		firingStrength, err := evaluateType2Antecedents(rule, inputs, e.variables, e.tnorm)
		if err != nil {
			return nil, err
		}

		if firingStrength.Upper == 0 {
			continue // rule does not fire at all
		}

		outVar, ok := e.variables[rule.Consequent.Variable]
		if !ok {
			return nil, fmt.Errorf("output var %d not found", rule.Consequent.Variable)
		}
		outTerm := outVar.GetTerm(rule.Consequent.Term)
		if outTerm == nil {
			return nil, fmt.Errorf("output term %d not found", rule.Consequent.Term)
		}

		ruleOutputUpper := fuzzy.NewFuzzySet(outTerm.ID, outTerm.Upper.Universe, pool)
		ruleOutputLower := fuzzy.NewFuzzySet(outTerm.ID, outTerm.Lower.Universe, pool)

		length := len(outTerm.Upper.Universe)
		for i := 0; i < length; i++ {
			ruleOutputUpper.Members[i] = e.implic(firingStrength.Upper, outTerm.Upper.Members[i])
			ruleOutputLower.Members[i] = e.implic(firingStrength.Lower, outTerm.Lower.Members[i])
		}

		if aggUpper == nil {
			aggUpper = ruleOutputUpper
			aggLower = ruleOutputLower
		} else {
			aggUpper = e.aggregateSets(aggUpper, ruleOutputUpper, pool)
			aggLower = e.aggregateSets(aggLower, ruleOutputLower, pool)
		}
	}

	if aggUpper == nil {
		return nil, core.NewLogicError("Type2Engine", "Evaluate", "no rules fired")
	}

	return &IntervalType2Set{
		ID:    0,
		Upper: aggUpper,
		Lower: aggLower,
	}, nil
}

func (e *Type2Engine) aggregateSets(a, b *fuzzy.FuzzySet, pool *memory.Pool) *fuzzy.FuzzySet {
	length := len(a.Universe)
	res := fuzzy.NewFuzzySet(0, a.Universe, pool)
	for i := 0; i < length; i++ {
		res.Members[i] = e.tconorm(a.Members[i], b.Members[i])
	}
	return res
}

// NieTanDefuzzify uses the Nie-Tan (NT) method to defuzzify an Interval Type-2 Set.
// It averages the upper and lower membership functions and computes the centroid.
// Time: O(n), Space: O(1) (zero allocations). CC=2.
func NieTanDefuzzify(set *IntervalType2Set) float64 {
	var num, den float64

	for i, x := range set.Upper.Universe {
		u := float64(set.Upper.Members[i])
		l := float64(set.Lower.Members[i])
		avgM := (u + l) / 2.0
		
		num += x * avgM
		den += avgM
	}

	if den == 0 {
		return 0
	}
	return num / den
}
