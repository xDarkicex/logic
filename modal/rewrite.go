package modal

import "github.com/xDarkicex/memory"

// Rewriter applies §14 eventuality/universality simplification and
// §15 independent-AP formula splitting in a single O(n) traversal.
type Rewriter struct {
	reg  *Registry
	pins *PINSRegistry
	pool *memory.Pool
}

// NewRewriter creates a Rewriter with its own internal PINS
// sharing the given registry for consistent formula IDs.
func NewRewriter(reg *Registry, pool *memory.Pool) *Rewriter {
	return &Rewriter{
		reg:  reg,
		pins: nil, // created lazily
		pool: pool,
	}
}

// ensurePINS creates the PINS registry on first use, sharing r.reg.
func (r *Rewriter) ensurePINS() {
	if r.pins == nil {
		r.pins = &PINSRegistry{
			reg:    r.reg,
			matrix: NewDepMatrix(r.reg.NextID()+256, 64, r.pool),
		}
	}
}

// Simplify applies §14 structural rewrite rules — properties computed
// bottom-up during the same traversal. Returns the canonical simplified formula.
// CC=5.
func (r *Rewriter) Simplify(f Formula) Formula {
	result, _, _ := r.simplifyProps(f)
	return r.reg.Intern(result)
}

// simplifyProps recursively simplifies f and returns (simplified, isEventual, isUniversal).
// Single pass, O(n). CC=7.
func (r *Rewriter) simplifyProps(f Formula) (Formula, bool, bool) {
	switch t := f.(type) {
	case Atom:
		return f, false, false

	case Not:
		inner, ev, un := r.simplifyProps(t.Formula)
		if inner != t.Formula {
			return r.reg.Intern(Not{Formula: inner}), ev, un
		}
		return f, ev, un

	case Box:
		inner, ev, un := r.simplifyProps(t.Formula)
		if un {
			return inner, ev, true
		}
		// G(F(e)) → F(e): Diamond is always eventual (§14)
		if _, ok := inner.(Diamond); ok && ev {
			return inner, ev, true
		}
		if inner != t.Formula {
			return r.reg.Intern(Box{Formula: inner, Rel: t.Rel}), ev, true
		}
		return f, ev, true

	case Diamond:
		inner, ev, un := r.simplifyProps(t.Formula)
		if ev {
			return inner, true, un
		}
		// F(G(u)) → G(u): Box is always universal (§14)
		if _, ok := inner.(Box); ok && un {
			return inner, true, un
		}
		if inner != t.Formula {
			return r.reg.Intern(Diamond{Formula: inner, Rel: t.Rel}), true, un
		}
		return f, true, un

	case And:
		left, lev, lun := r.simplifyProps(t.Left)
		right, rev, run := r.simplifyProps(t.Right)
		ev := lev || rev
		un := lun && run
		if left != t.Left || right != t.Right {
			return r.reg.Intern(And{Left: left, Right: right}), ev, un
		}
		return f, ev, un

	case Or:
		left, lev, lun := r.simplifyProps(t.Left)
		right, rev, run := r.simplifyProps(t.Right)
		ev := lev && rev
		un := lun || run
		if left != t.Left || right != t.Right {
			return r.reg.Intern(Or{Left: left, Right: right}), ev, un
		}
		return f, ev, un

	case Implies:
		ant, _, aun := r.simplifyProps(t.Antecedent)
		con, cev, _ := r.simplifyProps(t.Consequent)
		ev := aun || cev
		un := false
		if ant != t.Antecedent || con != t.Consequent {
			return r.reg.Intern(Implies{Antecedent: ant, Consequent: con}), ev, un
		}
		return f, ev, un

	case Iff:
		left, lev, lun := r.simplifyProps(t.Left)
		right, rev, run := r.simplifyProps(t.Right)
		ev := lev && rev
		un := lun && run
		if left != t.Left || right != t.Right {
			return r.reg.Intern(Iff{Left: left, Right: right}), ev, un
		}
		return f, ev, un

	case Next:
		inner, ev, un := r.simplifyProps(t.Formula)
		if inner != t.Formula {
			return r.reg.Intern(Next{Formula: inner}), ev, un
		}
		return f, ev, un

	case Until:
		left, _, _ := r.simplifyProps(t.Left)
		right, rev, _ := r.simplifyProps(t.Right)
		if left != t.Left || right != t.Right {
			return r.reg.Intern(Until{Left: left, Right: right}), rev, false
		}
		return f, rev, false
	}
	return f, false, false
}

