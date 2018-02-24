package runestream

import (
	"bytes"
	"io"
	"testing"
	"strings"
	"unicode"
)

type Item struct {
	Rune rune
	Size int
	Pos  Position
}

func At(offset, line, column uint64) Position {
	return Position{
		Offset: offset,
		Line:   line,
		Column: column,
	}
}

func AtSkip(offset, line, column uint64) Position {
	pos := At(offset, line, column)
	pos.SkipNextLF = true
	return pos
}

func TestRuneStream_loop(t *testing.T) {
	var o Options
	var stream RuneStream

	r := strings.NewReader("English\r\nespañol\r\n日本語\r\n")
	stream.Init(r, o)

	var actual []Item
	for stream.Advance() {
		actual = append(actual, Item{
			Rune: stream.Rune(),
			Size: stream.Size(),
			Pos:  stream.Position(),
		})
		stream.Commit()
	}
	if stream.Err() != io.EOF {
		t.Errorf("unexpected error: %v", stream.Err())
	}

	expected := []Item{
		Item{Rune: 'E', Size: 1, Pos: At(0, 1, 1)},
		Item{Rune: 'n', Size: 1, Pos: At(1, 1, 2)},
		Item{Rune: 'g', Size: 1, Pos: At(2, 1, 3)},
		Item{Rune: 'l', Size: 1, Pos: At(3, 1, 4)},
		Item{Rune: 'i', Size: 1, Pos: At(4, 1, 5)},
		Item{Rune: 's', Size: 1, Pos: At(5, 1, 6)},
		Item{Rune: 'h', Size: 1, Pos: At(6, 1, 7)},
		Item{Rune: '\r', Size: 1, Pos: At(7, 1, 8)},
		Item{Rune: '\n', Size: 1, Pos: AtSkip(8, 2, 1)},
		Item{Rune: 'e', Size: 1, Pos: At(9, 2, 1)},
		Item{Rune: 's', Size: 1, Pos: At(10, 2, 2)},
		Item{Rune: 'p', Size: 1, Pos: At(11, 2, 3)},
		Item{Rune: 'a', Size: 1, Pos: At(12, 2, 4)},
		Item{Rune: 'ñ', Size: 2, Pos: At(13, 2, 5)},
		Item{Rune: 'o', Size: 1, Pos: At(15, 2, 6)},
		Item{Rune: 'l', Size: 1, Pos: At(16, 2, 7)},
		Item{Rune: '\r', Size: 1, Pos: At(17, 2, 8)},
		Item{Rune: '\n', Size: 1, Pos: AtSkip(18, 3, 1)},
		Item{Rune: '日', Size: 3, Pos: At(19, 3, 1)},
		Item{Rune: '本', Size: 3, Pos: At(22, 3, 2)},
		Item{Rune: '語', Size: 3, Pos: At(25, 3, 3)},
		Item{Rune: '\r', Size: 1, Pos: At(28, 3, 4)},
		Item{Rune: '\n', Size: 1, Pos: AtSkip(29, 4, 1)},
	}

	min := len(actual)
	if min > len(expected) {
		min = len(expected)
	}
	for i := 0; i < min; i++ {
		a := expected[i]
		b := actual[i]
		if a == b {
			continue
		}
		t.Errorf("[%02d] expected %+v, got %+v", i, a, b)
	}
	if len(actual) > min {
		t.Errorf("%d extra item(s)", len(actual)-min)
	}
	if len(expected) > min {
		t.Errorf("%d missing item(s)", len(expected)-min)
	}
}

func TestRuneStream_boundary(t *testing.T) {
	input := make([]byte, 0, 8192+64)
	var written uint
	for len(input) < 8192 {
		input = append(input, 0xe6, 0x97, 0xa5, 0xe6, 0x9c, 0xac, 0xe8, 0xaa, 0x9e)
		written += 3
	}

	runes := []rune{'日', '本', '語'}
	var count uint
	pos := MakePosition()
	stream := NewRuneStream(bytes.NewReader(input))
	for stream.Advance() {
		index := (count % 3)
		count++
		expected := Item{
			Rune: runes[index],
			Size: 3,
			Pos:  pos,
		}
		pos.Advance(runes[index], 3)

		actual := Item{
			Rune: stream.Rune(),
			Size: stream.Size(),
			Pos:  stream.Position(),
		}
		stream.Commit()

		if actual == expected {
			continue
		}
		t.Errorf("[%04d] expected %+v, got %+v", count, expected, actual)
	}
	if stream.Err() != io.EOF {
		t.Errorf("unexpected error: %v", stream.Err())
	}
	if count != written {
		t.Errorf("wrote %d runes, only read back %d runes", written, count)
	}
}

