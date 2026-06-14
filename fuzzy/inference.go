package fuzzy

import (
	"fmt"

	"github.com/xDarkicex/logic/core"
	"github.com/xDarkicex/memory"
)

// evaluateAntecedents evaluates the antecedents of a rule and returns the firing strength.
// It uses the provided t-norm for combining conditions.
func evaluateAntecedents(rule FuzzyRule, inputs map[VarID]float64, variables map[VarID]*LinguisticVar, tnorm func(a, b TruthValue) TruthValue) (TruthValue, error) {
	if len(rule.Antecedents) == 0 {
		return rule.Weight, nil // Unconditional rule fires with its weight
	}

	firingStrength := TruthValue(1.0)
	for i, cond := range rule.Antecedents {
		lv, ok := variables[cond.Variable]
		if !ok {
			return 0, fmt.Errorf("variable %d not found", cond.Variable)
		}
		term := lv.GetTerm(cond.Term)
		if term == nil {
			return 0, fmt.Errorf("term %d not found for variable %d", cond.Term, cond.Variable)
		}

		val, hasInput := inputs[cond.Variable]
		if !hasInput {
			return 0, fmt.Errorf("missing input for variable %d", cond.Variable)
		}

		membership := term.Membership(val)
		if cond.Negated {
			membership = StandardNegation(membership)
		}

		if i == 0 {
			firingStrength = membership
		} else {
			firingStrength = tnorm(firingStrength, membership)
		}
	}

	// Apply rule weight
	return ProductTNorm(firingStrength, rule.Weight), nil
}

// --- Mamdani Engine ---

// MamdaniEngine implements a Mamdani fuzzy inference system.
type MamdaniEngine struct {
	rules     []FuzzyRule
	variables map[VarID]*LinguisticVar
	tnorm     func(a, b TruthValue) TruthValue // AND
	tconorm   func(a, b TruthValue) TruthValue // OR (Aggregation)
	implic    func(a, b TruthValue) TruthValue // Implication
}

// NewMamdaniEngine creates a new Mamdani Engine.
func NewMamdaniEngine(pool *memory.Pool) *MamdaniEngine {
	return &MamdaniEngine{
		variables: make(map[VarID]*LinguisticVar),
		tnorm:     MinTNorm,
		tconorm:   MaxTConorm,
		implic:    MinTNorm, // standard Mamdani uses Min for implication
	}
}

// AddRule adds a Mamdani rule.
func (e *MamdaniEngine) AddRule(rule FuzzyRule) {
	e.rules = append(e.rules, rule)
}

// AddVariable registers a linguistic variable.
func (e *MamdaniEngine) AddVariable(v *LinguisticVar) {
	e.variables[v.ID] = v
}

// GetVariable returns a registered variable by ID.
func (e *MamdaniEngine) GetVariable(id VarID) *LinguisticVar {
	return e.variables[id]
}

// SetTNorm sets the AND operator.
func (e *MamdaniEngine) SetTNorm(fn func(a, b TruthValue) TruthValue) {
	e.tnorm = fn
}

// SetTConorm sets the OR operator (aggregation).
func (e *MamdaniEngine) SetTConorm(fn func(a, b TruthValue) TruthValue) {
	e.tconorm = fn
}

// SetImplication sets the implication operator.
func (e *MamdaniEngine) SetImplication(fn func(a, b TruthValue) TruthValue) {
	e.implic = fn
}

// Evaluate performs fuzzification, rule evaluation, and aggregation.
// Returns an aggregated FuzzySet containing the output.
func (e *MamdaniEngine) Evaluate(inputs map[VarID]float64, pool *memory.Pool) (*FuzzySet, error) {
	var aggregated *FuzzySet

	for _, rule := range e.rules {
		firingStrength, err := evaluateAntecedents(rule, inputs, e.variables, e.tnorm)
		if err != nil {
			return nil, err
		}

		if firingStrength == 0 {
			continue // Skip non-firing rules
		}

		// Implication
		outVar, ok := e.variables[rule.Consequent.Variable]
		if !ok {
			return nil, fmt.Errorf("output var %d not found", rule.Consequent.Variable)
		}
		outTerm := outVar.GetTerm(rule.Consequent.Term)
		if outTerm == nil {
			return nil, fmt.Errorf("output term %d not found", rule.Consequent.Term)
		}

		// Apply implication to create the rule's output set
		ruleOutput := NewFuzzySet(outTerm.ID, outTerm.Universe, pool)
		length := len(outTerm.Universe)
		for i := 0; i < length; i++ {
			ruleOutput.Members[i] = e.implic(firingStrength, outTerm.Members[i])
		}

		// Aggregation
		if aggregated == nil {
			aggregated = ruleOutput
		} else {
			aggregated = e.aggregateOutputs(aggregated, ruleOutput, pool)
		}
	}

	if aggregated == nil {
		return nil, core.NewLogicError("Mamdani", "Evaluate", "no rules fired")
	}

	return aggregated, nil
}

