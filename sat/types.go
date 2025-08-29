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
	Literals     []Literal
	ID           int     // Unique identifier for tracking
	Learned      bool    // True if this is a learned clause
	Activity     float64 // For clause deletion heuristics
	LBD          int     // Literal Block Distance (number of decision levels)
	Glue         bool    // True if LBD <= 2 (very important clauses)
	Tier         int     // Clause tier classification (0=core, 1=mid, 2=local)
	ConflictType string
}

// NewClause creates a new clause with given literals and initializes LBD fields
func NewClause(literals ...Literal) *Clause {
	return &Clause{
		Literals: literals,
		Learned:  false,
		Activity: 0.0,
		LBD:      0,     // Will be computed during conflict analysis
		Glue:     false, // Will be set based on computed LBD
		Tier:     2,     // Default to local tier, will be updated based on LBD
	}
}

// SetLBD sets the LBD and updates derived fields (Glue, Tier)
func (c *Clause) SetLBD(lbd int) {
	c.LBD = lbd
	c.Glue = lbd <= 2

	// Set tier based on LBD
	if lbd <= 2 {
		c.Tier = 0 // Core clauses - never delete
	} else if lbd <= 6 {
		c.Tier = 1 // Mid-tier clauses - delete carefully
	} else {
		c.Tier = 2 // Local clauses - delete aggressively
	}
}

// IsGlue returns true if this is a glue clause (LBD <= 2)
func (c *Clause) IsGlue() bool {
	return c.Glue
}

// GetTier returns the clause tier (0=core, 1=mid, 2=local)
func (c *Clause) GetTier() int {
	return c.Tier
}

// String returns string representation of clause with LBD info for learned clauses
func (c *Clause) String() string {
	if len(c.Literals) == 0 {
		return "⊥" // Empty clause (false)
	}

	parts := make([]string, len(c.Literals))
	for i, lit := range c.Literals {
		parts[i] = lit.String()
	}

	result := "(" + strings.Join(parts, " ∨ ") + ")"

	// Add LBD info for learned clauses
	if c.Learned && c.LBD > 0 {
		result += fmt.Sprintf(" [LBD:%d,T:%d]", c.LBD, c.Tier)
	}

	return result
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

// WatchedClause represents a clause with watched literals for efficient propagation
// This is a key optimization in modern CDCL solvers for Boolean Constraint Propagation (BCP)
type WatchedClause struct {
	Clause  *Clause // The actual clause being watched
	Watch1  int     // Index of first watched literal in clause
	Watch2  int     // Index of second watched literal in clause (-1 for unit clauses)
	Blocker Literal // Blocking literal optimization for faster propagation
}

// NewWatchedClause creates a new watched clause with initial watches
func NewWatchedClause(clause *Clause) *WatchedClause {
	wc := &WatchedClause{
		Clause: clause,
		Watch1: 0,
		Watch2: -1,
	}

	// Set up initial watches
	if len(clause.Literals) >= 2 {
		wc.Watch1 = 0
		wc.Watch2 = 1
		wc.Blocker = clause.Literals[0] // Use first literal as initial blocker
	} else if len(clause.Literals) == 1 {
		wc.Watch1 = 0
		wc.Watch2 = -1 // No second watch for unit clauses
		wc.Blocker = clause.Literals[0]
	}

	return wc
}

// IsWatchingLiteral checks if this watched clause is watching the given literal index
func (wc *WatchedClause) IsWatchingLiteral(index int) bool {
	return wc.Watch1 == index || wc.Watch2 == index
}

// GetWatchedLiterals returns the currently watched literals
func (wc *WatchedClause) GetWatchedLiterals() []Literal {
	if wc.Clause == nil || len(wc.Clause.Literals) == 0 {
		return []Literal{}
	}

	literals := make([]Literal, 0, 2)
	if wc.Watch1 >= 0 && wc.Watch1 < len(wc.Clause.Literals) {
		literals = append(literals, wc.Clause.Literals[wc.Watch1])
	}
	if wc.Watch2 >= 0 && wc.Watch2 < len(wc.Clause.Literals) {
		literals = append(literals, wc.Clause.Literals[wc.Watch2])
	}

	return literals
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
				return true // Found a satisfied literal
			}
		} else {
			// Unassigned literal means clause is not yet falsified
			return true
		}
	}

	// All literals are assigned and falsified
	return false
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

