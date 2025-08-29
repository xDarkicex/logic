// pool.go
package sat

import (
	"sync"
)

// SATPool manages object pools for the SAT solver to reduce GC pressure
// and improve performance by reusing frequently allocated objects
type SATPool struct {
	// Core data structure pools
	assignmentPool   *sync.Pool
	literalSlicePool *sync.Pool
	stringSlicePool  *sync.Pool
	clauseSlicePool  *sync.Pool
	trailEntryPool   *sync.Pool

	// Conflict analysis pools
	resolutionStepPool   *sync.Pool
	conflictAnalysisPool *sync.Pool

	// Inprocessing pools
	probingCandidatePool     *sync.Pool
	eliminationCandidatePool *sync.Pool
	resolventClausePool      *sync.Pool
	subsumptionPairPool      *sync.Pool

	// Watch list and occurrence pools
	watchedClauseSlicePool *sync.Pool
	clausePtrSlicePool     *sync.Pool

	// Map pools for various algorithms
	stringBoolMapPool  *sync.Pool
	stringIntMapPool   *sync.Pool
	stringFloatMapPool *sync.Pool
	intClauseMapPool   *sync.Pool

	// Large buffer pools
	smallBufferPool  *sync.Pool // up to 64 elements
	mediumBufferPool *sync.Pool // up to 256 elements
	largeBufferPool  *sync.Pool // up to 1024 elements
}

// Global pool instance
var globalSATPool = NewSATPool()

// GetPool returns the global SAT pool instance
func GetPool() *SATPool {
	return globalSATPool
}

// NewSATPool creates a new SAT object pool with all necessary pools
func NewSATPool() *SATPool {
	return &SATPool{
		// Core data structures
		assignmentPool: &sync.Pool{
			New: func() interface{} {
				return make(Assignment, 32) // Pre-size for typical use
			},
		},
		literalSlicePool: &sync.Pool{
			New: func() interface{} {
				return make([]Literal, 0, 16) // Pre-allocate for typical clause size
			},
		},
		stringSlicePool: &sync.Pool{
			New: func() interface{} {
				return make([]string, 0, 32)
			},
		},
		clauseSlicePool: &sync.Pool{
			New: func() interface{} {
				return make([]*Clause, 0, 64)
			},
		},
		trailEntryPool: &sync.Pool{
			New: func() interface{} {
				return make([]TrailEntry, 0, 128)
			},
		},

		// Conflict analysis
		resolutionStepPool: &sync.Pool{
			New: func() interface{} {
				return make([]ResolutionStep, 0, 32)
			},
		},
		conflictAnalysisPool: &sync.Pool{
			New: func() interface{} {
				return make(map[string]bool, 64)
			},
		},

		// Inprocessing
		probingCandidatePool: &sync.Pool{
			New: func() interface{} {
				return make([]ProbingCandidate, 0, 200)
			},
		},
		eliminationCandidatePool: &sync.Pool{
			New: func() interface{} {
				return make([]EliminationCandidate, 0, 100)
			},
		},
		resolventClausePool: &sync.Pool{
			New: func() interface{} {
				return make([]ResolventClause, 0, 1000)
			},
		},
		subsumptionPairPool: &sync.Pool{
			New: func() interface{} {
				return make([]SubsumptionPair, 0, 100)
			},
		},

		// Watch lists and occurrences
		watchedClauseSlicePool: &sync.Pool{
			New: func() interface{} {
				return make([]*WatchedClause, 0, 32)
			},
		},
		clausePtrSlicePool: &sync.Pool{
			New: func() interface{} {
				return make([]*Clause, 0, 64)
			},
		},

		// Map pools
		stringBoolMapPool: &sync.Pool{
			New: func() interface{} {
				return make(map[string]bool, 64)
			},
		},
		stringIntMapPool: &sync.Pool{
			New: func() interface{} {
				return make(map[string]int, 64)
			},
		},
		stringFloatMapPool: &sync.Pool{
			New: func() interface{} {
				return make(map[string]float64, 64)
			},
		},
		intClauseMapPool: &sync.Pool{
			New: func() interface{} {
				return make(map[int]*Clause, 64)
			},
		},

		// Buffer pools for different sizes
		smallBufferPool: &sync.Pool{
			New: func() interface{} {
				return make([]interface{}, 0, 64)
			},
		},
		mediumBufferPool: &sync.Pool{
			New: func() interface{} {
				return make([]interface{}, 0, 256)
			},
		},
		largeBufferPool: &sync.Pool{
			New: func() interface{} {
				return make([]interface{}, 0, 1024)
			},
		},
	}
}

