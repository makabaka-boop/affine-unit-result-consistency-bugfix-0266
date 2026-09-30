package unitcalc

// RuneLen returns the length of src in runes (the index space used by
// syntax/type error spans and step spans).
func RuneLen(src string) int { return len([]rune(src)) }

// Slice returns the source sub-interval [start, end) in rune index space.
func Slice(src string, start, end int) string {
	r := []rune(src)
	if start < 0 {
		start = 0
	}
	if end > len(r) {
		end = len(r)
	}
	if start > end {
		return ""
	}
	return string(r[start:end])
}
