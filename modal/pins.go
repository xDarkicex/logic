package modal

import (
	"math/bits"

	"github.com/xDarkicex/logic/fuzzy"
	"github.com/xDarkicex/memory"
)

// DepMatrix is a PINS dependency matrix: formulas × atomic propositions.
// Stored as a flat 1D Pool-backed []uint64 for cache-friendly bit operations.
// Row f starts at bits[f * wordsPerRow]. Column a is bit (a % 64) in word (a / 64).
type DepMatrix struct {
	bits        []uint64
	rows        int
	wordsPerRow int
	atomCount   int
	pool        *memory.Pool
}

// NewDepMatrix creates a dependency matrix for maxFormulas formulas and maxAtoms atoms.
func NewDepMatrix(maxFormulas, maxAtoms int, pool *memory.Pool) *DepMatrix {
	wpr := (maxAtoms + 63) / 64
	if wpr == 0 {
		wpr = 1
	}
	total := maxFormulas * wpr
	bits := memory.MustPoolSlice[uint64](pool, total)
	bits = bits[:total]
	return &DepMatrix{
		bits:        bits,
		rows:        maxFormulas,
		wordsPerRow: wpr,
		atomCount:   maxAtoms,
		pool:        pool,
	}
}

// Row returns the uint64 slice for formula row f.
func (m *DepMatrix) Row(f int) []uint64 {
	start := f * m.wordsPerRow
	return m.bits[start : start+m.wordsPerRow]
}

// Set sets the dependency bit for formula f on atom a.
func (m *DepMatrix) Set(f int, a fuzzy.VarID) {
	col := int(a)
	if col < m.atomCount {
		word := col / 64
		bit := uint(col % 64)
		m.bits[f*m.wordsPerRow+word] |= 1 << bit
	}
}

// Has checks whether formula f depends on atom a.
func (m *DepMatrix) Has(f int, a fuzzy.VarID) bool {
	col := int(a)
	if col >= m.atomCount {
		return false
	}
	word := col / 64
	bit := uint(col % 64)
	return (m.bits[f*m.wordsPerRow+word] & (1 << bit)) != 0
}

// AnySet returns true if formula f has any dependency on any atom in the dirty mask.
// CC=2.
func (m *DepMatrix) AnySet(f int, dirty []uint64) bool {
	start := f * m.wordsPerRow
	for i := 0; i < m.wordsPerRow; i++ {
		if (m.bits[start+i] & dirty[i]) != 0 {
			return true
		}
	}
	return false
}

// Span returns the event span of formula f: (last_col - first_col + 1), or 0 if no deps.
func (m *DepMatrix) Span(f int) int {
	start := f * m.wordsPerRow
	first := -1
	last := -1
	for w := 0; w < m.wordsPerRow; w++ {
		v := m.bits[start+w]
		if v == 0 {
			continue
		}
		col := w*64 + bits.TrailingZeros64(v)
		if first < 0 || col < first {
			first = col
		}
		col = w*64 + 63 - bits.LeadingZeros64(v)
		if col > last {
			last = col
		}
	}
	if first < 0 {
		return 0
	}
	return last - first + 1
}

// EventSpan returns the total event span across all formulas. CC=3.
func (m *DepMatrix) EventSpan() int {
	total := 0
	for f := 0; f < m.rows; f++ {
		total += m.Span(f)
	}
	return total
}

// WorldDirty tracks which atomic propositions changed at each world.
// Stored as a flat []uint64 per world.
type WorldDirty struct {
	bits        []uint64 // world * wordsPerRow
	wordsPerRow int
	atoms       int
	pool        *memory.Pool
}

// NewWorldDirty creates a dirty tracker for maxWorlds worlds and maxAtoms atoms.
func NewWorldDirty(maxWorlds, maxAtoms int, pool *memory.Pool) *WorldDirty {
	wpr := (maxAtoms + 63) / 64
	if wpr == 0 {
		wpr = 1
	}
	bits := memory.MustPoolSlice[uint64](pool, maxWorlds*wpr)
	bits = bits[:maxWorlds*wpr]
	return &WorldDirty{
		bits:        bits,
		wordsPerRow: wpr,
		atoms:       maxAtoms,
		pool:        pool,
	}
}

