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
// Stores edges in a flat Pool slice — no maps, no pointer chasing.
type Frame struct {
	worlds []World // Arena-backed, grow-only
	edges  []Edge  // Pool-backed, append-only during construction
	pool   *memory.Pool
	arena  *memory.Arena
}

// NewFrame creates a Frame with the given allocators.
// The worlds slice is Arena-backed. The edges slice is Pool-backed.
func NewFrame(pool *memory.Pool, arena *memory.Arena) *Frame {
	worlds := memory.MustArenaSlice[World](arena, 64)
	worlds = worlds[:0]
	edges := memory.MustPoolSlice[Edge](pool, 256)
	edges = edges[:0]
	return &Frame{
		worlds: worlds,
		edges:  edges,
		pool:   pool,
		arena:  arena,
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
func NewModel(frame *Frame, numVars int, pool *memory.Pool, arena *memory.Arena) *Model {
	wc := frame.WorldCount()
	val := memory.MustArenaSlice[TruthValueSlice](arena, wc)
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