// SolverStatistics tracks solver performance metrics with LBD and inprocessing support
type SolverStatistics struct {
	Decisions      int64 // Number of decision variables chosen
	Propagations   int64 // Number of unit propagations
	Conflicts      int64 // Number of conflicts encountered
	Restarts       int64 // Number of restarts performed
	LearnedClauses int64 // Number of clauses learned
	DeletedClauses int64 // Number of clauses deleted
	TimeElapsed    int64 // Solving time in nanoseconds

	// LBD-related statistics
	GlueClauses     int64         // Number of glue clauses (LBD <= 2)
	AvgLBD          float64       // Average LBD of learned clauses
	LBDDistribution map[int]int64 // Distribution of LBD values

	// Inprocessing statistics
	InprocessRuns          int64 // Number of inprocessing runs
	ClausesReduced         int64 // Total clauses removed by inprocessing
	VariablesEliminated    int64 // Total variables eliminated by inprocessing
	InprocessingTime       int64 // Total time spent in inprocessing (nanoseconds)
	FormulaSimplifications int64 // Number of times formula was significantly simplified

	// ILB Statistics
	LazyBacktracks         int64 // Number of lazy backtracks attempted
	ReimplicationSuccesses int64 // Successful reimplication attempts
	ChronologicalAttempts  int64 // Chronological backtrack attempts
	ChronologicalSuccesses int64 // Successful chronological backtracks
	AvgReimplicationCost   int64 // Average reimplication cost in nanoseconds
}

// String returns formatted statistics with inprocessing information
func (s SolverStatistics) String() string {
	base := ""
	if s.LearnedClauses > 0 {
		base = fmt.Sprintf(
			"Decisions: %d, Propagations: %d, Conflicts: %d, Restarts: %d, Learned: %d, Glue: %d, AvgLBD: %.2f",
			s.Decisions, s.Propagations, s.Conflicts, s.Restarts, s.LearnedClauses, s.GlueClauses, s.AvgLBD,
		)
	} else {
		base = fmt.Sprintf(
			"Decisions: %d, Propagations: %d, Conflicts: %d, Restarts: %d, Learned: %d",
			s.Decisions, s.Propagations, s.Conflicts, s.Restarts, s.LearnedClauses,
		)
	}

	// Add inprocessing info if used
	if s.InprocessRuns > 0 {
		base += fmt.Sprintf(", Inprocess: %d runs, %d clauses reduced, %d vars eliminated",
			s.InprocessRuns, s.ClausesReduced, s.VariablesEliminated)
	}

	return base
}

// GetLBDDistribution returns a formatted string of LBD distribution
func (s SolverStatistics) GetLBDDistribution() string {
	if len(s.LBDDistribution) == 0 {
		return "No LBD data"
	}

	parts := make([]string, 0, len(s.LBDDistribution))
	for lbd := 1; lbd <= 10; lbd++ {
		if count, exists := s.LBDDistribution[lbd]; exists && count > 0 {
			parts = append(parts, fmt.Sprintf("LBD%d: %d", lbd, count))
		}
	}
	return strings.Join(parts, ", ")
}

// Inprocess types and funcs

// InprocessResult represents the result of inprocessing operations
type InprocessResult struct {
	ClausesRemoved       int
	ClausesStrengthened  int
	VariablesEliminated  int
	UnitsLearned         int
	FormulaReduced       bool
	SubsumptionsFound    int
	VivificationsApplied int
	FailedLiteralsFound  int
}

// InprocessConfig holds configuration for inprocessing techniques
type InprocessConfig struct {
	EnableVivification     bool
	EnableSubsumption      bool
	EnableVariableElim     bool
	EnableFailedLitProbing bool
	EnableInitialInprocess bool // Run inprocessing at start

	VivificationMaxSize  int
	VarElimMaxResolvent  int
	ProbingMaxCandidates int
	InprocessGap         int64

	// Integration parameters
	InprocessAfterRestarts     int   // Run every N restarts
	InprocessMinLearnedClauses int64 // Minimum learned clauses before inprocessing
	AdaptiveGap                bool  // Enable adaptive gap adjustment

	// Resource limits
	MaxVivificationTries int
	MaxSubsumptionRounds int
	MaxProbingDepth      int
	MaxInprocessTime     int64 // Maximum time per inprocessing round (nanoseconds)
}

// InprocessStatistics tracks inprocessing performance
type InprocessStatistics struct {
	InprocessRuns          int64
	ClausesVivified        int64
	ClausesSubsumed        int64
	VariablesEliminated    int64
	FailedLiteralsFound    int64
	TimeInVivification     int64
	TimeInSubsumption      int64
	TimeInVariableElim     int64
	TimeInFailedLitProbing int64
	TotalInprocessTime     int64
}

