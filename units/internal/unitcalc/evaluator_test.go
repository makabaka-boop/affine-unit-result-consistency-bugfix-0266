package unitcalc

import (
	"errors"
	"math/big"
	"testing"
)

func mustEval(t *testing.T, src string) *Result {
	t.Helper()
	res, err := Evaluate(src)
	if err != nil {
		t.Fatalf("Evaluate(%q) unexpected error: %v", src, err)
	}
	return res
}

func mustErr(t *testing.T, src string) error {
	t.Helper()
	_, err := Evaluate(src)
	if err == nil {
		t.Fatalf("Evaluate(%q) expected error, got none", src)
	}
	return err
}

func rat(t *testing.T, s string) *big.Rat {
	t.Helper()
	r, ok := new(big.Rat).SetString(s)
	if !ok {
		t.Fatalf("bad rat %q", s)
	}
	return r
}

func TestLinearUnits(t *testing.T) {
	cases := []struct {
		src  string
		want string // exact rational in canonical base unit
		unit string
	}{
		{"1 m", "1", "m"},
		{"1 cm", "1/100", "m"},
		{"2 m + 30 cm", "23/10", "m"},
		{"1 kg + 500 g", "3/2", "kg"},
		{"1 min + 30 s", "90", "s"},
		{"2 m * 3 m", "6", "m²"},
		{"10 m / 2 s", "5", "m·s⁻¹"},
		{"1/2 kg + 1/4 kg", "3/4", "kg"},
		{"3.5 m", "7/2", "m"},
		{"(2 + 3) * 4 m", "20", "m"},
		{"2 * (3 m + 4 m)", "14", "m"},
		{"10 g / 2", "1/200", "kg"},
		{"-3 m + 5 m", "2", "m"},
		{"2 m * 3 kg", "6", "m·kg"},
		{"1 m / 1 s * 1 s", "1", "m"},
	}
	for _, c := range cases {
		res := mustEval(t, c.src)
		if res.Base.R.Cmp(rat(t, c.want)) != 0 {
			t.Errorf("%s: base = %s, want %s", c.src, res.Base.R.RatString(), c.want)
		}
		if res.CanonicalUnit != c.unit {
			t.Errorf("%s: canonical unit = %q, want %q", c.src, res.CanonicalUnit, c.unit)
		}
	}
}

