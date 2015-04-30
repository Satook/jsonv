package jsonschema

import (
	"fmt"
	"io"
	"unicode"
)

const READ_LEN = 512
const TOK_TRUE = "true"
const TOK_FALSE = "false"
const TOK_NULL = "null"

type TokenType int

const (
	tokenError TokenType = iota // an IO error

	tokenObjectBegin
	tokenObjectEnd
	tokenArrayBegin
	tokenArrayEnd
	tokenItemSep // a single ',' token for arrays and objects
	tokenString
	tokenNumber
	tokenTrue
	tokenFalse
	tokenNull

	tokenEnd        // we're at the end of the stream
	tokenParseError // an un-parsable string was proveded
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
	case tokenEnd:
		return "End of input"
	case tokenParseError:
		return "Parse Error"
	default:
		return "IO Error"
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
/*
func (s *Scanner) ReadNumber() ([]byte, error) {
}

func (s *Scanner) ReadBool() (bool, error) {
}

func (s *Scanner) ReadString() ([]byte, error) {
}
*/
/*
Reads in one JSON token.

The underlying buffer for the returned byte slice is owned by this scanner.
Upon subsequent Read* calls, it may be overwritten or de-allocated.

On error, the returned byte slice will be the entire remaining read buffer.

There are 3 types of error:
 1. tokenEnd: There is no more data available
 2. tokenError: An IO error was encountered, socket dropped, etc.
 3. tokenParseError: We have the bytes, but they were malformed.
*/
func (s *Scanner) ReadToken() (TokenType, []byte, error) {
	// move to first non-space char (s.buf[s.roff] != space)
	var n int
	n, s.rerr = s.bytesUntilPred(0, notSpace) // could discardUntil to eliminate pointless allocations, but not the common case.
	s.roff += n
	s.rcount += n

	// have we run out of data?
	if s.roff >= len(s.buf) {
		var tok TokenType
		if s.rerr == io.EOF {
			tok = tokenEnd
		}
		return tok, s.buf[s.roff:], s.rerr
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
			if string(buf) == lookFor {
				s.roff += l
				s.rcount += l
				return tok, buf, nil
			} else {
				return tokenParseError, buf, nil
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
				// parse error
				return tokenParseError, s.buf[s.roff:], s.rerr
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
			buf := s.buf[s.roff : s.roff+offset+1]
			s.roff += len(buf)
			s.rcount += len(buf)
			return tokenNumber, s.buf, nil
		}
	}

	if s.rerr != nil {
		if s.rerr == io.EOF {
			tok = tokenEnd
		} else {
			tok = tokenError
		}
		return tok, s.buf[s.roff:], s.rerr
	} else {
		panic("Didn't get any more data but didn't get an EOF")
	}
}

/*
Will read in data in until there is at least count bytes in the buffer
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
Will keep reading in data until we get the specified character

Return semantics are the same as bytesUntilPred
*/
func (s *Scanner) bytesUntil(offset int, c byte) (int, error) {
	return s.bytesUntilPred(offset, func(b byte) bool { return b == c })
}

/*
Reads from s.roff+offset until it finds a byte where the pred returns true.
Returns the offset of that byte, relative to s.roff.
*/
func (s *Scanner) bytesUntilPred(offset int, p bytePred) (int, error) {
	for i := 0; i < 1024; i += 1 {
		// make sure there's at least 1-byte to read
		fmt.Printf("Prefil: (R: %d, O: %d, L: %d)\n", s.roff, offset, len(s.buf))
		for len(s.buf) <= s.roff+offset {
			if err := s.fillBuffer(); err != nil {
				return offset, err
			}
		}

		// scan through current read buff
		fmt.Printf("Scanning %q for something.\n", s.buf[s.roff+offset:])
		for _, c := range s.buf[s.roff+offset:] {
			fmt.Printf("Checking char %c\n", c)
			if p(c) {
				return offset, nil
			} else {
				offset += 1
			}
		}

	}

	return offset, fmt.Errorf("1024 iterations and no result")
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
		return nil, fmt.Errorf("expected digit in number literal")
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
		return nil, fmt.Errorf("expected digit in number literal")
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
		return nil, fmt.Errorf("expected digit or sign after 'e' in number literal")
	}
}

/*
Got up to the e (or E) and a + or - sign.
*/
func numStateESign(c byte) (NumParseState, error) {
	if c >= '0' && c <= '9' {
		return numStateE0, nil
	} else {
		return nil, fmt.Errorf("expected digit after exponent sign in number literal")
	}
}

func numStateE0(c byte) (NumParseState, error) {
	if c >= '0' && c <= '9' {
		return numStateE0, nil
	} else {
		return nil, nil
	}
}
