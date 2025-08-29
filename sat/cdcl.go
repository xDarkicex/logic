package sat

import (
	"time"

	"github.com/xDarkicex/logic/core"
)

// CDCLSolver implements state-of-the-art CDCL with all modern optimizations
type CDCLSolver struct {
	// Core solver state
	statistics SolverStatistics
	assignment Assignment
	cnf        *CNF
	trail      DecisionTrail
	startTime  time.Time

	// Advanced CDCL components
	heuristic       Heuristic
	restartStrategy RestartStrategy
	deletionPolicy  ClauseDeletionPolicy
	analyzer        ConflictAnalyzer

	// Two-watched literals optimization (variable-keyed watch lists)
	watchLists map[string][]*WatchedClause // Variable -> clauses watching it

	// Learned clause management (tiered)
	clauseDatabase *ClauseDatabase
	maxLearnedSize int
	clauseActivity map[int]float64
	clauseDecay    float64

	// Decision level management
	decisionLevel int

	// Variable activity (VSIDS with optimizations)
	variableActivity map[string]float64
	varActivityInc   float64
	varActivityDecay float64

	// Restart and deletion counters
	conflicts        int64
	restartThreshold int64

	// Preprocessing and optimization
	preprocessor *SATPreprocessor

	// Performance optimizations
	conflictLimit    int64
	propagationQueue []Literal
	queueHead        int

	// LBD tracking
	lbdSum          int64 // Sum of all LBDs for average tracking
	glueClauseCount int64 // Count of glue clauses (LBD <= 2)

	// Caches for performance
	unassignedCache []string
	cacheValid      bool

	// Additional performance optimizations
	propagationCache map[string]bool // Cache for propagation state

	// Inprocessor
	inprocessor            Inprocessor
	inprocessConfig        InprocessConfig
	lastInprocess          int64
	inprocessGap           int64
	lastInprocessReduction int   // reduction from last inprocessing run (clauses/vars/units)
	lastInprocessCostNs    int64 // cost of last inprocessing run (nanoseconds)

	// Incremental Lazy Backtracking (ILB)
	ilb *IncrementalLazyBacktrack

	// Chronological backtracking tracking
	chronologicalStats *ChronologicalStats

	// XOR constraint support
	extendedCNF        *ExtendedCNF        // Extended CNF with XOR clauses
	gaussianEliminator *GaussianEliminator // Gaussian eliminator
	xorEnabled         bool                // Enable XOR support

	// XOR-specific statistics
	xorPropagations int64
	xorConflicts    int64
	gaussianRuns    int64
}

// IncrementalLazyBacktrack manages lazy backtracking optimization
type IncrementalLazyBacktrack struct {
	enabled                bool
	reimplicationQueue     []Literal
	lazyBacktrackLevel     int
	chronologicalEnabled   bool
	reimplicationCache     map[string]bool
	levelImplicationCount  map[int]int
	lastReimplicationCost  int64 // nanoseconds
	reimplicationThreshold int64 // cost threshold for using reimplication
}

// ChronologicalStats tracks chronological backtracking effectiveness
type ChronologicalStats struct {
	chronologicalAttempts  int64
	chronologicalSuccesses int64
	recentSuccessRate      float64
	adaptiveThreshold      int    // maximum level difference for chronological
	successWindow          []bool // sliding window of recent successes
	windowIndex            int
}

func NewIncrementalLazyBacktrack() *IncrementalLazyBacktrack {
	return &IncrementalLazyBacktrack{
		enabled:                true,
		reimplicationQueue:     make([]Literal, 0, 100),
		chronologicalEnabled:   true,
		reimplicationCache:     make(map[string]bool),
		levelImplicationCount:  make(map[int]int),
		reimplicationThreshold: 1000000, // 1ms threshold
	}
}

func NewChronologicalStats() *ChronologicalStats {
	return &ChronologicalStats{
		adaptiveThreshold: 2,
		successWindow:     make([]bool, 20), // 20-conflict sliding window
		recentSuccessRate: 0.5,              // Start optimistic
	}
}

// NewCDCLSolver creates a new unified CDCL solver with advanced configuration by default
func NewCDCLSolver() *CDCLSolver {
	solver := &CDCLSolver{
		statistics: SolverStatistics{
			LBDDistribution: make(map[int]int64),
		},
		assignment:       make(Assignment),
		trail:            NewAdvancedDecisionTrail(), // Use advanced trail by default
		watchLists:       make(map[string][]*WatchedClause),
		maxLearnedSize:   2000,
		clauseActivity:   make(map[int]float64),
		clauseDecay:      0.999,
		decisionLevel:    0,
		variableActivity: make(map[string]float64),
		varActivityInc:   1.0,
		varActivityDecay: 0.95,
		conflicts:        0,
		restartThreshold: 100,
		conflictLimit:    10000000,
		propagationQueue: make([]Literal, 0),
		queueHead:        0,
		lbdSum:           0,
		glueClauseCount:  0,
		unassignedCache:  make([]string, 0),
		cacheValid:       false,
		propagationCache: make(map[string]bool),
		// Inprocessing initialization
		inprocessConfig:        DefaultInprocessConfig(),
		lastInprocess:          0,
		inprocessGap:           4000,
		lastInprocessReduction: 0,
		lastInprocessCostNs:    0,
		// ILB initialization
		ilb:                NewIncrementalLazyBacktrack(),
		chronologicalStats: NewChronologicalStats(),
		// XOR support initialization
		gaussianEliminator: NewGaussianEliminator(),
		xorEnabled:         true,
		xorPropagations:    0,
		xorConflicts:       0,
		gaussianRuns:       0,
	}

	// Initialize enhanced components by default (now the base versions include all enhancements)
	solver.heuristic = NewVSIDSHeuristic()             // Now includes LRB, polarity, anti-aging
	solver.restartStrategy = NewLubyRestartStrategy()  // Now hybrid Luby+Glucose
	solver.deletionPolicy = NewActivityBasedDeletion() // Now LBD-aware
	solver.analyzer = NewFirstUIPAnalyzer()
	solver.preprocessor = NewSATPreprocessor()
	solver.inprocessor = NewModernInprocessor()

	// Initialize the tiered clause database:
	// recentProtectionAge ~ 1000 conflicts is common; tune as needed or make it configurable.
	solver.clauseDatabase = NewClauseDatabase(solver.maxLearnedSize, 1000)

	return solver
}

// NewAdvancedCDCLSolver creates an alias for backward compatibility
// Deprecated: Use NewCDCLSolver instead, which now includes all advanced features by default
func NewAdvancedCDCLSolver() *CDCLSolver {
	return NewCDCLSolver()
}

// Name returns solver name
func (c *CDCLSolver) Name() string {
	return "CDCL"
}

// Solve solves the SAT problem using CDCL
func (c *CDCLSolver) Solve(cnf *CNF) *SolverResult {
	return c.SolveWithTimeout(cnf, 0)
}

