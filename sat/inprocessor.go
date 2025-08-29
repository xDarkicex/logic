package sat

import (
	"time"
)

// ModernInprocessor implements state-of-the-art inprocessing techniques
type ModernInprocessor struct {
	// Core components (will be implemented in subsequent steps)
	vivifier   *ClauseVivifier
	subsumer   *InprocessSubsumption
	eliminator *BoundedVariableElimination
	prober     *FailedLiteralProber

	// Configuration
	config     InprocessConfig
	statistics InprocessStatistics

	// State tracking
	lastInprocess int64 // Last conflict count when inprocessing ran
	inprocessGap  int64 // How often to run inprocessing

	// Performance tracking
	startTime time.Time
}

// NewModernInprocessor creates a new modern inprocessor with default settings
func NewModernInprocessor() *ModernInprocessor {
	return &ModernInprocessor{
		// Initialize real components
		vivifier:   NewClauseVivifier(),
		subsumer:   NewInprocessSubsumption(),
		eliminator: NewBoundedVariableElimination(), // Now real!
		prober:     NewFailedLiteralProber(),        // Now real!

		config:       DefaultInprocessConfig(),
		statistics:   InprocessStatistics{},
		inprocessGap: 4000,
	}
}

// NewModernInprocessorWithConfig creates inprocessor with custom configuration
func NewModernInprocessorWithConfig(config InprocessConfig) *ModernInprocessor {
	inprocessor := NewModernInprocessor()
	inprocessor.config = config
	inprocessor.inprocessGap = config.InprocessGap
	return inprocessor
}

// Name returns the inprocessor name
func (m *ModernInprocessor) Name() string {
	return "ModernInprocessor"
}

// Inprocess performs comprehensive inprocessing at decision level 0
func (m *ModernInprocessor) Inprocess(cnf *CNF, assignment Assignment, level int) (*InprocessResult, error) {
	// Safety check: only run at root level to avoid corrupting the trail
	if level != 0 {
		return &InprocessResult{}, nil
	}

	m.startTime = time.Now()
	result := &InprocessResult{}

	// Track that we ran inprocessing
	m.statistics.InprocessRuns++

	// Phase 1: Clause vivification (when implemented)
	if m.config.EnableVivification && m.vivifier != nil {
		startTime := time.Now()
		vivified := m.VivifyClauses(cnf.Clauses, assignment)
		m.statistics.TimeInVivification += time.Since(startTime).Nanoseconds()
		result.VivificationsApplied = vivified
		result.ClausesStrengthened += vivified
	}

	// Phase 2: Subsumption and strengthening (when implemented)
	if m.config.EnableSubsumption && m.subsumer != nil {
		startTime := time.Now()
		subsumed := m.SubsumeAndStrengthen(cnf)
		m.statistics.TimeInSubsumption += time.Since(startTime).Nanoseconds()
		result.SubsumptionsFound = subsumed
		result.ClausesRemoved += subsumed
	}

	// Phase 3: Variable elimination (when implemented)
	if m.config.EnableVariableElim && m.eliminator != nil {
		startTime := time.Now()
		eliminated := m.EliminateVariables(cnf.Variables, cnf)
		m.statistics.TimeInVariableElim += time.Since(startTime).Nanoseconds()
		result.VariablesEliminated = eliminated
	}

	// Phase 4: Failed literal probing (when implemented)
	if m.config.EnableFailedLitProbing && m.prober != nil {
		startTime := time.Now()
		// Create candidate literals from unassigned variables
		candidates := make([]Literal, 0, len(cnf.Variables))
		for _, variable := range cnf.Variables {
			if !assignment.IsAssigned(variable) {
				candidates = append(candidates, Literal{Variable: variable, Negated: false})
				candidates = append(candidates, Literal{Variable: variable, Negated: true})
			}
		}
		failedLiterals := m.ProbeFailedLiterals(candidates, cnf)
		m.statistics.TimeInFailedLitProbing += time.Since(startTime).Nanoseconds()
		result.FailedLiteralsFound = len(failedLiterals)
		result.UnitsLearned += len(failedLiterals)
	}

	// Update total statistics
	m.statistics.TotalInprocessTime += time.Since(m.startTime).Nanoseconds()

	// Determine if formula was significantly reduced
	totalChanges := result.ClausesRemoved + result.ClausesStrengthened + result.VariablesEliminated + result.UnitsLearned
	result.FormulaReduced = totalChanges > 0

	return result, nil
}

// VivifyClauses applies clause vivification to strengthen clauses
func (m *ModernInprocessor) VivifyClauses(clauses []*Clause, assignment Assignment) int {
	if m.vivifier == nil {
		return 0
	}

	vivifiedCount := 0

	// Only vivify learned clauses and long original clauses
	for _, clause := range clauses {
		if len(clause.Literals) <= 1 {
			continue // Skip unit and empty clauses
		}

		// Prioritize learned clauses or long original clauses
		if clause.Learned || len(clause.Literals) > 5 {
			if m.vivifier.VivifyClause(clause, assignment) {
				vivifiedCount++
				m.statistics.ClausesVivified++
			}
		}
	}

	return vivifiedCount
}

// EliminateVariables applies bounded variable elimination
func (m *ModernInprocessor) EliminateVariables(variables []string, cnf *CNF) int {
	if m.eliminator == nil {
		return 0
	}

	// Create a temporary assignment (empty for inprocessing)
	assignment := make(Assignment)

	eliminatedCount := m.eliminator.EliminateVariables(cnf, assignment)

	// Update statistics
	m.statistics.VariablesEliminated += int64(eliminatedCount)

	return eliminatedCount
}

// ProbeFailedLiterals applies failed literal probing
func (m *ModernInprocessor) ProbeFailedLiterals(literals []Literal, cnf *CNF) []Literal {
	if m.prober == nil {
		return []Literal{}
	}

	// Create a temporary assignment for probing
	assignment := make(Assignment)

	return m.prober.ProbeFailedLiterals(cnf, assignment)
}

// SubsumeAndStrengthen applies subsumption and strengthening to reduce CNF size
func (m *ModernInprocessor) SubsumeAndStrengthen(cnf *CNF) int {
	if m.subsumer == nil {
		return 0
	}

	subsumptionCount := m.subsumer.FindAndRemoveSubsumed(cnf)

	// Update statistics
	m.statistics.ClausesSubsumed += int64(subsumptionCount)

	return subsumptionCount
}

// Configuration and lifecycle methods

// Configure updates the inprocessor configuration
func (m *ModernInprocessor) Configure(config InprocessConfig) {
	m.config = config
	m.inprocessGap = config.InprocessGap
}

// GetStatistics returns current inprocessing statistics
func (m *ModernInprocessor) GetStatistics() InprocessStatistics {
	return m.statistics
}

// Reset clears all inprocessor state
func (m *ModernInprocessor) Reset() {
	m.statistics = InprocessStatistics{}
	m.lastInprocess = 0
}

// Helper methods

// ShouldRunInprocessing determines if inprocessing should run based on config
func (m *ModernInprocessor) ShouldRunInprocessing(conflicts int64) bool {
	return conflicts > 0 &&
		conflicts >= m.lastInprocess+m.inprocessGap
}

// OnInprocessingCompleted updates tracking after inprocessing runs
func (m *ModernInprocessor) OnInprocessingCompleted(conflicts int64) {
	m.lastInprocess = conflicts
}

// ClauseVivifier implements modern clause vivification with optimizations
type ClauseVivifier struct {
	// Configuration
	maxClauseSize int
	maxTries      int
	useUnitProp   bool
	useSATSolver  bool

	// Statistics
	vivified     int64
	strengthened int64

	// Temporary solver for checking clause satisfiability
	tempSolver Solver

	// Performance optimizations
	literalCache   map[string]bool
	candidateCache []Literal
}

// NewClauseVivifier creates a new clause vivifier with default settings
func NewClauseVivifier() *ClauseVivifier {
	return &ClauseVivifier{
		maxClauseSize: 20, // Only vivify clauses up to this size
		maxTries:      3,  // Maximum vivification attempts per clause
		useUnitProp:   true,
		useSATSolver:  false, // Use lightweight unit propagation by default

		tempSolver:     NewDPLLSolver(), // Use DPLL for temp solving
		literalCache:   make(map[string]bool),
		candidateCache: make([]Literal, 0, 20),
	}
}

