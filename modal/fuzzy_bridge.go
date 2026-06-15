package modal

// FuzzyConfig selects the t-norm, t-conorm, and implication for fuzzy modal evaluation.
// Defaults give standard crisp modal logic (Gödel min/max).
type FuzzyConfig struct {
	TNorm       func(a, b TruthValue) TruthValue // AND — default: Gödel min
	TConorm     func(a, b TruthValue) TruthValue // OR — default: Gödel max
	Implication func(a, b TruthValue) TruthValue // → for weighted □ — default: Łukasiewicz
}

// DefaultFuzzyConfig returns the default crisp modal config (Gödel min/max).
func DefaultFuzzyConfig() FuzzyConfig {
	return FuzzyConfig{
		TNorm:       godelMin,
		TConorm:     godelMax,
		Implication: lukasiewicz,
	}
}

// WeightedFuzzyConfig returns a config using Product t-norm, Probabilistic sum t-conorm,
// and Łukasiewicz implication — suitable for daemon hop expansion weights.
func WeightedFuzzyConfig() FuzzyConfig {
	return FuzzyConfig{
		TNorm:       productTNorm,
		TConorm:     probabilisticSum,
		Implication: lukasiewicz,
	}
}

// BoxFuzzy evaluates □p at world w using fuzzy operators over weighted accessibility.
// For each accessible world v with weight w_v: result = min(result, imp(w_v, p(v))).
// With default config (Łukasiewicz implication): result = min_v(min(1, 1 - w_v + p(v))).
// If no worlds are accessible, returns 1.0 (vacuously true).
// CC=5.
func BoxFuzzy(f Formula, w World, m *Model, cfg FuzzyConfig, rel RelationType) (TruthValue, error) {
	targets := m.frame.WeightedAccessible(w, rel)
	if len(targets) == 0 {
		return 1.0, nil
	}
	result := TruthValue(1.0)
	for _, t := range targets {
		tv, err := f.Evaluate(t.Dst, m)
		if err != nil {
			return 0, err
		}
		// imp(weight, truth) — weight modulates how strongly the successor matters
		imp := cfg.Implication(t.Weight, tv)
		result = cfg.TNorm(result, imp)
	}
	return result, nil
}

// DiamondFuzzy evaluates ◇p at world w using fuzzy operators over weighted accessibility.
// For each accessible world v with weight w_v: result = max(result, tconorm(w_v, p(v))).
// With default config (product t-norm): result = max_v(w_v * p(v)).
// If no worlds are accessible, returns 0.0 (vacuously false).
// CC=5.
func DiamondFuzzy(f Formula, w World, m *Model, cfg FuzzyConfig, rel RelationType) (TruthValue, error) {
	targets := m.frame.WeightedAccessible(w, rel)
	if len(targets) == 0 {
		return 0.0, nil
	}
	result := TruthValue(0.0)
	for _, t := range targets {
		tv, err := f.Evaluate(t.Dst, m)
		if err != nil {
			return 0, err
		}
		// tconorm(weight, truth) — weight amplifies the successor's contribution
		comb := cfg.TNorm(t.Weight, tv)
		result = cfg.TConorm(result, comb)
	}
	return result, nil
}

// WeightedFrameAccessibility returns the accessibility weight between two worlds.
// Returns 1.0 for unweighted edges, the stored weight for weighted edges, 0.0 if no edge.
// CC=3.
func WeightedFrameAccessibility(src, dst World, rel RelationType, frame *Frame) TruthValue {
	for i := range frame.weightedEdges {
		e := frame.weightedEdges[i]
		if e.Src == src && e.Dst == dst && e.Rel == rel {
			return e.Weight
		}
	}
	if frame.IsAccessible(src, dst, rel) {
		return 1.0
	}
	return 0.0
}

// FuzzyEntailment checks whether premises entail conclusion under fuzzy semantics.
// Returns the degree to which the entailment holds (1.0 = crisp entailment).
// Requires a model (with the frame embedded) to evaluate against.
// CC=5.
func FuzzyEntailment(premises []Formula, conclusion Formula, m *Model, cfg FuzzyConfig, rel RelationType) (TruthValue, error) {
	if len(premises) == 0 {
		return 0.0, nil
	}
	// Build conjunction of premises
	var conj Formula = premises[0]
	for i := 1; i < len(premises); i++ {
		conj = And{Left: conj, Right: premises[i]}
	}

	wc := m.Frame().WorldCount()
	minDegree := TruthValue(1.0)
	anyChecked := false
	for w := World(0); w < World(wc); w++ {
		if len(m.Frame().WeightedAccessible(w, rel)) == 0 {
			continue // vacuous — skip worlds with no accessible successors
		}
		premTV, err := BoxFuzzy(conj, w, m, cfg, rel)
		if err != nil {
			return 0, err
		}
		if premTV == 0 {
			continue
		}
		anyChecked = true
		concTV, err := DiamondFuzzy(conclusion, w, m, cfg, rel)
		if err != nil {
			return 0, err
		}
		deg := cfg.Implication(premTV, concTV)
		if deg < minDegree {
			minDegree = deg
		}
	}
	if !anyChecked {
		return 0.0, nil
	}
	return minDegree, nil
}

// --- Built-in fuzzy operators (no imports needed) ---

// goadelMin returns min(a, b).
func godelMin(a, b TruthValue) TruthValue {
	if a < b {
		return a
	}
	return b
}

// godelMax returns max(a, b).
func godelMax(a, b TruthValue) TruthValue {
	if a > b {
		return a
	}
	return b
}

// productTNorm returns a * b.
func productTNorm(a, b TruthValue) TruthValue { return a * b }

// probabilisticSum returns a + b - a*b.
func probabilisticSum(a, b TruthValue) TruthValue { return a + b - a*b }

// lukasiewicz returns min(1, 1 - a + b).
func lukasiewicz(a, b TruthValue) TruthValue {
	r := 1.0 - float64(a) + float64(b)
	if r > 1.0 {
		return 1.0
	}
	return TruthValue(r)
}
