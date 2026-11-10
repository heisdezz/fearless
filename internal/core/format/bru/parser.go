package bru

import (
	"strings"
)

// parser walks the source. It is a hand-written recursive-descent parser over a
// byte slice, tracking line and column for error reporting.
type parser struct {
	src  string
	pos  int
	line int
	col  int
}

// Parse parses a `.bru` file of any dialect (request, folder, collection,
// environment) into a Document.
//
// One parser serves all four dialects because Bruno's grammar dispatches on
// block names, not on the file's role: a folder file is simply one that lacks a
// method block. Distinguishing them is the mapper's job, not the parser's.
func Parse(src []byte) (*Document, error) {
	p := &parser{src: string(src), line: 1, col: 1}
	// A leading UTF-8 BOM would otherwise be read as part of the first block name.
	p.src = strings.TrimPrefix(p.src, "\ufeff")
	p.src = strings.ReplaceAll(p.src, "\r\n", "\n")
	return p.parseDocument()
}

// eof reports whether the input is exhausted.
func (p *parser) eof() bool { return p.pos >= len(p.src) }

// peek returns the current byte, or 0 at EOF.
func (p *parser) peek() byte {
	if p.eof() {
		return 0
	}
	return p.src[p.pos]
}

// advance consumes one byte, maintaining line and column.
func (p *parser) advance() byte {
	c := p.src[p.pos]
	p.pos++
	if c == '\n' {
		p.line++
		p.col = 1
	} else {
		p.col++
	}
	return c
}

// position reports the current location.
func (p *parser) position() Position { return Position{Line: p.line, Col: p.col} }

// errorf builds a SyntaxError at the current position.
func (p *parser) errorf(format string, args ...any) error {
	return &SyntaxError{
		Pos:  p.position(),
		Msg:  sprintf(format, args...),
		Near: p.currentLine(),
	}
}

// currentLine returns the line containing the cursor, for error context.
func (p *parser) currentLine() string {
	start := p.pos
	for start > 0 && p.src[start-1] != '\n' {
		start--
	}
	end := p.pos
	for end < len(p.src) && p.src[end] != '\n' {
		end++
	}
	line := p.src[start:end]
	if len(line) > 120 {
		line = line[:120] + "…"
	}
	return line
}

// skipBlankLines consumes whitespace-only lines and blank lines between blocks.
// Indentation on a blank line is insignificant.
func (p *parser) skipBlankLines() {
	for !p.eof() {
		// Consume horizontal whitespace.
		for !p.eof() && (p.peek() == ' ' || p.peek() == '\t') {
			p.advance()
		}
		if p.eof() {
			return
		}
		if p.peek() == '\n' {
			p.advance()
			continue
		}
		return
	}
}

// atLineStart reports whether only whitespace precedes the cursor on this line.
func (p *parser) atLineStart() bool {
	start := p.pos
	for start > 0 && p.src[start-1] != '\n' {
		start--
	}
	for i := start; i < p.pos; i++ {
		if p.src[i] != ' ' && p.src[i] != '\t' {
			return false
		}
	}
	return true
}

// parseDocument consumes top-level blocks and rules.
func (p *parser) parseDocument() (*Document, error) {
	doc := &Document{}
	for {
		p.skipBlankLines()
		if p.eof() {
			return doc, nil
		}

		// Top-level rules appear in environment files only, and are bare
		// `extends: X`, `extends [ … ]` or `color: #hex` lines with no braces.
		if p.matchTopLevelRule("extends") {
			if err := p.parseExtends(doc); err != nil {
				return nil, err
			}
			continue
		}
		if p.matchTopLevelRule("color") {
			if err := p.parseColor(doc); err != nil {
				return nil, err
			}
			continue
		}

		block, err := p.parseBlock()
		if err != nil {
			return nil, err
		}
		doc.Blocks = append(doc.Blocks, *block)
	}
}

// matchTopLevelRule reports whether the cursor is at the given bare keyword
// followed by ":" or whitespace, and whether the word is not a block name like
// "extendswhatever".
func (p *parser) matchTopLevelRule(word string) bool {
	if !p.atLineStart() {
		return false
	}
	save := *p
	defer func() { *p = save }()

	// Allow leading indentation before a rule.
	for !p.eof() && (p.peek() == ' ' || p.peek() == '\t') {
		p.advance()
	}
	if !strings.HasPrefix(p.src[p.pos:], word) {
		return false
	}
	rest := p.src[p.pos+len(word):]
	rest = strings.TrimLeft(rest, " \t")
	// A rule is followed by ":" or, for extends, "[".
	return strings.HasPrefix(rest, ":") || (word == "extends" && strings.HasPrefix(rest, "["))
}

