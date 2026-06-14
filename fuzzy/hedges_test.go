package fuzzy

import (
	"math"
	"testing"
)

func TestHedges(t *testing.T) {
	// Base MF that returns x
	mf := func(x float64) TruthValue {
		return TruthValue(x)
	}

	very := Very(mf)
	assertClose(t, 0.25, very(0.5), "Very 0.5")
	assertClose(t, 1.0, very(1.0), "Very 1.0")
	assertClose(t, 0.0, very(0.0), "Very 0.0")

	somewhat := Somewhat(mf)
	assertClose(t, TruthValue(math.Sqrt(0.5)), somewhat(0.5), "Somewhat 0.5")
	assertClose(t, 1.0, somewhat(1.0), "Somewhat 1.0")

	slightly := Slightly(mf)
	assertClose(t, TruthValue(math.Sqrt(0.5)*math.Sqrt(0.5)), slightly(0.5), "Slightly 0.5")

	extremely := Extremely(mf)
	assertClose(t, 0.125, extremely(0.5), "Extremely 0.5")
	assertClose(t, 1.0, extremely(1.0), "Extremely 1.0")

	indeed := Indeed(mf)
	// For <= 0.5: 2 * v^2
	assertClose(t, 2.0*0.2*0.2, indeed(0.2), "Indeed 0.2")
	assertClose(t, 2.0*0.5*0.5, indeed(0.5), "Indeed 0.5")
	// For > 0.5: 1 - 2*(1-v)^2
	inv := 1.0 - 0.8
	assertClose(t, TruthValue(1.0-2.0*inv*inv), indeed(0.8), "Indeed 0.8")

	not := Not(mf)
	assertClose(t, 0.7, not(0.3), "Not 0.3")
	assertClose(t, 0.0, not(1.0), "Not 1.0")
	assertClose(t, 1.0, not(0.0), "Not 0.0")
}
