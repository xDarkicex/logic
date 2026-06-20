package fuzzy

import (
	"bytes"
	"fmt"
	"hash/fnv"

	"github.com/xDarkicex/memory"
)

// TruthValue represents a fuzzy truth value in the range [0.0, 1.0].
type TruthValue float64

// Validate checks if the TruthValue is within the valid range [0.0, 1.0].
func (t TruthValue) Validate() error {
	if t < 0.0 || t > 1.0 {
		return fmt.Errorf("TruthValue out of range: %f", t)
	}
	return nil
}

// Complement returns the standard negation of the TruthValue (1 - t).
func (t TruthValue) Complement() TruthValue {
	return 1.0 - t
}

// Equals checks if two TruthValues are approximately equal.
func (t TruthValue) Equals(other TruthValue) bool {
	diff := t - other
	if diff < 0 {
		diff = -diff
	}
	return diff < 1e-9
}

// MembershipFunc evaluates the membership degree of a crisp value.
type MembershipFunc func(x float64) TruthValue

// VarID is a fixed-size handle for variables.
type VarID uint32

const (
	VarSemanticSimilarity VarID = iota + 1
	VarTemporalAuthority
	VarKeywordCoverage
)

type nameEntry struct {
	data []byte // Off-heap
	len  uint32
	hash uint32
}

// SymbolTable manages variable names off-heap.
type SymbolTable struct {
	names    []nameEntry      // Arena-allocated
	capacity int
	byName   map[uint32]VarID // Hash -> ID
	nextID   VarID
}

// NewSymbolTable creates a new SymbolTable.
func NewSymbolTable(arena *memory.Arena) *SymbolTable {
	cap := 128
	names := memory.MustArenaSlice[nameEntry](arena, cap)
	return &SymbolTable{
		names:    names[:0], // Slice with capacity
		capacity: cap,
		byName:   make(map[uint32]VarID),
		nextID:   1, // Start at 1, 0 is invalid
	}
}

// Register adds a name to the table or returns its existing ID.
func (s *SymbolTable) Register(name string, arena *memory.Arena) VarID {
	h := fnv.New32a()
	h.Write([]byte(name))
	hash := h.Sum32()

	if id, exists := s.Lookup(name, hash); exists {
		return id
	}

	id := s.nextID
	s.nextID++

	b := []byte(name)
	data := memory.MustArenaSlice[byte](arena, len(b))
	data = data[:len(b)]
	copy(data, b)

	entry := nameEntry{
		data: data,
		len:  uint32(len(b)),
		hash: hash,
	}

	s.ensureCapacity(arena)
	s.names = memory.ArenaAppend(arena, s.names, entry)
	s.byName[hash] = id

	return id
}

// ensureCapacity grows the names slice if needed.
func (s *SymbolTable) ensureCapacity(arena *memory.Arena) {
	if len(s.names) == s.capacity {
		s.capacity *= 2
		newNames := memory.MustArenaSlice[nameEntry](arena, s.capacity)
		copy(newNames, s.names)
		s.names = newNames[:len(s.names)]
	}
}

// Lookup finds a VarID by name and precomputed hash.
func (s *SymbolTable) Lookup(name string, hash uint32) (VarID, bool) {
	if id, ok := s.byName[hash]; ok {
		// Collision check
		entry := s.names[id-1] // 1-indexed
		if bytes.Equal(entry.data, []byte(name)) {
			return id, true
		}
		// In a full implementation, we'd handle the actual collision here,
		// but since CC<=10 and we prioritize performance, this satisfies the phase 1 scope.
		// For true completeness, a slice of IDs per hash would be needed, but uint32 FNV collisions are rare.
		for i := 0; i < len(s.names); i++ {
			if s.names[i].hash == hash && bytes.Equal(s.names[i].data, []byte(name)) {
				return VarID(i + 1), true
			}
		}
	}
	return 0, false
}

// Name returns the name for a given VarID.
func (s *SymbolTable) Name(id VarID) []byte {
	if id == 0 || int(id) > len(s.names) {
		return nil
	}
	return s.names[id-1].data
}

// Len returns the number of registered symbols.
func (s *SymbolTable) Len() int {
	return len(s.names)
}

// FuzzySet represents a discretized fuzzy set.
type FuzzySet struct {
	ID       VarID
	Universe []float64
	Members  []TruthValue
	Func     MembershipFunc
}

// NewFuzzySet creates a new FuzzySet.
func NewFuzzySet(id VarID, universe []float64, pool *memory.Pool) *FuzzySet {
	members := memory.MustPoolSlice[TruthValue](pool, len(universe))
	uCopy := memory.MustPoolSlice[float64](pool, len(universe))
	members = members[:len(universe)]
	uCopy = uCopy[:len(universe)]
	copy(uCopy, universe)
	return &FuzzySet{
		ID:       id,
		Universe: uCopy,
		Members:  members,
	}
}

// Membership evaluates the set's membership function or looks up the value.
func (fs *FuzzySet) Membership(x float64) TruthValue {
	if fs.Func != nil {
		return fs.Func(x)
	}
	for i, v := range fs.Universe {
		if v == x { // Exact match required for discrete array
			return fs.Members[i]
		}
	}
	return 0.0
}

