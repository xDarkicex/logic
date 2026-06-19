package modal

import (
	"github.com/xDarkicex/logic/fuzzy"
	"github.com/xDarkicex/memory"
)

// Edge is a directed accessibility relation between two worlds.
// All edges are stored in a contiguous Pool-backed slice for cache-friendly iteration.
type Edge struct {
	Src World
	Dst World
	Rel RelationType
}

// Frame is a Kripke frame: a set of worlds and accessibility relations between them.
// Stores edges in flat Pool slices — no maps, no pointer chasing.
// Weighted edges support the fuzzy-modal bridge (Phase 6).
type Frame struct {
	worlds        []World        // Arena-backed, grow-only
	edges         []Edge         // Pool-backed, unweighted edges
	weightedEdges []WeightedEdge // Pool-backed, weighted edges
	pool          *memory.Pool
	arena         *memory.Arena
}

// WeightedEdge is a directed accessibility relation with a fuzzy weight in [0,1].
// Used by the fuzzy-modal bridge for daemon hop expansion (etaWhy=0.7, etaHow=0.4).
type WeightedEdge struct {
	Src    World
	Dst    World
	Rel    RelationType
	Weight TruthValue
}

// WeightedTarget is a destination world with its edge weight.
type WeightedTarget struct {
	Dst    World
	Weight TruthValue
}

// NewFrame creates a Frame with the given allocators.
// The worlds slice is Arena-backed. The edges slices are Pool-backed.
func NewFrame(pool *memory.Pool, arena *memory.Arena) *Frame {
	worlds := memory.MustArenaSlice[World](arena, 64)
	worlds = worlds[:0]
	edges := memory.MustPoolSlice[Edge](pool, 256)
	edges = edges[:0]
	wEdges := memory.MustPoolSlice[WeightedEdge](pool, 64)
	wEdges = wEdges[:0]
	return &Frame{
		worlds:        worlds,
		edges:         edges,
		weightedEdges: wEdges,
		pool:          pool,
		arena:         arena,
	}
}

// AddWorld adds a new world to the frame and returns its handle.
func (f *Frame) AddWorld() World {
	w := World(len(f.worlds))
	f.worlds = memory.ArenaAppend(f.arena, f.worlds, w)
	return w
}

// WorldCount returns the number of worlds in the frame.
func (f *Frame) WorldCount() int { return len(f.worlds) }

// AddRelation adds a directed edge from src to dst with the given relation type.
func (f *Frame) AddRelation(src, dst World, rel RelationType) {
	f.edges = append(f.edges, Edge{Src: src, Dst: dst, Rel: rel})
}

// AddWeightedRelation adds a weighted edge for the fuzzy-modal bridge.
// The weight must be in [0, 1].
func (f *Frame) AddWeightedRelation(src, dst World, rel RelationType, weight TruthValue) {
	f.weightedEdges = append(f.weightedEdges, WeightedEdge{
		Src: src, Dst: dst, Rel: rel, Weight: weight,
	})
}

// WeightedAccessible returns all worlds accessible from w with their edge weights.
// Searches both unweighted (weight=1.0) and weighted edges. O(E).
func (f *Frame) WeightedAccessible(w World, rel RelationType) []WeightedTarget {
	count := 0
	for i := range f.edges {
		if f.edges[i].Src == w && f.edges[i].Rel == rel {
			count++
		}
	}
	for i := range f.weightedEdges {
		if f.weightedEdges[i].Src == w && f.weightedEdges[i].Rel == rel {
			count++
		}
	}
	if count == 0 {
		return nil
	}
	result := memory.MustPoolSlice[WeightedTarget](f.pool, count)
	result = result[:0]
	for i := range f.edges {
		if f.edges[i].Src == w && f.edges[i].Rel == rel {
			result = append(result, WeightedTarget{Dst: f.edges[i].Dst, Weight: 1.0})
		}
	}
	for i := range f.weightedEdges {
		if f.weightedEdges[i].Src == w && f.weightedEdges[i].Rel == rel {
			result = append(result, WeightedTarget{
				Dst: f.weightedEdges[i].Dst, Weight: f.weightedEdges[i].Weight,
			})
		}
	}
	return result
}

// WeightedEdgeCount returns the number of weighted edges.
func (f *Frame) WeightedEdgeCount() int { return len(f.weightedEdges) }

// Accessible returns all worlds accessible from w via the given relation type.
// O(E) linear scan — E is typically small (< 1000 edges).
func (f *Frame) Accessible(w World, rel RelationType) []World {
	// Count first to pre-allocate exactly
	count := 0
	for i := range f.edges {
		if f.edges[i].Src == w && f.edges[i].Rel == rel {
			count++
		}
	}
	if count == 0 {
		return nil
	}
	result := memory.MustPoolSlice[World](f.pool, count)
	result = result[:0]
	for i := range f.edges {
		if f.edges[i].Src == w && f.edges[i].Rel == rel {
			result = append(result, f.edges[i].Dst)
		}
	}
	return result
}

