package parser

import (
	"github.com/lus/fluent.go/fluent/parser/ast"
	"math"
	"strings"
	"unicode"
)

// Parser is used to parse a FTL source into an AST
type Parser struct {
	str *stream
}

// New creates a new FTL parser from a source string
func New(source string) *Parser {
	return &Parser{str: newStream(source)}
}

// Parse parses the underlying FTL source string into an AST.
// NOTE: This function returns all errors that occurred while parsing entries.
// This does not mean that the parsing failed at a whole.
func (parser *Parser) Parse() (*ast.Resource, []*Error) {
	// Blank space at the beginning of the file is ignored
	parser.skipBlankBlock()

	var errors []*Error
	entries := []ast.Node{}
	var lastComment *ast.Comment

	for parser.str.HasNext() {
		// Parse a new entry or junk.
		// Junk is content that could not be parsed due to an error
		entry, err := parser.parseEntryOrJunk()
		if err != nil {
			if pErr, ok := err.(*Error); ok {
				errors = append(errors, pErr)
			} else {
				errors = append(errors, newError(0, 0, err.Error()))
			}
			continue
		}

		// Blank space between entries is ignored
		blankBlock := parser.skipBlankBlock()

		// If the just parsed entry is a normal command we have to hold it until the next entry got parsed
		// as comments immediately before a message get attached to that message and are no standalone entries
		if comment, ok := entry.(*ast.Comment); ok && len(blankBlock) == 0 && parser.str.HasNext() {
			lastComment = comment
			continue
		}

		// If a comment preceded a message or term, attach it to it
		if lastComment != nil {
			if message, ok := entry.(*ast.Message); ok {
				message.Comment = lastComment
				message.Span[0] = lastComment.Span[0]
			} else if term, ok := entry.(*ast.Term); ok {
				term.Comment = lastComment
				term.Span[0] = lastComment.Span[0]
			} else {
				entries = append(entries, lastComment)
			}

			lastComment = nil
		}

		entries = append(entries, entry)
	}

	// Build the resource AST node
	return &ast.Resource{
		Base: ast.Base{
			Type: ast.TypeResource,
			Span: [2]uint{0, uint(parser.str.SrcLen())},
		},
		Body: entries,
	}, errors
}

// parseEntryOrJunk tries to parse a single entry node and turns it into a junk one if an error occurred while parsing it
func (parser *Parser) parseEntryOrJunk() (ast.Node, error) {
	start := parser.str.CurrentCursorPos()

	// Try to correctly parse an entry
	entry, err := parser.parseEntry()
	if entry != nil {
		err = parser.expect(EOL)
		if err == nil {
			return entry, nil
		}
	}

	// Check if there is an EOL after the one that started the broken entry and jump to it if there is one
	errorPos := parser.str.CurrentCursorPos()
	slice := parser.str.Src()[:errorPos]
	lastEOLRaw := strings.LastIndex(string(slice), "\n")
	// lastEOLRaw is the index in bytes; we have to calculate the rune index
	lastEOL := lastEOLRaw - (len(string(parser.str.Src())) - parser.str.SrcLen())
	if start < lastEOL {
		parser.str.SetCursorTo(lastEOL)
	}

	// Peek through the rest of the document and find the next EOL immediately followed by a character that may introduce a new entry
	cur := 0
	parser.str.PeekUntil(func(char rune) bool {
		if char != EOL {
			cur++
			return false
		}
		if !isEntryStart(parser.str.PeekNth(cur + 1)) {
			cur++
			return false
		}
		return true
	})
	parser.str.Skip(cur)

	// Extract the junk content
	nextEntryStart := parser.str.CurrentCursorPos()
	if nextEntryStart == len(parser.str.Src()) {
		nextEntryStart--
	}
	content := parser.str.Src()[start : nextEntryStart+1]

	// Build the junk AST node
	annotation := ""
	if err != nil {
		annotation = err.Error()
	}
	return &ast.Junk{
		Base: ast.Base{
			Type: ast.TypeJunk,
			Span: [2]uint{uint(start), uint(nextEntryStart)},
		},
		Content:     string(content),
		Annotations: []string{annotation},
	}, err
}

// parseEntry parses an entry node (comment, message or term)
func (parser *Parser) parseEntry() (ast.Node, error) {
	switch parser.str.Peek() {
	case '#':
		return parser.parseComment()
	case '-':
		return parser.parseTerm()
	default:
		return parser.parseMessage()
	}
}