// Assignment pool methods
func (p *SATPool) GetAssignment() Assignment {
	assignment := p.assignmentPool.Get().(Assignment)
	// Clear the map while preserving capacity
	for k := range assignment {
		delete(assignment, k)
	}
	return assignment
}

func (p *SATPool) PutAssignment(assignment Assignment) {
	if assignment != nil && len(assignment) < 1000 { // Don't pool extremely large assignments
		p.assignmentPool.Put(assignment)
	}
}

// Literal slice pool methods
func (p *SATPool) GetLiteralSlice(size int) []Literal {
	slice := p.literalSlicePool.Get().([]Literal)
	if cap(slice) < size {
		// If capacity is too small, create new slice
		return make([]Literal, 0, size)
	}
	return slice[:0] // Reset length while preserving capacity
}

func (p *SATPool) PutLiteralSlice(slice []Literal) {
	if slice != nil && cap(slice) <= 128 { // Don't pool extremely large slices
		p.literalSlicePool.Put(slice)
	}
}

// String slice pool methods
func (p *SATPool) GetStringSlice(size int) []string {
	slice := p.stringSlicePool.Get().([]string)
	if cap(slice) < size {
		return make([]string, 0, size)
	}
	return slice[:0]
}

func (p *SATPool) PutStringSlice(slice []string) {
	if slice != nil && cap(slice) <= 256 {
		p.stringSlicePool.Put(slice)
	}
}

// Clause slice pool methods
func (p *SATPool) GetClauseSlice(size int) []*Clause {
	slice := p.clauseSlicePool.Get().([]*Clause)
	if cap(slice) < size {
		return make([]*Clause, 0, size)
	}
	return slice[:0]
}

func (p *SATPool) PutClauseSlice(slice []*Clause) {
	if slice != nil && cap(slice) <= 512 {
		p.clauseSlicePool.Put(slice)
	}
}

// Trail entry pool methods
func (p *SATPool) GetTrailEntrySlice(size int) []TrailEntry {
	slice := p.trailEntryPool.Get().([]TrailEntry)
	if cap(slice) < size {
		return make([]TrailEntry, 0, size)
	}
	return slice[:0]
}

func (p *SATPool) PutTrailEntrySlice(slice []TrailEntry) {
	if slice != nil && cap(slice) <= 1000 {
		p.trailEntryPool.Put(slice)
	}
}

// Resolution step pool methods
func (p *SATPool) GetResolutionSteps() []ResolutionStep {
	return p.resolutionStepPool.Get().([]ResolutionStep)[:0]
}

func (p *SATPool) PutResolutionSteps(steps []ResolutionStep) {
	if steps != nil && cap(steps) <= 200 {
		p.resolutionStepPool.Put(steps)
	}
}

// Conflict analysis map pool
func (p *SATPool) GetConflictAnalysisMap() map[string]bool {
	m := p.conflictAnalysisPool.Get().(map[string]bool)
	for k := range m {
		delete(m, k)
	}
	return m
}

func (p *SATPool) PutConflictAnalysisMap(m map[string]bool) {
	if m != nil && len(m) < 500 {
		p.conflictAnalysisPool.Put(m)
	}
}

// Probing candidate pool methods
func (p *SATPool) GetProbingCandidates() []ProbingCandidate {
	return p.probingCandidatePool.Get().([]ProbingCandidate)[:0]
}

func (p *SATPool) PutProbingCandidates(candidates []ProbingCandidate) {
	if candidates != nil && cap(candidates) <= 500 {
		p.probingCandidatePool.Put(candidates)
	}
}

// Elimination candidate pool methods
func (p *SATPool) GetEliminationCandidates() []EliminationCandidate {
	return p.eliminationCandidatePool.Get().([]EliminationCandidate)[:0]
}

func (p *SATPool) PutEliminationCandidates(candidates []EliminationCandidate) {
	if candidates != nil && cap(candidates) <= 300 {
		p.eliminationCandidatePool.Put(candidates)
	}
}

// Resolvent clause pool methods
func (p *SATPool) GetResolventClauses() []ResolventClause {
	return p.resolventClausePool.Get().([]ResolventClause)[:0]
}

func (p *SATPool) PutResolventClauses(clauses []ResolventClause) {
	if clauses != nil && cap(clauses) <= 2000 {
		p.resolventClausePool.Put(clauses)
	}
}

// Subsumption pair pool methods
func (p *SATPool) GetSubsumptionPairs() []SubsumptionPair {
	return p.subsumptionPairPool.Get().([]SubsumptionPair)[:0]
}