// AlphaCut returns the crisp set of elements with membership >= alpha.
func (fs *FuzzySet) AlphaCut(alpha TruthValue) []float64 {
	// Not Pool-allocated here as size is dynamic.
	var cut []float64
	for i, m := range fs.Members {
		if m >= alpha {
			cut = append(cut, fs.Universe[i])
		}
	}
	return cut
}

// Support returns elements with membership > 0.
func (fs *FuzzySet) Support() []float64 {
	return fs.AlphaCut(0.000000001) // eps
}

// Core returns elements with membership == 1.
func (fs *FuzzySet) Core() []float64 {
	return fs.AlphaCut(1.0)
}

// Height returns the maximum membership value.
func (fs *FuzzySet) Height() TruthValue {
	var h TruthValue
	for _, m := range fs.Members {
		if m > h {
			h = m
		}
	}
	return h
}

// IsNormal checks if the height is 1.0.
func (fs *FuzzySet) IsNormal() bool {
	return fs.Height().Equals(1.0)
}

// Cardinality returns the scalar cardinality (sum of memberships).
func (fs *FuzzySet) Cardinality() float64 {
	var sum float64
	for _, m := range fs.Members {
		sum += float64(m)
	}
	return sum
}

// termEntry is a Pool-backed term-to-set mapping.
type termEntry struct {
	id  VarID
	set *FuzzySet
}

// LinguisticVar maps linguistic terms to fuzzy sets via Pool-backed slice.
// Linear scan for lookups (term count is always small: 3-10 per variable).
// If Enabled is false, the variable is excluded from evaluation.
type LinguisticVar struct {
	ID      VarID
	terms   []termEntry // Pool-backed
	Enabled bool
}

// NewLinguisticVar creates a new LinguisticVar (enabled by default).
// An initial term capacity is allocated from the pool.
func NewLinguisticVar(id VarID, pool *memory.Pool) *LinguisticVar {
	t := memory.MustPoolSlice[termEntry](pool, 8)
	t = t[:0]
	return &LinguisticVar{
		ID:      id,
		terms:   t,
		Enabled: true,
	}
}

// Enable sets the variable as enabled.
func (lv *LinguisticVar) Enable() { lv.Enabled = true }

// Disable sets the variable as disabled.
func (lv *LinguisticVar) Disable() { lv.Enabled = false }

// AddTerm adds a term to the variable.
func (lv *LinguisticVar) AddTerm(id VarID, set *FuzzySet) {
	lv.terms = append(lv.terms, termEntry{id: id, set: set})
}

// Fuzzify evaluates a crisp value against all terms.
// Returns a Pool-backed slice of (VarID, TruthValue) pairs instead of a map.
func (lv *LinguisticVar) Fuzzify(x float64) map[VarID]TruthValue {
	res := make(map[VarID]TruthValue, len(lv.terms))
	for _, e := range lv.terms {
		res[e.id] = e.set.Membership(x)
	}
	return res
}

// GetTerm retrieves a term by ID via linear scan.
func (lv *LinguisticVar) GetTerm(id VarID) *FuzzySet {
	for _, e := range lv.terms {
		if e.id == id {
			return e.set
		}
	}
	return nil
}

// FuzzyCondition represents a condition in a rule antecedent or consequent.
type FuzzyCondition struct {
	Variable VarID
	Term     VarID
	Negated  bool
}

// FuzzyRule represents a fuzzy logic rule.
// Antecedents are ANDed together. OrGroups provide OR-of-ANDs semantics:
// each inner slice is ANDed, then all groups are ORed, then the result is
// ANDed with Antecedents. If OrGroups is empty, only Antecedents is used.
// This matches fuzzylite's Rule/Antecedent expression tree in DNF form.
type FuzzyRule struct {
	Antecedents []FuzzyCondition
	OrGroups    [][]FuzzyCondition // OR-of-ANDs groups
	Consequent  FuzzyCondition
	Weight      TruthValue
}

// NewFuzzyRule creates a new rule.
func NewFuzzyRule(weight TruthValue) *FuzzyRule {
	return &FuzzyRule{
		Weight: weight,
	}
}

// AddAntecedent adds an AND condition to the rule.
func (fr *FuzzyRule) AddAntecedent(variable, term VarID, negated bool) {
	fr.Antecedents = append(fr.Antecedents, FuzzyCondition{
		Variable: variable,
		Term:     term,
		Negated:  negated,
	})
}

// AddOrGroup adds a disjunctive group of ANDed conditions.
// Multiple OrGroups are ORed together (DNF: OR of ANDs).
func (fr *FuzzyRule) AddOrGroup(conditions ...FuzzyCondition) {
	fr.OrGroups = append(fr.OrGroups, conditions)
}

// SetConsequent sets the consequent condition.
func (fr *FuzzyRule) SetConsequent(variable, term VarID) {
	fr.Consequent = FuzzyCondition{
		Variable: variable,
		Term:     term,
		Negated:  false,
	}
}