// parseComment parses a comment node
func (parser *Parser) parseComment() (ast.Node, error) {
	start := uint(parser.str.CurrentCursorPos())

	level := -1
	content := ""

lines:
	for {
		// Decide which level the comment has ('#', '##' or '###')
		if level == -1 {
			offset := 0
			for parser.str.PeekNth(offset) == '#' && level < 2 {
				offset++
				level++
			}
		}
		parser.str.Skip(level + 1)

		peek := parser.str.Peek()
		if peek != EOL {
			// The '#'s have to be followed by a space
			if err := parser.expect(' '); err != nil {
				return nil, err
			}

			// Append the rest of the line to the content of the comment
			line := parser.str.PeekUntil(func(char rune) bool {
				return char == EOL
			})
			parser.str.Skip(len(line))
			content += string(line)
		}

		// Check if the next line is comment with the same level as the current one
		for i := 0; i <= level; i++ {
			char := parser.str.PeekNth(1 + i)
			if char != '#' {
				break lines
			}
		}

		// Check if the next line is also a valid comment (a space or EOL after the '#'s)
		next := parser.str.PeekNth(level + 2)
		if next != ' ' && next != EOL {
			break
		}

		// Add the EOL to the comment content to then append the next line in the next loop iteration
		content += string(EOL)
		parser.str.Skip(1)
	}

	end := uint(parser.str.CurrentCursorPos())

	// Build the AST node corresponding to the comment level
	switch level {
	case 0:
		return &ast.Comment{
			Base: ast.Base{
				Type: ast.TypeComment,
				Span: [2]uint{start, end},
			},
			Content: content,
		}, nil
	case 1:
		return &ast.GroupComment{
			Base: ast.Base{
				Type: ast.TypeGroupComment,
				Span: [2]uint{start, end},
			},
			Content: content,
		}, nil
	case 2:
		return &ast.ResourceComment{
			Base: ast.Base{
				Type: ast.TypeResourceComment,
				Span: [2]uint{start, end},
			},
			Content: content,
		}, nil
	default:
		panic("unreachable")
	}
}

// parseTerm parses a term node
func (parser *Parser) parseTerm() (*ast.Term, error) {
	start := uint(parser.str.CurrentCursorPos())

	// A '-' is expected
	if err := parser.expect('-'); err != nil {
		return nil, err
	}

	// Parse the identifier
	id, err := parser.parseIdentifier()
	if err != nil {
		return nil, err
	}

	// Whitespace before the '=' is ignored
	parser.skipBlankInline()

	// A '=' is expected
	if err := parser.expect('='); err != nil {
		return nil, err
	}

	// Parse the pattern value
	value, err := parser.parseOptionalPattern()
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, newError(start, uint(parser.str.CurrentCursorPos()), "a pattern is required for terms")
	}

	// Parse the attributes
	attributes, err := parser.parseAttributes()
	if err != nil {
		return nil, err
	}

	// Build the term AST node
	return &ast.Term{
		Base: ast.Base{
			Type: ast.TypeTerm,
			Span: [2]uint{start, uint(parser.str.CurrentCursorPos())},
		},
		ID:         id,
		Value:      value,
		Attributes: attributes,
		Comment:    nil,
	}, nil
}

// parseMessage parses a message node
func (parser *Parser) parseMessage() (*ast.Message, error) {
	start := uint(parser.str.CurrentCursorPos())

	// Parse the identifier
	id, err := parser.parseIdentifier()
	if err != nil {
		return nil, err
	}

	// Whitespace before the '=' is ignored
	parser.skipBlankInline()

	// A '=' is expected
	if err := parser.expect('='); err != nil {
		return nil, err
	}

	// Parse the (optional; attributes are enough) pattern value
	value, err := parser.parseOptionalPattern()
	if err != nil {
		return nil, err
	}

	// Parse the attributes
	var attrErr error
	beforeAttributes := parser.str.CurrentCursorPos()
	attributes, err := parser.parseAttributes()
	if err != nil {
		parser.str.SetCursorTo(beforeAttributes)
		attrErr = err
	}
	if attributes == nil {
		attributes = []*ast.Attribute{}
	}

	// Raise an error if no attributes and no pattern value could be parsed
	if value == nil && len(attributes) == 0 {
		return nil, newError(start, uint(parser.str.CurrentCursorPos()), "message entries may not be completely blank")
	}

	// Build the message AST node
	return &ast.Message{
		Base: ast.Base{
			Type: ast.TypeMessage,
			Span: [2]uint{start, uint(parser.str.CurrentCursorPos())},
		},
		ID:         id,
		Value:      value,
		Attributes: attributes,
		Comment:    nil,
	}, attrErr
}

