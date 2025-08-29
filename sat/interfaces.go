package sat

import (
	"time"

	"github.com/xDarkicex/logic/core"
)

// Solver represents a SAT solver interface
type Solver interface {
	// Solve attempts to find a satisfying assignment
	Solve(cnf *CNF) *SolverResult
	// SolveWithTimeout solves with a time limit
	SolveWithTimeout(cnf *CNF, timeout time.Duration) *SolverResult
	// AddClause adds a clause during incremental solving
	AddClause(clause *Clause) error
	// GetStatistics returns solver performance metrics
	GetStatistics() SolverStatistics
	// Reset clears solver state for reuse
	Reset()
	// Name returns solver implementation name
	Name() string
}

// XORSolver extends Solver interface for XOR constraint support
type XORSolver interface {
	Solver

	// SolveExtended solves Extended CNF with XOR constraints
	SolveExtended(ecnf *ExtendedCNF) *SolverResult

	// SolveExtendedWithTimeout solves with timeout
	SolveExtendedWithTimeout(ecnf *ExtendedCNF, timeout time.Duration) *SolverResult

	// EnableXORSupport enables/disables XOR constraint handling
	EnableXORSupport(enabled bool)

	// GetXORStatistics returns XOR-specific statistics
	GetXORStatistics() map[string]int64
}

// Heuristic represents variable decision heuristics
type Heuristic interface {
	// ChooseVariable selects next decision variable
	ChooseVariable(unassigned []string, assignment Assignment) string
	// Update is called after conflicts to update heuristic state
	Update(conflictClause *Clause)
	// Reset clears heuristic state
	Reset()
	// Name returns heuristic name
	Name() string
}

// RestartStrategy determines when to restart search
type RestartStrategy interface {
	// ShouldRestart returns true if solver should restart
	ShouldRestart(stats SolverStatistics) bool
	// OnRestart is called when restart occurs
	OnRestart()
	// Reset clears strategy state
	Reset()
	// Name returns strategy name
	Name() string
}

// ClauseDeletionPolicy determines which learned clauses to delete
type ClauseDeletionPolicy interface {
	// ShouldDelete returns true if clause should be deleted
	ShouldDelete(clause *Clause, stats SolverStatistics) bool
	// Update is called to update clause activities
	Update(clauses []*Clause)
	// Reset clears policy state
	Reset()
	// Name returns policy name
	Name() string
}

// ConflictAnalyzer analyzes conflicts and generates learned clauses
type ConflictAnalyzer interface {
	// Analyze analyzes conflict and returns learned clause and backtrack level
	Analyze(conflictClause *Clause, trail DecisionTrail) (*Clause, int)
	// Reset clears analyzer state
	Reset()
	// Name returns analyzer name
	Name() string
}

// Preprocessor simplifies CNF before solving
type Preprocessor interface {
	// Preprocess simplifies CNF formula
	Preprocess(cnf *CNF) (*CNF, error)
	// PostProcess converts solution back to original variables
	PostProcess(assignment Assignment) Assignment
	// Name returns preprocessor name
	Name() string
}

// SATSystem integrates SAT solver with the logic engine
type SATSystem interface {
	core.LogicSystem
	// Solve solves CNF formula
	Solve(cnf *CNF) *SolverResult
	// ConvertToCNF converts logical expression to CNF
	ConvertToCNF(expr string) (*CNF, error)
	// VerifySolution verifies assignment satisfies original formula
	VerifySolution(expr string, assignment Assignment) (bool, error)
}

// MAXSATSolver extends SAT solver for Maximum Satisfiability
type MAXSATSolver interface {
	// SolveMAXSAT finds assignment satisfying maximum number of clauses
	SolveMAXSAT(cnf *CNF, weights []float64) *MAXSATResult
	// SolveWeightedMAXSAT solves weighted MAX-SAT
	SolveWeightedMAXSAT(cnf *CNF, weights []float64) *MAXSATResult
}

// MAXSATResult represents MAX-SAT solving result
type MAXSATResult struct {
	Assignment         Assignment
	SatisfiedCount     int
	TotalWeight        float64
	UnsatisfiedClauses []int // IDs of unsatisfied clauses
	Statistics         SolverStatistics
	Error              error
}

// DecisionTrail tracks variable assignments and their reasons
type DecisionTrail interface {
	// Assign adds a variable assignment
	Assign(variable string, value bool, level int, reason *Clause)
	// Backtrack undoes assignments back to given level
	Backtrack(level int) []string // Returns unassigned variables
	// GetLevel returns decision level of variable
	GetLevel(variable string) int
	// GetReason returns reason clause for variable assignment
	GetReason(variable string) *Clause
	// GetAssignment returns current assignment
	GetAssignment() Assignment
	// GetCurrentLevel returns current decision level
	GetCurrentLevel() int
	// Clear resets trail
	Clear()
}

// Inprocessor performs continuous simplification during search
type Inprocessor interface {
	// Main inprocessing entry point
	Inprocess(cnf *CNF, assignment Assignment, level int) (*InprocessResult, error)
	// Individual technique interfaces
	VivifyClauses(clauses []*Clause, assignment Assignment) int
	EliminateVariables(variables []string, cnf *CNF) int
	ProbeFailedLiterals(literals []Literal, cnf *CNF) []Literal
	SubsumeAndStrengthen(cnf *CNF) int
	// Configuration and lifecycle
	Configure(config InprocessConfig)
	GetStatistics() InprocessStatistics
	Reset()
	Name() string
}

// Ensure CDCLSolver implements XORSolver

func (c *CDCLSolver) EnableXORSupport(enabled bool) {
	c.xorEnabled = enabled
}

func (c *CDCLSolver) GetXORStatistics() map[string]int64 {
	stats := map[string]int64{
		"xorPropagations": c.xorPropagations,
		"xorConflicts":    c.xorConflicts,
		"gaussianRuns":    c.gaussianRuns,
	}

	gaussianStats := c.gaussianEliminator.GetStatistics()
	stats["gaussianTotalRuns"] = gaussianStats.TotalRuns
	stats["gaussianVariablesEliminated"] = gaussianStats.VariablesEliminated
	stats["gaussianXORsLearned"] = gaussianStats.XORClausesLearned
	stats["gaussianUnitProps"] = gaussianStats.UnitPropagations

	return stats
}