// SplitConjunction decomposes a conjunction into independent components (§15).
// Conjuncts sharing no atomic propositions are split apart and checked independently.
// Uses PINS footprint for O(1) overlap detection per pair.
// CC=7.
func (r *Rewriter) SplitConjunction(f Formula) []Formula {
	conjuncts := r.collectConjuncts(f)
	if len(conjuncts) <= 1 {
		return []Formula{f}
	}
	n := len(conjuncts)

	footprints := memory.MustPoolSlice[[]uint64](r.pool, n)
	footprints = footprints[:n]
	for i, c := range conjuncts {
		footprints[i] = computeFootprint(c, r.pins, r.pool)
	}

	visited := memory.MustPoolSlice[bool](r.pool, n)
	visited = visited[:n]
	var components [][]Formula
	for i := 0; i < n; i++ {
		if visited[i] {
			continue
		}
		comp := dfsComponent(i, conjuncts, footprints, visited, r.pool)
		components = append(components, comp)
	}

	if len(components) <= 1 {
		return []Formula{f}
	}

	result := memory.MustPoolSlice[Formula](r.pool, len(components))
	result = result[:0]
	for _, comp := range components {
		conj := comp[0]
		for j := 1; j < len(comp); j++ {
			conj = r.reg.Intern(And{Left: conj, Right: comp[j]})
		}
		result = append(result, conj)
	}
	return result
}

func (r *Rewriter) collectConjuncts(f Formula) []Formula {
	result := memory.MustPoolSlice[Formula](r.pool, 8)
	result = result[:0]
	r.collectRec(f, &result)
	return result
}

func (r *Rewriter) collectRec(f Formula, result *[]Formula) {
	if a, ok := f.(And); ok {
		// Only split at the TOP level — nested ANDs are preserved as groups.
		// And(And(a,b), And(c,d)) → [And(a,b), And(c,d)] (2 components)
		*result = append(*result, a.Left, a.Right)
		return
	}
	*result = append(*result, f)
}

func dfsComponent(start int, conjuncts []Formula, footprints [][]uint64, visited []bool, pool *memory.Pool) []Formula {
	comp := memory.MustPoolSlice[Formula](pool, 8)
	comp = comp[:0]
	stack := memory.MustPoolSlice[int](pool, 16)
	stack = stack[:0]
	stack = append(stack, start)
	visited[start] = true
	for len(stack) > 0 {
		i := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		comp = append(comp, conjuncts[i])
		for j := 0; j < len(conjuncts); j++ {
			if visited[j] {
				continue
			}
			if footprintOverlap(footprints[i], footprints[j]) {
				visited[j] = true
				stack = append(stack, j)
			}
		}
	}
	return comp
}

func footprintOverlap(a, b []uint64) bool {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	for i := 0; i < n; i++ {
		if (a[i] & b[i]) != 0 {
			return true
		}
	}
	return false
}

func computeFootprint(f Formula, _ *PINSRegistry, pool *memory.Pool) []uint64 {
	words := maxAtomWords
	bits := memory.MustPoolSlice[uint64](pool, words)
	bits = bits[:words]
	footprintRec(f, bits)
	return bits
}

// maxAtomWords provides a generous upper bound for footprint bitmasks.
const maxAtomWords = 8 // 512 atomic propositions

func footprintRec(f Formula, bits []uint64) {
	switch t := f.(type) {
	case Atom:
		col := int(t.ID)
		if col/64 < len(bits) {
			bits[col/64] |= 1 << (uint(col) % 64)
		}
	case Not:
		footprintRec(t.Formula, bits)
	case Box:
		footprintRec(t.Formula, bits)
	case Diamond:
		footprintRec(t.Formula, bits)
	case Next:
		footprintRec(t.Formula, bits)
	case And:
		footprintRec(t.Left, bits)
		footprintRec(t.Right, bits)
	case Or:
		footprintRec(t.Left, bits)
		footprintRec(t.Right, bits)
	case Implies:
		footprintRec(t.Antecedent, bits)
		footprintRec(t.Consequent, bits)
	case Iff:
		footprintRec(t.Left, bits)
		footprintRec(t.Right, bits)
	case Until:
		footprintRec(t.Left, bits)
		footprintRec(t.Right, bits)
	}
}
