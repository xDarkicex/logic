package sat

// AdvancedDecisionTrail provides optimized trail management with fast lookups
type AdvancedDecisionTrail struct {
	// Trail entries in chronological order
	trail []TrailEntry

	// Fast lookup maps
	varToIndex  map[string]int // Variable -> index in trail
	levelStarts map[int]int    // Decision level -> first index

	// Current state
	currentLevel int
	trailSize    int

	// Reason tracking for conflict analysis
	reasons map[string]*Clause
	levels  map[string]int

	// Performance optimization
	maxLevel int
}

func NewAdvancedDecisionTrail() *AdvancedDecisionTrail {
	return &AdvancedDecisionTrail{
		trail:        make([]TrailEntry, 0, 1000), // Pre-allocate
		varToIndex:   make(map[string]int),
		levelStarts:  make(map[int]int),
		reasons:      make(map[string]*Clause),
		levels:       make(map[string]int),
		currentLevel: 0,
		trailSize:    0,
		maxLevel:     0,
	}
}

func (t *AdvancedDecisionTrail) Assign(variable string, value bool, level int, reason *Clause) {
	// Update level tracking
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

	// Add to trail
	if t.trailSize < len(t.trail) {
		t.trail[t.trailSize] = entry
	} else {
		t.trail = append(t.trail, entry)
	}

	// Update lookup maps
	t.varToIndex[variable] = t.trailSize
	t.reasons[variable] = reason
	t.levels[variable] = level

	t.trailSize++
}

func (t *AdvancedDecisionTrail) Backtrack(level int) []string {
	if level >= t.currentLevel {
		return []string{}
	}

	// Find backtrack point
	backtrackIndex := t.trailSize
	if startIdx, exists := t.levelStarts[level+1]; exists {
		backtrackIndex = startIdx
	}

	// Collect unassigned variables
	unassigned := make([]string, 0, t.trailSize-backtrackIndex)
	for i := backtrackIndex; i < t.trailSize; i++ {
		variable := t.trail[i].Variable
		unassigned = append(unassigned, variable)

		// Clean up lookup maps
		delete(t.varToIndex, variable)
		delete(t.reasons, variable)
		delete(t.levels, variable)
	}

	// Update trail size
	t.trailSize = backtrackIndex

	// Clean up level tracking
	for l := level + 1; l <= t.maxLevel; l++ {
		delete(t.levelStarts, l)
	}

	t.currentLevel = level
	return unassigned
}

func (t *AdvancedDecisionTrail) GetLevel(variable string) int {
	if level, exists := t.levels[variable]; exists {
		return level
	}
	return -1
}

func (t *AdvancedDecisionTrail) GetReason(variable string) *Clause {
	return t.reasons[variable]
}

func (t *AdvancedDecisionTrail) GetAssignment() Assignment {
	assignment := make(Assignment, t.trailSize)
	for i := 0; i < t.trailSize; i++ {
		entry := t.trail[i]
		assignment[entry.Variable] = entry.Value
	}
	return assignment
}

func (t *AdvancedDecisionTrail) GetCurrentLevel() int {
	return t.currentLevel
}

func (t *AdvancedDecisionTrail) Clear() {
	t.trailSize = 0
	t.currentLevel = 0
	t.maxLevel = 0

	// Clear maps efficiently
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
}

func (t *AdvancedDecisionTrail) GetTrailAtLevel(level int) []TrailEntry {
	startIdx, exists := t.levelStarts[level]
	if !exists {
		return []TrailEntry{}
	}

	endIdx := t.trailSize
	if level < t.maxLevel {
		if nextStart, exists := t.levelStarts[level+1]; exists {
			endIdx = nextStart
		}
	}

	entries := make([]TrailEntry, endIdx-startIdx)
	copy(entries, t.trail[startIdx:endIdx])
	return entries
}

// GetDecisionVariablesAtLevel returns only decision variables at given level
func (t *AdvancedDecisionTrail) GetDecisionVariablesAtLevel(level int) []string {
	entries := t.GetTrailAtLevel(level)
	decisions := make([]string, 0)

	for _, entry := range entries {
		if entry.Reason == nil {
			decisions = append(decisions, entry.Variable)
		}
	}

	return decisions
}

// GetImplicationChain returns the implication chain for a variable
func (t *AdvancedDecisionTrail) GetImplicationChain(variable string) []TrailEntry {
	chain := make([]TrailEntry, 0)

	current := variable
	for current != "" {
		if idx, exists := t.varToIndex[current]; exists && idx < t.trailSize {
			entry := t.trail[idx]
			chain = append(chain, entry)

			if entry.Reason == nil {
				break // Reached decision variable
			}

			// Find next variable in chain (simplified)
			current = ""
			if len(entry.Reason.Literals) > 0 {
				current = entry.Reason.Literals[0].Variable
			}
		} else {
			break
		}
	}

	return chain
}
