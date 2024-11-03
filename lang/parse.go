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
)

type parser struct {
	peeked *item
	tokens chan item
	nodes  chan *node
}

func NewParser(tokens chan item) *parser {
	return &parser{
		tokens: tokens,
		nodes:  make(chan *node),
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
	Pos() int // position of first character belonging to the node
	End() int // position of first character immediately after the node
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

type parseFn func(*parser) (*node, error)
type infixFn func(*parser, *node) (*node, error)
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
}

func nullDenotationUnhandled(_ *parser) (*node, error) {
	return nil, errors.New("unhandled denotation reached")
}

func leftDenotationUnhandled(_ *parser, _ *node) (*node, error) {
	return nil, errors.New("unhandled left denotation")
}

func (self *parser) Parse() chan *node {
	go self.parseRun()
	return self.nodes
}

func (self *parser) parseRun() {
	for self.peek() != nil {
		res, err := self.parseExpression(precedenceLowest)
		if err != nil {
			// TODO: do something
			break
		}
		self.nodes <- res

	}
	close(self.tokens) // No more tokens will be delivered
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

func (self *parser) parseExpression(precedence Precedence) (*node, error) {
	token := self.peek()
	if token == nil {
		return nil, errors.New("TODO: Unexpected end of stream")
	}
	mapping := tokenMap[token.typ]
	lhs, err := mapping.parseFn(self)
	if err != nil {
		return nil, err
	}
	for precedence < mapping.Precedence {
		token = self.peek()
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

func mapping(_ *parser, _ *node) (*node, error) {
	return nil, errors.New("unhandled left denotation")
}
