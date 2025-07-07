package lang

// https://cs.opensource.google/go/go/+/master:src/go/ast/ast.go
// we don't have tagged  unions in go, so nodes are interfaces
// and we discriminate different types by implementing empty methods
// like go/ast.go
//
//
// // All node types implement the Node interface.
// type Node interface {
// 	Pos() token.Pos // position of first character belonging to the node
// 	End() token.Pos // position of first character immediately after the node
// }

// // All expression nodes implement the Expr interface.
// type Expr interface {
// 	Node
// 	exprNode()
// }

// // All statement nodes implement the Stmt interface.
// type Stmt interface {
// 	Node
// 	stmtNode()
// }
// // stmtNode() ensures that only statement nodes can be
// assigned to a Stmt.
// func (*BadStmt) stmtNode()        {}
// func (*DeclStmt) stmtNode()       {}
// func (*EmptyStmt) stmtNode()      {}

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.lsp.dev/uri"

	"github.com/wycleffsean/nostos/pkg/urispec"
)

type parser struct {
	current       *item
	peeked        *item
	currentIndent uint
	priorIndent   uint
	priorNode     node
	tokens        <-chan item
	uri           uri.URI
}

func NewParser(tokens <-chan item, u uri.URI) *parser {
	return &parser{
		tokens: tokens,
		uri:    u,
	}
}

type Precedence int

const (
	precedenceLowest Precedence = iota
	precedenceEquality
	precedenceLessGreater
	precedenceSum
	precedenceProduct
	precedencePrefix
	precedenceCall
)

type node interface {
	Pos() Position // position of first character belonging to the node
	// End() Position // position of first character immediately after the node
}

type binaryOpNode interface {
	node
	leftExpr() node
	rightExpr() node
}

type collectionNode interface {
	node
	Symbols() []node
}

type errorNode interface {
	node
	Error() string
}

// -----------------------------------------------------
// Errors
//
// We attempt to recover from parsing errors by turning
// parse errors into nodes. This hopefully enables
// LSP capabilities, and reporting all parse errors in
// a document instead of just the first
type ParseError struct {
	File    uri.URI
	Message string
	Token   *item
}

func (e *ParseError) Pos() Position { return e.Token.position }

func (e *ParseError) URI() uri.URI { return e.File }

func (e *ParseError) StackTrace() []string { return nil }

// func (e *ParseError) End() pos { return pos(e.Token.position.ByteOffset + e.Token.position.ByteLength) }
func (e *ParseError) Error() string {
	line := e.Token.position.LineNumber
	offset := e.Token.position.CharacterOffset
	tokenString := e.Token.val
	return fmt.Sprintf("ParseError:%d:%d '%s' - %s", line, offset, tokenString, e.Message)
}

// -----------------------------------------------------
// Comments
//
// A Comment node represents a single #-style comment.
//
// The Text field contains the comment text without
// carriage returns (\r) that may have been present in
// the source.  Because a comment's end position is
// computed using len(Text), the position reported by
// [Comment.End] does not match the true source end
// position for comments containing carriage returns.
type Comment struct {
	Hash Position // position of "#" starting the comment
	Text string   // comment text (excluding '\n')
}

func (c *Comment) Pos() Position { return c.Hash }

// func (c *Comment) End() pos { return pos(int(c.Hash) + len(c.Text)) }

// A CommentGroup represents a sequence of comments with
// no other tokens and no empty lines between.
type CommentGroup struct {
	List []*Comment // len(List) > 0
}

func (g *CommentGroup) Pos() Position { return g.List[0].Pos() }

// func (g *CommentGroup) End() pos { return g.List[len(g.List)-1].End() }

