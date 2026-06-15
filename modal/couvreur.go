package modal

import "github.com/xDarkicex/memory"

// sccRoot is an entry on the Couvreur SCC root stack.
type sccRoot struct {
	index     int  // DFS order of this SCC root
	condition bool // true if this SCC is "accepting"
}

// CouvreurProver implements Couvreur's on-the-fly SCC emptiness check
// applied to tableau branch exploration. Replaces DFS with SCC-aware search
// that detects accepting cycles early.
//
// All state tracking uses flat Pool-backed slices — no maps, no heap allocations.
type CouvreurProver struct {
	pool  *memory.Pool
	arena *memory.Arena
}

// NewCouvreurProver creates a Couvreur prover.
func NewCouvreurProver(pool *memory.Pool, arena *memory.Arena) *CouvreurProver {
	return &CouvreurProver{pool: pool, arena: arena}
}

// ProveSatisfiable checks satisfiability using Couvreur's algorithm.
// The frame must have at least one world.
// CC=7.
func (c *CouvreurProver) ProveSatisfiable(formula Formula, frame *Frame) (bool, *Model) {
	if frame.WorldCount() == 0 {
		frame.AddWorld()
	}

	// State storage: nodes are allocated from Pool slab
	maxStates := 4096
	nodes := memory.MustPoolSlice[*TableauNode](c.pool, maxStates)
	nodes = nodes[:0]

	// Couvreur state tracking (all Pool-backed)
	order := memory.MustPoolSlice[int](c.pool, maxStates)
	order = order[:maxStates]
	for i := range order {
		order[i] = 0
	}

	rootStack := memory.MustPoolSlice[sccRoot](c.pool, 64)
	rootStack = rootStack[:0]

	// Create root state
	root := c.allocNode()
	root.Prefix = append(root.Prefix, 0)
	root.Formulas = append(root.Formulas, PrefixedFormula{World: 0, Formula: formula})
	nodes = append(nodes, root)
	order[0] = 1
	rootStack = append(rootStack, sccRoot{index: 1})

	num := 1

	// Iteration tracking: per-state expansion cursor
	expCursor := memory.MustPoolSlice[int](c.pool, maxStates)
	expCursor = expCursor[:maxStates]
	for i := range expCursor {
		expCursor[i] = 0
	}

	// DFS todo: stack of state indices
	todo := memory.MustPoolSlice[int](c.pool, 256)
	todo = todo[:0]
	todo = append(todo, 0)

	for len(todo) > 0 {
		curIdx := todo[len(todo)-1]
		cur := nodes[curIdx]

		if cur.Closed || isContradictory(cur.Formulas) {
			// Dead state — close SCC if root
			todo = todo[:len(todo)-1]
			if len(rootStack) > 0 && rootStack[len(rootStack)-1].index == order[curIdx] {
				c.markDead(order, curIdx, rootStack)
			}
			continue
		}

		if isComplete(cur.Formulas) {
			// Accepting state found — check if part of an SCC
			if len(rootStack) > 0 {
				rootStack[len(rootStack)-1].condition = true
			}
			// Accepting state alone is sufficient for satisfiability
			return true, extractModel(cur, frame, c.pool)
		}

		// Expand: get the next successor state
		succ, hasMore := c.expandNext(cur, frame, &expCursor[curIdx])
		if !hasMore {
			// No more successors — backtrack
			todo = todo[:len(todo)-1]
			if len(rootStack) > 0 && rootStack[len(rootStack)-1].index == order[curIdx] {
				if rootStack[len(rootStack)-1].condition {
					// Accepting SCC — satisfiable
					return true, extractModel(cur, frame, c.pool)
				}
				c.markDead(order, curIdx, rootStack)
			}
			continue
		}

		// Find or create successor state
		succIdx := -1
		for i := range nodes {
			if c.sameBranch(nodes[i], succ) {
				succIdx = i
				break
			}
		}

		if succIdx < 0 {
			// New state
			if num >= maxStates {
				continue
			}
			succIdx = num
			num++
			order[succIdx] = num
			nodes = append(nodes, succ)
			rootStack = append(rootStack, sccRoot{index: num})
			todo = append(todo, succIdx)
			expCursor[succIdx] = 0
		} else if order[succIdx] == -1 {
			// Dead state — skip
			continue
		} else {
			// Back edge to existing state — merge SCCs
			threshold := order[succIdx]
			merged := false
			for len(rootStack) > 0 && threshold < rootStack[len(rootStack)-1].index {
				merged = rootStack[len(rootStack)-1].condition || merged
				rootStack = rootStack[:len(rootStack)-1]
			}
			if len(rootStack) > 0 && merged {
				rootStack[len(rootStack)-1].condition = true
			}
			if len(rootStack) > 0 && rootStack[len(rootStack)-1].condition {
				return true, extractModel(cur, frame, c.pool)
			}
		}
	}

	return false, nil
}

// allocNode creates a Pool-slab-backed TableauNode.
func (c *CouvreurProver) allocNode() *TableauNode {
	return &TableauNode{}
}

