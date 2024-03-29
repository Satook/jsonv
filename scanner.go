package jsonv

import (
	"fmt"
	"io"
	"strconv"
	"unicode"
	"unicode/utf16"
	"unicode/utf8"
)

const READ_LEN = 256
const TOK_TRUE = "true"
const TOK_FALSE = "false"
const TOK_NULL = "null"

/*
Represents an error in the input stream that renders it unparsable, i.e. not
valid JSON.

This should not be used for errors where parsing can continue.
*/
type ParseError struct {
	e string
}

func NewParseError(e string, args ...interface{}) error {
	if len(args) == 0 {
		return &ParseError{e}
	} else {
		return &ParseError{fmt.Sprintf(e, args...)}
	}
}

func (p *ParseError) Error() string {
	return p.e
}

type TokenType int

const (
	TokenError TokenType = iota // an IO error (legit or EOF)

	TokenObjectBegin
	TokenObjectEnd
	TokenArrayBegin
	TokenArrayEnd
	TokenItemSep // a single ',' token for arrays and objects
	TokenPropSep
	TokenString
	TokenNumber
	TokenTrue
	TokenFalse
	TokenNull
)

/*
Nice text versions for error messages to clients.
*/
func (t TokenType) String() string {
	switch t {
	case TokenObjectBegin:
		return "{"
	case TokenObjectEnd:
		return "}"
	case TokenArrayBegin:
		return "["
	case TokenArrayEnd:
		return "]"
	case TokenItemSep:
		return ","
	case TokenPropSep:
		return ":"
	case TokenString:
		return "string"
	case TokenNumber:
		return "number"
	case TokenTrue:
		return TOK_TRUE
	case TokenFalse:
		return TOK_FALSE
	case TokenNull:
		return TOK_NULL
	default:
		return "Error"
	}
}

type bytePred func(byte) bool

/*
Is the given byte a Whitepace character, as per the RFC.

This works even if there is a multi-byte character sequence in the UTF-8 byte
string, because all chars > 0x7F have their high bit set yet no JSON space chars
do.
*/
func isSpace(c byte) bool {
	// ' ' space, '\t' horizontal tab, '\n' newline, 'r' charriage return
	return c == 0x20 || c == 0x09 || c == 0x0A || c == 0x0D
}

func notSpace(c byte) bool {
	return !(c == 0x20 || c == 0x09 || c == 0x0A || c == 0x0D)
}

/*
Reads from a buffer parsing as JSON tokens.

Has convenience methods for requesting a specific type be read in.

Note: All the read methods that return []byte are returning a slice that
references a buffer owned by Scanner, and as such copies should be made before
handing off to functions that may modify the bytes.
*/
type Scanner struct {
	r      io.Reader
	rcount int // the number of bytes read in total
	buf    []byte
	roff   int   // the next byte to process
	rerr   error // most recent read error
}

func NewScanner(r io.Reader) *Scanner {
	return &Scanner{r: r}
}

/*
Skips over a single value in the input.
*/
func (s *Scanner) SkipValue() error {
	// read the first token
	tok, _, err := s.ReadToken()
	if tok == TokenError {
		return err
	}

	return s._skipValue(tok)
}

func (s *Scanner) _skipValue(tok TokenType) error {
	switch tok {
	default:
		return NewParseError("Expected JSON value, e.g. string, bool, etc.")
	case TokenObjectBegin:
		return s.skipObject()
	case TokenArrayBegin:
		return s.skipArray()
	case TokenString, TokenNumber, TokenTrue, TokenFalse, TokenNull:
		// aaaaand we're done
		return nil
	}
}

func (s *Scanner) skipObject() error {
	for {
		// read the key, or '}'
		if tok, _, err := s.ReadToken(); err != nil {
			return err
		} else if tok == TokenObjectEnd {
			break
		} else if tok != TokenString {
			return NewParseError("Expected string or '}', not " + tok.String())
		}

		// now read the ':'
		if tok, _, err := s.ReadToken(); err != nil {
			return err
		} else if tok != TokenPropSep {
			return NewParseError("Expected ':' not " + tok.String())
		}

		// now skip an entire value
		if err := s.SkipValue(); err != nil {
			return err
		}

		if tok, _, err := s.ReadToken(); err != nil {
			return err
		} else if tok == TokenItemSep {
			continue
		} else if tok == TokenObjectEnd {
			break
		} else {
			return NewParseError("Expected ',' or '}', not " + tok.String())
		}
	}

	return nil
}

