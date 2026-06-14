package fuzzy

import (
	"fmt"

	"github.com/xDarkicex/logic/core"
	"github.com/xDarkicex/memory"
)

// --- Variable lookup helpers (Pool-backed, no maps) ---

// findVariable returns the LinguisticVar with the given ID via linear scan.
// O(n) but n is always small (< 20 variables in practice).
func findVariable(vars []*LinguisticVar, id VarID) *LinguisticVar {
	for _, v := range vars {
		if v != nil && v.ID == id {
			return v
		}
	}
	return nil
}

// --- Antecedent Evaluation ---

// evaluateAndGroup evaluates a group of conditions ANDed together.
func evaluateAndGroup(conditions []FuzzyCondition, inputs map[VarID]float64, vars []*LinguisticVar, tnorm func(a, b TruthValue) TruthValue) (TruthValue, error) {
	if len(conditions) == 0 {
		return 1.0, nil
	}
	result := TruthValue(1.0)
	for _, cond := range conditions {
		membership, err := evaluateCondition(cond, inputs, vars)
		if err != nil {
			return 0, err
		}
		result = tnorm(result, membership)
	}
	return result, nil
}

// evaluateOrGroups evaluates OR-of-ANDs groups using the t-conorm for OR.
func evaluateOrGroups(groups [][]FuzzyCondition, inputs map[VarID]float64, vars []*LinguisticVar, tnorm, tconorm func(a, b TruthValue) TruthValue) (TruthValue, error) {
	if len(groups) == 0 {
		return 1.0, nil
	}
	result := TruthValue(0.0)
	for _, group := range groups {
		groupResult, err := evaluateAndGroup(group, inputs, vars, tnorm)
		if err != nil {
			return 0, err
		}
		result = tconorm(result, groupResult)
	}
	return result, nil
}