// SolveExtended adds XOR-aware solving method
func (c *CDCLSolver) SolveExtended(ecnf *ExtendedCNF) *SolverResult {
	c.extendedCNF = ecnf
	// If no XOR clauses, fall back to regular solving
	if !ecnf.HasXORClauses() {
		return c.Solve(ecnf.CNF)
	}

	return c.SolveWithTimeoutExtended(ecnf, 0)
}

// SolveWithTimeout solves with timeout using advanced CDCL algorithm with inprocessing
func (c *CDCLSolver) SolveWithTimeout(cnf *CNF, timeout time.Duration) *SolverResult {
	c.startTime = time.Now()
	c.cnf = cnf
	c.assignment = make(Assignment)
	c.statistics = SolverStatistics{LBDDistribution: make(map[int]int64)}
	c.decisionLevel = 0
	c.conflicts = 0
	c.lbdSum = 0
	c.glueClauseCount = 0
	c.unassignedCache = make([]string, 0)
	c.cacheValid = false
	c.propagationCache = make(map[string]bool)

	// Initialize components
	c.initializeWatchLists()
	c.initializeHeuristics()

	// Initial inprocessing at the start (if enabled)
	if c.inprocessor != nil && c.inprocessConfig.EnableInitialInprocess {
		c.performInprocessing()
	}

	// Setup timeout
	var timeoutChan <-chan time.Time
	if timeout > 0 {
		timeoutChan = time.After(timeout)
	}

	// Main CDCL loop with inprocessing integration
	for c.conflicts < c.conflictLimit {
		select {
		case <-timeoutChan:
			return &SolverResult{
				Error:      core.NewLogicError("sat", "CDCLSolver.SolveWithTimeout", "timeout exceeded"),
				Statistics: c.statistics,
			}
		default:
		}

		// **INPROCESSING INTEGRATION POINT 1**:
		// Run inprocessing at decision level 0 based on conflict intervals
		if c.shouldRunInprocessing() {
			c.performInprocessing()
		}

		// Use advanced two-watched literals propagation
		conflictClause := c.propagate()
		if conflictClause != nil {
			c.statistics.Conflicts++
			c.conflicts++
			if c.decisionLevel == 0 {
				c.statistics.TimeElapsed = time.Since(c.startTime).Nanoseconds()
				return &SolverResult{
					Satisfiable: false,
					Statistics:  c.statistics,
				}
			}

			// Conflict analysis and learning
			learnedClause, backtrackLevel := c.analyzer.Analyze(conflictClause, c.trail)
			if learnedClause != nil {
				c.learnClause(learnedClause)
				c.statistics.LearnedClauses++
			}

			// Use enhanced lazy backtracking instead of regular backtracking
			c.lazyBacktrack(backtrackLevel)

			// Update heuristic
			c.heuristic.Update(conflictClause)

			// **INPROCESSING INTEGRATION POINT 2**:
			// After backtracking to level 0, consider inprocessing
			if c.decisionLevel == 0 && c.shouldRunInprocessingAfterBacktrack() {
				c.performInprocessing()
			}

			// Promote aged recent clauses periodically (simple hook after conflict analysis)
			if c.clauseDatabase != nil {
				c.clauseDatabase.PromoteFromRecent(c.conflicts)
			}

			// Clause deletion
			if c.clauseDatabase != nil && c.clauseDatabase.Size() > c.maxLearnedSize {
				c.deleteClauses()
			}

			// Check for restart
			if c.restartStrategy.ShouldRestart(c.statistics) {
				c.restart()
				c.statistics.Restarts++
				// **INPROCESSING INTEGRATION POINT 3**:
				// After restart, we're at level 0 - good time for inprocessing
				if c.shouldRunInprocessingAfterRestart() {
					c.performInprocessing()
				}
			}
			continue
		}

		if c.allVariablesAssigned() {
			c.statistics.TimeElapsed = time.Since(c.startTime).Nanoseconds()
			return &SolverResult{
				Satisfiable: true,
				Assignment:  c.assignment.Clone(),
				Statistics:  c.statistics,
			}
		}

		// Make decision using advanced heuristics
		decisionVar := c.chooseDecisionVariable()
		if decisionVar == "" {
			break
		}

		c.decisionLevel++
		c.statistics.Decisions++
		// Use enhanced polarity heuristic
		polarity := c.choosePolarity(decisionVar)
		c.assign(decisionVar, polarity, nil)
	}

	return &SolverResult{
		Satisfiable: false,
		Statistics:  c.statistics,
	}
}

// SolveWithTimeoutExtended provides XOR-aware solving with timeout
func (c *CDCLSolver) SolveWithTimeoutExtended(ecnf *ExtendedCNF, timeout time.Duration) *SolverResult {
	c.startTime = time.Now()
	c.extendedCNF = ecnf
	c.cnf = ecnf.CNF
	c.assignment = make(Assignment)
	c.statistics = SolverStatistics{LBDDistribution: make(map[int]int64)}
	c.decisionLevel = 0
	c.conflicts = 0
	c.lbdSum = 0
	c.glueClauseCount = 0
	c.unassignedCache = make([]string, 0)
	c.cacheValid = false
	c.propagationCache = make(map[string]bool)

	// Initialize components
	c.initializeWatchLists()
	c.initializeHeuristics()

	// Initial inprocessing at the start (if enabled)
	if c.inprocessor != nil && c.inprocessConfig.EnableInitialInprocess {
		c.performInprocessing()
	}

	// Setup timeout
	var timeoutChan <-chan time.Time
	if timeout > 0 {
		timeoutChan = time.After(timeout)
	}

	// Main CDCL loop with XOR support
	for c.conflicts < c.conflictLimit {
		select {
		case <-timeoutChan:
			return &SolverResult{
				Error:      core.NewLogicError("sat", "CDCLSolver.SolveWithTimeoutExtended", "timeout exceeded"),
				Statistics: c.statistics,
			}
		default:
		}

		// **GAUSSIAN ELIMINATION INTEGRATION**
		if c.xorEnabled && c.gaussianEliminator.ShouldRunGaussian(c.conflicts, len(c.extendedCNF.XORClauses)) {
			c.performGaussianElimination()
		}

		// **INPROCESSING INTEGRATION POINT 1**:
		// Run inprocessing at decision level 0 based on conflict intervals
		if c.shouldRunInprocessing() {
			c.performInprocessing()
		}

		// Regular propagation
		conflictClause := c.propagate()

		// **XOR PROPAGATION**
		if conflictClause == nil && c.xorEnabled {
			xorConflict := c.propagateXOR()
			if xorConflict != nil {
				conflictClause = c.convertXORConflictToClause(xorConflict)
			}
		}

		if conflictClause != nil {
			c.statistics.Conflicts++
			c.conflicts++
			if c.decisionLevel == 0 {
				c.statistics.TimeElapsed = time.Since(c.startTime).Nanoseconds()
				return &SolverResult{
					Satisfiable: false,
					Statistics:  c.statistics,
				}
			}

			// Conflict analysis and learning
			learnedClause, backtrackLevel := c.analyzer.Analyze(conflictClause, c.trail)
			if learnedClause != nil {
				c.learnClause(learnedClause)
				c.statistics.LearnedClauses++
			}

			// Use enhanced lazy backtracking instead of regular backtracking
			c.lazyBacktrack(backtrackLevel)

			// Update heuristic
			c.heuristic.Update(conflictClause)

			// **INPROCESSING INTEGRATION POINT 2**:
			// After backtracking to level 0, consider inprocessing
			if c.decisionLevel == 0 && c.shouldRunInprocessingAfterBacktrack() {
				c.performInprocessing()
			}

			// Promote aged recent clauses periodically (simple hook after conflict analysis)
			if c.clauseDatabase != nil {
				c.clauseDatabase.PromoteFromRecent(c.conflicts)
			}

			// Clause deletion
			if c.clauseDatabase != nil && c.clauseDatabase.Size() > c.maxLearnedSize {
				c.deleteClauses()
			}

			// Check for restart
			if c.restartStrategy.ShouldRestart(c.statistics) {
				c.restart()
				c.statistics.Restarts++
				// **INPROCESSING INTEGRATION POINT 3**:
				// After restart, we're at level 0 - good time for inprocessing
				if c.shouldRunInprocessingAfterRestart() {
					c.performInprocessing()
				}
			}
			continue
		}

		if c.allVariablesAssigned() {
			c.statistics.TimeElapsed = time.Since(c.startTime).Nanoseconds()
			return &SolverResult{
				Satisfiable: true,
				Assignment:  c.assignment.Clone(),
				Statistics:  c.statistics,
			}
		}

		// Make decision using advanced heuristics
		decisionVar := c.chooseDecisionVariable()
		if decisionVar == "" {
			break
		}

		c.decisionLevel++
		c.statistics.Decisions++
		// Use enhanced polarity heuristic
		polarity := c.choosePolarity(decisionVar)
		c.assign(decisionVar, polarity, nil)
	}

	return &SolverResult{
		Satisfiable: false,
		Statistics:  c.statistics,
	}
}

