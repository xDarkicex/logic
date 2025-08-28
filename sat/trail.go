package sat

// TrailEntry represents a single assignment in the decision trail
type TrailEntry struct {
	Variable string
	Value    bool
	Level    int
	Reason   *Clause // nil for decision variables
}

// DecisionTrailImpl provides optimized trail management with fast lookups
// This is the unified implementation combining all advanced features
// Note: DecisionTrail interface is defined in interfaces.go
type DecisionTrailImpl struct {
	// Trail entries in chronological order
	trail []TrailEntry

	// Fast lookup maps for O(1) operations
	varToIndex  map[string]int // Variable -> index in trail
	levelStarts map[int]int    // Decision level -> first index

	// Current state tracking
	currentLevel int
	trailSize    int
	maxLevel     int

	// Reason tracking for conflict analysis and CDCL
	reasons map[string]*Clause
	levels  map[string]int
}

// NewDecisionTrail creates a new advanced decision trail (primary constructor)
func NewDecisionTrail() *DecisionTrailImpl {
	return &DecisionTrailImpl{
		trail:        make([]TrailEntry, 0, 1000), // Pre-allocate for performance
		varToIndex:   make(map[string]int),
		levelStarts:  make(map[int]int),
		reasons:      make(map[string]*Clause),
		levels:       make(map[string]int),
		currentLevel: 0,
		trailSize:    0,
		maxLevel:     0,
	}
}

// NewSimpleDecisionTrail creates a decision trail (backward compatibility)
// This now returns the advanced implementation for consistent performance
func NewSimpleDecisionTrail() *DecisionTrailImpl {
	return NewDecisionTrail()
}

// For compatibility with existing code that expects SimpleDecisionTrail type
type SimpleDecisionTrail = DecisionTrailImpl

// NewAdvancedDecisionTrail creates an advanced decision trail (backward compatibility)
func NewAdvancedDecisionTrail() *DecisionTrailImpl {
	return NewDecisionTrail()
}

// Assign adds a variable assignment to the trail with optimized performance
func (t *DecisionTrailImpl) Assign(variable string, value bool, level int, reason *Clause) {
	// Update level tracking with fast lookup preparation
	if level > t.currentLevel {
		t.levelStarts[level] = t.trailSize
		t.currentLevel = level
		if level > t.maxLevel {
			t.maxLevel = level
		}
	}

	entry := TrailEntry{
		Variable: variable,
		Value:    value,
		Level:    level,
		Reason:   reason,
	}

	// Efficient trail management with pre-allocation
	if t.trailSize < len(t.trail) {
		t.trail[t.trailSize] = entry
	} else {
		t.trail = append(t.trail, entry)
	}

	// Update fast lookup maps for O(1) access
	t.varToIndex[variable] = t.trailSize
	t.reasons[variable] = reason
	t.levels[variable] = level

	t.trailSize++
}

// Backtrack undoes assignments back to given level with optimized cleanup
func (t *DecisionTrailImpl) Backtrack(level int) []string {
	if level >= t.currentLevel {
		return []string{}
	}

	// Find backtrack point using fast level lookup
	backtrackIndex := t.trailSize
	if startIdx, exists := t.levelStarts[level+1]; exists {
		backtrackIndex = startIdx
	}

	// Collect unassigned variables efficiently
	unassigned := make([]string, 0, t.trailSize-backtrackIndex)
	for i := backtrackIndex; i < t.trailSize; i++ {
		variable := t.trail[i].Variable
		unassigned = append(unassigned, variable)

		// Clean up lookup maps for memory efficiency
		delete(t.varToIndex, variable)
		delete(t.reasons, variable)
		delete(t.levels, variable)
	}

	// Update trail size (reuse allocated memory)
	t.trailSize = backtrackIndex

	// Clean up level tracking efficiently
	for l := level + 1; l <= t.maxLevel; l++ {
		delete(t.levelStarts, l)
	}

	// Update max level tracking
	t.currentLevel = level
	if level < t.maxLevel {
		// Recalculate maxLevel
		t.maxLevel = level
		for l := range t.levelStarts {
			if l > t.maxLevel {
				t.maxLevel = l
			}
		}
	}

	return unassigned
}

// GetLevel returns decision level of variable with O(1) lookup
func (t *DecisionTrailImpl) GetLevel(variable string) int {
	if level, exists := t.levels[variable]; exists {
		return level
	}
	return -1 // Variable not assigned
}

// GetReason returns reason clause for variable assignment with O(1) lookup
func (t *DecisionTrailImpl) GetReason(variable string) *Clause {
	return t.reasons[variable] // Returns nil if not found
}

// GetAssignment returns current complete assignment
func (t *DecisionTrailImpl) GetAssignment() Assignment {
	assignment := make(Assignment, t.trailSize)
	for i := 0; i < t.trailSize; i++ {
		entry := t.trail[i]
		assignment[entry.Variable] = entry.Value
	}
	return assignment
}

