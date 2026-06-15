package modal

import "github.com/xDarkicex/memory"

// CutPointRelabeler identifies Boolean subformulas safe to rename via
// Hopcroft-Tarjan articulation points on the formula AST. Only subformulas
// that are articulation points AND whose children are articulation points
// can be safely abstracted — preventing loss of shared variable dependencies.
type CutPointRelabeler struct {
	pool *memory.Pool
}

// NewCutPointRelabeler creates a relabeler.
func NewCutPointRelabeler(pool *memory.Pool) *CutPointRelabeler {
	return &CutPointRelabeler{pool: pool}
}

// astNode is a node in the formula AST graph. CC=2.
type astNode struct {
	f        Formula  // the formula at this node
	children []int32  // neighbor indices in undirected graph
	dfn      int32    // DFS discovery number (-1 = unvisited)
	low      int32    // low-link value
	parent   int32    // parent in DFS tree
}

// RelabelCandidates returns Boolean subformulas safe for renaming. CC=8.
func (cr *CutPointRelabeler) RelabelCandidates(f Formula) []Formula {
	nodes := cr.buildGraph(f)
	art := cr.articulationPoints(nodes)
	return cr.filterCandidates(nodes, art)
}

// buildGraph converts a formula tree to an undirected graph. CC=8.
func (cr *CutPointRelabeler) buildGraph(f Formula) []astNode {
	n := cr.countNodes(f)
	nodes := memory.MustPoolSlice[astNode](cr.pool, n)
	nodes = nodes[:0]

	var build func(Formula, int32) int32
	build = func(f Formula, parent int32) int32 {
		idx := int32(len(nodes))
		nodes = append(nodes, astNode{f: f, dfn: -1, parent: parent})

		switch t := f.(type) {
		case Atom:
		case Not:
			c := build(t.Formula, idx)
			cr.addEdge(nodes, idx, c)
		case And:
			lc, rc := build(t.Left, idx), build(t.Right, idx)
			cr.addEdge(nodes, idx, lc)
			cr.addEdge(nodes, idx, rc)
		case Or:
			lc, rc := build(t.Left, idx), build(t.Right, idx)
			cr.addEdge(nodes, idx, lc)
			cr.addEdge(nodes, idx, rc)
		case Implies:
			lc, rc := build(t.Antecedent, idx), build(t.Consequent, idx)
			cr.addEdge(nodes, idx, lc)
			cr.addEdge(nodes, idx, rc)
		case Iff:
			lc, rc := build(t.Left, idx), build(t.Right, idx)
			cr.addEdge(nodes, idx, lc)
			cr.addEdge(nodes, idx, rc)
		case Box:
			c := build(t.Formula, idx)
			cr.addEdge(nodes, idx, c)
		case Diamond:
			c := build(t.Formula, idx)
			cr.addEdge(nodes, idx, c)
		case Next:
			c := build(t.Formula, idx)
			cr.addEdge(nodes, idx, c)
		case Until:
			lc, rc := build(t.Left, idx), build(t.Right, idx)
			cr.addEdge(nodes, idx, lc)
			cr.addEdge(nodes, idx, rc)
		}
		return idx
	}
	build(f, -1)
	return nodes
}

// addEdge adds an undirected edge between two nodes. CC=2.
func (cr *CutPointRelabeler) addEdge(nodes []astNode, u, v int32) {
	nodes[u].children = append(nodes[u].children, v)
	nodes[v].children = append(nodes[v].children, u)
}

// countNodes counts formula tree nodes for pre-allocation. CC=5.
func (cr *CutPointRelabeler) countNodes(f Formula) int {
	n := 1
	switch t := f.(type) {
	case Atom:
	case Not:
		n += cr.countNodes(t.Formula)
	case And:
		n += cr.countNodes(t.Left) + cr.countNodes(t.Right)
	case Or:
		n += cr.countNodes(t.Left) + cr.countNodes(t.Right)
	case Implies:
		n += cr.countNodes(t.Antecedent) + cr.countNodes(t.Consequent)
	case Iff:
		n += cr.countNodes(t.Left) + cr.countNodes(t.Right)
	case Box:
		n += cr.countNodes(t.Formula)
	case Diamond:
		n += cr.countNodes(t.Formula)
	case Next:
		n += cr.countNodes(t.Formula)
	case Until:
		n += cr.countNodes(t.Left) + cr.countNodes(t.Right)
	}
	return n
}

// articulationPoints finds articulation points via DFS with low-link values. CC=8.
func (cr *CutPointRelabeler) articulationPoints(nodes []astNode) []bool {
	nn := len(nodes)
	art := memory.MustPoolSlice[bool](cr.pool, nn)
	art = art[:nn]
	time := int32(0)

	var dfs func(u int32)
	dfs = func(u int32) {
		nodes[u].dfn = time
		nodes[u].low = time
		time++
		childCount := int32(0)

		for _, v := range nodes[u].children {
			if nodes[v].dfn >= 0 {
				if v != nodes[u].parent && nodes[v].dfn < nodes[u].low {
					nodes[u].low = nodes[v].dfn
				}
				continue
			}
			nodes[v].parent = u
			childCount++
			dfs(v)
			if nodes[v].low < nodes[u].low {
				nodes[u].low = nodes[v].low
			}
			if nodes[u].parent >= 0 && nodes[v].low >= nodes[u].dfn {
				art[u] = true
			}
		}
		if nodes[u].parent < 0 && childCount > 1 {
			art[u] = true
		}
	}

	for i := int32(0); i < int32(nn); i++ {
		if nodes[i].dfn < 0 {
			dfs(i)
		}
	}
	return art
}

// filterCandidates returns Boolean subformulas where both the node and all
// its children are articulation points. CC=4.
func (cr *CutPointRelabeler) filterCandidates(nodes []astNode, art []bool) []Formula {
	result := memory.MustPoolSlice[Formula](cr.pool, len(nodes))
	result = result[:0]
	for i, node := range nodes {
		if !art[i] {
			continue
		}
		if !isBooleanFormula(node.f) {
			continue
		}
		allChildrenArt := true
		for _, c := range node.children {
			if !art[c] {
				allChildrenArt = false
				break
			}
		}
		if allChildrenArt {
			result = append(result, node.f)
		}
	}
	return result
}

// isBooleanFormula returns true if f is purely Boolean (no modal operators). CC=5.
func isBooleanFormula(f Formula) bool {
	switch f.(type) {
	case Atom:
		return true
	case Not:
		return isBooleanFormula(f.(Not).Formula)
	case And:
		af := f.(And)
		return isBooleanFormula(af.Left) && isBooleanFormula(af.Right)
	case Or:
		of := f.(Or)
		return isBooleanFormula(of.Left) && isBooleanFormula(of.Right)
	case Implies:
		im := f.(Implies)
		return isBooleanFormula(im.Antecedent) && isBooleanFormula(im.Consequent)
	case Iff:
		iff := f.(Iff)
		return isBooleanFormula(iff.Left) && isBooleanFormula(iff.Right)
	default:
		return false
	}
}
