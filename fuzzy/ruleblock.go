package fuzzy

import "github.com/xDarkicex/memory"

// RuleBlock groups a set of rules with shared conjunction, disjunction, implication,
// and activation operators. This matches fuzzylite's RuleBlock class.
// A single engine can hold multiple RuleBlocks, each with different operator configurations.
type RuleBlock struct {
	Name        string
	Enabled     bool
	Rules       []FuzzyRule // Pool-backed via block pool
	Conjunction func(a, b TruthValue) TruthValue
	Disjunction func(a, b TruthValue) TruthValue
	Implication func(a, b TruthValue) TruthValue
	Activation  Activation
}

// NewRuleBlock creates a RuleBlock with sensible defaults (Mamdani-style).
// The rules slice is allocated from the given pool.
func NewRuleBlock(name string, capacity int, pool *memory.Pool) *RuleBlock {
	rules := memory.MustPoolSlice[FuzzyRule](pool, capacity)
	rules = rules[:0]
	return &RuleBlock{
		Name:        name,
		Enabled:     true,
		Rules:       rules,
		Conjunction: MinTNorm,
		Disjunction: MaxTConorm,
		Implication: MinTNorm,
		Activation:  NewGeneralActivation(),
	}
}

// AddRule appends a rule to the block. Uses Go append on the Pool-backed slice.
func (rb *RuleBlock) AddRule(rule FuzzyRule) {
	rb.Rules = append(rb.Rules, rule)
}

// SetConjunction sets the conjunction (AND) operator.
func (rb *RuleBlock) SetConjunction(fn func(a, b TruthValue) TruthValue) { rb.Conjunction = fn }

// SetDisjunction sets the disjunction (OR) operator for antecedents.
func (rb *RuleBlock) SetDisjunction(fn func(a, b TruthValue) TruthValue) { rb.Disjunction = fn }

// SetImplication sets the implication operator.
func (rb *RuleBlock) SetImplication(fn func(a, b TruthValue) TruthValue) { rb.Implication = fn }

// SetActivation sets the rule activation method.
func (rb *RuleBlock) SetActivation(a Activation) {
	if a != nil {
		rb.Activation = a
	}
}

// Len returns the number of rules.
func (rb *RuleBlock) Len() int { return len(rb.Rules) }
