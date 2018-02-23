package runestream

import (
	"unicode/utf8"
)

// Decoder is the interface for alternative charset decoders.
type Decoder interface {
	// Name returns the name of the charset.
	Name() string

	// Max returns the maximum number of bytes per rune.
	Max() int

	// FullRune returns true iff p contains all the bytes needed to decode
	// the next rune.
	FullRune(p []byte) bool

	// DecodeRune returns the next rune in p, and the number of bytes used
	// to represent it.
	DecodeRune(p []byte) (rune, int)
}

// UTF8Decoder implements Decoder for UTF-8.
type UTF8Decoder struct{}

var _ Decoder = UTF8Decoder{}

// Name fulfills the Decoder interface.
func (UTF8Decoder) Name() string { return "utf-8" }

// Max fulfills the Decoder interface.
func (UTF8Decoder) Max() int { return utf8.UTFMax }

// FullRune fulfills the Decoder interface.
func (UTF8Decoder) FullRune(p []byte) bool { return utf8.FullRune(p) }

// DecodeRune fulfills the Decoder interface.
func (UTF8Decoder) DecodeRune(p []byte) (rune, int) { return utf8.DecodeRune(p) }
