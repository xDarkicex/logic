package modal

import (
	"github.com/xDarkicex/memory"
)

// PrefixedFormula annotates a formula with the world where it must hold.
type PrefixedFormula struct {
	World   World
	Formula Formula
}

// TableauNode is a node in the tableau proof tree.
// Prefix is Arena-backed (grow-only world path via diamond rules).
// Formulas is Pool-backed (variable-length formula set).
// The node struct itself is allocated from a Pool-backed slab.
type TableauNode struct {
	Prefix   []World
	Formulas []PrefixedFormula
	Children []*TableauNode
	Closed   bool
}

// slabSize is the number of nodes per pre-allocated slab.
// Slabs are never resized, so pointers remain valid.
const slabSize = 1024

// Prover holds allocators for tableau construction.
// Nodes are batch-allocated in fixed-size Pool-backed slabs.
// Slices within nodes are Pool/Arena-backed.
type Prover struct {
	slabs  [][]TableauNode // each slab is Pool-backed
	cur    []TableauNode   // current slab being filled
	free   int             // next free index in cur slab
	pool   *memory.Pool
	arena  *memory.Arena
}

// NewProver creates a Prover.
func NewProver(pool *memory.Pool, arena *memory.Arena) *Prover {
	first := memory.MustPoolSlice[TableauNode](pool, slabSize)
	return &Prover{
		slabs: [][]TableauNode{first[:0]},
		cur:   first,
		pool:  pool,
		arena: arena,
	}
}

// Close is a no-op — pools are managed externally.
func (p *Prover) Close() {}

// allocNode returns a pointer into a Pool-backed node slab.
// Slabs are never resized, so previously returned pointers stay valid.
func (p *Prover) allocNode() *TableauNode {
	if p.free >= slabSize {
		next := memory.MustPoolSlice[TableauNode](p.pool, slabSize)
		p.cur = next[:0]
		p.slabs = append(p.slabs, p.cur)
		p.free = 0
	}
	n := len(p.cur)
	p.cur = p.cur[:n+1]
	p.free = n + 1
	np := &p.cur[n]
	np.Formulas = memory.MustPoolSlice[PrefixedFormula](p.pool, 0)
	np.Prefix = memory.MustArenaSlice[World](p.arena, 8)
	np.Prefix = np.Prefix[:0]
	np.Children = nil
	np.Closed = false
	return np
}

// ProveSatisfiable checks whether a formula is satisfiable in the given frame.
// The frame must have at least one world (world 0). Additional worlds are created
// during tableau expansion. The frame's arena must outlive the returned model.
// CC=6.
func (p *Prover) ProveSatisfiable(formula Formula, frame *Frame) (bool, *Model) {
	if frame.WorldCount() == 0 {
		frame.AddWorld()
	}

	root := p.allocNode()
	root.Prefix = append(root.Prefix, 0)
	root.Formulas = append(root.Formulas, PrefixedFormula{World: 0, Formula: formula})

	return p.prove(root, frame)
}

// ProveValid returns true if the formula is valid in all Kripke models over K.
// CC=2.
func (p *Prover) ProveValid(formula Formula, frame *Frame) bool {
	sat, _ := p.ProveSatisfiable(Not{Formula: formula}, frame)
	return !sat
}

// ProveEntails returns true if premises logically entail conclusion.
// CC=3.
func (p *Prover) ProveEntails(premises []Formula, conclusion Formula, frame *Frame) bool {
	var conj Formula = Not{Formula: conclusion}
	for i := len(premises) - 1; i >= 0; i-- {
		conj = And{Left: premises[i], Right: conj}
	}
	sat, _ := p.ProveSatisfiable(conj, frame)
	return !sat
}

// prove runs DFS over tableau branches.
// CC=7.
func (p *Prover) prove(root *TableauNode, frame *Frame) (bool, *Model) {
	stack := memory.MustPoolSlice[*TableauNode](p.pool, 64)
	stack = stack[:0]
	stack = append(stack, root)

	for len(stack) > 0 {
		node := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		if node.Closed {
			continue
		}
		p.expandAll(node, frame, &stack)
		if node.Closed {
			continue
		}
		if isContradictory(node.Formulas) {
			continue
		}
		if isComplete(node.Formulas) {
			return true, extractModel(node, frame, p.pool)
		}
	}
	return false, nil
}

// expandAll repeatedly applies expansion rules until a fixpoint or branch.
// CC=5.
func (p *Prover) expandAll(node *TableauNode, frame *Frame, stack *[]*TableauNode) {
	for {
		changed := false
		for i := 0; i < len(node.Formulas); i++ {
			pf := node.Formulas[i]
			switch pf.Formula.(type) {
			case And:
				changed = p.expandAndRule(node, i, pf.World)
			case Or:
				changed = p.expandOrRule(node, i, pf.World, stack)
			case Not:
				changed = p.expandNotRule(node, i, pf.World)
			case Box:
				changed = p.expandBoxRule(node, i, pf.World, frame)
			case Diamond:
				changed = p.expandDiamondRule(node, i, pf.World, frame)
			}
			if changed {
				break
			}
		}
		if !changed {
			return
		}
	}
}

