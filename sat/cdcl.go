package sat

import (
	"sort"
	"time"

	"github.com/xDarkicex/logic/core"
)

// CDCLSolver implements Conflict-Driven Clause Learning
type CDCLSolver struct {
	// Core components
	statistics SolverStatistics
	assignment Assignment
	cnf        *CNF
	trail      DecisionTrail
	startTime  time.Time

	// CDCL-specific components
	heuristic       Heuristic
	restartStrategy RestartStrategy
	deletionPolicy  ClauseDeletionPolicy
	analyzer        ConflictAnalyzer

	// Watch literals for efficient propagation
	watchLists map[Literal][]*Clause

	propagationQueue []Literal

	// Learned clauses management
	learnedClauses []*Clause
	maxLearnedSize int

	// Current decision level
	decisionLevel int

	// Activity scores for variables (VSIDS)
	variableActivity map[string]float64
	varActivityInc   float64
	varActivityDecay float64

	// Restart management
	conflicts   int64
	nextRestart int64
}

// NewCDCLSolver creates a new CDCL solver with default configuration
func NewCDCLSolver() *CDCLSolver {
	solver := &CDCLSolver{
		statistics:       SolverStatistics{},
		assignment:       make(Assignment),
		trail:            NewSimpleDecisionTrail(),
		watchLists:       make(map[Literal][]*Clause),
		learnedClauses:   make([]*Clause, 0),
		maxLearnedSize:   1000,
		decisionLevel:    0,
		variableActivity: make(map[string]float64),
		varActivityInc:   1.0,
		varActivityDecay: 0.95,
		conflicts:        0,
		nextRestart:      100, // Initial restart threshold
	}

	// Initialize components with defaults
	solver.heuristic = NewVSIDSHeuristic()
	solver.restartStrategy = NewLubyRestartStrategy()
	solver.deletionPolicy = NewActivityBasedDeletion()
	solver.analyzer = NewFirstUIPAnalyzer()

	return solver
}

// NewCDCLSolverWithConfig creates CDCL solver with custom configuration
func NewCDCLSolverWithConfig(config CDCLConfig) *CDCLSolver {
	solver := NewCDCLSolver()

	if config.Heuristic != nil {
		solver.heuristic = config.Heuristic
	}
	if config.RestartStrategy != nil {
		solver.restartStrategy = config.RestartStrategy
	}
	if config.DeletionPolicy != nil {
		solver.deletionPolicy = config.DeletionPolicy
	}
	if config.ConflictAnalyzer != nil {
		solver.analyzer = config.ConflictAnalyzer
	}
	if config.MaxLearnedSize > 0 {
		solver.maxLearnedSize = config.MaxLearnedSize
	}

	return solver
}

// CDCLConfig holds configuration for CDCL solver
type CDCLConfig struct {
	Heuristic        Heuristic
	RestartStrategy  RestartStrategy
	DeletionPolicy   ClauseDeletionPolicy
	ConflictAnalyzer ConflictAnalyzer
	MaxLearnedSize   int
}

// Name returns solver name
func (c *CDCLSolver) Name() string {
	return "CDCL"
}

// Solve solves the SAT problem using CDCL
func (c *CDCLSolver) Solve(cnf *CNF) *SolverResult {
	return c.SolveWithTimeout(cnf, 0)
}