// evaluateCondition evaluates a single FuzzyCondition.
func evaluateCondition(cond FuzzyCondition, inputs map[VarID]float64, vars []*LinguisticVar) (TruthValue, error) {
	lv := findVariable(vars, cond.Variable)
	if lv == nil || !lv.Enabled {
		return 0, fmt.Errorf("variable %d not found or disabled", cond.Variable)
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
	return membership, nil
}

// evaluateAntecedents evaluates all antecedents of a FuzzyRule.
func evaluateAntecedents(rule FuzzyRule, inputs map[VarID]float64, vars []*LinguisticVar, tnorm, tconorm func(a, b TruthValue) TruthValue) (TruthValue, error) {
	strength, err := evaluateAndGroup(rule.Antecedents, inputs, vars, tnorm)
	if err != nil {
		return 0, err
	}
	if strength == 0 {
		return 0, nil
	}
	if len(rule.OrGroups) > 0 {
		orResult, err := evaluateOrGroups(rule.OrGroups, inputs, vars, tnorm, tconorm)
		if err != nil {
			return 0, err
		}
		strength = tnorm(strength, orResult)
	}
	return ProductTNorm(strength, rule.Weight), nil
}

// --- Mamdani Engine ---

// MamdaniEngine implements a Mamdani fuzzy inference system with RuleBlocks and OutputVariables.
// Ported from fuzzylite's Engine class.
type MamdaniEngine struct {
	inputVars  []*LinguisticVar  // Pool-backed, linear scan for lookup
	outputVars []*OutputVariable // Pool-backed
	ruleBlocks []*RuleBlock      // Pool-backed
	pool       *memory.Pool
}

// NewMamdaniEngine creates a new Mamdani Engine.
func NewMamdaniEngine(pool *memory.Pool) *MamdaniEngine {
	vars := memory.MustPoolSlice[*LinguisticVar](pool, 16)
	vars = vars[:0]
	outputs := memory.MustPoolSlice[*OutputVariable](pool, 8)
	outputs = outputs[:0]
	blocks := memory.MustPoolSlice[*RuleBlock](pool, 4)
	blocks = blocks[:0]
	return &MamdaniEngine{
		inputVars:  vars,
		outputVars: outputs,
		ruleBlocks: blocks,
		pool:       pool,
	}
}

// AddInputVariable adds an input linguistic variable.
func (e *MamdaniEngine) AddInputVariable(v *LinguisticVar) {
	e.inputVars = append(e.inputVars, v)
}

// AddVariable is a convenience alias for AddInputVariable (backwards compat).
func (e *MamdaniEngine) AddVariable(v *LinguisticVar) {
	e.AddInputVariable(v)
}

// GetVariable finds a variable by ID across input and output variables.
func (e *MamdaniEngine) GetVariable(id VarID) *LinguisticVar {
	if v := findVariable(e.inputVars, id); v != nil {
		return v
	}
	for _, ov := range e.outputVars {
		if ov.Variable.ID == id {
			return ov.Variable
		}
	}
	return nil
}

// AddOutputVariable adds an output variable.
func (e *MamdaniEngine) AddOutputVariable(ov *OutputVariable) {
	e.outputVars = append(e.outputVars, ov)
}

// AddRuleBlock adds a rule block to the engine.
func (e *MamdaniEngine) AddRuleBlock(rb *RuleBlock) {
	e.ruleBlocks = append(e.ruleBlocks, rb)
}

// AddRule adds a rule to the first rule block, creating one if needed.
func (e *MamdaniEngine) AddRule(rule FuzzyRule) {
	if len(e.ruleBlocks) == 0 {
		rb := NewRuleBlock("default", 16, e.pool)
		e.ruleBlocks = append(e.ruleBlocks, rb)
	}
	e.ruleBlocks[0].AddRule(rule)
}

// SetTNorm sets the conjunction operator on the first rule block.
func (e *MamdaniEngine) SetTNorm(fn func(a, b TruthValue) TruthValue) {
	if len(e.ruleBlocks) > 0 {
		e.ruleBlocks[0].SetConjunction(fn)
	}
}

// GetTNorm returns the conjunction operator from the first rule block, or MinTNorm if none.
func (e *MamdaniEngine) GetTNorm() func(a, b TruthValue) TruthValue {
	if len(e.ruleBlocks) > 0 {
		return e.ruleBlocks[0].Conjunction
	}
	return MinTNorm
}

// SetTConorm sets the disjunction operator on the first rule block.
func (e *MamdaniEngine) SetTConorm(fn func(a, b TruthValue) TruthValue) {
	if len(e.ruleBlocks) > 0 {
		e.ruleBlocks[0].SetDisjunction(fn)
	}
}

// SetImplication sets the implication operator on the first rule block.
func (e *MamdaniEngine) SetImplication(fn func(a, b TruthValue) TruthValue) {
	if len(e.ruleBlocks) > 0 {
		e.ruleBlocks[0].SetImplication(fn)
	}
}

// SetActivation sets the activation method on the first rule block.
func (e *MamdaniEngine) SetActivation(a Activation) {
	if len(e.ruleBlocks) > 0 {
		e.ruleBlocks[0].SetActivation(a)
	}
}

// Evaluate runs the full Mamdani inference cycle across all rule blocks and output variables.
// If no output variables are registered, they are auto-created from rule consequents (backwards compat).
func (e *MamdaniEngine) Evaluate(inputs map[VarID]float64, pool *memory.Pool) (*FuzzySet, error) {
	if len(e.ruleBlocks) == 0 {
		return nil, core.NewLogicError("Mamdani", "Evaluate", "no rule blocks")
	}

	// Auto-create output variables from rule consequents if none registered
	if len(e.outputVars) == 0 {
		e.autoCreateOutputs()
	}

	// Clear output accumulators
	for _, ov := range e.outputVars {
		ov.clear()
	}

	// Process each rule block
	for _, rb := range e.ruleBlocks {
		if !rb.Enabled || rb.Len() == 0 {
			continue
		}
		e.processBlock(rb, inputs, pool)
	}

	return e.aggregateOutputs(pool)
}

// autoCreateOutputs scans all rule consequents and creates OutputVariables for any that aren't registered.
func (e *MamdaniEngine) autoCreateOutputs() {
	for _, rb := range e.ruleBlocks {
		for _, rule := range rb.Rules {
			id := rule.Consequent.Variable
			if e.findOutputVar(id) != nil {
				continue
			}
			lv := e.GetVariable(id)
			if lv != nil {
				e.outputVars = append(e.outputVars, NewOutputVariable(lv))
			}
		}
	}
}

// processBlock evaluates a single RuleBlock: compute strengths, activate, fire rules.
func (e *MamdaniEngine) processBlock(rb *RuleBlock, inputs map[VarID]float64, pool *memory.Pool) {
	n := rb.Len()
	strengths := memory.MustPoolSlice[TruthValue](pool, n)
	strengths = strengths[:n]

	// Compute firing strengths
	for i, rule := range rb.Rules {
		s, err := evaluateAntecedents(rule, inputs, e.inputVars, rb.Conjunction, rb.Disjunction)
		if err != nil {
			strengths[i] = 0
			continue
		}
		strengths[i] = s
	}

	// Apply activation
	rb.Activation.Activate(strengths)

	// Fire each rule
	for i, strength := range strengths {
		if strength == 0 {
			continue
		}
		e.fireRule(rb, i, strength, pool)
	}
}

// fireRule applies implication to the rule's consequent and accumulates into the output variable.
func (e *MamdaniEngine) fireRule(rb *RuleBlock, ruleIdx int, strength TruthValue, pool *memory.Pool) {
	rule := rb.Rules[ruleIdx]
	ov := e.findOutputVar(rule.Consequent.Variable)
	if ov == nil {
		return
	}
	outTerm := ov.Variable.GetTerm(rule.Consequent.Term)
	if outTerm == nil {
		return
	}

	// Implication: modulate the output term by firing strength
	implied := NewFuzzySet(outTerm.ID, outTerm.Universe, pool)
	for j := 0; j < len(outTerm.Universe); j++ {
		implied.Members[j] = rb.Implication(strength, outTerm.Members[j])
	}

	// Aggregate into output variable
	if ov.fuzzyOutput == nil {
		ov.fuzzyOutput = implied
	} else {
		ov.fuzzyOutput = e.mergeSets(ov.fuzzyOutput, implied, ov.Aggregation, pool)
	}
}

// findOutputVar finds an OutputVariable by its Variable's ID.
func (e *MamdaniEngine) findOutputVar(id VarID) *OutputVariable {
	for _, ov := range e.outputVars {
		if ov.Variable.ID == id {
			return ov
		}
	}
	return nil
}

// mergeSets combines two fuzzy sets using the given aggregation operator.
func (e *MamdaniEngine) mergeSets(a, b *FuzzySet, agg func(a, b TruthValue) TruthValue, pool *memory.Pool) *FuzzySet {
	res := NewFuzzySet(0, a.Universe, pool)
	for i := 0; i < len(a.Universe); i++ {
		res.Members[i] = agg(a.Members[i], b.Members[i])
	}
	return res
}

// aggregateOutputs defuzzifies each output variable and returns the combined result.
func (e *MamdaniEngine) aggregateOutputs(pool *memory.Pool) (*FuzzySet, error) {
	if len(e.outputVars) == 0 {
		return nil, core.NewLogicError("Mamdani", "Evaluate", "no outputs")
	}
	// For single output, return its fuzzy set directly
	if len(e.outputVars) == 1 {
		ov := e.outputVars[0]
		if ov.fuzzyOutput == nil {
			return nil, core.NewLogicError("Mamdani", "Evaluate", "no rules fired")
		}
		return ov.fuzzyOutput, nil
	}
	// Multiple outputs: merge
	var merged *FuzzySet
	for _, ov := range e.outputVars {
		if ov.fuzzyOutput == nil {
			continue
		}
		if merged == nil {
			merged = ov.fuzzyOutput
		} else {
			merged = e.mergeSets(merged, ov.fuzzyOutput, ov.Aggregation, pool)
		}
	}
	if merged == nil {
		return nil, core.NewLogicError("Mamdani", "Evaluate", "no rules fired")
	}
	return merged, nil
}

// --- TSK Engine ---

// TSKConsequent evaluates a Sugeno consequent function.
type TSKConsequent interface {
	Eval(inputs map[VarID]float64) float64
}

// ConstantTSK is a zero-order Sugeno consequent.
type ConstantTSK float64

func (c ConstantTSK) Eval(inputs map[VarID]float64) float64 { return float64(c) }

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
	OrGroups    [][]FuzzyCondition
	Consequent  TSKConsequent
	Weight      TruthValue
}

