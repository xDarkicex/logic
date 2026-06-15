package modal

import (
	"math/bits"

	"github.com/xDarkicex/memory"
)

// WorldSet is a fixed-size bit-vector representing a set of worlds.
// Backed by a flat Pool-allocated []uint64 for cache-friendly O(1) operations.
// Union, intersection, and subset checks compile to SIMD instructions on ARM64.
type WorldSet struct {
	bits  []uint64
	words int
}

// NewWorldSet creates a WorldSet for up to maxWorlds worlds.
func NewWorldSet(maxWorlds int, pool *memory.Pool) *WorldSet {
	words := (maxWorlds + 63) / 64
	if words == 0 {
		words = 1
	}
	bits := memory.MustPoolSlice[uint64](pool, words)
	bits = bits[:words]
	return &WorldSet{bits: bits, words: words}
}

// Add inserts world w into the set.
func (s *WorldSet) Add(w World) {
	idx := int(w) / 64
	if idx < s.words {
		s.bits[idx] |= 1 << (uint(w) % 64)
	}
}

// Remove deletes world w from the set.
func (s *WorldSet) Remove(w World) {
	idx := int(w) / 64
	if idx < s.words {
		s.bits[idx] &^= 1 << (uint(w) % 64)
	}
}

// Contains returns true if world w is in the set.
func (s *WorldSet) Contains(w World) bool {
	idx := int(w) / 64
	if idx >= s.words {
		return false
	}
	return (s.bits[idx] & (1 << (uint(w) % 64))) != 0
}

// Clear removes all worlds from the set.
func (s *WorldSet) Clear() {
	for i := range s.bits {
		s.bits[i] = 0
	}
}

// Count returns the number of worlds in the set via hardware popcount.
func (s *WorldSet) Count() int {
	n := 0
	for _, v := range s.bits {
		n += bits.OnesCount64(v)
	}
	return n
}

// IsEmpty returns true if the set contains no worlds.
func (s *WorldSet) IsEmpty() bool {
	for _, v := range s.bits {
		if v != 0 {
			return false
		}
	}
	return true
}

// Union replaces s with s ∪ other.
func (s *WorldSet) Union(other *WorldSet) {
	n := s.words
	if other.words < n {
		n = other.words
	}
	for i := 0; i < n; i++ {
		s.bits[i] |= other.bits[i]
	}
}

// Intersect replaces s with s ∩ other.
func (s *WorldSet) Intersect(other *WorldSet) {
	n := s.words
	if other.words < n {
		n = other.words
	}
	for i := 0; i < n; i++ {
		s.bits[i] &= other.bits[i]
	}
}

// Subtract replaces s with s \ other.
func (s *WorldSet) Subtract(other *WorldSet) {
	n := s.words
	if other.words < n {
		n = other.words
	}
	for i := 0; i < n; i++ {
		s.bits[i] &^= other.bits[i]
	}
}

// Equals returns true if s and other contain exactly the same worlds.
func (s *WorldSet) Equals(other *WorldSet) bool {
	n := s.words
	if other.words < n {
		n = other.words
	}
	for i := 0; i < n; i++ {
		if s.bits[i] != other.bits[i] {
			return false
		}
	}
	// Check remaining words in larger set are all zero
	for i := n; i < s.words; i++ {
		if s.bits[i] != 0 {
			return false
		}
	}
	for i := n; i < other.words; i++ {
		if other.bits[i] != 0 {
			return false
		}
	}
	return true
}

// IsSubset returns true if s ⊆ other.
func (s *WorldSet) IsSubset(other *WorldSet) bool {
	n := s.words
	if other.words < n {
		n = other.words
	}
	for i := 0; i < n; i++ {
		if (s.bits[i] &^ other.bits[i]) != 0 {
			return false
		}
	}
	for i := n; i < s.words; i++ {
		if s.bits[i] != 0 {
			return false
		}
	}
	return true
}

// CopyFrom copies other's contents into s.
func (s *WorldSet) CopyFrom(other *WorldSet) {
	n := s.words
	if other.words < n {
		n = other.words
	}
	for i := 0; i < n; i++ {
		s.bits[i] = other.bits[i]
	}
	for i := n; i < s.words; i++ {
		s.bits[i] = 0
	}
}

// Fill marks all worlds [0, maxWorld) as present.
func (s *WorldSet) Fill(maxWorld int) {
	lastWord := maxWorld / 64
	for i := 0; i < lastWord; i++ {
		s.bits[i] = ^uint64(0)
	}
	if lastWord < s.words {
		rem := uint(maxWorld % 64)
		if rem > 0 {
			s.bits[lastWord] = (1 << rem) - 1
		}
		for i := lastWord + 1; i < s.words; i++ {
			s.bits[i] = 0
		}
	}
}

// Next returns the first world in the set at or after w, and true if found.
// Use start=0 to begin iteration. Returns (0, false) when exhausted.
func (s *WorldSet) Next(start World) (World, bool) {
	wordIdx := int(start) / 64
	if wordIdx >= s.words {
		return 0, false
	}
	bitOffset := uint(start) % 64

	// Check current word from start position
	v := s.bits[wordIdx] >> bitOffset << bitOffset
	if v != 0 {
		return World(wordIdx*64 + bits.TrailingZeros64(v)), true
	}

	// Check subsequent words
	for i := wordIdx + 1; i < s.words; i++ {
		if s.bits[i] != 0 {
			return World(i*64 + bits.TrailingZeros64(s.bits[i])), true
		}
	}
	return 0, false
}

// ToSlice collects all worlds in the set into a Pool-backed slice.
func (s *WorldSet) ToSlice(pool *memory.Pool) []World {
	result := memory.MustPoolSlice[World](pool, s.Count())
	result = result[:0]
	w, ok := s.Next(0)
	for ok {
		result = append(result, w)
		w, ok = s.Next(w + 1)
	}
	return result
}

// Words returns the number of uint64 words in this set.
func (s *WorldSet) Words() int { return s.words }