// parseOptionalPattern parses a pattern if one exists, returns nil otherwise
func (parser *Parser) parseOptionalPattern() (*ast.Pattern, error) {
	// Retrieve the first non-empty character in the current line
	blank := parser.peekBlankInline()
	firstChar := parser.str.PeekNth(len(blank))

	// Return nothing if the file ends
	if firstChar == EOF {
		return nil, nil
	}

	// If the first non-empty character in the current line is no EOF and EOL, parse an inline-starting pattern
	if firstChar != EOL {
		parser.str.Skip(len(blank))
		return parser.parsePattern(false)
	}

	// Receive the first non-blank character
	blank, lenBlank := parser.peekBlankBlock()
	blankTargetLine := parser.str.PeekUntilWithOffset(lenBlank, func(char rune) bool {
		return char != ' '
	})
	first := parser.str.PeekNth(lenBlank + len(blankTargetLine))

	// If the first non-blank character is no '{' and is illegal or starts immediately after the EOL
	// (starting a new entry; indent is required), return nothing
	if first != '{' && (len(blankTargetLine) == 0 || anyOf(first, '}', '.', '[', '*')) {
		return nil, nil
	}

	// Skip to the first non-empty character and parse a block pattern
	parser.str.Skip(lenBlank)
	return parser.parsePattern(true)
}

// indent is an AST node used temporarily by parsePattern to format indents.
// It cannot be found in the final AST
type indent struct {
	ast.Base
	Value string
}

// parsePattern parses a pattern node
func (parser *Parser) parsePattern(block bool) (*ast.Pattern, error) {
	start := uint(parser.str.CurrentCursorPos())

	commonIndent := math.MaxInt
	var elements []ast.Node

	// If the multiline text block does not start in the same line as the identifier, its indent has to be considered
	if block {
		blank := parser.peekBlankInline()
		commonIndent = len(blank)
		parser.str.Skip(len(blank))
		elements = append(elements, &indent{
			Base: ast.Base{
				Type: "",
				Span: [2]uint{start, uint(parser.str.CurrentCursorPos())},
			},
			Value: string(blank),
		})
	}

	// Turn the pattern into a list of elements
	for parser.str.HasNext() {
		peek := parser.str.Peek()
		if peek == '{' {
			placeable, err := parser.parsePlaceable()
			if err != nil {
				return nil, err
			}
			elements = append(elements, placeable)
		} else if peek == '}' {
			pos := uint(parser.str.CurrentCursorPos())
			return nil, newError(pos, pos, "unexpected '}'")
		} else if peek == EOL {
			// Validate the indent and first character of the next line and skip all blank characters if the text block continues
			indentStart := uint(parser.str.CurrentCursorPos())
			blankBlock, lenBlankBlock := parser.peekBlankBlock()
			blankInline := parser.str.PeekUntilWithOffset(lenBlankBlock, func(char rune) bool {
				return char != ' '
			})
			first := parser.str.PeekNth(lenBlankBlock + len(blankInline))
			if first != '{' && (len(blankInline) == 0 || anyOf(first, '}', '.', '[', '*')) {
				break
			}
			commonIndent = minInt(commonIndent, len(blankInline))
			parser.str.Skip(lenBlankBlock + len(blankInline))

			// Append a temporary indent node to the element list
			elements = append(elements, &indent{
				Base: ast.Base{
					Type: "",
					Span: [2]uint{indentStart, uint(parser.str.CurrentCursorPos())},
				},
				Value: string(blankBlock) + string(blankInline),
			})
		} else {
			text, err := parser.parseText()
			if err != nil {
				return nil, err
			}
			elements = append(elements, text)
		}
	}

	// Process the temporary indent nodes and trim the elements according to the common indent shared between them
	trimmed := make([]ast.Node, 0, len(elements))
	for _, element := range elements {
		// Placeables don't get processed any further
		if placeable, ok := element.(*ast.Placeable); ok {
			trimmed = append(trimmed, placeable)
			continue
		}

		// Remove the common indent of every indent node
		if indent, ok := element.(*indent); ok {
			indent.Value = indent.Value[:len(indent.Value)-commonIndent]
			if len(indent.Value) == 0 {
				continue
			}
		}

		// Join consecutive text and indent nodes
		if len(trimmed) > 0 {
			previous := trimmed[len(trimmed)-1]
			if text, ok := previous.(*ast.Text); ok {
				var currentValue string
				var endSpan uint
				if cur, ok := element.(*ast.Text); ok {
					currentValue = cur.Value
					endSpan = cur.Span[1]
				} else if cur, ok := element.(*indent); ok {
					currentValue = cur.Value
					endSpan = cur.Span[1]
				}

				text.Value = text.Value + currentValue
				text.Span[1] = endSpan
				continue
			}
		}

		// Turn unjoined indent nodes into text ones (e.g. following a placeable)
		if in, ok := element.(*indent); ok {
			text := &ast.Text{
				Base: ast.Base{
					Type: ast.TypeText,
					Span: in.Span,
				},
				Value: in.Value,
			}
			element = text
		}

		trimmed = append(trimmed, element)
	}

	// Trim trailing whitespace of the last element if it is a text one. If it is empty afterwards, discard it
	if text, ok := trimmed[len(trimmed)-1].(*ast.Text); ok {
		text.Value = strings.TrimRightFunc(text.Value, func(char rune) bool {
			return char == ' '
		})
		if text.Value == "" {
			trimmed = trimmed[:len(trimmed)-1]
		}
	}

	// Build the pattern AST node
	return &ast.Pattern{
		Base: ast.Base{
			Type: ast.TypePattern,
			Span: [2]uint{start, uint(parser.str.CurrentCursorPos())},
		},
		Elements: trimmed,
	}, nil
}