// performGaussianElimination integrates Gaussian elimination
func (c *CDCLSolver) performGaussianElimination() {
	if c.decisionLevel != 0 {
		return // Only run at root level
	}

	result, err := c.gaussianEliminator.PerformGaussianElimination(c.extendedCNF, c.assignment, c.conflicts)
	if err != nil {
		return
	}

	c.gaussianRuns++

	// Handle contradiction
	if result.ConflictFound {
		c.xorConflicts++
		// This will be handled in the main loop
		return
	}

	// Apply unit propagations
	for _, unit := range result.UnitsLearned {
		value := !unit.Negated
		c.assign(unit.Variable, value, nil)
		c.xorPropagations++
	}

	// Add learned XOR clauses
	for _, xorClause := range result.XORClausesLearned {
		c.extendedCNF.AddXORClause(xorClause)
	}
}

// propagateXOR performs XOR constraint propagation
func (c *CDCLSolver) propagateXOR() *XORClause {
	changed := true
	for changed {
		changed = false
		for _, xorClause := range c.extendedCNF.XORClauses {
			satisfied, result := xorClause.IsSatisfied(c.assignment)
			if satisfied && !result {
				// XOR constraint violated
				return xorClause
			}

			// Check for unit XOR propagation
			unassignedVars := make([]string, 0)
			xorSum := false
			for _, variable := range xorClause.Variables {
				if value, assigned := c.assignment[variable]; assigned {
					if value {
						xorSum = !xorSum
					}
				} else {
					unassignedVars = append(unassignedVars, variable)
				}
			}

			// Unit XOR propagation
			if len(unassignedVars) == 1 {
				variable := unassignedVars[0]
				// The unassigned variable must make xorSum == xorClause.Parity
				requiredValue := xorSum != xorClause.Parity
				c.assign(variable, requiredValue, nil) // XOR reason handling would need extension
				c.xorPropagations++
				changed = true
			}
		}
	}
	return nil
}

// convertXORConflictToClause converts XOR conflict to proper conflict clause for CDCL analysis
func (c *CDCLSolver) convertXORConflictToClause(xorClause *XORClause) *Clause {
	// For a proper XOR conflict conversion, we need to:
	// 1. Identify the exact nature of the XOR violation
	// 2. Create a clause that prevents the current violating assignment
	// 3. Handle both assigned and unassigned variables correctly
	// 4. Ensure compatibility with CDCL conflict analysis

	assignedVars := make([]string, 0, len(xorClause.Variables))
	unassignedVars := make([]string, 0, len(xorClause.Variables))
	currentXorSum := false

	// Analyze current state of XOR variables
	for _, variable := range xorClause.Variables {
		if value, assigned := c.assignment[variable]; assigned {
			assignedVars = append(assignedVars, variable)
			if value {
				currentXorSum = !currentXorSum
			}
		} else {
			unassignedVars = append(unassignedVars, variable)
		}
	}

	// Determine the type of XOR conflict and create appropriate clause
	if len(unassignedVars) == 0 {
		// All variables assigned - direct XOR violation
		return c.createFullXORConflictClause(xorClause, assignedVars, currentXorSum)
	} else if len(unassignedVars) == 1 {
		// Unit XOR conflict - one unassigned variable
		return c.createUnitXORConflictClause(xorClause, assignedVars, unassignedVars[0], currentXorSum)
	} else {
		// Multiple unassigned variables - create implication-based conflict clause
		return c.createPartialXORConflictClause(xorClause, assignedVars, unassignedVars, currentXorSum)
	}
}

// createFullXORConflictClause handles XOR conflicts where all variables are assigned
func (c *CDCLSolver) createFullXORConflictClause(xorClause *XORClause, assignedVars []string, currentXorSum bool) *Clause {
	// When all variables are assigned and XOR is violated, we create a clause
	// that forces at least one variable to flip its current assignment
	literals := make([]Literal, 0, len(assignedVars))

	// Add negation of each current assignment to force a change
	for _, variable := range assignedVars {
		currentValue := c.assignment[variable]
		// Create literal that conflicts with current assignment
		literals = append(literals, Literal{
			Variable: variable,
			Negated:  currentValue, // If var is true, add ¬var; if var is false, add var
		})
	}

	// Create the conflict clause
	clause := NewClause(literals...)
	clause.Learned = true
	clause.ConflictType = "XOR_FULL"

	// Set LBD based on decision levels of involved variables
	c.setXORClauseLBD(clause)

	return clause
}