// NewClauseVivifierWithConfig creates vivifier with custom settings
func NewClauseVivifierWithConfig(maxSize, maxTries int, useSolver bool) *ClauseVivifier {
	v := NewClauseVivifier()
	v.maxClauseSize = maxSize
	v.maxTries = maxTries
	v.useSATSolver = useSolver
	if useSolver {
		v.tempSolver = NewCDCLSolver() // Use CDCL for more complex cases
	}
	return v
}

// VivifyClause attempts to strengthen a single clause by removing unnecessary literals
func (cv *ClauseVivifier) VivifyClause(clause *Clause, assignment Assignment) bool {
	// Skip large clauses or already minimal clauses
	if len(clause.Literals) <= 1 || len(clause.Literals) > cv.maxClauseSize {
		return false
	}

	// Skip if clause is already satisfied
	if assignment.Satisfies(clause) {
		return false
	}

	originalSize := len(clause.Literals)
	strengthened := false

	for attempt := 0; attempt < cv.maxTries; attempt++ {
		// Try to remove each literal and see if clause is still satisfiable
		for i := 0; i < len(clause.Literals); i++ {
			if cv.canRemoveLiteral(clause, i, assignment) {
				// Remove the literal
				clause.Literals = append(clause.Literals[:i], clause.Literals[i+1:]...)
				strengthened = true
				i-- // Adjust index since we removed an element

				// Stop if clause becomes unit
				if len(clause.Literals) <= 1 {
					break
				}
			}
		}

		// If no changes in this attempt, stop trying
		if len(clause.Literals) == originalSize {
			break
		}
		originalSize = len(clause.Literals)
	}

	if strengthened {
		cv.strengthened++
		cv.vivified++
	}

	return strengthened
}

// canRemoveLiteral checks if a literal can be removed from a clause
// This is the core of vivification - we check if the clause remains "strong enough"
// without this literal
func (cv *ClauseVivifier) canRemoveLiteral(clause *Clause, literalIndex int, assignment Assignment) bool {
	if literalIndex >= len(clause.Literals) {
		return false
	}

	// Create test clause without this literal
	testLiterals := make([]Literal, 0, len(clause.Literals)-1)
	for i, lit := range clause.Literals {
		if i != literalIndex {
			testLiterals = append(testLiterals, lit)
		}
	}

	if len(testLiterals) == 0 {
		return false // Can't remove the last literal
	}

	// Check if the reduced clause can be falsified
	// If it can't be falsified, then the removed literal was redundant
	return cv.testClauseStrength(testLiterals, assignment)
}

// testClauseStrength checks if a reduced clause is "strong enough"
// Returns true if the literal can be removed (clause remains strong)
func (cv *ClauseVivifier) testClauseStrength(literals []Literal, assignment Assignment) bool {
	if cv.useUnitProp {
		return cv.testWithUnitPropagation(literals, assignment)
	} else {
		return cv.testWithSATSolver(literals, assignment)
	}
}

// testWithUnitPropagation uses lightweight unit propagation to test clause strength
func (cv *ClauseVivifier) testWithUnitPropagation(literals []Literal, assignment Assignment) bool {
	// Create a test CNF with the negated literals (to try to falsify the clause)
	testCNF := NewCNF()

	// Add unit clauses that negate each literal in the test clause
	for _, lit := range literals {
		negatedLit := lit.Negate()
		testCNF.AddClause(NewClause(negatedLit))
	}

	// Add any existing assignments as constraints
	for variable, value := range assignment {
		constraintLit := Literal{Variable: variable, Negated: !value}
		testCNF.AddClause(NewClause(constraintLit))
	}

	// Perform unit propagation to see if we get a contradiction
	testAssignment := make(Assignment)

	// Apply existing assignments
	for variable, value := range assignment {
		testAssignment[variable] = value
	}

	// Try to propagate the negated literals
	conflict := cv.performUnitPropagation(testCNF, testAssignment)

	// If we get a conflict, the original clause was necessary
	// If no conflict, the clause is implied by other constraints and can be strengthened
	return !conflict
}

// performUnitPropagation performs unit propagation and returns true if conflict found
func (cv *ClauseVivifier) performUnitPropagation(cnf *CNF, assignment Assignment) bool {
	changed := true
	iterations := 0

	for changed && iterations < 100 { // Limit iterations to prevent infinite loops
		changed = false
		iterations++

		for _, clause := range cnf.Clauses {
			// Skip satisfied clauses
			if assignment.Satisfies(clause) {
				continue
			}

			// Check for conflict
			if assignment.ConflictsWith(clause) {
				return true // Conflict found
			}

			// Check for unit clause
			unassignedLiterals := cv.getUnassignedLiterals(clause, assignment)
			if len(unassignedLiterals) == 1 {
				// Unit clause found - propagate
				lit := unassignedLiterals[0]
				value := !lit.Negated

				// Check for immediate conflict
				if assignment.IsAssigned(lit.Variable) && assignment[lit.Variable] != value {
					return true // Conflict found
				}

				assignment[lit.Variable] = value
				changed = true
			}
		}
	}

	return false // No conflict found
}

// testWithSATSolver uses a full SAT solver to test clause strength (more expensive but complete)
func (cv *ClauseVivifier) testWithSATSolver(literals []Literal, assignment Assignment) bool {
	// Create a test CNF that tries to falsify all literals in the clause
	testCNF := NewCNF()

	// Add unit clauses that negate each literal
	for _, lit := range literals {
		negatedLit := lit.Negate()
		testCNF.AddClause(NewClause(negatedLit))
	}

	// Add assignment constraints
	for variable, value := range assignment {
		constraintLit := Literal{Variable: variable, Negated: !value}
		testCNF.AddClause(NewClause(constraintLit))
	}

	// Try to solve - if unsatisfiable, the original clause was necessary
	result := cv.tempSolver.Solve(testCNF)
	cv.tempSolver.Reset() // Clean up for next use

	// If unsatisfiable, the clause cannot be strengthened
	// If satisfiable, the clause can be strengthened
	return result.Satisfiable
}

// getUnassignedLiterals returns literals in clause that are not yet assigned
func (cv *ClauseVivifier) getUnassignedLiterals(clause *Clause, assignment Assignment) []Literal {
	cv.candidateCache = cv.candidateCache[:0] // Reuse slice

	for _, lit := range clause.Literals {
		if !assignment.IsAssigned(lit.Variable) {
			cv.candidateCache = append(cv.candidateCache, lit)
		}
	}

	return cv.candidateCache
}

// GetStatistics returns vivification statistics
func (cv *ClauseVivifier) GetStatistics() map[string]int64 {
	return map[string]int64{
		"vivified":     cv.vivified,
		"strengthened": cv.strengthened,
	}
}

// Reset clears vivifier statistics
func (cv *ClauseVivifier) Reset() {
	cv.vivified = 0
	cv.strengthened = 0
	cv.literalCache = make(map[string]bool)
	cv.candidateCache = cv.candidateCache[:0]
	if cv.tempSolver != nil {
		cv.tempSolver.Reset()
	}
}

// InprocessSubsumption implements efficient subsumption and self-subsumption during search
type InprocessSubsumption struct {
	// Configuration
	maxClauseSize         int
	maxSubsumptionTries   int
	enableSelfSubsumption bool

	// Statistics
	subsumptions        int64
	selfSubsumptions    int64
	strengthenedClauses int64

	// Performance optimizations
	literalOccurrence     map[string][]*Clause // Literal -> clauses containing it
	subsumptionCandidates []SubsumptionPair
	processed             map[int]bool // Clause ID -> processed
}

// SubsumptionPair represents a potential subsumption relationship
type SubsumptionPair struct {
	Subsumer *Clause
	Subsumed *Clause
	Strength int // Number of literals difference (lower = stronger subsumption)
}