// DefaultInprocessConfig returns sensible defaults with integration parameters
func DefaultInprocessConfig() InprocessConfig {
	return InprocessConfig{
		EnableVivification:     true,
		EnableSubsumption:      true,
		EnableVariableElim:     true,
		EnableFailedLitProbing: false, // More expensive, disabled by default
		EnableInitialInprocess: false, // Usually not needed

		VivificationMaxSize:  20,
		VarElimMaxResolvent:  16,
		ProbingMaxCandidates: 100,
		InprocessGap:         4000,

		// Integration parameters
		InprocessAfterRestarts:     3,    // Every 3 restarts
		InprocessMinLearnedClauses: 100,  // Minimum learned clauses
		AdaptiveGap:                true, // Enable adaptive adjustment

		MaxVivificationTries: 3,
		MaxSubsumptionRounds: 2,
		MaxProbingDepth:      10,
		MaxInprocessTime:     2e9, // 2 seconds maximum per round
	}
}

// ClauseDatabase manages learned clauses in a tiered structure for optimal performance
type ClauseDatabase struct {
	// Tiers
	coreClauses   []*Clause // LBD <= 2, never delete
	midClauses    []*Clause // LBD 3-6, careful deletion
	localClauses  []*Clause // LBD > 6, aggressive deletion
	recentClauses []*Clause // All newly learned clauses, protected for a period

	// Management state
	recentProtectionAge int64         // Conflicts to keep in recent before promotion
	maxSize             int           // Maximum database size before cleanup
	totalClauses        int           // Total across tiers
	bornAt              map[int]int64 // ClauseID -> conflict index when learned (only for recent)

	// Statistics
	coreCount   int
	midCount    int
	localCount  int
	recentCount int
}

// NewClauseDatabase creates an empty tiered database
func NewClauseDatabase(maxSize int, recentProtectionAge int64) *ClauseDatabase {
	return &ClauseDatabase{
		coreClauses:         make([]*Clause, 0, 1024),
		midClauses:          make([]*Clause, 0, 2048),
		localClauses:        make([]*Clause, 0, 4096),
		recentClauses:       make([]*Clause, 0, 4096),
		recentProtectionAge: recentProtectionAge,
		maxSize:             maxSize,
		totalClauses:        0,
		bornAt:              make(map[int]int64),
	}
}

// AddClause inserts a learned clause into the recent tier with protection
func (db *ClauseDatabase) AddClause(clause *Clause, conflicts int64) {
	db.recentClauses = append(db.recentClauses, clause)
	db.bornAt[clause.ID] = conflicts
	db.recentCount++
	db.totalClauses++
}

// PromoteFromRecent moves aged recent clauses into their permanent tier based on Clause.Tier
func (db *ClauseDatabase) PromoteFromRecent(conflicts int64) {
	if db.recentProtectionAge <= 0 || len(db.recentClauses) == 0 {
		return
	}
	dst := db.recentClauses[:0]
	for _, c := range db.recentClauses {
		born, ok := db.bornAt[c.ID]
		age := conflicts - born
		if ok && age >= db.recentProtectionAge {
			// Move to proper tier based on Clause.Tier
			db.placeToTier(c)
			delete(db.bornAt, c.ID)
			db.recentCount--
		} else {
			dst = append(dst, c)
		}
	}
	db.recentClauses = dst
}

// Size returns the total number of clauses across all tiers
func (db *ClauseDatabase) Size() int { return db.totalClauses }

// GetAllClauses returns a flat view over all tiers for stats/debug
func (db *ClauseDatabase) GetAllClauses() []*Clause {
	out := make([]*Clause, 0, db.totalClauses)
	out = append(out, db.coreClauses...)
	out = append(out, db.midClauses...)
	out = append(out, db.localClauses...)
	out = append(out, db.recentClauses...)
	return out
}

// GetTierSlices returns the slices for each tier (read-only view)
func (db *ClauseDatabase) GetTierSlices() (core []*Clause, mid []*Clause, local []*Clause, recent []*Clause) {
	return db.coreClauses, db.midClauses, db.localClauses, db.recentClauses
}

// RemoveClause removes a clause from whichever tier it belongs to
func (db *ClauseDatabase) RemoveClause(clause *Clause) bool {
	// If still in recent, remove from recent
	if _, ok := db.bornAt[clause.ID]; ok {
		removed := removeFromSlice(&db.recentClauses, clause)
		if removed {
			delete(db.bornAt, clause.ID)
			db.recentCount--
			db.totalClauses--
		}
		return removed
	}
	switch clause.Tier {
	case 0:
		removed := removeFromSlice(&db.coreClauses, clause)
		if removed {
			db.coreCount--
			db.totalClauses--
		}
		return removed
	case 1:
		removed := removeFromSlice(&db.midClauses, clause)
		if removed {
			db.midCount--
			db.totalClauses--
		}
		return removed
	default:
		removed := removeFromSlice(&db.localClauses, clause)
		if removed {
			db.localCount--
			db.totalClauses--
		}
		return removed
	}
}

