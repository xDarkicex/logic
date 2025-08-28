package sat

import (
	"fmt"

	"github.com/xDarkicex/logic/classical"
	"github.com/xDarkicex/logic/core"
)

// SATSystemImpl implements SATSystem interface
type SATSystemImpl struct {
	solver    Solver
	converter *CNFConverter
}

// NewSATSystem creates a new SAT system with CDCL solver
func NewSATSystem() *SATSystemImpl {
	return &SATSystemImpl{
		solver:    NewCDCLSolver(),
		converter: NewCNFConverter(),
	}
}

// NewSATSystemWithSolver creates SAT system with custom solver
func NewSATSystemWithSolver(solver Solver) *SATSystemImpl {
	return &SATSystemImpl{
		solver:    solver,
		converter: NewCNFConverter(),
	}
}

// Name returns system name
func (s *SATSystemImpl) Name() string {
	return "sat"
}

// Evaluate converts expression to CNF and solves
func (s *SATSystemImpl) Evaluate(expr string, ctx core.EvaluationContext) (interface{}, error) {
	// Convert expression to CNF
	cnf, err := s.converter.ConvertExpression(expr)
	if err != nil {
		return nil, core.NewLogicError("sat", "SATSystem.Evaluate",
			fmt.Sprintf("CNF conversion failed: %v", err))
	}

	// If context has constraints, add them
	if len(ctx.Variables()) > 0 {
		for _, variable := range ctx.Variables() {
			if value, exists := ctx.Get(variable); exists {
				if boolValue, ok := value.(bool); ok {
					// Add unit clause for this variable
					literal := Literal{Variable: variable, Negated: !boolValue}
					cnf.AddClause(NewClause(literal))
				}
			}
		}
	}

	// Solve
	result := s.solver.Solve(cnf)
	if result.Error != nil {
		return nil, result.Error
	}

	// Return structured result
	return map[string]interface{}{
		"satisfiable": result.Satisfiable,
		"assignment":  result.Assignment,
		"statistics":  result.Statistics,
	}, nil
}

// Validate checks if expression can be converted to CNF
func (s *SATSystemImpl) Validate(expr string) error {
	_, err := s.converter.ConvertExpression(expr)
	return err
}

// SupportedOperators returns supported logical operators
func (s *SATSystemImpl) SupportedOperators() []string {
	return []string{"&", "∧", "|", "∨", "^", "⊕", "!", "¬", "->", "→", "<->", "↔",
		"and", "or", "xor", "not", "nand", "nor", "implies", "iff"}
}

// Solve solves CNF directly
func (s *SATSystemImpl) Solve(cnf *CNF) *SolverResult {
	return s.solver.Solve(cnf)
}

// ConvertToCNF converts logical expression to CNF
func (s *SATSystemImpl) ConvertToCNF(expr string) (*CNF, error) {
	return s.converter.ConvertExpression(expr)
}

// VerifySolution verifies assignment satisfies original expression
func (s *SATSystemImpl) VerifySolution(expr string, assignment Assignment) (bool, error) {
	// Parse expression
	ast, err := classical.ParseExpression(expr)
	if err != nil {
		return false, err
	}

	// Convert assignment to evaluation context
	ctx := make(classical.EvaluationContext)
	for variable, value := range assignment {
		ctx[variable] = value
	}

	// Evaluate expression
	result, err := ast.Evaluate(ctx)
	return result, err
}
