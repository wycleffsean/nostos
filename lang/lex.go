package lang

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

type itemType int

const (
	itemError itemType = iota // error occurred;
	// value is text of error
	itemDot // the cursor, spelled '.'
	itemDocStart
	itemDocEnd
	itemEOF
	// itemElse
	// itemEnd
	// itemField
	// itemIdentifier
	// itemIf
	// itemLeftMeta
	itemNumber
	// itemPipe
	// itemRange
	// itemRawString
	// itemRightMeta
	itemString
	// itemText
)

type item struct {
	typ itemType // Type, such as itemNumber
	val string   // Value, such as "23.2"
}

func (i item) String() string {
	switch i.typ {
	case itemEOF:
		return "EOF"
	case itemError:
		return i.val
	}
	if len(i.val) > 50 {
		return fmt.Sprintf("%.50s...", i.val)
	}
	return fmt.Sprintf("%s", i.val)
}

// holds the state of the scanner
type lexer struct {
	// TODO: input should be a (buffered) reader
	input string    // the string being scanned
	start int       // start position of this item
	pos   int       // current position in the input
	width int       // width of last rune read
	items chan item // channel of scanned items
}

type stateFn func(*lexer) stateFn

func Lex(input string) (*lexer, chan item) {
	l := &lexer{
		input: input,
		items: make(chan item),
	}
	go l.run() // Concurrent run state machine
	return l, l.items
}

func (l *lexer) run() {
	for state := lexFile; state != nil; {
		state = state(l)
	}
	l.emit(itemEOF) // TODO: wrong place to put this :/
	close(l.items)  // No more tokens will be delivered
}

func (l *lexer) emit(t itemType) {
	l.items <- item{t, l.input[l.start:l.pos]}
	l.start = l.pos
}

func (l *lexer) next() rune {
	if l.pos >= len(l.input) {
		l.width = 0
		return 0
	}
	var r rune
	r, l.width =
		utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return r
}

// ignore skips over the pending input before this point
func (l *lexer) ignore() {
	l.start = l.pos
}

// backup steps back one rune
// Can be called only once per call of next
func (l *lexer) backup() {
	l.pos -= l.width
}

// peek returns but does not consume the
// next rune in the input
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// accept consumes the next rune
// if it's from the valid set
func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

// acceptRun consumes a run of runes from the valid set
func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}
	l.backup()
}

func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{
		itemError,
		fmt.Sprintf(format, args...),
	}
	return nil
}

func isAlphaNumeric(r rune) bool {
	return r >= '0' && r <= 'z'
}

func lexFile(l *lexer) stateFn {
	_ = l
	return lexInDocument
}

func lexInDocument(l *lexer) stateFn {
	return lexString
}

func lexString(l *lexer) stateFn {
	if l.next() == '"' {
		// quoted strings
		l.ignore() // skip quote
		for {
			r := l.peek()
			if r == 0 {
				return l.errorf("EOF reached in unterminated string")
			} else if r == '"' && l.input[l.pos - 1] != '\\' {
				break
			}
			l.next()
		}
		l.emit(itemString)
		l.next() // consume and skip double quote
		l.ignore()
	} else {
		// unquoted strings
		for isAlphaNumeric(l.peek()) {
			l.next()
		}
		l.emit(itemString)
	}
	return nil
}

func lexKeyValue(l *lexer) stateFn {
	for isAlphaNumeric(l.peek()) {
		l.next()
	}
	l.emit(itemString)
	// can there be blank spaces between key and colon delimiter?
	// l.acceptRun(" ")
	if l.peek() != ':' {
		l.next()
		return l.errorf("key must be followed by a colon %q",
			l.input[l.start:l.pos])
	}
	l.ignore()
	return nil
}

func lexNumber(l *lexer) stateFn {
	// optional leading sign
	l.accept("+-")
	// is it hex?
	digits := "0123456789"
	if l.accept("0") && l.accept("xX") {
		digits = "0123456789abcdefABCDEF"
	}
	l.acceptRun(digits)
	if l.accept(".") {
		l.acceptRun(digits)
	}
	if l.accept("eE") {
		l.accept("+-")
		l.acceptRun("0123456789")
	}
	// Next thing can't be alphanumeric
	if isAlphaNumeric(l.peek()) {
		l.next()
		return l.errorf("bad number syntax: %q",
			l.input[l.start:l.pos])
	}
	l.emit(itemNumber)
	return lexInDocument
}