// createUnitXORConflictClause handles XOR conflicts with one unassigned variable
func (c *CDCLSolver) createUnitXORConflictClause(xorClause *XORClause, assignedVars []string, unassignedVar string, currentXorSum bool) *Clause {
	// With one unassigned variable, the XOR constraint determines its required value
	// If this leads to a conflict with other constraints, we create a clause that
	// prevents the current assignment pattern of assigned variables
	requiredValue := currentXorSum != xorClause.Parity
	literals := make([]Literal, 0, len(assignedVars)+1)

	// Add the required assignment for the unassigned variable
	literals = append(literals, Literal{
		Variable: unassignedVar,
		Negated:  !requiredValue, // Negate to create conflict clause
	})

	// Add negations of current assignments that led to this unit implication
	for _, variable := range assignedVars {
		currentValue := c.assignment[variable]
		literals = append(literals, Literal{
			Variable: variable,
			Negated:  currentValue,
		})
	}

	clause := NewClause(literals...)
	clause.Learned = true
	clause.ConflictType = "XOR_UNIT"
	c.setXORClauseLBD(clause)

	return clause
}

// createPartialXORConflictClause handles XOR conflicts with multiple unassigned variables
func (c *CDCLSolver) createPartialXORConflictClause(xorClause *XORClause, assignedVars []string, unassignedVars []string, currentXorSum bool) *Clause {
	// With multiple unassigned variables, we create a clause that captures
	// the constraint violation based on current partial assignment
	literals := make([]Literal, 0, len(assignedVars)+len(unassignedVars))

	// Strategy: Create a clause that prevents the current partial assignment
	// while allowing flexibility for unassigned variables

	// Add negations of current assignments
	for _, variable := range assignedVars {
		currentValue := c.assignment[variable]
		literals = append(literals, Literal{
			Variable: variable,
			Negated:  currentValue,
		})
	}

	// For unassigned variables, we need to consider which assignments
	// would be compatible with the XOR constraint given current assignments
	remainingParity := currentXorSum != xorClause.Parity

	// If we need an odd number of unassigned vars to be true for satisfaction,
	// and we can't satisfy it, create appropriate constraint
	if len(unassignedVars) > 0 {
		// Add constraints for unassigned variables based on XOR requirements
		// This is a sophisticated approach that considers the parity constraint
		c.addUnassignedXORConstraints(&literals, unassignedVars, remainingParity)
	}

	clause := NewClause(literals...)
	clause.Learned = true
	clause.ConflictType = "XOR_PARTIAL"
	c.setXORClauseLBD(clause)

	return clause
}

// addUnassignedXORConstraints adds appropriate constraints for unassigned variables in XOR conflicts
func (c *CDCLSolver) addUnassignedXORConstraints(literals *[]Literal, unassignedVars []string, remainingParity bool) {
	// This is a sophisticated constraint generation that considers:
	// 1. The required parity for satisfaction
	// 2. The current decision levels and heuristics
	// 3. The most effective literals for conflict learning

	// Sort unassigned variables by decision order or activity for better conflict clauses
	c.sortVariablesByActivity(unassignedVars)

	// Strategy: Add constraints that capture the XOR violation effectively
	// For XOR constraints, we typically want to avoid certain combinations
	if remainingParity {
		// Need odd number of unassigned vars to be true
		// Add constraints that prevent all false or all true (depending on count)
		if len(unassignedVars)%2 == 0 {
			// Even number of vars, need odd number true - prevent all false
			for _, variable := range unassignedVars {
				*literals = append(*literals, Literal{Variable: variable, Negated: false})
			}
		} else {
			// Odd number of vars, already satisfiable - add most restrictive constraint
			// Choose highest activity variable to force true
			if len(unassignedVars) > 0 {
				*literals = append(*literals, Literal{Variable: unassignedVars[0], Negated: false})
			}
		}
	} else {
		// Need even number of unassigned vars to be true
		if len(unassignedVars)%2 == 1 {
			// Odd number of vars, need even number true - prevent all true
			for _, variable := range unassignedVars {
				*literals = append(*literals, Literal{Variable: variable, Negated: true})
			}
		} else {
			// Even number of vars, already satisfiable - add balanced constraint
			if len(unassignedVars) >= 2 {
				// Force at least one to be false and one to be true for better propagation
				*literals = append(*literals, Literal{Variable: unassignedVars[0], Negated: true})
				*literals = append(*literals, Literal{Variable: unassignedVars[1], Negated: false})
			}
		}
	}
}

// sortVariablesByActivity sorts variables by VSIDS activity for better conflict clause generation
func (c *CDCLSolver) sortVariablesByActivity(variables []string) {
	// Sort in descending order of activity
	for i := 0; i < len(variables)-1; i++ {
		for j := i + 1; j < len(variables); j++ {
			if c.variableActivity[variables[i]] < c.variableActivity[variables[j]] {
				variables[i], variables[j] = variables[j], variables[i]
			}
		}
	}
}

// setXORClauseLBD sets appropriate LBD (Literal Block Distance) for XOR-derived clauses
func (c *CDCLSolver) setXORClauseLBD(clause *Clause) {
	// Calculate LBD based on decision levels of literals in the clause
	levelSet := make(map[int]bool)
	for _, literal := range clause.Literals {
		level := c.trail.GetLevel(literal.Variable)
		if level > 0 { // Don't count level 0 (unit facts)
			levelSet[level] = true
		}
	}

	lbd := len(levelSet)
	if lbd == 0 {
		lbd = 1 // At least 1 for unit clauses
	}

	clause.LBD = lbd

	// Mark as glue clause if LBD is small (good for keeping)
	if lbd <= 2 {
		clause.Glue = true
	}

	// Set activity based on current conflict count for clause management
	if c.clauseActivity != nil {
		c.clauseActivity[clause.ID] = float64(c.conflicts) + float64(lbd)
	}
}

// lazyBacktrack implements incremental lazy backtracking
func (c *CDCLSolver) lazyBacktrack(targetLevel int) {
	if !c.ilb.enabled {
		c.backtrack(targetLevel)
		return
	}

	currentLevel := c.decisionLevel

	// Check if chronological backtracking is beneficial
	if c.shouldUseChronologicalBacktrack(currentLevel, targetLevel) {
		c.performChronologicalBacktrack(targetLevel)
		return
	}

	// Try reimplication if the level difference is small
	if c.canReimplyAtLevel(targetLevel) {
		startTime := time.Now()
		success := c.performReimplication(targetLevel)
		c.ilb.lastReimplicationCost = time.Since(startTime).Nanoseconds()

		if success {
			c.decisionLevel = targetLevel
			return
		}
	}

	// Fall back to traditional backtracking
	c.backtrack(targetLevel)
}

// canReimplyAtLevel checks if reimplication is feasible and beneficial
func (c *CDCLSolver) canReimplyAtLevel(targetLevel int) bool {
	currentLevel := c.decisionLevel

	// Only consider reimplication for small level differences
	if currentLevel-targetLevel > 3 {
		return false
	}

	// Check cost threshold from previous attempts
	if c.ilb.lastReimplicationCost > c.ilb.reimplicationThreshold {
		return false
	}

	// Count implications that would need to be restored
	implicationsToRestore := 0
	for level := targetLevel + 1; level <= currentLevel; level++ {
		implicationsToRestore += c.ilb.levelImplicationCount[level]
	}

	// Only use reimplication if restoration cost is reasonable
	return implicationsToRestore < 50
}