// SolveWithTimeout solves with timeout
func (c *CDCLSolver) SolveWithTimeout(cnf *CNF, timeout time.Duration) *SolverResult {
	c.startTime = time.Now()
	c.cnf = cnf
	c.assignment = make(Assignment)
	c.statistics = SolverStatistics{}
	c.decisionLevel = 0
	c.conflicts = 0

	// Initialize watch lists
	err := c.initializeWatchLists()
	if err != nil {
		return &SolverResult{Error: err}
	}

	// Initialize variable activities
	c.initializeVariableActivities()

	// Set up timeout
	var timeoutChan <-chan time.Time
	if timeout > 0 {
		timeoutChan = time.After(timeout)
	}

	// Main CDCL loop
	for {
		// Check timeout
		select {
		case <-timeoutChan:
			return &SolverResult{
				Error: core.NewLogicError("sat", "CDCLSolver.SolveWithTimeout",
					"timeout exceeded"),
				Statistics: c.statistics,
			}
		default:
		}

		// Boolean Constraint Propagation (BCP)
		conflictClause := c.propagate()

		if conflictClause != nil {
			// Conflict found
			c.statistics.Conflicts++
			c.conflicts++

			if c.decisionLevel == 0 {
				// Conflict at decision level 0 - unsatisfiable
				c.statistics.TimeElapsed = time.Since(c.startTime).Nanoseconds()
				return &SolverResult{
					Satisfiable: false,
					Statistics:  c.statistics,
				}
			}

			// Analyze conflict and learn clause
			learnedClause, backtrackLevel := c.analyzer.Analyze(conflictClause, c.trail)

			if learnedClause != nil {
				c.learnClause(learnedClause)
				c.statistics.LearnedClauses++
			}

			// Backtrack
			c.backtrack(backtrackLevel)

			// Update heuristic
			c.heuristic.Update(conflictClause)

			// Check for restart
			if c.restartStrategy.ShouldRestart(c.statistics) {
				c.restart()
				c.statistics.Restarts++
			}

			// Clause deletion
			if len(c.learnedClauses) > c.maxLearnedSize {
				c.deleteClauses()
			}

		} else {
			// No conflict - check if solved
			if c.allVariablesAssigned() {
				// SAT - found satisfying assignment
				c.statistics.TimeElapsed = time.Since(c.startTime).Nanoseconds()
				return &SolverResult{
					Satisfiable: true,
					Assignment:  c.assignment.Clone(),
					Statistics:  c.statistics,
				}
			}

			// Choose next decision variable
			decisionVar := c.chooseDecisionVariable()
			if decisionVar == "" {
				// This should not happen if allVariablesAssigned works correctly
				return &SolverResult{
					Error: core.NewLogicError("sat", "CDCLSolver.SolveWithTimeout",
						"no decision variable found but not all assigned"),
					Statistics: c.statistics,
				}
			}

			// Make decision
			c.decisionLevel++
			c.statistics.Decisions++

			// Simple decision: try true first
			// More sophisticated heuristics can be added
			c.assign(decisionVar, true, nil)
		}
	}
}

// propagate performs Boolean Constraint Propagation using watch lists
func (c *CDCLSolver) propagate() *Clause {
	// Process propagation queue
	for len(c.propagationQueue) > 0 {
		lit := c.propagationQueue[0]
		c.propagationQueue = c.propagationQueue[1:]

		// Check all clauses watching this literal
		for _, clause := range c.watchLists[lit] {
			// Check if clause is already satisfied
			if c.assignment.Satisfies(clause) {
				continue
			}

			// Try to find new watch literal
			newWatch, isUnit, isConflict := c.findNewWatch(clause, lit)
			if isConflict {
				return clause
			}
			if isUnit {
				// Get the unit literal
				unitLit := c.getUnitLiteral(clause)
				c.assign(unitLit.Variable, !unitLit.Negated, clause)
				c.statistics.Propagations++
			}
			if newWatch != (Literal{}) {
				// Update watch list
				c.updateWatchList(clause, lit, newWatch)
			}
		}
	}
	return nil
}

// isConflicted checks if clause conflicts with current assignment
func (c *CDCLSolver) isConflicted(clause *Clause) bool {
	for _, lit := range clause.Literals {
		if value, assigned := c.assignment[lit.Variable]; assigned {
			// If any literal is satisfied, no conflict
			if (value && !lit.Negated) || (!value && lit.Negated) {
				return false
			}
		} else {
			// Unassigned literal - no conflict yet
			return false
		}
	}
	return true // All literals falsified
}

// getUnassignedLiterals returns unassigned literals in clause
func (c *CDCLSolver) getUnassignedLiterals(clause *Clause) []Literal {
	var unassigned []Literal
	for _, lit := range clause.Literals {
		if !c.assignment.IsAssigned(lit.Variable) {
			unassigned = append(unassigned, lit)
		}
	}
	return unassigned
}