func TestAbsoluteTemperatureRules(t *testing.T) {
	// K = C + 27315/100
	res := mustEval(t, "0 C")
	if res.Base.R.Cmp(rat(t, "27315/100")) != 0 {
		t.Errorf("0 C = %s K, want 27315/100", res.Base.R.RatString())
	}
	// K = (F - 32) * 5/9 + 27315/100
	res = mustEval(t, "32 F")
	if res.Base.R.Cmp(rat(t, "27315/100")) != 0 {
		t.Errorf("32 F = %s K, want 27315/100", res.Base.R.RatString())
	}
	res = mustEval(t, "212 F")
	if res.Base.R.Cmp(rat(t, "37315/100")) != 0 {
		t.Errorf("212 F = %s K, want 37315/100", res.Base.R.RatString())
	}

	// absolute - absolute = delta
	res = mustEval(t, "30 C - 20 C")
	if res.BaseKind != "delta" || res.Base.R.Cmp(rat(t, "10")) != 0 {
		t.Errorf("30 C - 20 C = %s (%s), want 10 delta", res.Base.R.RatString(), res.BaseKind)
	}
	res = mustEval(t, "68 F - 32 F")
	if res.BaseKind != "delta" || res.Base.R.Cmp(rat(t, "20")) != 0 {
		t.Errorf("68 F - 32 F = %s (%s), want 20 delta", res.Base.R.RatString(), res.BaseKind)
	}

	// absolute ± delta = absolute
	res = mustEval(t, "20 C + 5 dC")
	if res.BaseKind != "absolute" || res.Base.R.Cmp(rat(t, "29815/100")) != 0 {
		t.Errorf("20 C + 5 dC = %s (%s), want 29815/100 absolute", res.Base.R.RatString(), res.BaseKind)
	}
	res = mustEval(t, "5 dC + 20 C")
	if res.BaseKind != "absolute" || res.Base.R.Cmp(rat(t, "29815/100")) != 0 {
		t.Errorf("5 dC + 20 C = %s (%s), want 29815/100 absolute", res.Base.R.RatString(), res.BaseKind)
	}
	res = mustEval(t, "20 C - 5 dC")
	if res.BaseKind != "absolute" || res.Base.R.Cmp(rat(t, "28815/100")) != 0 {
		t.Errorf("20 C - 5 dC = %s (%s), want 28815/100 absolute", res.Base.R.RatString(), res.BaseKind)
	}

	// Fahrenheit interval is 5/9 kelvin
	res = mustEval(t, "9 dF")
	if res.Base.R.Cmp(rat(t, "5")) != 0 {
		t.Errorf("9 dF = %s dK, want 5", res.Base.R.RatString())
	}

	// delta arithmetic
	res = mustEval(t, "10 dC + 9 dF")
	if res.Base.R.Cmp(rat(t, "15")) != 0 {
		t.Errorf("10 dC + 9 dF = %s dK, want 15", res.Base.R.RatString())
	}

	// illegal: absolute + absolute
	if err := mustErr(t, "20 C + 30 C"); err == nil {
		t.Fatal("20 C + 30 C should be illegal")
	}
	// illegal: absolute * anything
	if err := mustErr(t, "20 C * 2"); err == nil {
		t.Fatal("20 C * 2 should be illegal")
	}
	if err := mustErr(t, "2 * 20 C"); err == nil {
		t.Fatal("2 * 20 C should be illegal")
	}
	// illegal: absolute / anything
	if err := mustErr(t, "20 C / 2"); err == nil {
		t.Fatal("20 C / 2 should be illegal")
	}
	if err := mustErr(t, "20 C / 20 C"); err == nil {
		t.Fatal("20 C / 20 C should be illegal")
	}
	// illegal: delta - absolute
	if err := mustErr(t, "5 dC - 20 C"); err == nil {
		t.Fatal("5 dC - 20 C should be illegal")
	}
	// illegal: negate absolute
	if err := mustErr(t, "-(20 C)"); err == nil {
		t.Fatal("-(20 C) should be illegal")
	}
	// illegal: dimension mismatch
	if err := mustErr(t, "1 m + 1 kg"); err == nil {
		t.Fatal("1 m + 1 kg should be illegal")
	}
	if err := mustErr(t, "1 m - 1 s"); err == nil {
		t.Fatal("1 m - 1 s should be illegal")
	}
	// illegal: delta + regular
	if err := mustErr(t, "5 dC + 1 m"); err == nil {
		t.Fatal("5 dC + 1 m should be illegal")
	}
	// division by zero
	if err := mustErr(t, "1 m / 0"); err == nil {
		t.Fatal("1 m / 0 should be illegal")
	}
}

func TestDeltaMultiplication(t *testing.T) {
	// delta temperature behaves like a regular quantity under * and /
	res := mustEval(t, "2 * 5 dC")
	if res.Base.R.Cmp(rat(t, "10")) != 0 {
		t.Errorf("2 * 5 dC = %s, want 10", res.Base.R.RatString())
	}
	res = mustEval(t, "10 dC / 2")
	if res.Base.R.Cmp(rat(t, "5")) != 0 {
		t.Errorf("10 dC / 2 = %s, want 5", res.Base.R.RatString())
	}
	res = mustEval(t, "3 dC * 2 m")
	if res.Base.R.Cmp(rat(t, "6")) != 0 || res.BaseDimText != "L·Θ" {
		t.Errorf("3 dC * 2 m = %s [%s], want 6 [L·Θ]", res.Base.R.RatString(), res.BaseDimText)
	}
}

