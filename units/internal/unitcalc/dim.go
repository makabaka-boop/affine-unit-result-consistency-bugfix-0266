package unitcalc

// Dim is a dimensional vector over the four base dimensions:
// length (L), mass (M), time (T), thermodynamic temperature (Θ).
type Dim [4]int

const (
	dimLength = 0
	dimMass   = 1
	dimTime   = 2
	dimTemp   = 3
)

var (
	DimNone      = Dim{}
	DimLengthVec = Dim{1, 0, 0, 0}
	DimMassVec   = Dim{0, 1, 0, 0}
	DimTimeVec   = Dim{0, 0, 1, 0}
	DimTempVec   = Dim{0, 0, 0, 1}
)

func (d Dim) Add(o Dim) Dim {
	return Dim{d[0] + o[0], d[1] + o[1], d[2] + o[2], d[3] + o[3]}
}

func (d Dim) Neg() Dim {
	return Dim{-d[0], -d[1], -d[2], -d[3]}
}

func (d Dim) Equals(o Dim) bool { return d == o }

func (d Dim) Zero() bool { return d == DimNone }

// String renders the dimension like "L²·M·T⁻²"; the empty dimension is "1".
func (d Dim) String() string {
	sup := []string{"⁰", "¹", "²", "³", "⁴", "⁵", "⁶", "⁷", "⁸", "⁹"}
	names := []string{"L", "M", "T", "Θ"}
	out := ""
	for i, n := range d {
		if n == 0 {
			continue
		}
		out += names[i]
		if n != 1 {
			if n < 0 {
				out += "⁻"
				n = -n
			}
			s := ""
			for n > 0 {
				s = sup[n%10] + s
				n /= 10
			}
			out += s
		}
		out += "·"
	}
	if out == "" {
		return "1"
	}
	return out[:len(out)-len("·")]
}
