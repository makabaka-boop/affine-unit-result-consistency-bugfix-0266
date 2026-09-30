package unitcalc

import "testing"

func TestExtraEdges(t *testing.T) {
	mustErr(t, "-(0 C)")
	mustErr(t, "-(-(0 C))")
	r := mustEval(t, "(30 C - 20 C) in dF")
	if r.TargetValue.R.Cmp(rat(t, "18")) != 0 {
		t.Fatal("want 18 dF")
	}
	r = mustEval(t, "1 K - 1 dK")
	if r.BaseKind != "absolute" || r.Base.R.Sign() != 0 {
		t.Fatal("1 K - 1 dK")
	}
	// dC vs dF mixing
	r = mustEval(t, "18 dF in dC")
	if r.TargetValue.R.Cmp(rat(t, "10")) != 0 {
		t.Fatal("18 dF = 10 dC")
	}
	// speed dimensions
	r = mustEval(t, "100 m / 10 s")
	if r.BaseDimText != "L·T⁻¹" {
		t.Fatal(r.BaseDimText)
	}
	// zero division span on whole node
	err := mustErr(t, "(1 m)/(0)")
	if err.Error() != "division by zero" {
		t.Fatal(err)
	}
}