func TestTargetUnitConversion(t *testing.T) {
	cases := []struct {
		src    string
		target string
		want   string
	}{
		{"0 C in K", "K", "5463/20"},
		{"273.15 K in C", "C", "0"},
		{"100 C in F", "F", "212"},
		{"32 F in C", "C", "0"},
		{"212 F in K", "K", "7463/20"},
		{"1 m in cm", "cm", "100"},
		{"250 cm in m", "m", "5/2"},
		{"1 kg in g", "g", "1000"},
		{"1500 g in kg", "kg", "3/2"},
		{"1 min in s", "s", "60"},
		{"90 s in min", "min", "3/2"},
		{"30 C - 20 C in dF", "dF", "18"},
		{"10 dC in dF", "dF", "18"},
		{"9 dF in dK", "dK", "5"},
		{"2 m + 30 cm in cm", "cm", "230"},
		{"(68 F - 32 F) in dC", "dC", "20"},
	}
	for _, c := range cases {
		res := mustEval(t, c.src)
		if res.TargetUnit != c.target {
			t.Errorf("%s: target unit = %q, want %q", c.src, res.TargetUnit, c.target)
		}
		if res.TargetValue == nil || res.TargetValue.R.Cmp(rat(t, c.want)) != 0 {
			got := "<nil>"
			if res.TargetValue != nil {
				got = res.TargetValue.R.RatString()
			}
			t.Errorf("%s: target value = %s, want %s", c.src, got, c.want)
		}
	}

	// absolute -> delta unit is illegal
	if err := mustErr(t, "20 C in dC"); err == nil {
		t.Fatal("20 C in dC should be illegal")
	}
	// delta -> absolute unit is illegal
	if err := mustErr(t, "5 dC in C"); err == nil {
		t.Fatal("5 dC in C should be illegal")
	}
	// dimension mismatch
	if err := mustErr(t, "1 m in kg"); err == nil {
		t.Fatal("1 m in kg should be illegal")
	}
	// non-temperature -> temperature
	if err := mustErr(t, "1 m in K"); err == nil {
		t.Fatal("1 m in K should be illegal")
	}
}

func TestErrorSpans(t *testing.T) {
	cases := []struct {
		src   string
		start int
		end   int
	}{
		{"20 C * 2", 0, 8},
		{"(20 C + 30 C)", 1, 12}, // minimal: inner expression
		{"1 m + 1 kg", 0, 10},
		{"5 dC - 20 C", 0, 11},
		{"2 * (20 C * 3)", 5, 13},
		{"1 m / 0", 0, 7},
		{"20 C in dC", 0, 10},
		{"abc", 0, 3},
		{"1 m +", 5, 5},
	}
	for _, c := range cases {
		_, err := Evaluate(c.src)
		if err == nil {
			t.Errorf("%q: expected error", c.src)
			continue
		}
		var se *SyntaxError
		var te *TypedOpError
		var s, e int
		switch {
		case errors.As(err, &se):
			s, e = se.Start, se.End
		case errors.As(err, &te):
			s, e = te.Start, te.End
		default:
			t.Errorf("%q: unexpected error type %T", c.src, err)
			continue
		}
		if s != c.start || e != c.end {
			wantSpan := Slice(c.src, c.start, c.end)
			gotSpan := Slice(c.src, s, e)
			t.Errorf("%q: span = [%d,%d) %q, want [%d,%d) %q",
				c.src, s, e, gotSpan, c.start, c.end, wantSpan)
		}
	}
}

func TestStepsRecorded(t *testing.T) {
	res := mustEval(t, "2 m + 30 cm")
	if len(res.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(res.Steps))
	}
	// post-order: operands first
	if res.Steps[0].NodeType != "number" || res.Steps[1].NodeType != "number" {
		t.Errorf("first two steps should be numbers: %+v", res.Steps)
	}
	if res.Steps[2].NodeType != "binary" || res.Steps[2].Op != "+" {
		t.Errorf("last step should be binary +: %+v", res.Steps[2])
	}
	if res.Steps[2].Value.R.Cmp(rat(t, "23/10")) != 0 {
		t.Errorf("final step value = %s, want 23/10", res.Steps[2].Value.R.RatString())
	}
	// every step carries type and dimension info
	for i, st := range res.Steps {
		if st.Kind == "" || st.DimText == "" {
			t.Errorf("step %d missing kind/dim: %+v", i, st)
		}
	}
}

func TestRationalReduction(t *testing.T) {
	res := mustEval(t, "2/4 m")
	if res.Base.R.Cmp(rat(t, "1/2")) != 0 {
		t.Errorf("2/4 m = %s, want 1/2 (reduced)", res.Base.R.RatString())
	}
	res = mustEval(t, "1/3 m + 1/6 m")
	if res.Base.R.Cmp(rat(t, "1/2")) != 0 {
		t.Errorf("1/3 + 1/6 = %s, want 1/2", res.Base.R.RatString())
	}
}

func TestBareUnitAndDegreeSign(t *testing.T) {
	res := mustEval(t, "m")
	if res.Base.R.Cmp(rat(t, "1")) != 0 {
		t.Errorf("bare m = %s, want 1", res.Base.R.RatString())
	}
	res = mustEval(t, "20 °C")
	if res.Base.R.Cmp(rat(t, "29315/100")) != 0 {
		t.Errorf("20 °C = %s K, want 29315/100", res.Base.R.RatString())
	}
}
