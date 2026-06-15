package modal

import "github.com/xDarkicex/memory"

// StutterStatus describes stutter invariance of a formula.
type StutterStatus uint8

const (
	StutterInvariant StutterStatus = iota // safe for model compression
	StutterSensitive                      // depends on state count, needs full model
)

// IsStutterInvariant returns true if f can be checked on a stutter-compressed model.
// A formula without ○ (Next) is trivially invariant. Formulas with ○ only under
// □ (always) or ◇ (eventually) are also invariant — the Box/Diamond quantifies
// over all states, making exact state count irrelevant.
// CC=6.
func IsStutterInvariant(f Formula) bool {
	return checkStutter(f, false)
}

// checkStutter recursively checks stutter invariance.
// insideModal is true when we are inside a □ or ◇ (which absorb ○ sensitivity). CC=7.
func checkStutter(f Formula, insideModal bool) bool {
	switch t := f.(type) {
	case Atom:
		return true

	case Not:
		return checkStutter(t.Formula, insideModal)

	case And:
		return checkStutter(t.Left, insideModal) && checkStutter(t.Right, insideModal)

	case Or:
		return checkStutter(t.Left, insideModal) && checkStutter(t.Right, insideModal)

	case Implies:
		return checkStutter(t.Antecedent, insideModal) && checkStutter(t.Consequent, insideModal)

	case Iff:
		return checkStutter(t.Left, insideModal) && checkStutter(t.Right, insideModal)

	case Box:
		// □ absorbs ○ sensitivity: □○p is stutter-invariant.
		return checkStutter(t.Formula, true)

	case Diamond:
		// ◇ absorbs ○ sensitivity: ◇○p is stutter-invariant.
		return checkStutter(t.Formula, true)

	case Next:
		// ○ is NOT stutter-invariant unless inside □ or ◇.
		return insideModal

	case Until:
		// p U q: stutter-invariant if both sides are.
		return checkStutter(t.Left, insideModal) && checkStutter(t.Right, insideModal)

	default:
		return false
	}
}

// CanCompress returns true if the model can be stutter-compressed for this formula.
// A stutter-invariant formula permits removing consecutive duplicate states. CC=2.
func CanCompress(f Formula) bool {
	return IsStutterInvariant(f)
}

// StutterCompress compresses a valuation sequence by removing consecutive duplicates.
// The compressed slice preserves all distinct states in order. Uses Pool backing. CC=5.
func StutterCompress(val []TruthValueSlice, pool *memory.Pool) []TruthValueSlice {
	if len(val) <= 1 {
		return val
	}
	n := len(val)
	result := memory.MustPoolSlice[TruthValueSlice](pool, n)
	result = result[:0]
	result = append(result, val[0])
	for i := 1; i < n; i++ {
		if !valSliceEqual(val[i], val[i-1]) {
			result = append(result, val[i])
		}
	}
	return result
}

// valSliceEqual returns true if two valuation rows are identical. CC=2.
func valSliceEqual(a, b TruthValueSlice) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// CompressRatio returns the compression ratio achieved on a valuation sequence.
// A ratio of 0.0 means no compression (all distinct). 0.8 means 80% of states removed. CC=3.
func CompressRatio(original, compressed []TruthValueSlice) float64 {
	if len(original) == 0 {
		return 0
	}
	return 1.0 - float64(len(compressed))/float64(len(original))
}
