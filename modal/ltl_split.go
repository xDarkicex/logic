package modal

import "github.com/xDarkicex/memory"

// FormulaClass classifies a modal formula for decomposition strategy.
type FormulaClass uint8

const (
	ClassObligation FormulaClass = iota // safety: finite-prefix checkable
	ClassSuspendable                    // □◇p type: eventual + universal
	ClassRest                           // general: needs full tableau
)

// LTLClassifier computes formula properties for obligation/suspendable/rest splitting.
// Uses a single bottom-up traversal with Pool-backed result storage. CC ≤ 8 per method.
type LTLClassifier struct {
	pool *memory.Pool
}

// NewLTLClassifier creates a classifier.
func NewLTLClassifier(pool *memory.Pool) *LTLClassifier {
	return &LTLClassifier{pool: pool}
}

// Classify returns the formula class for f.
func (c *LTLClassifier) Classify(f Formula) FormulaClass {
	obl, ev, un := c.classifyRec(f)
	if obl {
		return ClassObligation
	}
	if ev && un {
		return ClassSuspendable
	}
	return ClassRest
}

// IsObligation returns true if f is a safety formula. CC=3.
func (c *LTLClassifier) IsObligation(f Formula) bool {
	obl, _, _ := c.classifyRec(f)
	return obl
}

// IsSuspendable returns true if f is both eventual and universal. CC=3.
func (c *LTLClassifier) IsSuspendable(f Formula) bool {
	_, ev, un := c.classifyRec(f)
	return ev && un
}

// classifyRec computes (isObligation, isEventual, isUniversal) in one pass. CC=8.
func (c *LTLClassifier) classifyRec(f Formula) (obl, ev, un bool) {
	switch t := f.(type) {
	case Atom:
		return true, false, false

	case Not:
		return c.classifyRec(t.Formula)

	case And:
		lobl, lev, lun := c.classifyRec(t.Left)
		robl, rev, run := c.classifyRec(t.Right)
		return lobl && robl, lev || rev, lun && run

	case Or:
		lobl, lev, lun := c.classifyRec(t.Left)
		robl, rev, run := c.classifyRec(t.Right)
		return lobl || robl, lev && rev, lun || run

	case Implies:
		_, _, aun := c.classifyRec(t.Antecedent)
		_, cev, _ := c.classifyRec(t.Consequent)
		return false, aun || cev, false

	case Iff:
		lobl, lev, lun := c.classifyRec(t.Left)
		robl, rev, run := c.classifyRec(t.Right)
		return lobl && robl, lev && rev, lun && run

	case Box:
		inner, iev, _ := c.classifyRec(t.Formula)
		// □f: universal. Obligation if inner is obligation.
		return inner, iev, true

	case Diamond:
		_, _, iun := c.classifyRec(t.Formula)
		// ◇f: always eventual, never obligation at top level.
		// Suspendable when inner is universal (◇□p type).
		return false, true, iun

	case Next:
		inner, iev, iun := c.classifyRec(t.Formula)
		return inner, iev, iun

	case Until:
		_, lev, _ := c.classifyRec(t.Left)
		_, rev, _ := c.classifyRec(t.Right)
		// a U b: not obligation, eventual if either side is eventual.
		return false, lev || rev, false

	default:
		return false, false, false
	}
}

// --- Splitter ---

// LTLGroup holds the three formula groups from splitting.
type LTLGroup struct {
	Obligation  []Formula // safety formulas
	Suspendable []Formula // □◇p type
	Rest        []Formula // general
}

// SplitConjunction decomposes an And-chain into obligation/suspendable/rest groups.
// Obligation conjuncts can be checked directly; rest needs full tableau.
// Uses Pool-backed slices. CC=7.
func (c *LTLClassifier) SplitConjunction(f Formula) LTLGroup {
	conjuncts := c.flattenAnd(f)
	var g LTLGroup
	for _, cj := range conjuncts {
		switch c.Classify(cj) {
		case ClassObligation:
			g.Obligation = append(g.Obligation, cj)
		case ClassSuspendable:
			g.Suspendable = append(g.Suspendable, cj)
		default:
			g.Rest = append(g.Rest, cj)
		}
	}
	return g
}

// SplitDisjunction decomposes an Or-chain into groups. CC=7.
func (c *LTLClassifier) SplitDisjunction(f Formula) LTLGroup {
	disjuncts := c.flattenOr(f)
	var g LTLGroup
	for _, dj := range disjuncts {
		switch c.Classify(dj) {
		case ClassObligation:
			g.Obligation = append(g.Obligation, dj)
		case ClassSuspendable:
			g.Suspendable = append(g.Suspendable, dj)
		default:
			g.Rest = append(g.Rest, dj)
		}
	}
	return g
}

// flattenAnd collects top-level And conjuncts into a Pool-backed slice. CC=3.
func (c *LTLClassifier) flattenAnd(f Formula) []Formula {
	result := memory.MustPoolSlice[Formula](c.pool, 16)
	result = result[:0]
	var walk func(Formula)
	walk = func(x Formula) {
		if a, ok := x.(And); ok {
			walk(a.Left)
			walk(a.Right)
		} else {
			result = append(result, x)
		}
	}
	walk(f)
	return result
}

// flattenOr collects top-level Or disjuncts. CC=3.
func (c *LTLClassifier) flattenOr(f Formula) []Formula {
	result := memory.MustPoolSlice[Formula](c.pool, 16)
	result = result[:0]
	var walk func(Formula)
	walk = func(x Formula) {
		if o, ok := x.(Or); ok {
			walk(o.Left)
			walk(o.Right)
		} else {
			result = append(result, x)
		}
	}
	walk(f)
	return result
}

// HasOnlyObligation returns true if every component is obligation. CC=2.
func (g LTLGroup) HasOnlyObligation() bool {
	return len(g.Suspendable) == 0 && len(g.Rest) == 0
}

// HasAnyRest returns true if the group contains general formulas. CC=2.
func (g LTLGroup) HasAnyRest() bool {
	return len(g.Rest) > 0
}

// IsEmpty returns true if all groups are empty. CC=2.
func (g LTLGroup) IsEmpty() bool {
	return len(g.Obligation) == 0 && len(g.Suspendable) == 0 && len(g.Rest) == 0
}
