package modal

import (
	"math/bits"

	"github.com/xDarkicex/memory"
)

// Registry provides formula hash consing — structurally identical formulas
// share the same pointer. This gives O(1) equality via pointer comparison
// and eliminates redundant formula objects.
//
// Backed by a Pool-allocated open-addressing hash table. No Go maps.
type Registry struct {
	buckets []regEntry // Pool-backed, power-of-2 sized
	mask    uint32
	count   int
	pool    *memory.Pool
}

type regEntry struct {
	f    Formula
	hash uint64
	used bool
}

// NewRegistry creates a formula hash cons registry with the given capacity.
// Capacity is rounded up to the next power of 2.
func NewRegistry(minCap int, pool *memory.Pool) *Registry {
	cap := nextPow2(minCap)
	if cap < 16 {
		cap = 16
	}
	buckets := memory.MustPoolSlice[regEntry](pool, cap)
	buckets = buckets[:cap]
	return &Registry{
		buckets: buckets,
		mask:    uint32(cap - 1),
		pool:    pool,
	}
}

func nextPow2(n int) int {
	if n <= 1 {
		return 1
	}
	return 1 << (bits.Len(uint(n - 1)))
}

// Intern returns the canonical Formula for f. If an equivalent formula
// already exists in the registry, it is returned. Otherwise f is stored
// and returned. CC=7.
func (r *Registry) Intern(f Formula) Formula {
	if f == nil {
		return nil
	}
	h := r.hashOf(f)
	idx := uint32(h) & r.mask

	for {
		e := &r.buckets[idx]
		if !e.used {
			if r.count >= len(r.buckets)/2 {
				r.grow()
				return r.Intern(f)
			}
			e.f = f
			e.hash = h
			e.used = true
			r.count++
			return f
		}
		if e.hash == h && formulaEqual(e.f, f) {
			return e.f
		}
		idx = (idx + 1) & r.mask
	}
}

// grow doubles the bucket array and rehashes. CC=3.
func (r *Registry) grow() {
	newCap := len(r.buckets) * 2
	newBuckets := memory.MustPoolSlice[regEntry](r.pool, newCap)
	newBuckets = newBuckets[:newCap]
	newMask := uint32(newCap - 1)

	for i := range r.buckets {
		if !r.buckets[i].used {
			continue
		}
		e := r.buckets[i]
		idx := uint32(e.hash) & newMask
		for newBuckets[idx].used {
			idx = (idx + 1) & newMask
		}
		newBuckets[idx] = e
	}
	r.buckets = newBuckets
	r.mask = newMask
}

// hashOf computes a structural hash for a formula. CC=5.
func (r *Registry) hashOf(f Formula) uint64 {
	switch t := f.(type) {
	case Atom:
		return hashAtom(uint32(t.ID))
	case Not:
		return hashNot(r.hashOf(t.Formula))
	case Box:
		return hashBox(r.hashOf(t.Formula), uint8(t.Rel))
	case Diamond:
		return hashDiamond(r.hashOf(t.Formula), uint8(t.Rel))
	case Next:
		return hashNext(r.hashOf(t.Formula))
	case And:
		return hashBin(r.hashOf(t.Left), r.hashOf(t.Right), tagAnd)
	case Or:
		return hashBin(r.hashOf(t.Left), r.hashOf(t.Right), tagOr)
	case Implies:
		return hashBin(r.hashOf(t.Antecedent), r.hashOf(t.Consequent), tagImplies)
	case Iff:
		return hashBin(r.hashOf(t.Left), r.hashOf(t.Right), tagIff)
	case Until:
		return hashBin(r.hashOf(t.Left), r.hashOf(t.Right), tagUntil)
	}
	return 0
}

// formulaEqual compares two formulas for structural equality. CC=5.
func formulaEqual(a, b Formula) bool {
	if a == b {
		return true
	}
	switch at := a.(type) {
	case Atom:
		bt, ok := b.(Atom)
		return ok && at.ID == bt.ID
	case Not:
		bt, ok := b.(Not)
		return ok && formulaEqual(at.Formula, bt.Formula)
	case Box:
		bt, ok := b.(Box)
		return ok && at.Rel == bt.Rel && formulaEqual(at.Formula, bt.Formula)
	case Diamond:
		bt, ok := b.(Diamond)
		return ok && at.Rel == bt.Rel && formulaEqual(at.Formula, bt.Formula)
	case Next:
		bt, ok := b.(Next)
		return ok && formulaEqual(at.Formula, bt.Formula)
	case And:
		bt, ok := b.(And)
		return ok && formulaEqual(at.Left, bt.Left) && formulaEqual(at.Right, bt.Right)
	case Or:
		bt, ok := b.(Or)
		return ok && formulaEqual(at.Left, bt.Left) && formulaEqual(at.Right, bt.Right)
	case Implies:
		bt, ok := b.(Implies)
		return ok && formulaEqual(at.Antecedent, bt.Antecedent) && formulaEqual(at.Consequent, bt.Consequent)
	case Iff:
		bt, ok := b.(Iff)
		return ok && formulaEqual(at.Left, bt.Left) && formulaEqual(at.Right, bt.Right)
	case Until:
		bt, ok := b.(Until)
		return ok && formulaEqual(at.Left, bt.Left) && formulaEqual(at.Right, bt.Right)
	}
	return false
}

// --- Hash primitives (FNV-1a inspired, deterministic) ---

const (
	fnvPrime  = 0x100000001b3
	fnvOffset = 0xcbf29ce484222325
)

// Operator tags for hash differentiation.
const (
	tagAtom    uint64 = 1
	tagNot     uint64 = 2
	tagBox     uint64 = 3
	tagDiamond uint64 = 4
	tagNext    uint64 = 5
	tagAnd     uint64 = 6
	tagOr      uint64 = 7
	tagImplies uint64 = 8
	tagIff     uint64 = 9
	tagUntil   uint64 = 10
)

func hashAtom(id uint32) uint64 {
	h := fnvOffset ^ tagAtom
	h ^= uint64(id)
	h *= fnvPrime
	return h
}

func hashNot(inner uint64) uint64 {
	h := inner ^ tagNot
	h *= fnvPrime
	return h
}

func hashBox(inner uint64, rel uint8) uint64 {
	h := inner ^ tagBox
	h ^= uint64(rel) << 32
	h *= fnvPrime
	return h
}

func hashDiamond(inner uint64, rel uint8) uint64 {
	h := inner ^ tagDiamond
	h ^= uint64(rel) << 32
	h *= fnvPrime
	return h
}

func hashNext(inner uint64) uint64 {
	h := inner ^ tagNext
	h *= fnvPrime
	return h
}

func hashBin(left, right, tag uint64) uint64 {
	h := left ^ bits.RotateLeft64(right, 17) ^ tag
	h *= fnvPrime
	return h
}