// parseText parses a text node
func (parser *Parser) parseText() (*ast.Text, error) {
	start := uint(parser.str.CurrentCursorPos())

	// Write every valid text character into a buffer string
	buffer := ""
	for parser.str.HasNext() {
		peek := parser.str.Peek()
		if peek == '{' || peek == '}' {
			break
		}
		if peek == EOL {
			break
		}
		buffer += string(parser.str.Consume())
	}

	// Build the text AST node
	return &ast.Text{
		Base: ast.Base{
			Type: ast.TypeText,
			Span: [2]uint{start, uint(parser.str.CurrentCursorPos())},
		},
		Value: buffer,
	}, nil
}

// parsePlaceable parses a placeable node
func (parser *Parser) parsePlaceable() (*ast.Placeable, error) {
	start := uint(parser.str.CurrentCursorPos())

	// A '{' is required
	if err := parser.expect('{'); err != nil {
		return nil, err
	}

	// Any blank content after the '{' is ignored
	parser.skipBlank()

	// Parse the expression inside the placeable
	expression, err := parser.parseExpression()
	if err != nil {
		return nil, err
	}

	// A '}' afterwards is required
	if err := parser.expect('}'); err != nil {
		return nil, err
	}

	// Build the placeable AST node
	return &ast.Placeable{
		Base: ast.Base{
			Type: ast.TypePlaceable,
			Span: [2]uint{start, uint(parser.str.CurrentCursorPos())},
		},
		Expression: expression,
	}, nil
}

// parseExpression parses an expression node
func (parser *Parser) parseExpression() (ast.Node, error) {
	start := uint(parser.str.CurrentCursorPos())

	// Parse the inline expression which is the selector of a potential select expression at the same time
	selector, err := parser.parseInlineExpression()
	if err != nil {
		return nil, err
	}

	// Any blank content afterwards is ignored
	parser.skipBlank()

	// If the expression is no select expression, the just parsed inline expression is the actual expression and no selector
	if !(parser.str.Peek() == '-' && parser.str.PeekNth(1) == '>') {
		// Term attribute references are not allowed in placeables
		if term, ok := selector.(*ast.TermReference); ok && term.Attribute != nil {
			return nil, newError(start, uint(parser.str.CurrentCursorPos()), "term attribute references are not allowed in placeables")
		}
		return selector, nil
	}

	// Message references may not be used as select expression selectors
	if _, ok := selector.(*ast.MessageReference); ok {
		return nil, newError(start, uint(parser.str.CurrentCursorPos()), "message references are not allowed as selectors")
	}

	// Other placeables may not be used as select expression selectors
	if _, ok := selector.(*ast.Placeable); ok {
		return nil, newError(start, uint(parser.str.CurrentCursorPos()), "placeables are not allowed as selectors")
	}

	// Term references without an attribute may not be used as select expression selectors
	if term, ok := selector.(*ast.TermReference); ok && term.Attribute == nil {
		return nil, newError(start, uint(parser.str.CurrentCursorPos()), "normal term references are not allowed as selectors; consider using a term attribute reference instead")
	}

	// Skip the '->'
	parser.str.Skip(2)

	// Blank spaces (inline) after the '->' is ignored
	parser.skipBlankInline()

	// There may be no more non-empty content in the same line as the '->'
	if err := parser.expect(EOL); err != nil {
		return nil, err
	}

	// Parse the select variants
	variants, err := parser.parseVariants()
	if err != nil {
		return nil, err
	}

	// Build the select expression AST node
	return &ast.SelectExpression{
		Base: ast.Base{
			Type: ast.TypeSelectExpression,
			Span: [2]uint{start, uint(parser.str.CurrentCursorPos())},
		},
		Selector: selector,
		Variants: variants,
	}, nil
}

