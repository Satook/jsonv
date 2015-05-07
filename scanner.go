package jsonv

import (
	"fmt"
	"io"
	"strconv"
	"unicode"
)

const READ_LEN = 512
const TOK_TRUE = "true"
const TOK_FALSE = "false"
const TOK_NULL = "null"

type ParseError struct {
	e string
}

func NewParseError(e string) error {
	return &ParseError{e}
}

func (p *ParseError) Error() string {
	return p.e
}

type TokenType int

const (
	tokenError TokenType = iota // an IO error (legit or EOF)

	tokenObjectBegin
	tokenObjectEnd
	tokenArrayBegin
	tokenArrayEnd
	tokenItemSep // a single ',' token for arrays and objects
	tokenPropSep
	tokenString
	tokenNumber
	tokenTrue
	tokenFalse
	tokenNull
)

/*
Nice text versions for error messages to clients.
*/
func (t TokenType) String() string {
	switch t {
	case tokenObjectBegin:
		return "{"
	case tokenObjectEnd:
		return "}"
	case tokenArrayBegin:
		return "["
	case tokenArrayEnd:
		return "]"
	case tokenItemSep:
		return ","
	case tokenPropSep:
		return ":"
	case tokenString:
		return "string"
	case tokenNumber:
		return "number"
	case tokenTrue:
		return TOK_TRUE
	case tokenFalse:
		return TOK_FALSE
	case tokenNull:
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
	if tok == tokenError {
		return err
	}

	return s._skipValue(tok)
}

func (s *Scanner) _skipValue(tok TokenType) error {
	switch tok {
	default:
		return NewParseError("Expected JSON value, e.g. string, bool, etc.")
	case tokenObjectBegin:
		return s.skipObject()
	case tokenArrayBegin:
		return s.skipArray()
	case tokenString, tokenNumber, tokenTrue, tokenFalse, tokenNull:
		// aaaaand we're done
		return nil
	}
}

func (s *Scanner) skipObject() error {
	for {
		// are we done?
		if tok, _, err := s.ReadToken(); err != nil {
			return err
		} else if tok == tokenObjectEnd {
			break
		} else if tok != tokenString {
			return NewParseError("Expected string or '}', not " + tok.String())
		}

		// now read the ':'
		if tok, _, err := s.ReadToken(); err != nil {
			return err
		} else if tok != tokenPropSep {
			return NewParseError("Expected ':' not " + tok.String())
		}

		// now skip an entire value
		if err := s.SkipValue(); err != nil {
			return err
		}

		if tok, _, err := s.ReadToken(); err != nil {
			return err
		} else if tok == tokenItemSep {
			continue
		} else if tok == tokenObjectEnd {
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
		} else if tok == tokenArrayEnd {
			break
		} else if err := s._skipValue(tok); err != nil {
			return err
		}

		// we want a , or a ']'
		tok, _, err := s.ReadToken()
		if err != nil {
			return err
		} else if tok == tokenItemSep {
			continue
		} else if tok == tokenArrayEnd {
			break
		} else {
			return NewParseError("Expected ',' or ']', not " + tok.String())
		}
	}

	return nil
}

/*
Reads a single null value from the stream. Returns an error if the next token is
not null
*/
/*
func (s *Scanner) ReadNull() error {
}
*/
/*
Reads a single number value. Returning it's characters. Returns an error if the
next token is not a valid or is not a number.
*/

func (s *Scanner) ReadInteger() (int64, error) {
	tok, buf, err := s.ReadToken()
	if tok == tokenError {
		return 0, err
	} else if tok != tokenNumber {
		return 0, NewParseError(fmt.Sprintf(ERROR_INVALID_INT, string(buf)))
	}

	tv, err := strconv.ParseInt(string(buf), 10, 64)
	if err != nil {
		return 0, err
	}

	return tv, nil
}

func (s *Scanner) ReadBool() (bool, error) {
	return true, nil
}

func (s *Scanner) ReadString() (string, error) {
	return "", nil
}

/*
Reads in one JSON token.

The underlying buffer for the returned byte slice is owned by this scanner.
Upon subsequent Read* calls, it may be overwritten or de-allocated.

On error, the returned byte slice will be the entire remaining read buffer.

There are 2 types of error:
 1. General: An IO error was encountered, EOF, socket dropped, etc.
 2. ParseError: We have the bytes, but they were malformed, parsing cannot
 continue.
*/
func (s *Scanner) ReadToken() (TokenType, []byte, error) {
	// move to first non-space char (s.buf[s.roff] != space)
	var n int
	n, s.rerr = s.bytesUntilPred(0, notSpace) // could discardUntil to eliminate pointless allocations, but not the common case.
	s.roff += n
	s.rcount += n

	// have we run out of data?
	if s.roff >= len(s.buf) {
		return tokenError, s.buf[s.roff:], s.rerr
	}

	// cover off single character token
	tok := tokenError
	first := s.buf[s.roff]
	switch first {
	case '{':
		tok = tokenObjectBegin
	case '}':
		tok = tokenObjectEnd
	case '[':
		tok = tokenArrayBegin
	case ']':
		tok = tokenArrayEnd
	case ',':
		tok = tokenItemSep
	case ':':
		tok = tokenPropSep
	}
	// return the single char token
	if tok != tokenError {
		buf := s.buf[s.roff : s.roff+1]
		s.roff += 1
		s.rcount += 1
		return tok, buf, nil
	}

	// now deal with string tokens (true, false, nill)
	var lookFor string
	switch first {
	case 't':
		tok = tokenTrue
		lookFor = TOK_TRUE
	case 'f':
		tok = tokenFalse
		lookFor = TOK_FALSE
	case 'n':
		tok = tokenNull
		lookFor = TOK_NULL
	}
	// read what we want, check it's correct, return it or a parse error
	if tok != tokenError {
		l := len(lookFor)
		if err := s.atLeast(l); err == nil {
			buf := s.buf[s.roff : s.roff+l]
			sbuf := string(buf)
			if sbuf == lookFor {
				s.roff += l
				s.rcount += l
				return tok, buf, nil
			} else {
				return tokenError, buf, NewParseError("Expected " + lookFor + ", not " + sbuf)
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
				// this is a non-escaped "
				tok = tokenString
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
				// TODO: make it a parse error
				return tokenError, s.buf[s.roff:], s.rerr
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
			return tokenNumber, buf, nil
		}
	}

	if s.rerr != nil {
		return tokenError, s.buf[s.roff:], s.rerr
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
			s.roff = 0
		} else {
			// need a bigger buffer
			newBuf := make([]byte, len(s.buf), 2*cap(s.buf)+READ_LEN)
			copy(newBuf, s.buf[s.roff:])
			s.buf = newBuf
			s.roff = 0
		}
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
