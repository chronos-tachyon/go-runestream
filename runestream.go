package runestream

import (
	"io"
	"unicode/utf8"
)

const BlockSize = 4096

// savedRune represents a single Unicode character read from a byte stream.
type savedRune struct {
	pos   Position
	value rune
	size  int
	err   error
}

// RuneStream is an engine for lexing runes from a byte stream.  This version of
// RuneStream only understands UTF-8.
//
// Using RuneStream is conceptually similar to using ReadRune() / UnreadRune()
// from bufio.Reader, but RuneStream adds the ability to UnreadRune() an
// arbitrary number of times.  It also tracks the human-friendly position
// (lines and columns) of each character within the text file, and it has some
// convenience methods for extracting multi-rune sequences that match a
// pattern.
//
// Bare-bones usage:
//
//	func StreamRunes(filename string, callback func(rune, int, Position)) error {
//		f, err := os.Open("filename.txt")
//		if err != nil {
//			return err
//		}
//
//		stream := NewRuneStream(f)
//		var out []rune
//		for stream.Advance() {
//			r := stream.Rune()
//			size := stream.Size()
//			pos := stream.Position()
//			callback(r, size, pos)
//			stream.Commit()
//		}
//
//		err = stream.Err()
//		if err == io.EOF {
//			err = nil
//		}
//		return err
//	}
//
// Advanced usage:
//
//	func LexWordOrNumber(stream *RuneStream) Token {
//		if !stream.Advance() {
//			err := stream.Err()
//			return ErrorToken(err)
//		}
//		r := stream.Rune()
//		pos := stream.Position()
//		if unicode.IsLetter(r) {
//			word := []rune{r}
//			word = stream.TakeWhile(-1, word, unicode.IsLetter)
//			stream.Commit()
//			return WordToken(pos, string(word))
//		}
//		if unicode.IsDigit(r) {
//			number := []rune{r}
//			number = stream.TakeWhile(-1, number, unicode.IsDigit)
//			stream.Commit()
//			return NumberToken(pos, string(number))
//		}
//		stream.Rewind()
//		return FailToken()
//	}
//
type RuneStream struct {
	// r is the byte stream to read.
	r io.Reader

	// bb is a byte buffer of (slightly more than) length BlockSize that
	// will be reused as bytes are read from r.
	bb []byte

	// b is the slice of bb corresponding to the leftover bytes that have
	// been read from the Reader but not yet processed as runes.
	b []byte

	// pos is the current position within r, i.e. the position of the start
	// of the next savedRune to be read from r.
	pos Position

	// buf is the list of savedRunes that have been read from r.
	buf []savedRune

	// curr is the savedRune in buf that the caller is working on.
	curr *savedRune

	// gen is the generation number, incremented on each Commit().
	gen uint

	// spec is the speculative read count, which is an index into buf.
	spec uint
}

// SavePoint is a snapshot of a stream position.
type SavePoint struct {
	gen  uint
	spec uint
}

// NewRuneStream constructs a new RuneStream.
func NewRuneStream(r io.Reader) *RuneStream {
	return &RuneStream{
		r:   r,
		bb:  make([]byte, BlockSize+utf8.UTFMax),
		pos: MakePosition(),
	}
}

// Reset returns this RuneStream to the newly-constructed state.
//
// This is useful for saving some GC overhead when prelexing multiple byte
// streams.
//
func (stream *RuneStream) Reset(r io.Reader) {
	stream.r = r
	stream.b = nil
	stream.pos.Reset()
	stream.buf = nil
	stream.curr = nil
	stream.gen++
	stream.spec = 0
}

// Save creates a save point.
func (stream *RuneStream) Save() SavePoint {
	return SavePoint{stream.gen, stream.spec}
}

// Restore rewinds the character stream to the given save point.
func (stream *RuneStream) Restore(sp SavePoint) {
	if sp.gen != stream.gen {
		panic("save point is stale")
	}
	stream.spec = sp.spec
	stream.curr = nil
}

// Rewind rewinds the character stream to the last Commit() call.
func (stream *RuneStream) Rewind() {
	stream.spec = 0
	stream.curr = nil
}

// Commit tells the RuneStream that the caller will never need to rewind past
// this point, allowing the RuneStream to free resources.
//
// Each call to Commit() invalidates all save points.
//
func (stream *RuneStream) Commit() {
	stream.buf = stream.buf[stream.spec:]
	stream.gen++
	stream.spec = 0
	stream.curr = nil
}

// load reads the next block of runes from the byte stream.
func (stream *RuneStream) load() {
	if len(stream.buf) >= 0x40000000 {
		panic("too many calls to Advance() without Commit()")
	}

	x := len(stream.b)
	y := x + BlockSize
	copy(stream.bb[0:x], stream.b)
	n, err := stream.r.Read(stream.bb[x:y])
	stream.b = stream.bb[0 : x+n]
	for utf8.FullRune(stream.b) {
		r, size := utf8.DecodeRune(stream.b)
		stream.b = stream.b[size:]
		stream.buf = append(stream.buf, savedRune{
			pos:   stream.pos,
			value: r,
			size:  size,
		})
		stream.pos.Advance(r, size)
	}
	if err != nil {
		stream.buf = append(stream.buf, savedRune{
			pos: stream.pos,
			err: err,
		})
	}
}

// Advance moves forward in the stream, returning true if a new character is
// available or false if an I/O error (such as io.EOF) was encountered.
func (stream *RuneStream) Advance() bool {
	if stream.curr != nil && stream.curr.err != nil {
		return false
	}
	if stream.spec >= uint(len(stream.buf)) {
		stream.load()
	}
	stream.curr = &stream.buf[stream.spec]
	stream.spec++
	return stream.curr.err == nil
}

// Rune returns the character at the current stream position.
func (stream *RuneStream) Rune() rune {
	return stream.curr.value
}

// Size returns the number of bytes occupied by the character at the current
// stream position.
func (stream *RuneStream) Size() int {
	return stream.curr.size
}

// Position returns the position of the stream.
func (stream *RuneStream) Position() Position {
	return stream.curr.pos
}

// Err returns the I/O error encountered while reading the stream.
func (stream *RuneStream) Err() error {
	return stream.curr.err
}

// Take consumes one character, advancing the stream only if the next rune
// matches pred.
func (stream *RuneStream) Take(pred func(rune) bool) (rune, bool) {
	sp := stream.Save()
	if stream.Advance() && pred(stream.curr.value) {
		return stream.curr.value, true
	}
	stream.Restore(sp)
	return 0, false
}

// TakeWhile consumes zero or more characters, advancing the stream so long as
// pred returns true for each new rune.
//
// If max is negative, then the number of runes that can match is unbounded;
// otherwise, max is the upper limit on the number of runes matched.
//
func (stream *RuneStream) TakeWhile(max int, out []rune, pred func(rune) bool) []rune {
	sp := stream.Save()
	count := 0
	for max < 0 || count < max {
		if !stream.Advance() {
			break
		}
		if !pred(stream.curr.value) {
			break
		}
		count++
		out = append(out, stream.curr.value)
		sp = stream.Save()
	}
	stream.Restore(sp)
	return out
}

// TakeUntil consumes zero or more characters, advancing the stream until pred
// returns true for a rune.
//
// If max is negative, then the number of runes that can match is unbounded;
// otherwise, max is the upper limit on the number of runes matched.
//
func (stream *RuneStream) TakeUntil(max int, out []rune, pred func(rune) bool) []rune {
	return stream.TakeWhile(max, out, func(r rune) bool { return !pred(r) })
}