// parseInlineExpression parses an inline expression node
func (parser *Parser) parseInlineExpression() (ast.Node, error) {
	start := uint(parser.str.CurrentCursorPos())

	peek := parser.str.Peek()

	// If the next character is a '{', the expression is a placeable
	if peek == '{' {
		return parser.parsePlaceable()
	}

	// If the next character(s) introduce a valid number, parse a number literal
	if unicode.IsNumber(peek) || (peek == '-' && unicode.IsNumber(parser.str.PeekNth(1))) {
		return parser.parseNumber()
	}

	// If the next character is a quote, parse a string literal
	if peek == '"' {
		return parser.parseString()
	}

	// If the next character is a '$', parse a variable reference
	if peek == '$' {
		parser.str.Skip(1)
		identifier, err := parser.parseIdentifier()
		if err != nil {
			return nil, err
		}
		return &ast.VariableReference{
			Base: ast.Base{
				Type: ast.TypeVariableReference,
				Span: [2]uint{start, uint(parser.str.CurrentCursorPos())},
			},
			ID: identifier,
		}, nil
	}

	// If the next character is a '-', parse a term reference
	if peek == '-' {
		// Skip the '-' and parse the term identifier
		parser.str.Skip(1)
		identifier, err := parser.parseIdentifier()
		if err != nil {
			return nil, err
		}

		// Parse an optional attribute reference
		var attribute *ast.Identifier
		if parser.str.Peek() == '.' {
			parser.str.Skip(1)
			attribute, err = parser.parseIdentifier()
			if err != nil {
				return nil, err
			}
		}

		// As term arguments receive variables through call arguments, parse these if they are present
		var arguments *ast.CallArguments
		blank := parser.peekBlank()
		first := parser.str.PeekNth(len(blank))
		if first == '(' {
			parser.str.Skip(len(blank))
			arguments, err = parser.parseCallArguments()
			if err != nil {
				return nil, err
			}
		}

		// Build the term AST node
		return &ast.TermReference{
			Base: ast.Base{
				Type: ast.TypeTermReference,
				Span: [2]uint{start, uint(parser.str.CurrentCursorPos())},
			},
			ID:        identifier,
			Attribute: attribute,
			Arguments: arguments,
		}, nil
	}

	// We'll parse a message or function reference. In both cases a valid identifier has to be present
	if !isIdentifierStart(peek) {
		return nil, newError(start, uint(parser.str.CurrentCursorPos()), "no inline expression")
	}

	// Parse the actual identifier
	idStart := uint(parser.str.CurrentCursorPos())
	identifier, err := parser.parseIdentifier()
	if err != nil {
		return nil, err
	}

	// If the first non-space character after the identifier is a '(', we'll parse a function reference
	blank := parser.peekBlank()
	first := parser.str.PeekNth(len(blank))
	if first == '(' {
		// Function names have to be all-uppercase
		if hasLowercase([]rune(identifier.Name)) {
			return nil, newError(idStart, uint(parser.str.CurrentCursorPos()), "function names only may have uppercase letters")
		}

		// Blank content before the '(' is ignored
		parser.str.Skip(len(blank))

		// Parse the arguments to pass to the funtion
		arguments, err := parser.parseCallArguments()
		if err != nil {
			return nil, err
		}

		// Build the function reference AST node
		return &ast.FunctionReference{
			Base: ast.Base{
				Type: ast.TypeFunctionReference,
				Span: [2]uint{start, uint(parser.str.CurrentCursorPos())},
			},
			ID:        identifier,
			Arguments: arguments,
		}, nil
	}

	// We'll parse a message reference

	// Parse an optional attribute reference
	var attribute *ast.Identifier
	if parser.str.Peek() == '.' {
		parser.str.Skip(1)
		attribute, err = parser.parseIdentifier()
		if err != nil {
			return nil, err
		}
	}

	// Build the message reference AST node
	return &ast.MessageReference{
		Base: ast.Base{
			Type: ast.TypeMessageReference,
			Span: [2]uint{start, uint(parser.str.CurrentCursorPos())},
		},
		ID:        identifier,
		Attribute: attribute,
	}, nil
}

// parseCallArguments parses a call arguments node
func (parser *Parser) parseCallArguments() (*ast.CallArguments, error) {
	start := uint(parser.str.CurrentCursorPos())

	positional := []ast.Node{}
	named := []*ast.NamedArgument{}
	names := make(map[string]bool)

	// A '(' is required
	if err := parser.expect('('); err != nil {
		return nil, err
	}

	// Any blank content after the '(' is ignored
	parser.skipBlank()

	for {
		// If the next character is a ')', we are done
		if parser.str.Peek() == ')' {
			break
		}

		// Parse a single call argument
		argStart := uint(parser.str.CurrentCursorPos())
		argument, err := parser.parseCallArgument()
		if err != nil {
			return nil, err
		}

		// Ensure named arguments are only provided once and positional arguments are not specified after named ones
		if namedArg, ok := argument.(*ast.NamedArgument); ok {
			if names[namedArg.Name.Name] {
				return nil, newError(argStart, uint(parser.str.CurrentCursorPos()), "argument name already satisfied")
			}
			names[namedArg.Name.Name] = true
			named = append(named, namedArg)
		} else if len(named) > 0 {
			return nil, newError(argStart, uint(parser.str.CurrentCursorPos()), "positional arguments may not follow named ones")
		} else {
			positional = append(positional, argument)
		}

		// Any blank content after the argument data is ignored
		parser.skipBlank()

		// If the next character is a ',', another argument follows
		if parser.str.Peek() == ',' {
			parser.str.Skip(1)
			parser.skipBlank()
			continue
		}

		break
	}

	// A closing ')' is required
	if err := parser.expect(')'); err != nil {
		return nil, err
	}

	// Build the call arguments AST node
	return &ast.CallArguments{
		Base: ast.Base{
			Type: ast.TypeCallArguments,
			Span: [2]uint{start, uint(parser.str.CurrentCursorPos())},
		},
		Positional: positional,
		Named:      named,
	}, nil
}

