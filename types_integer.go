package jsonv

import (
	"fmt"
	"reflect"
	"strconv"
)

/*
Parses any whole-integer JSON number value and stores it in any Go integer
primitive type, e.g. int8, int16, uint8, etc.

It can only parse values that are within the int64 range, even when stored into
a uint64 variable.
*/
type IntegerParser struct {
	vs      []IntegerValidator
	bitSize int
}

func Integer(vs ...IntegerValidator) *IntegerParser {
	return &IntegerParser{vs, 64}
}

func (p *IntegerParser) Prepare(t reflect.Type) error {
	switch t.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
	default:
		return fmt.Errorf("Want an integer type not %v", t)
	}

	p.bitSize = t.Bits()
	return nil
}

func (p *IntegerParser) Parse(path Pather, s *Scanner, v interface{}) error {
	tok, buf, err := s.ReadToken()
	if tok == tokenError {
		return err
	} else if tok != tokenNumber {
		return NewParseError(ERROR_INVALID_INT, string(buf))
	}

	var errs ValidationError

	tv, err := strconv.ParseInt(string(buf), 10, p.bitSize)
	if err != nil {
		errs = errs.Add(path(), err.Error())
		return errs
	}

	// check the value
	for _, v := range p.vs {
		if err := v.ValidateInteger(tv); err != nil {
			errs = errs.Add(path(), err.Error())
		}
	}

	// bail before setting if validation failed
	if len(errs) > 0 {
		return errs
	}

	// now assign the value with whatever precision we can
	switch t := v.(type) {
	default:
		return NewParseError(ERROR_BAD_INT_DEST, reflect.TypeOf(v))
	case *int:
		*t = int(tv)
	case *int8:
		*t = int8(tv)
	case *int16:
		*t = int16(tv)
	case *int64:
		*t = tv
	case *uint:
		*t = uint(tv)
	case *uint8:
		*t = uint8(tv)
	case *uint16:
		*t = uint16(tv)
	case *uint64:
		*t = uint64(tv)
	}

	return nil
}