func (s *Scanner) skipArray() error {
	for {
		if tok, _, err := s.ReadToken(); err != nil {
			return err
		} else if tok == TokenArrayEnd {
			break
		} else if err := s._skipValue(tok); err != nil {
			return err
		}

		// we want a , or a ']'
		tok, _, err := s.ReadToken()
		if err != nil {
			return err
		} else if tok == TokenItemSep {
			continue
		} else if tok == TokenArrayEnd {
			break
		} else {
			return NewParseError("Expected ',' or ']', not " + tok.String())
		}
	}

	return nil
}

/*
Reads forward to the next Token, but only returns its type, leaves the read
cursor pointed at its first byte, unlike ReadToken which leaves the read cursor
just past its last.
*/
func (s *Scanner) PeekToken() (TokenType, error) {
	var n int
	n, s.rerr = s.bytesUntilPred(0, notSpace)
	s.roff += n
	s.rcount += n

	// have we run out of data?
	if s.roff >= len(s.buf) {
		return TokenError, s.rerr
	}

	// we only need the first character
	tok := TokenError
	switch s.buf[s.roff] {
	case '{':
		tok = TokenObjectBegin
	case '}':
		tok = TokenObjectEnd
	case '[':
		tok = TokenArrayBegin
	case ']':
		tok = TokenArrayEnd
	case ',':
		tok = TokenItemSep
	case ':':
		tok = TokenPropSep
	case 't':
		tok = TokenTrue
	case 'f':
		tok = TokenFalse
	case 'n':
		tok = TokenNull
	case '"':
		tok = TokenString
	case '-', '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
		tok = TokenNumber
	default:
		return TokenError, NewParseError("Invaid JSON")
	}

	return tok, nil
}

/*
Reads in one JSON token.

The underlying buffer for the returned byte slice is owned by this scanner.
Upon subsequent Read* calls, it may be overwritten or de-allocated.

On error, the returned TokenType will be tokenError and the  byte slice refer to
the entire remaining read buffer.

The returned error can be of 2 different types:
 1. General: An IO error was encountered, EOF, socket dropped, etc.
 2. ParseError: We have the data, but it was malformed, parsing cannot continue.
*/
func (s *Scanner) ReadToken() (TokenType, []byte, error) {
	// move to first non-space char (s.buf[s.roff] != space)
	var n int
	n, s.rerr = s.bytesUntilPred(0, notSpace) // could discardUntil to eliminate pointless allocations, but not the common case.
	s.roff += n
	s.rcount += n

	// have we run out of data?
	if s.roff >= len(s.buf) {
		return TokenError, s.buf[s.roff:], s.rerr
	}

	// cover off single character Token
	tok := TokenError
	first := s.buf[s.roff]
	switch first {
	case '{':
		tok = TokenObjectBegin
	case '}':
		tok = TokenObjectEnd
	case '[':
		tok = TokenArrayBegin
	case ']':
		tok = TokenArrayEnd
	case ',':
		tok = TokenItemSep
	case ':':
		tok = TokenPropSep
	}
	// return the single char token
	if tok != TokenError {
		buf := s.buf[s.roff : s.roff+1]
		s.roff += 1
		s.rcount += 1
		return tok, buf, nil
	}

	// now deal with string tokens (true, false, nill)
	var lookFor string
	switch first {
	case 't':
		tok = TokenTrue
		lookFor = TOK_TRUE
	case 'f':
		tok = TokenFalse
		lookFor = TOK_FALSE
	case 'n':
		tok = TokenNull
		lookFor = TOK_NULL
	}
	// read what we want, check it's correct, return it or a parse error
	if tok != TokenError {
		l := len(lookFor)
		if err := s.atLeast(l); err == nil {
			buf := s.buf[s.roff : s.roff+l]
			sbuf := string(buf)
			if sbuf == lookFor {
				s.roff += l
				s.rcount += l
				return tok, buf, nil
			} else {
				return TokenError, buf, NewParseError("Expected " + lookFor + ", not " + sbuf)
			}
		}
	} else if first == '"' {
		// need to read until either an escape char or "
		// if we stop but are just next to the last escape, scan again
		// if escape, save it's location and scan again
		// if it's a ", we've found the end!
		escapePos := -100
		offset := 0
		for {
			// start reading from last stop character + 1
			offset += 1
			offset, err := s.bytesUntilPred(offset, func(c byte) bool { return c == '\\' || c == '"' })
			if err != nil {
				break
			}

			char := s.buf[s.roff+offset]
			if offset == escapePos+1 {
				// this char is escaped
			} else if char == '"' {
				// this is a non-escaped ", i.e. the end of the string
				tok = TokenString
				buf := s.buf[s.roff : s.roff+offset+1]
				s.roff += len(buf)
				s.rcount += len(buf)
				return tok, buf, nil
			} else {
				// it's the start of an escape, save it for later
				escapePos = offset
			}
		}
	} else if first == '-' || unicode.IsDigit(rune(first)) {
		// pick starting parser state
		var state NumParseState
		if first == '-' {
			state = numStateNeg
		} else if first == '0' {
			state = numState0
		} else {
			state = numState1
		}

		var perr error
		var offset int
		for offset = 1; s.atLeast(offset+1) == nil; offset += 1 {
			// push it through the machine
			state, perr = state(s.buf[s.roff+offset])
			if perr != nil {
				return TokenError, s.buf[s.roff:], perr
			} else if state == nil {
				// finished
				break
			}
		}

		// we might be at the end of our input, so hand a fake ' ' to finish off
		// an incomplete parse
		if state != nil {
			state, _ = state(0x20)
		}
		if state == nil {
			buf := s.buf[s.roff : s.roff+offset]
			s.roff += len(buf)
			s.rcount += len(buf)
			return TokenNumber, buf, nil
		}
	} else {
		return TokenError, s.buf[s.roff:], NewParseError("Expected valid JSON")
	}

	if s.rerr != nil {
		return TokenError, s.buf[s.roff:], s.rerr
	} else {
		panic("Didn't get any more data but didn't get an EOF")
	}
}

