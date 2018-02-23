# go-runestream

![GitHub release](https://img.shields.io/github/release/chronos-tachyon/go-runestream.svg)
[![GoDoc](https://img.shields.io/badge/godoc-reference-5272B4.svg)](http://godoc.org/github.com/chronos-tachyon/go-runestream)
[![Build Status](https://travis-ci.org/chronos-tachyon/go-runestream.svg?branch=master)](https://travis-ci.org/chronos-tachyon/go-runestream)

RuneStream implements streaming decodes of text data.

Using RuneStream is conceptually similar to using ReadRune() / UnreadRune()
from bufio.Reader, but with unbounded backtracking and a mechanism for
explicitly managing how deep the backtracking state goes.

RuneStream also tracks the human-friendly position (lines and columns) of
each character within the text file, and it has some convenience methods for
extracting multi-rune sequences that match a pattern.

## Bare-bones usage

```go
func StreamRunes(filename string, callback func(rune, int, Position)) error {
	f, err := os.Open("filename.txt")
	if err != nil {
		return err
	}

	stream := NewRuneStream(f)
	var out []rune
	for stream.Advance() {
		r := stream.Rune()
		size := stream.Size()
		pos := stream.Position()
		callback(r, size, pos)
		stream.Commit()
	}

	err = stream.Err()
	if err == io.EOF {
		err = nil
	}
	return err
}
```

## Advanced usage

```go
func LexWordOrNumber(stream *RuneStream) Token {
	if !stream.Advance() {
		err := stream.Err()
		return ErrorToken(err)
	}
	r := stream.Rune()
	pos := stream.Position()
	if unicode.IsLetter(r) {
		word := []rune{r}
		word = stream.TakeWhile(-1, word, unicode.IsLetter)
		stream.Commit()
		return WordToken(pos, string(word))
	}
	if unicode.IsDigit(r) {
		number := []rune{r}
		number = stream.TakeWhile(-1, number, unicode.IsDigit)
		stream.Commit()
		return NumberToken(pos, string(number))
	}
	stream.Rewind()
	return FailToken()
}
```

## Save points

```go
func Exactly(r rune) func(rune) bool {
	return func(rr rune) bool {
		return r == rr
	}
}

func IsSign(r rune) bool {
	return r == '+' || r == '-'
}

func IsRadix(r rune) bool {
	return r == 'b' || r == 'o' || r == 'x'
}

func IsBin(r rune) bool {
	return r == '0' || r == '1'
}

func IsOct(r rune) bool {
	return r >= '0' && r <= '7'
}

func IsDec(r rune) bool {
	return r >= '0' && r <= '9'
}

func IsHex(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')
}

func LexInteger(stream *RuneStream) Token {
	if !stream.Advance() {
		err := stream.Err()
		return ErrorToken(err)
	}

	pos := stream.Position()
	stream.Rewind()

	sign, hasSign := stream.Take(IsSign)

	savePoint := stream.Save()  // <-- create save point

	radix := 10
	isDigit := IsDec
	r0, ok0 := stream.Take(Exactly('0'))
	r1, ok1 := stream.Take(IsRadix)
	if ok0 && ok1 {
		switch r1 {
		case 'b':
			radix = 2
			isDigit = IsBin
		case 'o':
			radix = 8
			isDigit = IsOct
		case 'x':
			radix = 16
			isDigit = IsHex
		}
	} else {
		stream.Restore(savePoint)  // <-- use save point to unread '0'
	}

	digits := stream.TakeWhile(-1, nil, isDigit)
	stream.Commit()

	u64, err := strconv.ParseUint(string(digits), radix, 64)
	if err != nil {
		return ErrorToken(fmt.Errorf("failed to parse integer: %v", err))
	}
	i64 := int64(u64)
	if i64 < 0 {
		return ErrorToken(fmt.Errorf("integer out of range"))
	}
	if hasSign && sign == '-' {
		i64 = -i64
	}
	return IntegerToken(i64, pos)
}
```
