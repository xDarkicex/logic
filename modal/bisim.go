package modal

import (
	"github.com/xDarkicex/gobdd"
	"github.com/xDarkicex/memory"
)

// BisimContractor reduces a Kripke model via bisimulation contraction.
// Uses BDD signatures matching Spot's simulation-based reduction:
//   signature(s) = ∏ { bdd_var(class_of(dst)) : for each edge s → dst }
// Two worlds are bisimilar iff their BDD signatures are identical (O(1)).
type BisimContractor struct {
	ctx  *BDDCtx
	pool *memory.Pool
}

// NewBisimContractor creates a bisimulation contractor sharing a BDD context.
func NewBisimContractor(ctx *BDDCtx, pool *memory.Pool) *BisimContractor {
	return &BisimContractor{ctx: ctx, pool: pool}
}

// Contract computes the bisimulation quotient of m. CC=8.
func (bc *BisimContractor) Contract(m *Model) *Model {
	n := len(m.valuation)
	if n <= 1 {
		return m
	}
	// Step 1: initial partition by valuation (via BDD equivalence).
	classes := bc.initialByValuation(m, n)

	// Step 2: iterative refinement via BDD signatures.
	for changed := true; changed; {
		changed = bc.refineBySignature(m, classes)
	}
	return bc.buildReduced(m, classes)
}

// initialByValuation groups worlds with identical truth assignments. CC=5.
func (bc *BisimContractor) initialByValuation(m *Model, n int) []int32 {
	classes := memory.MustPoolSlice[int32](bc.pool, n)
	classes = classes[:n]
	classID := int32(0)
	for w := 0; w < n; w++ {
		assigned := false
		for v := 0; v < w; v++ {
			if valEqual(m.valuation, v, w) {
				classes[w] = classes[v]
				assigned = true
				break
			}
		}
		if !assigned {
			classes[w] = classID
			classID++
		}
	}
	return classes
}

// valEqual compares two valuation rows. CC=2.
func valEqual(val []TruthValueSlice, a, b int) bool {
	va := val[a]
	vb := val[b]
	if len(va) != len(vb) {
		return false
	}
	for i := range va {
		if va[i] != vb[i] {
			return false
		}
	}
	return true
}

// refineBySignature assigns each class a BDD variable, builds per-world
// signature BDDs, and splits classes where signatures differ. CC=9.
func (bc *BisimContractor) refineBySignature(m *Model, classes []int32) bool {
	n := len(classes)
	// At most n distinct classes exist. Use a dense var range [0, n).
	bc.ctx.ensureVars(int32(n))
	classVar := bc.denseClassVars(classes, int32(n))

	sigs := memory.MustPoolSlice[gobdd.NodeID](bc.pool, n)
	sigs = sigs[:n]
	for w := 0; w < n; w++ {
		sigs[w] = bc.worldSignature(m, World(w), classes, classVar)
	}

	old := memory.MustPoolSlice[int32](bc.pool, n)
	old = old[:n]
	copy(old, classes)

	nextID := maxClass32(classes) + 1
	newClasses := memory.MustPoolSlice[int32](bc.pool, n)
	newClasses = newClasses[:n]
	copy(newClasses, old)
	for w := 0; w < n; w++ {
		canon := int32(-1)
		for v := 0; v < w; v++ {
			if old[v] == old[w] && sigs[v] == sigs[w] {
				canon = newClasses[v]
				break
			}
		}
		if canon < 0 {
			newClasses[w] = nextID
			nextID++
		} else {
			newClasses[w] = canon
		}
	}
	// Detect structural splits: any old-class-mates now in different classes?
	changed := bc.hasSplit(old, newClasses)
	copy(classes, newClasses)
	bc.renumberClasses(classes) // always compact IDs
	return changed
}

// hasSplit returns true if any two worlds that shared an old class now differ. CC=3.
func (bc *BisimContractor) hasSplit(old, new_ []int32) bool {
	for i := 0; i < len(old); i++ {
		for j := i + 1; j < len(old); j++ {
			if old[i] == old[j] && new_[i] != new_[j] {
				return true
			}
		}
	}
	return false
}

// renumberClasses compacts class IDs to [0, k) in-place. CC=5.
func (bc *BisimContractor) renumberClasses(classes []int32) {
	next := int32(0)
	mx := maxClass32(classes)
	remap := memory.MustPoolSlice[int32](bc.pool, int(mx)+1)
	remap = remap[:int(mx)+1]
	for i := range remap {
		remap[i] = -1
	}
	for i, c := range classes {
		if remap[c] < 0 {
			remap[c] = next
			next++
		}
		classes[i] = remap[c]
	}
}

