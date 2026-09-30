package unitcalc

import (
	"math/big"
	"strings"
)

// Node is one node of the parsed expression tree.
type Node interface {
	Span() (int, int)
	nodeKind() string
}

// NumberNode is a rational literal with an optional unit suffix.
type NumberNode struct {
	Num        *big.Rat
	UnitSymbol string // "" means dimensionless
	RawUnit    string // source text of the unit ("°C" etc.)
	Start, End int
}

// UnaryNode is a unary minus.
type UnaryNode struct {
	Op         string // "-"
	Child      Node
	Start, End int
}

// BinaryNode is one of + - * /.
type BinaryNode struct {
	Op          string
	Left, Right Node
	Start, End  int
}

// ConvertNode is "expr in <target unit>".
type ConvertNode struct {
	Inner      Node
	Target     *Unit
	TargetRaw  string
	Start, End int
}

func (n *NumberNode) Span() (int, int)  { return n.Start, n.End }
func (n *UnaryNode) Span() (int, int)   { return n.Start, n.End }
func (n *BinaryNode) Span() (int, int)  { return n.Start, n.End }
func (n *ConvertNode) Span() (int, int) { return n.Start, n.End }

func (*NumberNode) nodeKind() string  { return "number" }
func (*UnaryNode) nodeKind() string   { return "unary" }
func (*BinaryNode) nodeKind() string  { return "binary" }
func (*ConvertNode) nodeKind() string { return "convert" }

type parser struct {
	src  string
	toks []token
	pos  int
}

// Parse builds the AST of src. A trailing "in <unit>" (also "to <unit>")
// selects the target unit for the result.
func Parse(src string) (Node, *SyntaxError) {
	toks, err := lex(src)
	if err != nil {
		return nil, err
	}
	p := &parser{src: src, toks: toks}
	node, perr := p.parseExpr()
	if perr != nil {
		return nil, perr
	}
	if p.peek().kind != tEOF {
		t := p.peek()
		return nil, synErr(t.start, t.end, "unexpected token %q", t.text)
	}
	return node, nil
}

func (p *parser) peek() token { return p.toks[p.pos] }

func (p *parser) advance() token {
	t := p.toks[p.pos]
	if p.pos < len(p.toks)-1 {
		p.pos++
	}
	return t
}

func (p *parser) parseExpr() (Node, *SyntaxError) {
	left, err := p.parseAdd()
	if err != nil {
		return nil, err
	}
	if p.peek().kind == tIn {
		inTok := p.advance()
		if p.peek().kind != tUnit {
			t := p.peek()
			return nil, synErr(inTok.start, t.end, `"in" must be followed by a target unit`)
		}
		uTok := p.advance()
		u, _ := LookupUnit(uTok.unit)
		start, _ := left.Span()
		return &ConvertNode{
			Inner: left, Target: u, TargetRaw: uTok.text,
			Start: start, End: uTok.end,
		}, nil
	}
	return left, nil
}

func (p *parser) parseAdd() (Node, *SyntaxError) {
	left, err := p.parseMul()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tPlus || p.peek().kind == tMinus {
		op := p.advance()
		right, rerr := p.parseMul()
		if rerr != nil {
			return nil, rerr
		}
		_, end := right.Span()
		start, _ := left.Span()
		left = &BinaryNode{Op: op.text, Left: left, Right: right, Start: start, End: end}
	}
	return left, nil
}

func (p *parser) parseMul() (Node, *SyntaxError) {
	left, err := p.parseUnary()
	if err != nil {
		return nil, err
	}
	for p.peek().kind == tStar || p.peek().kind == tSlash {
		op := p.advance()
		right, rerr := p.parseUnary()
		if rerr != nil {
			return nil, rerr
		}
		start, _ := left.Span()
		_, end := right.Span()
		left = &BinaryNode{Op: op.text, Left: left, Right: right, Start: start, End: end}
	}
	return left, nil
}

func (p *parser) parseUnary() (Node, *SyntaxError) {
	if p.peek().kind == tMinus {
		op := p.advance()
		child, err := p.parseUnary()
		if err != nil {
			return nil, err
		}
		_, end := child.Span()
		return &UnaryNode{Op: "-", Child: child, Start: op.start, End: end}, nil
	}
	if p.peek().kind == tPlus {
		// unary plus is accepted but produces no node
		p.advance()
		return p.parseUnary()
	}
	return p.parsePrimary()
}

func (p *parser) parsePrimary() (Node, *SyntaxError) {
	t := p.peek()
	switch t.kind {
	case tNumber:
		p.advance()
		if p.peek().kind == tUnit {
			ut := p.advance()
			u, _ := LookupUnit(ut.unit)
			if u == nil {
				return nil, synErr(ut.start, ut.end, "unknown unit %q", ut.text)
			}
			return &NumberNode{Num: t.rat, UnitSymbol: u.Symbol, RawUnit: ut.text,
				Start: t.start, End: ut.end}, nil
		}
		return &NumberNode{Num: t.rat, Start: t.start, End: t.end}, nil
	case tUnit:
		// bare unit means a quantity of 1
		ut := p.advance()
		u, _ := LookupUnit(ut.unit)
		return &NumberNode{Num: big.NewRat(1, 1), UnitSymbol: u.Symbol, RawUnit: ut.text,
			Start: ut.start, End: ut.end}, nil
	case tLParen:
		lp := p.advance()
		inner, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if p.peek().kind != tRParen {
			cur := p.peek()
			return nil, synErr(lp.start, cur.end, "missing closing parenthesis")
		}
		rp := p.advance()
		// Parentheses carry no operation: return the inner node directly so
		// that error intervals stay minimal (the smallest offending node),
		// while the outer span is still available if needed.
		_ = rp
		return inner, nil
	default:
		if t.kind == tEOF {
			return nil, synErr(t.start, t.end, "unexpected end of expression")
		}
		return nil, synErr(t.start, t.end, "unexpected token %q", t.text)
	}
}

var _ = strings.TrimSpace