// GetCurrentLevel returns current decision level
func (t *DecisionTrailImpl) GetCurrentLevel() int {
	return t.currentLevel
}

// Clear resets trail to initial state with efficient cleanup
func (t *DecisionTrailImpl) Clear() {
	// Reset counters
	t.trailSize = 0
	t.currentLevel = 0
	t.maxLevel = 0

	// Clear maps efficiently without reallocating
	for k := range t.varToIndex {
		delete(t.varToIndex, k)
	}
	for k := range t.levelStarts {
		delete(t.levelStarts, k)
	}
	for k := range t.reasons {
		delete(t.reasons, k)
	}
	for k := range t.levels {
		delete(t.levels, k)
	}

	// Keep pre-allocated trail slice for reuse
}

// GetTrailAtLevel returns all assignments at given level (advanced feature)
func (t *DecisionTrailImpl) GetTrailAtLevel(level int) []TrailEntry {
	startIdx, exists := t.levelStarts[level]
	if !exists {
		return []TrailEntry{}
	}

	// Calculate end index for this level
	endIdx := t.trailSize
	for l := level + 1; l <= t.maxLevel; l++ {
		if nextStart, exists := t.levelStarts[l]; exists {
			endIdx = nextStart
			break
		}
	}

	// Return copy of entries to prevent external modification
	if endIdx <= startIdx {
		return []TrailEntry{}
	}

	entries := make([]TrailEntry, endIdx-startIdx)
	copy(entries, t.trail[startIdx:endIdx])
	return entries
}

// GetDecisionVariablesAtLevel returns only decision variables at given level
// This is crucial for conflict analysis in CDCL algorithms
func (t *DecisionTrailImpl) GetDecisionVariablesAtLevel(level int) []string {
	entries := t.GetTrailAtLevel(level)
	decisions := make([]string, 0, len(entries)) // Pre-allocate reasonable capacity

	for _, entry := range entries {
		if entry.Reason == nil { // Decision variables have no reason clause
			decisions = append(decisions, entry.Variable)
		}
	}

	return decisions
}

// GetImplicationChain returns the implication chain for a variable
// Useful for debugging, learning, and conflict analysis
func (t *DecisionTrailImpl) GetImplicationChain(variable string) []TrailEntry {
	chain := make([]TrailEntry, 0, 10) // Pre-allocate reasonable chain size
	visited := make(map[string]bool)   // Prevent infinite loops

	current := variable
	for current != "" && !visited[current] {
		visited[current] = true

		idx, exists := t.varToIndex[current]
		if !exists || idx >= t.trailSize {
			break
		}

		entry := t.trail[idx]
		chain = append(chain, entry)

		// If this is a decision variable, we've reached the end of the chain
		if entry.Reason == nil {
			break
		}

		// Find the next variable in the implication chain
		// This is a simplified approach - in practice, you might want
		// to trace through the specific literal that caused this implication
		current = ""
		if len(entry.Reason.Literals) > 0 {
			// Find an assigned literal in the reason clause that could have
			// triggered this implication
			for _, literal := range entry.Reason.Literals {
				if literal.Variable != entry.Variable {
					if _, assigned := t.varToIndex[literal.Variable]; assigned {
						current = literal.Variable
						break
					}
				}
			}
		}
	}

	return chain
}

// GetTrailSize returns current number of assignments (utility method)
func (t *DecisionTrailImpl) GetTrailSize() int {
	return t.trailSize
}

// GetMaxLevel returns the highest decision level seen (utility method)
func (t *DecisionTrailImpl) GetMaxLevel() int {
	return t.maxLevel
}

// IsDecisionVariable checks if a variable is a decision variable (utility method)
func (t *DecisionTrailImpl) IsDecisionVariable(variable string) bool {
	reason := t.GetReason(variable)
	return reason == nil && t.GetLevel(variable) >= 0
}

// GetLevelSize returns number of assignments at given level (utility method)
func (t *DecisionTrailImpl) GetLevelSize(level int) int {
	return len(t.GetTrailAtLevel(level))
}

// GetAllLevels returns all active decision levels (utility method)
func (t *DecisionTrailImpl) GetAllLevels() []int {
	levels := make([]int, 0, len(t.levelStarts))
	for level := range t.levelStarts {
		levels = append(levels, level)
	}
	return levels
}

// AsAdvanced returns the implementation as the concrete type for access to advanced methods
// This allows accessing advanced methods while maintaining interface compatibility
func AsAdvanced(trail DecisionTrail) *DecisionTrailImpl {
	if impl, ok := trail.(*DecisionTrailImpl); ok {
		return impl
	}
	return nil
}

// NewDecisionTrailInterface returns a DecisionTrail interface for general use
func NewDecisionTrailInterface() DecisionTrail {
	return NewDecisionTrail()
}