// assign assigns value to variable with reason
func (c *CDCLSolver) assign(variable string, value bool, reason *Clause) {
	c.assignment[variable] = value
	c.trail.Assign(variable, value, c.decisionLevel, reason)
	// Add to propagation queue
	falseLit := Literal{Variable: variable, Negated: value}
	c.propagationQueue = append(c.propagationQueue, falseLit)
}

// allVariablesAssigned checks if all variables are assigned
func (c *CDCLSolver) allVariablesAssigned() bool {
	for _, variable := range c.cnf.Variables {
		if !c.assignment.IsAssigned(variable) {
			return false
		}
	}
	return true
}

// chooseDecisionVariable uses heuristic to choose next variable
func (c *CDCLSolver) chooseDecisionVariable() string {
	unassigned := make([]string, 0)
	for _, variable := range c.cnf.Variables {
		if !c.assignment.IsAssigned(variable) {
			unassigned = append(unassigned, variable)
		}
	}

	if len(unassigned) == 0 {
		return ""
	}

	return c.heuristic.ChooseVariable(unassigned, c.assignment)
}

// learnClause adds learned clause to solver
func (c *CDCLSolver) learnClause(clause *Clause) {
	clause.Learned = true
	clause.ID = c.cnf.nextID
	c.cnf.nextID++

	c.learnedClauses = append(c.learnedClauses, clause)

	// Update variable activities based on learned clause
	for _, lit := range clause.Literals {
		c.bumpVariableActivity(lit.Variable)
	}
}

// backtrack to given decision level
func (c *CDCLSolver) backtrack(level int) {
	unassignedVars := c.trail.Backtrack(level)

	// Remove assignments
	for _, variable := range unassignedVars {
		delete(c.assignment, variable)
	}

	c.decisionLevel = level
}

// restart clears assignment and starts over
func (c *CDCLSolver) restart() {
	c.assignment = make(Assignment)
	c.trail.Clear()
	c.decisionLevel = 0
	c.restartStrategy.OnRestart()
}

// deleteClauses removes less useful learned clauses
func (c *CDCLSolver) deleteClauses() {
	// Sort by activity (lower activity = more likely to delete)
	sort.Slice(c.learnedClauses, func(i, j int) bool {
		return c.learnedClauses[i].Activity < c.learnedClauses[j].Activity
	})

	// Delete half of the clauses
	deleteCount := len(c.learnedClauses) / 2
	for i := 0; i < deleteCount; i++ {
		if c.deletionPolicy.ShouldDelete(c.learnedClauses[i], c.statistics) {
			// Remove clause from watch lists before deletion
			c.removeFromWatchLists(c.learnedClauses[i])
		}
	}

	// Keep the more active clauses
	c.learnedClauses = c.learnedClauses[deleteCount:]
	c.statistics.DeletedClauses += int64(deleteCount)
}

// Watch literal management
func (c *CDCLSolver) initializeWatchLists() error {
	c.watchLists = make(map[Literal][]*Clause)

	for _, clause := range c.cnf.Clauses {
		if len(clause.Literals) >= 2 {
			// Watch first two literals
			lit1 := clause.Literals[0]
			lit2 := clause.Literals[1]

			c.watchLists[lit1] = append(c.watchLists[lit1], clause)
			c.watchLists[lit2] = append(c.watchLists[lit2], clause)
		} else if len(clause.Literals) == 1 {
			// Unit clause - add to appropriate watch list
			lit := clause.Literals[0]
			c.watchLists[lit] = append(c.watchLists[lit], clause)
		}
		// Empty clauses cause immediate conflict
	}

	return nil
}

// removeFromWatchLists removes clause from watch lists
func (c *CDCLSolver) removeFromWatchLists(clause *Clause) {
	for _, lit := range clause.Literals {
		if watchedClauses, exists := c.watchLists[lit]; exists {
			// Remove clause from watch list
			for i, watchedClause := range watchedClauses {
				if watchedClause.ID == clause.ID {
					c.watchLists[lit] = append(watchedClauses[:i], watchedClauses[i+1:]...)
					break
				}
			}
		}
	}
}