// NewInprocessSubsumption creates a new subsumption engine
func NewInprocessSubsumption() *InprocessSubsumption {
	return &InprocessSubsumption{
		maxClauseSize:         30,   // Only check clauses up to this size
		maxSubsumptionTries:   1000, // Limit subsumption attempts
		enableSelfSubsumption: true,

		literalOccurrence:     make(map[string][]*Clause),
		subsumptionCandidates: make([]SubsumptionPair, 0, 100),
		processed:             make(map[int]bool),
	}
}

// NewInprocessSubsumptionWithConfig creates subsumption engine with custom settings
func NewInprocessSubsumptionWithConfig(maxSize, maxTries int, enableSelf bool) *InprocessSubsumption {
	s := NewInprocessSubsumption()
	s.maxClauseSize = maxSize
	s.maxSubsumptionTries = maxTries
	s.enableSelfSubsumption = enableSelf
	return s
}

// FindAndRemoveSubsumed finds and removes subsumed clauses from CNF
func (is *InprocessSubsumption) FindAndRemoveSubsumed(cnf *CNF) int {
	if len(cnf.Clauses) == 0 {
		return 0
	}

	// Build occurrence lists for efficient subsumption checking
	is.buildOccurrenceLists(cnf.Clauses)

	subsumptionCount := 0
	attempts := 0

	// Find subsumption candidates
	candidates := is.findSubsumptionCandidates(cnf.Clauses)

	// Process candidates in order of strength (strongest first)
	for _, candidate := range candidates {
		if attempts >= is.maxSubsumptionTries {
			break
		}
		attempts++

		if is.checkSubsumption(candidate.Subsumer, candidate.Subsumed) {
			// Mark subsumed clause for removal
			is.markForRemoval(cnf, candidate.Subsumed)
			subsumptionCount++
			is.subsumptions++
		}
	}

	// Perform self-subsumption if enabled
	if is.enableSelfSubsumption {
		selfSubsumed := is.performSelfSubsumption(cnf)
		subsumptionCount += selfSubsumed
	}

	// Actually remove marked clauses
	is.removeMarkedClauses(cnf)

	return subsumptionCount
}

// buildOccurrenceLists creates efficient lookup structure for subsumption
func (is *InprocessSubsumption) buildOccurrenceLists(clauses []*Clause) {
	// Clear previous occurrence lists
	for k := range is.literalOccurrence {
		delete(is.literalOccurrence, k)
	}

	// Build new occurrence lists
	for _, clause := range clauses {
		if clause == nil || len(clause.Literals) == 0 {
			continue
		}

		// Skip very large clauses for performance
		if len(clause.Literals) > is.maxClauseSize {
			continue
		}

		for _, lit := range clause.Literals {
			key := is.literalKey(lit)
			is.literalOccurrence[key] = append(is.literalOccurrence[key], clause)
		}
	}
}

// findSubsumptionCandidates identifies potential subsumption relationships
func (is *InprocessSubsumption) findSubsumptionCandidates(clauses []*Clause) []SubsumptionPair {
	is.subsumptionCandidates = is.subsumptionCandidates[:0] // Reuse slice

	for i, clause1 := range clauses {
		if clause1 == nil || len(clause1.Literals) == 0 {
			continue
		}

		// Only consider reasonably sized clauses as potential subsumers
		if len(clause1.Literals) > is.maxClauseSize/2 {
			continue
		}

		// Find potential subsumed clauses using occurrence lists
		candidates := is.findCandidatesForClause(clause1, clauses[i+1:])

		for _, clause2 := range candidates {
			if clause2 == nil || clause1.ID == clause2.ID {
				continue
			}

			// Only consider if clause1 is smaller than clause2 (potential subsumption)
			if len(clause1.Literals) < len(clause2.Literals) {
				strength := len(clause2.Literals) - len(clause1.Literals)
				is.subsumptionCandidates = append(is.subsumptionCandidates, SubsumptionPair{
					Subsumer: clause1,
					Subsumed: clause2,
					Strength: strength,
				})
			}
		}
	}

	// Sort by strength (stronger subsumptions first)
	is.sortCandidatesByStrength()

	return is.subsumptionCandidates
}

// findCandidatesForClause finds clauses that might be subsumed by the given clause
func (is *InprocessSubsumption) findCandidatesForClause(clause *Clause, remaining []*Clause) []*Clause {
	if len(clause.Literals) == 0 {
		return nil
	}

	// Start with clauses containing the first literal
	firstLit := clause.Literals[0]
	key := is.literalKey(firstLit)
	candidates := make(map[int]*Clause)

	// Get all clauses containing the first literal
	if occurrences, exists := is.literalOccurrence[key]; exists {
		for _, candidate := range occurrences {
			if candidate != nil && candidate.ID != clause.ID {
				candidates[candidate.ID] = candidate
			}
		}
	}

	// Intersect with clauses containing other literals
	for _, lit := range clause.Literals[1:] {
		key := is.literalKey(lit)
		if occurrences, exists := is.literalOccurrence[key]; exists {
			// Keep only candidates that also contain this literal
			newCandidates := make(map[int]*Clause)
			for _, candidate := range occurrences {
				if candidate != nil {
					if _, exists := candidates[candidate.ID]; exists {
						newCandidates[candidate.ID] = candidate
					}
				}
			}
			candidates = newCandidates
		} else {
			// No clauses contain this literal, so no subsumption possible
			return nil
		}
	}

	// Convert map to slice
	result := make([]*Clause, 0, len(candidates))
	for _, candidate := range candidates {
		result = append(result, candidate)
	}

	return result
}

// checkSubsumption verifies if clause1 actually subsumes clause2
func (is *InprocessSubsumption) checkSubsumption(subsumer, subsumed *Clause) bool {
	if subsumer == nil || subsumed == nil {
		return false
	}

	// Basic checks
	if len(subsumer.Literals) > len(subsumed.Literals) {
		return false // Subsumer must be smaller or equal
	}

	if subsumer.ID == subsumed.ID {
		return false // Same clause
	}

	// Check if every literal in subsumer appears in subsumed
	for _, subLit := range subsumer.Literals {
		found := false
		for _, superLit := range subsumed.Literals {
			if subLit.Equals(superLit) {
				found = true
				break
			}
		}
		if !found {
			return false // Subsumer has a literal not in subsumed
		}
	}

	return true // All literals of subsumer found in subsumed
}

// performSelfSubsumption performs self-subsumption resolution
// This strengthens clauses by removing literals through resolution
func (is *InprocessSubsumption) performSelfSubsumption(cnf *CNF) int {
	strengthenedCount := 0

	for i, clause1 := range cnf.Clauses {
		if clause1 == nil || len(clause1.Literals) <= 1 {
			continue
		}

		for j := i + 1; j < len(cnf.Clauses); j++ {
			clause2 := cnf.Clauses[j]
			if clause2 == nil || len(clause2.Literals) <= 1 {
				continue
			}

			// Try self-subsumption in both directions
			if is.trySelfSubsumption(clause1, clause2) {
				strengthenedCount++
				is.selfSubsumptions++
			}
			if is.trySelfSubsumption(clause2, clause1) {
				strengthenedCount++
				is.selfSubsumptions++
			}
		}
	}

	return strengthenedCount
}

// trySelfSubsumption attempts to strengthen clause1 using clause2
func (is *InprocessSubsumption) trySelfSubsumption(clause1, clause2 *Clause) bool {
	// Find literals that can be resolved
	var resolveLit Literal
	var found bool

	// Look for a literal in clause1 whose negation is in clause2
	for _, lit1 := range clause1.Literals {
		negLit := lit1.Negate()
		for _, lit2 := range clause2.Literals {
			if negLit.Equals(lit2) {
				resolveLit = lit1
				found = true
				break
			}
		}
		if found {
			break
		}
	}

	if !found {
		return false // No resolvable literals
	}

	// Create resolvent (clause1 without resolveLit + clause2 without negated resolveLit)
	resolvent := make([]Literal, 0)

	// Add literals from clause1 except the resolve literal
	for _, lit := range clause1.Literals {
		if !lit.Equals(resolveLit) {
			resolvent = append(resolvent, lit)
		}
	}

	// Add literals from clause2 except the negated resolve literal
	negResolveLit := resolveLit.Negate()
	for _, lit := range clause2.Literals {
		if !lit.Equals(negResolveLit) {
			// Don't add duplicates
			duplicate := false
			for _, existing := range resolvent {
				if lit.Equals(existing) {
					duplicate = true
					break
				}
			}
			if !duplicate {
				resolvent = append(resolvent, lit)
			}
		}
	}

	// Check if resolvent subsumes clause1 (making clause1 weaker)
	if len(resolvent) < len(clause1.Literals) {
		// Replace clause1 literals with resolvent
		clause1.Literals = resolvent
		is.strengthenedClauses++
		return true
	}

	return false
}

