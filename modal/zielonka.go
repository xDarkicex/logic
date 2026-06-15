package modal

import "github.com/xDarkicex/memory"

// Player identifies which player controls a state in a parity game.
// Player 0 (Even) wants the minimum infinitely-often priority to be even.
// Player 1 (Odd) wants it to be odd.
type Player uint8

const (
	PlayerEven Player = 0
	PlayerOdd  Player = 1
)

// ParityGame is a deterministic parity game. States are numbered 0..n-1.
// Each state has an owner (Player), a priority (0..maxPri), and a list of
// successor states. All backing slices are Pool-allocated.
type ParityGame struct {
	owner    []Player  // owner[s] = player who controls state s
	priority []int32   // priority[s] = priority (color) of state s
	succ     [][]int32 // succ[s] = successors (Pool-backed per state)
	pred     [][]int32 // pred[s] = predecessors (reverse edges, lazily built)
	pool     *memory.Pool
}

// NewParityGame creates a parity game with n states.
func NewParityGame(n int, pool *memory.Pool) *ParityGame {
	owner := memory.MustPoolSlice[Player](pool, n)
	owner = owner[:n]
	priority := memory.MustPoolSlice[int32](pool, n)
	priority = priority[:n]
	succ := memory.MustPoolSlice[[]int32](pool, n)
	succ = succ[:n]
	for i := range succ {
		succ[i] = memory.MustPoolSlice[int32](pool, 0)
	}
	return &ParityGame{owner: owner, priority: priority, succ: succ, pool: pool}
}

// SetState configures a state's owner, priority, and successors.
func (pg *ParityGame) SetState(s int32, p Player, pri int32, successors []int32) {
	pg.owner[s] = p
	pg.priority[s] = pri
	n := memory.MustPoolSlice[int32](pg.pool, len(successors))
	n = n[:len(successors)]
	copy(n, successors)
	pg.succ[s] = n
}

// AddEdge adds a directed edge from src to dst.
func (pg *ParityGame) AddEdge(src, dst int32) {
	pg.succ[src] = append(pg.succ[src], dst)
}

// buildPred builds the reverse adjacency lists lazily. CC=3.
func (pg *ParityGame) buildPred() {
	if pg.pred != nil {
		return
	}
	n := len(pg.owner)
	pg.pred = memory.MustPoolSlice[[]int32](pg.pool, n)
	pg.pred = pg.pred[:n]
	for i := range pg.pred {
		pg.pred[i] = memory.MustPoolSlice[int32](pg.pool, 0)
	}
	for s := int32(0); s < int32(n); s++ {
		for _, d := range pg.succ[s] {
			pg.pred[d] = append(pg.pred[d], s)
		}
	}
}

// MaxPriority returns the maximum priority in the game. CC=2.
func (pg *ParityGame) MaxPriority() int32 {
	m := int32(-1)
	for _, p := range pg.priority {
		if p > m {
			m = p
		}
	}
	return m
}

// StatesWithPriority returns all states with the given priority. CC=2.
func (pg *ParityGame) StatesWithPriority(pri int32) []int32 {
	result := memory.MustPoolSlice[int32](pg.pool, len(pg.priority))
	result = result[:0]
	for s, p := range pg.priority {
		if p == pri {
			result = append(result, int32(s))
		}
	}
	return result
}

// ZielonkaResult holds the winning regions for each player.
type ZielonkaResult struct {
	WinEven []int32 // states won by Player 0 (Even)
	WinOdd  []int32 // states won by Player 1 (Odd)
}

// Solve solves the parity game using Zielonka's recursive algorithm.
// Returns the winning regions for each player. CC=9.
func (pg *ParityGame) Solve() ZielonkaResult {
	return pg.zielonka(pg.allStates())
}

// allStates returns a Pool-backed slice of all state indices. CC=2.
func (pg *ParityGame) allStates() []int32 {
	n := len(pg.owner)
	s := memory.MustPoolSlice[int32](pg.pool, n)
	s = s[:n]
	for i := range s {
		s[i] = int32(i)
	}
	return s
}