// Variable activity management (VSIDS)
func (c *CDCLSolver) initializeVariableActivities() {
	for _, variable := range c.cnf.Variables {
		c.variableActivity[variable] = 0.0
	}
}

func (c *CDCLSolver) bumpVariableActivity(variable string) {
	c.variableActivity[variable] += c.varActivityInc

	// Rescale if activities get too large
	if c.variableActivity[variable] > 1e100 {
		c.rescaleVariableActivities()
	}
}

func (c *CDCLSolver) rescaleVariableActivities() {
	for variable := range c.variableActivity {
		c.variableActivity[variable] *= 1e-100
	}
	c.varActivityInc *= 1e-100
}

func (c *CDCLSolver) decayVariableActivities() {
	c.varActivityInc /= c.varActivityDecay
}

// Interface implementations
func (c *CDCLSolver) AddClause(clause *Clause) error {
	c.cnf.AddClause(clause)

	// Update watch lists for new clause
	if len(clause.Literals) >= 2 {
		lit1 := clause.Literals[0]
		lit2 := clause.Literals[1]
		c.watchLists[lit1] = append(c.watchLists[lit1], clause)
		c.watchLists[lit2] = append(c.watchLists[lit2], clause)
	}

	return nil
}

func (c *CDCLSolver) GetStatistics() SolverStatistics {
	return c.statistics
}

func (c *CDCLSolver) Reset() {
	c.statistics = SolverStatistics{}
	c.assignment = make(Assignment)
	c.trail.Clear()
	c.watchLists = make(map[Literal][]*Clause)
	c.learnedClauses = make([]*Clause, 0)
	c.decisionLevel = 0
	c.variableActivity = make(map[string]float64)
	c.conflicts = 0

	// Reset components
	c.heuristic.Reset()
	c.restartStrategy.Reset()
	c.deletionPolicy.Reset()
	c.analyzer.Reset()
}

// Add these helper functions to cdcl.go

// findNewWatch finds a new literal to watch in the clause
// Returns: (newWatch, isUnit, isConflict)
func (c *CDCLSolver) findNewWatch(clause *Clause, falseLit Literal) (Literal, bool, bool) {
	unassigned := 0
	satisfied := false
	var newWatch Literal

	for _, lit := range clause.Literals {
		// Skip the literal that just became false
		if lit.Equals(falseLit) {
			continue
		}

		if !c.assignment.IsAssigned(lit.Variable) {
			unassigned++
			if newWatch == (Literal{}) {
				newWatch = lit // First unassigned literal found
			}
		} else {
			value := c.assignment[lit.Variable]
			if (value && !lit.Negated) || (!value && lit.Negated) {
				satisfied = true
				newWatch = lit // Satisfied literal is best watch
				break
			}
		}
	}

	if satisfied {
		return newWatch, false, false
	}

	if unassigned == 0 {
		// All literals are assigned and false - conflict
		return Literal{}, false, true
	}

	if unassigned == 1 {
		// Unit clause
		return Literal{}, true, false
	}

	// Multiple unassigned - return first one found
	return newWatch, false, false
}

// getUnitLiteral returns the unassigned literal in a unit clause
func (c *CDCLSolver) getUnitLiteral(clause *Clause) Literal {
	for _, lit := range clause.Literals {
		if !c.assignment.IsAssigned(lit.Variable) {
			return lit
		}
	}
	// Should not reach here if called correctly
	return clause.Literals[0]
}

// updateWatchList updates watch lists when changing watched literals
func (c *CDCLSolver) updateWatchList(clause *Clause, oldWatch Literal, newWatch Literal) {
	// Remove clause from old watch list
	if watchedClauses, exists := c.watchLists[oldWatch]; exists {
		for i, watchedClause := range watchedClauses {
			if watchedClause.ID == clause.ID {
				c.watchLists[oldWatch] = append(watchedClauses[:i], watchedClauses[i+1:]...)
				break
			}
		}
	}

	// Add clause to new watch list
	c.watchLists[newWatch] = append(c.watchLists[newWatch], clause)
}