// parseCallArgument parses a call argument node
func (parser *Parser) parseCallArgument() (ast.Node, error) {
	start := uint(parser.str.CurrentCursorPos())

	// Parse the expression that represents the identifier of a named argument or the value of a positional one
	expression, err := parser.parseInlineExpression()
	if err != nil {
		return nil, err
	}

	// Any blank content after the expression is ignored
	parser.skipBlank()

	// If the next character is no ':', the argument is positional
	if parser.str.Peek() != ':' {
		return expression, nil
	}

	// The name of a name argument has to be a valid identifier (message reference expression with no attributes)
	if exp, ok := expression.(*ast.MessageReference); !ok || exp.Attribute != nil {
		return nil, newError(start, uint(parser.str.CurrentCursorPos()), "argument name is no simple identifier")
	}

	// Skip the ':' and any blank content after it
	parser.str.Skip(1)
	parser.skipBlank()

	// Parse a literal as named arguments may only provide literals
	value, err := parser.parseLiteral()
	if err != nil {
		return nil, err
	}

	// Build the named argument AST node
	return &ast.NamedArgument{
		Base: ast.Base{
			Type: ast.TypeNamedArgument,
			Span: [2]uint{start, uint(parser.str.CurrentCursorPos())},
		},
		Name:  expression.(*ast.MessageReference).ID,
		Value: value,
	}, nil
}

// parseVariants parses a select variants node
func (parser *Parser) parseVariants() ([]*ast.Variant, error) {
	start := uint(parser.str.CurrentCursorPos())

	var variants []*ast.Variant
	setDefault := false

	// Blank content before the first variant is ignored
	parser.skipBlank()

	// Parse new variants as long as there are some remaining
	peek := parser.str.Peek()
	for peek == '[' || (peek == '*' && parser.str.PeekNth(1) == '[') {
		variantStart := uint(parser.str.CurrentCursorPos())

		// Ensure there is only one default variant
		isDefault := false
		if peek == '*' {
			if setDefault {
				return nil, newError(variantStart, variantStart, "only one default select variant is allowed")
			}
			setDefault = true
			isDefault = true
			parser.str.Skip(1)
			peek = parser.str.Peek()
		}

		// A '[' is required
		if err := parser.expect('['); err != nil {
			return nil, err
		}

		// Any blank content after the ']' is ignored
		parser.skipBlank()

		// Parse the key of the variant
		key, err := parser.parseVariantKey()
		if err != nil {
			return nil, err
		}

		// Any blank content after the key is ignored
		parser.skipBlank()

		// A closing ']' is required
		if err := parser.expect(']'); err != nil {
			return nil, err
		}

		// Parse the pattern that represents the variant's value
		pattern, err := parser.parseOptionalPattern()
		if err != nil {
			return nil, err
		}
		if pattern == nil {
			return nil, newError(variantStart, uint(parser.str.CurrentCursorPos()), "a value for the select variant is required")
		}

		// Build and append a new variant node
		variants = append(variants, &ast.Variant{
			Base: ast.Base{
				Type: ast.TypeVariant,
				Span: [2]uint{variantStart, uint(parser.str.CurrentCursorPos())},
			},
			Key:     key,
			Value:   pattern,
			Default: isDefault,
		})

		// An EOL is required after the variant pattern
		if err := parser.expect(EOL); err != nil {
			return nil, err
		}

		// Any blank content afterwards is ignored
		parser.skipBlank()

		peek = parser.str.Peek()
	}

	// Ensure at least one variant was provided
	if len(variants) == 0 {
		return nil, newError(start, uint(parser.str.CurrentCursorPos()), "at least one variant is required")
	}

	// A default variant is also required
	if !setDefault {
		return nil, newError(start, uint(parser.str.CurrentCursorPos()), "a default variant is required")
	}

	return variants, nil
}