/*
Will read in data in until there is at least count bytes in the buffer.
*/
func (s *Scanner) atLeast(count int) error {
	for len(s.buf) < s.roff+count {
		if err := s.fillBuffer(); err != nil {
			return err
		}
	}
	return nil
}

/*
Reads from s.roff+offset until it finds a byte where the pred returns true.
Returns the offset of that byte, relative to s.roff.
*/
func (s *Scanner) bytesUntilPred(offset int, p bytePred) (int, error) {
	for i := 0; i < 1024; i += 1 {
		// make sure there's at least 1-byte to read
		for len(s.buf) <= s.roff+offset {
			if err := s.fillBuffer(); err != nil {
				return offset, err
			}
		}

		// scan through current read buff
		for _, c := range s.buf[s.roff+offset:] {
			if p(c) {
				return offset, nil
			} else {
				offset += 1
			}
		}
	}

	return offset, NewParseError("1024 iterations and no result")
}

/*
Reads in up-to another READ_LEN count bytes into our buffer
*/
func (s *Scanner) fillBuffer() error {
	if s.rerr != nil {
		return s.rerr
	}

	// ensure space for the read
	if cap(s.buf)-len(s.buf) < READ_LEN {
		used := len(s.buf) - s.roff
		if cap(s.buf)-used >= READ_LEN {
			// buffer can fit if we eliminate already processed data
			rest := copy(s.buf, s.buf[s.roff:])
			s.buf = s.buf[0:rest]
		} else {
			// need a bigger buffer
			newBuf := make([]byte, used, 2*cap(s.buf)+READ_LEN)
			copy(newBuf, s.buf[s.roff:])
			s.buf = newBuf
		}
		s.roff = 0
	}

	// now read it in and store any potential error for post-parse checking
	var n int
	n, s.rerr = s.r.Read(s.buf[len(s.buf):cap(s.buf)])
	s.buf = s.buf[0 : len(s.buf)+n]

	// normalise to only return error with no data
	if n == 0 && s.rerr != nil {
		return s.rerr
	} else {
		return nil
	}
}

/* Number parsing states

These represent a state during the parsing of a single JSON number value.

They are used by ReadToken to parse a number. Each one is called with a single
byte to check and returns the new state function that should be used to parse
the next character.

If there is a parse error, no function is returned and an error is returned.
*/

type NumParseState func(c byte) (NumParseState, error)

/*
Up to the minus sign
*/
func numStateNeg(c byte) (NumParseState, error) {
	if c == '0' {
		return numState0, nil
	} else if c >= '1' && c <= '9' {
		return numState1, nil
	} else {
		return nil, NewParseError("expected digit in number literal")
	}
}

/*
first digit is 0
*/
func numState0(c byte) (NumParseState, error) {
	if c == '.' {
		return numStateDot, nil
	} else if c == 'e' || c == 'E' {
		return numStateE, nil
	} else {
		return nil, nil
	}
}

/*
First digit is non-zero
*/
func numState1(c byte) (NumParseState, error) {
	if c >= '0' && c <= '9' {
		return numState1, nil
	} else if c == '.' {
		return numStateDot, nil
	} else if c == 'e' || c == 'E' {
		return numStateE, nil
	} else {
		return nil, nil
	}
}