// performReimplication restores implications without full backtracking
func (c *CDCLSolver) performReimplication(targetLevel int) bool {
	c.ilb.reimplicationQueue = c.ilb.reimplicationQueue[:0]
	c.ilb.reimplicationCache = make(map[string]bool)

	// Collect literals that need reimplication
	for variable, value := range c.assignment {
		level := c.trail.GetLevel(variable)
		reason := c.trail.GetReason(variable)

		if level > targetLevel && reason != nil {
			// This is an implied literal that needs restoration
			literal := Literal{Variable: variable, Negated: !value}
			c.ilb.reimplicationQueue = append(c.ilb.reimplicationQueue, literal)
		}
	}

	// Perform selective unassignment (only above target level)
	unassignedVars := make([]string, 0)
	for variable := range c.assignment {
		if c.trail.GetLevel(variable) > targetLevel {
			unassignedVars = append(unassignedVars, variable)
		}
	}

	// Remove assignments above target level
	for _, variable := range unassignedVars {
		delete(c.assignment, variable)
	}

	// Update trail to target level
	c.trail.Backtrack(targetLevel)

	// Attempt to restore implications through propagation
	return c.performRestorativePropagation()
}

// performRestorativePropagation tries to restore implications through BCP
func (c *CDCLSolver) performRestorativePropagation() bool {
	// Use existing propagation but track success
	conflictClause := c.propagate()
	if conflictClause != nil {
		// Reimplication failed - we have a conflict
		return false
	}

	// Check if we successfully restored the desired implications
	restoredCount := 0
	for _, literal := range c.ilb.reimplicationQueue {
		if c.assignment.IsAssigned(literal.Variable) {
			currentValue := c.assignment[literal.Variable]
			expectedValue := !literal.Negated
			if currentValue == expectedValue {
				restoredCount++
			}
		}
	}

	// Success if we restored most implications
	successRate := float64(restoredCount) / float64(len(c.ilb.reimplicationQueue))
	return successRate > 0.8
}

// shouldUseChronologicalBacktrack decides between chronological and non-chronological
func (c *CDCLSolver) shouldUseChronologicalBacktrack(conflictLevel, computedLevel int) bool {
	if !c.ilb.chronologicalEnabled {
		return false
	}

	levelDiff := conflictLevel - computedLevel

	// Use chronological if difference is small and recent success rate is good
	if levelDiff <= c.chronologicalStats.adaptiveThreshold &&
		c.chronologicalStats.recentSuccessRate > 0.3 {
		return true
	}

	return false
}

// performChronologicalBacktrack implements chronological backtracking
func (c *CDCLSolver) performChronologicalBacktrack(targetLevel int) {
	c.chronologicalStats.chronologicalAttempts++

	// Store state for success measurement
	preConflicts := c.conflicts

	// Perform chronological backtracking (level by level)
	currentLevel := c.decisionLevel
	success := true

	for level := currentLevel; level > targetLevel && success; level-- {
		// Try to backtrack just one level
		c.backtrack(level - 1)

		// Test if this intermediate state is productive
		if !c.isChronologicalStateProductive() {
			success = false
		}
	}

	// Update statistics
	if success && c.conflicts-preConflicts < 10 {
		c.updateChronologicalSuccess(true)
	} else {
		c.updateChronologicalSuccess(false)
	}

	// Fall back to direct backtrack if chronological failed
	if c.decisionLevel != targetLevel {
		c.backtrack(targetLevel)
	}
}

// isChronologicalStateProductive checks if current state is likely to be beneficial
func (c *CDCLSolver) isChronologicalStateProductive() bool {
	// Simple heuristic: check if we have reasonable propagation potential
	unassignedCount := 0
	for _, variable := range c.cnf.Variables {
		if !c.assignment.IsAssigned(variable) {
			unassignedCount++
		}
	}

	// State is productive if we haven't eliminated too many options
	return unassignedCount > len(c.cnf.Variables)/4
}

// updateChronologicalSuccess updates the success rate tracking
func (c *CDCLSolver) updateChronologicalSuccess(success bool) {
	stats := c.chronologicalStats

	// Update sliding window
	stats.successWindow[stats.windowIndex] = success
	stats.windowIndex = (stats.windowIndex + 1) % len(stats.successWindow)

	if success {
		stats.chronologicalSuccesses++
	}

	// Recalculate success rate
	successCount := 0
	for _, s := range stats.successWindow {
		if s {
			successCount++
		}
	}

	stats.recentSuccessRate = float64(successCount) / float64(len(stats.successWindow))

	// Adapt threshold based on success rate
	if stats.recentSuccessRate > 0.6 {
		if stats.adaptiveThreshold < 4 {
			stats.adaptiveThreshold++
		}
	} else if stats.recentSuccessRate < 0.2 {
		if stats.adaptiveThreshold > 1 {
			stats.adaptiveThreshold--
		}
	}
}

// shouldRunInprocessing determines if inprocessing should run based on current state
func (c *CDCLSolver) shouldRunInprocessing() bool {
	// Only run at decision level 0 to avoid corrupting the trail
	if c.decisionLevel != 0 {
		return false
	}

	// Use the adaptive gap consistently
	adaptiveGap := c.calculateAdaptiveInprocessGap()
	if c.conflicts < c.lastInprocess+adaptiveGap {
		return false
	}

	// Don't run if we haven't learned enough clauses
	if c.statistics.LearnedClauses < 100 {
		return false
	}

	// Keep the existing learned-clauses shortcut, but base thresholds on the adaptive gap
	return c.conflicts >= c.lastInprocess+adaptiveGap ||
		c.statistics.LearnedClauses >= c.lastInprocess+adaptiveGap/2
}

// shouldRunInprocessingAfterBacktrack checks if inprocessing should run after backtracking to level 0
func (c *CDCLSolver) shouldRunInprocessingAfterBacktrack() bool {
	gap := c.calculateAdaptiveInprocessGap()
	return c.conflicts > c.lastInprocess+gap*2 &&
		c.statistics.LearnedClauses > 500
}

// shouldRunInprocessingAfterRestart checks if inprocessing should run after restart
func (c *CDCLSolver) shouldRunInprocessingAfterRestart() bool {
	gap := c.calculateAdaptiveInprocessGap()
	return c.statistics.Restarts > 0 &&
		c.statistics.Restarts%3 == 0 &&
		c.conflicts > c.lastInprocess+gap/2
}

