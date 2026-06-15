package modal

import (
	"github.com/xDarkicex/gobdd"
	"github.com/xDarkicex/logic/fuzzy"
	"github.com/xDarkicex/memory"
)

// BDDCtx bridges modal formulas to GOBDD for O(1) Boolean equivalence.
// All backing slices use Pool — zero heap allocations.
type BDDCtx struct {
	bdd        *gobdd.BDD
	pool       *memory.Pool
	atomVars   []int32     // atomVars[fuzzy.VarID] = bddVar, -1 = unassigned
	skelSeen   []skelEntry // persistent modal-subformula → var cache
	isopCache  []Formula   // ISOP memo table (indexed by NodeID)
	isopCached []bool      // true if isopCache[node] is valid
	nextVar    int32       // next available BDD variable
	capacity   int32       // current BDD var count
}

type skelEntry struct {
	f Formula
	v int32
}

// NewBDDCtx creates a BDD context with initial variable capacity.
// Variables are added lazily as atoms are encountered.
func NewBDDCtx(initialVars int, pool *memory.Pool) *BDDCtx {
	if initialVars < 4 {
		initialVars = 4
	}
	bdd := gobdd.New(initialVars, pool)
	av := memory.MustPoolSlice[int32](pool, 16)
	av = av[:16]
	for i := range av {
		av[i] = -1
	}
	se := memory.MustPoolSlice[skelEntry](pool, 64)
	se = se[:0]
	return &BDDCtx{
		bdd:      bdd,
		pool:     pool,
		atomVars: av,
		skelSeen: se,
		nextVar:  0,
		capacity: int32(initialVars),
	}
}

// BDD returns the underlying GOBDD manager for advanced operations.
func (ctx *BDDCtx) BDD() *gobdd.BDD { return ctx.bdd }

// ensureVars grows the BDD variable count if needed. CC=2.
func (ctx *BDDCtx) ensureVars(n int32) {
	if n <= ctx.capacity {
		return
	}
	ctx.bdd.ExtVarNum(n - ctx.capacity)
	ctx.capacity = n
}

// atomVar returns the BDD variable for a fuzzy VarID.
// Assigns a new variable on first encounter. CC=4.
func (ctx *BDDCtx) atomVar(id uint32) int32 {
	if int(id) >= len(ctx.atomVars) {
		ctx.growAtomVars(int(id) + 1)
	}
	idx := ctx.atomVars[id]
	if idx >= 0 {
		return idx
	}
	idx = ctx.nextVar
	ctx.nextVar++
	ctx.ensureVars(ctx.nextVar)
	ctx.atomVars[id] = idx
	return idx
}

// growAtomVars expands the atom-to-variable mapping slice. CC=2.
func (ctx *BDDCtx) growAtomVars(minLen int) {
	newLen := len(ctx.atomVars) * 2
	if newLen < minLen {
		newLen = minLen
	}
	n := memory.MustPoolSlice[int32](ctx.pool, newLen)
	n = n[:newLen]
	copy(n, ctx.atomVars)
	for i := len(ctx.atomVars); i < newLen; i++ {
		n[i] = -1
	}
	ctx.atomVars = n
}

// ToBDD converts a purely Boolean formula to a BDD node.
// Panics if the formula contains modal operators (Box, Diamond, Next, Until).
// CC=6.
func (ctx *BDDCtx) ToBDD(f Formula) gobdd.NodeID {
	switch t := f.(type) {
	case Atom:
		v := ctx.atomVar(uint32(t.ID))
		return ctx.bdd.Var(v)
	case Not:
		return ctx.bdd.Not(ctx.ToBDD(t.Formula))
	case And:
		return ctx.bdd.And(ctx.ToBDD(t.Left), ctx.ToBDD(t.Right))
	case Or:
		return ctx.bdd.Or(ctx.ToBDD(t.Left), ctx.ToBDD(t.Right))
	case Implies:
		return ctx.bdd.Implies(ctx.ToBDD(t.Antecedent), ctx.ToBDD(t.Consequent))
	case Iff:
		return ctx.bdd.Equiv(ctx.ToBDD(t.Left), ctx.ToBDD(t.Right))
	default:
		panic("BDDCtx.ToBDD: non-Boolean subformula — use Skeleton for modal formulas")
	}
}

