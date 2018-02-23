package runestream

import (
	"fmt"
)

// Position represents a position within a text file.
type Position struct {
	Offset     uint64
	Line       uint64
	Column     uint64
	SkipNextLF bool
}

// MakePosition returns the Position for the start of a text file.
func MakePosition() Position {
	return Position{Line: 1, Column: 1}
}

// Reset sets this position to the start of the file.
func (pos *Position) Reset() {
	*pos = MakePosition()
}

// Advance updates the position, given the character encountered at the current
// position and the number of bytes that were used to encode it.
//
// For UTF-8 files, the ch and size arguments are usually the values returned
// by "unicode/utf8".DecodeRune.  However, this method does not care which
// encoding was actually used.
//
func (pos *Position) Advance(ch rune, size int) {
	if size < 0 {
		panic("negative size")
	}
	if size == 0 {
		return
	}

	pos.Offset += uint64(size)
	if ch == '\r' {
		pos.Line++
		pos.Column = 1
		pos.SkipNextLF = true
	} else if ch == '\n' && pos.SkipNextLF {
		pos.SkipNextLF = false
	} else if ch == '\n' {
		pos.Line++
		pos.Column = 1
	} else if ch == '\t' {
		tabwidth := 8 - ((pos.Column - 1) % 8)
		pos.Column += tabwidth
		pos.SkipNextLF = false
	} else {
		pos.Column++
		pos.SkipNextLF = false
	}
}

func (pos Position) String() string {
	return fmt.Sprintf("line %d column %d (byte offset %d)", pos.Line, pos.Column, pos.Offset)
}