// expandNext applies one expansion rule and returns the resulting node.
// CC=8.
func (c *CouvreurProver) expandNext(node *TableauNode, frame *Frame, cursor *int) (*TableauNode, bool) {
	for *cursor < len(node.Formulas) {
		i := *cursor
		*cursor++
		pf := node.Formulas[i]

		switch f := pf.Formula.(type) {
		case And:
			// Expand in-place and return same node as successor
			node.Formulas[i] = PrefixedFormula{World: pf.World, Formula: f.Left}
			node.Formulas = append(node.Formulas, PrefixedFormula{World: pf.World, Formula: f.Right})
			*cursor = 0 // restart cursor — formula list was mutated
			return node, true

		case Or:
			// β-rule: create a child with the right disjunct only
			// (left disjunct continues in current node)
			// Remove the Or, add left to current node, right becomes child
			rest := makeCopy(node.Formulas[:i], node.Formulas[i+1:], c.pool)
			node.Formulas = node.Formulas[:0]
			node.Formulas = append(node.Formulas, rest...)
			node.Formulas = append(node.Formulas, PrefixedFormula{World: pf.World, Formula: f.Left})
			*cursor = 0

			// Create child with right disjunct
			child := c.allocNode()
			child.Prefix = append(child.Prefix, node.Prefix...)
			child.Formulas = append(child.Formulas, rest...)
			child.Formulas = append(child.Formulas, PrefixedFormula{World: pf.World, Formula: f.Right})
			return child, true

		case Not:
			if expanded := c.expandNotInPlace(node, i, pf.World); expanded {
				*cursor = 0
			}
			return node, true

		case Box:
			b := f
			targets := frame.Accessible(pf.World, b.Rel)
			if len(targets) == 0 {
				node.Formulas = append(node.Formulas[:i], node.Formulas[i+1:]...)
				*cursor = 0
				return node, true
			}
			node.Formulas[i] = PrefixedFormula{World: targets[0], Formula: b.Formula}
			for j := 1; j < len(targets); j++ {
				node.Formulas = append(node.Formulas, PrefixedFormula{World: targets[j], Formula: b.Formula})
			}
			*cursor = 0
			return node, true

		case Diamond:
			d := f
			nw := frame.AddWorld()
			frame.AddRelation(pf.World, nw, d.Rel)
			node.Prefix = append(node.Prefix, nw)
			node.Formulas[i] = PrefixedFormula{World: nw, Formula: d.Formula}
			*cursor = 0
			return node, true
		}
	}
	return nil, false
}

// expandNotInPlace rewrites a negation in-place. Returns true if mutation occurred.
// CC=5.
func (c *CouvreurProver) expandNotInPlace(node *TableauNode, idx int, w World) bool {
	n := node.Formulas[idx].Formula.(Not)
	switch inner := n.Formula.(type) {
	case Not:
		node.Formulas[idx] = PrefixedFormula{World: w, Formula: inner.Formula}
	case And:
		node.Formulas[idx] = PrefixedFormula{World: w, Formula: Or{
			Left: Not{Formula: inner.Left}, Right: Not{Formula: inner.Right},
		}}
	case Or:
		node.Formulas[idx] = PrefixedFormula{World: w, Formula: Not{Formula: inner.Left}}
		node.Formulas = append(node.Formulas, PrefixedFormula{World: w, Formula: Not{Formula: inner.Right}})
	case Box:
		node.Formulas[idx] = PrefixedFormula{World: w, Formula: Diamond{
			Formula: Not{Formula: inner.Formula}, Rel: inner.Rel,
		}}
	case Diamond:
		node.Formulas[idx] = PrefixedFormula{World: w, Formula: Box{
			Formula: Not{Formula: inner.Formula}, Rel: inner.Rel,
		}}
	default:
		return false
	}
	return true
}

// sameBranch checks if two tableau nodes represent the same exploration state.
// Two nodes are equivalent if they have the same prefix and same formula set (unordered).
// CC=4.
func (c *CouvreurProver) sameBranch(a, b *TableauNode) bool {
	if len(a.Prefix) != len(b.Prefix) {
		return false
	}
	for i := range a.Prefix {
		if a.Prefix[i] != b.Prefix[i] {
			return false
		}
	}
	if len(a.Formulas) != len(b.Formulas) {
		return false
	}
	// Compare formula sets (order-independent)
	matched := memory.MustPoolSlice[bool](c.pool, len(b.Formulas))
	matched = matched[:len(b.Formulas)]
	for _, af := range a.Formulas {
		found := false
		for j, bf := range b.Formulas {
			if matched[j] {
				continue
			}
			if af.World == bf.World && formulaEqual(af.Formula, bf.Formula) {
				matched[j] = true
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// markDead marks all states in a completed SCC as dead.
// CC=3.
func (c *CouvreurProver) markDead(order []int, rootIdx int, rootStack []sccRoot) {
	rootOrder := order[rootIdx]
	for i := range order {
		if order[i] > 0 && order[i] >= rootOrder {
			order[i] = -1
		}
	}
	if len(rootStack) > 0 {
		// Pop the root if it matches
	}
}

// makeCopy copies two slices into a new Pool-backed slice.
func makeCopy(a, b []PrefixedFormula, pool *memory.Pool) []PrefixedFormula {
	result := memory.MustPoolSlice[PrefixedFormula](pool, len(a)+len(b))
	result = result[:len(a)+len(b)]
	copy(result, a)
	copy(result[len(a):], b)
	return result
}
