package jsonv

import (
	"encoding/json"
	"fmt"
	"reflect"
)

var UnmarshalerType = reflect.TypeOf((*json.Unmarshaler)(nil)).Elem()

/*
Parses JSON strings value and stores it in a Go time.Time.

The string must be in the format "yyyy-mm-dd"
*/
type UnmarshalParser struct {
}

func Unmarshaler() *UnmarshalParser {
	return &UnmarshalParser{}
}

func (p *UnmarshalParser) Prepare(t reflect.Type) error {
	if !t.Implements(UnmarshalerType) && !reflect.PtrTo(t).Implements(UnmarshalerType) {
		return fmt.Errorf("Must implement the encoding/json Unmarshaler interface. %v does not.", t)
	}

	return nil
}

func (p *UnmarshalParser) Parse(path Pather, s *Scanner, v interface{}) error {
	tok, buf, err := s.ReadToken()
	if tok == TokenError {
		return err
	}

	if dest, ok := v.(json.Unmarshaler); !ok {
		return NewParseError(ERROR_BAD_UNMARSHAL_DEST, reflect.TypeOf(v), path())
	} else if err := dest.UnmarshalJSON(buf); err != nil {
		var errs ValidationError
		errs = errs.Add(path(), err.Error())
		return errs
	}

	return nil
}
