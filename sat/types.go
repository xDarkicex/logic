package sat

import (
	"fmt"
	"strings"
)

// Literal represents a boolean variable or its negation
// Positive literal: Variable = "A", Negated = false
// Negative literal: Variable = "A", Negated = true
type Literal struct {
	Variable string
	Negated  bool
}

// String returns string representation of literal
func (l Literal) String() string {
	if l.Negated {
		return "¬" + l.Variable
	}
	return l.Variable
}

// Negate returns the negation of this literal
func (l Literal) Negate() Literal {
	return Literal{Variable: l.Variable, Negated: !l.Negated}
}

// Equals checks if two literals are identical
func (l Literal) Equals(other Literal) bool {
	return l.Variable == other.Variable && l.Negated == other.Negated
}

// Clause represents a disjunction of literals (OR)
// Empty clause represents false (unsatisfiable)
// Unit clause has exactly one literal
type Clause struct {
	Literals []Literal
	ID       int     // Unique identifier for tracking
	Learned  bool    // True if this is a learned clause
	Activity float64 // For clause deletion heuristics
}

// NewClause creates a new clause with given literals
func NewClause(literals ...Literal) *Clause {
	return &Clause{
		Literals: literals,
		Learned:  false,
		Activity: 0.0,
	}
}

// String returns string representation of clause
func (c *Clause) String() string {
	if len(c.Literals) == 0 {
		return "⊥" // Empty clause (false)
	}

	parts := make([]string, len(c.Literals))
	for i, lit := range c.Literals {
		parts[i] = lit.String()
	}
	return "(" + strings.Join(parts, " ∨ ") + ")"
}

// IsUnit returns true if clause has exactly one literal
func (c *Clause) IsUnit() bool {
	return len(c.Literals) == 1
}

// IsEmpty returns true if clause has no literals (contradiction)
func (c *Clause) IsEmpty() bool {
	return len(c.Literals) == 0
}

// Contains checks if clause contains the given literal
func (c *Clause) Contains(lit Literal) bool {
	for _, l := range c.Literals {
		if l.Equals(lit) {
			return true
		}
	}
	return false
}

// CNF represents a formula in Conjunctive Normal Form
// It's a conjunction (AND) of clauses
type CNF struct {
	Clauses   []*Clause
	Variables []string // All variables in the formula
	nextID    int      // For generating unique clause IDs
}

// NewCNF creates a new CNF formula
func NewCNF() *CNF {
	return &CNF{
		Clauses:   make([]*Clause, 0),
		Variables: make([]string, 0),
		nextID:    1,
	}
}

// AddClause adds a clause to the CNF formula
func (cnf *CNF) AddClause(clause *Clause) {
	clause.ID = cnf.nextID
	cnf.nextID++
	cnf.Clauses = append(cnf.Clauses, clause)

	// Track variables properly
	for _, lit := range clause.Literals {
		if !cnf.containsVariable(lit.Variable) {
			cnf.Variables = append(cnf.Variables, lit.Variable)
		}
	}
}

// containsVariable checks if variable is already tracked
func (cnf *CNF) containsVariable(variable string) bool {
	for _, v := range cnf.Variables {
		if v == variable {
			return true
		}
	}
	return false
}

// String returns string representation of CNF
func (cnf *CNF) String() string {
	if len(cnf.Clauses) == 0 {
		return "⊤" // Empty CNF (true)
	}

	parts := make([]string, len(cnf.Clauses))
	for i, clause := range cnf.Clauses {
		parts[i] = clause.String()
	}
	return strings.Join(parts, " ∧ ")
}

// Assignment represents a partial or complete truth assignment
type Assignment map[string]bool

// Clone creates a deep copy of the assignment
func (a Assignment) Clone() Assignment {
	clone := make(Assignment, len(a))
	for k, v := range a {
		clone[k] = v
	}
	return clone
}

// IsAssigned checks if variable has been assigned
func (a Assignment) IsAssigned(variable string) bool {
	_, exists := a[variable]
	return exists
}

// Satisfies checks if assignment satisfies the given clause
func (a Assignment) Satisfies(clause *Clause) bool {
	if clause == nil || len(clause.Literals) == 0 {
		return false // Empty clause is never satisfied
	}

	// A clause is satisfied if at least one literal is satisfied
	for _, lit := range clause.Literals {
		if value, assigned := a[lit.Variable]; assigned {
			// Check if this literal is satisfied
			if (value && !lit.Negated) || (!value && lit.Negated) {
				return true // At least one literal is satisfied
			}
		} else {
			// Unassigned variable means clause could still be satisfied
			// Don't return true here - we need to check all assigned literals
		}
	}

	// If we reach here, either:
	// 1. All assigned literals are falsified (conflict if all are assigned)
	// 2. Some literals are unassigned

	// Check if all literals are assigned and falsified
	allAssigned := true
	for _, lit := range clause.Literals {
		if _, assigned := a[lit.Variable]; !assigned {
			allAssigned = false
			break
		}
	}

	// If all literals are assigned and we reached here, they're all falsified
	return !allAssigned
}

// ConflictsWith checks if assignment conflicts with the clause
func (a Assignment) ConflictsWith(clause *Clause) bool {
	unassignedCount := 0
	for _, lit := range clause.Literals {
		if value, assigned := a[lit.Variable]; assigned {
			// If any literal is satisfied, no conflict
			if (value && !lit.Negated) || (!value && lit.Negated) {
				return false
			}
		} else {
			unassignedCount++
		}
	}
	// Conflict only if all literals are assigned and falsified
	return unassignedCount == 0
}

// SolverResult represents the result of SAT solving
type SolverResult struct {
	Satisfiable bool
	Assignment  Assignment
	Statistics  SolverStatistics
	Error       error
}

// SolverStatistics tracks solver performance metrics
type SolverStatistics struct {
	Decisions      int64 // Number of decision variables chosen
	Propagations   int64 // Number of unit propagations
	Conflicts      int64 // Number of conflicts encountered
	Restarts       int64 // Number of restarts performed
	LearnedClauses int64 // Number of clauses learned
	DeletedClauses int64 // Number of clauses deleted
	TimeElapsed    int64 // Solving time in nanoseconds
}

// String returns formatted statistics
func (s SolverStatistics) String() string {
	return fmt.Sprintf(
		"Decisions: %d, Propagations: %d, Conflicts: %d, Restarts: %d, Learned: %d",
		s.Decisions, s.Propagations, s.Conflicts, s.Restarts, s.LearnedClauses,
	)
}
