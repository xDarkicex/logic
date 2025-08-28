package sat

import (
	"sort"
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

	// Learned clause management
	learnedClauses []*Clause
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
}

// WatchedClause represents a clause with two-watched literals
type WatchedClause struct {
	Clause  *Clause
	Watch1  int     // Index of first watched literal
	Watch2  int     // Index of second watched literal
	Blocker Literal // Blocking literal for optimization
}

// CDCLConfig holds configuration for CDCL solver
type CDCLConfig struct {
	Heuristic        Heuristic
	RestartStrategy  RestartStrategy
	DeletionPolicy   ClauseDeletionPolicy
	ConflictAnalyzer ConflictAnalyzer
	MaxLearnedSize   int
	Trail            DecisionTrail
	Preprocessor     *SATPreprocessor
	ConflictLimit    int64
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
		learnedClauses:   make([]*Clause, 0),
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
	}

	// Initialize enhanced components by default (now the base versions include all enhancements)
	solver.heuristic = NewVSIDSHeuristic()             // Now includes LRB, polarity, anti-aging
	solver.restartStrategy = NewLubyRestartStrategy()  // Now hybrid Luby+Glucose
	solver.deletionPolicy = NewActivityBasedDeletion() // Now LBD-aware
	solver.analyzer = NewFirstUIPAnalyzer()
	solver.preprocessor = NewSATPreprocessor()

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
	if config.Trail != nil {
		solver.trail = config.Trail
	}
	if config.Preprocessor != nil {
		solver.preprocessor = config.Preprocessor
	}
	if config.ConflictLimit > 0 {
		solver.conflictLimit = config.ConflictLimit
	}

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

// SolveWithTimeout solves with timeout using advanced CDCL algorithm
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

	// Setup timeout
	var timeoutChan <-chan time.Time
	if timeout > 0 {
		timeoutChan = time.After(timeout)
	}

	// Main CDCL loop with advanced propagation
	for c.conflicts < c.conflictLimit {
		select {
		case <-timeoutChan:
			return &SolverResult{
				Error:      core.NewLogicError("sat", "CDCLSolver.SolveWithTimeout", "timeout exceeded"),
				Statistics: c.statistics,
			}
		default:
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

	c.learnedClauses = append(c.learnedClauses, clause)

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
		// Fallback if heuristic returns invalid choice
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
	// Sort by activity (lower activity = more likely to delete)
	sort.Slice(c.learnedClauses, func(i, j int) bool {
		return c.learnedClauses[i].Activity < c.learnedClauses[j].Activity
	})

	// Delete clauses based on deletion policy
	deleteCount := 0
	for i := 0; i < len(c.learnedClauses)/2; i++ {
		if c.deletionPolicy.ShouldDelete(c.learnedClauses[i], c.statistics) {
			c.removeFromWatchLists(c.learnedClauses[i])
			deleteCount++
		}
	}

	// Keep the more active clauses
	c.learnedClauses = c.learnedClauses[len(c.learnedClauses)/2:]
	c.statistics.DeletedClauses += int64(deleteCount)
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
	c.learnedClauses = make([]*Clause, 0)
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
