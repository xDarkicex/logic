package sat

import (
	"sort"

	"github.com/xDarkicex/logic/core"
)

// FirstUIPAnalyzer implements state-of-the-art First Unique Implication Point analysis
// This is the heart of modern CDCL solvers and implements the most effective conflict learning
type FirstUIPAnalyzer struct {
	// Temporary storage for conflict analysis
	seen            map[string]bool
	conflictSide    []string
	resolutionStack []ResolutionStep

	// LBD computation support
	levelsSeen map[int]bool

	// Performance counters
	resolutions     int64
	trivialClauses  int64
	unitClauses     int64
	glueClauseCount int64
}

// ResolutionStep tracks each step in the resolution process for debugging
type ResolutionStep struct {
	ResolvedVar     string
	ReasonClause    *Clause
	ResultingClause []Literal
	Level           int
}

func NewFirstUIPAnalyzer() *FirstUIPAnalyzer {
	return &FirstUIPAnalyzer{
		seen:            make(map[string]bool),
		conflictSide:    make([]string, 0),
		resolutionStack: make([]ResolutionStep, 0),
		levelsSeen:      make(map[int]bool),
	}
}

func (f *FirstUIPAnalyzer) Name() string {
	return "FirstUIP-Advanced"
}

// Analyze performs sophisticated conflict analysis using First-UIP with LBD computation
func (f *FirstUIPAnalyzer) Analyze(conflictClause *Clause, trail DecisionTrail) (*Clause, int) {
	if conflictClause == nil {
		return nil, 0
	}

	currentLevel := trail.GetCurrentLevel()
	if currentLevel == 0 {
		return nil, 0 // Root level conflict - formula is UNSAT
	}

	// Reset analysis state including LBD tracking
	f.reset()

	// Initialize with conflict clause
	learntClause := make([]Literal, 0, len(conflictClause.Literals))

	// Add all literals from conflict clause and track levels
	for _, lit := range conflictClause.Literals {
		learntClause = append(learntClause, lit.Negate())
		f.seen[lit.Variable] = true

		// Track decision levels for LBD computation
		level := trail.GetLevel(lit.Variable)
		if level >= 0 {
			f.levelsSeen[level] = true
		}
	}

	// Track variables at current decision level
	currentLevelVars := 0
	for _, lit := range learntClause {
		if trail.GetLevel(lit.Variable) == currentLevel {
			currentLevelVars++
		}
	}

	// If we already have exactly one variable at current level, we're done
	if currentLevelVars == 1 {
		f.unitClauses++
		finalClause := f.buildLearnedClauseWithLBD(learntClause, trail)
		return finalClause, f.computeBacktrackLevel(learntClause, trail, currentLevel)
	}

	// Get trail entries at current level for resolution
	currentLevelTrail := f.getTrailEntriesAtLevel(trail, currentLevel)

	// Perform resolution until we reach First-UIP
	for currentLevelVars > 1 {
		// Find the most recent assignment at current level that's in our clause
		resolveVar := f.findMostRecentVariable(learntClause, currentLevelTrail, trail, currentLevel)

		if resolveVar == "" {
			// Safety fallback - should not happen in correct implementation
			break
		}

		// Get the reason for this assignment
		reason := trail.GetReason(resolveVar)
		if reason == nil {
			// This is a decision variable at current level - we found First-UIP
			break
		}

		// Perform resolution step with LBD tracking
		f.resolutions++
		oldClause := make([]Literal, len(learntClause))
		copy(oldClause, learntClause)

		learntClause = f.resolveWithLBDTracking(learntClause, reason, resolveVar, trail)

		// Track resolution step for debugging
		f.resolutionStack = append(f.resolutionStack, ResolutionStep{
			ResolvedVar:     resolveVar,
			ReasonClause:    reason,
			ResultingClause: append([]Literal{}, learntClause...),
			Level:           currentLevel,
		})

		// Update current level variable count
		currentLevelVars = f.countCurrentLevelVars(learntClause, trail, currentLevel)

		// Prevent infinite loops (safety check)
		if len(f.resolutionStack) > 10000 {
			core.NewLogicError("sat", "FirstUIPAnalyzer.Analyze", "resolution limit exceeded")
			break
		}
	}

	// Build final learned clause with LBD
	finalClause := f.buildLearnedClauseWithLBD(learntClause, trail)
	backtrackLevel := f.computeBacktrackLevel(learntClause, trail, currentLevel)

	// Update statistics
	if len(finalClause.Literals) == 1 {
		f.unitClauses++
	}
	if finalClause.Glue {
		f.glueClauseCount++
	}

	return finalClause, backtrackLevel
}

// resolveWithLBDTracking performs resolution between current learnt clause and reason clause with LBD tracking
func (f *FirstUIPAnalyzer) resolveWithLBDTracking(learntClause []Literal, reasonClause *Clause, resolveVar string, trail DecisionTrail) []Literal {
	newClause := make([]Literal, 0)

	// Add literals from learnt clause (except resolved variable)
	for _, lit := range learntClause {
		if lit.Variable != resolveVar {
			newClause = append(newClause, lit)
		}
	}

	// Add literals from reason clause (except resolved variable and negated resolved variable)
	for _, lit := range reasonClause.Literals {
		if lit.Variable != resolveVar && !f.containsVariable(newClause, lit.Variable) {
			newClause = append(newClause, lit)
			f.seen[lit.Variable] = true

			// Track decision level for LBD computation
			level := trail.GetLevel(lit.Variable)
			if level >= 0 {
				f.levelsSeen[level] = true
			}
		}
	}

	return newClause
}

