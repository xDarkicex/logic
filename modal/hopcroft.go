package modal

import "github.com/xDarkicex/memory"

// DFAState is a state index in a deterministic finite automaton.
type DFAState int32

// DFATransition is a flat transition table: trans[state*symbolCount + symbol] = dst.
// Pool-backed, zero heap allocations.
type DFATransition struct {
	data        []DFAState
	symbolCount int32
}

// NewDFATransition creates a zero-initialized transition table.
func NewDFATransition(stateCount, symbolCount int, pool *memory.Pool) *DFATransition {
	n := stateCount * symbolCount
	data := memory.MustPoolSlice[DFAState](pool, n)
	data = data[:n]
	return &DFATransition{data: data, symbolCount: int32(symbolCount)}
}

// Set sets transition (state, symbol) → dst.
func (dt *DFATransition) Set(state, symbol int32, dst DFAState) {
	dt.data[state*dt.symbolCount+symbol] = dst
}

// Get returns the destination for (state, symbol).
func (dt *DFATransition) Get(state, symbol int32) DFAState {
	return dt.data[state*dt.symbolCount+symbol]
}

// DFAMinimizer implements Hopcroft's O(k·n·log n) partition refinement.
// All state sets, worklists, and partition tables use Pool-backed slices.
type DFAMinimizer struct {
	pool *memory.Pool
}

// NewDFAMinimizer creates a minimizer.
func NewDFAMinimizer(pool *memory.Pool) *DFAMinimizer {
	return &DFAMinimizer{pool: pool}
}

// Minimize returns the minimized DFA: a mapping from old states to new state IDs.
// States mapping to the same new ID can be merged. CC=9.
func (dm *DFAMinimizer) Minimize(stateCount int, accepting []bool, trans *DFATransition) []DFAState {
	n := int32(stateCount)
	sc := trans.symbolCount

	// Partitions: each state has a block ID.
	// Initial split: accepting (1) vs non-accepting (0).
	part := memory.MustPoolSlice[int32](dm.pool, int(n))
	part = part[:n]
	blockCount := dm.initPartition(part, accepting, n)

	// Block→size mapping.
	bSize := memory.MustPoolSlice[int32](dm.pool, int(n+1))
	bSize = bSize[:n+1]
	for _, p := range part {
		bSize[p]++
	}

	// Worklist: Pool-backed queue of block IDs to process.
	wq := dm.newWorklist(part, bSize, blockCount, n)

	// Per-symbol inverse adjacency: for each (sym, dst), precomputed source list.
	adj := dm.buildAdjacency(n, sc, trans)

	// Scratch arrays.
	touched := memory.MustPoolSlice[int32](dm.pool, int(blockCount)+16)
	mark := memory.MustPoolSlice[int32](dm.pool, int(n))
	mark = mark[:n]
	markGen := int32(0)

	for wq.len() > 0 {
		splitter := wq.pop()

		for sym := int32(0); sym < sc; sym++ {
			markGen++
			touched = touched[:0]

			// Preimage: states that transition into splitter on this symbol.
			for _, s := range splitter.elems {
				srcs := adj.incoming(s, sym, n, sc)
				for _, src := range srcs {
					if mark[src] != markGen {
						mark[src] = markGen
						touched = append(touched, int32(src))
					}
				}
			}

			// Find which blocks need splitting.
			if len(touched) == 0 {
				continue
			}
			bSize = dm.applySplits(part, bSize, touched, &blockCount, wq)
		}
	}

	return dm.compactClasses(part, blockCount)
}

// initPartition assigns initial block IDs. CC=3.
func (dm *DFAMinimizer) initPartition(part []int32, accepting []bool, n int32) int32 {
	hasAcc, hasNon := false, false
	for i := int32(0); i < n; i++ {
		if accepting[i] {
			hasAcc = true
		} else {
			hasNon = true
		}
	}
	if hasAcc && hasNon {
		for i := int32(0); i < n; i++ {
			if accepting[i] {
				part[i] = 1
			} else {
				part[i] = 0
			}
		}
		return 2
	}
	return 1
}

// blockQ is a Pool-backed queue of blocks to split.
type blockQ struct {
	blocks []struct {
		elems []int32
	}
	head int32
	tail int32
}

func (dm *DFAMinimizer) newWorklist(part, bSize []int32, blockCount, n int32) *blockQ {
	// Collect states per block.
	lists := memory.MustPoolSlice[[]int32](dm.pool, int(blockCount))
	lists = lists[:blockCount]
	for i := range lists {
		lists[i] = memory.MustPoolSlice[int32](dm.pool, int(bSize[i]))
		lists[i] = lists[i][:0]
	}
	for i := int32(0); i < n; i++ {
		b := part[i]
		lists[b] = append(lists[b], i)
	}

	bq := memory.MustPoolSlice[struct{ elems []int32 }](dm.pool, int(blockCount))
	bq = bq[:blockCount]
	for i := int32(0); i < blockCount; i++ {
		bq[i].elems = lists[i]
	}
	return &blockQ{blocks: bq, head: 0, tail: blockCount}
}

func (wq *blockQ) len() int32 { return wq.tail - wq.head }

