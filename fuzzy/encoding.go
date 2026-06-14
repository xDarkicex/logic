package fuzzy

import "github.com/xDarkicex/memory"

// PopulationEncoder generates a population of `n` uniformly distributed triangular
// membership functions across the [min, max] domain. The returned slice is Pool-backed.
func PopulationEncoder(n int, min, max float64, pool *memory.Pool) []MembershipFunc {
	if n <= 0 {
		return nil
	}
	if n == 1 {
		mid := min + (max-min)/2.0
		funcs := memory.MustPoolSlice[MembershipFunc](pool, 1)
		funcs = funcs[:1]
		funcs[0] = Triangular(min, mid, max)
		return funcs
	}

	funcs := memory.MustPoolSlice[MembershipFunc](pool, n)
	funcs = funcs[:n]
	step := (max - min) / float64(n-1)

	for i := 0; i < n; i++ {
		peak := min + float64(i)*step
		left := peak - step
		right := peak + step
		funcs[i] = Triangular(left, peak, right)
	}

	return funcs
}
