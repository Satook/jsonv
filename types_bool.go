package jsonv

import (
	"fmt"
	"reflect"
)

/*
Parses true/false JSON values into a *bool/*string or bool/string struct field.

For strings, the literal text "false"/"true", without quotes, is assigned to the
string.
*/
type BooleanParser struct {
}

func Boolean() *BooleanParser {
	return &BooleanParser{}
}

func (p *BooleanParser) Prepare(t reflect.Type) error {
	if t.Kind() != reflect.Bool && t.Kind() != reflect.String {
		return fmt.Errorf("Want bool not %v", t)
	}

	return nil
}

func (p *BooleanParser) Parse(path Pather, s *Scanner, v interface{}) error {
	tok, buf, err := s.ReadToken()
	// wasn't the correct type
	if tok == tokenError {
		return err
	} else if tok != tokenTrue && tok != tokenFalse {
		return NewSingleVErr(path(), fmt.Sprintf(ERROR_INVALID_BOOL, string(buf)))
	}

	// now assign the value with whatever precision we can
	switch t := v.(type) {
	default:
		return fmt.Errorf(ERROR_BAD_BOOL_DEST, reflect.TypeOf(v), path())
	case *string:
		*t = string(buf)
	case *bool:
		*t = buf[0] == 't'
	}

	return nil
}