// placeToTier appends a clause to its permanent tier based on Clause.Tier
func (db *ClauseDatabase) placeToTier(c *Clause) {
	switch c.Tier {
	case 0:
		db.coreClauses = append(db.coreClauses, c)
		db.coreCount++
	case 1:
		db.midClauses = append(db.midClauses, c)
		db.midCount++
	default:
		db.localClauses = append(db.localClauses, c)
		db.localCount++
	}
}

// Helper to remove a given clause pointer from a slice in-place
func removeFromSlice(sl *[]*Clause, target *Clause) bool {
	a := *sl
	for i, c := range a {
		if c != nil && c.ID == target.ID {
			a[i] = a[len(a)-1]
			*sl = a[:len(a)-1]
			return true
		}
	}
	return false
}

// 1. Statistics and Monitoring Methods
func (db *ClauseDatabase) GetTierStatistics() map[string]interface{} {
	return map[string]interface{}{
		"core_count":   len(db.coreClauses),
		"mid_count":    len(db.midClauses),
		"local_count":  len(db.localClauses),
		"recent_count": len(db.recentClauses),
		"total_count":  db.totalClauses,
		"max_size":     db.maxSize,
	}
}

func (db *ClauseDatabase) GetTierSizes() (core, mid, local, recent int) {
	return len(db.coreClauses), len(db.midClauses), len(db.localClauses), len(db.recentClauses)
}

// 2. Configuration Management Methods
func (db *ClauseDatabase) UpdateMaxSize(newMaxSize int) {
	db.maxSize = newMaxSize
}

func (db *ClauseDatabase) UpdateRecentProtectionAge(newAge int64) {
	db.recentProtectionAge = newAge
}

func (db *ClauseDatabase) GetMaxSize() int {
	return db.maxSize
}

// 3. Cleanup and Maintenance Methods
func (db *ClauseDatabase) Clear() {
	db.coreClauses = db.coreClauses[:0]
	db.midClauses = db.midClauses[:0]
	db.localClauses = db.localClauses[:0]
	db.recentClauses = db.recentClauses[:0]

	for k := range db.bornAt {
		delete(db.bornAt, k)
	}

	db.totalClauses = 0
	db.coreCount = 0
	db.midCount = 0
	db.localCount = 0
	db.recentCount = 0
}

func (db *ClauseDatabase) Compact() {
	db.coreClauses = compactSlice(db.coreClauses)
	db.midClauses = compactSlice(db.midClauses)
	db.localClauses = compactSlice(db.localClauses)
	db.recentClauses = compactSlice(db.recentClauses)
}

// 4. Validation and Debugging Methods
func (db *ClauseDatabase) ValidateConsistency() error {
	counted := len(db.coreClauses) + len(db.midClauses) +
		len(db.localClauses) + len(db.recentClauses)

	if counted != db.totalClauses {
		return fmt.Errorf("clause count mismatch: counted %d, stored %d",
			counted, db.totalClauses)
	}

	// Check that recent clauses have birth records
	for _, clause := range db.recentClauses {
		if _, exists := db.bornAt[clause.ID]; !exists {
			return fmt.Errorf("recent clause %d missing birth record", clause.ID)
		}
	}

	return nil
}

func (db *ClauseDatabase) String() string {
	return fmt.Sprintf("ClauseDB[core:%d mid:%d local:%d recent:%d total:%d/%d]",
		len(db.coreClauses), len(db.midClauses), len(db.localClauses),
		len(db.recentClauses), db.totalClauses, db.maxSize)
}

// 5. Tier-Specific Operations
func (db *ClauseDatabase) GetClausesByTier(tier int) []*Clause {
	switch tier {
	case 0:
		return db.coreClauses
	case 1:
		return db.midClauses
	case 2:
		return db.localClauses
	default:
		return []*Clause{}
	}
}

func (db *ClauseDatabase) ForcePromoteToTier(clause *Clause, tier int) bool {
	// Remove from current location
	if !db.RemoveClause(clause) {
		return false
	}

	// Update tier and add back
	clause.Tier = tier
	db.placeToTier(clause)
	db.totalClauses++
	return true
}