// parseExtends parses `extends: Name` or `extends [ A, "b c" ]`.
func (p *parser) parseExtends(doc *Document) error {
	p.skipInlineSpace()
	p.consumeKeyword("extends")
	p.skipInlineSpace()

	if p.peek() == ':' {
		p.advance()
		p.skipInlineSpace()
		name := strings.TrimSpace(p.readLine())
		if name == "" {
			return p.errorf("empty extends value")
		}
		doc.Extends = append(doc.Extends, name)
		return nil
	}

	if p.peek() != '[' {
		return p.errorf("expected ':' or '[' after extends")
	}
	p.advance() // '['
	for {
		p.skipAllWhitespace()
		if p.eof() {
			return p.errorf("unterminated extends list")
		}
		if p.peek() == ']' {
			p.advance()
			return nil
		}
		if p.peek() == ',' {
			p.advance()
			continue
		}
		name, err := p.parseQuotedOrBareUntil(",]")
		if err != nil {
			return err
		}
		name = strings.TrimSpace(name)
		if name != "" {
			doc.Extends = append(doc.Extends, name)
		}
	}
}

// parseColor parses `color: #ff0000`.
func (p *parser) parseColor(doc *Document) error {
	p.skipInlineSpace()
	p.consumeKeyword("color")
	p.skipInlineSpace()
	if p.peek() != ':' {
		return p.errorf("expected ':' after color")
	}
	p.advance()
	doc.Color = strings.TrimSpace(p.readLine())
	return nil
}

// parseBlock reads one `name { … }` or `name [ … ]` construct.
func (p *parser) parseBlock() (*Block, error) {
	start := p.position()

	// Leading annotations on a block (rare, but the grammar allows them).
	var annotations []Annotation
	for p.peek() == '@' {
		ann, err := p.parseAnnotation()
		if err != nil {
			return nil, err
		}
		annotations = append(annotations, ann)
		p.skipInlineSpace()
	}

	header := strings.TrimSpace(p.readUntilOneOf("{["))
	if header == "" {
		return nil, p.errorf("expected a block name")
	}

	switch p.peek() {
	case '{':
		p.advance()
		return p.parseBracedBlock(header, start, annotations)
	case '[':
		p.advance()
		return p.parseListBlock(header, start, annotations)
	case 0:
		return nil, &SyntaxError{Pos: start, Msg: "unexpected end of file after " + strconvQuote(header)}
	default:
		return nil, p.errorf("expected '{' or '[' after block name %s", strconvQuote(header))
	}
}

// parseBracedBlock decides between a dictionary and a text block.
//
// The distinction is Bruno's, not ours: blocks whose names are text-bearing
// (body:<mode>, script:*, tests, docs, assert…) hold raw text, everything else
// holds `key: value` pairs. `assert` is the notable exception — it looks like a
// dictionary but its keys contain dots and its values contain spaces, so it is
// parsed as dictionaries whose keys are almost free-form.
func (p *parser) parseBracedBlock(name string, start Position, anns []Annotation) (*Block, error) {
	block := &Block{Name: blockName(name), Kind: DictBlock, Pos: start, Annotations: anns}

	if isTextBlock(name) {
		text, err := p.parseTextBody()
		if err != nil {
			return nil, err
		}
		block.Kind = TextBlock
		block.Text = text
		return block, nil
	}

	pairs, err := p.parsePairs(name)
	if err != nil {
		return nil, err
	}
	block.Pairs = pairs
	return block, nil
}

// parsePairs reads `key: value` entries until the closing brace line.
func (p *parser) parsePairs(blockName string) ([]Pair, error) {
	var pairs []Pair
	var pending []Annotation

	for {
		p.skipBlankLines()
		if p.eof() {
			return nil, p.errorf("unterminated block %s", strconvQuote(blockName))
		}

		// tagend: a line beginning with "}" closes the block.
		if p.peek() == '}' {
			p.advance()
			// Consume the rest of the closing line if any.
			rest := strings.TrimSpace(p.readLine())
			if rest != "" {
				return nil, p.errorf("unexpected %q after closing brace", rest)
			}
			return pairs, nil
		}

		// Annotations attach to the pair that follows.
		if p.peek() == '@' {
			ann, err := p.parseAnnotation()
			if err != nil {
				return nil, err
			}
			pending = append(pending, ann)
			// An annotation may be followed by end of line.
			p.skipInlineSpace()
			if p.peek() == '\n' {
				p.advance()
			}
			continue
		}

		pair, err := p.parsePair(pending)
		if err != nil {
			return nil, err
		}
		pending = nil
		pairs = append(pairs, pair)
	}
}

