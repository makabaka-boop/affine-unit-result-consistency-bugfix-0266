package unitcalc

import (
	"math/big"
	"math/rand"
	"strings"
	"testing"
)

// refValue is the independent reference evaluator's value representation.
// It deliberately does not reuse Value/addValues/mulValues: the typing rules
// and unit factors are re-derived here from scratch.
type refValue struct {
	n    *big.Rat // magnitude in base units
	dim  Dim
	kind Kind
}

type refUnit struct {
	sym    string
	dim    Dim
	kind   Kind
	factor *big.Rat
	offset *big.Rat
}

var refUnits = map[string]refUnit{
	"m":   {"m", DimLengthVec, KindRegular, big.NewRat(1, 1), big.NewRat(0, 1)},
	"cm":  {"cm", DimLengthVec, KindRegular, big.NewRat(1, 100), big.NewRat(0, 1)},
	"kg":  {"kg", DimMassVec, KindRegular, big.NewRat(1, 1), big.NewRat(0, 1)},
	"g":   {"g", DimMassVec, KindRegular, big.NewRat(1, 1000), big.NewRat(0, 1)},
	"s":   {"s", DimTimeVec, KindRegular, big.NewRat(1, 1), big.NewRat(0, 1)},
	"min": {"min", DimTimeVec, KindRegular, big.NewRat(60, 1), big.NewRat(0, 1)},
	"K":   {"K", DimTempVec, KindAbsolute, big.NewRat(1, 1), big.NewRat(0, 1)},
	// 273.15 = 5463/20
	"C":  {"C", DimTempVec, KindAbsolute, big.NewRat(1, 1), big.NewRat(5463, 20)},
	"dK": {"dK", DimTempVec, KindDelta, big.NewRat(1, 1), big.NewRat(0, 1)},
	"dC": {"dC", DimTempVec, KindDelta, big.NewRat(1, 1), big.NewRat(0, 1)},
	// Fahrenheit: factor 5/9, offset -32*5/9 + 5463/20 = 229835/900 = 45967/180
	"F":  {"F", DimTempVec, KindAbsolute, big.NewRat(5, 9), big.NewRat(45967, 180)},
	"dF": {"dF", DimTempVec, KindDelta, big.NewRat(5, 9), big.NewRat(0, 1)},
}

func refFromLiteral(r *big.Rat, sym string) refValue {
	if sym == "" {
		return refValue{new(big.Rat).Set(r), DimNone, KindRegular}
	}
	u := refUnits[sym]
	n := new(big.Rat).Mul(r, u.factor)
	if u.kind == KindAbsolute {
		n.Add(n, u.offset)
	}
	return refValue{n, u.dim, u.kind}
}

// refEval re-implements the type rules. Returns ok=false for any illegal op.
func refEval(node Node) (refValue, bool) {
	switch n := node.(type) {
	case *NumberNode:
		return refFromLiteral(n.Num, n.UnitSymbol), true
	case *UnaryNode:
		c, ok := refEval(n.Child)
		if !ok {
			return refValue{}, false
		}
		if c.kind == KindAbsolute {
			return refValue{}, false
		}
		return refValue{new(big.Rat).Neg(c.n), c.dim, c.kind}, true
	case *BinaryNode:
		l, ok := refEval(n.Left)
		if !ok {
			return refValue{}, false
		}
		r, ok := refEval(n.Right)
		if !ok {
			return refValue{}, false
		}
		return refApply(n.Op, l, r)
	}
	return refValue{}, false
}

func refApply(op string, a, b refValue) (refValue, bool) {
	switch op {
	case "+":
		if a.kind == KindAbsolute || b.kind == KindAbsolute {
			// abs+abs illegal; abs+delta (either order) -> abs
			if a.kind == b.kind {
				return refValue{}, false
			}
			return refValue{new(big.Rat).Add(a.n, b.n), DimTempVec, KindAbsolute}, true
		}
		if a.kind != b.kind || a.dim != b.dim {
			return refValue{}, false
		}
		return refValue{new(big.Rat).Add(a.n, b.n), a.dim, a.kind}, true
	case "-":
		switch {
		case a.kind == KindAbsolute && b.kind == KindAbsolute:
			return refValue{new(big.Rat).Sub(a.n, b.n), DimTempVec, KindDelta}, true
		case a.kind == KindAbsolute && b.kind == KindDelta:
			return refValue{new(big.Rat).Sub(a.n, b.n), DimTempVec, KindAbsolute}, true
		case a.kind == KindDelta && b.kind == KindAbsolute:
			return refValue{}, false
		default:
			if a.kind != b.kind || a.dim != b.dim {
				return refValue{}, false
			}
			return refValue{new(big.Rat).Sub(a.n, b.n), a.dim, a.kind}, true
		}
	case "*":
		if a.kind == KindAbsolute || b.kind == KindAbsolute {
			return refValue{}, false
		}
		return refValue{new(big.Rat).Mul(a.n, b.n), a.dim.Add(b.dim), KindRegular}, true
	case "/":
		if a.kind == KindAbsolute || b.kind == KindAbsolute || b.n.Sign() == 0 {
			return refValue{}, false
		}
		return refValue{new(big.Rat).Quo(a.n, b.n), a.dim.Add(b.dim.Neg()), KindRegular}, true
	}
	return refValue{}, false
}

// ---- random small tree generation ----

