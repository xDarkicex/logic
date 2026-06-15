package modal

import "fmt"

// System identifies a modal axiom system.
// Each system imposes constraints on the accessibility relation R.
type System int

const (
	SystemK  System = iota // no constraints (all frames)
	SystemD                // serial: every world has ≥1 successor
	SystemT                // reflexive: every world accesses itself
	SystemB                // reflexive + symmetric
	SystemS4               // reflexive + transitive
	SystemS5               // equivalence relation (reflexive + symmetric + transitive)
)

// String returns the system name.
func (s System) String() string {
	switch s {
	case SystemK:
		return "K"
	case SystemD:
		return "D"
	case SystemT:
		return "T"
	case SystemB:
		return "B"
	case SystemS4:
		return "S4"
	case SystemS5:
		return "S5"
	}
	return "unknown"
}

// EnforceSystemK is a no-op — K holds in all frames.
// CC=1.
func EnforceSystemK(frame *Frame) {}

// EnforceSystemD adds seriality: every world must have at least one successor.
// Worlds without outgoing edges get a self-loop for the given relation type.
// CC=3.
func EnforceSystemD(frame *Frame, rel RelationType) {
	wc := frame.WorldCount()
	for w := World(0); w < World(wc); w++ {
		if len(frame.Accessible(w, rel)) == 0 {
			frame.AddRelation(w, w, rel)
		}
	}
}

// EnforceSystemT adds reflexivity: every world accesses itself.
// CC=1.
func EnforceSystemT(frame *Frame, rel RelationType) {
	frame.ReflexiveClosure(rel)
}

// EnforceSystemB adds reflexivity and symmetry.
// CC=2.
func EnforceSystemB(frame *Frame, rel RelationType) {
	frame.ReflexiveClosure(rel)
	frame.SymmetricClosure(rel)
}

// EnforceSystemS4 adds reflexivity and transitivity (S4 = T + transitivity).
// CC=2.
func EnforceSystemS4(frame *Frame, rel RelationType) {
	frame.ReflexiveClosure(rel)
	frame.TransitiveClosure(rel)
}

// EnforceSystemS5 builds an equivalence relation (reflexive + symmetric + transitive).
// CC=2.
func EnforceSystemS5(frame *Frame, rel RelationType) {
	frame.ReflexiveClosure(rel)
	frame.SymmetricClosure(rel)
	frame.TransitiveClosure(rel)
}

// ValidateFrameAgainst checks whether a frame satisfies the given axiom system.
// Returns nil if valid, an error describing the violation otherwise.
// CC=4.
func ValidateFrameAgainst(frame *Frame, system System, rel RelationType) error {
	wc := frame.WorldCount()

	switch system {
	case SystemK:
		return nil // all frames satisfy K

	case SystemD:
		for w := World(0); w < World(wc); w++ {
			if len(frame.Accessible(w, rel)) == 0 {
				return fmt.Errorf("System D violation: world %d has no successors for relation %d", w, rel)
			}
		}

	case SystemT:
		for w := World(0); w < World(wc); w++ {
			if !frame.IsAccessible(w, w, rel) {
				return fmt.Errorf("System T violation: world %d does not access itself for relation %d", w, rel)
			}
		}

	case SystemB:
		if err := validateReflexive(frame, rel); err != nil {
			return err
		}
		if err := validateSymmetric(frame, rel); err != nil {
			return err
		}

	case SystemS4:
		if err := validateReflexive(frame, rel); err != nil {
			return err
		}
		if err := validateTransitive(frame, rel); err != nil {
			return err
		}

	case SystemS5:
		if err := validateReflexive(frame, rel); err != nil {
			return err
		}
		if err := validateSymmetric(frame, rel); err != nil {
			return err
		}
		if err := validateTransitive(frame, rel); err != nil {
			return err
		}
	}
	return nil
}

// validateReflexive checks that every world accesses itself.
func validateReflexive(frame *Frame, rel RelationType) error {
	wc := frame.WorldCount()
	for w := World(0); w < World(wc); w++ {
		if !frame.IsAccessible(w, w, rel) {
			return fmt.Errorf("reflexivity violation: world %d, relation %d", w, rel)
		}
	}
	return nil
}

// validateSymmetric checks that if w→v then v→w.
func validateSymmetric(frame *Frame, rel RelationType) error {
	wc := frame.WorldCount()
	for w := World(0); w < World(wc); w++ {
		targets := frame.Accessible(w, rel)
		for _, v := range targets {
			if !frame.IsAccessible(v, w, rel) {
				return fmt.Errorf("symmetry violation: %d→%d exists but %d→%d missing, relation %d",
					w, v, v, w, rel)
			}
		}
	}
	return nil
}

// validateTransitive checks that if w→v and v→u then w→u.
func validateTransitive(frame *Frame, rel RelationType) error {
	wc := frame.WorldCount()
	for w := World(0); w < World(wc); w++ {
		for _, v := range frame.Accessible(w, rel) {
			for _, u := range frame.Accessible(v, rel) {
				if !frame.IsAccessible(w, u, rel) {
					return fmt.Errorf("transitivity violation: %d→%d and %d→%d but %d→%d missing, relation %d",
						w, v, v, u, w, u, rel)
				}
			}
		}
	}
	return nil
}
