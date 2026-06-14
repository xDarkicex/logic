package fuzzy

import (
	"fmt"
	"hash/fnv"

	"github.com/xDarkicex/logic/core"
	"github.com/xDarkicex/memory"
)

// FuzzyLogicSystem implements core.LogicSystem for fuzzy inference.
// It wraps a MamdaniEngine and translates string expressions into condition evaluations.
type FuzzyLogicSystem struct {
	engine *MamdaniEngine
	pool   *memory.Pool
	arena  *memory.Arena
	sym    *SymbolTable
}

// NewFuzzyLogicSystem creates a new FuzzyLogicSystem.
func NewFuzzyLogicSystem(engine *MamdaniEngine, sym *SymbolTable, pool *memory.Pool, arena *memory.Arena) *FuzzyLogicSystem {
	return &FuzzyLogicSystem{
		engine: engine,
		pool:   pool,
		arena:  arena,
		sym:    sym,
	}
}

// Name returns the identifier of the logic system.
func (s *FuzzyLogicSystem) Name() string {
	return "fuzzy"
}

// Evaluate runs the inference engine or evaluates a specific condition.
// Since core.LogicSystem expects a boolean, we return true if the 
// truth value >= 0.5.
func (s *FuzzyLogicSystem) Evaluate(expression string, ctx core.EvaluationContext) (bool, error) {
	// Parse the expression as a condition
	lexer := NewLexer(expression, s.pool)
	tokens := lexer.Lex()
	parser := NewParser(tokens, s.sym, s.arena)

	conds, err := parser.ParseCondition()
	if err != nil {
		return false, fmt.Errorf("failed to parse fuzzy expression: %v", err)
	}

	// Prepare inputs from context
	inputs := make(map[VarID]float64)
	for _, vName := range ctx.Variables() {
		h := fnv.New32a()
		h.Write([]byte(vName))
		hash := h.Sum32()
		vID, exists := s.sym.Lookup(vName, hash)
		if exists {
			val, _ := ctx.Get(vName)
			if fval, ok := val.(float64); ok {
				inputs[vID] = fval
			}
		}
	}

	// Evaluate the condition
	if len(conds) == 0 {
		return false, fmt.Errorf("empty condition")
	}

	truth := TruthValue(1.0)
	for i, cond := range conds {
		lv := s.engine.GetVariable(cond.Variable)
		if lv == nil {
			return false, fmt.Errorf("variable %s not registered in engine", s.sym.Name(cond.Variable))
		}
		term := lv.GetTerm(cond.Term)
		if term == nil {
			return false, fmt.Errorf("term %s not found for variable %s", s.sym.Name(cond.Term), s.sym.Name(cond.Variable))
		}

		inVal, hasIn := inputs[cond.Variable]
		if !hasIn {
			return false, fmt.Errorf("missing input for variable %s", s.sym.Name(cond.Variable))
		}

		m := term.Membership(inVal)
		if cond.Negated {
			m = StandardNegation(m)
		}

		if i == 0 {
			truth = m
		} else {
			truth = s.engine.tnorm(truth, m)
		}
	}

	return truth >= 0.5, nil
}

// Validate checks if the expression syntax is valid.
func (s *FuzzyLogicSystem) Validate(expression string) error {
	lexer := NewLexer(expression, s.pool)
	tokens := lexer.Lex()
	parser := NewParser(tokens, s.sym, s.arena)
	_, err := parser.ParseCondition()
	return err
}

// SupportedOperators returns operators this system can parse.
func (s *FuzzyLogicSystem) SupportedOperators() []string {
	return []string{"AND", "IS", "NOT"}
}