// Helper function
func compactSlice(clauses []*Clause) []*Clause {
	result := make([]*Clause, 0, len(clauses))
	for _, clause := range clauses {
		if clause != nil {
			result = append(result, clause)
		}
	}
	return result
}

// XORClause represents an XOR constraint: x1 ⊕ x2 ⊕ ... ⊕ xn = parity
type XORClause struct {
	Variables []string
	Parity    bool    // true for odd parity, false for even
	ID        int     // Unique identifier
	Learned   bool    // True if learned during Gaussian elimination
	Activity  float64 // For XOR clause management
}

// NewXORClause creates a new XOR clause
func NewXORClause(variables []string, parity bool) *XORClause {
	return &XORClause{
		Variables: variables,
		Parity:    parity,
		Learned:   false,
		Activity:  0.0,
	}
}

// String returns string representation of XOR clause
func (x *XORClause) String() string {
	if len(x.Variables) == 0 {
		if x.Parity {
			return "⊤" // True
		} else {
			return "⊥" // False
		}
	}

	vars := strings.Join(x.Variables, " ⊕ ")
	parity := "0"
	if x.Parity {
		parity = "1"
	}
	return fmt.Sprintf("(%s = %s)", vars, parity)
}

// IsUnit returns true if XOR clause has exactly one variable
func (x *XORClause) IsUnit() bool {
	return len(x.Variables) == 1
}

// IsEmpty returns true if XOR clause has no variables
func (x *XORClause) IsEmpty() bool {
	return len(x.Variables) == 0
}

// IsSatisfied checks if XOR clause is satisfied by assignment
func (x *XORClause) IsSatisfied(assignment Assignment) (bool, bool) {
	unassignedCount := 0
	xorSum := false

	for _, variable := range x.Variables {
		if value, assigned := assignment[variable]; assigned {
			if value {
				xorSum = !xorSum
			}
		} else {
			unassignedCount++
		}
	}

	if unassignedCount == 0 {
		// All assigned - check satisfaction
		return true, xorSum == x.Parity
	}

	// Still has unassigned variables
	return false, false
}

// ToRegularClauses converts XOR clause to regular clauses (exponential expansion)
func (x *XORClause) ToRegularClauses() []*Clause {
	if len(x.Variables) == 0 {
		if x.Parity {
			return []*Clause{} // Always satisfied
		} else {
			return []*Clause{NewClause()} // Empty clause (contradiction)
		}
	}

	if len(x.Variables) == 1 {
		// Unit XOR: x = parity
		literal := Literal{Variable: x.Variables[0], Negated: !x.Parity}
		return []*Clause{NewClause(literal)}
	}

	// For larger XOR clauses, create exponential expansion
	// This is expensive but necessary for correctness
	clauses := make([]*Clause, 0)

	// Generate all possible assignments and keep those that violate the XOR
	numVars := len(x.Variables)
	for assignment := 0; assignment < (1 << numVars); assignment++ {
		xorSum := false
		literals := make([]Literal, numVars)

		for i, variable := range x.Variables {
			value := (assignment>>i)&1 == 1
			if value {
				xorSum = !xorSum
			}
			// Create literal that falsifies this assignment
			literals[i] = Literal{Variable: variable, Negated: value}
		}

		// If this assignment violates the XOR constraint, add the clause
		if xorSum != x.Parity {
			clauses = append(clauses, NewClause(literals...))
		}
	}

	return clauses
}

// ExtendedCNF represents a CNF with both regular and XOR clauses
type ExtendedCNF struct {
	*CNF                    // Embed regular CNF
	XORClauses []*XORClause // XOR constraints
	nextXORID  int          // For generating unique XOR clause IDs
}

// NewExtendedCNF creates a new extended CNF
func NewExtendedCNF() *ExtendedCNF {
	return &ExtendedCNF{
		CNF:        NewCNF(),
		XORClauses: make([]*XORClause, 0),
		nextXORID:  1,
	}
}

// AddXORClause adds an XOR clause to the formula
func (ecnf *ExtendedCNF) AddXORClause(xorClause *XORClause) {
	xorClause.ID = ecnf.nextXORID
	ecnf.nextXORID++
	ecnf.XORClauses = append(ecnf.XORClauses, xorClause)

	// Track variables
	for _, variable := range xorClause.Variables {
		if !ecnf.containsVariable(variable) {
			ecnf.Variables = append(ecnf.Variables, variable)
		}
	}
}

// HasXORClauses returns true if formula contains XOR clauses
func (ecnf *ExtendedCNF) HasXORClauses() bool {
	return len(ecnf.XORClauses) > 0
}
