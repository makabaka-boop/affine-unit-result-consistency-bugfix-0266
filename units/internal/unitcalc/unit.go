package unitcalc

import "math/big"

// Kind classifies the role of a temperature quantity.
//
//   - KindRegular: any non-temperature quantity (and dimensionless numbers)
//   - KindAbsolute: an absolute temperature (a point on a scale): K, C, F
//   - KindDelta: a temperature difference (interval): dK, dC, dF
type Kind int

const (
	KindRegular Kind = iota
	KindAbsolute
	KindDelta
)

func (k Kind) String() string {
	switch k {
	case KindAbsolute:
		return "absolute"
	case KindDelta:
		return "delta"
	default:
		return "regular"
	}
}

// Unit is a measurement unit.
//
// Every value is internally represented in base units (m, kg, s, K). For a
// non-temperature unit the conversion is purely linear:
//
//	base = numeric * Factor
//
// For absolute temperatures an offset also applies:
//
//	K = numeric * Factor + Offset
//
// Temperature differences use the factor only (Offset is zero).
type Unit struct {
	Symbol string
	Dim    Dim
	Kind   Kind
	Factor *big.Rat // one of this unit, expressed in the matching base unit
	Offset *big.Rat // absolute-scale origin, in kelvin (absolute kinds only)
}

func newLinearUnit(sym string, d Dim, num, den int64) *Unit {
	return &Unit{
		Symbol: sym, Dim: d, Kind: KindRegular,
		Factor: big.NewRat(num, den), Offset: new(big.Rat),
	}
}

var registry = buildRegistry()

func buildRegistry() map[string]*Unit {
	m := map[string]*Unit{}

	put := func(u *Unit) { m[u.Symbol] = u }

	// length: base metre
	put(newLinearUnit("m", DimLengthVec, 1, 1))
	put(newLinearUnit("cm", DimLengthVec, 1, 100))

	// mass: base kilogram
	put(newLinearUnit("kg", DimMassVec, 1, 1))
	put(newLinearUnit("g", DimMassVec, 1, 1000))

	// time: base second
	put(newLinearUnit("s", DimTimeVec, 1, 1))
	put(newLinearUnit("min", DimTimeVec, 60, 1))

	// absolute temperature scales, base kelvin.
	abs := func(sym string, factorNum, factorDen, offNum, offDen int64) *Unit {
		return &Unit{
			Symbol: sym, Dim: DimTempVec, Kind: KindAbsolute,
			Factor: big.NewRat(factorNum, factorDen),
			Offset: big.NewRat(offNum, offDen),
		}
	}
	put(abs("K", 1, 1, 0, 1))       // K
	put(abs("C", 1, 1, 27315, 100)) // K = C + 27315/100
	// K = (F - 32) * 5/9 + 27315/100 = F*5/9 + 229835/900
	put(abs("F", 5, 9, 229835, 900))

	// temperature differences (interval units): factor only.
	delta := func(sym string, num, den int64) *Unit {
		return &Unit{
			Symbol: sym, Dim: DimTempVec, Kind: KindDelta,
			Factor: big.NewRat(num, den), Offset: new(big.Rat),
		}
	}
	put(delta("dK", 1, 1)) // a kelvin interval
	put(delta("dC", 1, 1)) // a Celsius interval equals a kelvin interval
	put(delta("dF", 5, 9)) // a Fahrenheit interval is 5/9 kelvin

	return m
}

// LookupUnit returns the registered unit with the given symbol.
func LookupUnit(sym string) (*Unit, bool) {
	u, ok := registry[sym]
	return u, ok
}

// UnitSymbols lists every supported unit symbol.
func UnitSymbols() []string {
	return []string{"m", "cm", "kg", "g", "s", "min", "K", "C", "F", "dK", "dC", "dF"}
}