func (p *SATPool) PutSubsumptionPairs(pairs []SubsumptionPair) {
	if pairs != nil && cap(pairs) <= 300 {
		p.subsumptionPairPool.Put(pairs)
	}
}

// Watched clause slice pool methods
func (p *SATPool) GetWatchedClauseSlice() []*WatchedClause {
	return p.watchedClauseSlicePool.Get().([]*WatchedClause)[:0]
}

func (p *SATPool) PutWatchedClauseSlice(slice []*WatchedClause) {
	if slice != nil && cap(slice) <= 100 {
		p.watchedClauseSlicePool.Put(slice)
	}
}

// Map pool methods
func (p *SATPool) GetStringBoolMap() map[string]bool {
	m := p.stringBoolMapPool.Get().(map[string]bool)
	for k := range m {
		delete(m, k)
	}
	return m
}

func (p *SATPool) PutStringBoolMap(m map[string]bool) {
	if m != nil && len(m) < 500 {
		p.stringBoolMapPool.Put(m)
	}
}

func (p *SATPool) GetStringIntMap() map[string]int {
	m := p.stringIntMapPool.Get().(map[string]int)
	for k := range m {
		delete(m, k)
	}
	return m
}

func (p *SATPool) PutStringIntMap(m map[string]int) {
	if m != nil && len(m) < 500 {
		p.stringIntMapPool.Put(m)
	}
}

func (p *SATPool) GetStringFloatMap() map[string]float64 {
	m := p.stringFloatMapPool.Get().(map[string]float64)
	for k := range m {
		delete(m, k)
	}
	return m
}

func (p *SATPool) PutStringFloatMap(m map[string]float64) {
	if m != nil && len(m) < 500 {
		p.stringFloatMapPool.Put(m)
	}
}

func (p *SATPool) GetIntClauseMap() map[int]*Clause {
	m := p.intClauseMapPool.Get().(map[int]*Clause)
	for k := range m {
		delete(m, k)
	}
	return m
}

func (p *SATPool) PutIntClauseMap(m map[int]*Clause) {
	if m != nil && len(m) < 500 {
		p.intClauseMapPool.Put(m)
	}
}

// Buffer pool methods for generic use
func (p *SATPool) GetSmallBuffer() []interface{} {
	return p.smallBufferPool.Get().([]interface{})[:0]
}

func (p *SATPool) PutSmallBuffer(buf []interface{}) {
	if buf != nil && cap(buf) <= 128 {
		p.smallBufferPool.Put(buf)
	}
}

func (p *SATPool) GetMediumBuffer() []interface{} {
	return p.mediumBufferPool.Get().([]interface{})[:0]
}

func (p *SATPool) PutMediumBuffer(buf []interface{}) {
	if buf != nil && cap(buf) <= 512 {
		p.mediumBufferPool.Put(buf)
	}
}

func (p *SATPool) GetLargeBuffer() []interface{} {
	return p.largeBufferPool.Get().([]interface{})[:0]
}

func (p *SATPool) PutLargeBuffer(buf []interface{}) {
	if buf != nil && cap(buf) <= 2048 {
		p.largeBufferPool.Put(buf)
	}
}

// Convenience methods for common patterns

// CloneAssignmentPooled creates a pooled clone of an assignment
func (p *SATPool) CloneAssignmentPooled(src Assignment) Assignment {
	dst := p.GetAssignment()
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

// MergeLiteralSlicesPooled efficiently merges literal slices using pooled memory
func (p *SATPool) MergeLiteralSlicesPooled(slices ...[]Literal) []Literal {
	totalLen := 0
	for _, slice := range slices {
		totalLen += len(slice)
	}

	result := p.GetLiteralSlice(totalLen)
	for _, slice := range slices {
		result = append(result, slice...)
	}
	return result
}

// Reset clears all pools - useful for testing or memory management
func (p *SATPool) Reset() {
	// Note: sync.Pool doesn't have a direct way to clear all objects
	// This creates new pools, allowing old ones to be GC'd
	*p = *NewSATPool()
}

// Stats returns pool usage statistics (for debugging/monitoring)
type PoolStats struct {
	PoolName    string
	Allocations int64 // Would need instrumentation to track
	Hits        int64 // Would need instrumentation to track
	Misses      int64 // Would need instrumentation to track
	CurrentSize int64 // Not directly available from sync.Pool
}

// GetStats returns statistics for all pools (would need instrumentation)
func (p *SATPool) GetStats() []PoolStats {
	// This would require instrumenting the pools to track statistics
	// For now, return empty slice - could be implemented with metrics
	return []PoolStats{}
}
