package runestream

import (
	"bytes"
	"io"
	"testing"
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
	r := bytes.NewReader([]byte("English\r\nespañol\r\n日本語\r\n"))
	stream := NewRuneStream(r)

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
		t.Errorf("%d extra item(s)", len(actual) - min)
	}
	if len(expected) > min {
		t.Errorf("%d missing item(s)", len(expected) - min)
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
