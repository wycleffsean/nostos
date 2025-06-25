package lang

import (
	"fmt"
	"reflect"
	"runtime"
	"strings"
	"unicode/utf8"
)

type itemType int

//go:generate stringer -type=itemType
const (
	itemUndefined itemType = iota // undefined should never occur, we have a lexing error;
	itemError                     // error occurred;
	// value is text of error
	itemDot // the cursor, spelled '.'
	itemDocStart
	itemDocEnd
	itemEOF
	itemList
	itemColon
	itemArrow      // =>
	itemLeftParen  // (
	itemRightParen // )
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
	itemPath
	itemSymbol
	// itemText
)

type item struct {
	typ      itemType // Type, such as itemNumber
	val      string   // Value, such as "23.2"
	position Position
	indent   uint
}

type Position struct {
	// For slices
	ByteOffset uint
	ByteLength uint

	// For LSP
	// https://microsoft.github.io/language-server-protocol/specifications/lsp/3.17/specification/#position
	LineNumber      uint `json:"line"`
	CharacterOffset uint `json:"character"`
}

func (p *Position) Less(than Position) bool {
	if p.LineNumber < than.LineNumber {
		return true
	}
	if p.LineNumber == than.LineNumber {
		return p.CharacterOffset < than.CharacterOffset
	}
	return false
}

func (i item) String() string {
	switch i.typ {
	case itemEOF:
		return "EOF"
	case itemError:
		return i.val
	}
	if len(i.val) > 20 {
		return fmt.Sprintf("[%s]%.50s...", i.typ, i.val)
	}
	return fmt.Sprintf("[%s]%s", i.typ, i.val)
}

// holds the state of the scanner
type lexer struct {
	// TODO: input should be a (buffered) reader
	input         string // the string being scanned
	start         uint   // start position of this item
	pos           uint   // current position in the input
	currentLine   uint   // 0 indexed count of newline characters seen
	offset        uint   // 0 indexed count of character offset - corresponds to LSP spec PositionEncodingKind
	currentOffset uint   // counter for offset
	currentIndent uint
	width         uint      // width of last rune read
	items         chan item // channel of scanned items
}

type stateFn func(*lexer) stateFn

func NewStringLexer(input string) (*lexer, <-chan item) {
	l := &lexer{
		input: input,
		items: make(chan item),
	}
	go l.run() // Concurrent run state machine
	return l, l.items
}

func GetFunctionName(i interface{}) string {
	return runtime.FuncForPC(reflect.ValueOf(i).Pointer()).Name()
}

func (l *lexer) run() {
	for state := lexFile; state != nil; {
		state = state(l)
	}
	close(l.items) // No more tokens will be delivered
}

// Sometimes we want the lex token and the offset to be different
// e.g. strings - a token will carry the contents of the string,
//
//	but the offset will begin at the quote (they are off by 1)
func (l *lexer) markOffset() {
	l.offset = l.currentOffset
}

func (l *lexer) emit(t itemType) {
	position := Position{l.start, l.pos - l.start, l.currentLine, l.offset}
	l.publish(item{t, l.input[l.start:l.pos], position, l.currentIndent})
	l.markOffset()
	l.start = l.pos
}

func (l *lexer) next() rune {
	if l.pos >= uint(len(l.input)) {
		l.width = 0
		return 0
	}
	var r rune
	var runeWidth int
	r, runeWidth =
		utf8.DecodeRuneInString(l.input[l.pos:])
	l.width = uint(runeWidth)
	l.currentOffset += 1
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
	l.currentOffset -= 1
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
	if strings.ContainsRune(valid, l.next()) {
		return true
	}
	l.backup()
	return false
}

// acceptRun consumes a run of runes from the valid set
func (l *lexer) acceptRun(valid string) {
	for strings.ContainsRune(valid, l.next()) {
	}
	l.backup()
}

func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	position := Position{l.start, l.pos - l.start, l.currentLine, l.currentOffset}
	message := fmt.Sprintf(format, args...)
	l.publish(item{itemError, message, position, l.currentIndent})
	return nil
}

func (l *lexer) publish(item item) {
	// fmt.Printf("<- %v\n", item)
	l.items <- item
}