// calculateAdaptiveInprocessGap computes dynamic inprocessing interval
func (c *CDCLSolver) calculateAdaptiveInprocessGap() int64 {
	baseGap := c.inprocessGap

	// Existing adjustments
	if c.cnf != nil {
		clauseCount := int64(len(c.cnf.Clauses))
		if clauseCount < 1000 {
			baseGap = baseGap / 2 // Run more frequently on small formulas
		} else if clauseCount > 10000 {
			baseGap = baseGap * 2 // Run less frequently on large formulas
		}
	}

	// Adjust based on learning rate
	if c.statistics.Conflicts > 0 {
		learningRate := float64(c.statistics.LearnedClauses) / float64(c.statistics.Conflicts)
		if learningRate > 0.5 {
			// High learning rate - run more frequently
			baseGap = int64(float64(baseGap) * 0.8)
		} else if learningRate < 0.1 {
			// Low learning rate - run less frequently
			baseGap = int64(float64(baseGap) * 1.5)
		}
	}

	// NEW: effectiveness-aware scaling: reductions per millisecond
	if c.lastInprocessCostNs > 0 {
		effPerMs := float64(c.lastInprocessReduction) / (float64(c.lastInprocessCostNs) / 1e6)
		if effPerMs > 0.8 {
			baseGap = baseGap / 2 // more frequent if effective
		} else if effPerMs < 0.2 {
			baseGap = baseGap * 2 // less frequent if ineffective
		}
	}

	// Clamp to existing bounds used elsewhere
	if baseGap < 1000 {
		baseGap = 1000
	} else if baseGap > 20000 {
		baseGap = 20000
	}

	return baseGap
}

// performInprocessing executes inprocessing and handles state updates
func (c *CDCLSolver) performInprocessing() {
	if c.inprocessor == nil {
		return
	}

	startTime := time.Now()
	originalClauses := len(c.cnf.Clauses)
	originalVars := len(c.cnf.Variables)

	// Perform inprocessing
	result, err := c.inprocessor.Inprocess(c.cnf, c.assignment, c.decisionLevel)
	if err != nil {
		// Log error but don't fail - continue solving
		return
	}

	c.lastInprocess = c.conflicts

	// Update statistics
	c.statistics.InprocessRuns++
	inprocessTime := time.Since(startTime).Nanoseconds()

	// NEW: capture reduction and cost to drive adaptive scheduling
	totalReductions := result.ClausesRemoved + result.ClausesStrengthened +
		result.VariablesEliminated + result.UnitsLearned
	c.lastInprocessReduction = totalReductions
	c.lastInprocessCostNs = inprocessTime

	// Handle formula changes
	if result.FormulaReduced {
		newClauses := len(c.cnf.Clauses)
		newVars := len(c.cnf.Variables)

		// Update solver statistics
		c.statistics.ClausesReduced += int64(originalClauses - newClauses)
		c.statistics.VariablesEliminated += int64(originalVars - newVars)

		// **CRITICAL**: Rebuild watch lists if clauses were modified
		if result.ClausesRemoved > 0 || result.ClausesStrengthened > 0 {
			c.rebuildWatchLists()
		}
	}

	// Invalidate caches
	c.cacheValid = false

	// Update heuristic activities for new formula structure
	c.updateHeuristicsAfterInprocessing(result)

	// Adapt inprocessing frequency based on effectiveness
	c.adaptInprocessingFrequency(result, inprocessTime)
}

// rebuildWatchLists reconstructs watch lists after formula modification
func (c *CDCLSolver) rebuildWatchLists() {
	// Clear existing watch lists
	c.watchLists = make(map[string][]*WatchedClause)

	// Rebuild from current clauses
	c.initializeWatchLists()
}

// updateHeuristicsAfterInprocessing updates heuristics based on inprocessing results
func (c *CDCLSolver) updateHeuristicsAfterInprocessing(result *InprocessResult) {
	// Re-initialize variable activities for eliminated/modified variables
	c.initializeHeuristics()

	// Boost activity of variables in strengthened clauses
	if result.ClausesStrengthened > 0 {
		for _, clause := range c.cnf.Clauses {
			if clause.Learned && len(clause.Literals) <= 5 { // Likely strengthened
				for _, lit := range clause.Literals {
					c.bumpVariableActivity(lit.Variable)
				}
			}
		}
	}
}

// adaptInprocessingFrequency adjusts inprocessing parameters based on effectiveness
func (c *CDCLSolver) adaptInprocessingFrequency(result *InprocessResult, duration int64) {
	totalReductions := result.ClausesRemoved + result.ClausesStrengthened + result.VariablesEliminated

	// If inprocessing was very effective, run more frequently
	if totalReductions > 50 {
		c.inprocessGap = int64(float64(c.inprocessGap) * 0.8)
	} else if totalReductions < 5 {
		// If not very effective, run less frequently
		c.inprocessGap = int64(float64(c.inprocessGap) * 1.2)
	}

	// Bounds checking
	if c.inprocessGap < 1000 {
		c.inprocessGap = 1000
	} else if c.inprocessGap > 20000 {
		c.inprocessGap = 20000
	}
}

// propagate implements optimized BCP with two-watched literals
func (c *CDCLSolver) propagate() *Clause {
	c.queueHead = 0

	for c.queueHead < len(c.propagationQueue) {
		// Get next literal that was just falsified
		falseLit := c.propagationQueue[c.queueHead]
		c.queueHead++

		// Check if we've already processed this variable in this propagation round
		if c.propagationCache[falseLit.Variable] {
			continue
		}
		c.propagationCache[falseLit.Variable] = true

		// Check all clauses watching this literal's variable
		watchedClauses := c.watchLists[falseLit.Variable]
		i := 0

		for j := 0; j < len(watchedClauses); j++ {
			wc := watchedClauses[j]

			// Determine which watch this affects
			watchIdx := -1
			if wc.Watch1 < len(wc.Clause.Literals) {
				lit1 := wc.Clause.Literals[wc.Watch1]
				// Check if this literal is now false
				if lit1.Variable == falseLit.Variable && lit1.Negated == falseLit.Negated {
					watchIdx = wc.Watch1
				}
			}

			if watchIdx == -1 && wc.Watch2 >= 0 && wc.Watch2 < len(wc.Clause.Literals) {
				lit2 := wc.Clause.Literals[wc.Watch2]
				if lit2.Variable == falseLit.Variable && lit2.Negated == falseLit.Negated {
					watchIdx = wc.Watch2
				}
			}

			if watchIdx == -1 {
				// This clause is not affected by this literal
				watchedClauses[i] = wc
				i++
				continue
			}

			// Try to find a new literal to watch
			newWatch := c.findNewWatch(wc, watchIdx)
			if newWatch != -1 {
				// Found new watch - update and move to new watch list
				if watchIdx == wc.Watch1 {
					wc.Watch1 = newWatch
				} else {
					wc.Watch2 = newWatch
				}

				// Add to new variable's watch list
				newVar := wc.Clause.Literals[newWatch].Variable
				c.watchLists[newVar] = append(c.watchLists[newVar], wc)
				// Don't add back to current list
			} else {
				// Could not find new watch - check other watched literal
				otherIdx := wc.Watch1
				if watchIdx == wc.Watch1 {
					otherIdx = wc.Watch2
				}

				if otherIdx == -1 || otherIdx >= len(wc.Clause.Literals) {
					// Unit clause became empty - conflict
					c.propagationQueue = c.propagationQueue[:0]
					c.queueHead = 0
					c.propagationCache = make(map[string]bool)
					return wc.Clause
				}

				otherLit := wc.Clause.Literals[otherIdx]
				if c.assignment.IsAssigned(otherLit.Variable) {
					val := c.assignment[otherLit.Variable]
					satisfied := (val && !otherLit.Negated) || (!val && otherLit.Negated)

					if !satisfied {
						// Conflict - both watches are false
						c.propagationQueue = c.propagationQueue[:0]
						c.queueHead = 0
						c.propagationCache = make(map[string]bool)
						return wc.Clause
					}

					// Clause is satisfied - keep in watch list
					watchedClauses[i] = wc
					i++
				} else {
					// Unit propagation - assign the other literal
					value := !otherLit.Negated
					c.assign(otherLit.Variable, value, wc.Clause)
					c.statistics.Propagations++

					// Keep in watch list
					watchedClauses[i] = wc
					i++
				}
			}
		}

		// Update watch list (remove clauses that moved to other lists)
		c.watchLists[falseLit.Variable] = watchedClauses[:i]
	}

	// Clear propagation queue and cache
	c.propagationQueue = c.propagationQueue[:0]
	c.queueHead = 0
	c.propagationCache = make(map[string]bool)
	return nil
}

