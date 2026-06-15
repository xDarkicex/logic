package sat

import "github.com/xDarkicex/memory"

const discontain = -1

// VarHeap is a binary max-heap keyed by variable activity score.
// All backing arrays are allocated off-heap via Pool.
// Indices are integer variable handles; callers are responsible for
// name↔index mapping. Not safe for concurrent use.
//
// Time: push/pop/update O(log n), max/contains O(1).
// Space: O(n) where n is the maximum variable index.
type VarHeap struct {
	scores []float64 // activity score per variable index
	pos    []int     // heap position per variable, discontain if not in heap
	stack  []int     // heap-ordered variable indices (max at index 0)
	size   int       // number of elements currently in heap
	pool   *memory.Pool
}

// NewVarHeap creates a binary max-heap with initial capacity for n variables.
// Backing arrays are allocated from pool.
func NewVarHeap(n int, pool *memory.Pool) *VarHeap {
	if n < 1 {
		n = 16
	}
	h := &VarHeap{pool: pool}
	h.scores = memory.MustPoolSlice[float64](pool, n)[:n]
	h.pos = memory.MustPoolSlice[int](pool, n)[:n]
	h.stack = memory.MustPoolSlice[int](pool, n)[:0]
	for i := range h.pos {
		h.pos[i] = discontain
	}
	return h
}

// Max returns the variable with the highest score without removing it.
// Panics if the heap is empty.
func (h *VarHeap) Max() int {
	return h.stack[0]
}

// IsEmpty reports whether the heap contains no elements.
func (h *VarHeap) IsEmpty() bool {
	return h.size == 0
}

// Len returns the number of elements in the heap.
func (h *VarHeap) Len() int {
	return h.size
}

// Contains reports whether idx is currently in the heap.
func (h *VarHeap) Contains(idx int) bool {
	return idx < len(h.pos) && h.pos[idx] != discontain
}

// Score returns the activity score for idx.
func (h *VarHeap) Score(idx int) float64 {
	if idx < len(h.scores) {
		return h.scores[idx]
	}
	return 0
}

// siftUp bubbles the element at idx up to restore heap order.
// CC=3.
func (h *VarHeap) siftUp(idx int) {
	idxPos := h.pos[idx]
	idxScore := h.scores[idx]
	for idxPos > 0 {
		parentPos := (idxPos - 1) / 2
		parent := h.stack[parentPos]
		if h.scores[parent] >= idxScore {
			break
		}
		h.stack[idxPos] = parent
		h.pos[parent] = idxPos
		idxPos = parentPos
	}
	h.stack[idxPos] = idx
	h.pos[idx] = idxPos
}

// siftDown bubbles the element at idx down to restore heap order.
// CC=5.
func (h *VarHeap) siftDown(idx int) {
	idxPos := h.pos[idx]
	idxScore := h.scores[idx]
	end := h.size
	for {
		childPos := 2*idxPos + 1
		if childPos >= end {
			break
		}
		child := h.stack[childPos]
		childScore := h.scores[child]
		if siblingPos := childPos + 1; siblingPos < end {
			if sibling := h.stack[siblingPos]; h.scores[sibling] > childScore {
				child = sibling
				childPos = siblingPos
				childScore = h.scores[sibling]
			}
		}
		if childScore <= idxScore {
			break
		}
		h.stack[idxPos] = child
		h.pos[child] = idxPos
		idxPos = childPos
	}
	h.stack[idxPos] = idx
	h.pos[idx] = idxPos
}

// Push inserts idx into the heap with its current score.
// The score must be set via Update or SetScore before calling Push.
// CC=2.
func (h *VarHeap) Push(idx int) {
	if idx >= len(h.scores) {
		h.grow(idx + 1)
	}
	if h.Contains(idx) {
		return
	}
	h.pos[idx] = h.size
	if h.size >= len(h.stack) {
		h.growStack()
	}
	h.stack = h.stack[:h.size+1]
	h.stack[h.size] = idx
	h.siftUp(idx)
	h.size++
}

