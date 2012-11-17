// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package scan

import (
	"fmt"
	"strings"
	"testing"
	"unicode"
)

// We first implement a lexer for a simple language. The tests proper are at the end of the file.

// the tokens of the language
const (
	INTEGER = iota
	IDENTIFIER
	LPAREN
	RPAREN
	PLUS
	MINUS
)

// the start state of the state machine
func lexStart(s *Scanner) StateFn {
	// comments are (* ... *) and nested comments are allowed
	operators := "+-()"

	switch next := s.Peek(); {
	case unicode.IsLetter(next):
		return lexIdentifier
	case unicode.IsDigit(next):
		return lexInteger
	case strings.IndexRune(operators, next) >= 0:
		return lexOperator // also handles commens
	case next == ' ': // only spaces are legal whitespace
		return lexSpace
	case next == '\n':
		s.Next()
		s.Ignore()
		return lexStart
	case next == EOF:
		s.Emit(EOF)
		return nil
	}
	return s.Errorf("lex error")
}

func lexOperator(s *Scanner) StateFn {
	switch s.Next() {
	case '+':
		s.Emit(PLUS)
	case '-':
		s.Emit(MINUS)
	case '(':
		// have to check for comment
		if s.Peek() == '*' {
			return lexComment
		}
		s.Emit(LPAREN)
	case ')':
		s.Emit(RPAREN)
	}
	return lexStart
}

func isAlphaNumeric(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r)
}

func lexIdentifier(s *Scanner) StateFn {
	for isAlphaNumeric(s.Peek()) {
		s.Next()
	}
	// We could check for e.g. a keyword here.
	// word := s.Text()
	// if isKey(word) { ... }
	s.Emit(IDENTIFIER)
	return lexStart
}

func lexInteger(s *Scanner) StateFn {
	s.AcceptRun("0123456789")
	s.Emit(INTEGER)
	return lexStart
}

func lexSpace(s *Scanner) StateFn {
	for s.Peek() == ' ' {
		s.Next()
	}
	s.Ignore()
	return lexStart
}

// have already seen the first LPAREN. we do allow nested comments and
// "(*)" is treated as the opening of a comment.
func lexComment(s *Scanner) StateFn {
	next := s.Next()
	if next != '*' {
		s.Errorf("lex error")
	}

	level := 1
	for {
		switch s.Next() {
		case '(':
			if s.Peek() == '*' {
				s.Next() // make sure we don't match a '*)' in the next round
				level += 1
			}
		case '*':
			if s.Peek() == ')' {
				s.Next()
				level -= 1
			}
		default:
			// do nothing
		}
		if level == 0 {
			s.Ignore()
			return lexStart
		}
	}
	return lexStart
}

// The following tests the lexer above.

// Make the types prettyprint.
var itemName = map[ItemType]string{
	ERROR:      "error",
	EOF:        "EOF",
	INTEGER:    "integer",
	IDENTIFIER: "identifier",
	LPAREN:     "(",
	RPAREN:     ")",
	PLUS:       "+",
	MINUS:      "-",
}

func (i ItemType) String() string {
	s := itemName[i]
	if s == "" {
		return fmt.Sprintf("item%d", int(i))
	}
	return s
}

type lexTest struct {
	name  string
	input string
	items []Item
}

var (
	tEOF    = Item{EOF, 0, ""}
	tPlus   = Item{PLUS, 0, "+"}
	tMinus  = Item{MINUS, 0, "-"}
	tLparen = Item{LPAREN, 0, "("}
	tRparen = Item{RPAREN, 0, ")"}
)

var lexTests = []lexTest{
	{"empty", "", []Item{tEOF}},
	{"3 spaces", "   ", []Item{tEOF}},
	{"identifiers", `hokus pokus`, []Item{
		{IDENTIFIER, 0, "hokus"},
		{IDENTIFIER, 0, "pokus"},
		tEOF,
	}},
	{"identifiers with comment", `hokus (* first (*) nested *) last *) pokus`, []Item{
		{IDENTIFIER, 0, "hokus"},
		{IDENTIFIER, 0, "pokus"},
		tEOF,
	}},
	{"integers", "123 654 990", []Item{
		{INTEGER, 0, "123"},
		{INTEGER, 0, "654"},
		{INTEGER, 0, "990"},
		tEOF,
	}},
	{"expr with integers", "(123 + 654) - 990", []Item{
		tLparen,
		{INTEGER, 0, "123"},
		tPlus,
		{INTEGER, 0, "654"},
		tRparen,
		tMinus,
		{INTEGER, 0, "990"},
		tEOF,
	}},
}

// collect gathers the emitted items into a slice.
func collect(t *lexTest, left, right string) (items []Item) {
	s := New(t.name, t.input, lexStart)
	for {
		item := s.NextItem()
		items = append(items, item)
		if item.Typ == EOF || item.Typ == ERROR {
			break
		}
	}
	return
}

func equal(i1, i2 []Item, checkPos bool) bool {
	if len(i1) != len(i2) {
		return false
	}
	for k := range i1 {
		if i1[k].Typ != i2[k].Typ {
			return false
		}
		if i1[k].Val != i2[k].Val {
			return false
		}
		if checkPos && i1[k].Pos != i2[k].Pos {
			return false
		}
	}
	return true
}

func TestLex(t *testing.T) {
	for _, test := range lexTests {
		items := collect(&test, "", "")
		if !equal(items, test.items, false) {
			t.Errorf("%s: got\n\t%+v\nexpected\n\t%v", test.name, items, test.items)
		}
	}
}
