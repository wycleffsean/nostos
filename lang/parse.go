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
	"log"
	"math"
	"strings"
)

// TODO: this should come from "item" which should
// really be renamed "token".  So we'll refactor
// references to this type as "token.Pos"
type pos int

type parser struct {
	current       *item
	peeked        *item
	currentIndent uint
	priorIndent   uint
	priorNode     node
	tokens        chan item
	nodes         chan node
}

func NewParser(tokens chan item) *parser {
	return &parser{
		tokens: tokens,
		nodes:  make(chan node),
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
	Pos() pos // position of first character belonging to the node
	End() pos // position of first character immediately after the node
}

type unaryOpNode interface {
	node
	expr() node
}

type binaryOpNode interface {
	node
	leftExpr() node
	rightExpr() node
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
	Message string
	Token   *item
}

func (e *ParseError) Pos() pos { return pos(e.Token.position.ByteOffset) }
func (e *ParseError) End() pos { return pos(e.Token.position.ByteOffset + e.Token.position.ByteLength) }
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
	Hash pos    // position of "#" starting the comment
	Text string // comment text (excluding '\n')
}

func (c *Comment) Pos() pos { return c.Hash }
func (c *Comment) End() pos { return pos(int(c.Hash) + len(c.Text)) }

// A CommentGroup represents a sequence of comments with
// no other tokens and no empty lines between.
type CommentGroup struct {
	List []*Comment // len(List) > 0
}

func (g *CommentGroup) Pos() pos { return g.List[0].Pos() }
func (g *CommentGroup) End() pos { return g.List[len(g.List)-1].End() }

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
	Position pos
	Text     string
}

func (s *String) Pos() pos { return s.Position }
func (s *String) End() pos { return pos(int(s.Position) + len(s.Text)) }

// -----------------------------------------------------
// Symbol
type Symbol struct {
	Position pos
	Text     string
}

func (s *Symbol) Pos() pos { return s.Position }
func (s *Symbol) End() pos { return pos(int(s.Position) + len(s.Text)) }

// -----------------------------------------------------
// Map - dictionary/hash/associative array
type Map map[Symbol]node

// These functions have really terrible O(...) performance
func (m *Map) Pos() pos {
	var pos pos = 0
	for symbol, _ := range *m {
		if pos < symbol.Pos() {
			pos = symbol.Pos()
		}
	}
	return pos

}
func (m *Map) End() pos {
	var pos pos = math.MaxInt32
	for _, value := range *m {
		if pos < value.End() {
			pos = value.End()
		}
	}
	return pos
}

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
	tokenMap[itemError] = tokenMapping{precedenceCall, nullDenotationUnhandled, leftDenotationUnhandled}
	tokenMap[itemColon] = tokenMapping{precedenceCall, nullDenotationUnhandled, _map}
	tokenMap[itemString] = tokenMapping{precedenceLowest, _string, leftDenotationUnhandled}
	tokenMap[itemSymbol] = tokenMapping{precedenceLowest, symbol, leftDenotationUnhandled}
}

func (self *parser) _error(message string) node {
	return &ParseError{message, self.current}
}

func nullDenotationUnhandled(self *parser) node {
	return self._error(fmt.Sprintf("unhandled null denotation reached for '%v'", self.current.typ))
}

func leftDenotationUnhandled(self *parser, _ node) node {
	return self._error(fmt.Sprintf("unhandled left denotation reached for '%v'", self.current.typ))
}

func (self *parser) Parse() chan node {
	go self.parseRun()
	return self.nodes
}

func (self *parser) parseRun() {
	for !self.isEOF() {
		res := self.parseExpression(precedenceLowest)
		if err, ok := res.(errorNode); ok {
			log.Fatalf("Parse error: %v\n current token: %v\n next token: %v", err, self.current, self.peek())
		}
		self.nodes <- res

	}
	// close(self.tokens) // No more tokens will be delivered
}

func (self *parser) isEOF() bool {
	return self.peek().typ == itemEOF
}

func (self *parser) peek() *item {
	if self.peeked != nil {
		return self.peeked
	} else {
		peeked := <-self.tokens
		self.peeked = &peeked
		return &peeked
	}
}

func (self *parser) accept() *item {
	peeked := self.peek()
	self.peeked = nil
	self.current = peeked
	return peeked
}

func (self *parser) peekPrecedence() Precedence {
	token := self.peek()
	mapping, ok := tokenMap[token.typ]
	if !ok {
		return -1 // we've probably hit EOF
	}
	return mapping.Precedence
}

func (self *parser) slurpIndents() {
	for self.peek().typ == itemIndent {
		self.accept()
		self.currentIndent += 1
	}
}

func (self *parser) parseExpression(precedence Precedence) node {
	priorIndent := self.currentIndent
	self.slurpIndents()

	token := self.accept()
	if token == nil {
		return self._error("TODO: Unexpected end of stream")
	}
	mapping, ok := tokenMap[token.typ]
	if !ok {
		return self._error(fmt.Sprintf("missing parser production for '%v'", token.typ))
	}
	lhs := mapping.parseFn(self)
	if err, ok := lhs.(errorNode); ok {
		return err
	}
	for precedence < self.peekPrecedence() {
		token = self.accept()
		if token == nil {
			return self._error("TODO: Unexpected end of stream")
		}
		mapping := tokenMap[token.typ]
		lhs = mapping.infixFn(self, lhs)
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
	self.priorIndent = self.currentIndent
	self.priorNode = lhs
	self.currentIndent = priorIndent
	return lhs
}

func _string(self *parser) node {
	return &String{0, self.current.val}
}

func symbol(self *parser) node {
	return &Symbol{0, self.current.val}
}

func _map(self *parser, key node) node {
	var m Map

	priorMap, last_node_was_map := self.priorNode.(*Map)

	if self.currentIndent == self.priorIndent && last_node_was_map {
		// continue building existing map
		m = *priorMap
	} else {
		m = make(map[Symbol]node)
	}

	value := self.parseExpression(precedenceEquality)
	if err, ok := value.(errorNode); ok {
		return err
	}
	m[*key.(*Symbol)] = value
	return &m
}