// expandAndRule splits φ∧ψ into φ and ψ. CC=2.
func (p *Prover) expandAndRule(node *TableauNode, idx int, w World) bool {
	a := node.Formulas[idx].Formula.(And)
	node.Formulas[idx] = PrefixedFormula{World: w, Formula: a.Left}
	node.Formulas = append(node.Formulas, PrefixedFormula{World: w, Formula: a.Right})
	return true
}

// expandOrRule branches on φ∨ψ. Left replaces node, right goes on stack. CC=4.
func (p *Prover) expandOrRule(node *TableauNode, idx int, w World, stack *[]*TableauNode) bool {
	o := node.Formulas[idx].Formula.(Or)
	n := len(node.Formulas)
	rest := memory.MustPoolSlice[PrefixedFormula](p.pool, n-1)
	rest = rest[:n-1]
	copy(rest, node.Formulas[:idx])
	copy(rest[idx:], node.Formulas[idx+1:])

	right := p.allocNode()
	right.Prefix = append(right.Prefix, node.Prefix...)
	right.Formulas = append(right.Formulas, rest...)
	right.Formulas = append(right.Formulas, PrefixedFormula{World: w, Formula: o.Right})
	*stack = append(*stack, right)

	node.Formulas = node.Formulas[:0]
	node.Formulas = append(node.Formulas, rest...)
	node.Formulas = append(node.Formulas, PrefixedFormula{World: w, Formula: o.Left})
	return true
}

// expandNotRule applies negation rewriting. CC=6.
func (p *Prover) expandNotRule(node *TableauNode, idx int, w World) bool {
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

// expandBoxRule applies □φ at all accessible worlds. CC=3.
func (p *Prover) expandBoxRule(node *TableauNode, idx int, w World, frame *Frame) bool {
	b := node.Formulas[idx].Formula.(Box)
	targets := frame.Accessible(w, b.Rel)
	if len(targets) == 0 {
		node.Formulas = append(node.Formulas[:idx], node.Formulas[idx+1:]...)
		return true
	}
	node.Formulas[idx] = PrefixedFormula{World: targets[0], Formula: b.Formula}
	for i := 1; i < len(targets); i++ {
		node.Formulas = append(node.Formulas, PrefixedFormula{World: targets[i], Formula: b.Formula})
	}
	return true
}

// expandDiamondRule applies ◇φ by creating a new accessible world. CC=2.
func (p *Prover) expandDiamondRule(node *TableauNode, idx int, w World, frame *Frame) bool {
	d := node.Formulas[idx].Formula.(Diamond)
	nw := frame.AddWorld()
	frame.AddRelation(w, nw, d.Rel)
	node.Prefix = memory.ArenaAppend(p.arena, node.Prefix, nw)
	node.Formulas[idx] = PrefixedFormula{World: nw, Formula: d.Formula}
	return true
}

// isContradictory returns true if formulas contain σ:P and σ:¬P. CC=3.
func isContradictory(formulas []PrefixedFormula) bool {
	type pair struct {
		w  World
		id uint32
	}
	n := len(formulas)
	if n > 32 {
		n = 32
	}
	pos := make(map[pair]bool, n)
	neg := make(map[pair]bool, n)
	for _, pf := range formulas {
		switch f := pf.Formula.(type) {
		case Atom:
			k := pair{pf.World, uint32(f.ID)}
			pos[k] = true
			if neg[k] {
				return true
			}
		case Not:
			if a, ok := f.Formula.(Atom); ok {
				k := pair{pf.World, uint32(a.ID)}
				neg[k] = true
				if pos[k] {
					return true
				}
			}
		}
	}
	return false
}

// isComplete returns true when all formulas are literals. CC=2.
func isComplete(formulas []PrefixedFormula) bool {
	for _, pf := range formulas {
		if !isLiteral(pf.Formula) {
			return false
		}
	}
	return true
}

// isLiteral returns true for atoms and negated atoms. CC=2.
func isLiteral(f Formula) bool {
	switch f.(type) {
	case Atom:
		return true
	}
	if n, ok := f.(Not); ok {
		_, ok = n.Formula.(Atom)
		return ok
	}
	return false
}

// extractModel builds a Model from a completed open branch.
func extractModel(node *TableauNode, frame *Frame, pool *memory.Pool) *Model {
	maxID := uint32(0)
	for _, pf := range node.Formulas {
		if a, ok := pf.Formula.(Atom); ok && uint32(a.ID) > maxID {
			maxID = uint32(a.ID)
		}
	}
	model := NewModel(frame, int(maxID)+1, pool, nil)
	for _, pf := range node.Formulas {
		if a, ok := pf.Formula.(Atom); ok {
			model.SetTruth(pf.World, a.ID, 1.0)
		}
	}
	return model
}
