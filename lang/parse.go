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
	"errors"
	"log"
	"strings"
)

// TODO: this should come from "item" which should
// really be renamed "token".  So we'll refactor
// references to this type as "token.Pos"
type pos int

type parser struct {
	current *item
	peeked  *item
	tokens  chan item
	nodes   chan node
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

type parseFn func(*parser) (node, error)
type infixFn func(*parser, node) (node, error)
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
	tokenMap[itemColon] = tokenMapping{precedenceCall, nullDenotationUnhandled, mapping}
	tokenMap[itemString] = tokenMapping{precedenceLowest, _string, leftDenotationUnhandled}
}

func nullDenotationUnhandled(_ *parser) (node, error) {
	return nil, errors.New("unhandled denotation reached")
}

func leftDenotationUnhandled(_ *parser, _ node) (node, error) {
	return nil, errors.New("unhandled left denotation")
}

func (self *parser) Parse() chan node {
	go self.parseRun()
	return self.nodes
}

func (self *parser) parseRun() {
	for !self.isEOF() {
		res, err := self.parseExpression(precedenceLowest)
		if err != nil {
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

func (self *parser) parseExpression(precedence Precedence) (node, error) {
	token := self.accept()
	if token == nil {
		return nil, errors.New("TODO: Unexpected end of stream")
	}
	mapping := tokenMap[token.typ]
	lhs, err := mapping.parseFn(self)
	if err != nil {
		return nil, err
	}
	for precedence < mapping.Precedence {
		token = self.accept()
		if token == nil {
			return nil, errors.New("TODO: Unexpected end of stream")
		}
		mapping := tokenMap[token.typ]
		lhs, err = mapping.infixFn(self, lhs)
		if err != nil {
			return nil, err
		}
	}
	return lhs, nil
}

func _string(self *parser) (node, error) {
	return &String{0, self.current.val}, nil
}

func mapping(_ *parser, _ node) (node, error) {
	return nil, errors.New("unhandled left denotation")
}