// TSKEngine implements a Takagi-Sugeno-Kang fuzzy inference system.
type TSKEngine struct {
	variables  []*LinguisticVar // Pool-backed
	rules      []TSKRule
	tnorm      func(a, b TruthValue) TruthValue
	tconorm    func(a, b TruthValue) TruthValue
	activation Activation
}

// NewTSKEngine creates a new TSK Engine.
func NewTSKEngine() *TSKEngine {
	return &TSKEngine{
		tnorm:      MinTNorm,
		tconorm:    MaxTConorm,
		activation: NewGeneralActivation(),
	}
}

// AddRule adds a TSK rule.
func (e *TSKEngine) AddRule(rule TSKRule) {
	e.rules = append(e.rules, rule)
}

// AddVariable registers a linguistic variable.
func (e *TSKEngine) AddVariable(v *LinguisticVar) {
	e.variables = append(e.variables, v)
}

// GetVariable finds a variable by ID.
func (e *TSKEngine) GetVariable(id VarID) *LinguisticVar {
	return findVariable(e.variables, id)
}

// Rules returns the slice of TSK rules.
func (e *TSKEngine) Rules() []TSKRule { return e.rules }

// ReplaceConsequent updates the consequent of a specific rule.
func (e *TSKEngine) ReplaceConsequent(ruleIdx int, consequent TSKConsequent) {
	if ruleIdx >= 0 && ruleIdx < len(e.rules) {
		e.rules[ruleIdx].Consequent = consequent
	}
}

