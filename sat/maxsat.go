package sat

import "fmt"

// MAXSATSolverImpl implements MAX-SAT solving
type MAXSATSolverImpl struct {
	baseSolver Solver
}

// NewMAXSATSolver creates new MAX-SAT solver
func NewMAXSATSolver() *MAXSATSolverImpl {
	return &MAXSATSolverImpl{
		baseSolver: NewCDCLSolver(),
	}
}

// SolveMAXSAT finds assignment satisfying maximum clauses
func (m *MAXSATSolverImpl) SolveMAXSAT(cnf *CNF, weights []float64) *MAXSATResult {
	if len(weights) != len(cnf.Clauses) {
		weights = make([]float64, len(cnf.Clauses))
		for i := range weights {
			weights[i] = 1.0 // Unit weights
		}
	}

	return m.SolveWeightedMAXSAT(cnf, weights)
}

// SolveWeightedMAXSAT solves weighted MAX-SAT using binary search
func (m *MAXSATSolverImpl) SolveWeightedMAXSAT(cnf *CNF, weights []float64) *MAXSATResult {
	totalWeight := 0.0
	for _, w := range weights {
		totalWeight += w
	}

	// Binary search on weight threshold
	low := 0.0
	high := totalWeight
	bestAssignment := make(Assignment)
	bestWeight := 0.0
	bestUnsatisfied := make([]int, 0)

	for high-low > 0.01 { // Precision threshold
		mid := (low + high) / 2

		// Create SAT instance for this threshold
		testCNF := NewCNF()

		// Add original clauses with relaxation variables
		relaxVars := make([]string, len(cnf.Clauses))
		for i, clause := range cnf.Clauses {
			if weights[i] >= mid {
				// Hard clause - must be satisfied
				testCNF.AddClause(clause)
			} else {
				// Soft clause - add relaxation variable
				relaxVar := fmt.Sprintf("__relax_%d", i)
				relaxVars[i] = relaxVar

				// Add clause with relaxation: (original_clause ∨ relaxVar)
				relaxedLiterals := make([]Literal, len(clause.Literals)+1)
				copy(relaxedLiterals, clause.Literals)
				relaxedLiterals[len(clause.Literals)] = Literal{
					Variable: relaxVar,
					Negated:  false,
				}

				testCNF.AddClause(NewClause(relaxedLiterals...))
			}
		}

		// Try to solve
		result := m.baseSolver.Solve(testCNF)

		if result.Satisfiable {
			// Can achieve this weight - try higher
			bestAssignment = result.Assignment
			bestWeight = mid

			// Calculate actual satisfied weight
			actualWeight := 0.0
			unsatisfied := make([]int, 0)

			for i, clause := range cnf.Clauses {
				if result.Assignment.Satisfies(clause) {
					actualWeight += weights[i]
				} else {
					unsatisfied = append(unsatisfied, clause.ID)
				}
			}

			bestWeight = actualWeight
			bestUnsatisfied = unsatisfied
			low = mid
		} else {
			// Cannot achieve this weight - try lower
			high = mid
		}
	}

	return &MAXSATResult{
		Assignment:         bestAssignment,
		SatisfiedCount:     len(cnf.Clauses) - len(bestUnsatisfied),
		TotalWeight:        bestWeight,
		UnsatisfiedClauses: bestUnsatisfied,
		Statistics:         m.baseSolver.GetStatistics(),
	}
}