// AccessibleSet returns a WorldSet bit-vector of all worlds accessible from w via rel.
// O(E) to build, then O(1) membership tests. Pool-backed.
func (f *Frame) AccessibleSet(w World, rel RelationType, pool *memory.Pool) *WorldSet {
	ws := NewWorldSet(len(f.worlds), pool)
	for i := range f.edges {
		if f.edges[i].Src == w && f.edges[i].Rel == rel {
			ws.Add(f.edges[i].Dst)
		}
	}
	for i := range f.weightedEdges {
		if f.weightedEdges[i].Src == w && f.weightedEdges[i].Rel == rel {
			ws.Add(f.weightedEdges[i].Dst)
		}
	}
	return ws
}

// IsAccessible returns true if dst is accessible from src via rel.
func (f *Frame) IsAccessible(src, dst World, rel RelationType) bool {
	for i := range f.edges {
		if f.edges[i].Src == src && f.edges[i].Dst == dst && f.edges[i].Rel == rel {
			return true
		}
	}
	return false
}

// ReflexiveClosure adds self-loops for every world for the given relation type.
// Does not duplicate existing edges.
func (f *Frame) ReflexiveClosure(rel RelationType) {
	for _, w := range f.worlds {
		if !f.IsAccessible(w, w, rel) {
			f.AddRelation(w, w, rel)
		}
	}
}

// SymmetricClosure ensures that if src→dst exists for rel, dst→src also exists.
func (f *Frame) SymmetricClosure(rel RelationType) {
	n := len(f.edges)
	for i := 0; i < n; i++ {
		e := f.edges[i]
		if e.Rel == rel && !f.IsAccessible(e.Dst, e.Src, rel) {
			f.AddRelation(e.Dst, e.Src, rel)
		}
	}
}

// TransitiveClosure adds edges to make the relation transitive (Floyd-Warshall).
// O(W³) time, O(E) space — use only on small frames.
func (f *Frame) TransitiveClosure(rel RelationType) {
	wc := len(f.worlds)
	if wc == 0 {
		return
	}
	// Build reachability matrix using Pool-backed bool slice
	reach := memory.MustPoolSlice[bool](f.pool, wc*wc)
	reach = reach[:wc*wc]
	idx := func(i, j World) int { return int(i)*wc + int(j) }

	// Initialize with existing edges
	for i := range f.edges {
		e := f.edges[i]
		if e.Rel == rel {
			reach[idx(e.Src, e.Dst)] = true
		}
	}
	// Floyd-Warshall
	for k := World(0); k < World(wc); k++ {
		for i := World(0); i < World(wc); i++ {
			if !reach[idx(i, k)] {
				continue
			}
			for j := World(0); j < World(wc); j++ {
				if reach[idx(k, j)] {
					reach[idx(i, j)] = true
				}
			}
		}
	}
	// Add edges for newly reachable pairs
	for i := World(0); i < World(wc); i++ {
		for j := World(0); j < World(wc); j++ {
			if i != j && reach[idx(i, j)] && !f.IsAccessible(i, j, rel) {
				f.AddRelation(i, j, rel)
			}
		}
	}
}

// Model is a Kripke model: a frame plus a valuation assigning truth values
// to atomic propositions at each world.
// The valuation is an Arena-backed slice of slices: valuation[world][varID].
type Model struct {
	frame     *Frame
	valuation []TruthValueSlice // Arena-backed, indexed by world
	numVars   int
	arena     *memory.Arena
}

// TruthValueSlice is a Pool-backed slice of TruthValues for fast indexed access.
type TruthValueSlice []TruthValue

// NewModel creates a Model with the given frame and variable count.
// The valuation is pre-allocated for all worlds × variables.
// If arena is nil, Pool-backed storage is used for the outer valuation slice.
func NewModel(frame *Frame, numVars int, pool *memory.Pool, arena *memory.Arena) *Model {
	wc := frame.WorldCount()
	var val []TruthValueSlice
	if arena != nil {
		val = memory.MustArenaSlice[TruthValueSlice](arena, wc)
	} else {
		val = memory.MustPoolSlice[TruthValueSlice](pool, wc)
	}
	val = val[:wc]
	for i := 0; i < wc; i++ {
		slice := memory.MustPoolSlice[TruthValue](pool, numVars)
		slice = slice[:numVars]
		val[i] = slice
	}
	return &Model{
		frame:     frame,
		valuation: val,
		numVars:   numVars,
		arena:     arena,
	}
}

// SetTruth sets the truth value of atom at world w.
func (m *Model) SetTruth(w World, atom fuzzy.VarID, tv TruthValue) {
	if int(w) < len(m.valuation) && int(atom) < m.numVars {
		m.valuation[w][atom] = tv
	}
}

// Truth returns the truth value of atom at world w.
func (m *Model) Truth(w World, atom fuzzy.VarID) TruthValue {
	if int(w) < len(m.valuation) && int(atom) < m.numVars {
		return m.valuation[w][atom]
	}
	return 0.0
}

// Frame returns the model's frame.
func (m *Model) Frame() *Frame { return m.frame }

// NumVars returns the number of atomic proposition variables.
func (m *Model) NumVars() int { return m.numVars }

// Valuation returns the model's valuation matrix (indexed by world, then by variable).
// Returns the internal slice directly for read-only access. Callers must not mutate.
func (m *Model) Valuation() []TruthValueSlice {
	return m.valuation
}