// Helper methods for advanced CDCL
func (c *CDCLSolver) findNewWatch(wc *WatchedClause, excludeIdx int) int {
	for i, lit := range wc.Clause.Literals {
		if i == wc.Watch1 || i == wc.Watch2 || i == excludeIdx {
			continue
		}

		if !c.assignment.IsAssigned(lit.Variable) {
			return i // Unassigned literal - good watch
		}

		val := c.assignment[lit.Variable]
		if (val && !lit.Negated) || (!val && lit.Negated) {
			return i // Satisfied literal - excellent watch
		}
	}
	return -1
}

func (c *CDCLSolver) assign(variable string, value bool, reason *Clause) {
	c.assignment[variable] = value
	c.trail.Assign(variable, value, c.decisionLevel, reason)
	c.cacheValid = false // Invalidate unassigned cache

	// Track implications for ILB
	if reason != nil && c.ilb != nil {
		c.ilb.levelImplicationCount[c.decisionLevel]++
	}

	// Add to propagation queue
	falseLit := Literal{Variable: variable, Negated: value}
	c.propagationQueue = append(c.propagationQueue, falseLit)
}

func (c *CDCLSolver) learnClause(clause *Clause) {
	clause.Learned = true
	clause.ID = c.cnf.nextID
	c.cnf.nextID++
	c.clauseActivity[clause.ID] = 1.0

	// Track LBD statistics
	c.lbdSum += int64(clause.LBD)
	if clause.Glue {
		c.glueClauseCount++
		c.statistics.GlueClauses = c.glueClauseCount
	}

	// Update LBD distribution
	c.statistics.LBDDistribution[clause.LBD]++

	// Update average LBD
	if c.statistics.LearnedClauses > 0 {
		c.statistics.AvgLBD = float64(c.lbdSum) / float64(c.statistics.LearnedClauses)
	}

	// Add to tiered clause database as a recent clause with protection
	if c.clauseDatabase != nil {
		c.clauseDatabase.AddClause(clause, c.conflicts)
	}

	// Add to watch lists with LBD-aware blocking literal selection
	if len(clause.Literals) >= 2 {
		wc := &WatchedClause{
			Clause: clause,
			Watch1: 0,
			Watch2: 1,
			// Set blocker to highest level literal for better performance
			Blocker: c.selectBestBlocker(clause),
		}

		var1 := clause.Literals[0].Variable
		var2 := clause.Literals[1].Variable
		c.watchLists[var1] = append(c.watchLists[var1], wc)
		c.watchLists[var2] = append(c.watchLists[var2], wc)
	} else if len(clause.Literals) == 1 {
		// Unit clause
		wc := &WatchedClause{
			Clause:  clause,
			Watch1:  0,
			Watch2:  -1,
			Blocker: clause.Literals[0],
		}

		variable := clause.Literals[0].Variable
		c.watchLists[variable] = append(c.watchLists[variable], wc)
	}

	// Update variable activities
	for _, lit := range clause.Literals {
		c.bumpVariableActivity(lit.Variable)
	}
}

// selectBestBlocker selects the best blocking literal based on decision level
func (c *CDCLSolver) selectBestBlocker(clause *Clause) Literal {
	if len(clause.Literals) == 0 {
		return Literal{}
	}

	// Select literal from highest decision level as blocker
	bestLit := clause.Literals[0]
	bestLevel := c.trail.GetLevel(bestLit.Variable)

	for _, lit := range clause.Literals[1:] {
		level := c.trail.GetLevel(lit.Variable)
		if level > bestLevel {
			bestLevel = level
			bestLit = lit
		}
	}

	return bestLit
}

func (c *CDCLSolver) allVariablesAssigned() bool {
	if c.cnf == nil {
		return true
	}

	for _, variable := range c.cnf.Variables {
		if !c.assignment.IsAssigned(variable) {
			return false
		}
	}
	return true
}

func (c *CDCLSolver) chooseDecisionVariable() string {
	if c.cnf == nil {
		return ""
	}

	// Use cached unassigned variables if available and valid
	if !c.cacheValid {
		c.unassignedCache = c.unassignedCache[:0] // Reuse slice capacity
		for _, variable := range c.cnf.Variables {
			if !c.assignment.IsAssigned(variable) {
				c.unassignedCache = append(c.unassignedCache, variable)
			}
		}
		c.cacheValid = true
	}

	if len(c.unassignedCache) == 0 {
		return ""
	}

	// Use heuristic with proper error handling
	if c.heuristic != nil {
		chosen := c.heuristic.ChooseVariable(c.unassignedCache, c.assignment)
		// Validate that chosen variable is actually unassigned
		if chosen != "" && !c.assignment.IsAssigned(chosen) {
			return chosen
		}
	}

	// Fallback to first unassigned with bounds check
	if len(c.unassignedCache) > 0 {
		return c.unassignedCache[0]
	}

	return ""
}

func (c *CDCLSolver) choosePolarity(variable string) bool {
	// Use enhanced polarity if VSIDS heuristic supports it
	if vsids, ok := c.heuristic.(*VSIDSHeuristic); ok {
		return vsids.GetPreferredPolarity(variable)
	}

	return true // Fallback
}