// SetTNorm sets the AND operator.
func (e *TSKEngine) SetTNorm(fn func(a, b TruthValue) TruthValue) { e.tnorm = fn }

// SetTConorm sets the OR operator for disjunctive antecedents.
func (e *TSKEngine) SetTConorm(fn func(a, b TruthValue) TruthValue) { e.tconorm = fn }

// SetActivation sets the rule activation method.
func (e *TSKEngine) SetActivation(a Activation) {
	if a != nil {
		e.activation = a
	}
}

// evaluateTSKAntecedents evaluates antecedents for a TSK rule.
func evaluateTSKAntecedents(rule TSKRule, inputs map[VarID]float64, vars []*LinguisticVar, tnorm, tconorm func(a, b TruthValue) TruthValue) (TruthValue, error) {
	strength, err := evaluateAndGroup(rule.Antecedents, inputs, vars, tnorm)
	if err != nil {
		return 0, err
	}
	if strength == 0 {
		return 0, nil
	}
	if len(rule.OrGroups) > 0 {
		orResult, err := evaluateOrGroups(rule.OrGroups, inputs, vars, tnorm, tconorm)
		if err != nil {
			return 0, err
		}
		strength = tnorm(strength, orResult)
	}
	return ProductTNorm(strength, rule.Weight), nil
}

// Evaluate computes the weighted average of rule consequents.
func (e *TSKEngine) Evaluate(inputs map[VarID]float64) (float64, error) {
	n := len(e.rules)
	if n == 0 {
		return 0, core.NewLogicError("TSK", "Evaluate", "no rules")
	}

	var num, den float64
	for _, rule := range e.rules {
		strength, err := evaluateTSKAntecedents(rule, inputs, e.variables, e.tnorm, e.tconorm)
		if err != nil {
			return 0, err
		}
		if strength > 0 {
			w := float64(strength)
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