// denseClassVars maps each unique class ID to a dense BDD variable in [0, maxVar).
// Uses a compact Pool-backed array keyed by class ID. CC=4.
func (bc *BisimContractor) denseClassVars(classes []int32, maxVar int32) []int32 {
	// Find max class ID to size the mapping array.
	mx := maxClass32(classes)
	if mx < 0 {
		return nil
	}
	vars := memory.MustPoolSlice[int32](bc.pool, int(mx)+1)
	vars = vars[:int(mx)+1]
	for i := range vars {
		vars[i] = -1
	}
	next := int32(0)
	for _, c := range classes {
		if vars[c] < 0 {
			vars[c] = next
			next++
		}
	}
	return vars
}

// worldSignature builds the BDD signature for world w.
// For each relation, ORs the class vars of accessible worlds, then ANDs across relations.
// CC=6.
func (bc *BisimContractor) worldSignature(m *Model, w World, classes []int32, classVar []int32) gobdd.NodeID {
	const maxRel = 16
	var relBDDs [maxRel]gobdd.NodeID
	hasRel := memory.MustPoolSlice[bool](bc.pool, maxRel)
	hasRel = hasRel[:maxRel]
	hasEdge := false
	for _, e := range m.frame.edges {
		if e.Src != w || int(e.Rel) >= maxRel {
			continue
		}
		cv := classVar[classes[e.Dst]]
		if cv < 0 {
			continue
		}
		hasEdge = true
		bv := bc.ctx.bdd.Var(cv)
		if !hasRel[e.Rel] {
			relBDDs[e.Rel] = bv
			hasRel[e.Rel] = true
		} else {
			relBDDs[e.Rel] = bc.ctx.bdd.Or(relBDDs[e.Rel], bv)
		}
	}
	if !hasEdge {
		return gobdd.True
	}
	result := gobdd.True
	for r := 0; r < maxRel; r++ {
		if hasRel[r] {
			result = bc.ctx.bdd.And(result, relBDDs[r])
		}
	}
	return result
}

// buildReduced constructs the quotient model. CC=9.
func (bc *BisimContractor) buildReduced(m *Model, classes []int32) *Model {
	n := len(classes)
	mx := int(maxClass32(classes)) + 1
	seen := memory.MustPoolSlice[bool](bc.pool, mx)
	seen = seen[:mx]
	rep := memory.MustPoolSlice[World](bc.pool, mx)
	rep = rep[:mx]
	oldToNew := memory.MustPoolSlice[World](bc.pool, n)
	oldToNew = oldToNew[:n]
	newID := World(0)
	for w := 0; w < n; w++ {
		c := classes[w]
		if !seen[c] {
			seen[c] = true
			rep[c] = World(w)
		}
	}
	// Build old→new mapping using -1 sentinel (0 is a valid world ID).
	for w := 0; w < n; w++ {
		oldToNew[w] = World(1 << 30) // sentinel
	}
	for w := 0; w < n; w++ {
		c := classes[w]
		for x := 0; x < w; x++ {
			if classes[x] == c {
				oldToNew[w] = oldToNew[x]
				break
			}
		}
		if oldToNew[w] == World(1<<30) {
			oldToNew[w] = newID
			newID++
		}
	}

	newN := int(newID)
	newFrame := NewFrame(bc.pool, m.frame.arena)
	for i := 0; i < newN; i++ {
		newFrame.AddWorld()
	}
	newVal := memory.MustPoolSlice[TruthValueSlice](bc.pool, newN)
	newVal = newVal[:newN]
	for w := 0; w < n; w++ {
		if seen[classes[w]] {
			nw := oldToNew[w]
			newVal[nw] = m.valuation[w]
			seen[classes[w]] = false // only copy first
		}
	}

	// Add deduplicated edges.
	edgeSeen := memory.MustPoolSlice[bool](bc.pool, newN*newN*16)
	edgeSeen = edgeSeen[:newN*newN*16]
	stride := int32(newN)
	for _, e := range m.frame.edges {
		src := int32(oldToNew[e.Src])
		dst := int32(oldToNew[e.Dst])
		idx := src*stride*16 + dst*16 + int32(e.Rel)
		if !edgeSeen[idx] {
			edgeSeen[idx] = true
			newFrame.edges = append(newFrame.edges,
				Edge{Src: World(src), Dst: World(dst), Rel: e.Rel})
		}
	}

	return &Model{frame: newFrame, valuation: newVal}
}

func maxClass32(classes []int32) int32 {
	m := int32(-1)
	for _, c := range classes {
		if c > m {
			m = c
		}
	}
	return m
}