// Helper methods

func (is *InprocessSubsumption) sortCandidatesByStrength() {
	// Sort by strength (ascending - stronger subsumptions first)
	for i := 0; i < len(is.subsumptionCandidates)-1; i++ {
		for j := i + 1; j < len(is.subsumptionCandidates); j++ {
			if is.subsumptionCandidates[i].Strength > is.subsumptionCandidates[j].Strength {
				is.subsumptionCandidates[i], is.subsumptionCandidates[j] =
					is.subsumptionCandidates[j], is.subsumptionCandidates[i]
			}
		}
	}
}

func (is *InprocessSubsumption) markForRemoval(cnf *CNF, clause *Clause) {
	// Simple approach: set clause to nil (will be cleaned up later)
	for i, c := range cnf.Clauses {
		if c != nil && c.ID == clause.ID {
			cnf.Clauses[i] = nil
			break
		}
	}
}

func (is *InprocessSubsumption) removeMarkedClauses(cnf *CNF) {
	// Remove nil clauses
	validClauses := make([]*Clause, 0, len(cnf.Clauses))
	for _, clause := range cnf.Clauses {
		if clause != nil {
			validClauses = append(validClauses, clause)
		}
	}
	cnf.Clauses = validClauses
}

func (is *InprocessSubsumption) literalKey(lit Literal) string {
	if lit.Negated {
		return "¬" + lit.Variable
	}
	return lit.Variable
}

// GetStatistics returns subsumption statistics
func (is *InprocessSubsumption) GetStatistics() map[string]int64 {
	return map[string]int64{
		"subsumptions":        is.subsumptions,
		"selfSubsumptions":    is.selfSubsumptions,
		"strengthenedClauses": is.strengthenedClauses,
	}
}

// Reset clears subsumption state
func (is *InprocessSubsumption) Reset() {
	is.subsumptions = 0
	is.selfSubsumptions = 0
	is.strengthenedClauses = 0

	for k := range is.literalOccurrence {
		delete(is.literalOccurrence, k)
	}

	is.subsumptionCandidates = is.subsumptionCandidates[:0]

	for k := range is.processed {
		delete(is.processed, k)
	}
}

// BoundedVariableElimination implements efficient bounded variable elimination
type BoundedVariableElimination struct {
	// Configuration
	maxResolventSize    int     // Maximum size of resolvent clauses
	maxResolvents       int     // Maximum resolvents per variable
	maxEliminationTries int     // Maximum elimination attempts
	costThreshold       float64 // Cost threshold for elimination

	// Statistics
	eliminatedVars   int64
	resolvedClauses  int64
	addedResolvents  int64
	skippedVariables int64

	// Performance optimizations
	positiveOccurrence map[string][]*Clause // Variable -> positive clauses
	negativeOccurrence map[string][]*Clause // Variable -> negative clauses
	eliminationQueue   []EliminationCandidate
	substitutions      map[string][]Literal // Variable -> equivalent literals

	// Temporary storage for resolution
	resolutionCache  []ResolventClause
	processedClauses map[int]bool
}

// EliminationCandidate represents a variable candidate for elimination
type EliminationCandidate struct {
	Variable string
	Cost     float64 // Cost of eliminating this variable
	PosCount int     // Number of positive occurrences
	NegCount int     // Number of negative occurrences
	Priority int     // Priority for elimination (lower = higher priority)
}

// ResolventClause represents a clause generated during resolution
type ResolventClause struct {
	Literals  []Literal
	SourcePos *Clause // Source positive clause
	SourceNeg *Clause // Source negative clause
	Redundant bool    // True if this resolvent is redundant
}

// NewBoundedVariableElimination creates a new variable elimination engine
func NewBoundedVariableElimination() *BoundedVariableElimination {
	return &BoundedVariableElimination{
		maxResolventSize:    16,  // Conservative limit to prevent explosion
		maxResolvents:       100, // Maximum resolvents per variable
		maxEliminationTries: 500, // Limit total elimination attempts
		costThreshold:       2.0, // Only eliminate if cost < threshold

		positiveOccurrence: make(map[string][]*Clause),
		negativeOccurrence: make(map[string][]*Clause),
		eliminationQueue:   make([]EliminationCandidate, 0, 100),
		substitutions:      make(map[string][]Literal),
		resolutionCache:    make([]ResolventClause, 0, 1000),
		processedClauses:   make(map[int]bool),
	}
}

// NewBoundedVariableEliminationWithConfig creates elimination engine with custom settings
func NewBoundedVariableEliminationWithConfig(maxSize, maxResolvents int, threshold float64) *BoundedVariableElimination {
	ve := NewBoundedVariableElimination()
	ve.maxResolventSize = maxSize
	ve.maxResolvents = maxResolvents
	ve.costThreshold = threshold
	return ve
}

// EliminateVariables performs bounded variable elimination on the CNF
func (bve *BoundedVariableElimination) EliminateVariables(cnf *CNF, assignment Assignment) int {
	if len(cnf.Clauses) == 0 {
		return 0
	}

	// Build occurrence lists for all variables
	bve.buildOccurrenceLists(cnf.Clauses)

	// Find elimination candidates and rank them by cost
	candidates := bve.findEliminationCandidates(cnf.Variables, assignment)

	eliminatedCount := 0
	attempts := 0

	// Process candidates in order of elimination cost (lowest first)
	for _, candidate := range candidates {
		if attempts >= bve.maxEliminationTries {
			break
		}
		attempts++

		// Check if variable is still eliminable (may have been affected by previous eliminations)
		if !bve.isEliminable(candidate.Variable, cnf, assignment) {
			bve.skippedVariables++
			continue
		}

		// Calculate actual elimination cost
		actualCost := bve.calculateEliminationCost(candidate.Variable)
		if actualCost > bve.costThreshold {
			bve.skippedVariables++
			continue
		}

		// Perform the elimination
		if bve.eliminateVariable(candidate.Variable, cnf) {
			eliminatedCount++
			bve.eliminatedVars++
		}
	}

	// Clean up eliminated variable references
	bve.cleanupEliminatedVariables(cnf)

	return eliminatedCount
}

// eliminateVariable performs the actual elimination of a variable
func (bve *BoundedVariableElimination) eliminateVariable(variable string, cnf *CNF) bool {
	posOccurrences := bve.positiveOccurrence[variable]
	negOccurrences := bve.negativeOccurrence[variable]

	if len(posOccurrences) == 0 || len(negOccurrences) == 0 {
		// Pure variable - just remove clauses containing it
		bve.eliminatePureVariable(variable, cnf)
		return true
	}

	// Generate all resolvents
	resolvents := bve.generateResolvents(posOccurrences, negOccurrences, variable)

	// Check if we exceed resolvent limits
	if len(resolvents) > bve.maxResolvents {
		return false // Skip this variable
	}

	// Filter out redundant resolvents
	filteredResolvents := bve.filterRedundantResolvents(resolvents)

	// Remove original clauses containing the variable
	bve.removeClausesContaining(variable, cnf)
	bve.resolvedClauses += int64(len(posOccurrences) + len(negOccurrences))

	// Add new resolvent clauses
	for _, resolvent := range filteredResolvents {
		if len(resolvent.Literals) > 0 && len(resolvent.Literals) <= bve.maxResolventSize {
			newClause := NewClause(resolvent.Literals...)
			cnf.AddClause(newClause)
			bve.addedResolvents++
		}
	}

	// Update occurrence lists
	bve.updateOccurrenceListsAfterElimination(variable, filteredResolvents)

	return true
}