/*
Got up to the decimal point
*/
func numStateDot(c byte) (NumParseState, error) {
	if c >= '0' && c <= '9' {
		return numStateDot0, nil
	} else {
		return nil, NewParseError("expected digit in number literal")
	}
}

/*
Got at least 1 digit after the decimal point
*/
func numStateDot0(c byte) (NumParseState, error) {
	if c >= '0' && c <= '9' {
		return numStateDot0, nil
	} else if c == 'e' || c == 'E' {
		return numStateE, nil
	} else {
		return nil, nil
	}
}

/*
Got some leading number and an e (or E).
*/
func numStateE(c byte) (NumParseState, error) {
	if c >= '0' && c <= '9' {
		return numStateE0, nil
	} else if c == '-' || c == '+' {
		return numStateESign, nil
	} else {
		return nil, NewParseError("expected digit or sign after 'e' in number literal")
	}
}

/*
Got up to the e (or E) and a + or - sign.
*/
func numStateESign(c byte) (NumParseState, error) {
	if c >= '0' && c <= '9' {
		return numStateE0, nil
	} else {
		return nil, NewParseError("expected digit after exponent sign in number literal")
	}
}

func numStateE0(c byte) (NumParseState, error) {
	if c >= '0' && c <= '9' {
		return numStateE0, nil
	} else {
		return nil, nil
	}
}

// getu4 decodes \uXXXX from the beginning of s, returning the hex value,
// or it returns -1.
func getu4(s []byte) rune {
	if len(s) < 6 || s[0] != '\\' || s[1] != 'u' {
		return -1
	}
	r, err := strconv.ParseUint(string(s[2:6]), 16, 64)
	if err != nil {
		return -1
	}
	return rune(r)
}

// unquote converts a quoted JSON string literal s into an actual string t.
// The rules are different than for Go, so cannot use strconv.Unquote.
func Unquote(s []byte) (t string, ok bool) {
	s, ok = UnquoteBytes(s)
	t = string(s)
	return
}

func UnquoteBytes(s []byte) (t []byte, ok bool) {
	if len(s) < 2 || s[0] != '"' || s[len(s)-1] != '"' {
		return
	}
	s = s[1 : len(s)-1]

	// Check for unusual characters. If there are none,
	// then no unquoting is needed, so return a slice of the
	// original bytes.
	r := 0
	for r < len(s) {
		c := s[r]
		if c == '\\' || c == '"' || c < ' ' {
			break
		}
		if c < utf8.RuneSelf {
			r++
			continue
		}
		rr, size := utf8.DecodeRune(s[r:])
		if rr == utf8.RuneError && size == 1 {
			break
		}
		r += size
	}
	if r == len(s) {
		return s, true
	}

	b := make([]byte, len(s)+2*utf8.UTFMax)
	w := copy(b, s[0:r])
	for r < len(s) {
		// Out of room?  Can only happen if s is full of
		// malformed UTF-8 and we're replacing each
		// byte with RuneError.
		if w >= len(b)-2*utf8.UTFMax {
			nb := make([]byte, (len(b)+utf8.UTFMax)*2)
			copy(nb, b[0:w])
			b = nb
		}
		switch c := s[r]; {
		case c == '\\':
			r++
			if r >= len(s) {
				return
			}
			switch s[r] {
			default:
				return
			case '"', '\\', '/', '\'':
				b[w] = s[r]
				r++
				w++
			case 'b':
				b[w] = '\b'
				r++
				w++
			case 'f':
				b[w] = '\f'
				r++
				w++
			case 'n':
				b[w] = '\n'
				r++
				w++
			case 'r':
				b[w] = '\r'
				r++
				w++
			case 't':
				b[w] = '\t'
				r++
				w++
			case 'u':
				r--
				rr := getu4(s[r:])
				if rr < 0 {
					return
				}
				r += 6
				if utf16.IsSurrogate(rr) {
					rr1 := getu4(s[r:])
					if dec := utf16.DecodeRune(rr, rr1); dec != unicode.ReplacementChar {
						// A valid pair; consume.
						r += 6
						w += utf8.EncodeRune(b[w:], dec)
						break
					}
					// Invalid surrogate; fall back to replacement rune.
					rr = unicode.ReplacementChar
				}
				w += utf8.EncodeRune(b[w:], rr)
			}

		// Quote, control characters are invalid.
		case c == '"', c < ' ':
			return

		// ASCII
		case c < utf8.RuneSelf:
			b[w] = c
			r++
			w++

		// Coerce to well-formed UTF-8.
		default:
			rr, size := utf8.DecodeRune(s[r:])
			r += size
			w += utf8.EncodeRune(b[w:], rr)
		}
	}
	return b[0:w], true
}