// parsePair reads one key/value entry, including a multiline ”' value.
func (p *parser) parsePair(anns []Annotation) (Pair, error) {
	pos := p.position()
	pair := Pair{Annotations: anns, Pos: pos}

	// Disabled marker.
	if p.peek() == '~' {
		p.advance()
		pair.Disabled = true
	}

	// Quoted key.
	if p.peek() == '"' {
		p.advance()
		var b strings.Builder
		for !p.eof() && p.peek() != '"' {
			if p.peek() == '\\' && p.pos+1 < len(p.src) && p.src[p.pos+1] == '"' {
				p.advance()
				b.WriteByte(p.advance())
				continue
			}
			b.WriteByte(p.advance())
		}
		if p.eof() {
			return pair, p.errorf("unterminated quoted key")
		}
		p.advance() // closing quote
		pair.Key = b.String()
		pair.Quoted = true
	} else {
		pair.Key = strings.TrimSpace(p.readUntilOneOf(":\n"))
	}

	p.skipInlineSpace()
	if p.peek() != ':' {
		return pair, p.errorf("expected ':' after key %s", strconvQuote(pair.Key))
	}
	p.advance() // ':'
	p.skipInlineSpace()

	// A list value, as in `tags: [` then one item per line then `]`.
	if p.peek() == '[' {
		p.advance()
		items, err := p.parseListValue()
		if err != nil {
			return pair, err
		}
		pair.List = items
		pair.IsList = true
		p.skipInlineSpace()
		if p.peek() == '\n' {
			p.advance()
		}
		return pair, nil
	}

	// A multiline value.
	if strings.HasPrefix(p.src[p.pos:], "'''") {
		text, err := p.parseMultilineValue()
		if err != nil {
			return pair, err
		}
		pair.Value = text
		return pair, nil
	}

	pair.Value = strings.TrimSpace(p.readLine())
	return pair, nil
}

// parseListValue reads the items of a bracketed list value up to and including
// the closing bracket.
//
// Bruno's list rule separates items by newlines (with optional commas in the
// `extends` form), so newlines terminate an item rather than becoming part of it.
func (p *parser) parseListValue() ([]string, error) {
	var items []string
	for {
		// Skip separators.
		for !p.eof() {
			switch p.peek() {
			case ' ', '\t', '\n', ',':
				p.advance()
				continue
			}
			break
		}
		if p.eof() {
			return nil, p.errorf("unterminated list value")
		}
		if p.peek() == ']' {
			p.advance()
			return items, nil
		}

		// Read one item up to a separator or the closing bracket.
		item, err := p.parseQuotedOrBareUntil(",]\n")
		if err != nil {
			return nil, err
		}
		if s := strings.TrimSpace(item); s != "" {
			items = append(items, s)
		}
	}
}

// parseMultilineValue reads a ”'…”' value, de-indenting each line by four
// spaces as the grammar specifies.
func (p *parser) parseMultilineValue() (string, error) {
	p.advance()
	p.advance()
	p.advance() // "'''"

	var raw strings.Builder
	for {
		if p.eof() {
			return "", p.errorf("unterminated ''' value")
		}
		if strings.HasPrefix(p.src[p.pos:], "'''") {
			p.advance()
			p.advance()
			p.advance()
			break
		}
		raw.WriteByte(p.advance())
	}

	// A trailing @contentType(...) may follow the closing delimiter.
	p.skipInlineSpace()
	if p.peek() == '@' {
		if _, err := p.parseAnnotation(); err != nil {
			return "", err
		}
	}

	return dedentMultiline(raw.String()), nil
}

// dedentMultiline applies Bruno's multiline rule: every line loses exactly four
// leading characters (`line.slice(4)` in bruToJson.js), with no special case for
// short lines and no removal of the leading newline. This differs from the 2-
// space outdent used for text bodies, which is why both helpers exist.
func dedentMultiline(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		if len(line) >= 4 {
			lines[i] = line[4:]
		} else {
			// slice(4) on a shorter string yields "" in JS.
			lines[i] = ""
		}
	}
	return strings.Join(lines, "\n")
}