// zielonka is the recursive Zielonka solver. CC=9.
func (pg *ParityGame) zielonka(states []int32) ZielonkaResult {
	if len(states) == 0 {
		return ZielonkaResult{}
	}
	// Find max priority in the current state set.
	p := pg.maxPriIn(states)
	U := pg.statesWithPriIn(states, p)
	player := Player(p % 2) // even priority → Player 0 (Even), odd → Player 1 (Odd)

	// A = attractor for `player` to U within `states`.
	A := pg.attractor(player, U, states)

	// Recurse on remaining states.
	remaining := pg.setMinus(states, A)
	rec := pg.zielonka(remaining)

	// Check if `player` can force winning from all of A.
	var oppWon []int32
	if player == PlayerEven {
		oppWon = rec.WinOdd
	} else {
		oppWon = rec.WinEven
	}
	oppInA := pg.intersect(A, oppWon)
	if len(oppInA) == 0 {
		// Player wins all of A — add A to player's winning region.
		if player == PlayerEven {
			return ZielonkaResult{
				WinEven: pg.union(rec.WinEven, A),
				WinOdd:  rec.WinOdd,
			}
		}
		return ZielonkaResult{
			WinEven: rec.WinEven,
			WinOdd:  pg.union(rec.WinOdd, A),
		}
	}

	// Opponent can win some states in A — remove opponent's attractor to those states.
	opp := 1 - player
	B := pg.attractor(opp, oppInA, states)
	rest := pg.setMinus(states, B)
	return pg.zielonka(rest)
}

// maxPriIn returns the maximum priority among the given states. CC=2.
func (pg *ParityGame) maxPriIn(states []int32) int32 {
	m := int32(-1)
	for _, s := range states {
		if pg.priority[s] > m {
			m = pg.priority[s]
		}
	}
	return m
}

// statesWithPriIn returns states in `states` that have priority `pri`. CC=2.
func (pg *ParityGame) statesWithPriIn(states []int32, pri int32) []int32 {
	result := memory.MustPoolSlice[int32](pg.pool, len(states))
	result = result[:0]
	for _, s := range states {
		if pg.priority[s] == pri {
			result = append(result, s)
		}
	}
	return result
}

// attractor computes the attractor for a player to a target set within universe.
// Uses reverse adjacency for O(m) complexity. CC=7.
func (pg *ParityGame) attractor(player Player, target, universe []int32) []int32 {
	pg.buildPred()
	uniMark := pg.markSet(universe)
	inResult := memory.MustPoolSlice[bool](pg.pool, len(pg.owner))
	inResult = inResult[:len(pg.owner)]
	result := memory.MustPoolSlice[int32](pg.pool, len(target))
	result = result[:0]
	queue := memory.MustPoolSlice[int32](pg.pool, len(target))
	queue = queue[:0]

	for _, t := range target {
		if !inResult[t] {
			inResult[t] = true
			result = append(result, t)
			queue = append(queue, t)
		}
	}

	// Count remaining outgoing edges to states outside the attractor.
	rem := memory.MustPoolSlice[int32](pg.pool, len(pg.owner))
	rem = rem[:len(pg.owner)]
	for _, u := range universe {
		rem[u] = int32(len(pg.succ[u]))
	}

	for len(queue) > 0 {
		v := queue[0]
		queue = queue[1:]

		for _, u := range pg.pred[v] {
			if inResult[u] || uniMark[u] == 0 {
				continue
			}
			if pg.owner[u] == player {
				inResult[u] = true
				result = append(result, u)
				queue = append(queue, u)
			} else {
				rem[u]--
				if rem[u] == 0 {
					inResult[u] = true
					result = append(result, u)
					queue = append(queue, u)
				}
			}
		}
	}
	return result
}

// markSet returns a slice where mark[s] = 1 if s is in set. CC=2.
func (pg *ParityGame) markSet(set []int32) []int32 {
	m := memory.MustPoolSlice[int32](pg.pool, len(pg.owner))
	m = m[:len(pg.owner)]
	for _, s := range set {
		m[s] = 1
	}
	return m
}

// setMinus returns states \ remove. CC=2.
func (pg *ParityGame) setMinus(states, remove []int32) []int32 {
	rm := pg.markSet(remove)
	result := memory.MustPoolSlice[int32](pg.pool, len(states))
	result = result[:0]
	for _, s := range states {
		if rm[s] == 0 {
			result = append(result, s)
		}
	}
	return result
}

// intersect returns the intersection of two state sets. CC=2.
func (pg *ParityGame) intersect(a, b []int32) []int32 {
	bm := pg.markSet(b)
	result := memory.MustPoolSlice[int32](pg.pool, len(a))
	result = result[:0]
	for _, s := range a {
		if bm[s] != 0 {
			result = append(result, s)
		}
	}
	return result
}

// union returns the union of two state sets. CC=2.
func (pg *ParityGame) union(a, b []int32) []int32 {
	result := memory.MustPoolSlice[int32](pg.pool, len(a)+len(b))
	result = result[:0]
	seen := memory.MustPoolSlice[bool](pg.pool, len(pg.owner))
	seen = seen[:len(pg.owner)]
	for _, s := range a {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	for _, s := range b {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}
	return result
}
