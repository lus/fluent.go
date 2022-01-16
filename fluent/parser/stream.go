package parser

const (
	EOF rune = -1
	EOL rune = '\n'
)

// stream is used by the parser to navigate through the source
type stream struct {
	source    []rune
	sourceLen int
	curPos    int
}

// newStream creates a new stream from a source string
func newStream(source string) *stream {
	src := []rune(source)
	return &stream{
		source:    src,
		sourceLen: len(src),
		curPos:    0,
	}
}

// Src returns the underlying source rune array
func (str *stream) Src() []rune {
	return str.source
}

// SrcLen returns the length of the underlying source rune array
func (str *stream) SrcLen() int {
	return str.sourceLen
}

// HasNext returns whether there are characters left in the source
func (str *stream) HasNext() bool {
	return str.curPos < str.sourceLen
}

// CurrentCursorPos returns the current cursor position
func (str *stream) CurrentCursorPos() int {
	return str.curPos
}

// SetCursorTo sets the cursor to a specific position.
// NOTE: This does not respect CRLF sequences!
func (str *stream) SetCursorTo(i int) {
	str.curPos = i
}

// Consume returns the next character and moves the cursor forward.
// If there are no more characters left, EOF is returned.
// If the next character is a CRLF sequence, a normal LF EOL is returned and the CR is skipped over.
func (str *stream) Consume() rune {
	if !str.HasNext() {
		return EOF
	}
	if str.source[str.curPos] == '\r' && (str.curPos+1 < str.sourceLen && str.source[str.curPos+1] == '\n') {
		str.curPos++
	}
	next := str.source[str.curPos]
	str.curPos++
	return next
}

// Skip moves the cursor forward n positions.
// If n is zero or less, nothing is done.
// If the target index is bigger than the length of the underlying source rune array, the cursor moves to the end.
// If a CRLF sequence is found along the way, it only counts as 1 skip.
func (str *stream) Skip(n int) {
	if n <= 0 {
		return
	}
	skipped := 0
	for skipped < n {
		target := str.curPos + 1
		if target >= str.sourceLen {
			str.curPos = str.sourceLen
			return
		}

		if str.source[str.curPos] == '\r' && str.source[str.curPos+1] == '\n' {
			target++
		}
		if target < str.sourceLen-1 && str.source[target] == '\r' && str.source[target+1] == '\n' {
			target++
		}

		skipped++
		str.curPos = target
	}
}

// Peek returns the next character, not moving the cursor forward.
// If there are no more characters left, EOF is returned.
// If the next character is a CRLF sequence, a normal LF EOL is returned and the CR is skipped over.
func (str *stream) Peek() rune {
	if !str.HasNext() {
		return EOF
	}
	if str.source[str.curPos] == '\r' && (str.curPos+1 < str.sourceLen && str.source[str.curPos+1] == '\n') {
		return EOL
	}
	return str.source[str.curPos]
}

// PeekN returns the next n characters, not moving the cursor forward.
// If there are no more characters left, an empty rune array is returned.
// If a character is a CRLF sequence, a normal LF EOL is appended to the rune array and the CR is skipped over.
func (str *stream) PeekN(n int) []rune {
	if !str.HasNext() {
		return []rune{}
	}

	runes := make([]rune, 0, n)
	acc := 0
	for i := 0; i < n; i++ {
		index := str.curPos + acc
		if index >= str.sourceLen {
			break
		}
		if str.source[index] == '\r' && (index+1 < str.sourceLen && str.source[index+1] == '\n') {
			index++
			acc++
		}
		acc++
		runes = append(runes, str.source[index])
	}

	return runes
}

// PeekNth returns the nth character from the current position; 0 being the current one (equal to calling Peek).
// If n points to a position outside the range of the underlying source rune array, an EOF is returned.
// If n points to the CR of a CRLF sequence, the LF is returned instead.
func (str *stream) PeekNth(n int) rune {
	if n <= 0 {
		return str.Peek()
	}

	rune := EOF
	nth := 0
	offset := 0
	for nth <= n {
		index := str.curPos + offset
		if index >= str.sourceLen {
			return EOF
		}
		if str.source[index] == '\r' && (index+1 < str.sourceLen && str.source[index+1] == '\n') {
			index++
			offset++
		}
		offset++
		nth++
		rune = str.source[index]
	}
	return rune
}

// PeekUntilWithOffset peeks and returns the next characters after the given offset until a character matches the terminator (this character is excluded).
// If the terminator did not match any character when EOF is reached, the rune array contains the rest of the file content.
// If a CRLF sequence is reached, only the LF is given to the terminator and potentially added to the rune array.
func (str *stream) PeekUntilWithOffset(offset int, terminator func(char rune) bool) []rune {
	// We have to normalize the offset first (CRLF sequences only count as one character)
	nth := 0
	skip := 0
	for nth < offset && offset != 0 {
		index := str.curPos + skip
		if index >= str.sourceLen {
			return []rune{}
		}
		if str.source[index] == '\r' && (index+1 < str.sourceLen && str.source[index+1] == '\n') {
			skip++
		}
		skip++
		nth++
	}

	runes := []rune{}
	acc := 0
	for {
		index := str.curPos + skip + acc
		if index >= str.sourceLen {
			break
		}

		crlf := false
		if str.source[index] == '\r' && (index+1 < str.sourceLen && str.source[index+1] == '\n') {
			crlf = true
			index++
		}

		if terminator(str.source[index]) {
			break
		}

		if crlf {
			acc++
		}
		acc++
		runes = append(runes, str.source[index])
	}
	return runes
}

// PeekUntil peeks and returns the next characters until a character matches the terminator (this character is excluded).
// If the terminator did not match any character when EOF is reached, the rune array contains a single EOF at the end.
// If a CRLF sequence is reached, only the LF is given to the terminator and potentially added to the rune array.
func (str *stream) PeekUntil(terminator func(char rune) bool) []rune {
	return str.PeekUntilWithOffset(0, terminator)
}