func (wq *blockQ) pop() struct{ elems []int32 } {
	b := wq.blocks[wq.head]
	wq.head++
	return b
}

func (wq *blockQ) push(elems []int32, pool *memory.Pool) {
	if int(wq.tail) >= len(wq.blocks) {
		newCap := len(wq.blocks) * 2
		if newCap < 4 {
			newCap = 4
		}
		n := memory.MustPoolSlice[struct{ elems []int32 }](pool, newCap)
		n = n[:newCap]
		copy(n, wq.blocks)
		wq.blocks = n
	}
	wq.blocks[wq.tail].elems = elems
	wq.tail++
}

// adjacency stores incoming edges per (symbol, destination). CC=2.
type adjacency struct {
	sources []int32
	offsets []int32
}

// buildAdjacency precomputes inverse transitions. CC=5.
func (dm *DFAMinimizer) buildAdjacency(n, sc int32, trans *DFATransition) adjacency {
	total := n * sc
	counts := memory.MustPoolSlice[int32](dm.pool, int(n*sc))
	counts = counts[:total]
	// Count incoming edges per (sym, dst).
	for s := int32(0); s < n; s++ {
		for a := int32(0); a < sc; a++ {
			dst := trans.Get(s, a)
			counts[a*n+int32(dst)]++
		}
	}
	// Prefix sums → offsets.
	offsets := memory.MustPoolSlice[int32](dm.pool, int(total)+1)
	offsets = offsets[:total+1]
	sum := int32(0)
	for i := int32(0); i < total; i++ {
		offsets[i] = sum
		sum += counts[i]
	}
	offsets[total] = sum
	// Fill sources.
	sources := memory.MustPoolSlice[int32](dm.pool, int(sum))
	sources = sources[:sum]
	pos := memory.MustPoolSlice[int32](dm.pool, int(total))
	pos = pos[:total]
	copy(pos, offsets[:total])
	for s := int32(0); s < n; s++ {
		for a := int32(0); a < sc; a++ {
			dst := trans.Get(s, a)
			idx := a*n + int32(dst)
			sources[pos[idx]] = s
			pos[idx]++
		}
	}
	return adjacency{sources: sources, offsets: offsets}
}

// incoming returns source states for (dst, sym). CC=2.
func (adj adjacency) incoming(dst, sym, n, sc int32) []int32 {
	idx := sym*n + dst
	if int(idx)+1 >= len(adj.offsets) {
		return nil
	}
	start := adj.offsets[idx]
	end := adj.offsets[idx+1]
	return adj.sources[start:end]
}

// applySplits splits blocks that intersect the preimage. Returns updated bSize. CC=7.
func (dm *DFAMinimizer) applySplits(part, bSize []int32, touched []int32, blockCount *int32, wq *blockQ) []int32 {
	hitCount := memory.MustPoolSlice[int32](dm.pool, int(*blockCount)+16)
	hitCount = hitCount[:int(*blockCount)]
	hitList := memory.MustPoolSlice[int32](dm.pool, len(touched))
	hitList = hitList[:0]

	for _, s := range touched {
		b := part[s]
		if hitCount[b] == 0 {
			hitList = append(hitList, b)
		}
		hitCount[b]++
	}

	for _, b := range hitList {
		in := hitCount[b]
		if in == 0 || in == bSize[b] {
			continue
		}
		newB := *blockCount
		*blockCount++
		// Grow bSize if needed.
		for int(newB) >= len(bSize) {
			n := memory.MustPoolSlice[int32](dm.pool, len(bSize)*2)
			n = n[:len(bSize)*2]
			copy(n, bSize)
			bSize = n
		}
		out := bSize[b] - in
		bSize[b] = out
		bSize[newB] = in

		// Collect states for the new (smaller) block.
		newElems := memory.MustPoolSlice[int32](dm.pool, int(in))
		newElems = newElems[:0]
		for _, s := range touched {
			if part[s] == b {
				part[s] = newB
				newElems = append(newElems, s)
			}
		}

		if in < out {
			wq.push(newElems, dm.pool)
		} else {
			oldElems := memory.MustPoolSlice[int32](dm.pool, int(out))
			oldElems = oldElems[:0]
			for i := int32(0); i < int32(len(part)); i++ {
				if part[i] == b {
					oldElems = append(oldElems, i)
				}
			}
			wq.push(oldElems, dm.pool)
		}
	}
	return bSize
}

// compactClasses renumbers block IDs to dense [0..k). CC=3.
func (dm *DFAMinimizer) compactClasses(part []int32, blockCount int32) []DFAState {
	n := int32(len(part))
	result := memory.MustPoolSlice[DFAState](dm.pool, int(n))
	result = result[:n]
	m := int(blockCount) + 1
	remap := memory.MustPoolSlice[int32](dm.pool, m)
	remap = remap[:m]
	for i := range remap {
		remap[i] = -1
	}
	next := int32(0)
	for i := int32(0); i < n; i++ {
		b := part[i]
		if remap[b] < 0 {
			remap[b] = next
			next++
		}
		result[i] = DFAState(remap[b])
	}
	return result
}