// Skeleton extracts the Boolean skeleton: replaces each maximal modal subformula
// with a fresh BDD variable. The same modal subformula always maps to the same
// variable across calls (cached in skelSeen). Returns the skeleton BDD.
// Use ResetSkeleton to clear the cache between unrelated formula sets. CC=10.
func (ctx *BDDCtx) Skeleton(f Formula) (root gobdd.NodeID) {
	var collect func(Formula) gobdd.NodeID
	collect = func(f Formula) gobdd.NodeID {
		switch t := f.(type) {
		case Atom:
			v := ctx.atomVar(uint32(t.ID))
			return ctx.bdd.Var(v)
		case Not:
			return ctx.bdd.Not(collect(t.Formula))
		case And:
			return ctx.bdd.And(collect(t.Left), collect(t.Right))
		case Or:
			return ctx.bdd.Or(collect(t.Left), collect(t.Right))
		case Implies:
			return ctx.bdd.Implies(collect(t.Antecedent), collect(t.Consequent))
		case Iff:
			return ctx.bdd.Equiv(collect(t.Left), collect(t.Right))
		default:
			for _, e := range ctx.skelSeen {
				if formulaEqual(e.f, t) {
					return ctx.bdd.Var(e.v)
				}
			}
			idx := ctx.nextVar
			ctx.nextVar++
			ctx.ensureVars(ctx.nextVar)
			ctx.skelSeen = append(ctx.skelSeen, skelEntry{f: t, v: idx})
			return ctx.bdd.Var(idx)
		}
	}
	root = collect(f)
	return
}

// ResetSkeleton clears the modal-subformula-to-variable cache.
// Call this between processing unrelated formula sets.
func (ctx *BDDCtx) ResetSkeleton() {
	ctx.skelSeen = ctx.skelSeen[:0]
}

// SkeletonSubs returns the list of modal subformulas that were assigned
// BDD variables by Skeleton, in order of first occurrence.
func (ctx *BDDCtx) SkeletonSubs() []Formula {
	subs := memory.MustPoolSlice[Formula](ctx.pool, len(ctx.skelSeen))
	subs = subs[:len(ctx.skelSeen)]
	for i, e := range ctx.skelSeen {
		subs[i] = e.f
	}
	return subs
}


// Equiv returns true if f and g are logically equivalent (O(1) after BDD construction).
// Both formulas must be purely Boolean. CC=2.
func (ctx *BDDCtx) Equiv(f, g Formula) bool {
	a := ctx.ToBDD(f)
	b := ctx.ToBDD(g)
	return a == b
}

// IsTautology returns true if f is a Boolean tautology. CC=2.
func (ctx *BDDCtx) IsTautology(f Formula) bool {
	node := ctx.ToBDD(f)
	return node == gobdd.True
}

// IsContradiction returns true if f is a Boolean contradiction. CC=2.
func (ctx *BDDCtx) IsContradiction(f Formula) bool {
	node := ctx.ToBDD(f)
	return node == gobdd.False
}

// EquivSkel returns true if two modal formulas have equivalent Boolean skeletons.
// Modal subformulas are treated as independent variables. CC=3.
func (ctx *BDDCtx) EquivSkel(f, g Formula) bool {
	a := ctx.Skeleton(f)
	b := ctx.Skeleton(g)
	return a == b
}

// NodeCount returns the number of BDD nodes allocated.
func (ctx *BDDCtx) NodeCount() int { return ctx.bdd.NodeCount() }

// VarCount returns the number of BDD variables in use.
func (ctx *BDDCtx) VarCount() int32 { return ctx.nextVar }