func TestRuneStream_Take(t *testing.T) {
	var o Options
	var stream RuneStream

	type tuple struct {
		r0  rune
		ok0 bool
		r1  rune
		ok1 bool
		r2  rune
		ok2 bool
	}

	r := strings.NewReader("abc")
	stream.Init(r, o)
	expected := tuple{'a', true, 'b', true, 'c', true}
	var actual tuple
	actual.r0, actual.ok0 = stream.Take(func(ch rune) bool { return ch == 'a' })
	actual.r1, actual.ok1 = stream.Take(func(ch rune) bool { return ch == 'b' })
	actual.r2, actual.ok2 = stream.Take(func(ch rune) bool { return ch == 'c' })
	if expected != actual {
		t.Errorf("expected %+v, got %+v", expected, actual)
	}

	r = strings.NewReader("abx")
	stream.Init(r, o)
	expected = tuple{'a', true, 'b', true, 0, false}
	actual.r0, actual.ok0 = stream.Take(func(ch rune) bool { return ch == 'a' })
	actual.r1, actual.ok1 = stream.Take(func(ch rune) bool { return ch == 'b' })
	actual.r2, actual.ok2 = stream.Take(func(ch rune) bool { return ch == 'c' })
	if expected != actual {
		t.Errorf("expected %+v, got %+v", expected, actual)
	}

	r = strings.NewReader("ac")
	stream.Init(r, o)
	expected = tuple{'a', true, 0, false, 'c', true}
	actual.r0, actual.ok0 = stream.Take(func(ch rune) bool { return ch == 'a' })
	actual.r1, actual.ok1 = stream.Take(func(ch rune) bool { return ch == 'b' })
	actual.r2, actual.ok2 = stream.Take(func(ch rune) bool { return ch == 'c' })
	if expected != actual {
		t.Errorf("expected %+v, got %+v", expected, actual)
	}
}

func TestRuneStream_TakeWhile(t *testing.T) {
	var o Options
	var stream RuneStream

	r := strings.NewReader("123a")
	stream.Init(r, o)
	digits := stream.TakeWhile(-1, nil, unicode.IsDigit)
	actual := string(digits)
	expected := "123"
	if expected != actual {
		t.Errorf("expected %q, got %q", expected, actual)
	}

	r = strings.NewReader("123a")
	stream.Init(r, o)
	digits = []rune{'!'}
	digits = stream.TakeWhile(-1, digits, unicode.IsDigit)
	actual = string(digits)
	expected = "!123"
	if expected != actual {
		t.Errorf("expected %q, got %q", expected, actual)
	}

	r = strings.NewReader("123456")
	stream.Init(r, o)
	digits = []rune{'!'}
	digits = stream.TakeWhile(3, digits, unicode.IsDigit)
	actual = string(digits)
	expected = "!123"
	if expected != actual {
		t.Errorf("expected %q, got %q", expected, actual)
	}
}

func TestRuneStream_SaveRestore(t *testing.T) {
	var o Options
	var stream RuneStream

	var output bytes.Buffer

	consumeWord := func(ch0, ch1, ch2 rune) bool {
		sp := stream.Save()
		r0, ok0 := stream.Take(func(ch rune) bool { return ch == ch0 })
		r1, ok1 := stream.Take(func(ch rune) bool { return ch == ch1 })
		r2, ok2 := stream.Take(func(ch rune) bool { return ch == ch2 })
		if ok0 && ok1 && ok2 {
			output.WriteRune(r0)
			output.WriteRune(r1)
			output.WriteRune(r2)
			return true
		}
		stream.Restore(sp)
		return false
	}

	r := strings.NewReader("foobaz")
	stream.Init(r, o)
	foundFoo := consumeWord('f', 'o', 'o')
	foundBar := consumeWord('b', 'a', 'r')
	foundBaz := consumeWord('b', 'a', 'z')
	if !foundFoo {
		t.Errorf("foo failed")
	}
	if foundBar {
		t.Errorf("bar accidentally succeeded")
	}
	if !foundBaz {
		t.Errorf("baz failed")
	}
}

type ZeroReader struct {
	zeroes [4096]byte
}

func (r *ZeroReader) Read(p []byte) (int, error) {
	n := len(p)
	for len(p) > 4096 {
		copy(p[0:4096], r.zeroes[:])
		p = p[4096:]
	}
	copy(p, r.zeroes[:len(p)])
	return n, nil
}

func BenchmarkRuneStream_advance(b *testing.B) {
	r := new(ZeroReader)
	stream := NewRuneStream(r)
	for i := 0; i < b.N; i++ {
		stream.Advance()
		stream.Commit()
	}
}