var genUnits = []string{"", "m", "cm", "kg", "g", "s", "min", "K", "C", "F", "dK", "dC", "dF"}
var genOps = []string{"+", "-", "*", "/"}

func genLiteral(rng *rand.Rand) *NumberNode {
	num := int64(rng.Intn(5)) // 0..4 keeps denominators small too
	var r *big.Rat
	if rng.Intn(3) == 0 {
		den := int64(1 + rng.Intn(5))
		r = big.NewRat(num, den)
	} else {
		r = big.NewRat(num, 1)
	}
	sym := genUnits[rng.Intn(len(genUnits))]
	return &NumberNode{Num: r, UnitSymbol: sym, Start: 0, End: 0}
}

func genTree(rng *rand.Rand, depth int) Node {
	if depth <= 0 || rng.Intn(3) == 0 {
		return genLiteral(rng)
	}
	if rng.Intn(6) == 0 {
		return &UnaryNode{Op: "-", Child: genTree(rng, depth-1)}
	}
	op := genOps[rng.Intn(len(genOps))]
	return &BinaryNode{
		Op:    op,
		Left:  genTree(rng, depth-1),
		Right: genTree(rng, depth-1),
	}
}

// render turns a generated tree back into parseable source text.
func render(node Node) string {
	switch n := node.(type) {
	case *NumberNode:
		s := n.Num.RatString()
		if n.UnitSymbol != "" {
			s += " " + n.UnitSymbol
		}
		return s
	case *UnaryNode:
		return "-(" + render(n.Child) + ")"
	case *BinaryNode:
		return "(" + render(n.Left) + " " + n.Op + " " + render(n.Right) + ")"
	case *ConvertNode:
		return render(n.Inner) + " in " + n.Target.Symbol
	}
	return ""
}

// respan reassigns spans on a generated tree (roundtripped trees get real
// spans from the parser; this is only used for the direct EvalNode path).
func respan(node Node, src string, base int) {}

func TestDifferentialAgainstReference(t *testing.T) {
	rng := rand.New(rand.NewSource(42))
	const iterations = 3000
	checked := 0
	matched := 0
	for i := 0; i < iterations; i++ {
		tree := genTree(rng, 2+rng.Intn(3)) // depth 2..4
		want, wantOK := refEval(tree)

		// path 1: evaluate the generated tree directly
		directSrc := render(tree)
		gotRes, gotErr := EvalNode(directSrc, tree)
		gotOK := gotErr == nil
		if gotOK != wantOK {
			t.Fatalf("tree %q: direct ok=%v (%v), reference ok=%v",
				directSrc, gotOK, gotErr, wantOK)
		}

		// path 2: render -> lex/parse -> evaluate (parser/evaluator consistency)
		parsed, perr := Parse(directSrc)
		if perr != nil {
			t.Fatalf("tree %q: re-parse failed: %v", directSrc, perr)
		}
		rtRes, rtErr := EvalNode(directSrc, parsed)
		if (rtErr == nil) != wantOK {
			t.Fatalf("tree %q: roundtrip ok=%v (%v), reference ok=%v",
				directSrc, rtErr == nil, rtErr, wantOK)
		}

		if !wantOK {
			checked++
			continue
		}
		if gotRes.Base.R.Cmp(want.n) != 0 ||
			gotRes.BaseKind != want.kind.String() ||
			gotRes.BaseDim != want.dim {
			t.Fatalf("tree %q:\n direct  = %s [%s] %s\n reference = %s [%s] %s",
				directSrc,
				gotRes.Base.R.RatString(), gotRes.BaseDimText, gotRes.BaseKind,
				want.n.RatString(), want.dim.String(), want.kind)
		}
		if rtRes.Base.R.Cmp(want.n) != 0 ||
			rtRes.BaseKind != want.kind.String() ||
			rtRes.BaseDim != want.dim {
			t.Fatalf("tree %q:\n roundtrip = %s [%s] %s\n reference = %s [%s] %s",
				directSrc,
				rtRes.Base.R.RatString(), rtRes.BaseDimText, rtRes.BaseKind,
				want.n.RatString(), want.dim.String(), want.kind)
		}
		matched++
		checked++
	}
	t.Logf("differential: %d trees checked, %d produced values, %d rejected",
		checked, matched, checked-matched)
	// sanity: make sure the generator actually exercised both outcomes
	if matched == 0 || matched == checked {
		t.Fatalf("generator did not cover both success (%d) and failure (%d) cases",
			matched, checked-matched)
	}
}

// TestGeneratedTreeSpansAreMinimal checks that, on successful re-parses,
// every binary/unary node's span actually covers its rendered text.
func TestGeneratedTreeSpansCoverSource(t *testing.T) {
	rng := rand.New(rand.NewSource(7))
	for i := 0; i < 500; i++ {
		tree := genTree(rng, 3)
		src := render(tree)
		parsed, perr := Parse(src)
		if perr != nil {
			continue
		}
		var walk func(Node)
		walk = func(n Node) {
			switch x := n.(type) {
			case *BinaryNode:
				walk(x.Left)
				walk(x.Right)
			case *UnaryNode:
				walk(x.Child)
			case *ConvertNode:
				walk(x.Inner)
			}
			s, e := n.Span()
			got := Slice(src, s, e)
			if strings.TrimSpace(got) == "" {
				t.Errorf("node %T in %q has empty span [%d,%d)", n, src, s, e)
			}
		}
		walk(parsed)
	}
}
