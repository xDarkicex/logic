// gaussian.go
package sat

import (
	"time"
)

// GaussianEliminator implements Gauss-Jordan elimination for XOR constraints
type GaussianEliminator struct {
	// Configuration
	maxMatrixRows     int
	maxMatrixCols     int
	autoDisable       bool
	minXORSize        int
	maxXORSize        int
	gaussianFrequency int64 // Run every N conflicts

	// Matrix representation
	matrix       [][]bool       // Augmented matrix for Gaussian elimination
	originalXORs []*XORClause   // Original XOR clauses
	matrixToVar  []string       // Matrix column to variable mapping
	varToMatrix  map[string]int // Variable to matrix column mapping
	matrixRows   int
	matrixCols   int

	// State tracking
	lastGaussian     int64
	disabled         bool
	eliminationCount int64
	unitPropagations int64

	// Statistics
	stats GaussianStats
}

// GaussianStats tracks Gaussian elimination performance
type GaussianStats struct {
	TotalRuns           int64
	VariablesEliminated int64
	XORClausesLearned   int64
	UnitPropagations    int64
	MatrixReductions    int64
	TimeInGaussian      int64 // Nanoseconds
	ConflictsFound      int64
	AutoDisableCount    int64
}

// NewGaussianEliminator creates a new Gaussian eliminator
func NewGaussianEliminator() *GaussianEliminator {
	return &GaussianEliminator{
		maxMatrixRows:     300,  // Reasonable size limit
		maxMatrixCols:     200,  // Reasonable size limit
		autoDisable:       true, // Disable if not effective
		minXORSize:        3,    // Minimum XOR size to consider
		maxXORSize:        20,   // Maximum XOR size to consider
		gaussianFrequency: 5000, // Run every 5000 conflicts

		varToMatrix:      make(map[string]int),
		lastGaussian:     0,
		disabled:         false,
		eliminationCount: 0,
	}
}

// NewGaussianEliminatorWithConfig creates eliminator with custom config
func NewGaussianEliminatorWithConfig(maxRows, maxCols, minSize, maxSize int, frequency int64) *GaussianEliminator {
	ge := NewGaussianEliminator()
	ge.maxMatrixRows = maxRows
	ge.maxMatrixCols = maxCols
	ge.minXORSize = minSize
	ge.maxXORSize = maxSize
	ge.gaussianFrequency = frequency
	return ge
}

// ShouldRunGaussian determines if Gaussian elimination should run
func (ge *GaussianEliminator) ShouldRunGaussian(conflicts int64, xorClauseCount int) bool {
	if ge.disabled || xorClauseCount == 0 {
		return false
	}

	// Run based on conflict frequency
	if conflicts < ge.lastGaussian+ge.gaussianFrequency {
		return false
	}

	// Need sufficient XOR clauses
	return xorClauseCount >= 5
}

// PerformGaussianElimination performs Gauss-Jordan elimination on XOR clauses
func (ge *GaussianEliminator) PerformGaussianElimination(ecnf *ExtendedCNF, assignment Assignment, conflicts int64) (*GaussianResult, error) {
	if ge.disabled {
		return &GaussianResult{}, nil
	}

	startTime := time.Now()
	defer func() {
		ge.stats.TimeInGaussian += time.Since(startTime).Nanoseconds()
		ge.stats.TotalRuns++
		ge.lastGaussian = conflicts
	}()

	result := &GaussianResult{
		UnitsLearned:      make([]Literal, 0),
		XORClausesLearned: make([]*XORClause, 0),
		ConflictFound:     false,
	}

	// Build matrix from XOR clauses
	if !ge.buildMatrix(ecnf.XORClauses, assignment) {
		return result, nil // No suitable XOR clauses
	}

	// Perform Gaussian elimination
	eliminated := ge.eliminateMatrix()
	ge.stats.VariablesEliminated += int64(eliminated)

	// Extract results
	ge.extractResults(result, assignment)

	// Check if Gaussian elimination is effective
	if ge.autoDisable && ge.shouldDisable() {
		ge.disabled = true
		ge.stats.AutoDisableCount++
	}

	return result, nil
}