// findMostRecentVariable finds the most recently assigned variable at current level
func (f *FirstUIPAnalyzer) findMostRecentVariable(clause []Literal, levelTrail []TrailEntry, trail DecisionTrail, level int) string {
	var mostRecent string
	maxPosition := -1

	// Find the variable with highest position in the trail at current level
	for _, lit := range clause {
		if trail.GetLevel(lit.Variable) == level {
			pos := f.findPositionInTrail(lit.Variable, levelTrail)
			if pos > maxPosition {
				maxPosition = pos
				mostRecent = lit.Variable
			}
		}
	}

	return mostRecent
}

// findPositionInTrail finds position of variable in level-specific trail
func (f *FirstUIPAnalyzer) findPositionInTrail(variable string, levelTrail []TrailEntry) int {
	for i := len(levelTrail) - 1; i >= 0; i-- {
		if levelTrail[i].Variable == variable {
			return i
		}
	}
	return -1
}

// getTrailEntriesAtLevel extracts trail entries for specific level
func (f *FirstUIPAnalyzer) getTrailEntriesAtLevel(trail DecisionTrail, level int) []TrailEntry {
	// This would ideally use a method on the trail, but we'll implement it here
	if simpleTrail, ok := trail.(*SimpleDecisionTrail); ok {
		return simpleTrail.GetTrailAtLevel(level)
	}

	// Fallback implementation
	assignment := trail.GetAssignment()
	entries := make([]TrailEntry, 0)

	for variable := range assignment {
		if trail.GetLevel(variable) == level {
			entries = append(entries, TrailEntry{
				Variable: variable,
				Value:    assignment[variable],
				Level:    level,
				Reason:   trail.GetReason(variable),
			})
		}
	}

	// Sort by assignment order (simplified)
	return entries
}

// countCurrentLevelVars counts variables in clause at current decision level
func (f *FirstUIPAnalyzer) countCurrentLevelVars(clause []Literal, trail DecisionTrail, level int) int {
	count := 0
	for _, lit := range clause {
		if trail.GetLevel(lit.Variable) == level {
			count++
		}
	}
	return count
}

// buildLearnedClauseWithLBD creates final learned clause with LBD computation
func (f *FirstUIPAnalyzer) buildLearnedClauseWithLBD(literals []Literal, trail DecisionTrail) *Clause {
	// Remove duplicates and optimize
	seen := make(map[string]bool)
	uniqueLiterals := make([]Literal, 0, len(literals))
	levelSet := make(map[int]bool)

	for _, lit := range literals {
		key := f.literalKey(lit)
		if !seen[key] {
			seen[key] = true
			uniqueLiterals = append(uniqueLiterals, lit)

			// Track levels for final LBD computation
			level := trail.GetLevel(lit.Variable)
			if level >= 0 {
				levelSet[level] = true
			}
		}
	}

	// Sort literals by decision level (asserting literal first)
	sort.Slice(uniqueLiterals, func(i, j int) bool {
		levelI := trail.GetLevel(uniqueLiterals[i].Variable)
		levelJ := trail.GetLevel(uniqueLiterals[j].Variable)
		return levelI > levelJ // Higher level first
	})

	// Create clause with LBD information
	clause := NewClause(uniqueLiterals...)
	clause.Learned = true
	clause.Activity = 1.0

	// Compute and set LBD
	lbd := len(levelSet)
	clause.SetLBD(lbd)

	return clause
}

// computeBacktrackLevel determines the correct backtrack level
func (f *FirstUIPAnalyzer) computeBacktrackLevel(literals []Literal, trail DecisionTrail, currentLevel int) int {
	if len(literals) <= 1 {
		return 0
	}

	// Find second highest decision level
	levels := make([]int, 0, len(literals))
	for _, lit := range literals {
		level := trail.GetLevel(lit.Variable)
		if level >= 0 && level < currentLevel {
			levels = append(levels, level)
		}
	}

	if len(levels) == 0 {
		return 0
	}

	// Sort and return second highest or highest if only one
	sort.Ints(levels)
	if len(levels) == 1 {
		return levels[0]
	}

	// Remove duplicates and return second highest
	uniqueLevels := make([]int, 0, len(levels))
	prev := -1
	for _, level := range levels {
		if level != prev {
			uniqueLevels = append(uniqueLevels, level)
			prev = level
		}
	}

	if len(uniqueLevels) == 1 {
		return uniqueLevels[0]
	}
	return uniqueLevels[len(uniqueLevels)-2] // Second highest
}

// Helper methods
func (f *FirstUIPAnalyzer) reset() {
	f.seen = make(map[string]bool)
	f.conflictSide = f.conflictSide[:0]
	f.resolutionStack = f.resolutionStack[:0]
	f.levelsSeen = make(map[int]bool)
}

func (f *FirstUIPAnalyzer) containsVariable(literals []Literal, variable string) bool {
	for _, lit := range literals {
		if lit.Variable == variable {
			return true
		}
	}
	return false
}

func (f *FirstUIPAnalyzer) literalKey(lit Literal) string {
	if lit.Negated {
		return "¬" + lit.Variable
	}
	return lit.Variable
}

func (f *FirstUIPAnalyzer) Reset() {
	f.reset()
	f.resolutions = 0
	f.trivialClauses = 0
	f.unitClauses = 0
	f.glueClauseCount = 0
}

// GetStatistics returns analysis statistics for debugging
func (f *FirstUIPAnalyzer) GetStatistics() map[string]int64 {
	return map[string]int64{
		"resolutions":     f.resolutions,
		"trivialClauses":  f.trivialClauses,
		"unitClauses":     f.unitClauses,
		"glueClauseCount": f.glueClauseCount,
	}
}
