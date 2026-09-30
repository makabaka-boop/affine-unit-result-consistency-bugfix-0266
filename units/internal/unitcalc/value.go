package unitcalc

import (
	"fmt"
	"math/big"
)

// Value is a typed physical quantity. N holds the magnitude expressed in base
// units (m, kg, s, K); Dim is the dimensional vector; Kind distinguishes
// absolute temperatures from temperature intervals from regular quantities.
type Value struct {
	N    *big.Rat
	Dim  Dim
	Kind Kind
}

func newValue(n *big.Rat, d Dim, k Kind) Value {
	return Value{N: new(big.Rat).Set(n), Dim: d, Kind: k}
}

// FromNumeric turns a literal amount in unit u into a base-unit Value.
func FromNumeric(n *big.Rat, u *Unit) Value {
	v := Value{N: new(big.Rat).Mul(n, u.Factor), Dim: u.Dim, Kind: u.Kind}
	if u.Kind == KindAbsolute {
		v.N.Add(v.N, u.Offset) // K = n*factor + offset
	}
	return v
}

// ToNumeric converts a base-unit Value back to an amount measured in u.
func (v Value) ToNumeric(u *Unit) *big.Rat {
	r := new(big.Rat).Set(v.N)
	if v.Kind == KindAbsolute {
		r.Sub(r, u.Offset)
	}
	r.Quo(r, u.Factor)
	return r
}

// TypedOpError describes why an operation is illegal, pinned to an interval.
type TypedOpError struct {
	Start int
	End   int
	Msg   string
}

func (e *TypedOpError) Error() string { return e.Msg }

func opErr(start, end int, format string, args ...any) *TypedOpError {
	return &TypedOpError{Start: start, End: end, Msg: fmt.Sprintf(format, args...)}
}

// addValues implements the addition typing rules.
//
//	absolute + delta (either order) -> absolute
//	absolute + absolute            -> illegal
//	regular/regular, delta/delta   -> require equal dimensions
func addValues(a, b Value, start, end int) (Value, *TypedOpError) {
	if a.Kind == KindAbsolute || b.Kind == KindAbsolute {
		if a.Kind == KindAbsolute && b.Kind == KindAbsolute {
			return Value{}, opErr(start, end, "absolute temperatures cannot be added together; subtract them to get a difference")
		}
		// one is absolute, the other must be a delta (both carry DimTemp)
		return Value{N: new(big.Rat).Add(a.N, b.N), Dim: DimTempVec, Kind: KindAbsolute}, nil
	}
	if a.Kind != b.Kind {
		return Value{}, opErr(start, end, "cannot add a temperature difference to a non-temperature quantity")
	}
	if !a.Dim.Equals(b.Dim) {
		return Value{}, opErr(start, end, "cannot add quantities with dimensions %s and %s", a.Dim, b.Dim)
	}
	return Value{N: new(big.Rat).Add(a.N, b.N), Dim: a.Dim, Kind: a.Kind}, nil
}

// subValues implements the subtraction typing rules.
//
//	absolute - absolute -> delta
//	absolute - delta    -> absolute
//	delta    - absolute -> illegal
//	otherwise           -> same kind & dimension required
func subValues(a, b Value, start, end int) (Value, *TypedOpError) {
	switch {
	case a.Kind == KindAbsolute && b.Kind == KindAbsolute:
		return Value{N: new(big.Rat).Sub(a.N, b.N), Dim: DimTempVec, Kind: KindDelta}, nil
	case a.Kind == KindAbsolute && b.Kind == KindDelta:
		return Value{N: new(big.Rat).Sub(a.N, b.N), Dim: DimTempVec, Kind: KindAbsolute}, nil
	case a.Kind == KindDelta && b.Kind == KindAbsolute:
		return Value{}, opErr(start, end, "a temperature difference minus an absolute temperature is meaningless")
	case a.Kind == KindRegular && b.Kind == KindRegular:
		if !a.Dim.Equals(b.Dim) {
			return Value{}, opErr(start, end, "cannot subtract quantities with dimensions %s and %s", a.Dim, b.Dim)
		}
		return Value{N: new(big.Rat).Sub(a.N, b.N), Dim: a.Dim, Kind: KindRegular}, nil
	case a.Kind == KindDelta && b.Kind == KindDelta:
		if !a.Dim.Equals(b.Dim) {
			return Value{}, opErr(start, end, "cannot subtract temperature differences of different dimensions")
		}
		return Value{N: new(big.Rat).Sub(a.N, b.N), Dim: DimTempVec, Kind: KindDelta}, nil
	default:
		return Value{}, opErr(start, end, "cannot subtract a temperature difference from a non-temperature quantity")
	}
}

// mulValues: absolute temperatures may never be multiplied. Delta temperature
// behaves like a regular one-dimensional quantity; the result is regular.
func mulValues(a, b Value, start, end int) (Value, *TypedOpError) {
	if a.Kind == KindAbsolute || b.Kind == KindAbsolute {
		return Value{}, opErr(start, end, "absolute temperatures cannot be multiplied")
	}
	return Value{
		N:    new(big.Rat).Mul(a.N, b.N),
		Dim:  a.Dim.Add(b.Dim),
		Kind: KindRegular,
	}, nil
}

// divValues: absolute temperatures may never be divided; divisor must be
// non-zero.
func divValues(a, b Value, start, end int) (Value, *TypedOpError) {
	if a.Kind == KindAbsolute || b.Kind == KindAbsolute {
		return Value{}, opErr(start, end, "absolute temperatures cannot be divided")
	}
	if b.N.Sign() == 0 {
		return Value{}, opErr(start, end, "division by zero")
	}
	return Value{
		N:    new(big.Rat).Quo(a.N, b.N),
		Dim:  a.Dim.Add(b.Dim.Neg()),
		Kind: KindRegular,
	}, nil
}

// negValue: legal for regular quantities and intervals, never for points.
func negValue(a Value, start, end int) (Value, *TypedOpError) {
	if a.Kind == KindAbsolute {
		return Value{}, opErr(start, end, "an absolute temperature cannot be negated")
	}
	return Value{N: new(big.Rat).Neg(a.N), Dim: a.Dim, Kind: a.Kind}, nil
}
