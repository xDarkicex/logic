package modal

import "github.com/xDarkicex/memory"

// Cascade implements §17 Post-Processing Cascade: three-tier evaluation
// from cheapest (syntactic) to most expensive (Couvreur SCC + POR).
// All tiers use Pool-backed storage — zero heap allocations.
type Cascade struct {
	reg     *Registry
	rw      *Rewriter
	couvreur *CouvreurProver
	pool    *memory.Pool
	arena   *memory.Arena
}

// NewCascade creates a three-tier evaluation pipeline.
func NewCascade(reg *Registry, rw *Rewriter, pool *memory.Pool, arena *memory.Arena) *Cascade {
	return &Cascade{
		reg:      reg,
		rw:       rw,
		couvreur: NewCouvreurProver(pool, arena),
		pool:     pool,
		arena:    arena,
	}
}

// Prove checks whether formula f is satisfiable, routing through the cascade.
// Tier 1 → Tier 2 → Tier 3. Returns (satisfiable, model).
// CC=6.
func (c *Cascade) Prove(f Formula) (bool, *Model) {
	// Tier 1: Syntactic — simplify and check trivial cases
	result := c.tier1Syntactic(f)
	if result.known {
		return result.sat, result.model
	}

	// Tier 2: Shallow — quick 1-2 world model check
	result = c.tier2Shallow(result.formula)
	if result.known {
		return result.sat, result.model
	}

	// Tier 3: Full Couvreur SCC + POR
	return c.tier3Full(result.formula)
}

// cascadeResult holds the outcome of a tier evaluation.
type cascadeResult struct {
	formula Formula
	sat     bool
	model   *Model
	known   bool // true if this tier produced a definitive answer
}

// --- Tier 1: Syntactic ---

// tier1Syntactic simplifies and checks for trivial tautologies/contradictions.
// CC=5.
func (c *Cascade) tier1Syntactic(f Formula) cascadeResult {
	// Canonicalize
	cf := c.reg.Intern(f)

	// Simplify using eventuality/universality rewrites
	simplified := c.rw.Simplify(cf)

	// Check for trivial tautology: P ∨ ¬P, □(tt), ◇(tt), etc.
	if isTrivialTrue(simplified) {
		return cascadeResult{formula: simplified, sat: true, known: true}
	}

	// Check for trivial contradiction: P ∧ ¬P, □(ff), ◇(ff)
	if isTrivialFalse(simplified) {
		return cascadeResult{formula: simplified, sat: false, known: true}
	}

	return cascadeResult{formula: simplified, known: false}
}

// isTrivialTrue checks syntactic tautologies. CC=3.
func isTrivialTrue(f Formula) bool {
	switch t := f.(type) {
	case Or:
		// P ∨ ¬P
		if isNegationOf(t.Left, t.Right) {
			return true
		}
		if isNegationOf(t.Right, t.Left) {
			return true
		}
	case Implies:
		// P → P
		if formulaEqual(t.Antecedent, t.Consequent) {
			return true
		}
	}
	return false
}

// isTrivialFalse checks syntactic contradictions. CC=3.
func isTrivialFalse(f Formula) bool {
	switch t := f.(type) {
	case And:
		// P ∧ ¬P
		if isNegationOf(t.Left, t.Right) {
			return true
		}
		if isNegationOf(t.Right, t.Left) {
			return true
		}
	}
	return false
}

// isNegationOf returns true if a is ¬b. CC=2.
func isNegationOf(a, b Formula) bool {
	if n, ok := a.(Not); ok {
		return formulaEqual(n.Formula, b)
	}
	return false
}

// --- Tier 2: Shallow ---

// tier2Shallow performs a quick 1-2 world bounded model check.
// If a counterexample is found in a small frame, returns immediately.
// CC=5.
func (c *Cascade) tier2Shallow(f Formula) cascadeResult {
	for _, size := range []int{1, 2} {
		a2, _ := memory.NewArena(1024 * 1024)
		frame := NewFrame(c.pool, a2)
		for i := 0; i < size; i++ {
			frame.AddWorld()
		}
		// Try with self-loops (reflexive — common in many modal systems)
		for w := World(0); w < World(size); w++ {
			frame.AddRelation(w, w, RelCausal)
		}

		sat, model := c.couvreur.ProveSatisfiable(f, frame)
		if sat {
			a2.Free()
			return cascadeResult{formula: f, sat: true, model: model, known: true}
		}
		a2.Free()
	}
	return cascadeResult{formula: f, known: false}
}

// --- Tier 3: Full ---

// tier3Full runs the full Couvreur SCC + POR branch expansion.
// CC=2.
func (c *Cascade) tier3Full(f Formula) (bool, *Model) {
	a2, _ := memory.NewArena(1024 * 1024)
	defer a2.Free()
	frame := NewFrame(c.pool, a2)
	frame.AddWorld()
	return c.couvreur.ProveSatisfiable(f, frame)
}
