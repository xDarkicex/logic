package modal

import (
	"github.com/xDarkicex/gobdd"
	"github.com/xDarkicex/memory"
)

// Decomp holds the NOW / NEXT / PROMISE components of a decomposed formula.
type Decomp struct {
	Now     []Formula // Boolean conditions at current state
	Next    []Formula // ○-formulas for successors
	Promise []Formula // acceptance promises (U / ◇ formulas)
}

// Decomposer splits temporal formulas into NOW / NEXT / PROMISE components
// using BDDs for the Boolean skeleton. Matches Spot's Couvreur FM expansion:
// f ≡ NOW(f) ∧ ○NEXT(f) ∧ PROMISE(f).
type Decomposer struct {
	ctx  *BDDCtx
	pool *memory.Pool
}

// NewDecomposer creates a formula decomposer sharing the BDD context.
func NewDecomposer(ctx *BDDCtx, pool *memory.Pool) *Decomposer {
	return &Decomposer{ctx: ctx, pool: pool}
}

// Decompose splits f into NOW / NEXT / PROMISE components. CC=9.
func (d *Decomposer) Decompose(f Formula) Decomp {
	var dc Decomp
	d.decomposeRec(f, &dc)
	return dc
}

// decomposeRec recursively collects components. CC=6.
func (d *Decomposer) decomposeRec(f Formula, dc *Decomp) {
	switch t := f.(type) {
	case Atom:
		dc.Now = append(dc.Now, t)

	case Not:
		if _, ok := t.Formula.(Until); ok {
			dc.Promise = append(dc.Promise, f)
			return
		}
		if inner, ok := t.Formula.(Box); ok {
			dc.Promise = append(dc.Promise,
				Diamond{Formula: Not{Formula: inner.Formula}, Rel: inner.Rel})
			return
		}
		d.decomposeRec(t.Formula, dc)

	case And:
		d.decomposeRec(t.Left, dc)
		d.decomposeRec(t.Right, dc)

	case Or:
		d.decomposeRec(t.Left, dc)
		d.decomposeRec(t.Right, dc)

	case Implies:
		d.decomposeRec(Or{Left: Not{Formula: t.Antecedent}, Right: t.Consequent}, dc)

	case Iff:
		d.decomposeRec(And{
			Left:  Implies{Antecedent: t.Left, Consequent: t.Right},
			Right: Implies{Antecedent: t.Right, Consequent: t.Left},
		}, dc)

	case Box:
		if isPureBoolean(t.Formula) {
			dc.Now = append(dc.Now, t)
		} else {
			dc.Next = append(dc.Next, t)
		}

	case Diamond:
		if _, ok := t.Formula.(Box); ok {
			dc.Promise = append(dc.Promise, t)
		} else if isPureBoolean(t.Formula) {
			dc.Now = append(dc.Now, t)
		} else {
			dc.Promise = append(dc.Promise, t)
		}

	case Next:
		dc.Next = append(dc.Next, t)

	case Until:
		dc.Promise = append(dc.Promise, t)
	}
}

// isPureBoolean returns true if f contains no modal operators. CC=5.
func isPureBoolean(f Formula) bool {
	switch f.(type) {
	case Atom:
		return true
	case Not, And, Or, Implies, Iff:
		return isPureBooleanRec(f)
	default:
		return false
	}
}

// isPureBooleanRec recursively checks for modal operators. CC=5.
func isPureBooleanRec(f Formula) bool {
	switch t := f.(type) {
	case Atom:
		return true
	case Not:
		return isPureBooleanRec(t.Formula)
	case And:
		return isPureBooleanRec(t.Left) && isPureBooleanRec(t.Right)
	case Or:
		return isPureBooleanRec(t.Left) && isPureBooleanRec(t.Right)
	case Implies:
		return isPureBooleanRec(t.Antecedent) && isPureBooleanRec(t.Consequent)
	case Iff:
		return isPureBooleanRec(t.Left) && isPureBooleanRec(t.Right)
	default:
		return false
	}
}

// BuildNowBDD constructs the BDD for the NOW component. CC=2.
func (d *Decomposer) BuildNowBDD(now []Formula) gobdd.NodeID {
	if len(now) == 0 {
		return gobdd.True
	}
	result := d.ctx.bdd.And(d.ctx.ToBDD(now[0]), gobdd.True)
	for i := 1; i < len(now); i++ {
		result = d.ctx.bdd.And(result, d.ctx.ToBDD(now[i]))
	}
	return result
}

// BuildNextBDD constructs a BDD variable set for NEXT formulas. CC=5.
func (d *Decomposer) BuildNextBDD(next []Formula) (gobdd.NodeID, []Formula) {
	if len(next) == 0 {
		return gobdd.True, nil
	}
	conj := next[0]
	for i := 1; i < len(next); i++ {
		conj = And{Left: conj, Right: next[i]}
	}
	root := d.ctx.Skeleton(conj)
	return root, d.ctx.SkeletonSubs()
}

// BuildPromiseBDD constructs a BDD variable set for PROMISE formulas. CC=5.
func (d *Decomposer) BuildPromiseBDD(promise []Formula) (gobdd.NodeID, []Formula) {
	if len(promise) == 0 {
		return gobdd.True, nil
	}
	conj := promise[0]
	for i := 1; i < len(promise); i++ {
		conj = And{Left: conj, Right: promise[i]}
	}
	root := d.ctx.Skeleton(conj)
	return root, d.ctx.SkeletonSubs()
}

// HasPromise returns true if the decomposition contains acceptance promises. CC=2.
func (dc Decomp) HasPromise() bool { return len(dc.Promise) > 0 }

// HasNext returns true if the decomposition contains X-formulas. CC=2.
func (dc Decomp) HasNext() bool { return len(dc.Next) > 0 }

// IsTrivial returns true if only NOW constraints exist. CC=2.
func (dc Decomp) IsTrivial() bool { return !dc.HasNext() && !dc.HasPromise() }
