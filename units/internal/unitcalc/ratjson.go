package unitcalc

import (
	"encoding/json"
	"fmt"
	"math/big"
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

// decimalExact returns an exact decimal expansion, using parenthesized repeat
// notation for repeating fractions (e.g. 1/6 -> "0.1(6)").
func decimalExact(num, den *big.Int) string {
	sign := ""
	if num.Sign() < 0 {
		sign = "-"
		num = new(big.Int).Neg(num)
	}
	intPart := new(big.Int).Quo(num, den)
	rem := new(big.Int).Mod(num, den)
	if rem.Sign() == 0 {
		return sign + intPart.String()
	}
	seen := map[string]int{}
	digits := make([]byte, 0, 24)
	pos := 0
	repeatStart := -1
	for rem.Sign() != 0 {
		key := rem.String()
		if at, ok := seen[key]; ok {
			repeatStart = at
			break
		}
		seen[key] = pos
		rem.Mul(rem, big.NewInt(10))
		q := new(big.Int).Quo(rem, den)
		rem.Mod(rem, den)
		digits = append(digits, byte('0'+q.Int64()))
		pos++
	}
	if repeatStart >= 0 {
		return fmt.Sprintf("%s%s.%s(%s)", sign, intPart.String(),
			string(digits[:repeatStart]), string(digits[repeatStart:]))
	}
	return sign + intPart.String() + "." + string(digits)
}