// generateResolvents creates all possible resolvents between positive and negative occurrences
func (bve *BoundedVariableElimination) generateResolvents(posOccurrences, negOccurrences []*Clause, variable string) []ResolventClause {
	bve.resolutionCache = bve.resolutionCache[:0] // Reuse slice

	for _, posClause := range posOccurrences {
		for _, negClause := range negOccurrences {
			resolvent := bve.resolveClausesPair(posClause, negClause, variable)
			if resolvent != nil {
				bve.resolutionCache = append(bve.resolutionCache, *resolvent)
			}
		}
	}

	return bve.resolutionCache
}

// resolveClausesPair performs resolution between two clauses on the given variable
func (bve *BoundedVariableElimination) resolveClausesPair(posClause, negClause *Clause, variable string) *ResolventClause {
	// Collect literals from both clauses, excluding the resolved variable
	resolventLits := make([]Literal, 0, len(posClause.Literals)+len(negClause.Literals)-2)

	// Add literals from positive clause (except positive occurrence of variable)
	for _, lit := range posClause.Literals {
		if lit.Variable != variable {
			resolventLits = append(resolventLits, lit)
		}
	}

	// Add literals from negative clause (except negative occurrence of variable)
	// Also check for complementary literals (which would make the resolvent tautological)
	tautological := false
	for _, lit := range negClause.Literals {
		if lit.Variable != variable {
			// Check if this literal's negation is already in the resolvent
			negatedLit := lit.Negate()
			for _, existingLit := range resolventLits {
				if existingLit.Equals(negatedLit) {
					tautological = true
					break
				}
			}
			if tautological {
				break
			}

			// Check for duplicates
			duplicate := false
			for _, existingLit := range resolventLits {
				if existingLit.Equals(lit) {
					duplicate = true
					break
				}
			}
			if !duplicate {
				resolventLits = append(resolventLits, lit)
			}
		}
	}

	// Skip tautological resolvents
	if tautological {
		return nil
	}

	// Skip if resolvent is too large
	if len(resolventLits) > bve.maxResolventSize {
		return nil
	}

	return &ResolventClause{
		Literals:  resolventLits,
		SourcePos: posClause,
		SourceNeg: negClause,
		Redundant: false,
	}
}

// filterRedundantResolvents removes redundant clauses from the resolvent set
func (bve *BoundedVariableElimination) filterRedundantResolvents(resolvents []ResolventClause) []ResolventClause {
	if len(resolvents) <= 1 {
		return resolvents
	}

	filtered := make([]ResolventClause, 0, len(resolvents))

	for i, resolvent := range resolvents {
		if resolvent.Redundant {
			continue
		}

		redundant := false

		// Check against other resolvents
		for j, other := range resolvents {
			if i == j || other.Redundant {
				continue
			}

			// Check if this resolvent subsumes the other (or vice versa)
			if bve.resolventSubsumes(resolvent.Literals, other.Literals) {
				// Current resolvent subsumes other - mark other as redundant
				resolvents[j].Redundant = true
			} else if bve.resolventSubsumes(other.Literals, resolvent.Literals) {
				// Other subsumes current - current is redundant
				redundant = true
				break
			}
		}

		if !redundant {
			filtered = append(filtered, resolvent)
		}
	}

	return filtered
}

