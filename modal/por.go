package modal

import "github.com/xDarkicex/memory"

// POR implements §27 Guard-Based Partial Order Reduction.
// Two tableau expansion rules commute if their PINS atomic proposition
// footprints are disjoint. The stubborn set algorithm selects a minimal
// set of enabled formulas to expand, postponing independent ones.
type POR struct {
	matrix *DepMatrix
	reg    *Registry
	pool   *memory.Pool
}

// NewPOR creates a POR filter backed by the PINS dependency matrix.
func NewPOR(matrix *DepMatrix, reg *Registry, pool *memory.Pool) *POR {
	return &POR{matrix: matrix, reg: reg, pool: pool}
}

// Commute returns true if two formulas can be expanded in any order.
// They commute iff their atomic proposition footprints are disjoint.
// CC=2.
func (p *POR) Commute(a, b Formula) bool {
	fa := p.footprint(a)
	fb := p.footprint(b)
	return !footprintOverlap(fa, fb)
}

// footprint returns the PINS row for formula f as a uint64 slice.
// Falls back to on-the-fly computation if the formula is not in PINS.
func (p *POR) footprint(f Formula) []uint64 {
	id := p.reg.GetID(p.reg.Intern(f))
	if id >= 0 && id < p.matrix.rows {
		return p.matrix.Row(id)
	}
	return computeFootprint(f, nil, p.pool)
}

// StubbornSet computes a minimal stubborn set of formula indices.
// Given a slice of prefixed formulas, returns indices of formulas that
// must be expanded in this round. Formulas outside the set commute with
// everything inside and can be postponed safely.
// CC=7.
func (p *POR) StubbornSet(formulas []PrefixedFormula) []int {
	n := len(formulas)
	if n <= 1 {
		all := memory.MustPoolSlice[int](p.pool, n)
		all = all[:n]
		for i := 0; i < n; i++ {
			all[i] = i
		}
		return all
	}

	// Compute footprints for all formulas
	fps := memory.MustPoolSlice[[]uint64](p.pool, n)
	fps = fps[:n]
	for i, pf := range formulas {
		fps[i] = p.footprint(pf.Formula)
	}

	// dna[i][j] = true if formulas i and j do NOT accord (share atoms)
	dna := memory.MustPoolSlice[[]bool](p.pool, n)
	dna = dna[:n]
	for i := 0; i < n; i++ {
		row := memory.MustPoolSlice[bool](p.pool, n)
		row = row[:n]
		for j := 0; j < n; j++ {
			if i != j {
				row[j] = footprintOverlap(fps[i], fps[j])
			}
		}
		dna[i] = row
	}

	// Pick an enabled seed and close under dna
	inSet := memory.MustPoolSlice[bool](p.pool, n)
	inSet = inSet[:n]
	inSet[0] = true

	// Transitive closure: add everything that doesn't accord with anything in the set
	changed := true
	for changed {
		changed = false
		for i := 0; i < n; i++ {
			if inSet[i] {
				continue
			}
			for j := 0; j < n; j++ {
				if inSet[j] && dna[i][j] {
					inSet[i] = true
					changed = true
					break
				}
			}
		}
	}

	result := memory.MustPoolSlice[int](p.pool, n)
	result = result[:0]
	for i := 0; i < n; i++ {
		if inSet[i] {
			result = append(result, i)
		}
	}
	if len(result) == 0 {
		result = append(result, 0)
	}
	return result
}

// Filter returns only the formulas in the stubborn set.
// CC=1.
func (p *POR) Filter(formulas []PrefixedFormula) []PrefixedFormula {
	indices := p.StubbornSet(formulas)
	result := memory.MustPoolSlice[PrefixedFormula](p.pool, len(indices))
	result = result[:0]
	for _, idx := range indices {
		result = append(result, formulas[idx])
	}
	return result
}
