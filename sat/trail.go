package sat

// TrailEntry represents a single assignment in the decision trail
type TrailEntry struct {
	Variable string
	Value    bool
	Level    int
	Reason   *Clause // nil for decision variables
}

// SimpleDecisionTrail implements DecisionTrail interface
type SimpleDecisionTrail struct {
	trail        []TrailEntry
	levelIndices map[int]int // Maps decision level to start index in trail
	currentLevel int
}

// NewSimpleDecisionTrail creates a new decision trail
func NewSimpleDecisionTrail() *SimpleDecisionTrail {
	return &SimpleDecisionTrail{
		trail:        make([]TrailEntry, 0),
		levelIndices: make(map[int]int),
		currentLevel: 0,
	}
}

// Assign adds a variable assignment to the trail
func (t *SimpleDecisionTrail) Assign(variable string, value bool, level int, reason *Clause) {
	// Update level tracking
	if level > t.currentLevel {
		t.levelIndices[level] = len(t.trail)
		t.currentLevel = level
	}

	entry := TrailEntry{
		Variable: variable,
		Value:    value,
		Level:    level,
		Reason:   reason,
	}

	t.trail = append(t.trail, entry)
}

// Backtrack undoes assignments back to given level
func (t *SimpleDecisionTrail) Backtrack(level int) []string {
	if level >= t.currentLevel {
		return []string{}
	}

	// Find the cutoff point
	cutoff := len(t.trail)
	for i := len(t.trail) - 1; i >= 0; i-- {
		if t.trail[i].Level <= level {
			cutoff = i + 1
			break
		}
	}

	// Collect unassigned variables
	unassigned := make([]string, 0)
	for i := cutoff; i < len(t.trail); i++ {
		unassigned = append(unassigned, t.trail[i].Variable)
	}

	// Truncate trail
	t.trail = t.trail[:cutoff]

	// Update level tracking
	t.currentLevel = level
	newLevelIndices := make(map[int]int)
	for lvl, idx := range t.levelIndices {
		if lvl <= level && idx < cutoff {
			newLevelIndices[lvl] = idx
		}
	}
	t.levelIndices = newLevelIndices

	return unassigned
}

// GetLevel returns decision level of variable
func (t *SimpleDecisionTrail) GetLevel(variable string) int {
	for i := len(t.trail) - 1; i >= 0; i-- {
		if t.trail[i].Variable == variable {
			return t.trail[i].Level
		}
	}
	return -1 // Not found
}

// GetReason returns reason clause for variable assignment
func (t *SimpleDecisionTrail) GetReason(variable string) *Clause {
	for i := len(t.trail) - 1; i >= 0; i-- {
		if t.trail[i].Variable == variable {
			return t.trail[i].Reason
		}
	}
	return nil // Not found
}

// GetAssignment returns current assignment
func (t *SimpleDecisionTrail) GetAssignment() Assignment {
	assignment := make(Assignment)
	for _, entry := range t.trail {
		assignment[entry.Variable] = entry.Value
	}
	return assignment
}

// GetCurrentLevel returns current decision level
func (t *SimpleDecisionTrail) GetCurrentLevel() int {
	return t.currentLevel
}

// Clear resets trail
func (t *SimpleDecisionTrail) Clear() {
	t.trail = t.trail[:0]
	t.levelIndices = make(map[int]int)
	t.currentLevel = 0
}

// GetTrailAtLevel returns all assignments at given level
func (t *SimpleDecisionTrail) GetTrailAtLevel(level int) []TrailEntry {
	var entries []TrailEntry
	for _, entry := range t.trail {
		if entry.Level == level {
			entries = append(entries, entry)
		}
	}
	return entries
}
