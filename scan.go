// Copyright 2011 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package scan implements a generic scanner. A client needs to supply
// a state machine in the form of state functions returning other state
// functions as the next state.
//
// See Rob Pike's talk "Lexical Scanning in Go" for an introduction.
package scan

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// TODO: Remove or don't export Pos and Position?

// Item represents a token or text string returned from the scanner.
type Item struct {
	Typ ItemType // The type of this item.
	Pos Pos      // The starting position, in bytes, of this item in the input string.
	Val string   // The value of this item.
}

// Pos represents a byte position in the original input text.
type Pos int

func (p Pos) Position() Pos {
	return p
}

// ItemType identifies the type of scanned items.
type ItemType int

// Special items used by the package.
const (
	ERROR = -2
	EOF   = -1
)

// ItemToString can be defined by the client. It is used in the (Item).String method
// to print items with types declared by the client.
var ItemToString func(Item) string

func (i Item) String() string {
	switch {
	case i.Typ == EOF:
		return "EOF"
	case i.Typ == ERROR:
		return i.Val
	case ItemToString != nil:
		return ItemToString(i)
	case len(i.Val) > 10:
		return fmt.Sprintf("%.10q...", i.Val)
	}
	return fmt.Sprintf("%q", i.Val)
}

// StateFn represents the state of the scanner as a function that returns the next state.
type StateFn func(*Scanner) StateFn

// Scanner holds the state of the scanner.
type Scanner struct {
	name       string    // the name of the input; used only for error reports
	input      string    // the string being scanned
	state      StateFn   // the next scanning function to enter
	pos        Pos       // current position in the input
	start      Pos       // start position of this item
	width      Pos       // width of last rune read from input
	lastPos    Pos       // position of most recent item returned by nextItem
	items      chan Item // channel of scanned items
	parenDepth int       // nesting depth of ( ) exprs
}

// Next returns the next rune in the input.
func (s *Scanner) Next() rune {
	if int(s.pos) >= len(s.input) {
		s.width = 0
		return EOF
	}
	r, w := utf8.DecodeRuneInString(s.input[s.pos:])
	s.width = Pos(w)
	s.pos += s.width
	return r
}

// Peek returns but does not consume the next rune in the input.
func (s *Scanner) Peek() rune {
	r := s.Next()
	s.Backup()
	return r
}

// Backup steps back one rune. Can only be called once per call of next.
func (s *Scanner) Backup() {
	s.pos -= s.width
}

// Emit passes an item back to the client.
func (s *Scanner) Emit(t ItemType) {
	s.items <- Item{t, s.start, s.input[s.start:s.pos]}
	s.start = s.pos
}

// Ignore skips over the pending input before this point.
func (s *Scanner) Ignore() {
	s.start = s.pos
}

// Text returns the pending input before this point.
func (s *Scanner) Text() string {
	return s.input[s.start:s.pos]
}

// Accept consumes the next rune if it's from the valid set.
func (s *Scanner) Accept(valid string) bool {
	if strings.IndexRune(valid, s.Next()) >= 0 {
		return true
	}
	s.Backup()
	return false
}

// AcceptRun consumes a run of runes from the valid set.
func (s *Scanner) AcceptRun(valid string) {
	for strings.IndexRune(valid, s.Next()) >= 0 {
	}
	s.Backup()
}

// LineNumber reports which line we're on, based on the position of
// the previous Item returned by NextItem. Doing it this way
// means we don't have to worry about Peek double counting.
func (s *Scanner) LineNumber() int {
	return 1 + strings.Count(s.input[:s.lastPos], "\n")
}

// Errorf returns an error item and terminates the scan by passing
// back a nil pointer that will be the next state, terminating s.NextItem.
func (s *Scanner) Errorf(format string, args ...interface{}) StateFn {
	s.items <- Item{ERROR, s.start, fmt.Sprintf(format, args...)}
	return nil
}

// NextItem returns the next item from the input.
func (s *Scanner) NextItem() Item {
	item := <-s.items
	s.lastPos = item.Pos
	return item
}

// New creates a new scanner for the input string with initial state start.
func New(name, input string, start StateFn) *Scanner {
	s := &Scanner{
		name:  name,
		input: input,
		state: start,
		items: make(chan Item),
	}
	go s.run()
	return s
}

// run runs the state machine for the scanner.
func (s *Scanner) run() {
	for s.state != nil {
		s.state = s.state(s)
	}
}
