package sat

import (
	"time"

	"github.com/xDarkicex/logic/core"
)

// DPLLSolver implements the Davis-Putnam-Logemann-Loveland algorithm
type DPLLSolver struct {
	statistics SolverStatistics
	assignment Assignment
	cnf        *CNF
	startTime  time.Time
}

// NewDPLLSolver creates a new DPLL solver
func NewDPLLSolver() *DPLLSolver {
	return &DPLLSolver{
		statistics: SolverStatistics{},
		assignment: make(Assignment),
	}
}

// Name returns solver name
func (d *DPLLSolver) Name() string {
	return "DPLL"
}

// Solve solves the SAT problem using DPLL algorithm
func (d *DPLLSolver) Solve(cnf *CNF) *SolverResult {
	return d.SolveWithTimeout(cnf, 0) // No timeout
}

// SolveWithTimeout solves with timeout
func (d *DPLLSolver) SolveWithTimeout(cnf *CNF, timeout time.Duration) *SolverResult {
	d.startTime = time.Now()
	d.cnf = cnf
	d.assignment = make(Assignment)
	d.statistics = SolverStatistics{}

	// Set up timeout channel if needed
	var timeoutChan <-chan time.Time
	if timeout > 0 {
		timeoutChan = time.After(timeout)
	}

	result := &SolverResult{}

	// Start DPLL algorithm
	satisfiable, err := d.dpll(timeoutChan)
	if err != nil {
		result.Error = err
		return result
	}

	result.Satisfiable = satisfiable
	if satisfiable {
		result.Assignment = d.assignment.Clone()
	}

	d.statistics.TimeElapsed = time.Since(d.startTime).Nanoseconds()
	result.Statistics = d.statistics

	return result
}

// dpll implements the core DPLL algorithm
func (d *DPLLSolver) dpll(timeoutChan <-chan time.Time) (bool, error) {
	// Check for timeout
	select {
	case <-timeoutChan:
		return false, core.NewLogicError("sat", "DPLLSolver.dpll", "timeout exceeded")
	default:
	}

	// Step 1: Unit Propagation
	conflict, err := d.unitPropagation()
	if err != nil {
		return false, err
	}
	if conflict {
		return false, nil // Contradiction found
	}

	// Step 2: Pure Literal Elimination
	err = d.pureLiteralElimination()
	if err != nil {
		return false, err
	}

	// Step 3: Check if all clauses are satisfied
	if d.allClausesSatisfied() {
		return true, nil
	}

	// Step 4: Choose decision variable
	decisionVar := d.chooseDecisionVariable()
	if decisionVar == "" {
		// No unassigned variables but not all clauses satisfied
		return false, nil
	}

	d.statistics.Decisions++

	// Step 5: Try both assignments (backtracking search)
	for _, value := range []bool{true, false} {
		// Save current state
		savedAssignment := d.assignment.Clone()

		// Make assignment
		d.assignment[decisionVar] = value

		// Recursive call
		result, err := d.dpll(timeoutChan)
		if err != nil {
			return false, err
		}

		if result {
			return true, nil // Found satisfying assignment
		}

		// Backtrack
		d.assignment = savedAssignment
	}

	return false, nil
}

// unitPropagation propagates unit clauses
func (d *DPLLSolver) unitPropagation() (bool, error) {
	changed := true

	for changed {
		changed = false

		for _, clause := range d.cnf.Clauses {
			// Skip satisfied clauses
			if d.assignment.Satisfies(clause) {
				continue
			}

			// Check for conflict
			if d.assignment.ConflictsWith(clause) {
				return true, nil // Conflict found
			}

			// Check for unit clause
			unassignedLiterals := d.getUnassignedLiterals(clause)
			if len(unassignedLiterals) == 1 {
				// Unit clause found - propagate
				lit := unassignedLiterals[0]
				value := !lit.Negated // If literal is positive, assign true; if negative, assign false

				d.assignment[lit.Variable] = value
				d.statistics.Propagations++
				changed = true
			}
		}
	}

	return false, nil
}

// pureLiteralElimination eliminates pure literals
func (d *DPLLSolver) pureLiteralElimination() error {
	literalCount := make(map[string]int) // Count of positive occurrences
	variablesSeen := make(map[string]bool)

	// Count literal occurrences in unresolved clauses
	for _, clause := range d.cnf.Clauses {
		if d.assignment.Satisfies(clause) {
			continue // Skip satisfied clauses
		}

		for _, lit := range clause.Literals {
			if d.assignment.IsAssigned(lit.Variable) {
				continue // Skip assigned variables
			}

			variablesSeen[lit.Variable] = true
			if !lit.Negated {
				literalCount[lit.Variable]++
			} else {
				literalCount[lit.Variable]--
			}
		}
	}

	// Assign pure literals
	for variable := range variablesSeen {
		if !d.assignment.IsAssigned(variable) {
			count := literalCount[variable]
			if count > 0 {
				// Only positive occurrences - assign true
				d.assignment[variable] = true
			} else if count < 0 {
				// Only negative occurrences - assign false
				d.assignment[variable] = false
			}
			// If count == 0, variable appears both positively and negatively
		}
	}

	return nil
}

// allClausesSatisfied checks if all clauses are satisfied
func (d *DPLLSolver) allClausesSatisfied() bool {
	for _, clause := range d.cnf.Clauses {
		if !d.assignment.Satisfies(clause) {
			return false
		}
	}
	return true
}

// chooseDecisionVariable chooses next variable to assign (simple heuristic)
func (d *DPLLSolver) chooseDecisionVariable() string {
	// Simple heuristic: choose first unassigned variable
	// This will be replaced with VSIDS in CDCL
	for _, variable := range d.cnf.Variables {
		if !d.assignment.IsAssigned(variable) {
			return variable
		}
	}
	return ""
}

// getUnassignedLiterals returns literals in clause that are not yet assigned
func (d *DPLLSolver) getUnassignedLiterals(clause *Clause) []Literal {
	var unassigned []Literal

	for _, lit := range clause.Literals {
		if !d.assignment.IsAssigned(lit.Variable) {
			unassigned = append(unassigned, lit)
		}
	}

	return unassigned
}

// AddClause is not supported in basic DPLL
func (d *DPLLSolver) AddClause(clause *Clause) error {
	return core.NewLogicError("sat", "DPLLSolver.AddClause",
		"incremental solving not supported in DPLL")
}

// GetStatistics returns solver statistics
func (d *DPLLSolver) GetStatistics() SolverStatistics {
	return d.statistics
}

// Reset clears solver state
func (d *DPLLSolver) Reset() {
	d.statistics = SolverStatistics{}
	d.assignment = make(Assignment)
	d.cnf = nil
}