// trimIndent removes up to n leading spaces from a line.
func trimIndent(line string, n int) string {
	removed := 0
	for removed < n && removed < len(line) && line[removed] == ' ' {
		removed++
	}
	return line[removed:]
}

// parseTextBody reads a raw text block up to the closing brace line.
//
// Bruno's rule is `textblock = textline (~tagend nl textline)*` with
// `tagend = nl "}"` and `textchar = ~nl any`. Ohm backtracking means the block
// runs to the *last* line starting with "}" that still lets the whole file parse
// — which is why a JS object literal closed by `  });` mid-script is not a
// terminator while the bare `}` closing the script is. We reproduce that by
// choosing the last line that both begins with "}" (after no more than the
// block's own indent) and leaves a parseable remainder.
//
// Practical rule used here: candidate terminators are lines whose first
// non-space character is "}" and where the character after it is either nothing
// or a character that cannot continue an expression (`")"`, `")"` variants,
// whitespace). We then take the last candidate that leaves the rest of the file
// parseable; failing that, the first candidate.
func (p *parser) parseTextBody() (string, error) {
	start := p.pos

	// Collect candidate terminator offsets. The grammar is `tagend = nl "}"`,
	// so the closing brace must be the **first character of its line** — an
	// indented `  }` inside a script or JSON body is content, not a terminator.
	// This is the rule that makes nested braces in bodies work at all.
	type candidate struct{ nlPos, afterBrace int }
	var candidates []candidate

	for i := start; i < len(p.src); i++ {
		if p.src[i] != '\n' || i+1 >= len(p.src) {
			continue
		}
		if p.src[i+1] == '}' {
			candidates = append(candidates, candidate{nlPos: i, afterBrace: i + 2})
		}
	}

	if len(candidates) == 0 {
		return "", p.errorf("unterminated text block")
	}

	// Take the first column-0 "}" that leaves a parseable remainder. With the
	// column-0 rule, the first viable one is almost always correct; the retry
	// only matters for a block whose content itself contains a bare `}` line.
	chosen := -1
	for k := 0; k < len(candidates); k++ {
		c := candidates[k]
		rest := &parser{src: p.src, pos: c.afterBrace, line: p.lineAt(c.afterBrace), col: 1}
		if _, err := rest.parseDocument(); err == nil {
			chosen = k
			break
		}
	}
	if chosen < 0 {
		chosen = len(candidates) - 1
	}

	end := candidates[chosen].nlPos
	if end < start {
		end = start
	}
	// Drop the single newline immediately after the opening brace: Bruno's
	// grammar is `"{" nl* textblock`, so the newline is not part of the body.
	text := strings.TrimPrefix(p.src[start:end], "\n")

	// Advance the cursor to just after the terminator's "}".
	p.advanceTo(candidates[chosen].afterBrace)
	// Consume the rest of the closing line if any.
	if rest := strings.TrimSpace(p.readLine()); rest != "" {
		return "", p.errorf("unexpected %q after closing brace", rest)
	}

	return outdent(text, 2), nil
}

// lineAt returns the 1-based line number for a byte offset.
func (p *parser) lineAt(offset int) int {
	if offset > len(p.src) {
		offset = len(p.src)
	}
	n := 1
	for i := 0; i < offset; i++ {
		if p.src[i] == '\n' {
			n++
		}
	}
	return n
}

// advanceTo moves the cursor to offset, maintaining line and column.
func (p *parser) advanceTo(offset int) {
	for p.pos < offset && p.pos < len(p.src) {
		p.advance()
	}
}

// outdent removes up to n leading spaces from every line, matching Bruno's
// outdentString. Text bodies use 2; ”' multiline values use a different rule.
func outdent(s string, n int) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = trimIndent(line, n)
	}
	return strings.Join(lines, "\n")
}

// parseListBlock reads `name [ item item ]`.
func (p *parser) parseListBlock(name string, start Position, anns []Annotation) (*Block, error) {
	block := &Block{Name: blockName(name), Kind: ListBlock, Pos: start, Annotations: anns}
	for {
		p.skipAllWhitespace()
		if p.eof() {
			return nil, p.errorf("unterminated list block %s", strconvQuote(name))
		}
		if p.peek() == ']' {
			p.advance()
			return block, nil
		}
		if p.peek() == ',' || p.peek() == '\n' || p.peek() == ' ' || p.peek() == '\t' {
			p.advance()
			continue
		}
		item, err := p.parseQuotedOrBareUntil(",]\n")
		if err != nil {
			return nil, err
		}
		item = strings.TrimSpace(item)
		if item != "" {
			block.Items = append(block.Items, item)
		}
	}
}