// Mark atom a as dirty at world w.
func (d *WorldDirty) Mark(w World, a fuzzy.VarID) {
	col := int(a)
	if col >= d.atoms {
		return
	}
	word := col / 64
	bit := uint(col % 64)
	d.bits[int(w)*d.wordsPerRow+word] |= 1 << bit
}

// MarkAll sets all atoms as dirty at world w.
func (d *WorldDirty) MarkAll(w World) {
	start := int(w) * d.wordsPerRow
	for i := 0; i < d.wordsPerRow; i++ {
		d.bits[start+i] = ^uint64(0)
	}
}

// Clean clears all dirty flags at world w.
func (d *WorldDirty) Clean(w World) {
	start := int(w) * d.wordsPerRow
	for i := 0; i < d.wordsPerRow; i++ {
		d.bits[start+i] = 0
	}
}

// Row returns the dirty bitmask for world w.
func (d *WorldDirty) Row(w World) []uint64 {
	start := int(w) * d.wordsPerRow
	return d.bits[start : start+d.wordsPerRow]
}

// ComputeDeps walks a formula tree and sets dependency bits for all atoms found.
// CC=5.
func ComputeDeps(f Formula, m *DepMatrix, row int) {
	switch t := f.(type) {
	case Atom:
		m.Set(row, t.ID)
	case Not:
		ComputeDeps(t.Formula, m, row)
	case Box:
		ComputeDeps(t.Formula, m, row)
	case Diamond:
		ComputeDeps(t.Formula, m, row)
	case Next:
		ComputeDeps(t.Formula, m, row)
	case And:
		ComputeDeps(t.Left, m, row)
		ComputeDeps(t.Right, m, row)
	case Or:
		ComputeDeps(t.Left, m, row)
		ComputeDeps(t.Right, m, row)
	case Implies:
		ComputeDeps(t.Antecedent, m, row)
		ComputeDeps(t.Consequent, m, row)
	case Iff:
		ComputeDeps(t.Left, m, row)
		ComputeDeps(t.Right, m, row)
	case Until:
		ComputeDeps(t.Left, m, row)
		ComputeDeps(t.Right, m, row)
	}
}

// PINSRegistry wraps the formula registry with dependency tracking.
// Each Intern'd formula gets a sequential ID for the dependency matrix.
type PINSRegistry struct {
	reg    *Registry
	matrix *DepMatrix
}

// NewPINSRegistry creates a registry with integrated dependency tracking.
func NewPINSRegistry(regCap, maxForms, maxAtoms int, pool *memory.Pool) *PINSRegistry {
	return &PINSRegistry{
		reg:    NewRegistry(regCap, pool),
		matrix: NewDepMatrix(maxForms, maxAtoms, pool),
	}
}

// Intern canonicalizes f, assigns a PINS ID, and computes its dependency row. CC=2.
func (p *PINSRegistry) Intern(f Formula) Formula {
	canon := p.reg.Intern(f)
	id := p.reg.GetID(canon)
	if id >= 0 && id < p.matrix.rows {
		ComputeDeps(canon, p.matrix, id)
	}
	return canon
}

// LookupID returns the PINS matrix row index for a canonical formula. CC=1.
func (p *PINSRegistry) LookupID(f Formula) int { return p.reg.GetID(f) }

// IsClean returns true if formula f's deps don't overlap with dirty atoms at world w. CC=2.
func (p *PINSRegistry) IsClean(formulaID int, w World, dirty *WorldDirty) bool {
	if formulaID < 0 || formulaID >= p.matrix.rows {
		return false
	}
	return !p.matrix.AnySet(formulaID, dirty.Row(w))
}

// DepMatrix returns the underlying dependency matrix.
func (p *PINSRegistry) DepMatrix() *DepMatrix { return p.matrix }

// Registry returns the underlying formula registry.
func (p *PINSRegistry) Registry() *Registry { return p.reg }