// resolventSubsumes checks if one set of literals subsumes another
func (bve *BoundedVariableElimination) resolventSubsumes(literals1, literals2 []Literal) bool {
	if len(literals1) > len(literals2) {
		return false // Subsumer must be smaller or equal
	}

	// Check if every literal in literals1 appears in literals2
	for _, lit1 := range literals1 {
		found := false
		for _, lit2 := range literals2 {
			if lit1.Equals(lit2) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// buildOccurrenceLists creates efficient lookup structures for variable elimination
func (bve *BoundedVariableElimination) buildOccurrenceLists(clauses []*Clause) {
	// Clear previous occurrence lists
	for k := range bve.positiveOccurrence {
		delete(bve.positiveOccurrence, k)
	}
	for k := range bve.negativeOccurrence {
		delete(bve.negativeOccurrence, k)
	}

	// Build new occurrence lists
	for _, clause := range clauses {
		if clause == nil || len(clause.Literals) == 0 {
			continue
		}

		for _, lit := range clause.Literals {
			if lit.Negated {
				bve.negativeOccurrence[lit.Variable] = append(bve.negativeOccurrence[lit.Variable], clause)
			} else {
				bve.positiveOccurrence[lit.Variable] = append(bve.positiveOccurrence[lit.Variable], clause)
			}
		}
	}
}

// findEliminationCandidates identifies variables suitable for elimination
func (bve *BoundedVariableElimination) findEliminationCandidates(variables []string, assignment Assignment) []EliminationCandidate {
	bve.eliminationQueue = bve.eliminationQueue[:0] // Reuse slice

	for _, variable := range variables {
		// Skip assigned variables
		if assignment.IsAssigned(variable) {
			continue
		}

		posOccurrences := bve.positiveOccurrence[variable]
		negOccurrences := bve.negativeOccurrence[variable]

		posCount := len(posOccurrences)
		negCount := len(negOccurrences)

		// Skip variables that don't occur or occur only in one polarity
		if posCount == 0 || negCount == 0 {
			continue
		}

		// Calculate elimination cost (number of resolvents that would be created)
		cost := bve.calculateEliminationCost(variable)

		// Only consider variables with reasonable elimination cost
		if cost <= bve.costThreshold {
			candidate := EliminationCandidate{
				Variable: variable,
				Cost:     cost,
				PosCount: posCount,
				NegCount: negCount,
				Priority: bve.calculatePriority(posCount, negCount, cost),
			}
			bve.eliminationQueue = append(bve.eliminationQueue, candidate)
		}
	}

	// Sort candidates by cost (ascending - cheaper eliminations first)
	bve.sortCandidatesByCost()

	return bve.eliminationQueue
}

// calculateEliminationCost estimates the cost of eliminating a variable
func (bve *BoundedVariableElimination) calculateEliminationCost(variable string) float64 {
	posOccurrences := bve.positiveOccurrence[variable]
	negOccurrences := bve.negativeOccurrence[variable]

	posCount := len(posOccurrences)
	negCount := len(negOccurrences)

	if posCount == 0 || negCount == 0 {
		return 0.0 // No cost if variable is pure
	}

	// Basic cost: number of resolvents that would be created
	baseResolvents := posCount * negCount

	// Subtract the clauses that would be removed
	clausesRemoved := posCount + negCount

	// Net cost (positive means growth, negative means reduction)
	netCost := float64(baseResolvents - clausesRemoved)

	// Add penalty for creating large clauses
	sizePenalty := bve.calculateSizePenalty(posOccurrences, negOccurrences)

	return netCost + sizePenalty
}

// calculateSizePenalty penalizes eliminations that would create very large clauses
func (bve *BoundedVariableElimination) calculateSizePenalty(posOccurrences, negOccurrences []*Clause) float64 {
	penalty := 0.0

	// Estimate average resolvent size
	avgPosSize := 0.0
	for _, clause := range posOccurrences {
		avgPosSize += float64(len(clause.Literals) - 1) // -1 for eliminated variable
	}
	if len(posOccurrences) > 0 {
		avgPosSize /= float64(len(posOccurrences))
	}

	avgNegSize := 0.0
	for _, clause := range negOccurrences {
		avgNegSize += float64(len(clause.Literals) - 1) // -1 for eliminated variable
	}
	if len(negOccurrences) > 0 {
		avgNegSize /= float64(len(negOccurrences))
	}

	avgResolventSize := avgPosSize + avgNegSize

	// Penalize if expected resolvent size is large
	if avgResolventSize > 8 {
		penalty += (avgResolventSize - 8) * 0.5
	}

	// Heavy penalty if any resolvent would exceed maximum size
	for _, posClause := range posOccurrences {
		for _, negClause := range negOccurrences {
			estimatedSize := len(posClause.Literals) + len(negClause.Literals) - 2
			if estimatedSize > bve.maxResolventSize {
				penalty += 10.0 // Heavy penalty
			}
		}
	}

	return penalty
}

// calculatePriority determines elimination priority (lower = higher priority)
func (bve *BoundedVariableElimination) calculatePriority(posCount, negCount int, cost float64) int {
	// Prefer variables with lower cost and balanced occurrences
	balance := float64(posCount + negCount)
	if posCount > negCount {
		balance *= float64(posCount) / float64(negCount)
	} else if negCount > posCount {
		balance *= float64(negCount) / float64(posCount)
	}

	return int(cost*10 + balance)
}

// sortCandidatesByCost sorts elimination candidates by cost (ascending)
func (bve *BoundedVariableElimination) sortCandidatesByCost() {
	// Simple insertion sort for small arrays
	for i := 1; i < len(bve.eliminationQueue); i++ {
		key := bve.eliminationQueue[i]
		j := i - 1

		for j >= 0 && bve.eliminationQueue[j].Cost > key.Cost {
			bve.eliminationQueue[j+1] = bve.eliminationQueue[j]
			j--
		}
		bve.eliminationQueue[j+1] = key
	}
}

// isEliminable checks if a variable can still be eliminated
func (bve *BoundedVariableElimination) isEliminable(variable string, cnf *CNF, assignment Assignment) bool {
	// Skip if already assigned
	if assignment.IsAssigned(variable) {
		return false
	}

	// Check current occurrence counts
	posCount := len(bve.positiveOccurrence[variable])
	negCount := len(bve.negativeOccurrence[variable])

	// Must have both positive and negative occurrences
	if posCount == 0 || negCount == 0 {
		return false
	}

	// Check cost threshold
	cost := bve.calculateEliminationCost(variable)
	return cost <= bve.costThreshold
}

// eliminatePureVariable removes clauses containing a pure variable
func (bve *BoundedVariableElimination) eliminatePureVariable(variable string, cnf *CNF) {
	// Remove all clauses containing this variable (in any polarity)
	clausesToRemove := make([]*Clause, 0)

	// Collect clauses to remove
	if posOccurrences := bve.positiveOccurrence[variable]; len(posOccurrences) > 0 {
		clausesToRemove = append(clausesToRemove, posOccurrences...)
	}
	if negOccurrences := bve.negativeOccurrence[variable]; len(negOccurrences) > 0 {
		clausesToRemove = append(clausesToRemove, negOccurrences...)
	}

	// Remove clauses from CNF
	for _, clauseToRemove := range clausesToRemove {
		bve.removeClauseFromCNF(cnf, clauseToRemove)
	}

	bve.resolvedClauses += int64(len(clausesToRemove))
}

// removeClausesContaining removes all clauses containing the specified variable
func (bve *BoundedVariableElimination) removeClausesContaining(variable string, cnf *CNF) {
	clausesToRemove := make([]*Clause, 0)

	// Collect all clauses containing this variable
	if posOccurrences := bve.positiveOccurrence[variable]; len(posOccurrences) > 0 {
		clausesToRemove = append(clausesToRemove, posOccurrences...)
	}
	if negOccurrences := bve.negativeOccurrence[variable]; len(negOccurrences) > 0 {
		clausesToRemove = append(clausesToRemove, negOccurrences...)
	}

	// Remove duplicates
	uniqueClauses := make(map[int]*Clause)
	for _, clause := range clausesToRemove {
		uniqueClauses[clause.ID] = clause
	}

	// Remove from CNF
	for _, clause := range uniqueClauses {
		bve.removeClauseFromCNF(cnf, clause)
	}
}

// removeClauseFromCNF removes a specific clause from the CNF
func (bve *BoundedVariableElimination) removeClauseFromCNF(cnf *CNF, clauseToRemove *Clause) {
	for i, clause := range cnf.Clauses {
		if clause != nil && clause.ID == clauseToRemove.ID {
			cnf.Clauses[i] = nil // Mark for cleanup
			break
		}
	}
}

// updateOccurrenceListsAfterElimination updates occurrence lists after elimination
func (bve *BoundedVariableElimination) updateOccurrenceListsAfterElimination(eliminatedVar string, resolvents []ResolventClause) {
	// Remove eliminated variable from occurrence lists
	delete(bve.positiveOccurrence, eliminatedVar)
	delete(bve.negativeOccurrence, eliminatedVar)

	// Add new resolvents to occurrence lists
	for _, resolvent := range resolvents {
		// Create a temporary clause for occurrence tracking
		tempClause := NewClause(resolvent.Literals...)

		for _, lit := range resolvent.Literals {
			if lit.Negated {
				bve.negativeOccurrence[lit.Variable] = append(bve.negativeOccurrence[lit.Variable], tempClause)
			} else {
				bve.positiveOccurrence[lit.Variable] = append(bve.positiveOccurrence[lit.Variable], tempClause)
			}
		}
	}
}

// cleanupEliminatedVariables removes nil clauses and updates variable lists
func (bve *BoundedVariableElimination) cleanupEliminatedVariables(cnf *CNF) {
	// Remove nil clauses
	validClauses := make([]*Clause, 0, len(cnf.Clauses))
	for _, clause := range cnf.Clauses {
		if clause != nil {
			validClauses = append(validClauses, clause)
		}
	}
	cnf.Clauses = validClauses

	// Update variables list
	variableSet := make(map[string]bool)
	for _, clause := range cnf.Clauses {
		for _, lit := range clause.Literals {
			variableSet[lit.Variable] = true
		}
	}

	// Rebuild variables slice
	cnf.Variables = make([]string, 0, len(variableSet))
	for variable := range variableSet {
		cnf.Variables = append(cnf.Variables, variable)
	}
}

// GetStatistics returns variable elimination statistics
func (bve *BoundedVariableElimination) GetStatistics() map[string]int64 {
	return map[string]int64{
		"eliminatedVars":   bve.eliminatedVars,
		"resolvedClauses":  bve.resolvedClauses,
		"addedResolvents":  bve.addedResolvents,
		"skippedVariables": bve.skippedVariables,
	}
}

// Reset clears variable elimination state
func (bve *BoundedVariableElimination) Reset() {
	bve.eliminatedVars = 0
	bve.resolvedClauses = 0
	bve.addedResolvents = 0
	bve.skippedVariables = 0

	for k := range bve.positiveOccurrence {
		delete(bve.positiveOccurrence, k)
	}
	for k := range bve.negativeOccurrence {
		delete(bve.negativeOccurrence, k)
	}
	for k := range bve.substitutions {
		delete(bve.substitutions, k)
	}
	for k := range bve.processedClauses {
		delete(bve.processedClauses, k)
	}

	bve.eliminationQueue = bve.eliminationQueue[:0]
	bve.resolutionCache = bve.resolutionCache[:0]
}

// FailedLiteralProber implements advanced failed literal probing with modern optimizations
type FailedLiteralProber struct {
	// Configuration
	maxProbingDepth     int     // Maximum search depth during probing
	maxCandidates       int     // Maximum number of literals to probe
	maxProbingTime      int64   // Maximum time to spend probing (nanoseconds)
	enableDoubleProbing bool    // Enable probing both polarities
	enableHyperProbing  bool    // Enable hyper binary resolution during probing
	costThreshold       float64 // Cost threshold for candidate selection

	// Statistics
	failedLiteralsFound int64
	unitsLearned        int64
	equivalencesFound   int64
	hyperbinariesFound  int64
	probingAttempts     int64
	probingTime         int64

	// Temporary solver for probing
	probingSolver Solver

	// Performance optimizations
	candidateQueue      []ProbingCandidate
	probingCache        map[string]ProbingResult
	literalImplications map[string][]Literal // Literal -> implied literals
	binaryImplications  map[string][]Literal // For hyper binary resolution

	// State management
	originalCNF       *CNF
	probingAssignment Assignment
	impliedUnits      []Literal

	// Advanced features
	equivalenceClasses  map[string]string    // Variable -> representative
	probingOrder        []ProbingCandidate   // Optimized probing order
	watchedImplications map[string][]*Clause // For efficient implication tracking
}

// ProbingCandidate represents a literal candidate for failed literal probing
type ProbingCandidate struct {
	Literal     Literal
	Priority    float64 // Lower = higher priority
	Depth       int     // Probing depth
	Implied     int     // Number of literals this implies
	BinaryCount int     // Number of binary clauses involving this literal
	Occurrences int     // Total occurrences in formula
}

// ProbingResult caches the result of probing a literal
type ProbingResult struct {
	Failed      bool      // True if literal causes immediate conflict
	Implied     []Literal // Literals implied by this assignment
	Conflict    *Clause   // Conflict clause if failed
	Equivalents []Literal // Equivalent literals found
	Cost        int       // Cost of this probing (time/operations)
	Probed      Literal   // NEW: the literal that was probed
}

// NewFailedLiteralProber creates a new failed literal prober with default settings
func NewFailedLiteralProber() *FailedLiteralProber {
	return &FailedLiteralProber{
		maxProbingDepth:     10,   // Reasonable depth limit
		maxCandidates:       200,  // Limit probing attempts
		maxProbingTime:      5e9,  // 5 seconds maximum
		enableDoubleProbing: true, // Probe both polarities
		enableHyperProbing:  true, // Enable hyper binary resolution
		costThreshold:       50.0, // Cost threshold for candidate selection

		probingSolver:       NewDPLLSolver(), // Use lightweight solver for probing
		candidateQueue:      make([]ProbingCandidate, 0, 200),
		probingCache:        make(map[string]ProbingResult),
		literalImplications: make(map[string][]Literal),
		binaryImplications:  make(map[string][]Literal),
		impliedUnits:        make([]Literal, 0, 50),
		equivalenceClasses:  make(map[string]string),
		probingOrder:        make([]ProbingCandidate, 0, 200),
		watchedImplications: make(map[string][]*Clause),
	}
}

// ProbeFailedLiterals is the main entry point for failed literal probing
func (flp *FailedLiteralProber) ProbeFailedLiterals(cnf *CNF, assignment Assignment) []Literal {
	startTime := time.Now()
	defer func() {
		flp.probingTime += time.Since(startTime).Nanoseconds()
	}()

	flp.originalCNF = cnf
	flp.probingAssignment = assignment.Clone()
	flp.impliedUnits = flp.impliedUnits[:0] // Reset

	// Build efficient data structures for probing
	flp.buildImplicationStructures(cnf)
	flp.buildWatchedImplications(cnf)

	// Generate and rank probing candidates
	candidates := flp.generateProbingCandidates(cnf, assignment)

	// Perform probing on selected candidates
	for _, candidate := range candidates {
		// Check time limit
		if time.Since(startTime).Nanoseconds() > flp.maxProbingTime {
			break
		}

		flp.probingAttempts++
		result := flp.probeLiteral(candidate.Literal, cnf, assignment)

		// Cache the result
		flp.probingCache[flp.literalKey(candidate.Literal)] = result

		if result.Failed {
			// Found failed literal - add its negation as a unit clause
			failedLiteral := candidate.Literal.Negate()
			flp.impliedUnits = append(flp.impliedUnits, failedLiteral)
			flp.failedLiteralsFound++
			flp.unitsLearned++

			// Add unit clause to CNF immediately
			cnf.AddClause(NewClause(failedLiteral))
		}

		// Handle double probing if enabled
		if flp.enableDoubleProbing && !result.Failed {
			negResult := flp.probeLiteral(candidate.Literal.Negate(), cnf, assignment)
			flp.probingCache[flp.literalKey(candidate.Literal.Negate())] = negResult

			if negResult.Failed {
				// The negation failed, so the original literal is a unit
				flp.impliedUnits = append(flp.impliedUnits, candidate.Literal)
				flp.failedLiteralsFound++
				flp.unitsLearned++
				cnf.AddClause(NewClause(candidate.Literal))
			}
		}

		// Extract equivalences and implications
		flp.processEquivalences(result)
		flp.processImplications(result, cnf)
	}

	return flp.impliedUnits
}

// probeLiteral performs probing on a single literal
func (flp *FailedLiteralProber) probeLiteral(literal Literal, cnf *CNF, assignment Assignment) ProbingResult {
	// Check cache first
	if cached, exists := flp.probingCache[flp.literalKey(literal)]; exists {
		return cached
	}

	result := ProbingResult{
		Failed:      false,
		Implied:     make([]Literal, 0),
		Equivalents: make([]Literal, 0),
		Probed:      literal, // NEW
	}

	// Create test assignment with the probed literal
	testAssignment := assignment.Clone()
	testAssignment[literal.Variable] = !literal.Negated

	// Perform unit propagation to see what this implies
	implications, conflict := flp.performProbingWithUnitPropagation(cnf, testAssignment, literal)

	if conflict != nil {
		result.Failed = true
		result.Conflict = conflict
	} else {
		result.Implied = implications
	}

	return result
}

// performProbingWithUnitPropagation performs unit propagation during probing
func (flp *FailedLiteralProber) performProbingWithUnitPropagation(cnf *CNF, assignment Assignment, probedLiteral Literal) ([]Literal, *Clause) {
	implications := make([]Literal, 0)
	changed := true
	depth := 0

	for changed && depth < flp.maxProbingDepth {
		changed = false
		depth++

		for _, clause := range cnf.Clauses {
			if clause == nil || len(clause.Literals) == 0 {
				continue
			}

			// Skip if clause is satisfied
			if assignment.Satisfies(clause) {
				continue
			}

			// Check for conflict
			if assignment.ConflictsWith(clause) {
				return implications, clause
			}

			// Check for unit propagation
			unassignedCount := 0
			var unassignedLit Literal

			for _, lit := range clause.Literals {
				if !assignment.IsAssigned(lit.Variable) {
					unassignedCount++
					unassignedLit = lit
				}
			}

			// Unit clause found
			if unassignedCount == 1 {
				value := !unassignedLit.Negated
				assignment[unassignedLit.Variable] = value
				implications = append(implications, unassignedLit)
				changed = true
			}
		}
	}

	return implications, nil
}

// processEquivalences extracts equivalence relationships from probing results
func (flp *FailedLiteralProber) processEquivalences(result ProbingResult) {
	if result.Failed || len(result.Equivalents) == 0 {
		return
	}

	// Process equivalence relationships
	for _, equiv := range result.Equivalents {
		// Update equivalence classes
		flp.updateEquivalenceClasses(equiv)
		flp.equivalencesFound++
	}
}

// processImplications extracts and processes implications from probing
func (flp *FailedLiteralProber) processImplications(result ProbingResult, cnf *CNF) {
	if result.Failed {
		return
	}

	// Process hyper binary resolutions if enabled
	if flp.enableHyperProbing {
		flp.performHyperBinaryResolution(result, cnf)
	}
}

// performHyperBinaryResolution performs hyper binary resolution during probing
func (flp *FailedLiteralProber) performHyperBinaryResolution(result ProbingResult, cnf *CNF) {
	// Need the probed literal to encode (¬probed -> v) as (probed ∨ v)
	if result.Probed.Variable == "" {
		return
	}

	probed := result.Probed
	keyProbedNeg := flp.literalKey(probed.Negate())

	// First hop: (¬probed -> u)
	firstHops, ok := flp.binaryImplications[keyProbedNeg]
	if !ok || len(firstHops) == 0 {
		return
	}

	for _, u := range firstHops {
		// Second hop: (¬u -> v)
		keyUNeg := flp.literalKey(u.Negate())
		seconds, ok := flp.binaryImplications[keyUNeg]
		if !ok {
			continue
		}
		for _, v := range seconds {
			// Target clause (probed ∨ v) encodes (¬probed -> v)
			if v.Equals(probed.Negate()) {
				continue // (probed ∨ ¬probed) tautology
			}
			if flp.hasBinaryClause(cnf, probed, v) {
				continue // already present
			}
			// Create and register the new binary clause
			clause := NewClause(probed, v)
			cnf.AddClause(clause)
			flp.registerBinaryClause(probed, v)
			flp.hyperbinariesFound++
		}
	}
}

// hasBinaryClause checks if a binary clause (a ∨ b) already exists in the CNF
func (flp *FailedLiteralProber) hasBinaryClause(cnf *CNF, a, b Literal) bool {
	for _, c := range cnf.Clauses {
		if c == nil || len(c.Literals) != 2 {
			continue
		}
		l0, l1 := c.Literals[0], c.Literals[1]
		if (l0.Equals(a) && l1.Equals(b)) || (l0.Equals(b) && l1.Equals(a)) {
			return true
		}
	}
	return false
}

// registerBinaryClause updates the binary implication graph for (a ∨ b):
// adds (¬a -> b) and (¬b -> a)
func (flp *FailedLiteralProber) registerBinaryClause(a, b Literal) {
	keyANeg := flp.literalKey(a.Negate())
	keyBNeg := flp.literalKey(b.Negate())

	// append unique
	if !flp.containsImplication(keyANeg, b) {
		flp.binaryImplications[keyANeg] = append(flp.binaryImplications[keyANeg], b)
	}
	if !flp.containsImplication(keyBNeg, a) {
		flp.binaryImplications[keyBNeg] = append(flp.binaryImplications[keyBNeg], a)
	}
}

func (flp *FailedLiteralProber) containsImplication(key string, to Literal) bool {
	list := flp.binaryImplications[key]
	for _, x := range list {
		if x.Equals(to) {
			return true
		}
	}
	return false
}

// updateEquivalenceClasses manages equivalence class data structure
func (flp *FailedLiteralProber) updateEquivalenceClasses(literal Literal) {
	variable := literal.Variable
	if _, exists := flp.equivalenceClasses[variable]; !exists {
		flp.equivalenceClasses[variable] = variable // Self-representative initially
	}
}

// literalKey creates a string key for a literal
func (flp *FailedLiteralProber) literalKey(lit Literal) string {
	if lit.Negated {
		return "¬" + lit.Variable
	}
	return lit.Variable
}

// generateProbingCandidates generates and ranks literals for probing
func (flp *FailedLiteralProber) generateProbingCandidates(cnf *CNF, assignment Assignment) []ProbingCandidate {
	flp.candidateQueue = flp.candidateQueue[:0] // Reset

	for _, clause := range cnf.Clauses {
		if clause == nil || len(clause.Literals) == 0 {
			continue
		}

		for _, lit := range clause.Literals {
			// Skip assigned variables
			if assignment.IsAssigned(lit.Variable) {
				continue
			}

			// Check if we already have this candidate
			duplicate := false
			for _, candidate := range flp.candidateQueue {
				if candidate.Literal.Equals(lit) {
					duplicate = true
					break
				}
			}

			if !duplicate {
				priority := flp.calculateCandidatePriority(lit, cnf)
				candidate := ProbingCandidate{
					Literal:  lit,
					Priority: priority,
					Depth:    0,
				}
				flp.candidateQueue = append(flp.candidateQueue, candidate)
			}
		}
	}

	// Sort candidates by priority (higher priority first)
	flp.sortCandidatesByPriority()

	// Limit to maximum number of candidates
	if len(flp.candidateQueue) > flp.maxCandidates {
		flp.candidateQueue = flp.candidateQueue[:flp.maxCandidates]
	}

	return flp.candidateQueue
}

// calculateCandidatePriority calculates the priority for a probing candidate
func (flp *FailedLiteralProber) calculateCandidatePriority(lit Literal, cnf *CNF) float64 {
	priority := 0.0

	// Count occurrences of this literal
	occurrences := 0
	for _, clause := range cnf.Clauses {
		for _, clauseLit := range clause.Literals {
			if clauseLit.Equals(lit) {
				occurrences++
				break
			}
		}
	}

	// Prefer literals that appear frequently
	priority += float64(occurrences) * 0.1

	// Prefer literals in short clauses
	for _, clause := range cnf.Clauses {
		if len(clause.Literals) <= 2 {
			for _, clauseLit := range clause.Literals {
				if clauseLit.Equals(lit) {
					priority += 1.0 / float64(len(clause.Literals))
					break
				}
			}
		}
	}

	// Penalize literals that have been probed recently (using cache)
	if _, cached := flp.probingCache[flp.literalKey(lit)]; cached {
		priority -= 5.0
	}

	return priority
}

// sortCandidatesByPriority sorts probing candidates by priority (descending)
func (flp *FailedLiteralProber) sortCandidatesByPriority() {
	// Simple insertion sort
	for i := 1; i < len(flp.candidateQueue); i++ {
		key := flp.candidateQueue[i]
		j := i - 1

		for j >= 0 && flp.candidateQueue[j].Priority < key.Priority {
			flp.candidateQueue[j+1] = flp.candidateQueue[j]
			j--
		}
		flp.candidateQueue[j+1] = key
	}
}

// buildImplicationStructures builds efficient data structures for implication tracking
func (flp *FailedLiteralProber) buildImplicationStructures(cnf *CNF) {
	// Clear previous structures
	for k := range flp.literalImplications {
		delete(flp.literalImplications, k)
	}
	for k := range flp.binaryImplications {
		delete(flp.binaryImplications, k)
	}

	// Build binary implication graph
	for _, clause := range cnf.Clauses {
		if len(clause.Literals) == 2 {
			lit1 := clause.Literals[0]
			lit2 := clause.Literals[1]

			// Binary clause (A ∨ B) implies (¬A → B) and (¬B → A)
			negLit1 := lit1.Negate()
			negLit2 := lit2.Negate()

			key1 := flp.literalKey(negLit1)
			key2 := flp.literalKey(negLit2)

			flp.binaryImplications[key1] = append(flp.binaryImplications[key1], lit2)
			flp.binaryImplications[key2] = append(flp.binaryImplications[key2], lit1)
		}
	}
}

// buildWatchedImplications builds watched literal structures for efficient propagation
func (flp *FailedLiteralProber) buildWatchedImplications(cnf *CNF) {
	// Clear previous structures
	for k := range flp.watchedImplications {
		delete(flp.watchedImplications, k)
	}

	// Build watched literal lists for each clause
	for _, clause := range cnf.Clauses {
		if len(clause.Literals) < 2 {
			continue
		}

		// Watch the first two literals
		lit1 := clause.Literals[0]
		lit2 := clause.Literals[1]

		key1 := flp.literalKey(lit1)
		key2 := flp.literalKey(lit2)

		flp.watchedImplications[key1] = append(flp.watchedImplications[key1], clause)
		flp.watchedImplications[key2] = append(flp.watchedImplications[key2], clause)
	}
}

// GetStatistics returns failed literal probing statistics
func (flp *FailedLiteralProber) GetStatistics() map[string]int64 {
	return map[string]int64{
		"failedLiteralsFound": flp.failedLiteralsFound,
		"unitsLearned":        flp.unitsLearned,
		"equivalencesFound":   flp.equivalencesFound,
		"hyperbinariesFound":  flp.hyperbinariesFound,
		"probingAttempts":     flp.probingAttempts,
		"probingTimeNs":       flp.probingTime,
	}
}

// Reset clears failed literal prober state
func (flp *FailedLiteralProber) Reset() {
	flp.failedLiteralsFound = 0
	flp.unitsLearned = 0
	flp.equivalencesFound = 0
	flp.hyperbinariesFound = 0
	flp.probingAttempts = 0
	flp.probingTime = 0

	for k := range flp.probingCache {
		delete(flp.probingCache, k)
	}
	for k := range flp.literalImplications {
		delete(flp.literalImplications, k)
	}
	for k := range flp.binaryImplications {
		delete(flp.binaryImplications, k)
	}
	for k := range flp.equivalenceClasses {
		delete(flp.equivalenceClasses, k)
	}
	for k := range flp.watchedImplications {
		delete(flp.watchedImplications, k)
	}

	flp.candidateQueue = flp.candidateQueue[:0]
	flp.impliedUnits = flp.impliedUnits[:0]
	flp.probingOrder = flp.probingOrder[:0]

	if flp.probingSolver != nil {
		flp.probingSolver.Reset()
	}
}
