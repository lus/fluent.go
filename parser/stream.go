package parser

// stream is used by the parser to navigate through the source.
// Basically it is a string character iterator but should not be seen too generic as it has FTL-specific behaviours.
type stream struct {
	source string
	curPos int
}

// newStream creates a new stream from a source string
func newStream(source string) *stream {
	return &stream{
		source: source,
		curPos: 0,
	}
}

// HasNext returns whether there are characters left to consume
func (str *stream) HasNext() bool {
	return str.curPos < len(str.source)
}

// PeekN returns the next n characters without moving the cursor forward.
// If n is zero or less, nil is returned.
// If n is bigger than the amount of remaining characters, everything is returned.
// If there are no characters left, nil is returned.
func (str *stream) PeekN(n int) []rune {
	if n <= 0 {
		return nil
	}
	end := str.curPos + n
	if end > len(str.source) {
		end = len(str.source)
	}
	if end == str.curPos {
		return nil
	}
	return []rune(str.source[str.curPos:end])
}

// Peek returns the next character without moving the cursor forward.
// If no characters are left, 0 is returned.
func (str *stream) Peek() rune {
	peek := str.PeekN(1)
	if peek == nil {
		return 0
	}
	return peek[0]
}

// ConsumeN returns the next n characters and moves the cursor forward.
// If n is zero or less, nil is returned and the cursor is not moved.
// If n is bigger than the amount of remaining characters, everything is returned and the cursor is moved to the end of the source.
// If no characters are left, nil is returned and the cursor is not moved.
func (str *stream) ConsumeN(n int) []rune {
	peek := str.PeekN(n)
	if peek == nil {
		return nil
	}
	str.curPos += len(peek)
	return peek
}

// Consume returns the next character and moves the cursor forward.
// If no characters are left, 0 is returned and the cursor is not moved.
func (str *stream) Consume() rune {
	consumed := str.ConsumeN(1)
	if consumed == nil {
		return 0
	}
	return consumed[0]
}
