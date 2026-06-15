// Package modal implements Kripke semantics, tableau-based satisfiability,
// temporal/epistemic/deontic logic, and a fuzzy-modal bridge for agentic memory.
// All allocations use github.com/xDarkicex/memory (Pool, Arena, ShardedFreeList).
package modal

import "github.com/xDarkicex/logic/fuzzy"

// World is a handle to a possible world in a Kripke frame.
// Worlds are uint32 indices into the Frame's Arena-backed world list.
type World uint32

// RelationType identifies a kind of accessibility relation between worlds.
// Maps to daemon edge types: why_ids (causal), how_ids (procedural), hop_targets (association).
type RelationType uint8

// Standard relation types. User-defined types start at 16.
const (
	RelCausal     RelationType = iota // why_ids — upstream causal edges
	RelProcedural                     // how_ids — downstream procedural edges
	RelAssociation                    // hop_targets — undirected association edges
)

// TruthValue is a fuzzy truth value in [0.0, 1.0].
// Re-exported from the fuzzy package for the fuzzy-modal bridge (Phase 6).
type TruthValue = fuzzy.TruthValue

// Formula is a modal logic formula evaluable at a world in a Kripke model.
// Each concrete formula type implements this interface.
type Formula interface {
	// Evaluate returns the truth value of this formula at world w in model m.
	Evaluate(w World, m *Model) (TruthValue, error)
}

// Atom is an atomic proposition identified by a VarID.
// Atoms evaluate to the truth value assigned by the model's valuation at the given world.
type Atom struct {
	ID fuzzy.VarID
}

// Evaluate looks up the truth value of this atom at world w.
// Returns 0.0 for out-of-bounds world or variable index (no panic).
func (a Atom) Evaluate(w World, m *Model) (TruthValue, error) {
	if int(w) >= len(m.valuation) {
		return 0.0, nil
	}
	row := m.valuation[w]
	if int(a.ID) >= len(row) {
		return 0.0, nil
	}
	return row[a.ID], nil
}

// Box represents the necessity operator □ (must hold in all accessible worlds).
// The relation type selects which accessibility relation to use.
type Box struct {
	Formula Formula
	Rel     RelationType
}

// Evaluate returns the minimum truth value of f across all worlds accessible from w via rel.
// Returns 1.0 if no worlds are accessible (vacuously true).
func (b Box) Evaluate(w World, m *Model) (TruthValue, error) {
	return evalQuantified(b.Formula, w, m, b.Rel, true)
}

// Diamond represents the possibility operator ◇ (holds in at least one accessible world).
type Diamond struct {
	Formula Formula
	Rel     RelationType
}

// Evaluate returns the maximum truth value of f across all worlds accessible from w via rel.
// Returns 0.0 if no worlds are accessible (vacuously false).
func (d Diamond) Evaluate(w World, m *Model) (TruthValue, error) {
	return evalQuantified(d.Formula, w, m, d.Rel, false)
}

// evalQuantified evaluates a formula across all worlds accessible from w via rel.
// If isBox is true, returns the minimum (□). If false, returns the maximum (◇).
func evalQuantified(f Formula, w World, m *Model, rel RelationType, isBox bool) (TruthValue, error) {
	targets := m.frame.Accessible(w, rel)
	if len(targets) == 0 {
		if isBox {
			return 1.0, nil
		}
		return 0.0, nil
	}
	result := TruthValue(0.0)
	if isBox {
		result = 1.0
	}
	for _, v := range targets {
		tv, err := f.Evaluate(v, m)
		if err != nil {
			return 0, err
		}
		if isBox {
			if tv < result {
				result = tv
			}
		} else {
			if tv > result {
				result = tv
			}
		}
	}
	return result, nil
}

// Not represents the negation operator ¬.
type Not struct {
	Formula Formula
}

// Evaluate returns 1.0 - the truth value of the inner formula.
func (n Not) Evaluate(w World, m *Model) (TruthValue, error) {
	tv, err := n.Formula.Evaluate(w, m)
	if err != nil {
		return 0, err
	}
	return 1.0 - tv, nil
}

// And represents the conjunction operator ∧.
type And struct {
	Left, Right Formula
}

// Evaluate returns the minimum truth value of the two conjuncts (Gödel t-norm).
func (a And) Evaluate(w World, m *Model) (TruthValue, error) {
	l, err := a.Left.Evaluate(w, m)
	if err != nil {
		return 0, err
	}
	r, err := a.Right.Evaluate(w, m)
	if err != nil {
		return 0, err
	}
	if l < r {
		return l, nil
	}
	return r, nil
}

// Or represents the disjunction operator ∨.
type Or struct {
	Left, Right Formula
}

// Evaluate returns the maximum truth value of the two disjuncts (Gödel t-conorm).
func (o Or) Evaluate(w World, m *Model) (TruthValue, error) {
	l, err := o.Left.Evaluate(w, m)
	if err != nil {
		return 0, err
	}
	r, err := o.Right.Evaluate(w, m)
	if err != nil {
		return 0, err
	}
	if l > r {
		return l, nil
	}
	return r, nil
}

// Implies represents the implication operator →.
type Implies struct {
	Antecedent, Consequent Formula
}

// Evaluate returns the fuzzy implication (Gödel) of antecedent → consequent.
func (i Implies) Evaluate(w World, m *Model) (TruthValue, error) {
	a, err := i.Antecedent.Evaluate(w, m)
	if err != nil {
		return 0, err
	}
	c, err := i.Consequent.Evaluate(w, m)
	if err != nil {
		return 0, err
	}
	if a <= c {
		return 1.0, nil
	}
	return c, nil
}

// Iff represents the biconditional operator ↔.
type Iff struct {
	Left, Right Formula
}

// Evaluate returns the truth value of left ↔ right (Gödel biconditional).
func (i Iff) Evaluate(w World, m *Model) (TruthValue, error) {
	impl1 := Implies{Antecedent: i.Left, Consequent: i.Right}
	impl2 := Implies{Antecedent: i.Right, Consequent: i.Left}
	return And{Left: impl1, Right: impl2}.Evaluate(w, m)
}