// aggregateOutputs combines two sets using the configured t-conorm.
func (e *MamdaniEngine) aggregateOutputs(a, b *FuzzySet, pool *memory.Pool) *FuzzySet {
	length := len(a.Universe)
	res := NewFuzzySet(0, a.Universe, pool)
	for i := 0; i < length; i++ {
		res.Members[i] = e.tconorm(a.Members[i], b.Members[i])
	}
	return res
}

// --- TSK Engine ---

// TSKConsequent evaluates a Sugeno consequent function.
type TSKConsequent interface {
	Eval(inputs map[VarID]float64) float64
}

// ConstantTSK is a zero-order Sugeno consequent.
type ConstantTSK float64

// Eval returns the constant value.
func (c ConstantTSK) Eval(inputs map[VarID]float64) float64 {
	return float64(c)
}

// LinearTSK is a first-order Sugeno consequent.
type LinearTSK struct {
	Coeffs    map[VarID]float64
	Intercept float64
}

// Eval returns the linear combination of inputs.
func (l LinearTSK) Eval(inputs map[VarID]float64) float64 {
	sum := l.Intercept
	for id, coeff := range l.Coeffs {
		if val, ok := inputs[id]; ok {
			sum += coeff * val
		}
	}
	return sum
}

// TSKRule represents a TSK fuzzy rule.
type TSKRule struct {
	Antecedents []FuzzyCondition
	Consequent  TSKConsequent
	Weight      TruthValue
}

// TSKEngine implements a Takagi-Sugeno-Kang fuzzy inference system.
type TSKEngine struct {
	rules     []TSKRule
	variables map[VarID]*LinguisticVar
	tnorm     func(a, b TruthValue) TruthValue // AND
}

// NewTSKEngine creates a new TSK Engine.
func NewTSKEngine() *TSKEngine {
	return &TSKEngine{
		variables: make(map[VarID]*LinguisticVar),
		tnorm:     MinTNorm,
	}
}

// AddRule adds a TSK rule.
func (e *TSKEngine) AddRule(rule TSKRule) {
	e.rules = append(e.rules, rule)
}

// AddVariable registers a linguistic variable.
func (e *TSKEngine) AddVariable(v *LinguisticVar) {
	e.variables[v.ID] = v
}

// GetVariable returns a registered variable by ID.
func (e *TSKEngine) GetVariable(id VarID) *LinguisticVar {
	return e.variables[id]
}

// Rules returns the slice of TSK rules.
func (e *TSKEngine) Rules() []TSKRule {
	return e.rules
}

// ReplaceConsequent updates the consequent of a specific rule.
func (e *TSKEngine) ReplaceConsequent(ruleIdx int, consequent TSKConsequent) {
	if ruleIdx >= 0 && ruleIdx < len(e.rules) {
		e.rules[ruleIdx].Consequent = consequent
	}
}

// SetTNorm sets the AND operator.
func (e *TSKEngine) SetTNorm(fn func(a, b TruthValue) TruthValue) {
	e.tnorm = fn
}

// Evaluate computes the weighted average of the rule consequents.
func (e *TSKEngine) Evaluate(inputs map[VarID]float64) (float64, error) {
	var num, den float64

	for _, rule := range e.rules {
		// Convert TSKRule to FuzzyRule to share antecedent evaluation
		var fr FuzzyRule
		fr.Antecedents = rule.Antecedents
		fr.Weight = rule.Weight
		if fr.Weight == 0 {
			fr.Weight = 1.0 // default
		}

		firingStrength, err := evaluateAntecedents(fr, inputs, e.variables, e.tnorm)
		if err != nil {
			return 0, err
		}

		if firingStrength > 0 {
			w := float64(firingStrength)
			z := rule.Consequent.Eval(inputs)
			num += w * z
			den += w
		}
	}

	if den == 0 {
		return 0, core.NewLogicError("TSK", "Evaluate", "no rules fired")
	}

	return num / den, nil
}