// buildMatrix constructs the augmented matrix from XOR clauses
func (ge *GaussianEliminator) buildMatrix(xorClauses []*XORClause, assignment Assignment) bool {
	if len(xorClauses) == 0 {
		return false
	}

	// Collect suitable XOR clauses
	suitableXORs := make([]*XORClause, 0)
	variableSet := make(map[string]bool)

	for _, xor := range xorClauses {
		// Filter by size
		if len(xor.Variables) < ge.minXORSize || len(xor.Variables) > ge.maxXORSize {
			continue
		}

		// Skip if too many variables are already assigned
		unassignedCount := 0
		for _, variable := range xor.Variables {
			if !assignment.IsAssigned(variable) {
				unassignedCount++
				variableSet[variable] = true
			}
		}

		if unassignedCount >= 2 { // Need at least 2 unassigned variables
			suitableXORs = append(suitableXORs, xor)
		}
	}

	if len(suitableXORs) == 0 || len(variableSet) > ge.maxMatrixCols {
		return false
	}

	// Check matrix size limits
	if len(suitableXORs) > ge.maxMatrixRows {
		suitableXORs = suitableXORs[:ge.maxMatrixRows]
	}

	// Build variable mapping
	ge.matrixToVar = make([]string, 0, len(variableSet))
	ge.varToMatrix = make(map[string]int)

	for variable := range variableSet {
		ge.varToMatrix[variable] = len(ge.matrixToVar)
		ge.matrixToVar = append(ge.matrixToVar, variable)
	}

	ge.matrixRows = len(suitableXORs)
	ge.matrixCols = len(ge.matrixToVar) + 1 // +1 for augmented column (RHS)

	// Initialize matrix
	ge.matrix = make([][]bool, ge.matrixRows)
	for i := range ge.matrix {
		ge.matrix[i] = make([]bool, ge.matrixCols)
	}

	// Fill matrix
	ge.originalXORs = suitableXORs
	for rowIdx, xor := range suitableXORs {
		// Calculate RHS (right-hand side) considering assigned variables
		rhs := xor.Parity

		for _, variable := range xor.Variables {
			if colIdx, exists := ge.varToMatrix[variable]; exists {
				// Unassigned variable - add to matrix
				ge.matrix[rowIdx][colIdx] = true
			} else if value, assigned := assignment[variable]; assigned && value {
				// Assigned variable with value true - affects RHS
				rhs = !rhs
			}
		}

		// Set RHS in augmented column
		ge.matrix[rowIdx][ge.matrixCols-1] = rhs
	}

	return true
}

// eliminateMatrix performs Gauss-Jordan elimination
func (ge *GaussianEliminator) eliminateMatrix() int {
	if len(ge.matrix) == 0 {
		return 0
	}

	eliminated := 0
	currentRow := 0

	// Forward elimination
	for col := 0; col < ge.matrixCols-1 && currentRow < ge.matrixRows; col++ {
		// Find pivot
		pivotRow := ge.findPivot(currentRow, col)
		if pivotRow == -1 {
			continue // No pivot in this column
		}

		// Swap rows if needed
		if pivotRow != currentRow {
			ge.matrix[currentRow], ge.matrix[pivotRow] = ge.matrix[pivotRow], ge.matrix[currentRow]
		}

		// Eliminate column
		for row := 0; row < ge.matrixRows; row++ {
			if row != currentRow && ge.matrix[row][col] {
				ge.eliminateRow(row, currentRow)
			}
		}

		eliminated++
		currentRow++
	}

	ge.stats.MatrixReductions++
	return eliminated
}