// parseVariantKey parses a variant key node
func (parser *Parser) parseVariantKey() (ast.Node, error) {
	peek := parser.str.Peek()

	// An EOL is not allowed
	if peek == EOL {
		pos := uint(parser.str.CurrentCursorPos())
		return nil, newError(pos, pos, "no variant key was given")
	}

	// Parse a number if the variant key starts with a digit or '-'
	if unicode.IsNumber(peek) || peek == '-' {
		return parser.parseNumber()
	}

	// Parse an identifier otherwise
	return parser.parseIdentifier()
}

// ParseAttributes parses an attributes node
func (parser *Parser) parseAttributes() ([]*ast.Attribute, error) {
	attributes := []*ast.Attribute{}

	blank := parser.peekBlank()
	first := parser.str.PeekNth(len(blank))
	for first == '.' {
		parser.str.Skip(len(blank))

		// Parse and append a single attribute
		attribute, err := parser.parseAttribute()
		if err != nil {
			return nil, err
		}
		attributes = append(attributes, attribute)

		blank = parser.peekBlank()
		first = parser.str.PeekNth(len(blank))
	}

	return attributes, nil
}

// parseAttribute parses an attribute node
func (parser *Parser) parseAttribute() (*ast.Attribute, error) {
	start := uint(parser.str.CurrentCursorPos())

	// An attribute key has to start with a '.'
	if err := parser.expect('.'); err != nil {
		return nil, err
	}

	// Parse the identifier after the '.'
	identifier, err := parser.parseIdentifier()
	if err != nil {
		return nil, err
	}

	// Any blank inline content after the key is ignored
	parser.skipBlankInline()

	// A '=' is ignored
	if err := parser.expect('='); err != nil {
		return nil, err
	}

	// Parse the pattern
	value, err := parser.parseOptionalPattern()
	if err != nil {
		return nil, err
	}
	if value == nil {
		return nil, newError(start, uint(parser.str.CurrentCursorPos()), "a value for the attribute is required")
	}

	// Build the attribute AST node
	return &ast.Attribute{
		Base: ast.Base{
			Type: ast.TypeAttribute,
			Span: [2]uint{start, uint(parser.str.CurrentCursorPos())},
		},
		ID:    identifier,
		Value: value,
	}, nil
}

// parseLiteral parses a literal node
func (parser *Parser) parseLiteral() (ast.Node, error) {
	peek := parser.str.Peek()

	// Parse a number literal if the next character is a digit or '-'
	if unicode.IsNumber(peek) || peek == '-' {
		return parser.parseNumber()
	}

	// Parse a string literal if the next character is a quote
	if peek == '"' {
		return parser.parseString()
	}

	pos := uint(parser.str.CurrentCursorPos())
	return nil, newError(pos, pos, "invalid literal beginning (-, 0-9 or \" required)")
}

// parseNumber parses a number node
func (parser *Parser) parseNumber() (*ast.NumberLiteral, error) {
	start := uint(parser.str.CurrentCursorPos())

	raw := ""

	// If the next character is a '-', append it
	if parser.str.Peek() == '-' {
		raw += string(parser.str.Consume())
	}

	// While any digits are remaining, append them
	for unicode.IsNumber(parser.str.Peek()) {
		raw += string(parser.str.Consume())
	}

	// Go on if the number is a decimal
	if parser.str.Peek() == '.' {
		raw += string(parser.str.Consume())
		hasDecimal := false
		for unicode.IsNumber(parser.str.Peek()) {
			if !hasDecimal {
				hasDecimal = true
			}
			raw += string(parser.str.Consume())
		}
		if !hasDecimal {
			pos := uint(parser.str.CurrentCursorPos())
			return nil, newError(pos, pos, "no numbers after the decimal point")
		}
	}

	// Return the number AST node
	return &ast.NumberLiteral{
		Base: ast.Base{
			Type: ast.TypeNumberLiteral,
			Span: [2]uint{start, uint(parser.str.CurrentCursorPos())},
		},
		Value: raw,
	}, nil
}

// parseString parses a string node
func (parser *Parser) parseString() (*ast.StringLiteral, error) {
	start := uint(parser.str.CurrentCursorPos())

	// A '"' is required
	if err := parser.expect('"'); err != nil {
		return nil, err
	}

	// Append any following valid character
	buffer := ""
	for parser.str.HasNext() && parser.str.Peek() != '"' && parser.str.Peek() != EOL {
		if parser.str.Peek() == '\\' {
			seq, err := parser.parseEscapeSequence()
			if err != nil {
				return nil, err
			}
			buffer += seq
		} else {
			buffer += string(parser.str.Consume())
		}
	}

	// A closing '"' is required
	if err := parser.expect('"'); err != nil {
		return nil, err
	}

	// Build the string AST node
	return &ast.StringLiteral{
		Base: ast.Base{
			Type: ast.TypeStringLiteral,
			Span: [2]uint{start, uint(parser.str.CurrentCursorPos())},
		},
		Value: buffer,
	}, nil
}