func isWhitespace(ch byte) bool { return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' }

func skipTrailingWhitespace(s string) string {
	i := len(s)
	for i > 0 && isWhitespace(s[i-1]) {
		i--
	}
	return s[0:i]
}

// Text returns the text of the comment.
// Comment markers, the first space of a line comment,
// and leading and trailing empty lines are removed.
// Multiple empty lines are reduced to one, and trailing
// space on lines is trimmed.
// Unless the result is empty, it is newline-terminated
func (g *CommentGroup) Text() string {
	if g == nil {
		return ""
	}
	comments := make([]string, len(g.List))
	for i, c := range g.List {
		comments[i] = c.Text
	}

	lines := make([]string, 0, 10) // most comments are less than 10 lines
	for _, c := range comments {
		c = c[1:] // drop leading '#'
		if len(c) > 0 && c[0] == ' ' {
			// strip first space
			c = c[1:]
		}

		// Split on newlines
		cl := strings.Split(c, "\n")

		// Walk lines, stripping trailing whitespace
		// and adding to list.
		for _, l := range cl {
			lines = append(lines, skipTrailingWhitespace(l))
		}
	}

	// Remove leading blank lines; convert runs of
	// interior blank lines to a single blank line
	n := 0
	for _, line := range lines {
		if line != "" || n > 0 && lines[n-1] != "" {
			lines[n] = line
			n++
		}
	}
	lines = lines[0:n]

	// Add final "" entry to get trailing newline
	// from Join
	if n > 0 && lines[n-1] != "" {
		lines = append(lines, "")
	}

	return strings.Join(lines, "\n")
}

// -----------------------------------------------------
// Strings
type String struct {
	Position Position
	Text     string
}

func (s *String) Pos() Position { return s.Position }

// func (s *String) End() pos { return pos(int(s.Position) + len(s.Text)) }

// -----------------------------------------------------
// Symbol
type Symbol struct {
	Position Position
	Text     string
}

func (s *Symbol) Pos() Position { return s.Position }

// -----------------------------------------------------
// Path literal parsing is defined in path.go

// -----------------------------------------------------
// Map - dictionary/hash/associative array
type Map map[Symbol]node

// These functions have really terrible O(...) performance
func (m *Map) Pos() Position {
	var (
		pos   Position
		first = true
	)
	for symbol := range *m {
		sPos := symbol.Pos()
		if first || sPos.Less(pos) {
			pos = sPos
			first = false
		}
	}
	return pos

}

// TODO: we're cheating here, this is
// a poorly defined interface because collections
// might be key/value or just value.  We're looking
// for the symbols not the value nodes
func (m *Map) Symbols() []node {
	symbols := make([]node, 0)
	for key := range *m {
		symbols = append(symbols, &key)
	}
	return symbols
}

// func (m *Map) End() pos {
// 	var pos pos = math.MaxInt32
// 	for _, value := range *m {
// 		if pos < value.End() {
// 			pos = value.End()
// 		}
// 	}
// 	return pos
// }

type parseFn func(*parser) node
type infixFn func(*parser, node) node
type tokenMapping struct {
	Precedence
	parseFn
	infixFn
}

// TODO: do exhaustive check on this
// https://github.com/nishanths/exhaustive?tab=readme-ov-file#example
// stupid this can't be a const :/
var tokenMap map[itemType]tokenMapping

func init() {
	tokenMap = make(map[itemType]tokenMapping)
	tokenMap[itemError] = tokenMapping{precedenceCall, lexError, leftDenotationUnhandled}
	tokenMap[itemColon] = tokenMapping{precedenceCall, nullDenotationUnhandled, _map}
	tokenMap[itemArrow] = tokenMapping{precedenceCall, nullDenotationUnhandled, _function}
	tokenMap[itemShovel] = tokenMapping{precedenceCall, nullDenotationUnhandled, _shovel}
	tokenMap[itemLeftParen] = tokenMapping{precedenceCall, nullDenotationUnhandled, _call}
	tokenMap[itemRightParen] = tokenMapping{precedenceLowest, nullDenotationUnhandled, leftDenotationUnhandled}
	tokenMap[itemList] = tokenMapping{precedenceLowest, _list, leftDenotationUnhandled}
	tokenMap[itemNumber] = tokenMapping{precedenceLowest, _number, leftDenotationUnhandled}
	tokenMap[itemString] = tokenMapping{precedenceLowest, _string, leftDenotationUnhandled}
	tokenMap[itemPath] = tokenMapping{precedenceLowest, _path, leftDenotationUnhandled}
	tokenMap[itemSymbol] = tokenMapping{precedenceLowest, symbol, leftDenotationUnhandled}
	tokenMap[itemLet] = tokenMapping{precedenceLowest, _let, leftDenotationUnhandled}
	tokenMap[itemIn] = tokenMapping{precedenceLowest, nullDenotationUnhandled, leftDenotationUnhandled}
}

func (p *parser) _error(message string) node {
	return &ParseError{File: p.uri, Message: message, Token: p.current}
}

func nullDenotationUnhandled(p *parser) node {
	return p._error(fmt.Sprintf("unhandled null denotation reached for '%v'", p.current.typ))
}

func leftDenotationUnhandled(p *parser, _ node) node {
	return p._error(fmt.Sprintf("unhandled left denotation reached for '%v'", p.current.typ))
}

func lexError(p *parser) node {
	return p._error(p.current.val)
}

func (p *parser) Parse() node {
	var root node
	for !p.isEOF() {
		res := p.parseExpression(precedenceLowest)
		// we only loop to manage sibling leafs
		// e.g.
		//   foo: "first loop"
		//   bar: "second loop"
		// this is true for maps and arrays
		// both iterations should yield the same node, if they
		// aren't then this would be a statement which is always
		// an error
		// TODO: this check doesn't actually work
		// if root != nil && res != root {
		// 	log.Fatalf("Parse error: document must be a single expression\n\tcurrent token: %v\n\tnext token: %v\n\troot: %v\n\tres:  %v\n", self.current, self.peek(), root, res)
		// }
		if rootMap, ok := root.(*Map); ok {
			if resMap, ok := res.(*Map); ok {
				for k, v := range *resMap {
					(*rootMap)[k] = v
				}
				root = rootMap
			} else {
				root = res
			}
		} else if root == nil {
			root = res
		} else {
			root = res
		}
		if err, ok := root.(errorNode); ok {
			return err
		}

	}
	return root
}

func (p *parser) isEOF() bool {
	return p.peek().typ == itemEOF
}

func (p *parser) peek() *item {
	if p.peeked != nil {
		return p.peeked
	} else {
		select {
		case tok, ok := <-p.tokens:
			if !ok {
				eof := &item{typ: itemEOF}
				p.peeked = eof
				return eof

			}
			// fmt.Printf("-> %v\n", tok)
			p.peeked = &tok
			return &tok
		case <-time.After(2 * time.Second):
			err := &item{typ: itemError, val: "lexer timeout"}
			p.peeked = err
			return err
		}
	}
}

func (p *parser) accept() *item {
	peeked := p.peek()
	p.peeked = nil
	p.current = peeked
	p.currentIndent = peeked.indent
	// fmt.Printf("accept: parse.current: %v\n", peeked)
	return peeked
}

func (p *parser) peekPrecedence() Precedence {
	token := p.peek()
	mapping, ok := tokenMap[token.typ]
	if !ok {
		return -1 // we've probably hit EOF
	}
	return mapping.Precedence
}

func (p *parser) parseExpression(precedence Precedence) node {
	token := p.accept()
	if token == nil {
		return p._error("TODO: Unexpected end of stream")
	}
	mapping, ok := tokenMap[token.typ]
	if !ok {
		return p._error(fmt.Sprintf("missing parser production for '%v'", token.typ))
	}
	lhs := mapping.parseFn(p)
	if err, ok := lhs.(errorNode); ok {
		return err
	}
	for precedence < p.peekPrecedence() {
		token = p.accept()
		if token == nil {
			return p._error("TODO: Unexpected end of stream")
		}
		mapping := tokenMap[token.typ]
		lhs = mapping.infixFn(p, lhs)
		if err, ok := lhs.(errorNode); ok {
			return err
		}
	}

	// We save values for indent
	// and prior nodes because some expressions
	// are siblings that need to be bound to
	// one another
	// e.g. successive key value pairs with like
	// indentation form map literals
	p.priorIndent = p.currentIndent
	p.priorNode = lhs
	return lhs
}

func _string(p *parser) node {
	return &String{p.current.position, p.current.val}
}

func _path(p *parser) node {
	spec := urispec.Parse(p.current.val)
	return &Path{p.current.position, spec}
}

func _number(p *parser) node {
	v, err := strconv.ParseFloat(p.current.val, 64)
	if err != nil {
		return p._error(err.Error())
	}
	return &Number{p.current.position, v}
}

func symbol(p *parser) node {
	return &Symbol{p.current.position, p.current.val}
}

func _map(p *parser, key node) node {
	var m *Map

	if prior, ok := p.priorNode.(*Map); ok && p.currentIndent == p.priorIndent {
		m = prior
	} else {
		tmp := make(Map)
		m = &tmp
	}

	indent := p.currentIndent

	oldNode := p.priorNode
	oldIndent := p.priorIndent
	p.priorNode = nil
	p.priorIndent = 0

	value := p.parseExpression(precedenceEquality)
	if err, ok := value.(errorNode); ok {
		return err
	}
	(*m)[*key.(*Symbol)] = value
	p.priorNode = oldNode
	p.priorIndent = oldIndent

	for {
		next := p.peek()
		if next.typ != itemSymbol || next.indent != indent {
			break
		}
		p.accept()
		k := symbol(p)
		if p.peek().typ != itemColon {
			return p._error("expected ':'")
		}
		p.accept()

		oldNode := p.priorNode
		oldIndent := p.priorIndent
		p.priorNode = nil
		p.priorIndent = 0

		val := p.parseExpression(precedenceEquality)
		if err, ok := val.(errorNode); ok {
			return err
		}
		(*m)[*k.(*Symbol)] = val

		p.priorNode = oldNode
		p.priorIndent = oldIndent
	}

	// Update parser state so subsequent key-value pairs at the same indent
	// are added to this map
	p.priorNode = m
	p.priorIndent = indent
	// Reset currentIndent so the calling parseExpression records the
	// indentation of the key rather than the value.
	p.currentIndent = indent

	return m
}

func _list(p *parser) node {
	var l *List
	listIndent := p.currentIndent
	if prior, ok := p.priorNode.(*List); ok && listIndent == p.priorIndent {
		l = prior
	} else {
		l = new(List)
	}

	for {
		value := p.parseExpression(precedenceEquality)
		if err, ok := value.(errorNode); ok {
			return err
		}

		// consume additional expressions that belong to the same list item
		for {
			next := p.peek()
			if next.typ == itemEOF || next.indent <= listIndent {
				break
			}
			if next.typ == itemList && next.indent == listIndent {
				break
			}
			value = p.parseExpression(precedenceEquality)
			if err, ok := value.(errorNode); ok {
				return err
			}
		}

		*l = append(*l, value)

		next := p.peek()
		if next.typ != itemList || next.indent != listIndent {
			break
		}
		p.accept()
	}

	p.priorNode = l
	p.priorIndent = listIndent
	return l
}

func _function(p *parser, param node) node {
	sym, ok := param.(*Symbol)
	if !ok {
		return p._error("function parameter must be a symbol")
	}

	body := p.parseExpression(precedenceEquality)
	if err, ok := body.(errorNode); ok {
		return err
	}

	f := &Function{Param: sym, Body: body}
	return f
}

func _let(p *parser) node {
	pos := p.current.position
	bindingsExpr := p.parseExpression(precedenceEquality)
	if err, ok := bindingsExpr.(errorNode); ok {
		return err
	}
	m, ok := bindingsExpr.(*Map)
	if !ok {
		return p._error("let bindings must be a map")
	}

	// Parse additional binding expressions until we encounter 'in'.
	for {
		if p.peek().typ == itemIn {
			break
		}
		next := p.parseExpression(precedenceEquality)
		if err, ok := next.(errorNode); ok {
			return err
		}
		nm, ok := next.(*Map)
		if !ok {
			return p._error("let bindings must be a map")
		}
		for k, v := range *nm {
			(*m)[k] = v
		}
	}

	// Reset parser state so the body expression doesn't
	// merge with the binding map via priorNode logic.
	p.priorNode = nil
	p.priorIndent = 0

	if p.peek().typ != itemIn {
		return p._error("expected 'in'")
	}
	p.accept()
	body := p.parseExpression(precedenceEquality)
	if err, ok := body.(errorNode); ok {
		return err
	}
	return &Let{Position: pos, Bindings: m, Body: body}
}

func _shovel(p *parser, left node) node {
	right := p.parseExpression(precedenceEquality)
	if err, ok := right.(errorNode); ok {
		return err
	}
	return &Shovel{Left: left, Right: right}
}

func _call(p *parser, left node) node {
	arg := p.parseExpression(precedenceLowest)
	if err, ok := arg.(errorNode); ok {
		return err
	}
	if p.peek().typ != itemRightParen {
		return p._error("expected right paren")
	}
	p.accept()
	return &Call{Func: left, Arg: arg}
}
