// Package lex provides a very basic JSON lexer.
// The lexer only emits string values together with their positions in the
// JSON file and does not attempt to validate anything; it just searches for
// string values.
package lex

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"unicode"
	"unicode/utf8"
)

// Token types
const (
	Error = TokenType(iota)
	EOF
	String
)

const (
	eof = rune(-1)
)

// Token is a lexer token.
type Token struct {
	Pos   int
	Type  TokenType
	Value string
}

func (t Token) String() string {
	return fmt.Sprintf("%s (%s) at pos %d", t.Type, t.Value, t.Pos)
}

// TokenType is the token type.
type TokenType int

func (tt TokenType) String() string {
	switch tt {
	case Error:
		return "<ERROR>"
	case EOF:
		return "<EOF>"
	case String:
		return "<string>"
	default:
		return "<UNKNOWN>"
	}
}

// Lex lexes the JSON in r and returns a channel of Tokens.
func Lex(r io.Reader) <-chan Token {
	l := &lexer{
		input:  bufio.NewReader(r),
		tokens: make(chan Token),
	}
	go l.lex()
	return l.tokens
}

type lexer struct {
	input         io.RuneReader
	tokens        chan Token
	bufferedInput string
	bufPos        int
	consumed      int
	width         int
}

func (l *lexer) emit(tt TokenType) {
	val := l.bufferedInput[:l.bufPos]
	l.tokens <- Token{
		Pos:   l.pos() - len(val),
		Type:  tt,
		Value: val,
	}
	l.ignore()
}

func (l *lexer) emitIf(cond bool, tt TokenType) {
	if cond {
		l.emit(tt)
	}
}

func (l *lexer) emitEOF() {
	l.ignore()
	l.emit(EOF)
}

func (l *lexer) pos() int {
	return l.consumed + l.bufPos
}

func (l *lexer) ignore() {
	l.bufferedInput = l.bufferedInput[l.bufPos:]
	l.consumed += l.bufPos
	l.bufPos = 0
}

func (l *lexer) backup() {
	l.bufPos -= l.width
}

func (l *lexer) next() (r rune) {
	for l.bufPos >= len(l.bufferedInput) {
		err := l.readRune()
		if err == nil {
			continue
		}

		if errors.Is(err, io.EOF) {
			break
		}

		l.tokens <- Token{
			Type:  Error,
			Value: err.Error(),
		}
		break
	}

	if l.bufPos >= len(l.bufferedInput) {
		l.width = 0
		return eof
	}

	r, l.width = utf8.DecodeRuneInString(l.bufferedInput[l.bufPos:])
	l.bufPos += l.width

	return
}

func (l *lexer) readRune() error {
	r, _, err := l.input.ReadRune()
	if err != nil {
		return fmt.Errorf("read rune from input: %w", err)
	}
	l.bufferedInput += string(r)
	return nil
}

func (l *lexer) skipWhitespace() {
	for {
		r := l.next()
		if r == eof {
			l.backup()
			break
		}

		if !unicode.IsSpace(r) {
			l.backup()
			break
		}
	}
}

func (l *lexer) errorf(format string, args ...interface{}) stateFunc {
	l.tokens <- Token{
		Pos:   l.pos(),
		Type:  Error,
		Value: fmt.Sprintf(format, args...),
	}
	return nil
}

func (l *lexer) invalid(r rune, expected ...rune) stateFunc {
	tok := string(r)
	if r == eof {
		tok = "end of file"
	}

	if len(expected) == 0 {
		return l.errorf("invalid token %s at pos %d", tok, l.pos())
	}

	sexpected := make([]string, len(expected))
	for i, r := range expected {
		sexpected[i] = string(r)
	}
	return l.errorf("expected token at pos %d to be one of %s; got %s", l.pos(), sexpected, tok)
}

func (l *lexer) must(r rune) (rune, bool) {
	n := l.next()
	if n != r {
		l.backup()
		l.invalid(n, r)
		return n, false
	}
	return n, true
}

func (l *lexer) lex() {
	defer close(l.tokens)
	for state := lexDocument; state != nil; {
		state = state(l)
	}
}

type stateFunc func(*lexer) stateFunc

func lexDocument(l *lexer) stateFunc {
	l.skipWhitespace()

	r := l.next()
	switch r {
	case '"': // document is just a string
		l.backup()
		return lexString
	case '[':
		l.backup()
		return lexArray
	}

	return lexIgnore
}

func lexIgnore(l *lexer) stateFunc {
	l.skipWhitespace()
	for {
		r := l.next()
		switch r {
		case eof:
			l.emitEOF()
			return nil
		case '"':
			l.backup()
			return lexProperty
		default:
			return lexIgnore
		}
	}
}

func lexProperty(l *lexer) stateFunc {
	r := l.next()
	switch r {
	case '"':
		l.backup()
		return lexPropertyName
	default:
		l.backup()
		l.invalid(r, '"')
		return nil
	}
}

func lexPropertyName(l *lexer) stateFunc {
	r, ok := l.must('"')
	if !ok {
		return nil
	}

	for {
		r = l.next()
		switch r {
		case '\\':
			r = l.next() // don't check the escaped character
			break
		case '"':
			l.skipWhitespace()
			r = l.next()
			if r != ':' {
				l.backup()
				l.invalid(r, ':')
				return nil
			}
			l.skipWhitespace()

			r = l.next()
			switch r {
			case '"':
				l.backup()
				l.ignore()
				return lexString
			case '[':
				l.backup()
				return lexArray
			default:
				return lexIgnore
			}
		case eof:
			l.invalid(eof)
			return nil
		}
	}
}

func lexString(l *lexer) stateFunc {
	r, ok := l.must('"')
	if !ok {
		return nil
	}

	for {
		r = l.next()

		switch r {
		case '\\':
			r = l.next()
			break
		case '"':
			l.emit(String)
			return lexIgnore
		case eof:
			l.backup()
			l.invalid(eof)
			return nil
		}
	}
}

func lexArray(l *lexer) stateFunc {
	r, ok := l.must('[')
	if !ok {
		return nil
	}
	l.skipWhitespace()

	r = l.next()
	switch r {
	case '"':
		l.backup()
		l.ignore()
		return lexArrayString
	default:
		l.backup()
		return lexIgnore
	}
}

func lexArrayString(l *lexer) stateFunc {
	if lexString(l) == nil {
		return nil
	}

	l.skipWhitespace()
	r := l.next()
	switch r {
	case ',':
		l.skipWhitespace()
		r = l.next()
		switch r {
		case '"':
			l.backup()
			l.ignore()
			return lexArrayString
		default:
			return lexIgnore
		}
	default:
		return lexIgnore
	}
}