func (parser *Parser) parseEscapeSequence() (string, error) {
	// A leading '\' is required
	if err := parser.expect('\\'); err != nil {
		return "", err
	}

	// Decide which escape sequence to use
	peek := parser.str.Peek()
	switch peek {
	case '\\', '"':
		return "\\" + string(parser.str.Consume()), nil
	case 'u':
		return parser.parseUnicodeEscapeSequence(false)
	case 'U':
		return parser.parseUnicodeEscapeSequence(true)
	default:
		pos := uint(parser.str.CurrentCursorPos())
		return "", newError(pos, pos, "unknown escape sequence")
	}
}

// parseUnicodeEscapeSequence parses a unicode escape sequence
func (parser *Parser) parseUnicodeEscapeSequence(sixDigits bool) (string, error) {
	// Define the amount of digits to parse and the character to expect
	char := 'u'
	digits := 4
	if sixDigits {
		char = 'U'
		digits = 6
	}

	// Expect the character ('u' or 'U')
	if err := parser.expect(char); err != nil {
		return "", err
	}

	// Append the valid characters
	raw := "\\" + string(char)
	for i := 0; i < digits; i++ {
		peek := parser.str.Peek()
		if !((peek >= '0' && peek <= '9') || (peek >= 'a' && peek <= 'f') || (peek >= 'A' && peek <= 'F')) {
			pos := uint(parser.str.CurrentCursorPos())
			return "", newError(pos, pos, "no valid HEX character (0-9a-fA-F)")
		}
		raw += string(parser.str.Consume())
	}

	return raw, nil
}

// parseIdentifier parses an identifier node
func (parser *Parser) parseIdentifier() (*ast.Identifier, error) {
	start := uint(parser.str.CurrentCursorPos())

	id := ""

	// Validate and append the starting character (a-zA-Z only)
	startChar := parser.str.Peek()
	if !isIdentifierStart(startChar) {
		return nil, newError(start, start, "invalid identifier start character (only a-zA-Z are allowed)")
	}
	id += string(startChar)
	parser.str.Skip(1)

	// Append any following valid character
	for {
		peek := parser.str.Peek()
		if !isIdentifierFollowing(peek) {
			break
		}
		id += string(peek)
		parser.str.Skip(1)
	}

	end := uint(parser.str.CurrentCursorPos())

	// Build the identifier AST node
	return &ast.Identifier{
		Base: ast.Base{
			Type: ast.TypeIdentifier,
			Span: [2]uint{start, end},
		},
		Name: id,
	}, nil
}

// peekBlankInline peeks until a character is found that is no space
func (parser *Parser) peekBlankInline() []rune {
	blank := parser.str.PeekUntil(func(char rune) bool {
		return char != ' '
	})
	return blank
}

// skipBlankInline moves the stream cursor forward until a character is found that is no space
func (parser *Parser) skipBlankInline() []rune {
	blank := parser.peekBlankInline()
	parser.str.Skip(len(blank))
	return blank
}

// peekBlankBlock peeks until a line is found that contains a character that is no space and no line ending
func (parser *Parser) peekBlankBlock() ([]rune, int) {
	blank := ""
	offset := 0
	for {
		blankInline := parser.str.PeekUntilWithOffset(offset, func(char rune) bool {
			return char != ' '
		})
		if parser.str.PeekNth(offset+len(blankInline)) == EOL {
			blank += string(EOL)
			offset += len(blankInline) + 1
		} else {
			break
		}
	}
	return []rune(blank), offset
}

// skipBlankBlock moves the stream cursor forward until a line is found that contains a character that is no space and no line ending
func (parser *Parser) skipBlankBlock() []rune {
	blank, blankLen := parser.peekBlankBlock()
	parser.str.Skip(blankLen)
	return blank
}

// peekBlank peeks until a character is found that is no space and no line ending
func (parser *Parser) peekBlank() []rune {
	blank := parser.str.PeekUntil(func(char rune) bool {
		return char != ' ' && char != EOL
	})
	return blank
}

// peekBlank moves the stream cursor forward until a character is found that is no space and no line ending
func (parser *Parser) skipBlank() []rune {
	blank := parser.peekBlank()
	parser.str.Skip(len(blank))
	return blank
}

// expect expects a sequence of runes
func (parser *Parser) expect(runes ...rune) error {
	if len(runes) == 1 && runes[0] == EOL && parser.str.Peek() == EOF {
		return nil
	}
	found := 0
	for _, char := range runes {
		if parser.str.PeekNth(found) != char {
			pos := uint(parser.str.CurrentCursorPos())
			return newError(pos, pos, "'%s' expected", string(char))
		}
		found++
	}
	parser.str.Skip(found)
	return nil
}
