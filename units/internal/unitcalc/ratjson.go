package unitcalc

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
)

// RatJSON wraps *big.Rat with exact JSON serialization. A rational is emitted
// as an object {"num":..,"den":..,"fraction":"a/b","decimal":"..."} so the
// browser can show the exact value without floating point rounding.
type RatJSON struct {
	R *big.Rat
}

func NewRatJSON(r *big.Rat) RatJSON { return RatJSON{R: new(big.Rat).Set(r)} }

type ratWire struct {
	Num      string `json:"num"`
	Den      string `json:"den"`
	Fraction string `json:"fraction"`
	Decimal  string `json:"decimal"`
}

func (j RatJSON) MarshalJSON() ([]byte, error) {
	if j.R == nil {
		return json.Marshal(nil)
	}
	num := j.R.Num()
	den := j.R.Denom()
	w := ratWire{
		Num:      num.String(),
		Den:      den.String(),
		Fraction: j.R.RatString(),
		Decimal:  decimalExact(num, den),
	}
	return json.Marshal(w)
}

// Bounds for the decimal rendering. The expansion of a rational can be
// arbitrarily long (1/99999989 has a period of ~1e8 digits), so an unbounded
// expansion can hang the service and bloat the response without adding any
// useful information. The rendering below is always bounded:
//
//   - values whose decimal exponent lies in [minPlainExp, maxPlainExp] use
//     plain notation (the same convention as JavaScript's Number#toString),
//     emitting at most maxPlainFrac fraction digits;
//   - anything larger or smaller uses scientific notation with sciDigits
//     significant digits.
//
// Truncated output ends in "…", so a non-zero value never renders as "0"
// and a huge value never renders as "Infinity": the decimal can never
// contradict the exact fraction shown next to it.
const (
	minPlainExp  = -6
	maxPlainExp  = 20
	maxPlainFrac = 100
	sciDigits    = 21
)

var bigTen = big.NewInt(10)

// decimalExact renders num/den in decimal. The result is exact whenever it
// fits the bounds above (terminating expansion, or a repeating one whose
// period fits), and is otherwise an explicitly truncated prefix.
func decimalExact(num, den *big.Int) string {
	sign := ""
	if num.Sign() < 0 {
		sign = "-"
		num = new(big.Int).Neg(num)
	}
	if num.Sign() == 0 {
		return "0"
	}
	exp := decimalExponent(num, den)
	if exp < minPlainExp || exp > maxPlainExp {
		return sign + scientificDecimal(num, den, exp)
	}
	return sign + plainDecimal(num, den)
}

// decimalExponent returns k such that 10^k <= num/den < 10^(k+1), for
// positive num and den.
func decimalExponent(num, den *big.Int) int {
	// k is len(num)-len(den) or one less; disambiguate with one comparison.
	e := len(num.String()) - len(den.String())
	if e >= 0 {
		if num.Cmp(new(big.Int).Mul(den, pow10(e))) < 0 {
			e--
		}
	} else {
		if new(big.Int).Mul(num, pow10(-e)).Cmp(den) < 0 {
			e--
		}
	}
	return e
}

func pow10(n int) *big.Int {
	return new(big.Int).Exp(bigTen, big.NewInt(int64(n)), nil)
}

// plainDecimal renders num/den as "intPart.frac", using parenthesized repeat
// notation for repeating fractions (e.g. 1/6 -> "0.1(6)"). Expansions longer
// than maxPlainFrac fraction digits are truncated with an ellipsis.
func plainDecimal(num, den *big.Int) string {
	intPart := new(big.Int).Quo(num, den)
	rem := new(big.Int).Mod(num, den)
	if rem.Sign() == 0 {
		return intPart.String()
	}
	seen := map[string]int{}
	digits := make([]byte, 0, 64)
	for rem.Sign() != 0 {
		key := rem.String()
		if at, ok := seen[key]; ok {
			return fmt.Sprintf("%s.%s(%s)", intPart,
				string(digits[:at]), string(digits[at:]))
		}
		if len(digits) >= maxPlainFrac {
			return intPart.String() + "." + string(digits) + "…"
		}
		seen[key] = len(digits)
		rem.Mul(rem, bigTen)
		q := new(big.Int).Quo(rem, den)
		rem.Mod(rem, den)
		digits = append(digits, byte('0'+q.Int64()))
	}
	return intPart.String() + "." + string(digits)
}

// scientificDecimal renders num/den as "d.ddd×10ᵉ" with sciDigits significant
// digits. When the expansion terminates inside that window the mantissa is
// exact (trailing zeros trimmed); otherwise it ends in "…".
func scientificDecimal(num, den *big.Int, exp int) string {
	// d holds the first sciDigits significant digits of num/den: since
	// 10^exp <= num/den < 10^(exp+1), scaling by 10^(sciDigits-1-exp) lands
	// the value in [10^(sciDigits-1), 10^sciDigits).
	scale := sciDigits - 1 - exp
	num2, den2 := num, den
	if scale >= 0 {
		num2 = new(big.Int).Mul(num, pow10(scale))
	} else {
		den2 = new(big.Int).Mul(den, pow10(-scale))
	}
	d := new(big.Int)
	rem := new(big.Int)
	d.QuoRem(num2, den2, rem)
	mant := d.String() // exactly sciDigits digits
	frac := mant[1:]
	exact := rem.Sign() == 0
	if exact {
		frac = strings.TrimRight(frac, "0")
	}
	out := mant[:1]
	if frac != "" {
		out += "." + frac
	}
	if !exact {
		out += "…"
	}
	if exp != 0 {
		out += "×10" + superscript(exp)
	}
	return out
}
