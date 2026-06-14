package fuzzy

// PopulationEncoder generates a population of `n` uniformly distributed triangular
// membership functions across the [min, max] domain. This enables population encoding
// of continuous values into fuzzy sets, useful for bridging sensory inputs to fuzzy rules.
func PopulationEncoder(n int, min, max float64) []MembershipFunc {
	if n <= 0 {
		return nil
	}
	if n == 1 {
		// Single element covers everything identically
		mid := min + (max-min)/2.0
		return []MembershipFunc{Triangular(min, mid, max)}
	}

	funcs := make([]MembershipFunc, n)
	step := (max - min) / float64(n-1)

	for i := 0; i < n; i++ {
		peak := min + float64(i)*step
		left := peak - step
		right := peak + step
		funcs[i] = Triangular(left, peak, right)
	}

	return funcs
}