// BDDEquiv returns true if two BDD nodes represent the same function.
// O(1) — BDDs are canonical. CC=2.
func BDDEquiv(a, b gobdd.NodeID) bool { return a == b }


// --- ISOP: Minato's Irredundant Sum-of-Products ---
//
// Converts a BDD to a minimized DNF formula. Uses recursive decomposition
// with Pool-backed memoization to avoid exponential blowup.
// Output Atoms use VarID = BDD variable index.
//
// Reference: S. Minato, "Fast Generation of Irredundant Sum-of-Products
// Forms from Binary Decision Diagrams," 1992.

// ISOP returns an irredundant sum-of-products for the BDD node.
// CC=5.
func (ctx *BDDCtx) ISOP(node gobdd.NodeID) Formula {
	ctx.ensureISOPCache(int(node) + 1)
	if ctx.isopCached[node] {
		return ctx.isopCache[node]
	}
	result := ctx.isopRec(node)
	ctx.isopCache[node] = result
	ctx.isopCached[node] = true
	return result
}

// isopRec is the recursive Minato ISOP worker. CC=8.
func (ctx *BDDCtx) isopRec(f gobdd.NodeID) Formula {
	if f == gobdd.False {
		return nil
	}
	if f == gobdd.True {
		return trueFormula
	}
	ctx.ensureISOPCache(int(f) + 1)
	if ctx.isopCached[f] {
		return ctx.isopCache[f]
	}

	x := ctx.bdd.VarOf(f)
	f0 := ctx.bdd.Restrict(f, x, false)
	f1 := ctx.bdd.Restrict(f, x, true)

	// g0 = ISOP(f0 \\ f1): cubes unique to cofactor x=0.
	diff0 := ctx.bdd.Diff(f0, f1)
	g0 := ctx.ISOP(diff0)

	// g1 = ISOP(f1 \\ f0): cubes unique to cofactor x=1.
	diff1 := ctx.bdd.Diff(f1, f0)
	g1 := ctx.ISOP(diff1)

	// h = ISOP(shared \\cap not(union)) where shared=f0\\cap f1, union is BDD of g0\\cup g1.
	// The unique cubes (g0, g1) are removed from shared before recursion.
	shared := ctx.bdd.And(f0, f1)
	union := ctx.bdd.Or(diff0, diff1)
	h := ctx.ISOP(ctx.bdd.And(shared, ctx.bdd.Not(union)))

	v := Atom{ID: fuzzy.VarID(x)}
	result := h
	if g0 != nil {
		result = orISOP(result, isopCube(Not{Formula: v}, g0))
	}
	if g1 != nil {
		result = orISOP(result, isopCube(v, g1))
	}
	ctx.isopCache[f] = result
	ctx.isopCached[f] = true
	return result
}

// ensureISOPCache grows the memo table to at least the given size. CC=2.
func (ctx *BDDCtx) ensureISOPCache(size int) {
	if size <= len(ctx.isopCache) {
		return
	}
	n := size * 2
	newF := memory.MustPoolSlice[Formula](ctx.pool, n)
	newF = newF[:n]
	copy(newF, ctx.isopCache)
	newB := memory.MustPoolSlice[bool](ctx.pool, n)
	newB = newB[:n]
	copy(newB, ctx.isopCached)
	ctx.isopCache = newF
	ctx.isopCached = newB
}

// isopCube builds a cube: (literal ∧ g), collapsing (literal ∧ True) → literal. CC=2.
func isopCube(lit, g Formula) Formula {
	if g == trueFormula {
		return lit
	}
	return And{Left: lit, Right: g}
}

// orISOP builds Or{left, right}, stripping nil (False). CC=2.
func orISOP(left, right Formula) Formula {
	if left == nil {
		return right
	}
	if right == nil {
		return left
	}
	return Or{Left: left, Right: right}
}

// trueFormula is the singleton sentinel for logical True in ISOP output.
var trueFormula = &trueSentinel{}

type trueSentinel struct{}

func (*trueSentinel) Evaluate(World, *Model) (TruthValue, error) { return 1.0, nil }
