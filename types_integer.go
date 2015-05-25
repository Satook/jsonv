package jsonv

import (
	"fmt"
	"reflect"
	"strconv"
)

/*
Parses any whole-integer JSON number value and stores it in any Go integer
primitive type, e.g. int8, int16, uint8, etc.
*/
type IntegerParser struct {
	vs []IntegerValidator
}

func Integer(vs ...IntegerValidator) *IntegerParser {
	return &IntegerParser{vs}
}

func (p *IntegerParser) Prepare(t reflect.Type) error {
	if t.Kind() != reflect.Int64 {
		return fmt.Errorf("Want int64 not %v", t)
	}

	return nil
}

func (p *IntegerParser) Parse(path string, s *Scanner, v interface{}) error {
	tok, buf, err := s.ReadToken()
	if tok == tokenError {
		return err
	} else if tok != tokenNumber {
		return NewParseError(fmt.Sprintf(ERROR_INVALID_INT, string(buf)))
	}

	var errs ValidationError

	tv, err := strconv.ParseInt(string(buf), 10, 64)
	if err != nil {
		errs = errs.Add(path, err.Error())
		return errs
	}

	// check the value
	for _, v := range p.vs {
		if err := v.ValidateInteger(tv); err != nil {
			errs = errs.Add(path, err.Error())
		}
	}
	if len(errs) > 0 {
		return errs
	}

	// now assign the value with whatever precision we can
	switch t := v.(type) {
	default:
		return fmt.Errorf(ERROR_BAD_INT_DEST, reflect.TypeOf(v))
	case *int64:
		*t = tv
	case *uint64:
		*t = uint64(tv)
	case *int:
		*t = int(tv)
	case *uint:
		*t = uint(tv)
	case *int16:
		*t = int16(tv)
	case *uint16:
		*t = uint16(tv)
	case *int8:
		*t = int8(tv)
	case *uint8:
		*t = uint8(tv)
	}

	return nil
}