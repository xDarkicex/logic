package classical

import (
	"fmt"
	"math/bits"
)

// BitwiseInt provides high-performance bitwise operations on 64-bit integers.
// It wraps a uint64 value and provides methods for bitwise manipulation,
// bit querying, and conversion operations. All operations return new instances
// to maintain immutability.
type BitwiseInt struct {
	value uint64
}

// NewBitwiseInt creates a new BitwiseInt with the given value.
// The value is stored as a uint64 internally.
//
// Example:
//
//	bi := NewBitwiseInt(42)        // Binary: 101010
//	bi := NewBitwiseInt(0b1010)    // Using binary literal
//	bi := NewBitwiseInt(0xFF)      // Using hex literal
func NewBitwiseInt(value uint64) BitwiseInt {
	return BitwiseInt{value: value}
}

// Value returns the underlying uint64 value.
// This provides direct access to the wrapped integer value.
//
// Example:
//
//	bi := NewBitwiseInt(42)
//	val := bi.Value() // 42
func (bi BitwiseInt) Value() uint64 {
	return bi.value
}

// And performs bitwise AND operation with another BitwiseInt.
// Returns a new BitwiseInt with the result of the operation.
//
// Example:
//
//	a := NewBitwiseInt(0b1010) // 10
//	b := NewBitwiseInt(0b1100) // 12
//	result := a.And(b)         // 0b1000 (8)
func (bi BitwiseInt) And(other BitwiseInt) BitwiseInt {
	return BitwiseInt{value: bi.value & other.value}
}

// Or performs bitwise OR operation with another BitwiseInt.
// Returns a new BitwiseInt with the result of the operation.
//
// Example:
//
//	a := NewBitwiseInt(0b1010) // 10
//	b := NewBitwiseInt(0b1100) // 12
//	result := a.Or(b)          // 0b1110 (14)
func (bi BitwiseInt) Or(other BitwiseInt) BitwiseInt {
	return BitwiseInt{value: bi.value | other.value}
}

// Xor performs bitwise XOR operation with another BitwiseInt.
// Returns a new BitwiseInt with the result of the operation.
//
// Example:
//
//	a := NewBitwiseInt(0b1010) // 10
//	b := NewBitwiseInt(0b1100) // 12
//	result := a.Xor(b)         // 0b0110 (6)
func (bi BitwiseInt) Xor(other BitwiseInt) BitwiseInt {
	return BitwiseInt{value: bi.value ^ other.value}
}

// Not performs bitwise NOT operation (one's complement).
// Returns a new BitwiseInt with all bits flipped.
//
// Example:
//
//	a := NewBitwiseInt(0b1010)
//	result := a.Not() // All bits flipped
func (bi BitwiseInt) Not() BitwiseInt {
	return BitwiseInt{value: ^bi.value}
}

// SetBit sets the bit at the specified position to 1.
// Position 0 is the least significant bit. Returns a new BitwiseInt.
//
// Example:
//
//	a := NewBitwiseInt(0b1010) // 10
//	result := a.SetBit(0)      // 0b1011 (11)
func (bi BitwiseInt) SetBit(pos uint) BitwiseInt {
	return BitwiseInt{value: bi.value | (1 << pos)}
}

// ClearBit sets the bit at the specified position to 0.
// Position 0 is the least significant bit. Returns a new BitwiseInt.
//
// Example:
//
//	a := NewBitwiseInt(0b1011) // 11
//	result := a.ClearBit(0)    // 0b1010 (10)
func (bi BitwiseInt) ClearBit(pos uint) BitwiseInt {
	return BitwiseInt{value: bi.value &^ (1 << pos)}
}

// ToggleBit flips the bit at the specified position.
// Position 0 is the least significant bit. Returns a new BitwiseInt.
//
// Example:
//
//	a := NewBitwiseInt(0b1010) // 10
//	result := a.ToggleBit(0)   // 0b1011 (11)
//	result2 := a.ToggleBit(1)  // 0b1000 (8)
func (bi BitwiseInt) ToggleBit(pos uint) BitwiseInt {
	return BitwiseInt{value: bi.value ^ (1 << pos)}
}

// GetBit returns the value of the bit at the specified position.
// Position 0 is the least significant bit.
//
// Example:
//
//	a := NewBitwiseInt(0b1010) // 10
//	bit0 := a.GetBit(0)        // false
//	bit1 := a.GetBit(1)        // true
func (bi BitwiseInt) GetBit(pos uint) bool {
	return (bi.value>>pos)&1 == 1
}

// CountSetBits returns the number of set bits (population count).
// Uses hardware-optimized bit counting when available.
//
// Example:
//
//	a := NewBitwiseInt(0b1011) // 11
//	count := a.CountSetBits()  // 3
func (bi BitwiseInt) CountSetBits() int {
	return bits.OnesCount64(bi.value)
}

// IsPowerOfTwo checks if the value is a power of 2.
// Returns true if the value is a power of 2, false otherwise.
// Zero is not considered a power of 2.
//
// Example:
//
//	a := NewBitwiseInt(8)
//	a.IsPowerOfTwo() // true
//	b := NewBitwiseInt(10)
//	b.IsPowerOfTwo() // false
func (bi BitwiseInt) IsPowerOfTwo() bool {
	return bi.value != 0 && (bi.value&(bi.value-1)) == 0
}

// LeftShift performs left bit shift by n positions.
// Equivalent to multiplying by 2^n. Returns a new BitwiseInt.
//
// Example:
//
//	a := NewBitwiseInt(5)      // 0b101
//	result := a.LeftShift(2)   // 0b10100 (20)
func (bi BitwiseInt) LeftShift(n uint) BitwiseInt {
	return BitwiseInt{value: bi.value << n}
}

// RightShift performs right bit shift by n positions.
// Equivalent to dividing by 2^n (integer division). Returns a new BitwiseInt.
//
// Example:
//
//	a := NewBitwiseInt(20)     // 0b10100
//	result := a.RightShift(2)  // 0b101 (5)
func (bi BitwiseInt) RightShift(n uint) BitwiseInt {
	return BitwiseInt{value: bi.value >> n}
}

// ToBoolVector converts the BitwiseInt to a BoolVector of 64 bits.
// The least significant bit becomes index 0 in the vector.
//
// Example:
//
//	bi := NewBitwiseInt(5) // 0b101
//	vec := bi.ToBoolVector()
//	// vec[0] = true, vec[1] = false, vec[2] = true, vec[3..63] = false
func (bi BitwiseInt) ToBoolVector() BoolVector {
	result := make(BoolVector, 64)
	value := bi.value
	for i := 0; i < 64; i++ {
		result[i] = (value & 1) == 1
		value >>= 1
	}
	return result
}

// String returns a string representation of the BitwiseInt.
// Shows both binary and decimal representations for clarity.
//
// Example:
//
//	bi := NewBitwiseInt(42)
//	str := bi.String() // "0b0000000000000000000000000000000000000000000000000000000000101010 (42)"
func (bi BitwiseInt) String() string {
	return fmt.Sprintf("0b%064b (%d)", bi.value, bi.value)
}
