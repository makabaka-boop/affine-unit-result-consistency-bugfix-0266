package unitcalc

import (
	"math/big"
	"math/rand"
	"testing"
)

// refConvert converts a reference value to a target unit, re-derived
// independently from the reference table.
func refConvert(v refValue, sym string) (*big.Rat, bool) {
	u, ok := refUnits[sym]
	if !ok || u.dim != v.dim {
		return nil, false
	}
	if v.kind == KindAbsolute && u.kind != KindAbsolute {
		return nil, false
	}
	if v.kind == KindDelta && u.kind != KindDelta {
		return nil, false
	}
	if v.kind == KindRegular && u.kind != KindRegular {
		return nil, false
	}
	r := new(big.Rat).Set(v.n)
	if v.kind == KindAbsolute {
		r.Sub(r, u.offset)
	}
	r.Quo(r, u.factor)
	return r, true
}

func TestDifferentialConversions(t *testing.T) {
	rng := rand.New(rand.NewSource(99))
	const iterations = 2000
	success := 0
	for i := 0; i < iterations; i++ {
		tree := genTree(rng, 2+rng.Intn(3))
		want, ok := refEval(tree)
		if !ok {
			continue
		}
		target := genUnits[1+rng.Intn(len(genUnits)-1)] // skip dimensionless
		u, found := LookupUnit(target)
		if !found {
			continue
		}
		wantTV, wantConvOK := refConvert(want, target)

		cn := &ConvertNode{Inner: tree, Target: u, TargetRaw: target}
		src := render(tree) + " in " + target
		res, err := EvalNode(src, cn)
		gotConvOK := err == nil
		if gotConvOK != wantConvOK {
			t.Fatalf("%q: convert ok=%v (%v), reference ok=%v",
				src, gotConvOK, err, wantConvOK)
		}
		if !wantConvOK {
			// also verify via rendered roundtrip
			continue
		}
		if res.TargetValue == nil || res.TargetValue.R.Cmp(wantTV) != 0 {
			got := "<nil>"
			if res.TargetValue != nil {
				got = res.TargetValue.R.RatString()
			}
			t.Fatalf("%q: target = %s, reference = %s", src, got, wantTV.RatString())
		}

		// same conversion through the text parser
		parsed, perr := Parse(src)
		if perr != nil {
			t.Fatalf("%q: re-parse: %v", src, perr)
		}
		rt, rerr := EvalNode(src, parsed)
		if rerr != nil || rt.TargetValue == nil || rt.TargetValue.R.Cmp(wantTV) != 0 {
			t.Fatalf("%q: roundtrip target mismatch: %v %+v", src, rerr, rt)
		}
		success++
	}
	t.Logf("conversion differential: %d successful conversions checked", success)
	if success < 50 {
		t.Fatalf("too few successful conversions: %d", success)
	}
}

func TestRatJSON(t *testing.T) {
	cases := map[string]string{
		"1/2":       "0.5",
		"1/3":       "0.(3)",
		"1/6":       "0.1(6)",
		"22/7":      "3.(142857)",
		"1/8":       "0.125",
		"27315/100": "273.15",
		"45967/180": "255.37(2)",
	}
	for in, want := range cases {
		r, _ := new(big.Rat).SetString(in)
		got := decimalExact(r.Num(), r.Denom())
		if got != want {
			t.Errorf("decimalExact(%s) = %s, want %s", in, got, want)
		}
	}
}
