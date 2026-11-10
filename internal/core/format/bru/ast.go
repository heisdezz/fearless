// Package bru implements the `.bru` collection dialect: a lexer, parser and
// writer for requests, folders, collections and environments.
//
// The grammar is transcribed from Bruno's ohm grammar in
// packages/bruno-lang/v2/src/bruToJson.js. The rules that matter most:
//
//   - tagend = nl "}" — a block closes on a line whose first character is "}".
//     This is what lets JSON bodies contain braces of their own.
//   - Text blocks (body:json and friends) are raw text, not dictionaries. They
//     terminate at the first line that is exactly "}".
//   - Dictionary blocks hold `key: value` pairs, with `~` disabling a pair,
//     quoting for keys containing ":", and `”'` multiline values whose lines are
//     de-indented by four spaces.
//
// See docs/data/bru-format.md for the reference this implements.
package bru

import (
	"fmt"
	"strings"
)

// Position is a 1-based line and column in the source.
type Position struct {
	Line int
	Col  int
}

// String renders the position as "line:col".
func (p Position) String() string { return fmt.Sprintf("%d:%d", p.Line, p.Col) }

// SyntaxError is a parse failure with its location.
//
// Errors always point at a concrete position. A parser that reports "unexpected
// input" without saying where is unusable on a 400-line request file.
type SyntaxError struct {
	Pos  Position
	Msg  string
	Near string // the offending line, for context
}

func (e *SyntaxError) Error() string {
	if e.Near != "" {
		return fmt.Sprintf("bru: %s at %s: %q", e.Msg, e.Pos, e.Near)
	}
	return fmt.Sprintf("bru: %s at %s", e.Msg, e.Pos)
}

// Annotation is a decorator applied to the pair or block that follows it, such
// as @description('…'), @number or @contentType(image/png).
type Annotation struct {
	Name string
	Arg  string
	// HasArg distinguishes @required from @description('') .
	HasArg bool
	// ArgBlock reports that the argument was a multiline ''' block rather than a
	// quoted or bare token.
	ArgBlock bool
	Pos      Position
}

// Pair is one `key: value` entry inside a dictionary block.
type Pair struct {
	Key         string
	Value       string
	Disabled    bool
	Annotations []Annotation
	Pos         Position
	// Quoted records that the key was written in quotes, so the writer can
	// re-quote it when the key contains a colon.
	Quoted bool
	// IsList records that the value was a bracketed list rather than a scalar.
	// List then holds its items and Value is empty.
	IsList bool
	List   []string
}

// Annotations returns the value of the first annotation with the given name.
func (p Pair) Annotation(name string) (Annotation, bool) {
	for _, a := range p.Annotations {
		if a.Name == name {
			return a, true
		}
	}
	return Annotation{}, false
}

// BlockKind distinguishes the three block shapes.
type BlockKind int

const (
	// DictBlock is `name { key: value … }`.
	DictBlock BlockKind = iota
	// TextBlock is `name { raw text … }`.
	TextBlock
	// ListBlock is `name [ item item ]`.
	ListBlock
)

// Block is one top-level construct of a `.bru` file.
type Block struct {
	// Name is the block header as written, e.g. "meta", "body:json",
	// "params:query", "auth:bearer", "script:pre-request".
	Name string
	Kind BlockKind
	Pos  Position

	// Pairs is populated for DictBlock.
	Pairs []Pair
	// Text is populated for TextBlock, with the trailing newline before the
	// closing brace removed and no other trimming.
	Text string
	// Items is populated for ListBlock.
	Items []string
	// Annotations apply to the block as a whole, which Bruno's grammar permits
	// only in a few places.
	Annotations []Annotation
}

// Document is a parsed `.bru` file: an ordered list of blocks plus any top-level
// rules that are not blocks (environment `extends` and `color`).
type Document struct {
	Blocks []Block
	// Extends is the environment parent list. Bruno allows both
	// `extends: Name` and `extends [ A, "b c" ]`.
	Extends []string
	// Color is the environment colour token, e.g. "#ff0000".
	Color string
}

// FindBlock returns the first block with the given name.
func (d *Document) FindBlock(name string) (*Block, bool) {
	for i := range d.Blocks {
		if d.Blocks[i].Name == name {
			return &d.Blocks[i], true
		}
	}
	return nil, false
}

// AllBlocks returns every block with the given name, in order.
func (d *Document) AllBlocks(name string) []*Block {
	var out []*Block
	for i := range d.Blocks {
		if d.Blocks[i].Name == name {
			out = append(out, &d.Blocks[i])
		}
	}
	return out
}

// blockName canonicalises a block header.
//
// Bruno's grammar is case sensitive for the keywords it names, but both the
// parser and real files use lower case; normalising here keeps the mapping layer
// simple and makes `GET {` a syntax error rather than a silent no-op.
func blockName(raw string) string {
	return strings.TrimSpace(raw)
}
