package jsonv

import (
	"fmt"
	"reflect"
)

/*
Parses strings into byte slices, i.e []byte, This still decodes the string value
but avoids making additional copies of the data when a byte slice is required
(e.g. decoding/decryption/etc).
*/
type ByteSliceParser struct {
	vs []BytesValidator
}

func Bytes(vs ...BytesValidator) *ByteSliceParser {
	return &ByteSliceParser{vs}
}

func (p *ByteSliceParser) Prepare(t reflect.Type) error {
	if t.Kind() != reflect.Slice || t.Elem().Kind() != reflect.Uint8 {
		return fmt.Errorf("Want []byte not %v", t)
	}

	return nil
}

func (p *ByteSliceParser) Parse(path string, s *Scanner, v interface{}) error {
	tok, buf, err := s.ReadToken()
	if tok == tokenError {
		return err
	} else if tok != tokenString {
		return NewSingleVErr(path, fmt.Sprintf(ERROR_INVALID_STRING, string(buf)))
	}

	if bdest, ok := v.(*[]byte); !ok {
		return fmt.Errorf(ERROR_BAD_BYTE_DEST, reflect.TypeOf(v), path)
	} else {
		var errs ValidationError

		buff, ok := UnquoteBytes(buf)
		if !ok {
			return errs.Add(path, "Invalid string")
		}

		*bdest = buff

		// validate the contents
		for _, v := range p.vs {
			if err := v.ValidateBytes(*bdest); err != nil {
				errs = errs.Add(path, err.Error())
			}
		}

		if len(errs) > 0 {
			return errs
		} else {
			return nil
		}
	}

	return nil
}

/*
Parses strings into byte slices, i.e []byte. This does not decode the string
value, so escape sequences (e.g. "\n", "\u0020") will be left as-is.

This is useful if the value is only ever meant to be non-escaped chars, e.g. a
base64 encoded string.
*/
type RawByteSliceParser struct {
}

func RawBytes() *RawByteSliceParser {
	return &RawByteSliceParser{}
}

func (p *RawByteSliceParser) Prepare(t reflect.Type) error {
	if t.Kind() != reflect.Slice || t.Elem().Kind() != reflect.Uint8 {
		return fmt.Errorf("Want []byte not %v", t)
	}

	return nil
}

func (p *RawByteSliceParser) Parse(path string, s *Scanner, v interface{}) error {
	tok, buf, err := s.ReadToken()
	if tok == tokenError {
		return err
	} else if tok != tokenString {
		return NewSingleVErr(path, fmt.Sprintf(ERROR_INVALID_STRING, string(buf)))
	}

	if bdest, ok := v.(*[]byte); !ok {
		return fmt.Errorf(ERROR_BAD_BYTE_DEST, reflect.TypeOf(v), path)
	} else {
		// scanner owns buf, so we need to make a copy
		*bdest = make([]byte, len(buf)-2)
		copy(*bdest, buf[1:len(buf)-1])
	}

	return nil
}