// PopMax removes and returns the variable with the highest score.
// Returns -1 if the heap is empty.
// CC=4.
func (h *VarHeap) PopMax() int {
	if h.size == 0 {
		return -1
	}
	idx := h.stack[0]
	h.size--
	last := h.stack[h.size]
	h.pos[last] = discontain
	if last == idx {
		return idx
	}
	h.pos[idx] = discontain
	h.stack[0] = last
	h.pos[last] = 0
	h.siftDown(last)
	return idx
}

// Pop removes a specific variable from the heap.
// CC=3.
func (h *VarHeap) Pop(idx int) {
	if !h.Contains(idx) {
		return
	}
	h.size--
	last := h.stack[h.size]
	h.pos[last] = discontain
	if last == idx {
		h.pos[idx] = discontain
		return
	}
	idxPos := h.pos[idx]
	h.pos[idx] = discontain
	h.stack[idxPos] = last
	h.pos[last] = idxPos
	h.siftUp(last)
	h.siftDown(last)
}

// Update changes the score of idx and restores heap order.
// If idx is not in the heap, it is inserted. CC=4.
func (h *VarHeap) Update(idx int, score float64) {
	if idx >= len(h.scores) {
		h.grow(idx + 1)
	}
	if !h.Contains(idx) {
		h.scores[idx] = score
		h.Push(idx)
		return
	}
	oldScore := h.scores[idx]
	if oldScore == score {
		return
	}
	h.scores[idx] = score
	if score > oldScore {
		h.siftUp(idx)
	} else {
		h.siftDown(idx)
	}
}

// Rebuild clears the heap and inserts all given indices with their current scores.
// This is O(m log m) where m is len(indices). CC=2.
func (h *VarHeap) Rebuild(indices []int) {
	h.stack = h.stack[:0]
	h.size = 0
	for i := range h.pos {
		h.pos[i] = discontain
	}
	for _, idx := range indices {
		if idx < len(h.scores) {
			h.Push(idx)
		}
	}
}

// Rescale multiplies all scores by factor. Used to prevent floating-point overflow
// when scores grow too large. CC=1.
func (h *VarHeap) Rescale(factor float64) {
	for i := range h.scores {
		h.scores[i] *= factor
	}
}

// Reset clears the heap, resetting size to zero and marking all positions as discontain.
// Score array is zeroed. CC=1.
func (h *VarHeap) Reset() {
	h.stack = h.stack[:0]
	h.size = 0
	for i := range h.pos {
		h.pos[i] = discontain
		h.scores[i] = 0
	}
}

// grow increases capacity to at least minVars.
// CC=2.
func (h *VarHeap) grow(minVars int) {
	oldLen := len(h.scores)
	newCap := oldLen * 2
	if newCap < minVars {
		newCap = minVars
	}
	if newCap < 16 {
		newCap = 16
	}
	h.resize(newCap)
}

// growStack ensures the stack slice can hold at least one more element.
// CC=1.
func (h *VarHeap) growStack() {
	if h.size < cap(h.stack) {
		return
	}
	newCap := cap(h.stack) * 2
	if newCap < 16 {
		newCap = 16
	}
	newStack := memory.MustPoolSlice[int](h.pool, newCap)[:h.size]
	copy(newStack, h.stack)
	h.stack = newStack
}

// resize allocates new backing arrays of the given capacity and copies existing data.
// CC=2.
func (h *VarHeap) resize(newCap int) {
	oldLen := len(h.scores)

	newScores := memory.MustPoolSlice[float64](h.pool, newCap)[:newCap]
	copy(newScores, h.scores)

	newPos := memory.MustPoolSlice[int](h.pool, newCap)[:newCap]
	copy(newPos, h.pos)
	for i := oldLen; i < newCap; i++ {
		newPos[i] = discontain
	}

	newStack := memory.MustPoolSlice[int](h.pool, newCap)[:h.size]
	copy(newStack, h.stack)

	h.scores = newScores
	h.pos = newPos
	h.stack = newStack
}