// findPivot finds a row with 1 in the given column starting from startRow
func (ge *GaussianEliminator) findPivot(startRow, col int) int {
	for row := startRow; row < ge.matrixRows; row++ {
		if ge.matrix[row][col] {
			return row
		}
	}
	return -1
}

// eliminateRow performs XOR elimination: row1 = row1 XOR row2
func (ge *GaussianEliminator) eliminateRow(row1, row2 int) {
	for col := 0; col < ge.matrixCols; col++ {
		ge.matrix[row1][col] = ge.matrix[row1][col] != ge.matrix[row2][col] // XOR
	}
}

// extractResults extracts unit propagations and learned XOR clauses
func (ge *GaussianEliminator) extractResults(result *GaussianResult, assignment Assignment) {
	for row := 0; row < ge.matrixRows; row++ {
		activeVars := make([]string, 0)

		// Count active variables in this row
		for col := 0; col < ge.matrixCols-1; col++ {
			if ge.matrix[row][col] {
				activeVars = append(activeVars, ge.matrixToVar[col])
			}
		}

		rhs := ge.matrix[row][ge.matrixCols-1]

		if len(activeVars) == 0 {
			// No variables left
			if rhs {
				// 0 = 1 - contradiction
				result.ConflictFound = true
				ge.stats.ConflictsFound++
				return
			}
			// 0 = 0 - tautology, ignore
		} else if len(activeVars) == 1 {
			// Unit XOR: single variable
			variable := activeVars[0]
			value := rhs // Variable must equal RHS

			// Only propagate if not already assigned
			if !assignment.IsAssigned(variable) {
				literal := Literal{Variable: variable, Negated: !value}
				result.UnitsLearned = append(result.UnitsLearned, literal)
				ge.stats.UnitPropagations++
				ge.unitPropagations++
			}
		} else if len(activeVars) <= ge.maxXORSize {
			// Learned XOR clause
			learnedXOR := NewXORClause(activeVars, rhs)
			learnedXOR.Learned = true
			result.XORClausesLearned = append(result.XORClausesLearned, learnedXOR)
			ge.stats.XORClausesLearned++
		}
	}
}

// shouldDisable determines if Gaussian elimination should be auto-disabled
func (ge *GaussianEliminator) shouldDisable() bool {
	if ge.stats.TotalRuns < 5 {
		return false // Need more data
	}

	// Disable if very few eliminations per run
	avgEliminations := float64(ge.stats.VariablesEliminated) / float64(ge.stats.TotalRuns)
	if avgEliminations < 0.1 {
		return true
	}

	// Disable if very few unit propagations
	avgUnits := float64(ge.stats.UnitPropagations) / float64(ge.stats.TotalRuns)
	if avgUnits < 0.5 {
		return true
	}

	return false
}

// GetStatistics returns Gaussian elimination statistics
func (ge *GaussianEliminator) GetStatistics() GaussianStats {
	return ge.stats
}

// Reset clears Gaussian eliminator state
func (ge *GaussianEliminator) Reset() {
	ge.matrix = nil
	ge.originalXORs = nil
	ge.matrixToVar = nil
	ge.varToMatrix = make(map[string]int)
	ge.lastGaussian = 0
	ge.disabled = false
	ge.eliminationCount = 0
	ge.unitPropagations = 0
	ge.stats = GaussianStats{}
}

// IsDisabled returns true if Gaussian elimination is disabled
func (ge *GaussianEliminator) IsDisabled() bool {
	return ge.disabled
}

// Enable re-enables Gaussian elimination
func (ge *GaussianEliminator) Enable() {
	ge.disabled = false
}

// GaussianResult represents the result of Gaussian elimination
type GaussianResult struct {
	UnitsLearned        []Literal    // Unit literals learned
	XORClausesLearned   []*XORClause // New XOR clauses learned
	ConflictFound       bool         // True if contradiction found
	VariablesEliminated int          // Number of variables eliminated
}
