package modal

import "github.com/xDarkicex/memory"

// Level tracks which acceptance set is expected next in degeneralized Büchi.
// Range: 0..totalSets-1. Value totalSets means "accepting edge just taken."
type Level int32

// Degeneralizer converts generalized Büchi (n acceptance sets) to classic
// Büchi (1 set) via level tracking. Each acceptance set corresponds to
// one ◇-formula (eventuality) from the PROMISE decomposition (§18).
//
// Algorithm: states become (original, level) pairs. An edge visiting
// acceptance set `k` at level `k` advances to level `k+1`. When all
// sets have been visited in order, an accepting edge is emitted and
// the level resets to 0. Optimization: skip levels by jumping directly
// to the highest visited set.
type Degeneralizer struct {
	pool *memory.Pool
}

// NewDegeneralizer creates a degeneralizer.
func NewDegeneralizer(pool *memory.Pool) *Degeneralizer {
	return &Degeneralizer{pool: pool}
}

// Degeneralize expands n acceptance sets into level-tracked states and edges.
// edges[i] contains the set of acceptance indices visited on that transition.
// Returns expanded edges with acceptance flag and the level count.
//
// totalSets: number of acceptance sets (promises).
// stateCount: original number of states.
// edges: per-edge visited acceptance set indices.
// CC=9.
func (dg *Degeneralizer) Degeneralize(totalSets int, stateCount int, edges []struct {
	Src, Dst int
	Visited  []int
}) (newEdges []DegEdge, levelCount int) {
	if totalSets == 0 {
		// No acceptance sets: every state is accepting.
		newEdges = memory.MustPoolSlice[DegEdge](dg.pool, len(edges))
		newEdges = newEdges[:len(edges)]
		for i, e := range edges {
			newEdges[i] = DegEdge{Src: e.Src, Dst: e.Dst, Accept: true}
		}
		return newEdges, 1
	}

	// Each original state becomes totalSets copies (one per level).
	levelCount = totalSets
	totalStates := stateCount * totalSets

	// Count edges: each original edge produces totalSets degeneralized edges
	// (one for each source level).
	edgeCap := len(edges) * totalSets
	newEdges = memory.MustPoolSlice[DegEdge](dg.pool, edgeCap)
	newEdges = newEdges[:0]

	for _, e := range edges {
		for level := 0; level < totalSets; level++ {
			nextLev, accept := dg.advance(level, e.Visited, totalSets)
			newSrc := e.Src*totalSets + level
			newDst := e.Dst*totalSets + nextLev
			if nextLev == level && !accept {
				// Self-loop on level with no visited sets: skip to avoid bloat
				// by merging with the base edge.
			}
			newEdges = append(newEdges, DegEdge{
				Src:    newSrc,
				Dst:    newDst,
				Accept: accept,
			})
		}
	}

	// Deduplicate: if src and dst are the same, keep only one per (src, dst, accept).
	newEdges = dg.dedupEdges(newEdges, totalStates)
	return newEdges, levelCount
}

// advance computes the next level given visited acceptance sets.
// Returns (nextLevel, isAcceptingEdge). CC=5.
func (dg *Degeneralizer) advance(level int, visited []int, totalSets int) (int, bool) {
	if len(visited) == 0 {
		return level, false
	}
	// Find highest visited set for skip optimization.
	maxV := visited[0]
	for _, v := range visited[1:] {
		if v > maxV {
			maxV = v
		}
	}
	// Does this edge satisfy the expected set?
	if contains(visited, level) {
		next := maxV + 1
		if next >= totalSets {
			return 0, true // accepting
		}
		return next, false
	}
	// Also advance if we've passed the current level (skip optimization).
	if maxV >= level {
		next := maxV + 1
		if next >= totalSets {
			return 0, true
		}
		return next, false
	}
	return level, false
}

// contains returns true if s is in visited. CC=2.
func contains(visited []int, s int) bool {
	for _, v := range visited {
		if v == s {
			return true
		}
	}
	return false
}

// DegEdge is an edge in the degeneralized automaton.
type DegEdge struct {
	Src, Dst int
	Accept   bool
}

// dedupEdges removes duplicate edges with the same (src, dst, accept). CC=4.
func (dg *Degeneralizer) dedupEdges(edges []DegEdge, totalStates int) []DegEdge {
	stride := totalStates
	seen := memory.MustPoolSlice[bool](dg.pool, stride*stride*2)
	seen = seen[:stride*stride*2]
	result := memory.MustPoolSlice[DegEdge](dg.pool, len(edges))
	result = result[:0]
	for _, e := range edges {
		accBit := 0
		if e.Accept {
			accBit = 1
		}
		idx := e.Src*stride*2 + e.Dst*2 + accBit
		if !seen[idx] {
			seen[idx] = true
			result = append(result, e)
		}
	}
	return result
}

// MakeAcceptingSet creates acceptance set indices from the PROMISE decomposition.
// Each promise formula gets an index 0..n-1. Returns the set indices for a given
// set of satisfied promises.
func (dg *Degeneralizer) MakeAcceptingSet(satisfiedPromises []int, promiseMap []int) []int {
	result := memory.MustPoolSlice[int](dg.pool, len(satisfiedPromises))
	result = result[:0]
	for _, p := range satisfiedPromises {
		if p >= 0 && p < len(promiseMap) {
			result = append(result, promiseMap[p])
		}
	}
	return result
}