func isAlpha(r rune) bool {
	return r >= 'A' && r <= 'z'
}

func isNumber(r rune) bool {
	return r >= '0' && r <= '9'
}

func isAlphaNumeric(r rune) bool {
	return isAlpha(r) || isNumber(r)
}

func isValidKey(r rune) bool {
	return isAlphaNumeric(r) || r == '/' || r == '.'
}

func isPathStart(l *lexer) bool {
	if strings.HasPrefix(l.input[l.pos:], "../") {
		return true
	}
	if strings.HasPrefix(l.input[l.pos:], "./") {
		return true
	}
	if strings.HasPrefix(l.input[l.pos:], "/") {
		return true
	}
	return false
}

func isPathRune(r rune) bool {
	switch r {
	case 0, ' ', '\n', '\t', ':', '(', ')':
		return false
	}
	return true
}

func lexFile(l *lexer) stateFn {
	_ = l
	return lexInDocument
}

func lexInDocument(l *lexer) stateFn {
	r := l.peek()
	switch r {
	case 0:
		l.emit(itemEOF)
		return nil
	case '\n':
		return lexIndent
	case '\t':
		return l.errorf("horizontal tabs are not supported")
	case '.':
		if isPathStart(l) {
			return lexPath
		}
		l.next()
		l.emit(itemDot)
		return lexInDocument
	case '/':
		return lexPath
	case '-':
		return lexList
	case '(':
		l.next()
		l.emit(itemLeftParen)
		return lexInDocument
	case ')':
		l.next()
		l.emit(itemRightParen)
		return lexInDocument
	case '=':
		l.next()
		if l.next() == '>' {
			l.emit(itemArrow)
			return lexInDocument
		}
		return l.errorf("unexpected '='")
	case ':':
		l.next()
		l.emit(itemColon)
		return lexInDocument
	default:
		l.acceptRun(" ")
		l.ignore()
		if isNumber(l.peek()) {
			return lexNumber
		}
		if l.peek() == '"' {
			return lexString
		}
		if isAlpha(l.peek()) {
			return lexSymbol
		}
		// TODO: there's a potential for infinite loops here, we need
		//   to redesign the lexer in a way that forces the cursor to make
		//   progress at all times
		// if there's a bug in the lexer we've likely hit this branch
		// fmt.Printf("===lexInDocument - hit default case with r='%c'\n", r)
		return lexInDocument
	}
}

// a newline or a prior indent can get us here
func lexIndent(l *lexer) stateFn {
	if l.peek() == '\n' {
		l.next()
		l.ignore()
		l.currentLine += 1
		l.currentOffset = 0
		l.currentIndent = 0
		l.markOffset()
	}
	if l.peek() == ' ' {
		l.next()
		if l.next() != ' ' {
			return l.errorf("indents must contain two spaces")
		}
		l.markOffset()
		l.currentIndent += 1
		if l.peek() == ' ' {
			return lexIndent
		}
	}
	return lexInDocument
}

func lexList(l *lexer) stateFn {
	l.emit(itemList)
	if l.next() != '-' {
		return l.errorf("oops! we were expecting a list here")
	}
	l.ignore()
	return lexInDocument
}

func lexSymbol(l *lexer) stateFn {
	// unquoted strings
	for isValidKey(l.peek()) {
		l.next()
	}
	l.emit(itemSymbol)
	return lexInDocument
}

func lexString(l *lexer) stateFn {
	l.markOffset() // "character offset" for LSP will begin at double quote
	if l.next() != '"' {
		return l.errorf("Strings must be quoted")
	}
	l.ignore() // skip quote
	for {
		r := l.peek()
		if r == 0 {
			l.ignore()
			return l.errorf("EOF reached in unterminated string")
		} else if r == '"' && l.input[l.pos-1] != '\\' {
			break
		}
		l.next()
	}
	l.emit(itemString)
	l.next() // consume and skip double quote
	l.ignore()
	return lexInDocument
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

func lexPath(l *lexer) stateFn {
	for isPathRune(l.peek()) {
		l.next()
	}
	l.emit(itemPath)
	return lexInDocument
}