// parseAnnotation reads @name, @name(arg), @name('arg'), @name("arg") or
// @name(”'arg”').
func (p *parser) parseAnnotation() (Annotation, error) {
	pos := p.position()
	if p.peek() != '@' {
		return Annotation{}, p.errorf("expected '@'")
	}
	p.advance()

	var name strings.Builder
	for !p.eof() {
		c := p.peek()
		if c == '(' || c == ')' || c == ' ' || c == '\t' || c == '\n' || c == ':' || c == '\r' {
			break
		}
		name.WriteByte(p.advance())
	}
	ann := Annotation{Name: name.String(), Pos: pos}
	if ann.Name == "" {
		return ann, p.errorf("empty annotation name")
	}

	if p.peek() != '(' {
		return ann, nil
	}
	p.advance() // '('
	ann.HasArg = true

	// Multiline argument.
	if strings.HasPrefix(p.src[p.pos:], "'''") {
		p.advance()
		p.advance()
		p.advance()
		var b strings.Builder
		for {
			if p.eof() {
				return ann, p.errorf("unterminated ''' annotation argument")
			}
			if strings.HasPrefix(p.src[p.pos:], "'''") {
				p.advance()
				p.advance()
				p.advance()
				break
			}
			b.WriteByte(p.advance())
		}
		ann.Arg = dedentMultiline(b.String())
		ann.ArgBlock = true
	} else if p.peek() == '\'' || p.peek() == '"' {
		quote := p.advance()
		var b strings.Builder
		for !p.eof() && p.peek() != quote {
			if quote == '"' && p.peek() == '\\' && p.pos+1 < len(p.src) {
				p.advance()
				b.WriteByte(p.advance())
				continue
			}
			b.WriteByte(p.advance())
		}
		if p.eof() {
			return ann, p.errorf("unterminated quoted annotation argument")
		}
		p.advance() // closing quote
		ann.Arg = b.String()
	} else {
		var b strings.Builder
		for !p.eof() && p.peek() != ')' {
			b.WriteByte(p.advance())
		}
		ann.Arg = b.String()
	}

	if p.peek() != ')' {
		return ann, p.errorf("expected ')' to close @%s", ann.Name)
	}
	p.advance()
	return ann, nil
}

// readLine returns the rest of the current line and consumes the newline.
func (p *parser) readLine() string {
	var b strings.Builder
	for !p.eof() && p.peek() != '\n' {
		b.WriteByte(p.advance())
	}
	if !p.eof() && p.peek() == '\n' {
		p.advance()
	}
	return b.String()
}

// readUntilOneOf returns text up to any byte in stops, without consuming it.
func (p *parser) readUntilOneOf(stops string) string {
	start := p.pos
	for !p.eof() && !strings.ContainsRune(stops, rune(p.peek())) {
		p.advance()
	}
	return p.src[start:p.pos]
}

// parseQuotedOrBareUntil reads a token, honouring quotes, up to a stop byte
// outside quotes.
func (p *parser) parseQuotedOrBareUntil(stops string) (string, error) {
	var b strings.Builder
	for !p.eof() {
		c := p.peek()
		if c == '"' || c == '\'' {
			quote := p.advance()
			for !p.eof() && p.peek() != quote {
				b.WriteByte(p.advance())
			}
			if p.eof() {
				return "", p.errorf("unterminated quoted value")
			}
			p.advance()
			continue
		}
		if strings.ContainsRune(stops, rune(c)) {
			return b.String(), nil
		}
		b.WriteByte(p.advance())
	}
	return b.String(), nil
}

// skipInlineSpace consumes spaces and tabs, not newlines.
func (p *parser) skipInlineSpace() {
	for !p.eof() && (p.peek() == ' ' || p.peek() == '\t') {
		p.advance()
	}
}

// skipAllWhitespace consumes spaces, tabs and newlines.
func (p *parser) skipAllWhitespace() {
	for !p.eof() {
		switch p.peek() {
		case ' ', '\t', '\n':
			p.advance()
		default:
			return
		}
	}
}

// consumeKeyword consumes an expected bare word.
func (p *parser) consumeKeyword(word string) {
	if strings.HasPrefix(p.src[p.pos:], word) {
		for range word {
			p.advance()
		}
	}
}
