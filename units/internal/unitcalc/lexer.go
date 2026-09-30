package unitcalc

import (
	"fmt"
	"math/big"
	"strings"
	"unicode"
	"unicode/utf8"
)

type tokenKind int

const (
	tEOF tokenKind = iota
	tNumber
	tUnit
	tIn
	tPlus
	tMinus
	tStar
	tSlash
	tLParen
	tRParen
)

type token struct {
	kind  tokenKind
	start int // rune index
	end   int // rune index (exclusive)
	text  string
	rat   *big.Rat // tNumber
	unit  string   // tUnit: canonical symbol, e.g. "C" for "°C"
}

// SyntaxError is a parse/lex error pinned to a minimal source interval.
// Span offsets are rune (Unicode code point) offsets, matching JavaScript
// string indices for the BMP characters the grammar accepts.
type SyntaxError struct {
	Start int
	End   int
	Msg   string
}

func (e *SyntaxError) Error() string { return e.Msg }

func synErr(start, end int, format string, args ...any) *SyntaxError {
	return &SyntaxError{Start: start, End: end, Msg: fmt.Sprintf(format, args...)}
}

func lex(src string) ([]token, *SyntaxError) {
	runes := []rune(src)
	toks := []token{}
	i := 0
	n := len(runes)

	// matchUnit tries to read a unit at rune position pos, tolerating a
	// leading degree sign. Returns the canonical symbol and rune length.
	matchUnit := func(pos int) (sym string, length int) {
		body := pos
		if body < n && runes[body] == '°' {
			body++
		}
		rest := string(runes[body:])
		best := ""
		for _, u := range UnitSymbols() {
			if strings.HasPrefix(rest, u) && len(u) > len(best) {
				best = u
			}
		}
		if best == "" {
			return "", 0
		}
		return best, (body - pos) + len([]rune(best))
	}

	for i < n {
		c := runes[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '+':
			toks = append(toks, token{kind: tPlus, start: i, end: i + 1, text: "+"})
			i++
		case c == '-':
			toks = append(toks, token{kind: tMinus, start: i, end: i + 1, text: "-"})
			i++
		case c == '*':
			toks = append(toks, token{kind: tStar, start: i, end: i + 1, text: "*"})
			i++
		case c == '/':
			toks = append(toks, token{kind: tSlash, start: i, end: i + 1, text: "/"})
			i++
		case c == '(':
			toks = append(toks, token{kind: tLParen, start: i, end: i + 1, text: "("})
			i++
		case c == ')':
			toks = append(toks, token{kind: tRParen, start: i, end: i + 1, text: ")"})
			i++
		case c >= '0' && c <= '9':
			start := i
			for i < n && runes[i] >= '0' && runes[i] <= '9' {
				i++
			}
			if i < n && runes[i] == '.' && i+1 < n && runes[i+1] >= '0' && runes[i+1] <= '9' {
				i++
				for i < n && runes[i] >= '0' && runes[i] <= '9' {
					i++
				}
			}
			// Rational literal "a/b": slash must be directly adjacent to
			// digits on both sides (no whitespace).
			if i < n && runes[i] == '/' && i+1 < n && runes[i+1] >= '0' && runes[i+1] <= '9' {
				numText := string(runes[start:i])
				i++ // consume '/'
				denStart := i
				for i < n && runes[i] >= '0' && runes[i] <= '9' {
					i++
				}
				denText := string(runes[denStart:i])
				r, ok := new(big.Rat).SetString(numText + "/" + denText)
				if !ok {
					return nil, synErr(start, i, "invalid rational literal")
				}
				toks = append(toks, token{kind: tNumber, start: start, end: i,
					text: string(runes[start:i]), rat: r})
				continue
			}
			r, ok := new(big.Rat).SetString(string(runes[start:i]))
			if !ok {
				return nil, synErr(start, i, "invalid number %q", string(runes[start:i]))
			}
			toks = append(toks, token{kind: tNumber, start: start, end: i,
				text: string(runes[start:i]), rat: r})
		case unicode.IsLetter(c) || c == '°':
			start := i
			if strings.HasPrefix(string(runes[i:]), "in") && wordEndsAt(runes, i+2) {
				toks = append(toks, token{kind: tIn, start: i, end: i + 2, text: "in"})
				i += 2
				continue
			}
			if strings.HasPrefix(string(runes[i:]), "to") && wordEndsAt(runes, i+2) {
				toks = append(toks, token{kind: tIn, start: i, end: i + 2, text: "to"})
				i += 2
				continue
			}
			sym, length := matchUnit(i)
			if sym == "" {
				j := i
				for j < n && isIdentRune(runes[j]) {
					j++
				}
				return nil, synErr(start, j, "unknown unit or identifier %q", string(runes[start:j]))
			}
			end := i + length
			toks = append(toks, token{kind: tUnit, start: i, end: end,
				text: string(runes[i:end]), unit: sym})
			i = end
		default:
			if !utf8.ValidRune(c) {
				return nil, synErr(i, i+1, "invalid UTF-8 in expression")
			}
			return nil, synErr(i, i+1, "unexpected character %q", string(c))
		}
	}
	toks = append(toks, token{kind: tEOF, start: n, end: n})
	return toks, nil
}

func wordEndsAt(runes []rune, pos int) bool {
	return pos >= len(runes) || !isIdentRune(runes[pos])
}

func isIdentRune(r rune) bool {
	return unicode.IsLetter(r) || (r >= '0' && r <= '9') || r == '°'
}
