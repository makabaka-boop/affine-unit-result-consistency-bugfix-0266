package unitcalc

import (
	"math/big"
	"sort"
)

// Step is one recorded derivation step (one syntax node), emitted in
// post-order: operands precede the operation that consumes them.
type Step struct {
	NodeType string  `json:"nodeType"`
	Op       string  `json:"op,omitempty"`
	SrcStart int     `json:"srcStart"`
	SrcEnd   int     `json:"srcEnd"`
	Source   string  `json:"source"`
	Kind     string  `json:"kind"`
	Dim      Dim     `json:"dim"`
	DimText  string  `json:"dimText"`
	Value    RatJSON `json:"value"` // magnitude in canonical base units
	UnitText string  `json:"unitText"`
}

// Result is the full evaluation answer.
type Result struct {
	Base          RatJSON  `json:"base"` // canonical (base-unit) value
	BaseKind      string   `json:"baseKind"`
	BaseDim       Dim      `json:"baseDim"`
	BaseDimText   string   `json:"baseDimText"`
	CanonicalUnit string   `json:"canonicalUnit"`
	Display       string   `json:"display"` // exact human-readable base value
	TargetUnit    string   `json:"targetUnit,omitempty"`
	TargetValue   *RatJSON `json:"targetValue,omitempty"`
	TargetText    string   `json:"targetText,omitempty"`
	Steps         []Step   `json:"steps"`
}

type evalState struct {
	src   string
	steps []Step
}

// Evaluate parses (optionally via an existing AST), type-checks and computes.
func Evaluate(src string) (*Result, error) {
	node, err := Parse(src)
	if err != nil {
		return nil, err
	}
	return EvalNode(src, node)
}

// EvalNode type-checks and computes an already-parsed tree.
func EvalNode(src string, node Node) (*Result, error) {
	st := &evalState{src: src}
	v, err := st.eval(node)
	if err != nil {
		return nil, err
	}
	res := &Result{
		Base:        RatJSON{R: new(big.Rat).Set(v.N)},
		BaseKind:    v.Kind.String(),
		BaseDim:     v.Dim,
		BaseDimText: v.Dim.String(),
		Steps:       st.steps,
	}
	res.CanonicalUnit = canonicalUnit(v)
	res.Display = formatBase(v, res.CanonicalUnit)

	// apply target unit if the tree ends in a conversion
	if cn, ok := node.(*ConvertNode); ok {
		tv, terr := convertTo(v, cn)
		if terr != nil {
			return nil, terr
		}
		res.TargetUnit = cn.Target.Symbol
		res.TargetValue = &RatJSON{R: tv}
		res.TargetText = tv.RatString() + " " + cn.Target.Symbol
	}
	return res, nil
}

func (st *evalState) eval(node Node) (Value, *TypedOpError) {
	s, e := node.Span()
	source := Slice(st.src, s, e)
	switch n := node.(type) {
	case *NumberNode:
		if n.UnitSymbol == "" {
			v := newValue(n.Num, DimNone, KindRegular)
			st.record(s, e, "number", "", source, v)
			return v, nil
		}
		u, _ := LookupUnit(n.UnitSymbol)
		v := FromNumeric(n.Num, u)
		st.record(s, e, "number", "", source, v)
		return v, nil

	case *UnaryNode:
		child, err := st.eval(n.Child)
		if err != nil {
			return Value{}, err
		}
		v, verr := negValue(child, s, e)
		if verr != nil {
			return Value{}, verr
		}
		st.record(s, e, "unary", n.Op, source, v)
		return v, nil

	case *BinaryNode:
		lv, err := st.eval(n.Left)
		if err != nil {
			return Value{}, err
		}
		rv, err := st.eval(n.Right)
		if err != nil {
			return Value{}, err
		}
		var v Value
		var verr *TypedOpError
		switch n.Op {
		case "+":
			v, verr = addValues(lv, rv, s, e)
		case "-":
			v, verr = subValues(lv, rv, s, e)
		case "*":
			v, verr = mulValues(lv, rv, s, e)
		case "/":
			v, verr = divValues(lv, rv, s, e)
		}
		if verr != nil {
			return Value{}, verr
		}
		st.record(s, e, "binary", n.Op, source, v)
		return v, nil

	case *ConvertNode:
		v, err := st.eval(n.Inner)
		if err != nil {
			return Value{}, err
		}
		if _, cerr := convertTo(v, n); cerr != nil {
			return Value{}, cerr
		}
		// the conversion itself is the outer step; value in base units
		st.record(s, e, "convert", "in", source, v)
		return v, nil
	}
	return Value{}, opErr(s, e, "unknown node type")
}

func convertTo(v Value, n *ConvertNode) (*big.Rat, *TypedOpError) {
	s, e := n.Span()
	t := n.Target
	if !v.Dim.Equals(t.Dim) {
		return nil, opErr(s, e, "target unit %q has dimension %s but the expression has dimension %s",
			t.Symbol, t.Dim, v.Dim)
	}
	switch {
	case v.Kind == KindAbsolute && t.Kind != KindAbsolute:
		return nil, opErr(s, e, "cannot express an absolute temperature in the difference unit %q; use %s",
			t.Symbol, plainCounterpart(t))
	case v.Kind == KindDelta && t.Kind != KindDelta:
		return nil, opErr(s, e, "cannot express a temperature difference in the absolute scale %q; use %s",
			t.Symbol, "d"+t.Symbol)
	case v.Kind == KindRegular && (t.Kind == KindAbsolute || t.Kind == KindDelta):
		return nil, opErr(s, e, "cannot express a non-temperature quantity in temperature unit %q", t.Symbol)
	}
	return v.ToNumeric(t), nil
}

func plainCounterpart(t *Unit) string {
	switch t.Symbol {
	case "dK":
		return "K"
	case "dC":
		return "C"
	case "dF":
		return "F"
	}
	return t.Symbol
}

func (st *evalState) record(start, end int, nt, op, source string, v Value) {
	st.steps = append(st.steps, Step{
		NodeType: nt,
		Op:       op,
		SrcStart: start,
		SrcEnd:   end,
		Source:   source,
		Kind:     v.Kind.String(),
		Dim:      v.Dim,
		DimText:  v.Dim.String(),
		Value:    RatJSON{R: new(big.Rat).Set(v.N)},
		UnitText: canonicalUnit(v),
	})
}

// canonicalUnit chooses the base unit for a value's dimensions/kind.
func canonicalUnit(v Value) string {
	if v.Kind == KindAbsolute {
		return "K"
	}
	if v.Kind == KindDelta {
		return "dK"
	}
	bases := []string{"m", "kg", "s", "K"}
	out := ""
	for i, p := range v.Dim {
		if p == 0 {
			continue
		}
		out += bases[i]
		if p != 1 {
			out += superscript(p)
		}
		out += "·"
	}
	if out == "" {
		return ""
	}
	return out[:len(out)-len("·")]
}

func superscript(n int) string {
	if n < 0 {
		return "⁻" + superscript(-n)
	}
	digits := []string{"⁰", "¹", "²", "³", "⁴", "⁵", "⁶", "⁷", "⁸", "⁹"}
	s := ""
	for n > 0 {
		s = digits[n%10] + s
		n /= 10
	}
	return s
}

func formatBase(v Value, unit string) string {
	s := v.N.RatString()
	if unit != "" {
		s += " " + unit
	}
	return s
}

var _ = sort.Ints
