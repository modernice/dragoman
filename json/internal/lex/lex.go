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
	"strings"
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

func (l *lexer) advance(n int) {
	l.bufPos += n
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

func (l *lexer) skipWhitespace() int {
	var skipped int
	for {
		r := l.next()
		if r == eof {
			l.backup()
			break
		}
		skipped++

		if !unicode.IsSpace(r) {
			l.backup()
			break
		}
	}
	return skipped
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

func (l *lexer) skipUntil(find ...rune) (rune, bool) {
	for {
		n := l.next()
		if n == eof {
			return n, false
		}
		for _, r := range find {
			if r == n {
				return n, true
			}
		}
	}
}

func (l *lexer) hasPrefix(prefix string) bool {
	for len(prefix) > len(l.bufferedInput[l.bufPos:]) {
		err := l.readRune()
		if err == nil {
			continue
		}

		if errors.Is(err, io.EOF) {
			break
		}

		l.tokens <- Token{
			Type:  Error,
			Value: (fmt.Errorf("has prefix: %w", err)).Error(),
		}
		return false
	}

	return strings.HasPrefix(l.bufferedInput[l.bufPos:], prefix)
}

func (l *lexer) lex() {
	defer close(l.tokens)
	for state := lexString; state != nil; {
		state = state(l)
	}
}

type stateFunc func(*lexer) stateFunc

func lexString(l *lexer) stateFunc {
	// skip until we find a double quote (string delimiter)
	r, ok := l.skipUntil('"')
	if !ok {
		l.emitEOF()
		return nil
	}
	l.backup()
	l.ignore()
	l.advance(1)

	// find the closing double quote
L:
	for {
		r, ok = l.skipUntil('"', '\\')
		if !ok {
			l.emitEOF()
			return nil
		}

		switch r {
		case '\\':
			r = l.next()
			switch r {
			case '"':
				continue L
			case eof:
				l.emitEOF()
				return nil
			default:
				continue L
			}
		case '"':
			break L
		case eof:
			l.emitEOF()
			return nil
		}
	}

	// Here we check if the string is a property key.
	// If the string is not followed by a colon (preceding whitespaces allowed),
	// then it's not a key and we can emit it as a string value.
	// Otherwise we just lex the next string.

	// store the currently buffered string, so we can emit it at a later time
	str := l.bufferedInput[:l.bufPos]
	pos := l.pos() - len(str)

	l.skipWhitespace()
	r = l.next()
	switch r {
	case ':': // str is a property key, so we just lex the next string
		return lexString
	default:
		l.backup()
		l.tokens <- Token{
			Pos:   pos,
			Type:  String,
			Value: str,
		}
		return lexString
	}
}
