package jsonv

import (
	"fmt"
	"reflect"
)

/*
A simple parser that accepts only a JSON string value and stores the result in
a *string or string field on a struct.

The value will be parsed (i.e. escaped chars and unicode chars parsed). Invalid
unicode code points will be replaced with unicode.ReplacementChar.
*/
type StringParser struct {
	vs []StringValidator
}

func String(vs ...StringValidator) *StringParser {
	return &StringParser{vs}
}

func (p *StringParser) Prepare(t reflect.Type) error {
	if t.Kind() != reflect.String {
		return fmt.Errorf("Want string not %v", t)
	}

	return nil
}

func (p *StringParser) Parse(path Pather, s *Scanner, v interface{}) error {
	tok, buf, err := s.ReadToken()
	if tok == TokenError {
		return err
	} else if tok != TokenString {
		return NewSingleVErr(path(), fmt.Sprintf(ERROR_INVALID_STRING, string(buf)))
	}

	if ss, ok := v.(*string); !ok {
		return fmt.Errorf(ERROR_BAD_STRING_DEST, reflect.TypeOf(v), path())
	} else {
		// now check for validation errors
		var errs ValidationError

		s, ok := Unquote(buf)
		if !ok {
			return errs.Add(path(), "Invalid string")
		}

		*ss = s

		// validate the contents
		for _, v := range p.vs {
			if err := v.ValidateString(*ss); err != nil {
				errs = errs.Add(path(), err.Error())
			}
		}

		if len(errs) > 0 {
			return errs
		} else {
			return nil
		}
	}
}