func (c *CDCLSolver) backtrack(level int) {
	// Use the trail to backtrack
	unassignedVars := c.trail.Backtrack(level)

	// Remove assignments
	for _, variable := range unassignedVars {
		delete(c.assignment, variable)
	}

	c.cacheValid = false // Invalidate unassigned cache
	c.decisionLevel = level
}

func (c *CDCLSolver) restart() {
	c.assignment = make(Assignment)
	c.trail.Clear()
	c.decisionLevel = 0
	c.cacheValid = false // Invalidate unassigned cache
	c.restartStrategy.OnRestart()
}

func (c *CDCLSolver) deleteClauses() {
	if c.clauseDatabase == nil {
		return
	}

	// Ensure recent clauses are promoted before deletion decisions
	c.clauseDatabase.PromoteFromRecent(c.conflicts)

	over := c.clauseDatabase.Size() - c.maxLearnedSize
	if over <= 0 {
		return
	}

	// If the deletion policy provides tier-aware selection, use it
	type tierAware interface {
		GetDeletionCandidates(db *ClauseDatabase, stats SolverStatistics) []*Clause
	}

	var candidates []*Clause
	if da, ok := c.deletionPolicy.(tierAware); ok {
		candidates = da.GetDeletionCandidates(c.clauseDatabase, c.statistics)
	} else {
		// Fallback: pick from local (aggressive) then mid (careful) using legacy ShouldDelete
		_, mid, local, _ := c.clauseDatabase.GetTierSlices()

		pick := func(src []*Clause, need int) {
			for _, cl := range src {
				if need == 0 {
					break
				}
				if c.deletionPolicy.ShouldDelete(cl, c.statistics) {
					candidates = append(candidates, cl)
					need--
				}
			}
		}

		need := over
		pick(local, need)
		if len(candidates) < over {
			pick(mid, over-len(candidates))
		}
	}

	// Trim to exactly what is needed
	if len(candidates) > over {
		candidates = candidates[:over]
	}

	// Remove candidates from watch lists and the database
	deleted := 0
	for _, cl := range candidates {
		if cl == nil {
			continue
		}
		c.removeFromWatchLists(cl)
		if c.clauseDatabase.RemoveClause(cl) {
			deleted++
		}
	}

	c.statistics.DeletedClauses += int64(deleted)
}

func (c *CDCLSolver) initializeWatchLists() error {
	c.watchLists = make(map[string][]*WatchedClause)

	// Set up watch lists for all clauses
	for _, clause := range c.cnf.Clauses {
		if len(clause.Literals) >= 2 {
			// Watch first two literals
			wc := &WatchedClause{
				Clause: clause,
				Watch1: 0,
				Watch2: 1,
			}

			var1 := clause.Literals[0].Variable
			var2 := clause.Literals[1].Variable
			c.watchLists[var1] = append(c.watchLists[var1], wc)
			c.watchLists[var2] = append(c.watchLists[var2], wc)
		} else if len(clause.Literals) == 1 {
			// Unit clause - create watched clause with single watch
			wc := &WatchedClause{
				Clause: clause,
				Watch1: 0,
				Watch2: -1, // No second watch
			}

			variable := clause.Literals[0].Variable
			c.watchLists[variable] = append(c.watchLists[variable], wc)
		}
	}

	return nil
}

func (c *CDCLSolver) initializeHeuristics() {
	// Initialize variable activities
	for _, variable := range c.cnf.Variables {
		c.variableActivity[variable] = 0.0
	}
}

func (c *CDCLSolver) removeFromWatchLists(clause *Clause) {
	for _, lit := range clause.Literals {
		if watchedClauses, exists := c.watchLists[lit.Variable]; exists {
			for i, watchedClause := range watchedClauses {
				if watchedClause.Clause.ID == clause.ID {
					c.watchLists[lit.Variable] = append(watchedClauses[:i], watchedClauses[i+1:]...)
					break
				}
			}
		}
	}
}

// Variable activity management (VSIDS)
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
	c.cacheValid = false // Invalidate unassigned cache

	// Update watch lists for new clause (complete implementation)
	if len(clause.Literals) >= 2 {
		wc := &WatchedClause{
			Clause: clause,
			Watch1: 0,
			Watch2: 1,
		}

		var1 := clause.Literals[0].Variable
		var2 := clause.Literals[1].Variable
		c.watchLists[var1] = append(c.watchLists[var1], wc)
		c.watchLists[var2] = append(c.watchLists[var2], wc)
	} else if len(clause.Literals) == 1 {
		// Unit clause
		wc := &WatchedClause{
			Clause: clause,
			Watch1: 0,
			Watch2: -1,
		}

		variable := clause.Literals[0].Variable
		c.watchLists[variable] = append(c.watchLists[variable], wc)
	}

	return nil
}

func (c *CDCLSolver) GetStatistics() SolverStatistics {
	return c.statistics
}

func (c *CDCLSolver) Reset() {
	c.statistics = SolverStatistics{LBDDistribution: make(map[int]int64)}
	c.assignment = make(Assignment)
	c.trail.Clear()
	c.watchLists = make(map[string][]*WatchedClause)
	c.decisionLevel = 0
	c.variableActivity = make(map[string]float64)
	c.conflicts = 0
	c.propagationQueue = make([]Literal, 0)
	c.queueHead = 0
	c.lbdSum = 0
	c.glueClauseCount = 0
	c.unassignedCache = make([]string, 0)
	c.cacheValid = false
	c.propagationCache = make(map[string]bool)

	// Reset inprocessing tracking
	c.lastInprocess = 0
	c.inprocessGap = 4000
	c.lastInprocessReduction = 0
	c.lastInprocessCostNs = 0

	// Reset XOR-specific statistics
	c.xorPropagations = 0
	c.xorConflicts = 0
	c.gaussianRuns = 0

	// Reset ILB state
	if c.ilb != nil {
		c.ilb.reimplicationQueue = c.ilb.reimplicationQueue[:0]
		c.ilb.lazyBacktrackLevel = 0

		for k := range c.ilb.reimplicationCache {
			delete(c.ilb.reimplicationCache, k)
		}
		for k := range c.ilb.levelImplicationCount {
			delete(c.ilb.levelImplicationCount, k)
		}
		c.ilb.lastReimplicationCost = 0
	}

	// Reset chronological stats
	if c.chronologicalStats != nil {
		c.chronologicalStats.chronologicalAttempts = 0
		c.chronologicalStats.chronologicalSuccesses = 0
		c.chronologicalStats.recentSuccessRate = 0.5
		c.chronologicalStats.windowIndex = 0
		for i := range c.chronologicalStats.successWindow {
			c.chronologicalStats.successWindow[i] = false
		}
	}

	// Reset components
	if c.heuristic != nil {
		c.heuristic.Reset()
	}
	if c.restartStrategy != nil {
		c.restartStrategy.Reset()
	}
	if c.deletionPolicy != nil {
		c.deletionPolicy.Reset()
	}
	if c.analyzer != nil {
		c.analyzer.Reset()
	}
}
